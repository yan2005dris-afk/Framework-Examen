package cache

import (
	"context"
	"time"
)

/** NoopCache omite sólo caché de catálogos; no es un adaptador de seguridad.
 * OTP y bloqueo de intentos rechazan expresamente este adaptador.
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
