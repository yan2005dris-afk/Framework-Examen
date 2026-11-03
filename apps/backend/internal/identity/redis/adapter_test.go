package redis_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"

	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	identityredis "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity/redis"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/cache"
)

/**
 * nuevoCliente levanta un Redis efímero para probar el adaptador contra la
 * misma infraestructura que usa la aplicación.
 */
func nuevoCliente(t *testing.T) *cache.RedisClient {
	t.Helper()
	server := miniredis.RunT(t)
	client, err := cache.NewRedis(context.Background(), "redis://"+server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestAdaptadorTraduceReenvioLimitado(t *testing.T) {
	store := identityredis.NewOTPStore(cache.NewOTPStore(nuevoCliente(t), identity.OTPTTL))
	ctx := context.Background()

	if _, err := store.Issue(ctx, identity.PurposeRegistration, "cuenta", "a@example.test", "123456"); err != nil {
		t.Fatalf("primera emisión: %v", err)
	}
	// El reenvío inmediato excede el enfriamiento: la caché responde con su
	// propio error y el adaptador debe entregar un error del dominio.
	_, err := store.Issue(ctx, identity.PurposeRegistration, "cuenta", "a@example.test", "234567")
	if !errors.Is(err, identity.ErrResendLimited) {
		t.Fatalf("se esperaba ErrResendLimited del dominio, se obtuvo %v", err)
	}
	var retry *identity.RetryError
	if !errors.As(err, &retry) {
		t.Fatal("se perdió el vencimiento del reintento")
	}
	if retry.RetryAfter <= 0 || retry.RetryAfter > time.Minute {
		t.Fatalf("vencimiento inesperado: %v", retry.RetryAfter)
	}
}

func TestAdaptadorTraduceDesafioBloqueado(t *testing.T) {
	store := identityredis.NewOTPStore(cache.NewOTPStore(nuevoCliente(t), identity.OTPTTL))
	ctx := context.Background()

	challenge, err := store.Issue(ctx, identity.PurposeRegistration, "cuenta", "a@example.test", "123456")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		_, err = store.Verify(ctx, identity.PurposeRegistration, "cuenta", "a@example.test", challenge.ID, "000000")
	}
	if !errors.Is(err, identity.ErrOTPLocked) {
		t.Fatalf("se esperaba ErrOTPLocked del dominio, se obtuvo %v", err)
	}
}

func TestAdaptadorBloqueaTrasLosIntentosDelDominio(t *testing.T) {
	attempts := identityredis.NewAttemptsStore(cache.NewAttemptsStore(nuevoCliente(t), identity.MaxLoginAttempts, identity.LockoutDuration))
	ctx := context.Background()

	for i := 0; i < identity.MaxLoginAttempts; i++ {
		count, err := attempts.RecordFailure(ctx, "cuenta")
		if err != nil {
			t.Fatalf("intento %d: %v", i+1, err)
		}
		if count != int64(i+1) {
			t.Fatalf("contador inesperado: %d", count)
		}
	}
	locked, err := attempts.IsLocked(ctx, "cuenta")
	if err != nil || !locked {
		t.Fatalf("la cuenta no quedó bloqueada tras %d intentos: %v", identity.MaxLoginAttempts, err)
	}
	remaining, err := attempts.LockRemaining(ctx, "cuenta")
	if err != nil || remaining <= 0 || remaining > identity.LockoutDuration {
		t.Fatalf("vencimiento de bloqueo inesperado: %v %v", remaining, err)
	}
}

func TestAdaptadorFallaCerradoSinCache(t *testing.T) {
	ctx := context.Background()
	attempts := identityredis.NewAttemptsStore(cache.NewAttemptsStore(&cache.NoopCache{}, identity.MaxLoginAttempts, identity.LockoutDuration))
	if _, err := attempts.IsLocked(ctx, "cuenta"); !errors.Is(err, identity.ErrUnavailable) {
		t.Fatalf("se esperaba ErrUnavailable, se obtuvo %v", err)
	}
	otp := identityredis.NewOTPStore(cache.NewOTPStore(&cache.NoopCache{}, identity.OTPTTL))
	if _, err := otp.Issue(ctx, identity.PurposeRegistration, "cuenta", "a@example.test", "123456"); !errors.Is(err, identity.ErrUnavailable) {
		t.Fatalf("se esperaba ErrUnavailable, se obtuvo %v", err)
	}
}

func TestAdaptadorFallaCerradoSinAlmacen(t *testing.T) {
	ctx := context.Background()
	var attempts *identityredis.AttemptsStore
	if _, err := attempts.IsLocked(ctx, "cuenta"); !errors.Is(err, identity.ErrUnavailable) {
		t.Fatalf("se esperaba ErrUnavailable, se obtuvo %v", err)
	}
	var otp *identityredis.OTPStore
	if _, err := otp.Verify(ctx, identity.PurposeRecovery, "cuenta", "a@example.test", "id", "123456"); !errors.Is(err, identity.ErrUnavailable) {
		t.Fatalf("se esperaba ErrUnavailable, se obtuvo %v", err)
	}
}

func TestAdaptadorRechazaFinalidadDesconocida(t *testing.T) {
	otp := identityredis.NewOTPStore(cache.NewOTPStore(nuevoCliente(t), identity.OTPTTL))
	if _, err := otp.Issue(context.Background(), identity.OTPPurpose("inventada"), "cuenta", "a@example.test", "123456"); !errors.Is(err, identity.ErrUnavailable) {
		t.Fatalf("se esperaba ErrUnavailable, se obtuvo %v", err)
	}
}
