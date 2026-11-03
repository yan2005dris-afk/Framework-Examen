package cache_test

import (
	"context"
	"errors"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/cache"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func newAtomicTestCache(t *testing.T) *cache.RedisClient {
	t.Helper()
	server := miniredis.RunT(t)
	r, err := cache.NewRedis(context.Background(), "redis://"+server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close() })
	return r
}

func TestOTPPurposeEmailAndSingleConcurrentConsumer(t *testing.T) {
	r := newAtomicTestCache(t)
	s := cache.NewOTPStore(r, otpTTLReplica)
	ctx := context.Background()
	reg, err := s.Issue(ctx, cache.PurposeRegistration, "sujeto", "a@example.test", "123456")
	if err != nil {
		t.Fatal(err)
	}
	rec, err := s.Issue(ctx, cache.PurposeRecovery, "sujeto", "a@example.test", "654321")
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := s.Verify(ctx, cache.PurposeRecovery, "sujeto", "a@example.test", rec.ID, "123456"); err != nil || ok {
		t.Fatal("código cruzado entre finalidades", err)
	}
	if ok, err := s.Verify(ctx, cache.PurposeRegistration, "sujeto", "b@example.test", reg.ID, "123456"); err != nil || ok {
		t.Fatal("correo sustituido", err)
	}
	var wg sync.WaitGroup
	var winners atomic.Int32
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			valid, err := s.Verify(ctx, cache.PurposeRegistration, "sujeto", "a@example.test", reg.ID, "123456")
			if err != nil {
				t.Error(err)
			}
			if valid {
				winners.Add(1)
			}
		}()
	}
	wg.Wait()
	if winners.Load() != 1 {
		t.Fatalf("ganadores=%d", winners.Load())
	}
	if valid, err := s.Verify(ctx, cache.PurposeRecovery, "sujeto", "a@example.test", rec.ID, "654321"); err != nil || !valid {
		t.Fatal("un flujo consumió el otro", err)
	}
}

