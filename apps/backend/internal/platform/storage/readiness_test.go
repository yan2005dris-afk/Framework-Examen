package storage

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestBucketHeadDoesNotRequirePathStyle(t *testing.T) {
	s, e := NewS3(S3Config{Endpoint: "https://bucket.s3.example.test", Region: "us-east-1", AccessKey: "test", SecretKey: "test"})
	if e != nil {
		t.Fatal(e)
	}
	signed, e := s.presignRequest(s.config.Endpoint, "HEAD", "", nil, time.Minute, true)
	if e != nil {
		t.Fatal(e)
	}
	u, e := url.Parse(signed)
	if e != nil || u.Path != "/" {
		t.Fatal("HEAD virtual-host inválido")
	}
	if _, e = s.presign(s.config.Endpoint, "GET", "", nil, time.Minute); e == nil {
		t.Fatal("GET de clave vacía permitido")
	}
}
func TestStorageReadinessUsesSignedHeadWithoutObjects(t *testing.T) {
	for _, status := range []int{200, 403, 404, 503} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "HEAD" || r.URL.Path != "/private-bucket" || r.URL.Query().Get("X-Amz-Signature") == "" {
					t.Error("sonda incorrecta")
				}
				w.WriteHeader(status)
			}))
			defer server.Close()
			s, e := NewS3(S3Config{Endpoint: server.URL + "/private-bucket", Region: "us-east-1", AccessKey: "test", SecretKey: "test"})
			if e != nil {
				t.Fatal(e)
			}
			e = s.Ready(context.Background())
			if (e == nil) != (status == 200) {
				t.Fatal("readiness no refleja respuesta", status)
			}
		})
	}
}
