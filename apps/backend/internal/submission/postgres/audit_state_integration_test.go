package postgres

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/shared/domain"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"strings"
	"testing"
	"time"
)

func TestStateJournalCapturesActorBeforeAfterAndRollsBackOnFailure(t *testing.T) {
	r, item := integrationRepository(t)
	request := uuid.New()
	ctx := domain.WithAuditActor(domain.WithAuditRequest(context.Background(), request), item.ApplicantID, "postulante")
	_, err := r.Mutate(ctx, item.ID, func(_ context.Context, s *submission.Submission) error {
		return s.UpdateContent(json.RawMessage(`{"datoPrivado":"NO-COPIAR-ESTE-CONTENIDO"}`), time.Now())
	})
	if err != nil {
		t.Fatal(err)
	}
	err = r.transactions.WithinTx(ctx, func(tx pgx.Tx) error {
		var actor uuid.UUID
		var role, reason string
		var req uuid.UUID
		var before, after []byte
		if e := tx.QueryRow(ctx, `SELECT actor_id,actor_role,reason,request_id,previous_state,next_state FROM audit_state_events WHERE source_table='submissions' AND resource_id=$1 ORDER BY created_at DESC,id DESC LIMIT 1`, item.ID).Scan(&actor, &role, &reason, &req, &before, &after); e != nil {
			return e
		}
		if actor != item.ApplicantID || role != "postulante" || req != request || reason == "" || len(before) == 0 || len(after) == 0 || string(before) == string(after) || strings.Contains(string(after), "NO-COPIAR") {
			t.Fatal("atribución o proyección insegura")
		}
		_, e := tx.Exec(ctx, `CREATE FUNCTION reject_state_journal() RETURNS trigger LANGUAGE plpgsql AS $$BEGIN RAISE EXCEPTION 'QA: auditoría no disponible';END$$; CREATE TRIGGER reject_state_journal BEFORE INSERT ON audit_state_events FOR EACH ROW EXECUTE FUNCTION reject_state_journal()`)
		return e
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.Mutate(ctx, item.ID, func(_ context.Context, s *submission.Submission) error {
		return s.UpdateContent(json.RawMessage(`{"noDebeConfirmarse":true}`), time.Now())
	})
	if err == nil {
		t.Fatal("cambio confirmado sin auditoría")
	}
	saved, err := r.FindByID(ctx, item.ID)
	if err != nil || strings.Contains(string(saved.Content), "noDebeConfirmarse") {
		t.Fatal("rollback incompleto", err)
	}
	err = r.transactions.WithinTx(ctx, func(tx pgx.Tx) error {
		_, e := tx.Exec(ctx, `DROP TRIGGER reject_state_journal ON audit_state_events;DROP FUNCTION reject_state_journal()`)
		return e
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, sql := range []string{`UPDATE audit_state_events SET reason='alterado'`, `DELETE FROM audit_state_events`, `TRUNCATE audit_state_events`} {
		if err = r.transactions.WithinTx(ctx, func(tx pgx.Tx) error { _, e := tx.Exec(ctx, sql); return e }); err == nil {
			t.Fatal("historial alterable")
		}
	}
	// La referencia pseudónima sobrevive a eliminar un borrador.
	if err = r.Delete(ctx, item.ID); err != nil {
		t.Fatal(err)
	}
	err = r.transactions.WithinTx(ctx, func(tx pgx.Tx) error {
		var n int
		e := tx.QueryRow(ctx, `SELECT count(*) FROM audit_state_events WHERE resource_id=$1`, item.ID).Scan(&n)
		if e == nil && n < 3 {
			t.Fatal("se perdió el historial al borrar el borrador")
		}
		return e
	})
	if err != nil {
		t.Fatal(err)
	}
}
