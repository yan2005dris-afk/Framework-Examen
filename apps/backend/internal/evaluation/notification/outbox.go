package notification

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/evaluation"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/db"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/mail"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"time"
)

/** Outbox sólo comunica existencia de una actuación. Los informes y puntajes sensibles se consultan con sesión. */
type Outbox struct {
	Queue   *mail.Outbox
	Queries *sqlcgen.Queries
}

func (o *Outbox) Enqueue(ctx context.Context, submissionID, eventID uuid.UUID, action string) error {
	tx, ok := db.Transaction(ctx)
	if !ok || o == nil || o.Queue == nil || o.Queries == nil {
		return evaluation.ErrUnavailable
	}
	q := o.Queries.WithTx(tx)
	item, err := q.GetSubmissionByID(ctx, submissionID)
	if err != nil {
		return err
	}
	owner, err := q.GetApplicantByID(ctx, item.ApplicantID)
	if err != nil {
		return err
	}
	kind := "stage.result"
	if action == "reconsideracion.resuelta" {
		kind = "reconsideration.decision"
	}
	return o.Queue.Send(ctx, mail.Message{Kind: kind, ResourceID: eventID.String(), To: owner.Correo, Subject: "Actualización de su expediente de postulación", Body: fmt.Sprintf("Existe una nueva actuación en su expediente %s. Inicie sesión en el portal para consultar el resultado y, cuando corresponda, los plazos de reconsideración. Identificador de actuación: %s. Plantilla: resultado-etapa-v1.", item.Code, eventID), ExpiresAt: time.Now().UTC().Add(7 * 24 * time.Hour)})
}
