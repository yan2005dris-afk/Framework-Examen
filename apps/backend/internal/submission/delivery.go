package submission

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"time"
)

var ErrNotificationUnavailable = errors.New("no se pudo registrar la confirmación del envío")

/** DeliveryStatus es metadato operativo separado del JSON sellado y de su hash. */
type DeliveryStatus struct {
	Status    string    `json:"estado"`
	Attempts  int       `json:"intentos"`
	ErrorCode string    `json:"codigoError,omitempty"`
	UpdatedAt time.Time `json:"actualizadoEn"`
}

/** ConfirmationQueue persiste con la transacción del sello; no realiza I/O SMTP dentro del commit. */
type ConfirmationQueue interface {
	Enqueue(context.Context, *Submission) error
	State(context.Context, uuid.UUID) (*DeliveryStatus, error)
}

/** WithConfirmationQueue hace obligatorio registrar la confirmación antes de confirmar el sello. */
func (s *Service) WithConfirmationQueue(queue ConfirmationQueue) *Service {
	s.confirmations = queue
	return s
}
