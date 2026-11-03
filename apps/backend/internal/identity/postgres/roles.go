package postgres

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
)

/** ChangeRole bloquea actor y destino en orden estable y audita mediante el trigger SQL.
 * @returns {Applicant, error} Cuenta actualizada únicamente después del commit.
 */
func (r *Repository) ChangeRole(ctx context.Context, actorID, target uuid.UUID, role, reason string) (*identity.Applicant, error) {
	if !identity.ValidRoleChange(role, reason) {
		return nil, identity.ErrInvalidRoleChange
	}
	if r.transactions == nil {
		return nil, errors.New("transacciones de roles no configuradas")
	}
	var applicant *identity.Applicant
	err := r.transactions.WithinTx(ctx, func(tx pgx.Tx) error {
		queries := r.queries.WithTx(tx)
		rows, err := queries.LockApplicantsForRoleChange(ctx, []uuid.UUID{actorID, target})
		if err != nil {
			return err
		}
		actorAllowed, targetExists := false, false
		for _, row := range rows {
			if row.ID == actorID && row.Role == identity.RoleAdministrator {
				actorAllowed = true
			}
			if row.ID == target {
				targetExists = true
			}
		}
		if !actorAllowed {
			return identity.ErrRoleForbidden
		}
		if !targetExists {
			return identity.ErrApplicantNotFound
		}
		if err := queries.SetRoleChangeContext(ctx, sqlcgen.SetRoleChangeContextParams{
			ActorID: actorID.String(), Operator: "administrador:" + actorID.String(), Reason: reason,
		}); err != nil {
			return err
		}
		row, err := queries.ChangeApplicantRole(ctx, sqlcgen.ChangeApplicantRoleParams{ID: target, Role: role})
		if err != nil {
			return err
		}
		applicant = toDomainApplicant(row)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return applicant, nil
}
