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

func TestEnsureBucket(t *testing.T) {
	t.Run("bucket ya existe", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "HEAD" {
				w.WriteHeader(http.StatusOK)
				return
			}
			t.Errorf("no esperaba método %s cuando HEAD responde 200", r.Method)
		}))
		defer server.Close()

		s, err := NewS3(S3Config{
			Endpoint:  server.URL + "/existing-bucket",
			Region:    "us-east-1",
			AccessKey: "test",
			SecretKey: "test",
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := s.EnsureBucket(context.Background()); err != nil {
			t.Fatalf("error inesperado en EnsureBucket: %v", err)
		}
	})

	t.Run("crea bucket si no existe", func(t *testing.T) {
		created := false
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "HEAD" {
				if created {
					w.WriteHeader(http.StatusOK)
				} else {
					w.WriteHeader(http.StatusNotFound)
				}
				return
			}
			if r.Method == "PUT" {
				created = true
				w.WriteHeader(http.StatusOK)
				return
			}
			t.Errorf("método no esperado: %s", r.Method)
		}))
		defer server.Close()

		s, err := NewS3(S3Config{
			Endpoint:  server.URL + "/new-bucket",
			Region:    "us-east-1",
			AccessKey: "test",
			SecretKey: "test",
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := s.EnsureBucket(context.Background()); err != nil {
			t.Fatalf("error al crear bucket: %v", err)
		}
		if !created {
			t.Fatal("no se ejecutó PUT para crear el bucket")
		}
	})
}
