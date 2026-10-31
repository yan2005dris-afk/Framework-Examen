package submission

import (
	"context"
	"time"

	"github.com/google/uuid"
)

/**
 * Repository es el contrato de persistencia declarado por la postulación.
 * El adaptador concreto que habla con PostgreSQL vive en ./postgres.
 */
type Repository interface {
	Create(ctx context.Context, item *Submission) error
	FindByID(ctx context.Context, id uuid.UUID) (*Submission, error)
	ListByApplicant(ctx context.Context, applicantID uuid.UUID, limit int, offset int) ([]*Submission, int64, error)
	ListByApplicantCursor(ctx context.Context, applicantID uuid.UUID, before time.Time, beforeID uuid.UUID, limit int) ([]*Submission, error)
	Update(ctx context.Context, item *Submission) error
	Delete(ctx context.Context, id uuid.UUID) error
	Mutate(ctx context.Context, id uuid.UUID, change func(*Submission) error) (*Submission, error)
}
