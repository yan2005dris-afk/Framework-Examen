package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/evaluation"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/db"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
)

/** RecordFinalTheory es exclusivamente un puerto interno para el commit definitivo de T27. No tiene ruta HTTP ni simula el motor de examen. */
func (r *Repository) RecordFinalTheory(ctx context.Context, result evaluation.Result) error {
	tx, ok := db.Transaction(ctx)
	if !ok {
		return audit.ErrTransactionRequired
	}
	q := r.queries.WithTx(tx)
	if result.ID == uuid.Nil || result.SourceID == uuid.Nil || result.SubmissionID == uuid.Nil || result.Phase != "oposicion_teorica" || result.Score == nil || result.Maximum == nil || *result.Maximum < 1 || *result.Maximum > 1e9 || *result.Score < 0 || *result.Score > *result.Maximum || result.PolicyID == uuid.Nil || len(result.PolicyHash) != 64 || result.Motive == "" {
		return evaluation.ErrInvalid
	}
	c, err := loadCase(ctx, q, result.SubmissionID, uuid.Nil, true)
	if err != nil {
		return err
	}
	if result.SnapshotHash != c.Snapshot.Hash {
		return evaluation.ErrConflict
	}
	prior, err := q.GetEvaluationRecord(ctx, sqlcgen.GetEvaluationRecordParams{ID: result.ID, SubmissionID: result.SubmissionID})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if err == nil {
		var saved evaluation.Result
		if json.Unmarshal(prior.Payload, &saved) != nil {
			return evaluation.ErrConflict
		}
		a, _ := evaluation.Digest(saved)
		b, _ := evaluation.Digest(result)
		if a == b && prior.Action == "teorica.finalizada" {
			return nil
		}
		return evaluation.ErrConflict
	}
	old, exists := c.Sources["oposicion_teorica"]
	expected := int64(1)
	if exists {
		expected = old.Version + 1
		if result.PublishedAt.Before(old.PublishedAt) {
			return evaluation.ErrInvalid
		}
		if result.SupersedesID == nil || *result.SupersedesID != old.ID {
			return evaluation.ErrConflict
		}
	} else if result.SupersedesID != nil {
		return evaluation.ErrConflict
	}
	if result.Version != expected || result.PublishedAt.IsZero() || result.PublishedAt.After(c.Now) || result.PublishedAt.Before(*c.Snapshot.SubmittedAt) {
		return evaluation.ErrInvalid
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return err
	}
	if err = q.CreateEvaluationRecord(ctx, sqlcgen.CreateEvaluationRecordParams{ID: result.ID, SubmissionID: result.SubmissionID, Action: "teorica.finalizada", Phase: result.Phase, Payload: raw, CreatedAt: timestamp(result.PublishedAt)}); err != nil {
		return mapError(err)
	}
	if r.auditor == nil || r.notifier == nil {
		return evaluation.ErrUnavailable
	}
	if err = r.auditor.RecordWorkflow(ctx, audit.Event{Role: "sistema", Action: "evaluacion.finalizada", SubmissionID: result.SubmissionID, ResourceID: result.SubmissionID, ResourceType: "postulacion", Result: "permitido", CreatedAt: c.Now}); err != nil {
		return err
	}
	return r.notifier.Enqueue(ctx, result.SubmissionID, result.ID, "evaluacion.finalizada")
}
