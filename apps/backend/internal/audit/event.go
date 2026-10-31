package audit

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"time"
)

var ErrUnavailable = errors.New("no se pudo registrar el acceso administrativo")

/** Event registra metadatos de acceso sin copiar formularios, nombres ni URLs firmadas. */
type Event struct {
	SubmissionID uuid.UUID
	ActorID      uuid.UUID
	Role         string
	Action       string
	ResourceID   uuid.UUID
	ResourceType string
	Result       string
	CreatedAt    time.Time
}

/** Recorder es el puerto de auditoría; el acceso se entrega solo si pudo registrarse. */
type Recorder interface {
	Record(context.Context, Event) error
}

/** Record conserva el resultado antes de devolver datos o el error original.
 * @returns {error} Fallo de auditoría o resultado de la consulta.
 */
func Record(ctx context.Context, recorder Recorder, event Event, result error) error {
	if recorder == nil {
		return ErrUnavailable
	}
	if event.SubmissionID == uuid.Nil && event.ResourceType == "postulacion" {
		event.SubmissionID = event.ResourceID
	}
	event.Result = "permitido"
	if result != nil {
		event.Result = "rechazado"
	}
	if err := recorder.Record(ctx, event); err != nil {
		return ErrUnavailable
	}
	return result
}
