package identity

import (
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/cache"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOperationalErrorUsesRemainingServerTTL(t *testing.T) {
	r := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
	w := httptest.NewRecorder()
	if !writeOperationalError(w, r, &cache.RetryError{Cause: ErrAccountLocked, RetryAfter: 27 * time.Second}) || w.Code != 429 || w.Header().Get("Retry-After") != "27" {
		t.Fatal(w.Code, w.Header())
	}
	expiry, err := time.Parse(time.RFC3339, w.Header().Get("X-Retry-At"))
	if err != nil || time.Until(expiry) > 27*time.Second || time.Until(expiry) < 25*time.Second {
		t.Fatal(expiry, err)
	}
}
