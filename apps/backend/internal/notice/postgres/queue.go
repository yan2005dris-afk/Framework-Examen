package postgres

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/notice"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/db"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/mail"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"time"
)

type Queue struct {
	Outbox  *mail.Outbox
	Queries *sqlcgen.Queries
}

/** Enqueue integra las etapas existentes; no envía SMTP ni copia puntajes a correo. */
func (q *Queue) Enqueue(ctx context.Context, sub, source uuid.UUID, action string) error {
	tx, ok := db.Transaction(ctx)
	if sub == uuid.Nil || source == uuid.Nil || !ok || q == nil || q.Outbox == nil || q.Queries == nil {
		return notice.ErrUnavailable
	}
	label, kind, template := "", "stage.result", "resultado-etapa-v2"
	switch action {
	case "resultado.publicado":
		label = "requisitos y méritos"
	case "psicotecnica.publicada":
		label = "oposición psicotécnica"
	case "evaluacion.finalizada":
		label = "oposición teórica"
	case "expediente.consolidado":
		label = "resultado consolidado"
	case "reconsideracion.resuelta":
		label = "resolución de reconsideración"
		kind = "reconsideration.decision"
		template = "reconsideracion-v2"
	case "lista.publicada":
		label = "publicación de lista de elegibles"
		kind = "results.list"
		template = "lista-elegibles-v1"
	default:
		return notice.ErrInvalid
	}
	queries := q.Queries.WithTx(tx)
	item, err := queries.GetSubmissionByID(ctx, sub)
	if err != nil {
		return err
	}
	owner, err := queries.GetApplicantByID(ctx, item.ApplicantID)
	if err != nil {
		return err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return err
	}
	if err = queries.CreateStageNotice(ctx, sqlcgen.CreateStageNoticeParams{ID: id, SubmissionID: sub, SourceID: source, Action: action, TemplateVersion: template}); err != nil {
		return err
	}
	event, err := queries.GetStageNotice(ctx, sqlcgen.GetStageNoticeParams{SubmissionID: sub, SourceID: source, Action: action})
	if err != nil {
		return err
	}
	return q.Outbox.Send(ctx, mail.Message{Kind: kind, ResourceID: event.ID.String(), To: owner.Correo, Subject: "Actualización de su expediente de postulación", Body: fmt.Sprintf("Hay una actualización de %s en su expediente %s. Inicie sesión en el portal para consultar el resultado, sus respaldos y los plazos aplicables. Actuación: %s. Plantilla: %s.", label, item.Code, source, event.TemplateVersion), ExpiresAt: time.Now().UTC().Add(7 * 24 * time.Hour)})
}
