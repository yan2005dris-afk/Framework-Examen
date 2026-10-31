package postgres

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/db"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/shared/domain"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/testutil"
	"testing"
	"time"
)

func TestPostgresHistorySurvivesDeletionIsImmutableAndWorkflowRollsBack(t *testing.T) {
	pool := testutil.Postgres(t, "AUDIT_TEST_DATABASE_URL")
	ctx := context.Background()
	owner, id := uuid.New(), uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO applicants(id,cedula,nombres,apellidos,correo) VALUES($1,'1710034008','Persona','Prueba','audit@example.test')`, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO submissions(id,applicant_id,code,competition_id) VALUES($1,$2,'AUDIT-TEST',$3)`, id, owner, testutil.OpenCompetition(t, pool)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO submission_events(submission_id,applicant_id,event_type) VALUES($1,$2,'guardada')`, id, owner); err != nil {
		t.Fatal(err)
	}
	repo := NewRepository(sqlcgen.New(pool))
	event := audit.Event{SubmissionID: id, Role: "sistema", Action: "elegibilidad.verificada", ResourceID: id, ResourceType: "postulacion", Result: "permitido", CreatedAt: time.Now()}
	if err := repo.RecordWorkflow(ctx, event); !errors.Is(err, audit.ErrTransactionRequired) {
		t.Fatal("evento fuera del commit", err)
	}
	sentinel := errors.New("revertir")
	err := db.NewTxManager(pool).WithinTx(ctx, func(tx pgx.Tx) error {
		if err := repo.RecordWorkflow(db.WithTransaction(ctx, tx), event); err != nil {
			return err
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatal(err)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM workflow_events`).Scan(&count); err != nil || count != 0 {
		t.Fatal("evento no revertido", err)
	}
	if err := db.NewTxManager(pool).WithinTx(ctx, func(tx pgx.Tx) error { return repo.RecordWorkflow(db.WithTransaction(ctx, tx), event) }); err != nil {
		t.Fatal(err)
	}
	if err := repo.Record(ctx, audit.Event{ActorID: uuid.New(), Role: "analista", Action: "formulario.consultado", ResourceID: id, ResourceType: "postulacion", Result: "rechazado", CreatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"submission_events", "workflow_events", "administrative_access_events", "applicant_role_events"} {
		for _, statement := range []string{"UPDATE " + table + " SET id=id", "DELETE FROM " + table, "TRUNCATE " + table} {
			if _, err := pool.Exec(ctx, statement); err == nil {
				t.Fatal("historial modificable: ", statement)
			}
		}
	}
	if _, err := pool.Exec(ctx, `DELETE FROM submissions WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM applicants WHERE id=$1`, owner); err != nil {
		t.Fatal(err)
	}
	history, err := repo.History(ctx, id, domain.Position{}, 20)
	if err != nil || len(history) != 4 {
		t.Fatalf("historial perdido: %d %v", len(history), err)
	}
	got, err := repo.Owner(ctx, id)
	if err != nil || got == nil || got.ApplicantID != owner || got.Status != "deleted" {
		t.Fatal("titularidad histórica perdida", err)
	}
	systemFound := false
	for _, entry := range history {
		if entry.Role == "sistema" {
			systemFound = entry.ActorID == nil
		}
	}
	if !systemFound {
		t.Fatal("actor nulo del sistema no se interpretó")
	}
}
