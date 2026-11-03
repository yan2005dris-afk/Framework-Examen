package notification

import (
	"context"
	"github.com/google/uuid"
	noticepostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/notice/postgres"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/mail"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
)

/** Outbox adapta el puerto de evaluación a las notificaciones de etapa compartidas. */
type Outbox struct {
	Queue   *mail.Outbox
	Queries *sqlcgen.Queries
}

func (o *Outbox) Enqueue(ctx context.Context, sub, event uuid.UUID, action string) error {
	return (&noticepostgres.Queue{Outbox: o.Queue, Queries: o.Queries}).Enqueue(ctx, sub, event, action)
}
