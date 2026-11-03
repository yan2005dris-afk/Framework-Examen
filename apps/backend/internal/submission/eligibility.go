package submission

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"time"
)

var ErrEligibilityUnavailable = errors.New("no se pudo verificar la elegibilidad; inténtelo más tarde")
var ErrEligibilityClosed = errors.New("la postulación está cerrada por elegibilidad y no admite cambios")

/** EligibilityAssessment es evidencia del servidor, separada de los campos declarados y del sello. */
type EligibilityAssessment struct {
	ID            uuid.UUID `json:"id"`
	ApplicantID   uuid.UUID `json:"cuentaId"`
	CompetitionID uuid.UUID `json:"convocatoriaId"`
	Sector        string    `json:"sector"`
	Eligible      bool      `json:"elegible"`
	Source        string    `json:"origen"`
	CheckedAt     time.Time `json:"verificadoEn"`
	Nationality   string    `json:"nacionalidadDeclarada,omitempty"`
	Reason        string    `json:"codigo"`
	Message       string    `json:"mensaje"`
}
type EligibilityAssessor interface {
	Assess(context.Context, *Submission, *Form) (*EligibilityAssessment, error)
}

func (s *Service) WithEligibility(a EligibilityAssessor) *Service { s.eligibility = a; return s }

/** PrepareEligibility sólo se usa bajo la transacción del agregado. Un cierre se confirma, no se revierte como error. */
func PrepareEligibility(ctx context.Context, a EligibilityAssessor, item *Submission, raw json.RawMessage, now time.Time) (json.RawMessage, bool, error) {
	if !item.IsEditable() {
		return nil, false, editableError(item)
	}
	if a == nil {
		return raw, false, nil
	}
	form, err := DecodeForm(raw)
	if err != nil {
		return nil, false, err
	}
	evidence, err := a.Assess(ctx, item, form)
	if err != nil {
		return nil, false, err
	}
	item.Eligibility = evidence
	if evidence != nil && !evidence.Eligible {
		item.Status = StatusClosed
		item.UpdatedAt = now
		item.Eligibility = evidence
		// Se conserva la causa del cierre, pero no se aceptan cambios ajenos en un formulario rechazado.
		return item.Content, true, nil
	}
	if form.Position != nil {
		form.Position.Eligible = nil
		form.Position.VerificationDetail = ""
		if evidence != nil {
			eligible := evidence.Eligible
			form.Position.Eligible = &eligible
			form.Position.VerificationDetail = evidence.Message + " · " + evidence.Source
		}
	}
	encoded, err := json.Marshal(form)
	return encoded, false, err
}

/** ClosureError acompaña el rechazo HTTP después del commit del cierre. */
type ClosureError struct{ Assessment *EligibilityAssessment }

func (e *ClosureError) Error() string {
	if e.Assessment != nil {
		return e.Assessment.Message
	}
	return ErrEligibilityClosed.Error()
}
func (e *ClosureError) Unwrap() error { return ErrEligibilityClosed }
func closureResult(item *Submission, err error) (*Submission, error) {
	if err != nil {
		return nil, err
	}
	if item.Status == StatusClosed {
		return nil, &ClosureError{item.Eligibility}
	}
	return item, nil
}

/** IneligibleMessage expone el literal por sector para el adaptador de verificación. */
func IneligibleMessage(sector string) string { return ineligibleMessage(sector) }

/** contentForRead no presenta una bandera legada del navegador como verificación del servidor. No toca sellos. */
func contentForRead(item *Submission) json.RawMessage {
	if !item.IsEditable() {
		return item.Content
	}
	form, err := DecodeForm(item.Content)
	if err != nil || form.Position == nil {
		return item.Content
	}
	form.Position.Eligible = nil
	form.Position.VerificationDetail = ""
	e := item.Eligibility
	if e != nil && e.Eligible && e.ApplicantID == item.ApplicantID && item.CompetitionID != nil && e.CompetitionID == *item.CompetitionID && e.Sector == form.Position.Sector && e.Source != "" {
		value := true
		form.Position.Eligible = &value
		form.Position.VerificationDetail = e.Message + " · " + e.Source
	}
	raw, err := json.Marshal(form)
	if err != nil {
		return item.Content
	}
	return raw
}

func editableError(item *Submission) error {
	if item.Status == StatusClosed {
		return &ClosureError{item.Eligibility}
	}
	return ErrNotEditable
}
