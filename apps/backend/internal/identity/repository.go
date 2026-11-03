package identity

import (
	"context"
	"time"

	"github.com/google/uuid"
)

/** AttemptsStore cuenta los intentos fallidos de acceso y administra el bloqueo temporal de la cuenta. */
type AttemptsStore interface {
	// IsLocked indica si la cuenta está bloqueada por exceso de intentos.
	IsLocked(ctx context.Context, identifier string) (bool, error)
	// RecordFailure acumula un intento fallido y devuelve el total.
	RecordFailure(ctx context.Context, identifier string) (int64, error)
	// ResetAttempts borra el contador tras un acceso exitoso.
	ResetAttempts(ctx context.Context, identifier string) error
	// LockRemaining devuelve el tiempo que falta para desbloquear la cuenta.
	LockRemaining(ctx context.Context, identifier string) (time.Duration, error)
}

/** OTPStore emite, verifica y cancela desafíos OTP ligados a una finalidad, una cuenta y un correo. */
type OTPStore interface {
	// Issue emite un desafío para la finalidad y la cuenta indicadas.
	Issue(ctx context.Context, purpose OTPPurpose, identifier, email, code string) (OTPChallenge, error)
	// Cancel invalida un desafío que no pudo entregarse.
	Cancel(ctx context.Context, challenge OTPChallenge) error
	// Verify consume el desafío si la finalidad, la cuenta, el correo y el código coinciden.
	Verify(ctx context.Context, purpose OTPPurpose, identifier, email, id, code string) (bool, error)
	// Active indica si el desafío sigue vigente: no fue reemplazado, consumido
	// ni agotó sus intentos. La cola de correo la consulta antes de entregar.
	Active(ctx context.Context, key, id string) (bool, error)
}

/** PasswordRecoveryRepository confirma cambio de clave y revocación de sesiones como una unidad. */
type PasswordRecoveryRepository interface {
	ResetPasswordAndSessions(context.Context, uuid.UUID, string) error
}

/** AuthenticatedSessionRepository serializa la emisión con cambios de clave y comprueba la versión validada. */
type AuthenticatedSessionRepository interface {
	CreateAuthenticatedSession(context.Context, *Session, string) error
}

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
