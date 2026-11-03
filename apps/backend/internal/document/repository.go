package document

import (
	"context"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"io"
	"time"
)

/** Repository define la persistencia y el bloqueo compartido con el envío.
 * WithinSubmission bloquea primero el postulante y luego la postulación;
 * change recibe un repositorio y contexto ligados a la misma transacción.
 */
type Repository interface {
	FindSubmission(context.Context, uuid.UUID) (*submission.Submission, error)
	WithinSubmission(context.Context, uuid.UUID, uuid.UUID, func(context.Context, *submission.Submission, Repository) error) error
	Find(context.Context, uuid.UUID) (*Document, error)
	List(context.Context, uuid.UUID) ([]Document, error)
	TotalByApplicant(context.Context, uuid.UUID) (int64, error)
	Create(context.Context, *Document) error
	Delete(context.Context, uuid.UUID) error
	QueueDeletion(context.Context, string) error
	PendingDeletions(context.Context) ([]string, error)
	CompleteDeletion(context.Context, string) error
}

/** Storage es el puerto de objetos; las descargas ocurren fuera de la API. */
type Storage interface {
	Save(context.Context, string, io.Reader) error
	Delete(context.Context, string) error
	PresignGet(context.Context, string, string, time.Duration) (string, error)
}
