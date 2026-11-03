package cache_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/cache"
)

// ---- Doble de prueba en memoria ----

type memCache struct {
	mu   sync.Mutex
	data map[string]string
}

func newMemCache() *memCache {
	return &memCache{data: make(map[string]string)}
}

func (m *memCache) Get(_ context.Context, key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.data[key]
	if !ok {
		return "", cache.ErrNotFound
	}
	return v, nil
}

func (m *memCache) Set(_ context.Context, key, value string, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
	return nil
}

func (m *memCache) Incr(_ context.Context, key string, _ time.Duration) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var cur int64
	if v, ok := m.data[key]; ok {
		for _, c := range v {
			cur = cur*10 + int64(c-'0')
		}
	}
	cur++
	m.data[key] = string(rune('0' + cur%10)) // simplificado para valores 1-9 del test
	// Reescritura correcta para valores > 9
	m.data[key] = itoa(cur)
	return cur, nil
}

func (m *memCache) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

func (m *memCache) Close() error { return nil }

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 20)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	return string(buf)
}

// ---- Tests de AttemptsStore ----

func TestAttemptsStore_NoBloqueoAntesDelLimite(t *testing.T) {
	store := cache.NewAttemptsStore(newAtomicTestCache(t))
	ctx := context.Background()

	// Dos intentos fallidos no deben bloquear la cuenta.
	for i := 0; i < cache.MaxLoginAttempts-1; i++ {
		if _, err := store.RecordFailure(ctx, "1234567890"); err != nil {
			t.Fatalf("RecordFailure: %v", err)
		}
	}

	locked, err := store.IsLocked(ctx, "1234567890")
	if err != nil {
		t.Fatalf("IsLocked: %v", err)
	}
	if locked {
		t.Fatal("la cuenta no debería estar bloqueada antes de alcanzar el límite")
	}
}

func TestAttemptsStore_BloqueoAlAlcanzarLimite(t *testing.T) {
	store := cache.NewAttemptsStore(newAtomicTestCache(t))
	ctx := context.Background()

	// Tres intentos fallidos deben bloquear la cuenta.
	for i := 0; i < cache.MaxLoginAttempts; i++ {
		if _, err := store.RecordFailure(ctx, "1234567890"); err != nil {
			t.Fatalf("RecordFailure intento %d: %v", i+1, err)
		}
	}

	locked, err := store.IsLocked(ctx, "1234567890")
	if err != nil {
		t.Fatalf("IsLocked: %v", err)
	}
	if !locked {
		t.Fatal("la cuenta debe estar bloqueada tras alcanzar el límite de intentos")
	}
}

func TestAttemptsStore_ResetDespuesDeExito(t *testing.T) {
	store := cache.NewAttemptsStore(newAtomicTestCache(t))
	ctx := context.Background()

	// Dos intentos fallidos seguidos de un inicio de sesión exitoso.
	for i := 0; i < cache.MaxLoginAttempts-1; i++ {
		if _, err := store.RecordFailure(ctx, "1234567890"); err != nil {
			t.Fatalf("RecordFailure: %v", err)
		}
	}

	if err := store.ResetAttempts(ctx, "1234567890"); err != nil {
		t.Fatalf("ResetAttempts: %v", err)
	}

	// Después del reset, un nuevo intento fallido no debe bloquear de inmediato.
	if _, err := store.RecordFailure(ctx, "1234567890"); err != nil {
		t.Fatalf("RecordFailure tras reset: %v", err)
	}

	locked, err := store.IsLocked(ctx, "1234567890")
	if err != nil {
		t.Fatalf("IsLocked: %v", err)
	}
	if locked {
		t.Fatal("la cuenta no debe estar bloqueada tras un solo intento después del reset")
	}
}

func TestAttemptsStore_CuentasSeparadas(t *testing.T) {
	store := cache.NewAttemptsStore(newAtomicTestCache(t))
	ctx := context.Background()

	// Bloquear solo la primera cédula.
	for i := 0; i < cache.MaxLoginAttempts; i++ {
		if _, err := store.RecordFailure(ctx, "1111111111"); err != nil {
			t.Fatalf("RecordFailure: %v", err)
		}
	}

	// La segunda cédula no debe estar bloqueada.
	locked, err := store.IsLocked(ctx, "2222222222")
	if err != nil {
		t.Fatalf("IsLocked: %v", err)
	}
	if locked {
		t.Fatal("el bloqueo de una cédula no debe afectar a otras")
	}
}

// ---- Tests de CatalogStore ----

func TestCatalogStore_GetOrSet_CacheMiss(t *testing.T) {
	store := cache.NewCatalogStore(newMemCache())
	ctx := context.Background()

	llamadas := 0
	fetch := func() (string, error) {
		llamadas++
		return `["EC"]`, nil
	}

	val, err := store.GetOrSet(ctx, cache.CountriesKey(), fetch)
	if err != nil {
		t.Fatalf("GetOrSet: %v", err)
	}
	if val != `["EC"]` {
		t.Fatalf("valor inesperado: %q", val)
	}
	if llamadas != 1 {
		t.Fatalf("fetch debe haberse llamado 1 vez, se llamó %d", llamadas)
	}
}

func TestCatalogStore_GetOrSet_CacheHit(t *testing.T) {
	store := cache.NewCatalogStore(newMemCache())
	ctx := context.Background()

	llamadas := 0
	fetch := func() (string, error) {
		llamadas++
		return `["EC"]`, nil
	}

	// Primera llamada: miss.
	if _, err := store.GetOrSet(ctx, cache.CountriesKey(), fetch); err != nil {
		t.Fatalf("primera GetOrSet: %v", err)
	}

	// Segunda llamada: hit, fetch no debe ejecutarse.
	val, err := store.GetOrSet(ctx, cache.CountriesKey(), fetch)
	if err != nil {
		t.Fatalf("segunda GetOrSet: %v", err)
	}
	if val != `["EC"]` {
		t.Fatalf("valor inesperado: %q", val)
	}
	if llamadas != 1 {
		t.Fatalf("fetch solo debe llamarse una vez; se llamó %d veces", llamadas)
	}
}

func TestCatalogStore_Keys(t *testing.T) {
	casos := []struct {
		nombre   string
		clave    string
		esperado string
	}{
		{"países", cache.CountriesKey(), "catalog:countries"},
		{"provincias Ecuador", cache.ProvincesKey("EC"), "catalog:provinces:EC"},
		{"cantones Pichincha", cache.CitiesKey("EC", "17"), "catalog:cities:EC:17"},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			if c.clave != c.esperado {
				t.Fatalf("clave incorrecta: quería %q, obtuve %q", c.esperado, c.clave)
			}
		})
	}
}
