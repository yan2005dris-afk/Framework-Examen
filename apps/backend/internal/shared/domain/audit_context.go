package domain

import (
	"context"
	"github.com/google/uuid"
)

type auditContextKey struct{}

/** AuditContext lleva sólo identidad autenticada y correlación, nunca formularios o tokens. */
type AuditContext struct {
	ActorID   uuid.UUID
	Role      string
	RequestID uuid.UUID
}

/** AuditFromContext obtiene metadatos sin crear identidad ni atribuciones retrospectivas.
 * @param {context.Context} ctx - Contexto de la solicitud o trabajo.
 * @returns {AuditContext} Metadatos presentes o valor vacío.
 */
func AuditFromContext(ctx context.Context) AuditContext {
	a, _ := ctx.Value(auditContextKey{}).(AuditContext)
	return a
}

/** WithAuditActor agrega la identidad que resolvió la sesión, sin modificar el contexto anterior.
 * @returns {context.Context} Contexto hijo con actor y rol.
 */
func WithAuditActor(ctx context.Context, id uuid.UUID, role string) context.Context {
	a := AuditFromContext(ctx)
	a.ActorID = id
	a.Role = role
	return context.WithValue(ctx, auditContextKey{}, a)
}

/** WithAuditRequest enlaza un UUID de solicitud validado con la auditoría transaccional.
 * @returns {context.Context} Contexto hijo con correlación.
 */
func WithAuditRequest(ctx context.Context, id uuid.UUID) context.Context {
	a := AuditFromContext(ctx)
	a.RequestID = id
	return context.WithValue(ctx, auditContextKey{}, a)
}
