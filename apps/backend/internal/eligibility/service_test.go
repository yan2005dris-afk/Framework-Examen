package eligibility

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"testing"
)

type providerStub struct {
	value bool
	fail  bool
}

func (p providerStub) Resolve(context.Context, uuid.UUID, string) (Condition, error) {
	if p.fail {
		return Condition{}, errors.New("caída técnica")
	}
	return Condition{p.value, "simulacion_controlada_servidor"}, nil
}

type recorderStub struct {
	items []*submission.EligibilityAssessment
	fail  bool
}

func (r *recorderStub) Record(_ context.Context, _ uuid.UUID, e *submission.EligibilityAssessment) error {
	if r.fail {
		return errors.New("sin commit")
	}
	r.items = append(r.items, e)
	return nil
}
func TestEligibilityUsesServerProviderAndBindsEvidence(t *testing.T) {
	period := uuid.New()
	item := &submission.Submission{ID: uuid.New(), ApplicantID: uuid.New(), CompetitionID: &period}
	for _, sector := range []string{"afiliados", "jubilados", "empleadores"} {
		for _, eligible := range []bool{true, false} {
			t.Run(sector+map[bool]string{true: "_elegible", false: "_no_elegible"}[eligible], func(t *testing.T) {
				r := &recorderStub{}
				s := NewService(providerStub{value: eligible}, r)
				clientFlag := !eligible
				form := &submission.Form{Position: &submission.PositionSection{Sector: sector, Eligible: &clientFlag}}
				e, err := s.Assess(context.Background(), item, form)
				if err != nil || e.Eligible != eligible || e.ApplicantID != item.ApplicantID || e.CompetitionID != period || e.Sector != sector || e.CheckedAt.IsZero() || len(r.items) != 1 || e.Source != "simulacion_controlada_servidor" {
					t.Fatal(e, err)
				}
				if !eligible && e.Message != submission.IneligibleMessage(sector) {
					t.Fatal("mensaje de cierre incorrecto")
				}
			})
		}
	}
}
func TestTechnicalFailureNeverBecomesIneligibility(t *testing.T) {
	period := uuid.New()
	r := &recorderStub{}
	s := NewService(providerStub{fail: true}, r)
	e, err := s.Assess(context.Background(), &submission.Submission{ID: uuid.New(), ApplicantID: uuid.New(), CompetitionID: &period}, &submission.Form{Position: &submission.PositionSection{Sector: "afiliados"}})
	if !errors.Is(err, submission.ErrEligibilityUnavailable) || e != nil || len(r.items) != 0 {
		t.Fatal("fallo técnico registrado como cierre", e, err)
	}
}
func TestNationalityClosesWithoutPretendingCivilVerification(t *testing.T) {
	period := uuid.New()
	r := &recorderStub{}
	s := NewService(nil, r)
	e, err := s.Assess(context.Background(), &submission.Submission{ID: uuid.New(), ApplicantID: uuid.New(), CompetitionID: &period}, &submission.Form{Personal: &submission.PersonalSection{Nationality: "Extranjera"}})
	if err != nil || e.Eligible || e.Source != "nacionalidad_declarada" || e.Reason != "nacionalidad_no_admitida" {
		t.Fatal(e, err)
	}
}
