// Package merit conserva revisiones, reglas aprobadas y resultados de méritos sin modificar el formulario sellado.
package merit

import (
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"time"
)

type Criterion struct {
	Code          string `json:"codigo"`
	Label         string `json:"nombre"`
	Metric        string `json:"metrica"`
	Filter        string `json:"filtro,omitempty"`
	WholeUnits    bool   `json:"unidadesCompletas,omitempty"`
	UnitSize      int64  `json:"tamanoUnidad"`
	PointsPerUnit int64  `json:"centipuntosPorUnidad"`
	MaximumPoints int64  `json:"maximoCentipuntos"`
	Weight        int64  `json:"pesoPuntosBase"`
	Required      bool   `json:"requiereEvidencia"`
}
type Requirement struct {
	Code  string `json:"codigo"`
	Label string `json:"nombre"`
}
type QuestionRule struct {
	ID             string `json:"pregunta"`
	AcceptedAnswer string `json:"respuestaAdmitida"`
}
type RuleDefinition struct {
	Criteria              []Criterion    `json:"criterios"`
	Requirements          []Requirement  `json:"requisitos"`
	Questions             []QuestionRule `json:"inhabilidades"`
	MaximumTotal          *int64         `json:"maximoTotalCentipuntos"`
	Threshold             *int64         `json:"umbralCentipuntos"`
	Rounding              string         `json:"redondeo"`
	ExperienceAggregation string         `json:"experiencias"`
	DayCounting           string         `json:"computoDias"`
	TieBreakers           []string       `json:"desempates"`
}
type RuleSet struct {
	ApprovedBy        *uuid.UUID     `json:"aprobadaPor,omitempty"`
	ApprovedAt        *time.Time     `json:"aprobadaEn,omitempty"`
	ID                uuid.UUID      `json:"id"`
	CompetitionID     uuid.UUID      `json:"convocatoriaId"`
	Sector            string         `json:"sector"`
	Version           int64          `json:"version"`
	Definition        RuleDefinition `json:"definicion"`
	Hash              string         `json:"hashDefinicion"`
	Approved          bool           `json:"aprobada"`
	ApprovalReference string         `json:"referenciaAprobacion,omitempty"`
	ActorID           uuid.UUID      `json:"responsableId"`
	CreatedAt         time.Time      `json:"creadaEn"`
}
type RecordReview struct {
	Section     int    `json:"seccion"`
	RecordID    string `json:"registroId"`
	DocumentID  string `json:"documentoId"`
	Verdict     string `json:"resultado"`
	Observation string `json:"observacion"`
}
type RequirementReview struct {
	Code        string `json:"codigo"`
	Satisfied   bool   `json:"cumple"`
	Observation string `json:"observacion"`
}
type QuestionReview struct {
	ID          string `json:"pregunta"`
	Confirmed   bool   `json:"verificada"`
	Observation string `json:"observacion"`
}
type ReviewInput struct {
	Version      int64               `json:"versionAnterior"`
	RuleSetID    uuid.UUID           `json:"reglasId"`
	Documents    []RecordReview      `json:"documentos"`
	Requirements []RequirementReview `json:"requisitos"`
	Questions    []QuestionReview    `json:"inhabilidades"`
	Motive       string              `json:"motivo"`
}
type Review struct {
	ID           uuid.UUID   `json:"id"`
	SubmissionID uuid.UUID   `json:"postulacionId"`
	SnapshotHash string      `json:"hashEnvio"`
	Version      int64       `json:"version"`
	Input        ReviewInput `json:"revision"`
	ActorID      uuid.UUID   `json:"responsableId"`
	CreatedAt    time.Time   `json:"creadaEn"`
}
type Breakdown struct {
	Code   string `json:"criterio"`
	Units  int64  `json:"unidadesVerificadas"`
	Points int64  `json:"centipuntos"`
}
type Evaluation struct {
	Total      int64       `json:"totalCentipuntos"`
	Maximum    int64       `json:"maximoCentipuntos"`
	Threshold  int64       `json:"umbralCentipuntos"`
	CanAdvance bool        `json:"puedeAvanzar"`
	Breakdown  []Breakdown `json:"desglose"`
	Blockers   []string    `json:"impedimentos"`
	TieValues  []Breakdown `json:"desempate"`
	ManualTie  bool        `json:"desempateManual"`
}

const EngineVersion = "meritos-v1"

type Calculation struct {
	EngineVersion string     `json:"versionMotor"`
	ID            uuid.UUID  `json:"id"`
	SubmissionID  uuid.UUID  `json:"postulacionId"`
	Version       int64      `json:"version"`
	ReviewID      uuid.UUID  `json:"revisionId"`
	RuleSetID     uuid.UUID  `json:"reglasId"`
	RulesHash     string     `json:"hashReglas"`
	SnapshotHash  string     `json:"hashEnvio"`
	Evaluation    Evaluation `json:"evaluacion"`
	Motive        string     `json:"motivo"`
	ActorID       uuid.UUID  `json:"responsableId"`
	CreatedAt     time.Time  `json:"creadoEn"`
}
type Publication struct {
	ID            uuid.UUID  `json:"id"`
	SubmissionID  uuid.UUID  `json:"postulacionId"`
	Version       int64      `json:"version"`
	CalculationID uuid.UUID  `json:"calculoId"`
	SupersedesID  *uuid.UUID `json:"sustituyeId,omitempty"`
	Decision      string     `json:"decision"`
	Motive        string     `json:"motivo"`
	Evaluation    Evaluation `json:"evaluacion"`
	SnapshotHash  string     `json:"hashEnvio"`
	RuleSetID     uuid.UUID  `json:"reglasId"`
	RulesHash     string     `json:"hashReglas"`
	ActorID       uuid.UUID  `json:"responsableId"`
	CreatedAt     time.Time  `json:"publicadaEn"`
}
type Document struct {
	ID       string
	Section  int
	RecordID string
	SHA256   string
}
type Case struct {
	Snapshot    *submission.Submission `json:"-"`
	Documents   []Document             `json:"-"`
	ActiveRules *RuleSet               `json:"reglasActivas,omitempty"`
	Review      *Review                `json:"revision,omitempty"`
	Calculation *Calculation           `json:"calculo,omitempty"`
	Publication *Publication           `json:"publicacion,omitempty"`
}
type CalculateInput struct {
	ReviewID uuid.UUID `json:"revisionId"`
	Version  int64     `json:"versionAnterior"`
	Motive   string    `json:"motivo"`
}
type PublishInput struct {
	CalculationID uuid.UUID  `json:"calculoId"`
	Version       int64      `json:"versionAnterior"`
	SupersedesID  *uuid.UUID `json:"sustituyeId,omitempty"`
	Decision      string     `json:"decision"`
	Motive        string     `json:"motivo"`
}
