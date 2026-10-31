// Package notification adapta la confirmación de postulaciones al correo SMTP.
package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/mail"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
)

/** ApplicantReader consulta el correo registrado del dueño de la postulación. */
type ApplicantReader interface {
	GetByID(context.Context, uuid.UUID) (*identity.Applicant, error)
}

/** Confirmation envía al correo registrado, nunca a uno arbitrario del cuerpo HTTP. */
type Confirmation struct {
	Sender     mail.Sender
	Applicants ApplicantReader
}

/**
 * Confirm remite el contenido sellado, código, fecha y hash al postulante.
 * @param {Submission} item - Postulación ya confirmada en PostgreSQL.
 * @returns {error} Fallo de consulta o de entrega SMTP.
 */
func (c *Confirmation) Confirm(ctx context.Context, item *submission.Submission) error {
	applicant, err := c.Applicants.GetByID(ctx, item.ApplicantID)
	if err != nil {
		return err
	}
	if applicant == nil || c.Sender == nil {
		return fmt.Errorf("no se pudo preparar la confirmación")
	}
	return c.Sender.Send(ctx, mail.Message{
		Kind: "submission.confirmation", ResourceID: item.ID.String(), ExpiresAt: item.SubmittedAt.Add(7 * 24 * time.Hour),
		To:      applicant.Email,
		Subject: "Confirmación de postulación " + item.Code,
		Body:    fmt.Sprintf("Su postulación %s fue enviada correctamente.\nFecha: %s\nHash SHA-256: %s\n\nFormulario sellado:\n%s\n\nImprima el formulario y entréguelo con firma manuscrita según las condiciones del concurso.", item.Code, item.SubmittedAt.Format("2006-01-02T15:04:05Z07:00"), item.Hash, item.Content),
	})
}
