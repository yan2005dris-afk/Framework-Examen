package evaluation

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/shared/domain"
	"time"
)

type Repository interface {
	CreatePolicy(context.Context, *identity.Applicant, *Policy) error
	ApprovePolicy(context.Context, *identity.Applicant, uuid.UUID, string) (*Policy, error)
	GetPolicy(context.Context, uuid.UUID) (*Policy, error)
	ListPolicies(context.Context, uuid.UUID, string, string, int64, int) ([]Policy, error)
	ReadCase(context.Context, uuid.UUID, uuid.UUID) (*Case, error)
	Execute(context.Context, *identity.Applicant, uuid.UUID, Command, func(*Case) (Change, error)) (Receipt, error)
	ListAppeals(context.Context, uuid.UUID, domain.Position, int) ([]Appeal, error)
	ListHistory(context.Context, uuid.UUID, domain.Position, int) ([]HistoryEntry, error)
	GetEvidence(context.Context, uuid.UUID) (*Evidence, error)
	BeginUpload(context.Context, *Evidence) error
	ExpiredUploads(context.Context) ([]Evidence, error)
	CompleteUpload(context.Context, uuid.UUID) error
}
type HistoryEntry struct {
	ID     uuid.UUID       `json:"id"`
	Action string          `json:"accion"`
	At     time.Time       `json:"fecha"`
	Data   json.RawMessage `json:"detalle,omitempty"`
}

/** Notifier debe encolar dentro de la misma transacción, nunca enviar SMTP durante un commit. */
type Notifier interface {
	Enqueue(context.Context, uuid.UUID, uuid.UUID, string) error
}
