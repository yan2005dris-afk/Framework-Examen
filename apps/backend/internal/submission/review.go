package submission

import (
	"context"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
)

/** WithAudit configura la trazabilidad obligatoria de consultas administrativas. */
func (s *Service) WithAudit(recorder audit.Recorder) *Service {
	s.auditor = recorder
	return s
}

/** GetForReview consulta el snapshot enviado sin precargarlo ni modificarlo.
 * @returns {Submission, error} Expediente sellado para personal autorizado.
 */
func (s *Service) GetForReview(ctx context.Context, actor *identity.Applicant, id uuid.UUID) (item *Submission, err error) {
	if !identity.HasPermission(actor, identity.PermissionReview) {
		return nil, ErrForbidden
	}
	defer func() {
		err = audit.Record(ctx, s.auditor, audit.Event{ActorID: actor.ID, Role: actor.Role,
			Action: "formulario.consultado", ResourceID: id, ResourceType: "postulacion", CreatedAt: s.now().UTC()}, err)
		if err != nil {
			item = nil
		}
	}()
	item, err = s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, ErrNotFound
	}
	if item.Status != StatusSubmitted || item.SubmittedAt == nil || item.Hash == "" {
		return nil, ErrForbidden
	}
	return item, nil
}
