// Package evaluation conserva oposición, consolidación y reconsideraciones sin alterar el EF-12 sellado.
package evaluation

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"time"
)

const EngineVersion = "consolidacion-v1"
const MaxEvidenceBytes = 2 << 20

type OppositionRules struct {
	Instrument          string   `json:"instrumento"`
	InstrumentVersion   string   `json:"versionInstrumento"`
	PsychometricMode    string   `json:"modoPsicotecnico"`
	PsychometricMinimum *int64   `json:"umbralPsicotecnico,omitempty"`
	PsychometricMaximum *int64   `json:"maximoPsicotecnico"`
	Conclusions         []string `json:"conclusiones"`
	AcceptedConclusions []string `json:"conclusionesAdmitidas"`
	TheoryMaximum       int64    `json:"maximoTeorico"`
	MeritWeight         int64    `json:"pesoMeritos"`
	TheoryWeight        int64    `json:"pesoTeorica"`
	PsychometricWeight  int64    `json:"pesoPsicotecnico"`
	TotalMaximum        int64    `json:"maximoTotal"`
	Threshold           int64    `json:"umbral"`
	Rounding            string   `json:"redondeo"`
}
type Term struct {
	Quantity         int    `json:"cantidad"`
	Unit             string `json:"unidad"`
	IncludeAnchorDay bool   `json:"incluyeDiaInicial"`
	Cutoff           string `json:"horaLimite"`
}
type Calendar struct {
	EndPolicy string   `json:"finVentana"`
	TimeZone  string   `json:"zonaHoraria"`
	Weekdays  []int    `json:"diasLaborables"`
	Holidays  []string `json:"feriados"`
}
type AppealRules struct {
	Applicant  Term     `json:"postulante"`
	Talent     Term     `json:"talentoHumano"`
	Commission Term     `json:"comision"`
	Calendar   Calendar `json:"calendario"`
}
type Definition struct {
	Opposition      *OppositionRules `json:"oposicion,omitempty"`
	Reconsideration *AppealRules     `json:"reconsideracion,omitempty"`
}
type Policy struct {
	ID            uuid.UUID  `json:"id"`
	CompetitionID uuid.UUID  `json:"convocatoriaId"`
	Sector        string     `json:"sector"`
	Kind          string     `json:"tipo"`
	Version       int64      `json:"version"`
	Definition    Definition `json:"definicion"`
	Hash          string     `json:"hash"`
	Approved      bool       `json:"aprobada"`
	Reference     string     `json:"referenciaAprobacion,omitempty"`
	ActorID       uuid.UUID  `json:"responsableId"`
	CreatedAt     time.Time  `json:"creadaEn"`
	ApprovedBy    *uuid.UUID `json:"aprobadaPor,omitempty"`
	ApprovedAt    *time.Time `json:"aprobadaEn,omitempty"`
}
type EvidenceInput struct {
	Name string `json:"nombre"`
	PDF  []byte `json:"pdfBase64"`
}
type Evidence struct {
	Published       bool      `json:"-"`
	ID              uuid.UUID `json:"id"`
	SubmissionID    uuid.UUID `json:"postulacionId"`
	ApplicantID     uuid.UUID `json:"-"`
	ActorID         uuid.UUID `json:"responsableId"`
	ApplicantUpload bool      `json:"-"`
	Name            string    `json:"nombre"`
	Size            int64     `json:"tamano"`
	SHA256          string    `json:"sha256"`
	StorageKey      string    `json:"-"`
	CreatedAt       time.Time `json:"creadaEn"`
}
type PsychometricInput struct {
	PreviousVersion int64         `json:"versionAnterior"`
	PolicyID        uuid.UUID     `json:"reglasId"`
	EvaluatedAt     time.Time     `json:"evaluadaEn"`
	Score           *int64        `json:"puntaje"`
	Conclusion      string        `json:"conclusion"`
	Motive          string        `json:"motivo"`
	Evidence        EvidenceInput `json:"respaldo"`
}
type Psychometric struct {
	ID                uuid.UUID `json:"id"`
	SubmissionID      uuid.UUID `json:"postulacionId"`
	Version           int64     `json:"version"`
	PolicyID          uuid.UUID `json:"reglasId"`
	PolicyHash        string    `json:"hashReglas"`
	SnapshotHash      string    `json:"hashEnvio"`
	Instrument        string    `json:"instrumento"`
	InstrumentVersion string    `json:"versionInstrumento"`
	Score             *int64    `json:"puntaje"`
	Maximum           *int64    `json:"maximo"`
	Conclusion        string    `json:"conclusion"`
	EvaluatedAt       time.Time `json:"evaluadaEn"`
	Evidence          Evidence  `json:"respaldo"`
	Motive            string    `json:"motivo"`
	ActorID           uuid.UUID `json:"responsableId"`
	CreatedAt         time.Time `json:"registradaEn"`
}

