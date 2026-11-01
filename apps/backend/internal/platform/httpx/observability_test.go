package httpx_test

import (
	"bytes"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/httpx"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLogsUseRouteTemplateAndSafeRequestID(t *testing.T) {
	var output bytes.Buffer
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
	defer slog.SetDefault(old)
	r := chi.NewRouter()
	r.Get("/users/{id}", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	h := httpx.RequestID(httpx.Logging(r))
	req := httptest.NewRequest("GET", "/users/cedula-privada?correo=privado@example.test", nil)
	req.Header.Set("X-Request-ID", "privado@example.test")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if _, e := uuid.Parse(w.Header().Get("X-Request-ID")); e != nil {
		t.Fatal(e)
	}
	if strings.Contains(output.String(), "privado") || strings.Contains(output.String(), "cedula-") || !strings.Contains(output.String(), "/users/{id}") {
		t.Fatal(output.String())
	}
	if w.Header().Get("Traceparent") == "" {
		t.Fatal("sin contexto de traza")
	}
}

func TestDeadlineCancelsDownstreamContext(t *testing.T) {
	observed := false
	h := httpx.Deadline(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
		observed = true
		w.WriteHeader(503)
	}), time.Millisecond)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if !observed || w.Code != 503 {
		t.Fatal("no se propagó el límite de tiempo")
	}
}
