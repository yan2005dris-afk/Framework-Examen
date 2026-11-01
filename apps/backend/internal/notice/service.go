// Package notice consulta entregas y controla reintentos sin exponer correos ni cargas cifradas.
package notice

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/shared/domain"
	"time"
)

var (
	ErrForbidden   = errors.New("no tiene permiso para esta notificación")
	ErrNotFound    = errors.New("la notificación no existe")
	ErrConflict    = errors.New("la entrega cambió o no admite reintento")
	ErrInvalid     = errors.New("revise los parámetros de notificación")
	ErrUnavailable = errors.New("no se pudo confirmar la notificación")
)

type Delivery struct {
	ID        uuid.UUID `json:"id"`
	Action    string    `json:"accion"`
	Template  string    `json:"plantilla"`
	Status    string    `json:"estado"`
	Attempts  int       `json:"intentosCiclo"`
	Total     int       `json:"intentosTotales"`
	Retries   int       `json:"reenvios"`
	ErrorCode string    `json:"codigoError,omitempty"`
	CreatedAt time.Time `json:"creadaEn"`
	UpdatedAt time.Time `json:"actualizadaEn"`
}
type Event struct {
	ID        uuid.UUID `json:"id"`
	Status    string    `json:"estado"`
	ErrorCode string    `json:"codigoError,omitempty"`
	Attempt   int       `json:"intento"`
	Cycle     int       `json:"ciclo"`
	At        time.Time `json:"fecha"`
}
type RetryInput struct {
	PreviousRetries int    `json:"reenviosAnteriores"`
	Reason          string `json:"motivo"`
}
type Repository interface {
	Owner(context.Context, uuid.UUID) (uuid.UUID, error)
	List(context.Context, uuid.UUID, domain.Position, int) ([]Delivery, error)
	History(context.Context, uuid.UUID, uuid.UUID, domain.Position, int) ([]Event, error)
	Retry(context.Context, *identity.Applicant, uuid.UUID, uuid.UUID, RetryInput) error
}
type Service struct {
	repo    Repository
	auditor audit.Recorder
}

func NewService(r Repository, a audit.Recorder) *Service { return &Service{r, a} }
func (s *Service) access(ctx context.Context, a *identity.Applicant, id uuid.UUID, admin bool) error {
	if a == nil || a.ID == uuid.Nil || (admin && !identity.HasPermission(a, identity.PermissionNotifications)) {
		return ErrForbidden
	}
	owner, err := s.repo.Owner(ctx, id)
	if err != nil {
		return err
	}
	if !admin && owner != a.ID {
		return ErrForbidden
	}
	return nil
}
func (s *Service) record(ctx context.Context, a *identity.Applicant, id uuid.UUID, err error) error {
	return audit.Record(ctx, s.auditor, audit.Event{ActorID: a.ID, Role: a.Role, SubmissionID: id, ResourceID: id, ResourceType: "postulacion", Action: "notificaciones.consultadas", CreatedAt: time.Now().UTC()}, err)
}

/** List devuelve metadatos seguros; SMTP aceptado no significa llegada al buzón. */
func (s *Service) List(ctx context.Context, a *identity.Applicant, id uuid.UUID, q domain.CursorRequest, admin bool) (page domain.CursorPage[Delivery], err error) {
	if a == nil {
		return page, ErrForbidden
	}
	if admin {
		defer func() {
			err = s.record(ctx, a, id, err)
			if err != nil {
				page = domain.CursorPage[Delivery]{}
			}
		}()
	}
	if err = s.access(ctx, a, id, admin); err != nil {
		return page, err
	}
	q = domain.NewCursorRequest(q.Limit, q.Cursor)
	scope := "notices:" + id.String()
	p, err := domain.ParsePosition(q.Cursor, scope)
	if err != nil {
		return page, err
	}
	items, err := s.repo.List(ctx, id, p, q.Limit+1)
	if err != nil {
		return page, err
	}
	more := len(items) > q.Limit
	next := ""
	if more {
		items = items[:q.Limit]
		last := items[len(items)-1]
		next = domain.EncodePosition(scope, last.CreatedAt, last.ID)
	}
	return domain.NewCursorPage(items, next, more, q.Limit), nil
}

/** History conserva intentos anteriores sin destinatario, texto SMTP ni motivos libres. */
func (s *Service) History(ctx context.Context, a *identity.Applicant, id, mailID uuid.UUID, q domain.CursorRequest, admin bool) (page domain.CursorPage[Event], err error) {
	if a == nil {
		return page, ErrForbidden
	}
	if admin {
		defer func() {
			err = s.record(ctx, a, id, err)
			if err != nil {
				page = domain.CursorPage[Event]{}
			}
		}()
	}
	if err = s.access(ctx, a, id, admin); err != nil {
		return page, err
	}
	q = domain.NewCursorRequest(q.Limit, q.Cursor)
	scope := "notice-history:" + id.String() + ":" + mailID.String()
	p, err := domain.ParsePosition(q.Cursor, scope)
	if err != nil {
		return page, err
	}
	items, err := s.repo.History(ctx, id, mailID, p, q.Limit+1)
	if err != nil {
		return page, err
	}
	more := len(items) > q.Limit
	next := ""
	if more {
		items = items[:q.Limit]
		last := items[len(items)-1]
		next = domain.EncodePosition(scope, last.At, last.ID)
	}
	return domain.NewCursorPage(items, next, more, q.Limit), nil
}
func (s *Service) Retry(ctx context.Context, a *identity.Applicant, id, mailID uuid.UUID, input RetryInput) error {
	if !identity.HasPermission(a, identity.PermissionPhases) {
		return ErrForbidden
	}
	if input.PreviousRetries < 0 || len(input.Reason) > 2000 || len(input.Reason) == 0 {
		return ErrInvalid
	}
	return s.repo.Retry(ctx, a, id, mailID, input)
}
