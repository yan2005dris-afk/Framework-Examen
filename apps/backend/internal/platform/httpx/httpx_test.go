package httpx_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/httpx"
)

func TestCSRF_AllowedOrigins(t *testing.T) {
	origins := []string{"http://localhost:5173", "https://postulacion.biess.ec"}

	nextCalled := false
	handler := httpx.CSRF(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}), origins)

	// 1. GET sin Origin debe pasar
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	nextCalled = false
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !nextCalled {
		t.Fatalf("GET esperado 200, obtuve %d", rec.Code)
	}

	// 2. POST con origen permitido debe pasar
	req = httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec = httptest.NewRecorder()
	nextCalled = false
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !nextCalled {
		t.Fatalf("POST con origen permitido esperado 200, obtuve %d", rec.Code)
	}

	// 3. POST con origen malicioso debe responder 403
	req = httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Origin", "https://malicious-site.com")
	rec = httptest.NewRecorder()
	nextCalled = false
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden || nextCalled {
		t.Fatalf("POST malicioso esperado 403, obtuve %d", rec.Code)
	}

	var errResp map[string]httpx.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("error al parsear respuesta: %v", err)
	}
	if errResp["error"].Code != "origen_no_permitido" {
		t.Errorf("código de error esperado 'origen_no_permitido', obtuve %q", errResp["error"].Code)
	}

	// 4. POST con Sec-Fetch-Site cross-site debe responder 403
	req = httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	rec = httptest.NewRecorder()
	nextCalled = false
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden || nextCalled {
		t.Fatalf("POST cross-site esperado 403, obtuve %d", rec.Code)
	}
}

func TestReadyHandler_NoDatabase(t *testing.T) {
	router := httpx.NewRouter(httpx.RouterDependencies{
		Pool:    nil,
		Version: "1.0.0",
	})

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("readyz sin BD esperado 503, obtuve %d", rec.Code)
	}

	var resp struct {
		Status string              `json:"status"`
		Checks map[string]string   `json:"checks"`
		Error  httpx.ErrorResponse `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("error al deserializar respuesta: %v", err)
	}
	if resp.Status != "not_ready" {
		t.Errorf("status esperado 'not_ready', obtuve %q", resp.Status)
	}
	if resp.Checks["database"] != "no_configurada" {
		t.Errorf("check database esperado 'no_configurada', obtuve %q", resp.Checks["database"])
	}
}
