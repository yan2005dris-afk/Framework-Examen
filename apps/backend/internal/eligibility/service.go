// Package eligibility verifica condiciones mediante un proveedor controlado por el servidor, nunca por una bandera del cliente.
package eligibility

import (
	"context"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"time"
)

type Condition struct {
	Eligible bool
	Source   string
}

/** Provider representa la situación laboral; un fallo técnico no es un resultado no elegible. */
type Provider interface {
	Resolve(context.Context, uuid.UUID, string) (Condition, error)
}
type Recorder interface {
	Record(context.Context, uuid.UUID, *submission.EligibilityAssessment) error
}
type Service struct {
	provider Provider
	recorder Recorder
	now      func() time.Time
}

func NewService(p Provider, r Recorder) *Service { return &Service{p, r, time.Now} }
func (s *Service) Assess(ctx context.Context, item *submission.Submission, form *submission.Form) (*submission.EligibilityAssessment, error) {
	if item.CompetitionID == nil {
		return nil, submission.ErrInvalidCompetition
	}
	sector := ""
	if form.Position != nil {
		sector = form.Position.Sector
	}
	if sector != "" && sector != "afiliados" && sector != "jubilados" && sector != "empleadores" {
		return nil, submission.ErrInvalidContent
	}
	var evidence *submission.EligibilityAssessment
	if form.Personal != nil && form.Personal.Nationality != "" && form.Personal.Nationality != "Ecuatoriana" {
		if form.Personal.Nationality != "Extranjera" {
			return nil, submission.ErrInvalidContent
		}
		evidence = &submission.EligibilityAssessment{Eligible: false, Source: "nacionalidad_declarada", Reason: "nacionalidad_no_admitida", Message: "la nacionalidad debe ser ecuatoriana; no puede continuar con la postulación"}
	} else if sector != "" {
		if s.provider == nil {
			return nil, submission.ErrEligibilityUnavailable
		}
		condition, err := s.provider.Resolve(ctx, item.ApplicantID, sector)
		if err != nil || condition.Source == "" {
			return nil, submission.ErrEligibilityUnavailable
		}
		evidence = &submission.EligibilityAssessment{Eligible: condition.Eligible, Source: condition.Source, Reason: "sector_verificado", Message: map[string]string{"afiliados": "Postulante afiliado al IESS", "jubilados": "Postulante acredita jubilación", "empleadores": "Postulante es empleador"}[sector]}
		if !condition.Eligible {
			evidence.Reason = "sector_no_elegible"
			evidence.Message = submission.IneligibleMessage(sector)
		}
	}
	if evidence == nil {
		return nil, nil
	}
	id, err := uuid.NewV7()
	if err != nil {
		return nil, submission.ErrEligibilityUnavailable
	}
	if form.Personal != nil {
		evidence.Nationality = form.Personal.Nationality
	}
	evidence.ID = id
	evidence.ApplicantID = item.ApplicantID
	evidence.CompetitionID = *item.CompetitionID
	evidence.Sector = sector
	evidence.CheckedAt = s.now().UTC()
	if s.recorder == nil || s.recorder.Record(ctx, item.ID, evidence) != nil {
		return nil, submission.ErrEligibilityUnavailable
	}
	return evidence, nil
}
