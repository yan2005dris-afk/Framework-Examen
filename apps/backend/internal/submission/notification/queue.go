package notification

import (
	"context"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/db"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/mail"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
)

/** QueuedConfirmation usa un outbox cifrado, nunca el remitente SMTP directo. */
type QueuedConfirmation struct {
	Outbox     *mail.Outbox
	Applicants ApplicantReader
}

/** Enqueue participa del commit de la postulación, sin publicar un correo antes de confirmar. */
func (q *QueuedConfirmation) Enqueue(ctx context.Context, item *submission.Submission) error {
	if _, ok := db.Transaction(ctx); !ok {
		return submission.ErrNotificationUnavailable
	}
	return (&Confirmation{Sender: q.Outbox, Applicants: q.Applicants}).Confirm(ctx, item)
}

/** State devuelve estado de cola/SMTP, no una promesa de llegada al buzón. */
func (q *QueuedConfirmation) State(ctx context.Context, id uuid.UUID) (*submission.DeliveryStatus, error) {
	state, err := q.Outbox.State(ctx, id)
	if err != nil {
		return nil, err
	}
	return &submission.DeliveryStatus{Status: state.Status, Attempts: state.Attempts, ErrorCode: state.ErrorCode, UpdatedAt: state.UpdatedAt}, nil
}
