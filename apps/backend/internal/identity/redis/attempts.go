package redis

import (
	"context"
	"time"

	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/cache"
)

/**
 * AttemptsStore implementa el puerto de intentos del contexto de identidad
 * sobre la caché de Redis.
 */
type AttemptsStore struct {
	store *cache.AttemptsStore
}

// El adaptador satisface el puerto declarado por el dominio.
var _ identity.AttemptsStore = (*AttemptsStore)(nil)

/**
 * NewAttemptsStore envuelve el almacén de intentos de la infraestructura.
 *
 * @param {*cache.AttemptsStore} store - Almacén de intentos ya configurado.
 * @returns {*AttemptsStore} Adaptador listo para el dominio.
 */
func NewAttemptsStore(store *cache.AttemptsStore) *AttemptsStore {
	return &AttemptsStore{store: store}
}

/**
 * IsLocked indica si la cuenta está bloqueada por exceso de intentos.
 *
 * @param {context.Context} ctx - Contexto de la petición.
 * @param {string} identifier - Identificador de la cuenta.
 * @returns {bool} True si la cuenta está bloqueada.
 * @returns {error} ErrUnavailable si el control de seguridad no responde.
 */
func (s *AttemptsStore) IsLocked(ctx context.Context, identifier string) (bool, error) {
	if s == nil || s.store == nil {
		return false, identity.ErrUnavailable
	}
	locked, err := s.store.IsLocked(ctx, identifier)
	if err != nil {
		return false, translate(err)
	}
	return locked, nil
}

/**
 * RecordFailure acumula un intento fallido y devuelve el total.
 *
 * @param {context.Context} ctx - Contexto de la petición.
 * @param {string} identifier - Identificador de la cuenta.
 * @returns {int64} Número de intentos acumulados tras este fallo.
 * @returns {error} ErrUnavailable si el control de seguridad no responde.
 */
func (s *AttemptsStore) RecordFailure(ctx context.Context, identifier string) (int64, error) {
	if s == nil || s.store == nil {
		return 0, identity.ErrUnavailable
	}
	count, err := s.store.RecordFailure(ctx, identifier)
	if err != nil {
		return 0, translate(err)
	}
	return count, nil
}

/**
 * ResetAttempts borra el contador tras un acceso exitoso.
 *
 * @param {context.Context} ctx - Contexto de la petición.
 * @param {string} identifier - Identificador de la cuenta.
 * @returns {error} ErrUnavailable si el control de seguridad no responde.
 */
func (s *AttemptsStore) ResetAttempts(ctx context.Context, identifier string) error {
	if s == nil || s.store == nil {
		return identity.ErrUnavailable
	}
	return translate(s.store.ResetAttempts(ctx, identifier))
}

/**
 * LockRemaining devuelve el tiempo que falta para desbloquear la cuenta,
 * sin renovar el bloqueo.
 *
 * @param {context.Context} ctx - Contexto de la petición.
 * @param {string} identifier - Identificador de la cuenta.
 * @returns {time.Duration} Tiempo restante del bloqueo.
 * @returns {error} ErrUnavailable si el control de seguridad no responde.
 */
func (s *AttemptsStore) LockRemaining(ctx context.Context, identifier string) (time.Duration, error) {
	if s == nil || s.store == nil {
		return 0, identity.ErrUnavailable
	}
	ttl, err := s.store.LockRemaining(ctx, identifier)
	if err != nil {
		return 0, translate(err)
	}
	return ttl, nil
}
