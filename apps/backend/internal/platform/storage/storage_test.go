package storage

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLocalSaveCancellationAndTraversal(t *testing.T) {
	local, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := local.Save(ctx, "postulacion/5/respaldo.pdf", strings.NewReader("%PDF-1.7")); err != nil {
		t.Fatal(err)
	}
	f, err := local.Get(ctx, "postulacion/5/respaldo.pdf")
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(f)
	f.Close()
	if string(data) != "%PDF-1.7" {
		t.Fatal("contenido corrupto")
	}
	for _, key := range []string{"../fuera", "/absoluto", "C:/fuera", "a\\..\\fuera", "a/../../fuera", ".", "a/./b", "a//b"} {
		if err := local.Save(ctx, key, strings.NewReader("datos")); err == nil {
			t.Fatalf("aceptó recorrido %q", key)
		}
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if err := local.Save(canceled, "postulacion/5/nuevo.pdf", strings.NewReader("%PDF-1.7")); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Join(local.base, "postulacion", "5"))
	if err != nil || len(entries) != 1 {
		t.Fatal("quedó archivo parcial", err)
	}
	if err := local.Delete(ctx, "postulacion/5/respaldo.pdf"); err != nil {
		t.Fatal(err)
	}
	if err := local.Delete(ctx, "postulacion/5/respaldo.pdf"); err != nil {
		t.Fatal("baja no idempotente", err)
	}
}

func TestLocalCannotFollowSymlinkOutsideRoot(t *testing.T) {
	base, outside := t.TempDir(), t.TempDir()
	local, _ := NewLocal(base)
	if err := os.Symlink(outside, filepath.Join(base, "escape")); err != nil {
		t.Skip("el sistema no permite crear enlaces simbólicos")
	}
	if err := local.Save(context.Background(), "escape/archivo.pdf", strings.NewReader("datos")); err == nil {
		t.Fatal("escribió fuera de la raíz")
	}
}

func TestLocalSignedDownloadExpiryTamperingAndNoListing(t *testing.T) {
	store, err := NewSignedLocal(t.TempDir(), "http://objects.example.test", strings.Repeat("k", 32))
	if err != nil {
		t.Fatal(err)
	}
	clock := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return clock }
	if err := store.Save(context.Background(), "postulacion/5/random.pdf", strings.NewReader("%PDF-1.7")); err != nil {
		t.Fatal(err)
	}
	signed, err := store.PresignGet(context.Background(), "postulacion/5/random.pdf", "título.pdf", 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	perform := func(link string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		store.ServeHTTP(w, httptest.NewRequest("GET", link, nil))
		return w
	}
	w := perform(signed)
	if w.Code != 200 || w.Body.String() != "%PDF-1.7" || w.Header().Get("Cache-Control") != "no-store" || !strings.HasPrefix(w.Header().Get("Content-Disposition"), "attachment") {
		t.Fatal("descarga incorrecta", w.Code, w.Body.String())
	}
	for _, mutate := range []func(*url.URL){func(u *url.URL) { u.Path = "/objects/postulacion/5/otro.pdf" }, func(u *url.URL) { q := u.Query(); q.Set("name", "otro.pdf"); u.RawQuery = q.Encode() }, func(u *url.URL) { q := u.Query(); q.Set("expires", "9999999999"); u.RawQuery = q.Encode() }} {
		u, _ := url.Parse(signed)
		mutate(u)
		if got := perform(u.String()); got.Code != 403 {
			t.Fatal("aceptó alteración", got.Code)
		}
	}
	clock = clock.Add(5 * time.Minute)
	if got := perform(signed); got.Code != 403 {
		t.Fatal("URL no expiró")
	}
	if got := perform("http://objects.example.test/objects/postulacion/"); got.Code != 403 {
		t.Fatal("directorio accesible")
	}
}

// Vector publicado en https://docs.aws.amazon.com/AmazonS3/latest/API/sigv4-query-string-auth.html
func TestS3SignatureMatchesOfficialAWSVector(t *testing.T) {
	s, err := NewS3(S3Config{Endpoint: "https://examplebucket.s3.amazonaws.com", Region: "us-east-1", AccessKey: "AKIAIOSFODNN7EXAMPLE", SecretKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"})
	if err != nil {
		t.Fatal(err)
	}
	s.now = func() time.Time { return time.Date(2013, 5, 24, 0, 0, 0, 0, time.UTC) }
	link, err := s.presign(s.config.Endpoint, "GET", "test.txt", nil, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(link)
	if u.Query().Get("X-Amz-Signature") != "aeeed9bbccd4d02ee5c0109b86d86835f995330da4c265957d157751f604d404" {
		t.Fatal("la firma difiere del vector oficial")
	}
}

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestS3HTTPRequestsAndSanitizedFailures(t *testing.T) {
	s, err := NewS3(S3Config{Endpoint: "http://minio.test:9000/private", PublicEndpoint: "https://files.example.test/private", Region: "us-east-1", AccessKey: "test", SecretKey: "test-secret", SessionToken: "test-session"})
	if err != nil {
		t.Fatal(err)
	}
	status := 200
	calls := []string{}
	s.client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
		calls = append(calls, r.Method)
		if r.URL.Path != "/private/submission/5/file.pdf" || r.URL.Query().Get("X-Amz-Signature") == "" || r.URL.Query().Get("X-Amz-Security-Token") != "test-session" {
			t.Fatal("solicitud sin firmar")
		}
		if r.Method == "PUT" {
			data, _ := io.ReadAll(r.Body)
			if string(data) != "%PDF-1.7" || r.Header.Get("Content-Type") != "application/pdf" {
				t.Fatal("carga incorrecta")
			}
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader("sensitive-error-body")), Header: make(http.Header)}, nil
	})
	ctx := context.Background()
	if err := s.Save(ctx, "submission/5/file.pdf", strings.NewReader("%PDF-1.7")); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, "submission/5/file.pdf"); err != nil {
		t.Fatal(err)
	}
	status = 503
	if err := s.Save(ctx, "submission/5/file.pdf", strings.NewReader("%PDF-1.7")); err == nil || strings.Contains(err.Error(), "sensitive") || strings.Contains(err.Error(), "X-Amz") {
		t.Fatal("error sin sanear", err)
	}
	link, err := s.PresignGet(ctx, "submission/5/file.pdf", "documento.pdf", 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(link)
	if u.Host != "files.example.test" || u.Query().Get("X-Amz-Expires") != "300" || u.Query().Get("response-cache-control") != "no-store" {
		t.Fatal("enlace público incorrecto")
	}
	if len(calls) != 3 {
		t.Fatal("faltaron solicitudes")
	}
}
