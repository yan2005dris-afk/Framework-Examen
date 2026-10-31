package cache

import (
	"context"
	"time"
)

/**
 * NoopCache es un adaptador de caché que no hace nada.
 * Se usa cuando Redis no está disponible para que la API arranque en modo
 * degradado: el rate limiting y la caché de catálogos quedan desactivados,
 * pero el resto de la aplicación funciona con normalidad.
 */
type NoopCache struct{}

func (n *NoopCache) Get(_ context.Context, _ string) (string, error) {
	return "", ErrNotFound
}

func (n *NoopCache) Set(_ context.Context, _ string, _ string, _ time.Duration) error {
	return nil
}

func (n *NoopCache) Incr(_ context.Context, _ string, _ time.Duration) (int64, error) {
	return 0, nil
}

func (n *NoopCache) Delete(_ context.Context, _ string) error {
	return nil
}

func (n *NoopCache) Close() error {
	return nil
}
