package evaluation

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"net/http"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

func (s *Service) withEvidence(ctx context.Context, a *identity.Applicant, id uuid.UUID, cmd Command, input *EvidenceInput, applicant bool, mutate func(*Case, *Evidence) (Change, error)) (receipt Receipt, err error) {
	c, err := s.repo.ReadCase(ctx, id, uuid.Nil)
	if err != nil {
		return Receipt{}, err
	}
	if applicant && c.Snapshot.ApplicantID != a.ID {
		return Receipt{}, ErrForbidden
	}
	var e *Evidence
	if input != nil {
		if s.storage == nil || len(input.PDF) == 0 || len(input.PDF) > MaxEvidenceBytes || !bytes.HasPrefix(input.PDF, []byte("%PDF-")) || http.DetectContentType(input.PDF) != "application/pdf" || !strings.Contains(string(input.PDF[max(0, len(input.PDF)-1024):]), "%%EOF") || !utf8.ValidString(input.Name) || !validText(input.Name, 150) || strings.ContainsAny(input.Name, "/\\") || strings.IndexFunc(input.Name, unicode.IsControl) >= 0 {
			return Receipt{}, ErrInvalid
		}
		sum := sha256.Sum256(input.PDF)
		e = &Evidence{ID: newID(), SubmissionID: id, ApplicantID: c.Snapshot.ApplicantID, ActorID: a.ID, ApplicantUpload: applicant, Name: input.Name, Size: int64(len(input.PDF)), SHA256: hex.EncodeToString(sum[:]), CreatedAt: time.Now().UTC()}
		if e.ID == uuid.Nil {
			return Receipt{}, ErrUnavailable
		}
		e.StorageKey = fmt.Sprintf("evaluation/%s/%s.pdf", id, e.ID)
		if err = s.repo.BeginUpload(ctx, e); err != nil {
			return Receipt{}, err
		}
		defer func() {
			if err != nil || !receipt.Applied {
				cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if removeErr := s.storage.Delete(cleanup, e.StorageKey); removeErr == nil {
					_ = s.repo.CompleteUpload(cleanup, e.ID)
				}
			}
		}()
		if err = s.storage.Save(ctx, e.StorageKey, bytes.NewReader(input.PDF)); err != nil {
			return Receipt{}, ErrUnavailable
		}
	}
	return s.repo.Execute(ctx, a, id, cmd, func(c *Case) (Change, error) { return mutate(c, e) })
}

/** Download exige titularidad o lectura administrativa auditada y emite URL de cinco minutos, nunca proxy del PDF. */
func (s *Service) Download(ctx context.Context, a *identity.Applicant, id, evidenceID uuid.UUID, administrative bool) (url string, err error) {
	if _, err = s.Read(ctx, a, id, uuid.Nil, administrative); err != nil {
		return "", err
	}
	e, err := s.repo.GetEvidence(ctx, evidenceID)
	if err != nil {
		return "", err
	}
	if e.SubmissionID != id {
		return "", ErrForbidden
	}
	if administrative {
		defer func() {
			err = audit.Record(ctx, s.auditor, audit.Event{ActorID: a.ID, Role: a.Role, SubmissionID: id, Action: "evaluacion.respaldo_descarga", ResourceID: evidenceID, ResourceType: "documento", CreatedAt: time.Now().UTC()}, err)
			if err != nil {
				url = ""
			}
		}()
	}
	if !administrative && !e.ApplicantUpload && !e.Published {
		return "", ErrForbidden
	}
	if s.storage == nil {
		return "", ErrUnavailable
	}
	return s.storage.PresignGet(ctx, e.StorageKey, e.Name, 5*time.Minute)
}

/** CleanupUploads retira sólo intentos vencidos; los objetos de evidencia confirmada nunca se purgan automáticamente. */
func (s *Service) CleanupUploads(ctx context.Context) error {
	if s.storage == nil {
		return ErrUnavailable
	}
	items, err := s.repo.ExpiredUploads(ctx)
	if err != nil {
		return err
	}
	for _, e := range items {
		if err = s.storage.Delete(ctx, e.StorageKey); err != nil {
			return err
		}
		if err = s.repo.CompleteUpload(ctx, e.ID); err != nil {
			return err
		}
	}
	return nil
}
