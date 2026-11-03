package postgres

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/db"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/shared/domain"
)

/** Owner recupera titularidad vigente o la del borrador eliminado sin revivir datos personales. */
func (r *Repository) Owner(ctx context.Context, id uuid.UUID) (*audit.Owner, error) {
	row, err := r.queries.GetHistoryOwner(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &audit.Owner{ApplicantID: row.ApplicantID, Status: row.Status}, nil
}

/** History utiliza UNION ALL con índice por recurso en cada historial y keyset estable. */
func (r *Repository) History(ctx context.Context, id uuid.UUID, p domain.Position, limit int) ([]audit.Entry, error) {
	rows, err := r.queries.ListSubmissionHistory(ctx, sqlcgen.ListSubmissionHistoryParams{
		TargetID: id, HasCursor: !p.At.IsZero(), BeforeAt: pgtype.Timestamptz{Time: p.At, Valid: true}, BeforeID: p.ID, RowLimit: int32(limit)})
	if err != nil {
		return nil, err
	}
	items := make([]audit.Entry, 0, len(rows))
	for _, row := range rows {
		var actor *uuid.UUID
		if row.ActorID != uuid.Nil {
			id := row.ActorID
			actor = &id
		}
		items = append(items, audit.Entry{ID: row.ID, ActorID: actor, Role: row.ActorRole, Action: row.Action,
			ResourceID: row.ResourceID, ResourceType: row.ResourceType, Result: row.Result, CreatedAt: row.CreatedAt})
	}
	return items, nil
}

/** RecordWorkflow se une obligatoriamente a la transacción del módulo que cambió el estado.
 * @returns {error} El llamador debe propagar el error para revertir el cambio y su evento.
 */
func (r *Repository) RecordWorkflow(ctx context.Context, e audit.Event) error {
	if !audit.ValidWorkflow(e) {
		return audit.ErrInvalidEvent
	}
	tx, ok := db.Transaction(ctx)
	if !ok {
		return audit.ErrTransactionRequired
	}
	return r.queries.WithTx(tx).CreateWorkflowEvent(ctx, sqlcgen.CreateWorkflowEventParams{
		SubmissionID: pgtype.UUID{Bytes: e.SubmissionID, Valid: e.SubmissionID != uuid.Nil},
		ActorID:      pgtype.UUID{Bytes: e.ActorID, Valid: e.ActorID != uuid.Nil}, ActorRole: e.Role, Action: e.Action,
		ResourceID: e.ResourceID, ResourceType: e.ResourceType, Result: e.Result,
		CreatedAt: pgtype.Timestamptz{Time: e.CreatedAt, Valid: true}})
}
