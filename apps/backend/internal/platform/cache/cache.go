package cache

import (
	"context"
	"time"
)

/**
 * Cache define el puerto de caché que usa la aplicación.
 * Cualquier adaptador (Redis, en memoria, noop) debe implementar esta interfaz.
 */
type Cache interface {
	// Get devuelve el valor asociado a la clave.
	// Devuelve ErrNotFound si la clave no existe o ya expiró.
	Get(ctx context.Context, key string) (string, error)

	// Set guarda el valor con el TTL indicado.
	// Si ttl es cero, la clave no expira.
	Set(ctx context.Context, key string, value string, ttl time.Duration) error

	// Incr incrementa en 1 el contador de la clave y devuelve el valor resultante.
	// Si la clave no existe la crea con valor 1.
	// Si ttl es mayor que cero y la clave acaba de crearse, le asigna ese TTL.
	Incr(ctx context.Context, key string, ttl time.Duration) (int64, error)

	// Delete elimina la clave. No devuelve error si no existía.
	Delete(ctx context.Context, key string) error

	// Close libera los recursos del adaptador.
	Close() error
}

// ErrNotFound se devuelve cuando la clave no existe o ya expiró.
var ErrNotFound = errNotFound("caché: clave no encontrada")

type errNotFound string

func (e errNotFound) Error() string { return string(e) }
