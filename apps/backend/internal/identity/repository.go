package identity

import (
	"context"
	"time"

	"github.com/google/uuid"
)

/**
 * ApplicantRepository define los métodos de persistencia para postulantes.
 */
type ApplicantRepository interface {
	Create(ctx context.Context, applicant *Applicant) error
	GetByID(ctx context.Context, id uuid.UUID) (*Applicant, error)
	GetByIDCard(ctx context.Context, idCard IDCard) (*Applicant, error)
	GetByEmail(ctx context.Context, email string) (*Applicant, error)
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error
}

/**
 * SessionRepository define los métodos de persistencia para sesiones de usuario.
 */
type SessionRepository interface {
	CreateSession(ctx context.Context, session *Session) error
	GetWithApplicant(ctx context.Context, tokenHash string) (*Session, *Applicant, error)
	DeleteByTokenHash(ctx context.Context, tokenHash string) error
	DeleteByApplicantID(ctx context.Context, applicantID uuid.UUID) error
	DeleteExpired(ctx context.Context) error
}

/**
 * PasswordResetRepository define los métodos de persistencia para solicitudes de recuperación.
 */
type PasswordResetRepository interface {
	CreatePasswordReset(ctx context.Context, applicantID uuid.UUID, tokenHash string, expiresAt time.Time) error
	GetValid(ctx context.Context, tokenHash string) (uuid.UUID, error)
	MarkUsed(ctx context.Context, tokenHash string) error
}