func TestOTPResendChallengeAttemptsAndExpiry(t *testing.T) {
	server := miniredis.RunT(t)
	r, err := cache.NewRedis(context.Background(), "redis://"+server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	s := cache.NewOTPStore(r, otpTTLReplica)
	ctx := context.Background()
	old, err := s.Issue(ctx, cache.PurposeRegistration, "sujeto", "a@example.test", "123456")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Issue(ctx, cache.PurposeRegistration, "sujeto", "b@example.test", "222222"); !errors.Is(err, cache.ErrResendLimited) {
		t.Fatal("reenvío ilimitado", err)
	}
	server.FastForward(61 * time.Second)
	current, err := s.Issue(ctx, cache.PurposeRegistration, "sujeto", "b@example.test", "222222")
	if err != nil {
		t.Fatal(err)
	}
	if active, err := s.Active(ctx, old.Key, old.ID); err != nil || active {
		t.Fatal("correo viejo aún entregable", err)
	}
	ttl := server.TTL(current.Key)
	for i := 0; i < 3; i++ {
		_, err := s.Verify(ctx, cache.PurposeRegistration, "sujeto", "b@example.test", current.ID, "000000")
		if i == 2 && !errors.Is(err, cache.ErrOTPLocked) {
			t.Fatal("sin bloqueo del desafío", err)
		}
	}
	if server.TTL(current.Key) > ttl {
		t.Fatal("un error extendió el desafío")
	}
	server.FastForward(61 * time.Second)
	if _, err := s.Issue(ctx, cache.PurposeRegistration, "sujeto", "b@example.test", "333333"); !errors.Is(err, cache.ErrOTPLocked) {
		t.Fatal("reenvío elude bloqueo", err)
	}
	server.FastForward(otpTTLReplica)
	if valid, err := s.Verify(ctx, cache.PurposeRegistration, "sujeto", "b@example.test", current.ID, "222222"); err != nil || valid {
		t.Fatal("OTP vencido aceptado", err)
	}
	if _, err := s.Issue(ctx, cache.PurposeRegistration, "sujeto", "b@example.test", "333333"); err != nil {
		t.Fatal("ventana vencida no habilita nuevo intento", err)
	}
}

func TestSecurityCannotUseNoopAndCatalogCanDegrade(t *testing.T) {
	ctx := context.Background()
	noop := &cache.NoopCache{}
	if _, err := cache.NewAttemptsStore(noop, maxAttemptsReplica, lockoutReplica).IsLocked(ctx, "sujeto"); !errors.Is(err, cache.ErrUnavailable) {
		t.Fatal("Noop habilitó login", err)
	}
	if _, err := cache.NewOTPStore(noop, otpTTLReplica).Issue(ctx, cache.PurposeRecovery, "sujeto", "a@example.test", "123456"); !errors.Is(err, cache.ErrUnavailable) {
		t.Fatal("Noop aceptó OTP", err)
	}
	value, err := cache.NewCatalogStore(noop).GetOrSet(ctx, "catalog:prueba", func() (string, error) { return `["Ecuador"]`, nil })
	if err != nil || value != `["Ecuador"]` {
		t.Fatal("catálogo no degradó", err)
	}
}

func TestRedisSecurityIntegrationTTLAndAtomicity(t *testing.T) {
	url := os.Getenv("REDIS_TEST_URL")
	if url == "" {
		t.Skip("defina REDIS_TEST_URL para Redis real")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	r, err := cache.NewRedis(ctx, url)
	if err != nil {
		t.Fatal("Redis de pruebas no disponible")
	}
	defer r.Close()
	key := "test:counter:" + uuid.NewString()
	defer r.Delete(context.Background(), key)
	ttl := 3 * time.Second
	if _, err := r.Incr(ctx, key, ttl); err != nil {
		t.Fatal(err)
	}
	time.Sleep(2100 * time.Millisecond)
	if _, err := r.Incr(ctx, key, ttl); err != nil {
		t.Fatal(err)
	}
	time.Sleep(1200 * time.Millisecond)
	if _, err := r.Get(ctx, key); !errors.Is(err, cache.ErrNotFound) {
		t.Fatal("INCR renovó la ventana o perdió TTL", err)
	}
	s := cache.NewOTPStore(r, otpTTLReplica)
	subject := uuid.NewString()
	// La cuota también es por correo: aislar cada corrida sin borrar controles de Redis.
	email := subject + "@example.test"
	c, err := s.Issue(ctx, cache.PurposeRecovery, subject, email, "123456")
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	var winners atomic.Int32
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := s.Verify(ctx, cache.PurposeRecovery, subject, email, c.ID, "123456")
			if err != nil {
				t.Error(err)
			}
			if ok {
				winners.Add(1)
			}
		}()
	}
	wg.Wait()
	if winners.Load() != 1 {
		t.Fatalf("Redis: consumidores=%d", winners.Load())
	}
}

func TestEmailSendQuotaSurvivesDifferentSubjectsAndSeparatesPurposes(t *testing.T) {
	r := newAtomicTestCache(t)
	s := cache.NewOTPStore(r, otpTTLReplica)
	ctx := context.Background()
	for n := 0; n < cache.MaxOTPSends; n++ {
		if _, e := s.Issue(ctx, cache.PurposeRegistration, uuid.NewString(), " Same@Example.Test ", "123456"); e != nil {
			t.Fatal(e)
		}
	}
	if _, e := s.Issue(ctx, cache.PurposeRegistration, uuid.NewString(), "same@example.test", "654321"); !errors.Is(e, cache.ErrResendLimited) {
		t.Fatal("se evadió límite de correo", e)
	}
	if _, e := s.Issue(ctx, cache.PurposeRecovery, uuid.NewString(), "same@example.test", "654321"); e != nil {
		t.Fatal("mezcló finalidades", e)
	}
}
