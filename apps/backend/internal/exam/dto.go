// Package exam mantiene el banco, la asignación y la calificación en el servidor.
package exam

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/evaluation"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"time"
)

var (
	ErrInvalid   = errors.New("revise los datos de evaluación")
	ErrForbidden = errors.New("no tiene permiso para esta evaluación")
	ErrNotFound  = errors.New("no existe la evaluación")
	ErrConflict  = errors.New("la evaluación cambió; consulte su versión vigente")
	ErrPhase     = errors.New("la evaluación no está habilitada en este horario")
	ErrExpired   = errors.New("el intento ya no admite respuestas")
	ErrExhausted = errors.New("se agotaron las asignaciones únicas del banco; requiere revisión administrativa")
)

type Option struct {
	ID   string `json:"id"`
	Text string `json:"texto"`
}
type Question struct {
	ID      string   `json:"id"`
	Text    string   `json:"enunciado"`
	Options []Option `json:"opciones"`
	Correct string   `json:"correcta"`
}
type BankInput struct {
	PreviousVersion int64      `json:"versionAnterior"`
	Questions       []Question `json:"preguntas"`
	Reference       string     `json:"referencia"`
}
type Bank struct {
	ID            uuid.UUID `json:"id"`
	CompetitionID uuid.UUID `json:"convocatoriaId"`
	Sector        string    `json:"sector"`
	Version       int64     `json:"version"`
	Input         BankInput `json:"definicion"`
	Hash          string    `json:"hash"`
	ActorID       uuid.UUID `json:"responsableId"`
	At            time.Time `json:"creadaEn"`
}
type ConfigRules struct {
	Count           int       `json:"cantidadPreguntas"`
	DurationSeconds int       `json:"duracionSegundos"`
	Maximum         int64     `json:"maximo"`
	Threshold       int64     `json:"umbral"`
	Rounding        string    `json:"redondeo"`
	AttemptPolicy   string    `json:"politicaIntentos"`
	UniquePolicy    string    `json:"politicaUnicidad"`
	Reference       string    `json:"referenciaAprobacion"`
	OpensAt         time.Time `json:"abreEn"`
	ClosesAt        time.Time `json:"cierraEn"`
}
type Config struct {
	ID            uuid.UUID   `json:"id"`
	BankID        uuid.UUID   `json:"bancoId"`
	CompetitionID uuid.UUID   `json:"convocatoriaId"`
	Sector        string      `json:"sector"`
	Rules         ConfigRules `json:"reglas"`
	Hash          string      `json:"hash"`
	At            time.Time   `json:"creadaEn"`
	ActorID       uuid.UUID   `json:"responsableId"`
}
type EnrollmentInput struct {
	ConfigID uuid.UUID `json:"configuracionId"`
	MeritID  uuid.UUID `json:"resultadoMeritosId"`
}
type Attempt struct {
	ID           uuid.UUID          `json:"id"`
	SubmissionID uuid.UUID          `json:"postulacionId"`
	Config       Config             `json:"configuracion"`
	SnapshotHash string             `json:"hashEnvio"`
	BankHash     string             `json:"hashBanco"`
	MeritID      uuid.UUID          `json:"meritosId"`
	Questions    []Question         `json:"matriz"`
	Answers      map[string]string  `json:"respuestas"`
	Version      int64              `json:"version"`
	Status       string             `json:"estado"`
	StartedAt    time.Time          `json:"iniciadaEn"`
	ExpiresAt    time.Time          `json:"venceEn"`
	FinishedAt   *time.Time         `json:"finalizadaEn,omitempty"`
	Result       *evaluation.Result `json:"resultado,omitempty"`
	CorrectCount int                `json:"aciertos"`
	Reason       string             `json:"motivoCierre,omitempty"`
}
type AnswerInput struct {
	PreviousVersion int64  `json:"versionAnterior"`
	QuestionID      string `json:"preguntaId"`
	OptionID        string `json:"opcionId"`
}
type View struct {
	ID               uuid.UUID          `json:"id"`
	SubmissionID     uuid.UUID          `json:"postulacionId"`
	Version          int64              `json:"version"`
	Status           string             `json:"estado"`
	StartedAt        time.Time          `json:"iniciadaEn"`
	ExpiresAt        time.Time          `json:"venceEn"`
	ServerAt         time.Time          `json:"servidorEn"`
	RemainingSeconds int64              `json:"segundosRestantes"`
	Count            int                `json:"cantidadPreguntas"`
	Answered         int                `json:"respondidas"`
	Result           *evaluation.Result `json:"resultado,omitempty"`
}
type PublicQuestion struct {
	ID       string   `json:"id"`
	Text     string   `json:"enunciado"`
	Options  []Option `json:"opciones"`
	Selected string   `json:"seleccionada,omitempty"`
	Index    int      `json:"indice"`
	Version  int64    `json:"versionIntento"`
}
type Report struct {
	PDF  []byte
	Hash string
}
type Repository interface {
	CreateBank(context.Context, *identity.Applicant, uuid.UUID, string, BankInput) (*Bank, error)
	Bank(context.Context, uuid.UUID) (*Bank, error)
	Banks(context.Context, uuid.UUID, string, int64, int) ([]Bank, error)
	Configure(context.Context, *identity.Applicant, uuid.UUID, ConfigRules) (*Config, error)
	Enroll(context.Context, *identity.Applicant, uuid.UUID, EnrollmentInput) error
	Start(context.Context, *identity.Applicant, uuid.UUID) (*Attempt, time.Time, error)
	Read(context.Context, *identity.Applicant, uuid.UUID, bool) (*Attempt, time.Time, error)
	Answer(context.Context, *identity.Applicant, uuid.UUID, AnswerInput) (*Attempt, time.Time, error)
	Finish(context.Context, *identity.Applicant, uuid.UUID, bool) (*Attempt, time.Time, error)
	Report(context.Context, *identity.Applicant, uuid.UUID, bool) (Report, error)
	Expire(context.Context) error
}
type Finalizer interface {
	RecordFinalTheory(context.Context, evaluation.Result) error
}
type Renderer interface {
	Render(context.Context, *Attempt) ([]byte, error)
}
type Gate interface {
	Allow(context.Context, uuid.UUID, string) error
}
