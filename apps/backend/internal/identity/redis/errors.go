// Package redis adapta la caché de Redis a los puertos del contexto de
// identidad.
//
// El dominio (internal/identity) no conoce la caché: declara los puertos
// AttemptsStore y OTPStore, y este adaptador los implementa sobre la
// infraestructura, traduciendo sus fallos a errores de dominio (ADR-002,
// regla 3). Ningún tipo de la caché cruza el borde: el dominio recibe
// desafíos, contadores y errores propios.
package redis

import (
	"errors"

	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/cache"
)

/**
 * translate convierte un fallo de la caché en un error del dominio. El
 * vencimiento que acompaña a un error de reintento se conserva para que la
 * capa HTTP pueda responder con Retry-After sin consultar la infraestructura.
 *
 * @param {error} err - ErrorDevuelto por la infraestructura.
 * @returns {error} Error del dominio equivalente o el original si no hay traducción.
 */
func translate(err error) error {
	if err == nil {
		return nil
	}
	var retry *cache.RetryError
	if errors.As(err, &retry) {
		return &identity.RetryError{Cause: translate(retry.Cause), RetryAfter: retry.RetryAfter}
	}
	switch {
	case errors.Is(err, cache.ErrUnavailable):
		return identity.ErrUnavailable
	case errors.Is(err, cache.ErrOTPLocked):
		return identity.ErrOTPLocked
	case errors.Is(err, cache.ErrResendLimited):
		return identity.ErrResendLimited
	}
	return err
}
