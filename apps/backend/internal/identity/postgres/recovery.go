package postgres

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
)

/** ResetPasswordAndSessions evita confirmar una contraseña dejando sesiones anteriores activas. */
func (r *Repository) ResetPasswordAndSessions(ctx context.Context, id uuid.UUID, hash string) error {
	if r.transactions == nil {
		return errors.New("recuperación transaccional no configurada")
	}
	return r.transactions.WithinTx(ctx, func(tx pgx.Tx) error {
		q := r.queries.WithTx(tx)
		if _, err := q.LockApplicantCredentials(ctx, id); err != nil {
			return err
		}
		if err := q.UpdateApplicantPassword(ctx, sqlcgen.UpdateApplicantPasswordParams{ID: id, PasswordHash: hash}); err != nil {
			return err
		}
		return q.DeleteSessionsByApplicantID(ctx, id)
	})
}

/** CreateAuthenticatedSession impide que un login validado con una clave anterior sobreviva al reset. */
func (r *Repository) CreateAuthenticatedSession(ctx context.Context, session *identity.Session, expectedHash string) error {
	if r.transactions == nil {
		return errors.New("emisión transaccional no configurada")
	}
	return r.transactions.WithinTx(ctx, func(tx pgx.Tx) error {
		q := r.queries.WithTx(tx)
		account, err := q.LockApplicantCredentials(ctx, session.ApplicantID)
		if err != nil {
			return err
		}
		if account.PasswordHash != expectedHash {
			return identity.ErrInvalidCredentials
		}
		return (&Repository{queries: q}).CreateSession(ctx, session)
	})
}
