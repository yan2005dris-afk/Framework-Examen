// Package privacy aplica políticas institucionales explícitas sin inventar textos legales ni purgar evidencia.
package privacy

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"strings"
	"time"
)

var ErrDenied = submission.ErrPrivacyPending
var ErrInvalid = errors.New("revise la política de privacidad")
var ErrForbidden = errors.New("no tiene permiso de privacidad")
var ErrConflict = errors.New("la versión de privacidad cambió")
var ErrNotFound = errors.New("la política no existe")

type Rules struct {
	Text         string `json:"texto"`
	Treatment    string `json:"informacionTratamiento"`
	Reference    string `json:"referenciaAprobacion"`
	AccessDays   int    `json:"diasAcceso"`
	DocumentDays int    `json:"diasDocumentos"`
	ReportDays   int    `json:"diasReportes"`
	PublicDays   int    `json:"diasPublicacionNombres"`
	EmailDays    int    `json:"diasCorreo"`
	ReviewDays   int    `json:"diasRevisionConservacion"`
	Historical   string `json:"disposicionHistoricos"`
	AllowExports bool   `json:"permiteExportacion"`
	AllowEmail   bool   `json:"permiteCorreo"`
	AllowNames   bool   `json:"permiteNombresPublicos"`
}

func (r Rules) Validate() error {
	for _, v := range []string{r.Text, r.Treatment, r.Reference} {
		if strings.TrimSpace(v) == "" || len(v) > 64000 {
			return ErrInvalid
		}
	}
	for _, d := range []int{r.AccessDays, r.DocumentDays, r.ReportDays, r.PublicDays, r.EmailDays, r.ReviewDays} {
		if d < 1 || d > 36500 {
			return ErrInvalid
		}
	}
	if r.Historical != "revision_humana_sin_borrado_automatico" {
		return ErrInvalid
	}
	return nil
}
func (r Rules) Allow(accepted, now time.Time, channel string) error {
	days := r.AccessDays
	switch channel {
	case "documentos":
		days = min(days, r.DocumentDays)
	case "reportes":
		days = min(days, r.ReportDays)
	case "exportacion":
		if !r.AllowExports {
			return ErrDenied
		}
	case "correo":
		if !r.AllowEmail {
			return ErrDenied
		}
		days = min(days, r.EmailDays)
	case "nombres":
		if !r.AllowNames {
			return ErrDenied
		}
		days = min(days, r.PublicDays)
	case "datos":
	default:
		return ErrInvalid
	}
	if r.Validate() != nil || accepted.IsZero() || now.Before(accepted) || !now.Before(accepted.AddDate(0, 0, days)) {
		return ErrDenied
	}
	return nil
}

type Policy struct {
	ControlVersion int64     `json:"versionControl"`
	Blocked        bool      `json:"restringido"`
	ID             uuid.UUID `json:"id"`
	CompetitionID  uuid.UUID `json:"convocatoriaId"`
	Version        int64     `json:"version"`
	Rules          Rules     `json:"definicion"`
	Hash           string    `json:"hash"`
	ActorID        uuid.UUID `json:"aprobadaPor"`
	At             time.Time `json:"publicadaEn"`
}
type PublishInput struct {
	PreviousVersion int64 `json:"versionAnterior"`
	Rules           Rules `json:"definicion"`
}
type BlockInput struct {
	PreviousVersion int64  `json:"versionAnterior"`
	Blocked         bool   `json:"restringido"`
	Reason          string `json:"motivo"`
}
type Review struct {
	SubmissionID uuid.UUID `json:"postulacionId"`
	PolicyID     uuid.UUID `json:"politicaId"`
	Due          time.Time `json:"revisarDesde"`
}
type Repository interface {
	Publish(context.Context, *identity.Applicant, uuid.UUID, PublishInput) (*Policy, error)
	Current(context.Context, uuid.UUID) (*Policy, error)
	Get(context.Context, uuid.UUID) (*Policy, error)
	List(context.Context, uuid.UUID, int64, int) ([]Policy, error)
	Block(context.Context, *identity.Applicant, uuid.UUID, BlockInput) error
	Bind(context.Context, *submission.Submission, *submission.Consent) error
	Allow(context.Context, uuid.UUID, string) error
	Period(context.Context, uuid.UUID, string) error
	Reviews(context.Context, uuid.UUID, time.Time, uuid.UUID, int) ([]Review, error)
}