/** Result identifica una versión publicada; la nota teórica sólo procede del productor interno T27. */
type Result struct {
	ID            uuid.UUID   `json:"id"`
	SubmissionID  uuid.UUID   `json:"postulacionId"`
	Phase         string      `json:"fase"`
	Version       int64       `json:"version"`
	SupersedesID  *uuid.UUID  `json:"sustituyeId,omitempty"`
	SourceID      uuid.UUID   `json:"fuenteId"`
	PolicyID      uuid.UUID   `json:"reglasId"`
	PolicyHash    string      `json:"hashReglas"`
	SnapshotHash  string      `json:"hashEnvio"`
	Score         *int64      `json:"puntaje"`
	Maximum       *int64      `json:"maximo"`
	CanAdvance    bool        `json:"habilitaAvance"`
	Motive        string      `json:"motivo"`
	ActorID       uuid.UUID   `json:"responsableId"`
	PublishedAt   time.Time   `json:"publicadaEn"`
	EngineVersion string      `json:"versionMotor,omitempty"`
	Breakdown     []Component `json:"desglose,omitempty"`
}
type Component struct {
	Phase        string    `json:"fase"`
	ResultID     uuid.UUID `json:"resultadoId"`
	Score        int64     `json:"puntajeOrigen"`
	Maximum      int64     `json:"maximoOrigen"`
	Weight       int64     `json:"pesoPuntosBase"`
	Contribution int64     `json:"aporte"`
}
type PublishInput struct {
	ExpectedSources map[string]uuid.UUID `json:"fuentesEsperadas,omitempty"`
	PreviousVersion int64                `json:"versionAnterior"`
	SourceID        uuid.UUID            `json:"fuenteId"`
	PolicyID        uuid.UUID            `json:"reglasId"`
	SupersedesID    *uuid.UUID           `json:"sustituyeId,omitempty"`
	Motive          string               `json:"motivo"`
}
type FileAppealInput struct {
	Phase       string         `json:"fase"`
	ResultID    uuid.UUID      `json:"resultadoId"`
	Motive      string         `json:"motivo"`
	Evidence    *EvidenceInput `json:"respaldo,omitempty"`
	DocumentIDs []uuid.UUID    `json:"documentosOriginales,omitempty"`
}
type AttendInput struct {
	PreviousVersion int64  `json:"versionAnterior"`
	Observation     string `json:"observacion"`
	Recommendation  string `json:"recomendacion"`
}
type ResolveInput struct {
	PreviousVersion int64      `json:"versionAnterior"`
	Decision        string     `json:"decision"`
	Motive          string     `json:"motivo"`
	ReplacementID   *uuid.UUID `json:"resultadoRectificadoId,omitempty"`
}
type Attention struct {
	ActorID        uuid.UUID `json:"responsableId"`
	At             time.Time `json:"fecha"`
	Observation    string    `json:"observacion"`
	Recommendation string    `json:"recomendacion"`
}
type Resolution struct {
	ActorID         uuid.UUID `json:"responsableId"`
	At              time.Time `json:"fecha"`
	Decision        string    `json:"decision"`
	Motive          string    `json:"motivo"`
	EffectiveResult Result    `json:"resultadoVigente"`
	ScoreDelta      *int64    `json:"diferenciaPuntaje,omitempty"`
}
type Appeal struct {
	ID                 uuid.UUID   `json:"id"`
	SubmissionID       uuid.UUID   `json:"postulacionId"`
	Version            int64       `json:"version"`
	Status             string      `json:"estado"`
	Original           Result      `json:"resultadoOriginal"`
	Policy             Policy      `json:"reglasPlazo"`
	Motive             string      `json:"motivo"`
	Evidence           *Evidence   `json:"respaldo,omitempty"`
	DocumentIDs        []uuid.UUID `json:"documentosOriginales"`
	ApplicantID        uuid.UUID   `json:"responsableId"`
	FiledAt            time.Time   `json:"presentadaEn"`
	WindowEnd          time.Time   `json:"cierreVentana"`
	FilingDeadline     time.Time   `json:"limitePresentacion"`
	TalentDeadline     time.Time   `json:"limiteTalentoHumano"`
	CommissionDeadline *time.Time  `json:"limiteComision,omitempty"`
	Attention          *Attention  `json:"atencion,omitempty"`
	Resolution         *Resolution `json:"resolucion,omitempty"`
}
type State struct {
	Psychometric *Psychometric     `json:"psicotecnicaRegistrada,omitempty"`
	Results      map[string]Result `json:"resultados"`
}
type Case struct {
	Snapshot         *submission.Submission `json:"-"`
	Sector           string                 `json:"sector"`
	State            State                  `json:"expediente"`
	OppositionPolicy *Policy                `json:"reglasOposicion,omitempty"`
	AppealPolicy     *Policy                `json:"reglasReconsideracion,omitempty"`
	Sources          map[string]Result      `json:"fuentesPublicadas"`
	Appeal           *Appeal                `json:"reconsideracion,omitempty"`
	Now              time.Time              `json:"consultadoEn"`
	PhaseStarts      map[string]time.Time   `json:"-"`
	PhaseEnds        map[string]time.Time   `json:"-"`
	Documents        map[uuid.UUID]string   `json:"-"`
}
type Command struct {
	Key        uuid.UUID
	Operation  string
	ResourceID uuid.UUID
	Hash       string
	AppealID   uuid.UUID
}
type Change struct {
	ID       uuid.UUID
	Action   string
	Record   json.RawMessage
	Evidence *Evidence
	Appeal   *Appeal
	Notify   bool
	Phase    string
}
type Receipt struct {
	Data    json.RawMessage
	Applied bool
}
