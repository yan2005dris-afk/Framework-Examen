package submission

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"time"
)

var ErrInvalidSeal = errors.New("el contenido no coincide con el sello del envío")
var ErrPrintTooLarge = errors.New("el formulario supera el límite técnico de impresión")
var ErrPrintUnavailable = errors.New("no se pudo generar la copia final del formulario")

/** Printer produce una copia del mismo snapshot, sin consultar perfiles ni catálogos actuales. */
type Printer interface {
	Render(context.Context, *Submission) ([]byte, error)
}

func (s *Service) WithPrinter(p Printer) *Service { s.printer = p; return s }

/** ValidateSeal verifica estado y hash canónico antes de usar un envío como evidencia. */
func ValidateSeal(item *Submission) error {
	if item == nil || item.ID == uuid.Nil || item.ApplicantID == uuid.Nil || item.Status != StatusSubmitted || item.SubmittedAt == nil || item.SubmittedAt.IsZero() {
		return ErrInvalidSeal
	}
	raw, err := CanonicalContent(item.Content)
	if err != nil || HashContent(raw) != item.Hash {
		return ErrInvalidSeal
	}
	return nil
}

/** PrintFinal exige titularidad o permiso administrativo y audita antes de entregar bytes privados. */
func (s *Service) PrintFinal(ctx context.Context, actor *identity.Applicant, id uuid.UUID, admin bool) (data []byte, err error) {
	if actor == nil || actor.ID == uuid.Nil || admin && !identity.HasPermission(actor, identity.PermissionReview) {
		return nil, ErrForbidden
	}
	if admin {
		defer func() {
			err = audit.Record(ctx, s.auditor, audit.Event{ActorID: actor.ID, Role: actor.Role, Action: "formulario.impreso", ResourceID: id, ResourceType: "postulacion", CreatedAt: time.Now().UTC()}, err)
			if err != nil {
				data = nil
			}
		}()
	}
	item, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, ErrNotFound
	}
	if !admin && item.ApplicantID != actor.ID {
		return nil, ErrForbidden
	}
	if err = ValidateSeal(item); err != nil {
		return nil, err
	}
	if s.printer == nil {
		return nil, ErrPrintUnavailable
	}
	return s.printer.Render(ctx, item)
}
