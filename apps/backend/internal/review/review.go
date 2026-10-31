package review

import (
	"encoding/json"
	"github.com/google/uuid"
	"time"
)

/** Competition identifica un período estable; no reemplaza fases/aprobaciones de T18. */
type Competition struct {
	ID        uuid.UUID `json:"id"`
	Code      string    `json:"codigo"`
	Title     string    `json:"titulo"`
	OpensAt   time.Time `json:"apertura"`
	ClosesAt  time.Time `json:"cierre"`
	CreatedAt time.Time `json:"createdAt"`
	Closed    bool      `json:"cerrada"`
}

/** Filter exige un período explícito o el conjunto legado sin convocatoria, nunca ambos. */
type Filter struct {
	CompetitionID *uuid.UUID `json:"convocatoriaId,omitempty"`
	Legacy        bool       `json:"sinConvocatoria"`
	Sector        string     `json:"sector,omitempty"`
	Status        string     `json:"status,omitempty"`
}

/** Snapshot guarda únicamente el JSON e identificadores sellados; no consulta el perfil actual. */
type Snapshot struct {
	ID              uuid.UUID
	Code            string
	CompetitionCode string
	Status          string
	Content         json.RawMessage
	Hash            string
	SubmittedAt     time.Time
	CreatedAt       time.Time
}

/** Count representa postulantes distintos por cargo/sector y estado enviado. */
type Count struct {
	Sector      string `json:"sector"`
	Status      string `json:"status"`
	Applicants  int64  `json:"postulantes"`
	Submissions int64  `json:"postulaciones"`
}
