// Package review consulta expedientes sellados y genera la matriz técnica de EF-13.
package review

import (
	"context"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/shared/domain"
)

/** Repository aplica exactamente los mismos filtros a listas, conteos y exportaciones. */
type Repository interface {
	Competition(context.Context, uuid.UUID) (*Competition, error)
	Competitions(context.Context, domain.Position, int) ([]Competition, error)
	List(context.Context, Filter, domain.Position, int) ([]Snapshot, error)
	Export(context.Context, Filter) ([]Snapshot, error)
	Counts(context.Context, Filter) ([]Count, error)
}
