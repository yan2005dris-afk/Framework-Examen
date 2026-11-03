package storage

import (
	"context"
	"github.com/google/uuid"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

/** TestS3RuntimeRoundTrip usa sólo un bucket de pruebas explícito en RustFS/S3. */
func TestS3RuntimeRoundTrip(t *testing.T) {
	endpoint := os.Getenv("STORAGE_TEST_ENDPOINT")
	if endpoint == "" {
		t.Skip("defina STORAGE_TEST_ENDPOINT para el bucket de pruebas RustFS/S3")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	s, err := NewS3(S3Config{Endpoint: endpoint, Region: "us-east-1", AccessKey: os.Getenv("STORAGE_TEST_ACCESS_KEY"), SecretKey: os.Getenv("STORAGE_TEST_SECRET_KEY")})
	if err != nil {
		t.Fatal("configuración de almacenamiento de pruebas inválida")
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	bucket := strings.Trim(u.Path, "/")
	if bucket == "" || strings.Contains(bucket, "/") {
		t.Fatal("la prueba requiere un bucket propio de un solo segmento")
	}
	create, err := s.presign(u.Scheme+"://"+u.Host, "PUT", bucket, nil, time.Minute)
	if err != nil {
		t.Fatal("no se pudo firmar el alta del bucket de pruebas")
	}
	request, err := http.NewRequestWithContext(ctx, "PUT", create, nil)
	if err != nil {
		t.Fatal("petición de bucket inválida")
	}
	response, err := s.client.Do(request)
	if err != nil {
		t.Fatal("no se pudo crear el bucket de pruebas")
	}
	_ = response.Body.Close()
	if response.StatusCode != 200 && response.StatusCode != 409 {
		t.Fatalf("alta de bucket: HTTP %d", response.StatusCode)
	}
	key := "test-" + uuid.NewString() + ".pdf"
	payload := "%PDF-1.7\nprueba sin datos personales\n%%EOF"
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.Delete(ctx, key)
	})
	if err := s.Save(ctx, key, strings.NewReader(payload)); err != nil {
		t.Fatal("no se pudo cargar el objeto")
	}
	link, err := s.PresignGet(ctx, key, "prueba.pdf", time.Minute)
	if err != nil {
		t.Fatal("no se pudo firmar la descarga")
	}
	request, err = http.NewRequestWithContext(ctx, "GET", link, nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err = s.client.Do(request)
	if err != nil {
		t.Fatal("no se pudo descargar el objeto")
	}
	data, err := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if err != nil || response.StatusCode != 200 || string(data) != payload {
		t.Fatal("el almacenamiento no conservó los bytes")
	}
	if err := s.Delete(ctx, key); err != nil {
		t.Fatal("no se pudo borrar el objeto de prueba")
	}
	if err := s.Delete(ctx, key); err != nil {
		t.Fatal("la baja no es idempotente")
	}
}
