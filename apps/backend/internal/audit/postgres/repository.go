package postgres

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/db"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
)

/** Repository persiste eventos de acceso sin documentos ni direcciones firmadas. */
type Repository struct {
	queries      *sqlcgen.Queries
	transactions *db.TxManager
}

/** NewRepository conecta la auditoría con sus consultas tipadas. */
func NewRepository(queries *sqlcgen.Queries, managers ...*db.TxManager) *Repository {
	r := &Repository{queries: queries}
	if len(managers) > 0 {
		r.transactions = managers[0]
	}
	return r
}

/** Record confirma el evento antes de permitir entregar los datos solicitados. */
func (r *Repository) Record(ctx context.Context, event audit.Event) error {
	if event.SubmissionID == [16]byte{} && event.ResourceType == "postulacion" {
		event.SubmissionID = event.ResourceID
	}
	params := sqlcgen.CreateAdministrativeAccessEventParams{
		SubmissionID: pgtype.UUID{Bytes: event.SubmissionID, Valid: event.SubmissionID != [16]byte{}},
		ActorID:      event.ActorID, ActorRole: event.Role, Action: event.Action,
		ResourceID: event.ResourceID, ResourceType: event.ResourceType, Result: event.Result,
		CreatedAt: event.CreatedAt,
	}
	if tx, ok := db.Transaction(ctx); ok {
		return r.queries.WithTx(tx).CreateAdministrativeAccessEvent(ctx, params)
	}
	if r.transactions != nil {
		return r.transactions.WithinTx(ctx, func(tx pgx.Tx) error { return r.queries.WithTx(tx).CreateAdministrativeAccessEvent(ctx, params) })
	}
	return r.queries.CreateAdministrativeAccessEvent(ctx, params)
}
