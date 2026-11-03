package cache_test

import (
	"context"
	"errors"
	"github.com/alicebob/miniredis/v2"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/cache"
	"testing"
	"time"
)

func TestConfiguredOTPResendReportsActualQuotaAndCooldownTTL(t *testing.T) {
	server := miniredis.RunT(t)
	ctx := context.Background()
	redis, err := cache.NewRedis(ctx, "redis://"+server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = redis.Close() })
	store, err := cache.NewConfiguredOTPStore(redis, 2*time.Second, 2)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.Issue(ctx, cache.PurposeRegistration, "account", "a@example.test", "123456"); err != nil {
		t.Fatal(err)
	}
	_, err = store.Issue(ctx, cache.PurposeRegistration, "account", "a@example.test", "234567")
	var retry *cache.RetryError
	if !errors.As(err, &retry) || retry.RetryAfter > 2*time.Second || retry.RetryAfter < time.Second {
		t.Fatal("cooldown incorrecto", err)
	}
	server.FastForward(2 * time.Second)
	if _, err = store.Issue(ctx, cache.PurposeRegistration, "account", "a@example.test", "234567"); err != nil {
		t.Fatal(err)
	}
	server.FastForward(2 * time.Second)
	_, err = store.Issue(ctx, cache.PurposeRegistration, "account", "a@example.test", "345678")
	if !errors.As(err, &retry) || retry.RetryAfter < 595*time.Second || retry.RetryAfter > 596*time.Second {
		t.Fatal("la cuota no conserva su ventana original", err, retry)
	}
}
func TestLockRemainingNeverRenewsWindow(t *testing.T) {
	server := miniredis.RunT(t)
	ctx := context.Background()
	redis, err := cache.NewRedis(ctx, "redis://"+server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = redis.Close() })
	store := cache.NewAttemptsStore(redis)
	for range 3 {
		if _, err := store.RecordFailure(ctx, "account"); err != nil {
			t.Fatal(err)
		}
	}
	server.FastForward(time.Minute)
	if _, err := store.RecordFailure(ctx, "account"); err != nil {
		t.Fatal(err)
	}
	ttl, err := store.LockRemaining(ctx, "account")
	if err != nil || ttl > 9*time.Minute || ttl < 9*time.Minute-time.Second {
		t.Fatal(ttl, err)
	}
	server.FastForward(time.Minute)
	ttl, err = store.LockRemaining(ctx, "account")
	if err != nil || ttl > 8*time.Minute {
		t.Fatal("bloqueo prolongado", ttl, err)
	}
}
