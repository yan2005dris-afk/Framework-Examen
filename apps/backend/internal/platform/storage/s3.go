package storage

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

/** S3Config configura un bucket privado con credenciales inyectadas por entorno.
 * Endpoint incluye el bucket: https://bucket.s3.region.amazonaws.com o
 * http://minio:9000/bucket. PublicEndpoint permite otro host para las descargas.
 */
type S3Config struct{ Endpoint, PublicEndpoint, Region, AccessKey, SecretKey, SessionToken string }

/** S3 implementa las tres operaciones necesarias con Signature Version 4.
 * La firma sigue la especificación y el vector oficial de AWS; no requiere SDK.
 */
type S3 struct {
	config S3Config
	client *http.Client
	now    func() time.Time
}

/** NewS3 valida configuración sin enviar credenciales durante el arranque.
 * @returns {S3, error} Adaptador para un bucket ya aprovisionado y privado.
 */
func NewS3(config S3Config) (*S3, error) {
	if config.PublicEndpoint == "" {
		config.PublicEndpoint = config.Endpoint
	}
	for _, endpoint := range []string{config.Endpoint, config.PublicEndpoint} {
		u, err := url.Parse(endpoint)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return nil, errors.New("endpoint S3 no válido")
		}
	}
	if config.AccessKey == "" || config.SecretKey == "" || config.Region == "" {
		return nil, errors.New("S3 requiere región y credenciales")
	}
	return &S3{config: config, client: &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, now: time.Now}, nil
}

/** Save sube un PDF sin ACL pública y sin incluir su nombre en la clave.
 * @returns {error} Fallo remoto sin filtrar URL firmada ni cuerpo de error.
 */
func (s *S3) Save(ctx context.Context, key string, content io.Reader) error {
	return s.request(ctx, http.MethodPut, key, content)
}

/** Delete es idempotente, como DeleteObject de S3.
 * @returns {error} Fallo HTTP o transporte; la cola mantendrá la baja pendiente.
 */
func (s *S3) Delete(ctx context.Context, key string) error {
	return s.request(ctx, http.MethodDelete, key, nil)
}

func (s *S3) request(ctx context.Context, method, key string, body io.Reader) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	u, err := s.presign(s.config.Endpoint, method, key, nil, 5*time.Minute)
	if err != nil {
		return err
	}
	r, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return errors.New("no se pudo crear la solicitud S3")
	}
	if method == http.MethodPut {
		r.Header.Set("Content-Type", "application/pdf")
	}
	resp, err := s.client.Do(r)
	if err != nil {
		return errors.New("no se pudo contactar con el almacenamiento S3")
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 8192))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("el almacenamiento S3 respondió HTTP %d", resp.StatusCode)
	}
	return nil
}

/** PresignGet firma la descarga de cinco minutos con nombre seguro y sin caché.
 * @returns {string, error} URL directa al bucket accesible por el navegador.
 */
func (s *S3) PresignGet(ctx context.Context, key, name string, ttl time.Duration) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if ttl < time.Second || ttl > 15*time.Minute {
		return "", errors.New("duración de descarga no válida")
	}
	q := url.Values{"response-content-type": {"application/pdf"}, "response-cache-control": {"no-store"}, "response-content-disposition": {mime.FormatMediaType("attachment", map[string]string{"filename": name})}}
	return s.presign(s.config.PublicEndpoint, http.MethodGet, key, q, ttl)
}

// awsEncode codifica cada byte con las reglas RFC 3986 exigidas por S3.
func awsEncode(value string, slash bool) string {
	var b strings.Builder
	const digits = "0123456789ABCDEF"
	for i := 0; i < len(value); i++ {
		c := value[i]
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("-_.~", rune(c)) || c == '/' && !slash {
			b.WriteByte(c)
		} else {
			b.WriteByte('%')
			b.WriteByte(digits[c>>4])
			b.WriteByte(digits[c&15])
		}
	}
	return b.String()
}

func awsQuery(q url.Values) string { return strings.ReplaceAll(q.Encode(), "+", "%20") }
func hmacBytes(key []byte, value string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(value))
	return h.Sum(nil)
}

func (s *S3) presign(endpoint, method, key string, q url.Values, ttl time.Duration) (string, error) {
	if key == "" || strings.HasPrefix(key, "/") || strings.Contains(key, "\\") {
		return "", errors.New("clave S3 no válida")
	}
	for _, part := range strings.Split(key, "/") {
		if part == ".." || part == "." {
			return "", errors.New("clave S3 no válida")
		}
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/" + key
	u.RawPath = awsEncode(u.Path, false)
	if q == nil {
		q = make(url.Values)
	}
	now := s.now().UTC()
	date := now.Format("20060102")
	stamp := now.Format("20060102T150405Z")
	scope := date + "/" + s.config.Region + "/s3/aws4_request"
	q.Set("X-Amz-Algorithm", "AWS4-HMAC-SHA256")
	q.Set("X-Amz-Credential", s.config.AccessKey+"/"+scope)
	q.Set("X-Amz-Date", stamp)
	q.Set("X-Amz-Expires", strconv.FormatInt(int64(ttl/time.Second), 10))
	q.Set("X-Amz-SignedHeaders", "host")
	if s.config.SessionToken != "" {
		q.Set("X-Amz-Security-Token", s.config.SessionToken)
	}
	canonical := method + "\n" + u.EscapedPath() + "\n" + awsQuery(q) + "\nhost:" + u.Host + "\n\nhost\nUNSIGNED-PAYLOAD"
	sum := sha256.Sum256([]byte(canonical))
	toSign := "AWS4-HMAC-SHA256\n" + stamp + "\n" + scope + "\n" + hex.EncodeToString(sum[:])
	k := hmacBytes([]byte("AWS4"+s.config.SecretKey), date)
	k = hmacBytes(k, s.config.Region)
	k = hmacBytes(k, "s3")
	k = hmacBytes(k, "aws4_request")
	q.Set("X-Amz-Signature", hex.EncodeToString(hmacBytes(k, toSign)))
	u.RawQuery = awsQuery(q)
	return u.String(), nil
}
