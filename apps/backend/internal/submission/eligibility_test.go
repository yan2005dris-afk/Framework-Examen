package submission

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"testing"
	"time"
)

type assessorStub struct {
	eligible bool
	fail     bool
}

func TestReadNeverCertifiesClientFlagAndNeverRewritesSealedContent(t *testing.T) {
	period := uuid.New()
	item, _ := NewSubmission(uuid.New(), uuid.New(), "QA-LECTURA", json.RawMessage(`{"seccion4":{"sector":"afiliados","verificacionIessCumplida":true,"detalleVerificacionIess":"Afirmación del cliente"}}`), time.Now())
	item.CompetitionID = &period
	form, err := DecodeForm(NewResponse(item).Content)
	if err != nil || form.Position.Eligible != nil || form.Position.VerificationDetail != "" {
		t.Fatal("la lectura certificó al cliente", err)
	}
	before := string(item.Content)
	item.Status = StatusSubmitted
	if string(NewResponse(item).Content) != before {
		t.Fatal("se reconstruyó un envío histórico")
	}
}

func (a assessorStub) Assess(_ context.Context, item *Submission, _ *Form) (*EligibilityAssessment, error) {
	if a.fail {
		return nil, ErrEligibilityUnavailable
	}
	return &EligibilityAssessment{ID: uuid.New(), ApplicantID: item.ApplicantID, CompetitionID: *item.CompetitionID, Eligible: a.eligible, Source: "simulacion_controlada_servidor", CheckedAt: time.Now(), Message: "Cierre sintético", Reason: "sector_no_elegible"}, nil
}
func TestIneligibleClosureCommitsBeforeReturningRejection(t *testing.T) {
	for _, operation := range []string{"actualizar", "seccion", "enviar"} {
		t.Run(operation, func(t *testing.T) {
			period := uuid.New()
			item, _ := NewSubmission(uuid.New(), uuid.New(), "QA-CIERRE", json.RawMessage(`{}`), time.Now())
			item.CompetitionID = &period
			repo := &memoryRepository{item: item}
			service := NewService(repo).WithEligibility(assessorStub{})
			var err error
			switch operation {
			case "actualizar":
				_, err = service.Update(context.Background(), item.ID, json.RawMessage(`{"seccion4":{"sector":"afiliados"}}`))
			case "seccion":
				_, err = service.SaveSection(context.Background(), item.ID, 4, json.RawMessage(`{"sector":"afiliados"}`))
			case "enviar":
				_, err = service.Submit(context.Background(), item.ID, nil)
			}
			if !errors.Is(err, ErrEligibilityClosed) || repo.item.Status != StatusClosed || repo.item.Eligibility == nil || repo.item.Hash != "" || repo.item.SubmittedAt != nil {
				t.Fatal("cierre no confirmado", repo.item, err)
			}
			if _, err = service.SaveSection(context.Background(), item.ID, 1, json.RawMessage(`{}`)); !errors.Is(err, ErrEligibilityClosed) {
				t.Fatal("se reabrió el cierre", err)
			}
		})
	}
}
func TestEligibilityTechnicalAndCommitFailuresDoNotClose(t *testing.T) {
	for _, technical := range []bool{true, false} {
		period := uuid.New()
		item, _ := NewSubmission(uuid.New(), uuid.New(), "QA-ROLLBACK", json.RawMessage(`{}`), time.Now())
		item.CompetitionID = &period
		repo := &memoryRepository{item: item, fail: !technical}
		service := NewService(repo).WithEligibility(assessorStub{fail: technical})
		if _, err := service.Update(context.Background(), item.ID, json.RawMessage(`{}`)); err == nil || errors.Is(err, ErrEligibilityClosed) {
			t.Fatal("se afirmó un cierre sin commit", err)
		}
		if repo.item.Status != StatusDraft || repo.item.Eligibility != nil {
			t.Fatal("se alteró el borrador")
		}
	}
}
func TestClientEligibilityFlagIsReplacedByServerEvidence(t *testing.T) {
	period := uuid.New()
	item, _ := NewSubmission(uuid.New(), uuid.New(), "QA-SERVIDOR", json.RawMessage(`{}`), time.Now())
	item.CompetitionID = &period
	repo := &memoryRepository{item: item}
	service := NewService(repo).WithEligibility(assessorStub{eligible: true})
	result, err := service.Update(context.Background(), item.ID, json.RawMessage(`{"seccion4":{"sector":"afiliados","verificacionIessCumplida":false,"detalleVerificacionIess":"Inventado"}}`))
	if err != nil {
		t.Fatal(err)
	}
	form, _ := DecodeForm(result.Content)
	if form.Position.Eligible == nil || !*form.Position.Eligible || form.Position.VerificationDetail == "Inventado" {
		t.Fatal("se confió en el cliente")
	}
}
