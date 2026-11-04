package storage

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestS3SimulationRoundTrip(t *testing.T) {
	var mu sync.Mutex
	objects := map[string][]byte{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		path := r.URL.Path
		switch r.Method {
		case "HEAD":
			w.WriteHeader(200)
		case "PUT":
			v, _ := io.ReadAll(io.LimitReader(r.Body, 4<<20))
			objects[path] = v
			w.WriteHeader(200)
		case "GET":
			v, ok := objects[path]
			if !ok {
				w.WriteHeader(404)
				return
			}
			_, _ = w.Write(v)
		case "DELETE":
			delete(objects, path)
			w.WriteHeader(204)
		default:
			w.WriteHeader(501)
		}
	}))
	defer server.Close()

	s, err := NewS3(S3Config{
		Endpoint:  server.URL + "/test-bucket",
		Region:    "us-east-1",
		AccessKey: "test-access",
		SecretKey: "test-secret",
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	key := "submission/5/roundtrip.pdf"
	payload := "%PDF-1.7\n%%EOF"
	if err := s.Save(ctx, key, strings.NewReader(payload)); err != nil {
		t.Fatal(err)
	}

	link, err := s.PresignGet(ctx, key, "prueba.pdf", 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", link, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal("no se pudo consultar el servidor simulado:", err)
	}
	data, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != 200 || string(data) != payload {
		t.Fatalf("la simulación no devolvió el archivo (status %d, contenido %s)", resp.StatusCode, string(data))
	}

	if err := s.Delete(ctx, key); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, key); err != nil {
		t.Fatal("DELETE no fue idempotente:", err)
	}
}
