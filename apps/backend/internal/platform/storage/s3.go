package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

/** S3Config configura un bucket privado con credenciales inyectadas por entorno.
 * Endpoint incluye el bucket: https://bucket.s3.region.amazonaws.com o
 * http://minio:9000/bucket. PublicEndpoint permite otro host para las descargas públicas.
 */
type S3Config struct {
	Endpoint       string
	PublicEndpoint string
	Region         string
	AccessKey      string
	SecretKey      string
	SessionToken   string
}

/** S3 implementa la interfaz document.Storage utilizando el AWS SDK for Go v2 oficial.
 * Soporta AWS S3 nativo y motores compatibles (MinIO, RustFS, Cloudflare R2, Ceph).
 */
type S3 struct {
	bucket        string
	client        *s3.Client
	presignClient *s3.PresignClient
}

/** parseEndpoint descompone un endpoint con bucket (ej: http://rustfs:9000/biess)
 * en el BaseEndpoint (http://rustfs:9000) y el nombre del bucket (biess).
 */
func parseEndpoint(raw string) (baseEndpoint, bucket string, err error) {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", "", errors.New("endpoint S3 no válido")
	}
	baseEndpoint = fmt.Sprintf("%s://%s", u.Scheme, u.Host)
	bucket = strings.Trim(u.Path, "/")
	if bucket == "" {
		return "", "", errors.New("el endpoint S3 debe incluir el nombre del bucket en la ruta (ej: http://host:9000/mi-bucket)")
	}
	return baseEndpoint, bucket, nil
}

/** NewS3 valida configuración y construye el cliente S3 con aws-sdk-go-v2.
 * @returns {S3, error} Adaptador para un bucket aprovisionado y privado.
 */
func NewS3(cfg S3Config) (*S3, error) {
	if cfg.Endpoint == "" {
		return nil, errors.New("endpoint S3 no válido")
	}
	baseEndpoint, bucket, err := parseEndpoint(cfg.Endpoint)
	if err != nil {
		return nil, err
	}
	if cfg.PublicEndpoint != "" {
		if _, _, err := parseEndpoint(cfg.PublicEndpoint); err != nil {
			return nil, err
		}
	}
	if cfg.AccessKey == "" || cfg.SecretKey == "" || cfg.Region == "" {
		return nil, errors.New("S3 requiere región y credenciales")
	}

	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKey,
			cfg.SecretKey,
			cfg.SessionToken,
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("error al inicializar configuración de AWS SDK: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(baseEndpoint)
		o.UsePathStyle = true
	})

	// El presign para el broker de descargas internas debe usar el endpoint interno (baseEndpoint)
	// de modo que la API pueda alcanzar el bucket dentro de la red del cluster/Docker.
	presignClient := s3.NewPresignClient(client)

	return &S3{
		bucket:        bucket,
		client:        client,
		presignClient: presignClient,
	}, nil
}

/** Save sube un documento PDF sin ACL pública y sin exponer información sensible.
 * @returns {error} Fallo remoto sin filtrar credenciales ni detalles de conexión.
 */
func (s *S3) Save(ctx context.Context, key string, content io.Reader) error {
	if key == "" || strings.HasPrefix(key, "/") || strings.Contains(key, "..") {
		return errors.New("clave S3 no válida")
	}
	// Si el contenido viene como io.Reader, se puede leer en un buffer o subir directamente
	var body io.Reader = content
	if b, ok := content.(*bytes.Buffer); ok {
		body = bytes.NewReader(b.Bytes())
	}
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String("application/pdf"),
	})
	if err != nil {
		return fmt.Errorf("el almacenamiento S3 respondió error al guardar: %w", err)
	}
	return nil
}

/** Delete es idempotente, como DeleteObject de S3.
 * @returns {error} Fallo de comunicación con el bucket.
 */
func (s *S3) Delete(ctx context.Context, key string) error {
	if key == "" || strings.HasPrefix(key, "/") || strings.Contains(key, "..") {
		return errors.New("clave S3 no válida")
	}
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("el almacenamiento S3 respondió error al eliminar: %w", err)
	}
	return nil
}

/** PresignGet firma la descarga temporal (default 5m) con Content-Disposition attachment.
 * @returns {string, error} URL pre-firmada válida con Signature Version 4.
 */
func (s *S3) PresignGet(ctx context.Context, key, name string, ttl time.Duration) (string, error) {
	if key == "" || strings.HasPrefix(key, "/") || strings.Contains(key, "..") {
		return "", errors.New("clave S3 no válida")
	}
	if ttl < time.Second || ttl > 15*time.Minute {
		return "", errors.New("duración de descarga no válida")
	}
	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": name})

	req, err := s.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket:                     aws.String(s.bucket),
		Key:                        aws.String(key),
		ResponseContentType:        aws.String("application/pdf"),
		ResponseCacheControl:       aws.String("no-store"),
		ResponseContentDisposition: aws.String(disposition),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("no se pudo pre-firmar la descarga S3: %w", err)
	}
	return req.URL, nil
}

/** Ready comprueba que el bucket privado responda y esté operativo usando HeadBucket.
 * @returns {error} Error descriptivo si el almacenamiento no está listo.
 */
func (s *S3) Ready(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucket),
	})
	if err != nil {
		return fmt.Errorf("almacenamiento no disponible: %w", err)
	}
	return nil
}

/** EnsureBucket crea el bucket configurado si no existe aún (idempotente).
 * @param {context.Context} ctx - Contexto de ejecución.
 * @returns {error} Error si no se pudo asegurar la existencia del bucket.
 */
func (s *S3) EnsureBucket(ctx context.Context) error {
	if err := s.Ready(ctx); err == nil {
		return nil
	}
	_, err := s.client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(s.bucket),
	})
	if err != nil {
		var bae *s3types.BucketAlreadyExists
		var bao *s3types.BucketAlreadyOwnedByYou
		if errors.As(err, &bae) || errors.As(err, &bao) {
			return nil
		}
		return fmt.Errorf("no se pudo crear bucket S3: %w", err)
	}
	return nil
}
