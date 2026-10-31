package identity

import (
	"errors"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/cache"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/httpx"
	"net/http"
)

/** writeOperationalError distingue bloqueo y caída del control de seguridad de un OTP incorrecto. */
func writeOperationalError(w http.ResponseWriter, r *http.Request, err error) bool {
	switch {
	case errors.Is(err, cache.ErrUnavailable):
		httpx.RespondError(w, r, 503, "seguridad_no_disponible", "La autenticación no está disponible temporalmente.")
	case errors.Is(err, ErrDeliveryUnavailable):
		httpx.RespondError(w, r, 503, "notificacion_no_disponible", ErrDeliveryUnavailable.Error())
	case errors.Is(err, cache.ErrResendLimited):
		w.Header().Set("Retry-After", "60")
		httpx.RespondError(w, r, 429, "reenvio_limitado", cache.ErrResendLimited.Error())
	case errors.Is(err, cache.ErrOTPLocked), errors.Is(err, ErrAccountLocked):
		w.Header().Set("Retry-After", "600")
		httpx.RespondError(w, r, 429, "intentos_excedidos", ErrAccountLocked.Error())
	default:
		return false
	}
	return true
}
