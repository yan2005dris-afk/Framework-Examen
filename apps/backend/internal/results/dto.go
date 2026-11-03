// Package results conserva listas históricas a partir de fuentes definitivas y consentimientos sellados.
package results

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/evaluation"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/merit"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/shared/domain"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"time"
)

var (
	ErrForbidden = errors.New("no tiene permiso para esta lista")
	ErrNotFound  = errors.New("la lista no existe o no es pública")
	ErrInvalid   = errors.New("revise las reglas y parámetros de la lista")
	ErrPending   = errors.New("faltan resultados definitivos o reconsideraciones resueltas")
	ErrConflict  = errors.New("las fuentes o la versión publicada cambiaron")
	ErrPhase     = errors.New("se requiere fase de resultados y cierre de reconsideraciones")
	ErrPrivacy   = errors.New("faltan aprobación de privacidad o consentimiento de publicación de nombres")
	ErrTooLarge  = errors.New("la selección excede el presupuesto de lista")
)

const MaxRows = 10000
const MaxBytes = 32 << 20

type Rules struct {
	Order     []string `json:"orden"`
	Reference string   `json:"referenciaOrden"`
}
type GenerateInput struct {
	Rules Rules `json:"reglas"`
}
type PublishInput struct {
	PreviousVersion  int64      `json:"versionPublicadaAnterior"`
	SupersedesID     *uuid.UUID `json:"sustituyeId,omitempty"`
	Reference        string     `json:"referenciaAprobacion"`
	Public           bool       `json:"publica"`
	PrivacyReference string     `json:"referenciaPrivacidad"`
}
type Entry struct {
	SubmissionID uuid.UUID                    `json:"postulacionId"`
	Code         string                       `json:"codigo"`
	Hash         string                       `json:"hashEnvio"`
	FirstName    string                       `json:"nombres"`
	LastName     string                       `json:"apellidos"`
	Eligible     bool                         `json:"elegible"`
	Rank         int                          `json:"puesto,omitempty"`
	Tied         bool                         `json:"empate"`
	NameConsent  bool                         `json:"consentimientoNombre"`
	Results      map[string]evaluation.Result `json:"resultados"`
	Appeals      []AppealRef                  `json:"reconsideraciones"`
}
type AppealRef struct {
	ID          uuid.UUID `json:"id"`
	Version     int64     `json:"version"`
	EffectiveID uuid.UUID `json:"resultadoEfectivoId"`
}
type List struct {
	ID            uuid.UUID `json:"id"`
	CompetitionID uuid.UUID `json:"convocatoriaId"`
	Sector        string    `json:"sector"`
	Version       int64     `json:"version"`
	Rules         Rules     `json:"reglas"`
	SourcesHash   string    `json:"hashFuentes"`
	ActorID       uuid.UUID `json:"responsableId"`
	CreatedAt     time.Time `json:"creadaEn"`
	Entries       []Entry   `json:"-"`
}
type Publication struct {
	ID               uuid.UUID  `json:"id"`
	ListID           uuid.UUID  `json:"listaId"`
	CompetitionID    uuid.UUID  `json:"convocatoriaId"`
	Sector           string     `json:"sector"`
	Version          int64      `json:"version"`
	PreviousID       *uuid.UUID `json:"sustituyeId,omitempty"`
	Reference        string     `json:"referenciaAprobacion"`
	Public           bool       `json:"publica"`
	PrivacyReference string     `json:"referenciaPrivacidad,omitempty"`
	ActorID          uuid.UUID  `json:"responsableId"`
	CreatedAt        time.Time  `json:"publicadaEn"`
	Names            []string   `json:"-"`
}
type Source struct {
	Snapshot *submission.Submission
	Merit    *merit.Publication
	State    evaluation.State
	Theory   *evaluation.Result
	Appeals  []evaluation.Appeal
}
type Repository interface {
	Generate(context.Context, *identity.Applicant, uuid.UUID, string, uuid.UUID, GenerateInput) (*List, error)
	Publish(context.Context, *identity.Applicant, uuid.UUID, uuid.UUID, PublishInput) (*Publication, error)
	Get(context.Context, uuid.UUID) (*List, error)
	Publication(context.Context, uuid.UUID) (*Publication, error)
	Current(context.Context, uuid.UUID, string) (*Publication, error)
	Lists(context.Context, uuid.UUID, string, domain.Position, int) ([]List, error)
	Publications(context.Context, uuid.UUID, string, domain.Position, int) ([]Publication, error)
}
type Notifier interface {
	Enqueue(context.Context, uuid.UUID, uuid.UUID, string) error
}
