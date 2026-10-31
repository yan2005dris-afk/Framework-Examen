// Package testutil prepara únicamente esquemas de pruebas explícitos, nunca una base de aplicación.
package testutil

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

/** Postgres aplica todas las migraciones a un esquema nuevo sólo con URL de pruebas optativa. */
func Postgres(t *testing.T, variable string) *pgxpool.Pool {
	t.Helper()
	if testing.Short() {
		t.Skip("integración PostgreSQL omitida en modo corto")
	}
	url := os.Getenv(variable)
	if url == "" {
		t.Skip("defina " + variable + " para ejecutar integración PostgreSQL 18")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal("no se pudo abrir la base de pruebas")
	}
	t.Cleanup(admin.Close)
	schema := "audit_review_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	quoted := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+quoted); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := admin.Exec(ctx, "DROP SCHEMA "+quoted+" CASCADE"); err != nil {
			t.Error(err)
		}
	})
	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal("URL de pruebas inválida")
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal("no se pudo abrir el esquema aislado")
	}
	t.Cleanup(pool.Close)
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no se localizó la carpeta de pruebas")
	}
	paths, err := filepath.Glob(filepath.Join(filepath.Dir(source), "..", "..", "sql", "migrations", "*.sql"))
	if err != nil || len(paths) == 0 {
		t.Fatal("no se encontraron migraciones")
	}
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		up := strings.Split(strings.Split(string(raw), "-- +goose Up")[1], "-- +goose Down")[0]
		if _, err := pool.Exec(ctx, up); err != nil {
			t.Fatalf("migración %s: %v", filepath.Base(path), err)
		}
	}
	return pool
}
