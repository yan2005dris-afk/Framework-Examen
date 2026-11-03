package competition

import (
	"github.com/google/uuid"
	"time"
)

type Phase struct {
	Code      string    `json:"codigo"`
	Title     string    `json:"titulo"`
	OpensAt   time.Time `json:"apertura"`
	ClosesAt  time.Time `json:"cierre"`
	Published bool      `json:"publicada"`
	Enabled   bool      `json:"habilitada"`
}
type Configuration struct {
	Version     int64    `json:"version"`
	Approved    bool     `json:"inicioAprobado"`
	Published   bool     `json:"publicada"`
	FormEnabled bool     `json:"ef12Habilitado"`
	Sectors     []string `json:"sectores"`
	Phases      []Phase  `json:"fases"`
}
type Competition struct {
	ID       uuid.UUID `json:"id"`
	Code     string    `json:"codigo"`
	Title    string    `json:"titulo"`
	OpensAt  time.Time `json:"apertura"`
	ClosesAt time.Time `json:"cierre"`
	Configuration
	ApprovedAt *time.Time `json:"aprobadoEn,omitempty"`
	ApprovedBy *uuid.UUID `json:"aprobadoPor,omitempty"`
}
