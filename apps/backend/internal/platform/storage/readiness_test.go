package storage

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStorageReadinessUsesHeadBucket(t *testing.T) {
	for _, status := range []int{200, 403, 404, 503} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "HEAD" {
					t.Errorf("esperado HEAD, obtenido %s", r.Method)
				}
				w.WriteHeader(status)
			}))
			defer server.Close()

			s, err := NewS3(S3Config{
				Endpoint:  server.URL + "/private-bucket",
				Region:    "us-east-1",
				AccessKey: "test",
				SecretKey: "test",
			})
			if err != nil {
				t.Fatal(err)
			}
			err = s.Ready(context.Background())
			if (err == nil) != (status == 200) {
				t.Errorf("readiness con status %d: error inesperado: %v", status, err)
			}
		})
	}
}
