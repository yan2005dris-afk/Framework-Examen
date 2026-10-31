package document

import (
	"context"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
)

/** WithAudit configura el registro obligatorio antes de entregar certificados. */
func (s *Service) WithAudit(recorder audit.Recorder) *Service { s.auditor = recorder; return s }

func (s *Service) authorizeAdministrativeRead(ctx context.Context, id uuid.UUID) error {
	item, err := s.repo.FindSubmission(ctx, id)
	if err != nil {
		return err
	}
	if item == nil {
		return submission.ErrNotFound
	}
	if item.Status != submission.StatusSubmitted || item.SubmittedAt == nil || item.Hash == "" {
		return submission.ErrForbidden
	}
	return nil
}

/** ListForReview entrega metadatos de certificados de un expediente enviado. */
func (s *Service) ListForReview(ctx context.Context, actor *identity.Applicant, id uuid.UUID) (items []Document, err error) {
	if !identity.HasPermission(actor, identity.PermissionDocuments) {
		return nil, submission.ErrForbidden
	}
	defer func() {
		err = audit.Record(ctx, s.auditor, audit.Event{ActorID: actor.ID, Role: actor.Role,
			Action: "certificados.consultados", ResourceID: id, ResourceType: "postulacion", CreatedAt: s.now().UTC()}, err)
		if err != nil {
			items = nil
		}
	}()
	if err = s.authorizeAdministrativeRead(ctx, id); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, id)
}

/** DownloadForReview emite URL temporal sin alterar la titularidad del certificado.
 * @returns {DownloadResponse, error} Enlace por cinco minutos, únicamente si se auditó.
 */
func (s *Service) DownloadForReview(ctx context.Context, actor *identity.Applicant, id uuid.UUID) (result DownloadResponse, err error) {
	if !identity.HasPermission(actor, identity.PermissionDocuments) {
		return result, submission.ErrForbidden
	}
	var submissionID uuid.UUID
	defer func() {
		err = audit.Record(ctx, s.auditor, audit.Event{ActorID: actor.ID, Role: actor.Role,
			SubmissionID: submissionID,
			Action:       "certificado.descarga_emitida", ResourceID: id, ResourceType: "documento", CreatedAt: s.now().UTC()}, err)
		if err != nil {
			result = DownloadResponse{}
		}
	}()
	d, err := s.repo.Find(ctx, id)
	if err != nil {
		return result, err
	}
	if d == nil {
		return result, ErrNotFound
	}
	submissionID = d.SubmissionID
	if err = s.authorizeAdministrativeRead(ctx, d.SubmissionID); err != nil {
		return result, err
	}
	url, err := s.storage.PresignGet(ctx, d.StorageKey, d.Name, DownloadTTL)
	if err != nil {
		return result, err
	}
	return DownloadResponse{URL: url, ExpiresAt: s.now().UTC().Add(DownloadTTL)}, nil
}
