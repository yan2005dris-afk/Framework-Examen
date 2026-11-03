package postgres

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/db"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
)

type Repository struct{ queries *sqlcgen.Queries }

func NewRepository(q *sqlcgen.Queries) *Repository { return &Repository{q} }

/** Record exige el commit del agregado: la evidencia no puede sobrevivir a un rollback del guardado. */
func (r *Repository) Record(ctx context.Context, id uuid.UUID, e *submission.EligibilityAssessment) error {
	tx, ok := db.Transaction(ctx)
	if !ok {
		return submission.ErrEligibilityUnavailable
	}
	raw, err := json.Marshal(e)
	if err != nil {
		return err
	}
	return r.queries.WithTx(tx).RecordEligibilityCheck(ctx, sqlcgen.RecordEligibilityCheckParams{ID: e.ID, SubmissionID: id, ApplicantID: e.ApplicantID, CompetitionID: e.CompetitionID, Sector: e.Sector, Result: e.Eligible, Source: e.Source, CheckedAt: pgtype.Timestamptz{Time: e.CheckedAt, Valid: true}, Assessment: raw})
}
