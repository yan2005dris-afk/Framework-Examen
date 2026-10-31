package catalog_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/catalog"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/cache"
)

type catalogRepository struct {
	calls []string
	fail  bool
}

func (r *catalogRepository) List(_ context.Context, kind, parent string) ([]string, error) {
	r.calls = append(r.calls, kind+":"+parent)
	if r.fail {
		return nil, errors.New("base de datos no disponible")
	}
	switch kind + ":" + parent {
	case "countries:":
		return []string{"Ecuador", "Argentina", "Bután"}, nil
	case "provinces:EC":
		return []string{"Azuay", "Pichincha"}, nil
	case "cities:EC/Pichincha":
		return []string{"Quito", "Cayambe"}, nil
	case "cities:EC/Azuay":
		return []string{"Cuenca"}, nil
	case "instituciones:":
		return []string{"IESS", "BIESS"}, nil
	default:
		return []string{}, nil
	}
}

type cacheEntry struct {
	value   string
	expires time.Time
}
type catalogCache struct {
	cache.Cache
	data             map[string]cacheEntry
	now              time.Time
	failGet, failSet bool
	ttls             []time.Duration
}

func newCache() *catalogCache {
	return &catalogCache{data: map[string]cacheEntry{}, now: time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)}
}
func (c *catalogCache) Get(ctx context.Context, key string) (string, error) {
	if c.failGet {
		return "", errors.New("Redis no disponible")
	}
	entry, ok := c.data[key]
	if !ok || !c.now.Before(entry.expires) {
		return "", cache.ErrNotFound
	}
	return entry.value, nil
}
func (c *catalogCache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	c.ttls = append(c.ttls, ttl)
	if c.failSet {
		return errors.New("Redis no disponible")
	}
	c.data[key] = cacheEntry{value: value, expires: c.now.Add(ttl)}
	return nil
}

func TestCatalogCacheHitMissExpirationAndTTL(t *testing.T) {
	repo, cached := &catalogRepository{}, newCache()
	s := catalog.NewService(repo, cache.NewCatalogStore(cached))
	for range 2 {
		values, err := s.Countries(context.Background())
		if err != nil || !reflect.DeepEqual(values, []string{"Ecuador", "Argentina", "Bután"}) {
			t.Fatalf("catálogo incorrecto: %v %v", values, err)
		}
		values[0] = "alterado"
	}
	if len(repo.calls) != 1 || len(cached.ttls) != 1 || cached.ttls[0] != 24*time.Hour {
		t.Fatal("no se reutilizó la caché o el TTL no es de 24h")
	}
	cached.now = cached.now.Add(24 * time.Hour)
	if _, err := s.Countries(context.Background()); err != nil || len(repo.calls) != 2 {
		t.Fatal("no se recargó el catálogo expirado")
	}
}

func TestCatalogsFailOpenOnRedisReadWriteAndCorruptEntries(t *testing.T) {
	for _, tc := range []struct {
		name, value      string
		failGet, failSet bool
	}{
		{name: "lectura", failGet: true}, {name: "escritura", failSet: true},
		{name: "json roto", value: "{"}, {name: "objeto", value: `{"paises":[]}`},
		{name: "null", value: "null"}, {name: "números", value: "[1]"},
		{name: "elemento null", value: "[null]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo, cached := &catalogRepository{}, newCache()
			cached.failGet, cached.failSet = tc.failGet, tc.failSet
			if tc.value != "" {
				cached.data["catalog:countries"] = cacheEntry{value: tc.value, expires: cached.now.Add(time.Hour)}
			}
			s := catalog.NewService(repo, cache.NewCatalogStore(cached))
			values, err := s.Countries(context.Background())
			if err != nil || len(values) != 3 || len(repo.calls) != 1 {
				t.Fatalf("Redis bloqueó la fuente persistente: %v %v", values, err)
			}
			if !tc.failSet {
				var fresh []string
				if json.Unmarshal([]byte(cached.data["catalog:countries"].value), &fresh) != nil || len(fresh) != 3 {
					t.Fatal("no se reparó la entrada")
				}
			}
		})
	}
	for _, store := range []catalog.Cache{nil, cache.NewCatalogStore(nil), cache.NewCatalogStore(&cache.NoopCache{})} {
		if values, err := catalog.NewService(&catalogRepository{}, store).Countries(context.Background()); err != nil || len(values) != 3 {
			t.Fatal("la caché opcional impidió consultar PostgreSQL")
		}
	}
}

func TestCatalogDoesNotHideOrRetryDatabaseFailure(t *testing.T) {
	repo, cached := &catalogRepository{fail: true}, newCache()
	cached.failGet = true
	s := catalog.NewService(repo, cache.NewCatalogStore(cached))
	if values, err := s.Countries(context.Background()); values != nil || err == nil || len(repo.calls) != 1 {
		t.Fatal("fallo de PostgreSQL oculto o reintentado")
	}
	if len(cached.data) != 0 || len(cached.ttls) != 0 {
		t.Fatal("se cacheó un error")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.Countries(ctx); !errors.Is(err, context.Canceled) || len(repo.calls) != 1 {
		t.Fatal("se ignoró la cancelación")
	}
}

func TestCatalogGeographyParentsAndDomainOptions(t *testing.T) {
	repo, cached := &catalogRepository{}, newCache()
	s := catalog.NewService(repo, cache.NewCatalogStore(cached))
	ctx := context.Background()
	for _, country := range []string{"EC", "ec", "Ecuador", " ecuador "} {
		if items, err := s.Provinces(ctx, country); err != nil || len(items) != 2 {
			t.Fatalf("alias rechazado: %s %v", country, err)
		}
	}
	if items, err := s.Cities(ctx, "EC", " pichincha "); err != nil || !reflect.DeepEqual(items, []string{"Quito", "Cayambe"}) {
		t.Fatal("cascada incorrecta")
	}
	if items, err := s.Cities(ctx, "EC", "Azuay"); err != nil || !reflect.DeepEqual(items, []string{"Cuenca"}) {
		t.Fatal("se mezclaron provincias")
	}
	if _, ok := cached.data["catalog:cities:EC:Pichincha"]; !ok {
		t.Fatal("clave no canónica")
	}
	for _, country := range []string{"Argentina", "Bután"} {
		if items, err := s.Provinces(ctx, country); err != nil || items == nil || len(items) != 0 {
			t.Fatal("debe permitir captura manual extranjera")
		}
		if items, err := s.Cities(ctx, country, "Estado declarado"); err != nil || len(items) != 0 {
			t.Fatal("se inventó geografía extranjera")
		}
	}
	for _, country := range []string{"", "Atlantis", "Otro país"} {
		if _, err := s.Provinces(ctx, country); !errors.Is(err, catalog.ErrInvalidFilter) {
			t.Fatal("país ajeno al RF aceptado")
		}
	}
	for _, province := range []string{"", "Quito", "Provincia inventada"} {
		if _, err := s.Cities(ctx, "EC", province); !errors.Is(err, catalog.ErrInvalidFilter) {
			t.Fatal("provincia inválida aceptada")
		}
	}
	if items, err := s.Options(ctx, "instituciones"); err != nil || !reflect.DeepEqual(items, []string{"IESS", "BIESS"}) {
		t.Fatal("opciones incorrectas")
	}
	if _, err := s.Options(ctx, "inventado"); !errors.Is(err, catalog.ErrUnknownCatalog) {
		t.Fatal("catálogo desconocido aceptado")
	}
}
