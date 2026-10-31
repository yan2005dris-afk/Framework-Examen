package cache

import (
	"context"
	"time"
)

/** UnavailableCache deniega seguridad; sólo catálogos pueden usar NoopCache. */
type UnavailableCache struct{}

func (*UnavailableCache) Get(context.Context, string) (string, error) { return "", ErrUnavailable }
func (*UnavailableCache) Set(context.Context, string, string, time.Duration) error {
	return ErrUnavailable
}
func (*UnavailableCache) Incr(context.Context, string, time.Duration) (int64, error) {
	return 0, ErrUnavailable
}
func (*UnavailableCache) Delete(context.Context, string) error { return ErrUnavailable }
func (*UnavailableCache) Close() error                         { return nil }
