package competition

import (
	"context"
	"github.com/google/uuid"
)

/** Repository confirma configuración, versión histórica y auditoría como una unidad. */
type Repository interface {
	Create(context.Context, *Competition, uuid.UUID) error
	Get(context.Context, uuid.UUID) (*Competition, error)
	Configure(context.Context, uuid.UUID, Configuration, uuid.UUID) (*Competition, error)
	ListPublished(context.Context, int, uuid.UUID) ([]Competition, error)
}
