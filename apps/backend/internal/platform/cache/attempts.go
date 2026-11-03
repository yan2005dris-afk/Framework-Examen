package cache

import (
	"context"
	"errors"
	"fmt"
	"time"
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
 *
 * El límite de intentos y la duración del bloqueo los fija el dominio y
 * llegan por parámetro: la caché aplica la política, no la decide.
 */
type AttemptsStore struct {
	cache           Cache
	maxAttempts     int64
	lockoutDuration time.Duration
}

/**
 * NewAttemptsStore crea un AttemptsStore que usa el Cache provisto.
 *
 * Una política inválida (límite o duración no positivos) deja el control de
 * seguridad inoperante a propósito: el dominio falla cerrado en lugar de
 * aplicar un límite que nadie definió.
 *
 * @param {Cache} c - Adaptador de caché (Redis u otro).
 * @param {int} maxAttempts - Intentos fallidos que disparan el bloqueo.
 * @param {time.Duration} lockoutDuration - Duración del bloqueo de la cuenta.
 * @returns {*AttemptsStore} Store listo para usar.
 */
func NewAttemptsStore(c Cache, maxAttempts int, lockoutDuration time.Duration) *AttemptsStore {
	if maxAttempts <= 0 || lockoutDuration <= 0 {
		c = &UnavailableCache{}
	}
	if _, atomic := c.(ScriptCache); !atomic {
		c = &UnavailableCache{}
	}
	if c == nil {
		c = &UnavailableCache{}
	}
	if _, noop := c.(*NoopCache); noop {
		c = &UnavailableCache{}
	}
	return &AttemptsStore{cache: c, maxAttempts: int64(maxAttempts), lockoutDuration: lockoutDuration}
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
	if s == nil {
		return false, ErrUnavailable
	}
	_, err := s.cache.Get(ctx, lockoutKey(cedula))
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, ErrUnavailable
	}
	return true, nil
}

/**
 * RecordFailure incrementa el contador de intentos fallidos y activa el
 * bloqueo cuando se alcanza el límite configurado.
 *
 * El TTL de ambas claves es la duración del bloqueo; cuando expiran Redis las
 * elimina automáticamente, sin necesidad de ningún proceso externo.
 *
 * @param {Context} ctx - Contexto de la petición.
 * @param {string} cedula - Número de cédula del postulante.
 * @returns {int64} Número de intentos acumulados tras este fallo.
 * @throws {error} Error de Redis.
 */
func (s *AttemptsStore) RecordFailure(ctx context.Context, cedula string) (int64, error) {
	if s == nil {
		return 0, ErrUnavailable
	}
	a, ok := s.cache.(ScriptCache)
	if !ok {
		return 0, ErrUnavailable
	}
	count, err := a.RunScript(ctx, `if redis.call('EXISTS',KEYS[2])==1 then return tonumber(ARGV[2]) end;local n=redis.call('INCR',KEYS[1]);if n==1 then redis.call('PEXPIRE',KEYS[1],ARGV[1]) end;if n>=tonumber(ARGV[2]) then redis.call('SET',KEYS[2],'1','PX',ARGV[1],'NX') end;return n`, []string{attemptsKey(cedula), lockoutKey(cedula)}, s.lockoutDuration.Milliseconds(), s.maxAttempts)
	if err != nil {
		return 0, ErrUnavailable
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
	if s == nil {
		return ErrUnavailable
	}
	if err := s.cache.Delete(ctx, attemptsKey(cedula)); err != nil {
		return ErrUnavailable
	}
	return nil
}

// attemptsKey devuelve la clave del contador de intentos para una cédula.
func attemptsKey(cedula string) string {
	return fmt.Sprintf("identity:auth:{%s}:attempts", digest(cedula))
}

// lockoutKey devuelve la clave de bloqueo para una cédula.
func lockoutKey(cedula string) string {
	return fmt.Sprintf("identity:auth:{%s}:lock", digest(cedula))
}
