package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/db"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
)

/** SaveSubmission crea el agregado usando una sola conexión y un único commit.
 * Las secciones permanecen en content JSONB. El callback recibe el ID definitivo
 * y el contexto que los repositorios de documentos resuelven con db.Executor.
 * @param {RelatedChanges} related - Escritura de metadatos y referencias, sin iniciar otra transacción.
 * @returns {error} Ante cualquier fallo se revierten cabecera, documentos y evento.
 */
func (r *Repository) SaveSubmission(ctx context.Context, item *submission.Submission, related submission.RelatedChanges) error {
	if r.transactions == nil {
		return errors.New("transacciones no configuradas")
	}
	working := *item
	working.Content = append(json.RawMessage(nil), item.Content...)
	err := r.transactions.WithinTx(ctx, func(tx pgx.Tx) error {
		txctx := db.WithTransaction(ctx, tx)
		queries := r.queries.WithTx(tx)
		// El mismo orden que T4 evita carreras sobre la cuota entre postulaciones.
		if _, err := queries.LockDocumentApplicant(txctx, working.ApplicantID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return submission.ErrInvalidApplicant
			}
			return err
		}
		adapter := &Repository{queries: queries}
		if err := adapter.Create(txctx, &working); err != nil {
			return err
		}
		if related != nil {
			if err := related(txctx, &working); err != nil {
				return err
			}
		}
		if err := adapter.Update(txctx, &working); err != nil {
			return err
		}
		return adapter.RecordEvent(txctx, &working, submission.EventSaved)
	})
	if err != nil {
		return err
	}
	*item = working
	return nil
}

/** RecordEvent inserta auditoría con las queries ligadas a la transacción del cambio.
 * No almacena el formulario, nombres, cédula ni claves del almacenamiento en payload.
 * @returns {error} Fallo de auditoría que el llamador debe propagar para provocar rollback.
 */
func (r *Repository) RecordEvent(ctx context.Context, item *submission.Submission, eventType string) error {
	id, err := uuid.NewV7()
	if err != nil {
		return err
	}
	return r.queries.CreateSubmissionEvent(ctx, sqlcgen.CreateSubmissionEventParams{
		ID: id, SubmissionID: item.ID, ApplicantID: item.ApplicantID,
		EventType: eventType, CreatedAt: item.UpdatedAt,
	})
}
