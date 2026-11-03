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
	req.AddCookie(&http.Cookie{Name: "session", Value: "presente"})
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
	req.AddCookie(&http.Cookie{Name: "session", Value: "presente"})
	rec = httptest.NewRecorder()
	nextCalled = false
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden || nextCalled {
		t.Fatalf("POST cross-site esperado 403, obtuve %d", rec.Code)
	}
}

func TestCSRF_OriginVerification(t *testing.T) {
	nextCalled := false
	handler := httpx.CSRF(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}), nil)

	// 1. POST con cookie y sin Origin ni Sec-Fetch-Site no tiene origen verificable.
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "presente"})
	rec := httptest.NewRecorder()
	nextCalled = false
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden || nextCalled {
		t.Fatalf("POST sin señal de origen esperado 403, obtuve %d", rec.Code)
	}

	var errResp map[string]httpx.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("error al parsear respuesta: %v", err)
	}
	if errResp["error"].Code != "origen_no_verificable" {
		t.Errorf("código de error esperado 'origen_no_verificable', obtuve %q", errResp["error"].Code)
	}

	// 1b. Sin cookie no hay credenciales ambientales que falsificar: la
	// petición sigue su curso y el control de sesión responderá 401.
	req = httptest.NewRequest(http.MethodPost, "/test", nil)
	rec = httptest.NewRecorder()
	nextCalled = false
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !nextCalled {
		t.Fatalf("POST sin cookie esperado 200, obtuve %d", rec.Code)
	}

	// 2. El mismo origen se acepta aunque la lista blanca esté vacía.
	req = httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	req.AddCookie(&http.Cookie{Name: "session", Value: "presente"})
	rec = httptest.NewRecorder()
	nextCalled = false
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !nextCalled {
		t.Fatalf("POST del mismo origen esperado 200, obtuve %d", rec.Code)
	}

	// 3. Detrás de un proxy se acepta el esquema reenviado.
	req = httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Host = "concurso.biess.gob.ec"
	req.Header.Set("Origin", "https://concurso.biess.gob.ec")
	req.Header.Set("X-Forwarded-Proto", "https")
	req.AddCookie(&http.Cookie{Name: "session", Value: "presente"})
	rec = httptest.NewRecorder()
	nextCalled = false
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !nextCalled {
		t.Fatalf("POST del mismo origen tras proxy esperado 200, obtuve %d", rec.Code)
	}

	// 4. Sec-Fetch-Site same-origin sin Origin sigue siendo verificable.
	req = httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.AddCookie(&http.Cookie{Name: "session", Value: "presente"})
	rec = httptest.NewRecorder()
	nextCalled = false
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !nextCalled {
		t.Fatalf("POST con Sec-Fetch-Site same-origin esperado 200, obtuve %d", rec.Code)
	}

	// 5. Un origen ajeno con lista blanca vacía se rechaza.
	req = httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Origin", "https://malicious-site.com")
	req.AddCookie(&http.Cookie{Name: "session", Value: "presente"})
	rec = httptest.NewRecorder()
	nextCalled = false
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden || nextCalled {
		t.Fatalf("POST de origen ajeno esperado 403, obtuve %d", rec.Code)
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
