package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/document"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/db"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/storage"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	submissionpostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission/postgres"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func postgresFixture(t *testing.T) (*Repository, *pgxpool.Pool, *submission.Submission) {
	t.Helper()
	databaseURL := os.Getenv("DOCUMENT_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("defina DOCUMENT_TEST_DATABASE_URL para probar PostgreSQL real")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	admin, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	schema := "document_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	quoted := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+quoted); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if _, err := admin.Exec(cleanup, "DROP SCHEMA "+quoted+" CASCADE"); err != nil {
			t.Error(err)
		}
	})
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	cfg.MaxConns = 4
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
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
	owner, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO applicants(id,cedula,nombres,apellidos,correo) VALUES($1,'1710034008','Persona','Prueba','prueba@example.test')`, owner); err != nil {
		t.Fatal(err)
	}
	id, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	item, err := submission.NewSubmission(id, owner, "TEST-DOCUMENTS", json.RawMessage(`{"seccion5":{"titulos":[{"id":"nuevo-1"},{"id":"nuevo-2"}]}}`), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := submissionpostgres.NewRepository(sqlcgen.New(pool), db.NewTxManager(pool)).Create(ctx, item); err != nil {
		t.Fatal(err)
	}
	return NewRepository(pool, db.NewTxManager(pool)), pool, item
}

func TestPostgresConcurrentQuotaAndRollback(t *testing.T) {
	repo, _, item := postgresFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := repo.WithinSubmission(ctx, item.ApplicantID, item.ID, func(txctx context.Context, _ *submission.Submission, tx document.Repository) error {
		for i := range 24 {
			id, _ := uuid.NewV7()
			if err := tx.Create(txctx, &document.Document{ID: id, SubmissionID: item.ID, Section: 5, RecordID: fmt.Sprintf("existente-%d", i), Name: "prueba.pdf", MIME: "application/pdf", SizeBytes: document.MaxFileBytes, SHA256: strings.Repeat("0", 64), StorageKey: fmt.Sprintf("%s/5/%s.pdf", item.ID, id), CreatedAt: time.Now()}); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	objects, err := storage.NewSignedLocal(t.TempDir(), "http://objects.example.test", strings.Repeat("k", 32))
	if err != nil {
		t.Fatal(err)
	}
	service := document.NewService(repo, objects)
	data := make([]byte, document.MaxFileBytes)
	copy(data, "%PDF-1.7")
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for _, record := range []string{"nuevo-1", "nuevo-2"} {
		wg.Go(func() {
			_, err := service.Upload(ctx, document.UploadInput{ApplicantID: item.ApplicantID, SubmissionID: item.ID, Section: 5, RecordID: record, Name: "prueba.pdf", MIME: "application/pdf", Content: bytes.NewReader(data)})
			results <- err
		})
	}
	wg.Wait()
	close(results)
	success, quota := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, document.ErrQuotaExceeded) {
			quota++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || quota != 1 {
		t.Fatalf("éxitos=%d cuotas=%d", success, quota)
	}
	total, err := repo.TotalByApplicant(ctx, item.ApplicantID)
	if err != nil || total != document.MaxApplicantBytes {
		t.Fatal("cuota sin serializar", total, err)
	}
	forced := errors.New("fallo entre cambios")
	err = repo.WithinSubmission(ctx, item.ApplicantID, item.ID, func(txctx context.Context, current *submission.Submission, tx document.Repository) error {
		docs, err := tx.List(txctx, item.ID)
		if err != nil {
			return err
		}
		if err := tx.Delete(txctx, docs[0].ID); err != nil {
			return err
		}
		current.Content = json.RawMessage(`{}`)
		return forced
	})
	if !errors.Is(err, forced) {
		t.Fatal(err)
	}
	if total, _ := repo.TotalByApplicant(ctx, item.ApplicantID); total != document.MaxApplicantBytes {
		t.Fatal("rollback consumió datos")
	}
	if keys, _ := repo.PendingDeletions(ctx); len(keys) != 0 {
		t.Fatal("se encoló un rollback")
	}
}

func TestPostgresSealBlocksDocumentsAndCascadesQueueObjects(t *testing.T) {
	repo, pool, item := postgresFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	objects, err := storage.NewSignedLocal(t.TempDir(), "http://objects.example.test", strings.Repeat("k", 32))
	if err != nil {
		t.Fatal(err)
	}
	service := document.NewService(repo, objects)
	in := document.UploadInput{ApplicantID: item.ApplicantID, SubmissionID: item.ID, Section: 5, RecordID: "nuevo-1", Name: "prueba.pdf", MIME: "application/pdf", Content: strings.NewReader("%PDF-1.7")}
	d, err := service.Upload(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	subrepo := submissionpostgres.NewRepository(sqlcgen.New(pool), db.NewTxManager(pool))
	if _, err := subrepo.Mutate(ctx, item.ID, func(txctx context.Context, current *submission.Submission) error {
		// El verificador debe usar esta conexión: otro pool no es necesario.
		form, _ := submission.DecodeForm(current.Content)
		form.Education.Degrees = form.Education.Degrees[:1]
		if err := service.VerifyReferences(txctx, item.ID, form); err != nil {
			return err
		}
		raw, _ := json.Marshal(form)
		if err := current.UpdateContent(raw, time.Now()); err != nil {
			return err
		}
		return current.MarkAsSent(time.Now())
	}); err != nil {
		t.Fatal(err)
	}
	if err := service.Delete(ctx, item.ApplicantID, d.ID); !errors.Is(err, submission.ErrNotEditable) {
		t.Fatal(err)
	}
	in.RecordID = "nuevo-2"
	in.Content = strings.NewReader("%PDF-1.7")
	if _, err := service.Upload(ctx, in); !errors.Is(err, submission.ErrNotEditable) {
		t.Fatal(err)
	}
	if err := subrepo.Delete(ctx, item.ID); !errors.Is(err, submission.ErrNotEditable) {
		t.Fatal("borrado obsoleto", err)
	}
	// Borrador independiente: la cascada debe conservar la tarea de limpieza.
	secondID, _ := uuid.NewV7()
	second, err := submission.NewSubmission(secondID, item.ApplicantID, "TEST-CASCADE", item.Content, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := subrepo.Create(ctx, second); err != nil {
		t.Fatal(err)
	}
	in.SubmissionID = secondID
	in.RecordID = "nuevo-1"
	in.Content = strings.NewReader("%PDF-1.7")
	if _, err := service.Upload(ctx, in); err != nil {
		t.Fatal(err)
	}
	if err := subrepo.Delete(ctx, secondID); err != nil {
		t.Fatal(err)
	}
	keys, err := repo.PendingDeletions(ctx)
	if err != nil || len(keys) != 1 {
		t.Fatal("cascada sin cola", err)
	}
	if err := service.Cleanup(ctx); err != nil {
		t.Fatal(err)
	}
	if keys, _ := repo.PendingDeletions(ctx); len(keys) != 0 {
		t.Fatal("cola no vaciada")
	}
}
