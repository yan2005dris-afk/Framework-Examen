// Package postgres es el adaptador de persistencia para el contexto de identidad.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/db"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
)

/**
 * Repository implementa la persistencia de postulantes, sesiones y recuperaciones
 * utilizando sqlc sobre PostgreSQL.
 */
type Repository struct {
	queries *sqlcgen.Queries
}

/**
 * NewRepository crea el adaptador de persistencia con las consultas de sqlc.
 *
 * @param {*sqlcgen.Queries} queries - Consultas tipadas de sqlc.
 * @returns {*Repository} Repositorio listo para usar.
 */
func NewRepository(queries *sqlcgen.Queries) *Repository {
	return &Repository{queries: queries}
}

// ── ApplicantRepository ──────────────────────────────────────────────────────

func (r *Repository) Create(ctx context.Context, item *identity.Applicant) error {
	params := sqlcgen.CreateApplicantParams{
		ID:               item.ID,
		Cedula:           item.IDCard.String(),
		Nombres:          item.FirstName,
		Apellidos:        item.LastName,
		Correo:           item.Email,
		Role:             item.Role,
		PasswordHash:     item.PasswordHash,
		MetodoValidacion: item.ValidationMethod,
		IsVerified:       item.IsVerified,
	}

	_, err := r.queries.CreateApplicant(ctx, params)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return identity.ErrApplicantAlreadyExists
		}
		return fmt.Errorf("postgres: error al crear postulante: %w", err)
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*identity.Applicant, error) {
	queries := r.queries
	if tx, ok := db.Transaction(ctx); ok {
		queries = queries.WithTx(tx)
	}
	row, err := queries.GetApplicantByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, identity.ErrApplicantNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: GetByID: %w", err)
	}
	return toDomainApplicant(row), nil
}

func (r *Repository) GetByIDCard(ctx context.Context, idCard identity.IDCard) (*identity.Applicant, error) {
	row, err := r.queries.GetApplicantByCedula(ctx, idCard.String())
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, identity.ErrApplicantNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: GetByIDCard: %w", err)
	}
	return toDomainApplicant(row), nil
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (*identity.Applicant, error) {
	row, err := r.queries.GetApplicantByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, identity.ErrApplicantNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: GetByEmail: %w", err)
	}
	return toDomainApplicant(row), nil
}

func (r *Repository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	params := sqlcgen.UpdateApplicantPasswordParams{
		ID:           id,
		PasswordHash: passwordHash,
	}
	if err := r.queries.UpdateApplicantPassword(ctx, params); err != nil {
		return fmt.Errorf("postgres: UpdatePassword: %w", err)
	}
	return nil
}

// ── SessionRepository ────────────────────────────────────────────────────────

func (r *Repository) CreateSession(ctx context.Context, session *identity.Session) error {
	var ipPtr, uaPtr *string
	if session.IP != "" {
		ipPtr = &session.IP
	}
	if session.UserAgent != "" {
		uaPtr = &session.UserAgent
	}

	params := sqlcgen.CreateSessionParams{
		ID:          session.ID,
		ApplicantID: session.ApplicantID,
		TokenHash:   session.TokenHash,
		ExpiresAt:   session.ExpiresAt,
		Ip:          ipPtr,
		UserAgent:   uaPtr,
	}

	_, err := r.queries.CreateSession(ctx, params)
	if err != nil {
		return fmt.Errorf("postgres: CreateSession: %w", err)
	}
	return nil
}

func (r *Repository) GetWithApplicant(ctx context.Context, tokenHash string) (*identity.Session, *identity.Applicant, error) {
	row, err := r.queries.GetSessionWithApplicant(ctx, tokenHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, identity.ErrSessionExpired
	}
	if err != nil {
		return nil, nil, fmt.Errorf("postgres: GetWithApplicant: %w", err)
	}

	session := &identity.Session{
		ID:          row.SessionID,
		ApplicantID: row.ApplicantID,
		TokenHash:   row.TokenHash,
		ExpiresAt:   row.ExpiresAt,
		CreatedAt:   row.SessionCreatedAt,
	}
	if row.Ip != nil {
		session.IP = *row.Ip
	}
	if row.UserAgent != nil {
		session.UserAgent = *row.UserAgent
	}

	applicant := &identity.Applicant{
		ID:               row.ApplicantID,
		IDCard:           identity.IDCard(row.Cedula),
		FirstName:        row.Nombres,
		LastName:         row.Apellidos,
		Email:            row.Correo,
		Role:             row.Role,
		ValidationMethod: row.MetodoValidacion,
		IsVerified:       row.IsVerified,
		RegisteredAt:     row.FechaRegistro,
	}

	return session, applicant, nil
}

func (r *Repository) DeleteByTokenHash(ctx context.Context, tokenHash string) error {
	return r.queries.DeleteSessionByTokenHash(ctx, tokenHash)
}

func (r *Repository) DeleteByApplicantID(ctx context.Context, applicantID uuid.UUID) error {
	return r.queries.DeleteSessionsByApplicantID(ctx, applicantID)
}

func (r *Repository) DeleteExpired(ctx context.Context) error {
	return r.queries.DeleteExpiredSessions(ctx)
}

// ── PasswordResetRepository ──────────────────────────────────────────────────

func (r *Repository) CreatePasswordReset(ctx context.Context, applicantID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	params := sqlcgen.CreatePasswordResetParams{
		ID:          uuid.New(),
		ApplicantID: applicantID,
		TokenHash:   tokenHash,
		ExpiresAt:   expiresAt,
	}
	_, err := r.queries.CreatePasswordReset(ctx, params)
	if err != nil {
		return fmt.Errorf("postgres: CreatePasswordReset: %w", err)
	}
	return nil
}

func (r *Repository) GetValid(ctx context.Context, tokenHash string) (uuid.UUID, error) {
	row, err := r.queries.GetValidPasswordReset(ctx, tokenHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, identity.ErrInvalidResetToken
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("postgres: GetValidPasswordReset: %w", err)
	}
	return row.ApplicantID, nil
}

func (r *Repository) MarkUsed(ctx context.Context, tokenHash string) error {
	return r.queries.MarkPasswordResetUsed(ctx, tokenHash)
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func toDomainApplicant(row sqlcgen.Applicant) *identity.Applicant {
	return &identity.Applicant{
		ID:               row.ID,
		IDCard:           identity.IDCard(row.Cedula),
		FirstName:        row.Nombres,
		LastName:         row.Apellidos,
		Email:            row.Correo,
		Role:             row.Role,
		PasswordHash:     row.PasswordHash,
		IsVerified:       row.IsVerified,
		ValidationMethod: row.MetodoValidacion,
		RegisteredAt:     row.FechaRegistro,
	}
}
