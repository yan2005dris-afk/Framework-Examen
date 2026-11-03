package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/mail"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
)

/** PrintEnvelope viaja cifrado con la cola; el worker no vuelve a consultar un perfil o formulario mutable. */
type PrintEnvelope struct {
	Version  string                `json:"version"`
	Snapshot submission.Submission `json:"snapshot"`
}

/** PrintingSender prepara la copia después del commit del sello, antes de entregar a SMTP. */
type PrintingSender struct {
	Next    mail.Sender
	Printer submission.Printer
}

func (s *PrintingSender) Send(ctx context.Context, message mail.Message) error {
	if len(message.RenderPayload) > 0 {
		var envelope PrintEnvelope
		if message.Kind != "submission.confirmation" || json.Unmarshal(message.RenderPayload, &envelope) != nil || envelope.Version != "ef12-print-v1" || s.Printer == nil {
			return mail.ErrUnpreparedMessage
		}
		if err := submission.ValidateSeal(&envelope.Snapshot); err != nil || message.ResourceID != envelope.Snapshot.ID.String() {
			return mail.ErrUnpreparedMessage
		}
		pdf, err := s.Printer.Render(ctx, &envelope.Snapshot)
		if err != nil {
			return fmt.Errorf("%w: no se pudo representar el formulario", mail.ErrUnpreparedMessage)
		}
		message.Attachments = []mail.Attachment{{Name: "EF12-" + envelope.Snapshot.ID.String() + ".pdf", MIME: "application/pdf", Data: pdf}}
		message.RenderPayload = nil
	}
	if s.Next == nil {
		return mail.ErrSMTPDisabled
	}
	return s.Next.Send(ctx, message)
}
