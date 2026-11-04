package storage

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestNewS3Validation(t *testing.T) {
	// Endpoint sin bucket
	_, err := NewS3(S3Config{
		Endpoint:  "http://localhost:9000",
		Region:    "us-east-1",
		AccessKey: "key",
		SecretKey: "secret",
	})
	if err == nil {
		t.Fatal("debió rechazar endpoint sin bucket")
	}

	// Sin credenciales
	_, err = NewS3(S3Config{
		Endpoint: "http://localhost:9000/bucket",
		Region:   "us-east-1",
	})
	if err == nil {
		t.Fatal("debió exigir credenciales")
	}

	// Válido
	s, err := NewS3(S3Config{
		Endpoint:       "http://rustfs:9000/biess",
		PublicEndpoint: "http://localhost:9000/biess",
		Region:         "us-east-1",
		AccessKey:      "rustfsadmin",
		SecretKey:      "rustfsadmin",
	})
	if err != nil {
		t.Fatalf("configuración válida falló: %v", err)
	}
	if s.bucket != "biess" {
		t.Fatalf("bucket esperado biess, obtenido %s", s.bucket)
	}
}

func TestPresignURLFormat(t *testing.T) {
	s, err := NewS3(S3Config{
		Endpoint:       "http://rustfs:9000/biess",
		PublicEndpoint: "http://localhost:9000/biess",
		Region:         "us-east-1",
		AccessKey:      "rustfsadmin",
		SecretKey:      "rustfsadmin",
	})
	if err != nil {
		t.Fatal(err)
	}

	link, err := s.PresignGet(context.Background(), "01a1/4/cv.pdf", "curriculum.pdf", 5*time.Minute)
	if err != nil {
		t.Fatalf("PresignGet falló: %v", err)
	}

	u, err := url.Parse(link)
	if err != nil {
		t.Fatalf("URL inválida: %v", err)
	}

	if !strings.Contains(u.Host, "9000") {
		t.Errorf("host esperado con puerto 9000, obtenido %s", u.Host)
	}
	if !strings.HasPrefix(u.Path, "/biess/01a1/4/cv.pdf") {
		t.Errorf("ruta esperada con bucket y clave, obtenida %s", u.Path)
	}
	q := u.Query()
	if q.Get("X-Amz-Signature") == "" {
		t.Error("falta firma X-Amz-Signature")
	}
	if q.Get("X-Amz-Expires") != "300" {
		t.Errorf("expiración esperada 300s, obtenida %s", q.Get("X-Amz-Expires"))
	}
	if !strings.Contains(q.Get("response-content-disposition"), "curriculum.pdf") {
		t.Errorf("disposition incorrecta: %s", q.Get("response-content-disposition"))
	}
}
