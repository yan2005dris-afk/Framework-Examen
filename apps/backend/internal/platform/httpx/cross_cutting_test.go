package httpx_test

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/httpx"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCSRFAllMutationsFailClosedWithoutExactOrigin(t *testing.T) {
	for _, method := range []string{"POST", "PUT", "PATCH", "DELETE"} {
		for _, origin := range []string{"", "null", "https://portal.test.evil", "https://portal.test/", "https://evil.test"} {
			t.Run(method+origin, func(t *testing.T) {
				called := false
				h := httpx.CSRF(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }), []string{"https://portal.test"})
				r := httptest.NewRequest(method, "/", nil)
				r.AddCookie(&http.Cookie{Name: "session", Value: "activa"})
				if origin != "" {
					r.Header.Set("Origin", origin)
				}
				r.Header.Set("Referer", "https://portal.test/")
				r.Header.Set("Sec-Fetch-Site", "cross-site")
				w := httptest.NewRecorder()
				h.ServeHTTP(w, r)
				if w.Code != 403 || called {
					t.Fatal("mutación sin origen confiable", w.Code)
				}
			})
		}
	}
	h := httpx.CSRF(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) }), nil)
	r := httptest.NewRequest("POST", "/", nil)
	r.AddCookie(&http.Cookie{Name: "session", Value: "activa"})
	r.Header.Set("Origin", "https://portal.test")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("lista vacía permitió escritura")
	}
}
func TestDependencyReadinessReportsRequiredAndOptionalFailures(t *testing.T) {
	for _, required := range []bool{true, false} {
		t.Run(map[bool]string{true: "obligatoria", false: "opcional"}[required], func(t *testing.T) {
			checks := map[string]httpx.Dependency{"database": {Required: true, Probe: func(context.Context) error { return nil }}, "redis": {Required: required, Probe: func(context.Context) error { return errors.New("NO-REVELAR-credenciales") }}}
			r := httpx.NewRouter(httpx.RouterDependencies{Dependencies: checks})
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest("GET", "/readyz", nil))
			want := 200
			if required {
				want = 503
			}
			if w.Code != want {
				t.Fatal(w.Code, want)
			}
			var body struct{ Checks map[string]string }
			if json.Unmarshal(w.Body.Bytes(), &body) != nil || body.Checks["redis"] != "no_disponible" {
				t.Fatal(w.Body.String())
			}
			w = httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest("GET", "/healthz", nil))
			if w.Code != 200 {
				t.Fatal("liveness depende del proveedor")
			}
		})
	}
}
func TestReadinessChecksHaveBoundedContext(t *testing.T) {
	r := httpx.NewRouter(httpx.RouterDependencies{Dependencies: map[string]httpx.Dependency{"database": {Required: true, Probe: func(ctx context.Context) error {
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > 3*time.Second {
			t.Fatal("sonda sin plazo")
		}
		return nil
	}}}})
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/readyz", nil))
}

func TestCORSAllowsIdempotencyAndExposesRetryMetadata(t *testing.T) {
	h := httpx.CORS(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }), []string{"https://portal.test"})
	r := httptest.NewRequest("OPTIONS", "/", nil)
	r.Header.Set("Origin", "https://portal.test")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if !strings.Contains(w.Header().Get("Access-Control-Allow-Headers"), "Idempotency-Key") || !strings.Contains(w.Header().Get("Access-Control-Expose-Headers"), "Retry-After") {
		t.Fatal("contrato CORS incompleto")
	}
}
