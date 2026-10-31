package storage

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// Ejecutar la simulación Postman document-storage y apuntar S3_SIMULATION_ENDPOINT
// a http://127.0.0.1:4011/s3/bucket. Nunca usar un bucket real en esta prueba.
func TestS3SimulationRoundTrip(t *testing.T) {
	endpoint := os.Getenv("S3_SIMULATION_ENDPOINT")
	if endpoint == "" {
		t.Skip("defina S3_SIMULATION_ENDPOINT para probar el doble HTTP de Postman")
	}
	s, err := NewS3(S3Config{Endpoint: endpoint, Region: "us-east-1", AccessKey: "simulation", SecretKey: "simulation-only"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	key := "submission/5/roundtrip.pdf"
	if err := s.Save(ctx, key, strings.NewReader("%PDF-1.7\n%%EOF")); err != nil {
		t.Fatal(err)
	}
	link, err := s.PresignGet(ctx, key, "prueba.pdf", 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	r, err := http.NewRequestWithContext(ctx, "GET", link, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := s.client.Do(r)
	if err != nil {
		t.Fatal("no se pudo consultar el doble HTTP")
	}
	data, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 || string(data) != "%PDF-1.7\n%%EOF" {
		t.Fatal("la simulación no conservó el archivo")
	}
	if err := s.Delete(ctx, key); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, key); err != nil {
		t.Fatal("DELETE no fue idempotente", err)
	}
}
