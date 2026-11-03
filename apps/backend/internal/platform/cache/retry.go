package cache

import (
	"context"
	"time"
)

/** RetryError conserva el código de dominio y el vencimiento leído en Redis. */
type RetryError struct {
	Cause      error
	RetryAfter time.Duration
}

func (e *RetryError) Error() string { return e.Cause.Error() }
func (e *RetryError) Unwrap() error { return e.Cause }

func remaining(ctx context.Context, c Cache, keys ...string) (time.Duration, error) {
	a, ok := c.(ScriptCache)
	if !ok {
		return 0, ErrUnavailable
	}
	ms, err := a.RunScript(ctx, `local ttl=0;for _,k in ipairs(KEYS) do ttl=math.max(ttl,redis.call('PTTL',k)) end;return ttl`, keys)
	if err != nil {
		return 0, ErrUnavailable
	}
	return time.Duration(ms) * time.Millisecond, nil
}

/** LockRemaining devuelve el TTL efectivo, sin prolongar el bloqueo. */
func (s *AttemptsStore) LockRemaining(ctx context.Context, identifier string) (time.Duration, error) {
	return remaining(ctx, s.cache, lockoutKey(identifier))
}
