package audit

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/shared/domain"
	"time"
)

var (
	ErrForbidden           = errors.New("no tiene permiso para consultar este historial")
	ErrNotFound            = errors.New("el expediente no existe")
	ErrInvalidEvent        = errors.New("los metadatos del evento no son válidos")
	ErrTransactionRequired = errors.New("el evento de negocio requiere la transacción del cambio")
)

/** Entry es la vista mínima del historial: nunca devuelve payloads, operadores ni motivos libres. */
type Entry struct {
	ID           uuid.UUID  `json:"id"`
	ActorID      *uuid.UUID `json:"actorId,omitempty"`
	Role         string     `json:"rol"`
	Action       string     `json:"accion"`
	ResourceID   uuid.UUID  `json:"recursoId"`
	ResourceType string     `json:"tipoRecurso"`
	Result       string     `json:"resultado"`
	CreatedAt    time.Time  `json:"createdAt"`
}

/** Owner conserva la titularidad incluso si el borrador ya fue retirado. */
type Owner struct {
	ApplicantID uuid.UUID
	Status      string
}

/** HistoryRepository une los historiales existentes sin copiar información del expediente. */
type HistoryRepository interface {
	Owner(context.Context, uuid.UUID) (*Owner, error)
	History(context.Context, uuid.UUID, domain.Position, int) ([]Entry, error)
}

/** WorkflowRecorder se usa dentro del commit de cada futuro módulo de negocio. */
type WorkflowRecorder interface {
	RecordWorkflow(context.Context, Event) error
}

/** ValidWorkflow limita el puerto a acciones conocidas y metadatos sin texto libre. */
func ValidWorkflow(e Event) bool {
	actions := map[string]bool{"convocatoria.creada": true, "convocatoria.configurada": true, "fase.publicada": true, "fase.habilitada": true, "fase.cerrada": true,
		"elegibilidad.verificada": true, "meritos.revisados": true, "meritos.calificados": true,
		"evaluacion.iniciada": true, "evaluacion.finalizada": true, "reconsideracion.presentada": true,
		"reconsideracion.resuelta": true, "resultado.publicado": true}
	resources := map[string]bool{"postulacion": true, "convocatoria": true, "evaluacion": true, "reconsideracion": true}
	return actions[e.Action] && resources[e.ResourceType] && e.ResourceID != uuid.Nil && !e.CreatedAt.IsZero() &&
		(e.Result == "permitido" || e.Result == "rechazado") &&
		((e.Role == "sistema" && e.ActorID == uuid.Nil) ||
			((e.Role == identity.RoleApplicant || e.Role == identity.RoleAnalyst || e.Role == identity.RoleAdministrator) && e.ActorID != uuid.Nil))
}

/** HistoryService aplica titularidad o permiso administrativo antes de consultar eventos. */
type HistoryService struct {
	repo     HistoryRepository
	recorder Recorder
}

/** NewHistoryService recibe consultas y auditoría de la lectura administrativa. */
func NewHistoryService(repo HistoryRepository, recorder Recorder) *HistoryService {
	return &HistoryService{repo: repo, recorder: recorder}
}

/** List pagina por fecha/UUID sin count ni OFFSET; la ruta propia nunca eleva privilegios. */
func (s *HistoryService) List(ctx context.Context, actor *identity.Applicant, id uuid.UUID, request domain.CursorRequest, admin bool) (page domain.CursorPage[Entry], err error) {
	if actor == nil || actor.ID == uuid.Nil || id == uuid.Nil {
		return page, ErrForbidden
	}
	if admin && !identity.HasPermission(actor, identity.PermissionReview) {
		return page, ErrForbidden
	}
	if admin {
		defer func() {
			err = Record(ctx, s.recorder, Event{ActorID: actor.ID, Role: actor.Role, SubmissionID: id,
				ResourceID: id, ResourceType: "postulacion", Action: "historial.consultado", CreatedAt: time.Now().UTC()}, err)
			if err != nil {
				page = domain.CursorPage[Entry]{}
			}
		}()
	}
	owner, err := s.repo.Owner(ctx, id)
	if err != nil {
		return page, err
	}
	if owner == nil {
		return page, ErrNotFound
	}
	if (!admin && owner.ApplicantID != actor.ID) || (admin && owner.Status != "submitted") {
		return page, ErrForbidden
	}
	request = domain.NewCursorRequest(request.Limit, request.Cursor)
	scope := "historial:" + id.String()
	position, err := domain.ParsePosition(request.Cursor, scope)
	if err != nil {
		return page, err
	}
	items, err := s.repo.History(ctx, id, position, request.Limit+1)
	if err != nil {
		return page, err
	}
	more := len(items) > request.Limit
	next := ""
	if more {
		items = items[:request.Limit]
		last := items[len(items)-1]
		next = domain.EncodePosition(scope, last.CreatedAt, last.ID)
	}
	return domain.NewCursorPage(items, next, more, request.Limit), nil
}
