package cache

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Constantes del bloqueo de autenticación según el RF EF-11.
const (
	// MaxLoginAttempts es el número de intentos fallidos que dispara el bloqueo.
	MaxLoginAttempts = 3

	// LockoutDuration es el tiempo que dura el bloqueo de la cuenta.
	LockoutDuration = 10 * time.Minute
)

/**
 * AttemptsStore gestiona el contador de intentos fallidos de autenticación
 * y el bloqueo automático por tiempo (TTL) que exige el EF-11:
 *
 *   - Clave de contador : identity:login-attempts:{cedula}  (TTL: 10 min)
 *   - Clave de bloqueo  : identity:lockout:{cedula}         (TTL: 10 min)
 *
 * El bloqueo desaparece solo cuando Redis purga la clave; no requiere
 * ningún proceso adicional.
 */
type AttemptsStore struct {
	cache Cache
}

/**
 * NewAttemptsStore crea un AttemptsStore que usa el Cache provisto.
 *
 * @param {Cache} c - Adaptador de caché (Redis u otro).
 * @returns {*AttemptsStore} Store listo para usar.
 */
func NewAttemptsStore(c Cache) *AttemptsStore {
	return &AttemptsStore{cache: c}
}

/**
 * IsLocked indica si la cédula está bloqueada por exceso de intentos.
 *
 * @param {Context} ctx - Contexto de la petición.
 * @param {string} cedula - Número de cédula del postulante.
 * @returns {bool} true si la cuenta está bloqueada.
 * @throws {error} Error de Redis.
 */
func (s *AttemptsStore) IsLocked(ctx context.Context, cedula string) (bool, error) {
	_, err := s.cache.Get(ctx, lockoutKey(cedula))
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

/**
 * RecordFailure incrementa el contador de intentos fallidos y activa el
 * bloqueo cuando se alcanza el límite (MaxLoginAttempts).
 *
 * El TTL de ambas claves es LockoutDuration; cuando expiran Redis las
 * elimina automáticamente, sin necesidad de ningún proceso externo.
 *
 * @param {Context} ctx - Contexto de la petición.
 * @param {string} cedula - Número de cédula del postulante.
 * @returns {int64} Número de intentos acumulados tras este fallo.
 * @throws {error} Error de Redis.
 */
func (s *AttemptsStore) RecordFailure(ctx context.Context, cedula string) (int64, error) {
	count, err := s.cache.Incr(ctx, attemptsKey(cedula), LockoutDuration)
	if err != nil {
		return 0, fmt.Errorf("attempts: RecordFailure(%q): %w", cedula, err)
	}

	if count >= MaxLoginAttempts {
		// Activa el bloqueo. Si ya existía, lo renueva.
		if err := s.cache.Set(ctx, lockoutKey(cedula), "1", LockoutDuration); err != nil {
			return count, fmt.Errorf("attempts: activar bloqueo para %q: %w", cedula, err)
		}
	}

	return count, nil
}

/**
 * ResetAttempts borra el contador de intentos tras una autenticación exitosa.
 *
 * @param {Context} ctx - Contexto de la petición.
 * @param {string} cedula - Número de cédula del postulante.
 * @throws {error} Error de Redis.
 */
func (s *AttemptsStore) ResetAttempts(ctx context.Context, cedula string) error {
	if err := s.cache.Delete(ctx, attemptsKey(cedula)); err != nil {
		return fmt.Errorf("attempts: ResetAttempts(%q): %w", cedula, err)
	}
	return nil
}

// attemptsKey devuelve la clave del contador de intentos para una cédula.
func attemptsKey(cedula string) string {
	return fmt.Sprintf("identity:login-attempts:%s", cedula)
}

// lockoutKey devuelve la clave de bloqueo para una cédula.
func lockoutKey(cedula string) string {
	return fmt.Sprintf("identity:lockout:%s", cedula)
}
