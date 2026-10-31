package document

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"strconv"
	"time"
)

/** referenceSlot localiza el campo del respaldo sin aceptar índices de listas.
 * @returns {**DocumentReference, error} Campo del registro identificado por su ID estable.
 */
func referenceSlot(f *submission.Form, section int, record string) (**submission.DocumentReference, error) {
	switch section {
	case 4:
		if record == "" && f.Position != nil && f.Position.HasOrganization == "Sí" {
			return &f.Position.Document, nil
		}
	case 5:
		if f.Education != nil {
			for i := range f.Education.Degrees {
				if record != "" && f.Education.Degrees[i].ID == record {
					return &f.Education.Degrees[i].Document, nil
				}
			}
		}
	case 6:
		if f.Experience != nil {
			for i := range f.Experience.Experiences {
				if record != "" && f.Experience.Experiences[i].ID == record {
					return &f.Experience.Experiences[i].Document, nil
				}
			}
		}
	case 7:
		if f.Publications != nil {
			for i := range f.Publications.Publications {
				if record != "" && f.Publications.Publications[i].ID == record {
					return &f.Publications.Publications[i].Document, nil
				}
			}
		}
	case 8:
		if f.Training != nil {
			for i := range f.Training.Events {
				if record != "" && f.Training.Events[i].ID == record {
					return &f.Training.Events[i].Document, nil
				}
			}
		}
	}
	return nil, ErrInvalidRecord
}

func documentReference(d Document) *submission.DocumentReference {
	return &submission.DocumentReference{ID: d.ID.String(), Name: d.Name, Size: strconv.FormatInt(d.SizeBytes, 10), SHA256: d.SHA256, UploadedAt: d.CreatedAt.UTC().Format(time.RFC3339)}
}

/** VerifyReferences exige los respaldos al sellar y sustituye metadatos del cliente.
 * @param {Form} form - Formulario bajo el bloqueo de la postulación.
 * @returns {error} Regla incumplida; nunca valida por nombre o tamaño declarados.
 */
func (s *Service) VerifyReferences(ctx context.Context, id uuid.UUID, form *submission.Form) error {
	docs, err := s.repo.List(ctx, id)
	if err != nil {
		return err
	}
	byID := make(map[string]Document, len(docs))
	used := make(map[string]bool, len(docs))
	for _, d := range docs {
		byID[d.ID.String()] = d
	}
	check := func(section int, record string, ref **submission.DocumentReference) error {
		if *ref == nil {
			return fmt.Errorf("%w: falta el respaldo PDF de la sección %d", submission.ErrBusinessRule, section)
		}
		d, ok := byID[(*ref).ID]
		if !ok || d.Section != section || d.RecordID != record || d.SubmissionID != id {
			return fmt.Errorf("%w: el respaldo PDF no pertenece al registro de la sección %d", submission.ErrBusinessRule, section)
		}
		*ref = documentReference(d)
		used[d.ID.String()] = true
		return nil
	}
	if form.Position != nil && form.Position.HasOrganization == "Sí" {
		if err := check(4, "", &form.Position.Document); err != nil {
			return err
		}
	}
	if form.Education != nil {
		for i := range form.Education.Degrees {
			d := &form.Education.Degrees[i]
			if err := check(5, d.ID, &d.Document); err != nil {
				return err
			}
		}
	}
	if form.Experience != nil {
		for i := range form.Experience.Experiences {
			d := &form.Experience.Experiences[i]
			if err := check(6, d.ID, &d.Document); err != nil {
				return err
			}
		}
	}
	if form.Publications != nil {
		for i := range form.Publications.Publications {
			d := &form.Publications.Publications[i]
			if err := check(7, d.ID, &d.Document); err != nil {
				return err
			}
		}
	}
	if form.Training != nil {
		for i := range form.Training.Events {
			d := &form.Training.Events[i]
			if err := check(8, d.ID, &d.Document); err != nil {
				return err
			}
		}
	}
	if len(used) != len(docs) {
		return fmt.Errorf("%w: elimine los respaldos de registros retirados antes de enviar", submission.ErrBusinessRule)
	}
	return nil
}
