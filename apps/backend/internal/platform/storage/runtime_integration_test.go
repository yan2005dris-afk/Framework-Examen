package storage

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

/** TestS3RuntimeRoundTrip prueba operaciones reales contra RustFS o S3 compatible. */
func TestS3RuntimeRoundTrip(t *testing.T) {
	endpoint := os.Getenv("STORAGE_TEST_ENDPOINT")
	if endpoint == "" {
		endpoint = os.Getenv("S3_ENDPOINT")
	}
	if endpoint == "" {
		t.Skip("defina STORAGE_TEST_ENDPOINT o S3_ENDPOINT para el bucket de pruebas RustFS/S3")
	}

	accessKey := os.Getenv("STORAGE_TEST_ACCESS_KEY")
	if accessKey == "" {
		accessKey = os.Getenv("S3_ACCESS_KEY")
	}
	secretKey := os.Getenv("STORAGE_TEST_SECRET_KEY")
	if secretKey == "" {
		secretKey = os.Getenv("S3_SECRET_KEY")
	}

	s, err := NewS3(S3Config{
		Endpoint:  endpoint,
		Region:    "us-east-1",
		AccessKey: accessKey,
		SecretKey: secretKey,
	})
	if err != nil {
		t.Fatalf("configuración de almacenamiento de pruebas inválida: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := s.Ready(ctx); err != nil {
		t.Skipf("bucket no disponible para pruebas: %v", err)
	}

	key := "test-" + uuid.NewString() + ".pdf"
	payload := "%PDF-1.7\nprueba sin datos personales\n%%EOF"
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.Delete(ctx, key)
	})

	if err := s.Save(ctx, key, strings.NewReader(payload)); err != nil {
		t.Fatalf("no se pudo cargar el objeto: %v", err)
	}

	link, err := s.PresignGet(ctx, key, "prueba.pdf", time.Minute)
	if err != nil {
		t.Fatalf("no se pudo firmar la descarga: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", link, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("no se pudo descargar el objeto con la presigned URL: %v", err)
	}
	data, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil || resp.StatusCode != 200 || string(data) != payload {
		t.Fatalf("el almacenamiento no conservó los bytes (status: %d)", resp.StatusCode)
	}

	if err := s.Delete(ctx, key); err != nil {
		t.Fatalf("no se pudo borrar el objeto de prueba: %v", err)
	}
	if err := s.Delete(ctx, key); err != nil {
		t.Fatalf("la baja no es idempotente: %v", err)
	}
}
