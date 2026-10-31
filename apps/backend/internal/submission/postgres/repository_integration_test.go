package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/db"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
)

// Usa únicamente una URL de pruebas explícita y un esquema aislado por ejecución.
func integrationRepository(t *testing.T) (*Repository, *submission.Submission) {
	t.Helper()
	url := os.Getenv("SUBMISSION_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("defina SUBMISSION_TEST_DATABASE_URL para probar PostgreSQL real")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal("no se pudo preparar el pool de pruebas")
	}
	t.Cleanup(admin.Close)
	schema := "submission_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	quotedSchema := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+quotedSchema); err != nil {
		t.Fatal("no se pudo crear el esquema aislado")
	}
	t.Cleanup(func() {
		cleanupCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if _, err := admin.Exec(cleanupCtx, "DROP SCHEMA "+quotedSchema+" CASCADE"); err != nil {
			t.Error("no se pudo eliminar el esquema de pruebas")
		}
	})
	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal("URL de pruebas inválida")
	}
	config.MaxConns = 4
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal("no se pudo abrir el pool aislado")
	}
	t.Cleanup(pool.Close)
	for _, name := range []string{"00001_init.sql", "00004_documents.sql", "00006_submission_events.sql"} {
		raw, err := os.ReadFile(filepath.Join("..", "..", "..", "sql", "migrations", name))
		if err != nil {
			t.Fatal(err)
		}
		up := strings.Split(strings.Split(string(raw), "-- +goose Up")[1], "-- +goose Down")[0]
		if _, err := pool.Exec(ctx, up); err != nil {
			t.Fatal(err)
		}
	}
	applicantID, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO applicants (id, cedula, nombres, apellidos, correo) VALUES ($1, '1710034008', 'Persona', 'Prueba', 'prueba@example.test')`, applicantID); err != nil {
		t.Fatal(err)
	}
	id, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	item, err := submission.NewSubmission(id, applicantID, "EF12-2026-DBTEST", json.RawMessage(`{}`), time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	repo := NewRepository(sqlcgen.New(pool), db.NewTxManager(pool))
	if err := repo.Create(ctx, item); err != nil {
		t.Fatal(err)
	}
	return repo, item
}

func TestPostgresSerializesMutationsAndRollsBack(t *testing.T) {
	repo, item := integrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Go(func() {
			_, err := repo.Mutate(ctx, item.ID, func(ctx context.Context, current *submission.Submission) error {
				var fields map[string]int
				if err := json.Unmarshal(current.Content, &fields); err != nil {
					return err
				}
				fields[fmt.Sprintf("campo%d", i)] = i
				raw, err := json.Marshal(fields)
				if err != nil {
					return err
				}
				return current.UpdateContent(raw, time.Now().UTC())
			})
			if err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	stored, err := repo.FindByID(ctx, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]int
	if err := json.Unmarshal(stored.Content, &fields); err != nil {
		t.Fatal(err)
	}
	if len(fields) != 8 {
		t.Fatalf("se perdieron cambios concurrentes: %d/8", len(fields))
	}
	before, _ := submission.CanonicalContent(stored.Content)
	rollback := errors.New("fallo forzado antes del commit")
	_, err = repo.Mutate(ctx, item.ID, func(ctx context.Context, current *submission.Submission) error {
		current.Content = json.RawMessage(`{"alterado":true}`)
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatal(err)
	}
	stored, err = repo.FindByID(ctx, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	after, _ := submission.CanonicalContent(stored.Content)
	if string(before) != string(after) {
		t.Fatal("el rollback modificó la fila")
	}
}

func TestPostgresSealsOnceAndRejectsStaleUpdatesAndDeletes(t *testing.T) {
	repo, item := integrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if _, err := repo.Mutate(ctx, item.ID, func(ctx context.Context, current *submission.Submission) error {
		return current.UpdateContent(json.RawMessage(`{"z":[{"b":2,"a":1}],"a":true}`), time.Now().UTC())
	}); err != nil {
		t.Fatal(err)
	}
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() {
			_, err := repo.Mutate(ctx, item.ID, func(ctx context.Context, current *submission.Submission) error {
				return current.MarkAsSent(time.Now().UTC())
			})
			results <- err
		})
	}
	wg.Wait()
	close(results)
	success, conflicts := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, submission.ErrNotEditable) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || conflicts != 1 {
		t.Fatalf("envíos=%d conflictos=%d", success, conflicts)
	}
	stored, err := repo.FindByID(ctx, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := submission.CanonicalContent(stored.Content)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Hash != submission.HashContent(canonical) || stored.SubmittedAt == nil {
		t.Fatal("hash no reproducible desde JSONB")
	}
	if err := repo.Update(ctx, item); !errors.Is(err, submission.ErrNotEditable) {
		t.Fatalf("se aceptó escritura obsoleta: %v", err)
	}
	if err := repo.Delete(ctx, item.ID); !errors.Is(err, submission.ErrNotEditable) {
		t.Fatalf("se aceptó borrado tras el sello: %v", err)
	}
}
