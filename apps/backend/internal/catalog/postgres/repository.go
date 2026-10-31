package postgres

import (
	"context"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
)

/** Repository consulta el catálogo persistente mediante SQLC. */
type Repository struct{ queries *sqlcgen.Queries }

/** NewRepository recibe las consultas tipadas de los catálogos persistentes. */
func NewRepository(queries *sqlcgen.Queries) *Repository { return &Repository{queries: queries} }

/** List conserva el orden de captura de la semilla y filtra por su padre canónico. */
func (r *Repository) List(ctx context.Context, kind, parent string) ([]string, error) {
	return r.queries.ListCatalogValues(ctx, sqlcgen.ListCatalogValuesParams{Kind: kind, ParentKey: parent})
}
