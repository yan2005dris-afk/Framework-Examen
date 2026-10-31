package postgres

import (
	"context"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
)

/** Repository persiste eventos de acceso sin documentos ni direcciones firmadas. */
type Repository struct{ queries *sqlcgen.Queries }

/** NewRepository conecta la auditoría con sus consultas tipadas. */
func NewRepository(queries *sqlcgen.Queries) *Repository { return &Repository{queries: queries} }

/** Record confirma el evento antes de permitir entregar los datos solicitados. */
func (r *Repository) Record(ctx context.Context, event audit.Event) error {
	return r.queries.CreateAdministrativeAccessEvent(ctx, sqlcgen.CreateAdministrativeAccessEventParams{
		ActorID: event.ActorID, ActorRole: event.Role, Action: event.Action,
		ResourceID: event.ResourceID, ResourceType: event.ResourceType, Result: event.Result,
		CreatedAt: event.CreatedAt,
	})
}
