package postgres

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
)

func catalogPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("CATALOG_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("defina CATALOG_TEST_DATABASE_URL para comprobar catálogos en PostgreSQL 18")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal("no se pudo abrir la base de pruebas")
	}
	t.Cleanup(admin.Close)
	schema := "catalog_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	quoted := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+quoted); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := admin.Exec(ctx, "DROP SCHEMA "+quoted+" CASCADE"); err != nil {
			t.Error(err)
		}
	})
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal("URL de pruebas inválida")
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal("no se pudo abrir el esquema aislado")
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestCatalogSQLMigrationAndStandaloneSeedAreCompleteAndIdempotent(t *testing.T) {
	definition, err := submission.FormDefinition()
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"migration", "standalone"} {
		t.Run(mode, func(t *testing.T) {
			pool := catalogPool(t)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			path := filepath.Join("..", "..", "..", "sql", "seeds", "catalogs.sql")
			if mode == "migration" {
				path = filepath.Join("..", "..", "..", "sql", "migrations", "00008_catalogs.sql")
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			body := string(raw)
			if mode == "migration" {
				body = strings.Split(strings.Split(body, "-- +goose Up")[1], "-- +goose Down")[0]
			}
			if _, err := pool.Exec(ctx, body); err != nil {
				t.Fatal(err)
			}
			var before uuid.UUID
			if err := pool.QueryRow(ctx, `SELECT id FROM catalogs WHERE kind='countries' AND value='Ecuador'`).Scan(&before); err != nil || before.Version() != 7 {
				t.Fatal("identificador no es UUIDv7", err)
			}
			if _, err := pool.Exec(ctx, body); err != nil {
				t.Fatal("la semilla no es idempotente", err)
			}
			var after uuid.UUID
			if err := pool.QueryRow(ctx, `SELECT id FROM catalogs WHERE kind='countries' AND value='Ecuador'`).Scan(&after); err != nil || before != after {
				t.Fatal("reaplicar alteró identificadores", err)
			}
			repo := NewRepository(sqlcgen.New(pool))
			countries, err := repo.List(ctx, "countries", "")
			if err != nil || !reflect.DeepEqual(countries, definition.Options["paises"]) {
				t.Fatal("países incompletos o desordenados", err)
			}
			provinces, err := repo.List(ctx, "provinces", "EC")
			expected := []string{}
			for province := range definition.Provinces {
				expected = append(expected, province)
			}
			sort.Strings(expected)
			if err != nil || !reflect.DeepEqual(provinces, expected) {
				t.Fatal("provincias incompletas", err)
			}
			total := 0
			for province, cities := range definition.Provinces {
				got, err := repo.List(ctx, "cities", "EC/"+province)
				if err != nil || !reflect.DeepEqual(got, cities) {
					t.Fatalf("cantones incorrectos de %s: %v", province, err)
				}
				total += len(got)
			}
			if total != 221 {
				t.Fatal("se perdió cobertura de cantones")
			}
			for kind, values := range definition.Options {
				if kind == "paises" {
					continue
				}
				got, err := repo.List(ctx, kind, "")
				if err != nil || !reflect.DeepEqual(got, values) {
					t.Fatalf("opciones incorrectas: %s %v", kind, err)
				}
			}
			var duplicates int
			if err := pool.QueryRow(ctx, `SELECT count(*) FROM (SELECT kind,parent_key,value FROM catalogs GROUP BY kind,parent_key,value HAVING count(*)>1) repeated`).Scan(&duplicates); err != nil || duplicates != 0 {
				t.Fatal("reaplicar generó duplicados", err)
			}
			var brokenParents int
			if err := pool.QueryRow(ctx, `SELECT count(*) FROM catalogs child JOIN catalogs parent ON child.parent_id=parent.id WHERE (child.kind='provinces' AND (parent.kind<>'countries' OR parent.value<>'Ecuador')) OR (child.kind='cities' AND (parent.kind<>'provinces' OR child.parent_key<>'EC/'||parent.value))`).Scan(&brokenParents); err != nil || brokenParents != 0 {
				t.Fatal("relaciones geográficas incorrectas", err)
			}
			if _, err := pool.Exec(ctx, `INSERT INTO catalogs(kind,parent_key,parent_id,value,position) VALUES('cities','EC/Pichincha',$1,'Huérfano',0)`, uuid.New()); err == nil {
				t.Fatal("se admitió un cantón huérfano")
			}
			if mode == "migration" {
				down := strings.Split(string(raw), "-- +goose Down")[1]
				if _, err := pool.Exec(ctx, down); err != nil {
					t.Fatal("rollback de migración inválido", err)
				}
			}
		})
	}
}
