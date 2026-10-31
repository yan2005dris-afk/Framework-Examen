package merit

import (
	"context"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
)

/** Repository serializa actor, reglas activas y cabeceras del expediente con su auditoría. */
type Repository interface {
	CreateRules(context.Context, *identity.Applicant, *RuleSet) error
	ApproveRules(context.Context, *identity.Applicant, uuid.UUID, string) (*RuleSet, error)
	GetRules(context.Context, uuid.UUID) (*RuleSet, error)
	ListRules(context.Context, uuid.UUID, string, int64, int) ([]RuleSet, error)
	ReadCase(context.Context, uuid.UUID) (*Case, error)
	MutateCase(context.Context, *identity.Applicant, uuid.UUID, func(*Case) error) (*Case, error)
}
