package submission

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	shareddomain "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/shared/domain"
)

/**
 * CreateInput son los datos necesarios para abrir una postulación.
 */
type CreateInput struct {
	ApplicantID uuid.UUID
	Code        string
	Content     json.RawMessage
}

/**
 * Service expone los casos de uso del contexto de postulaciones.
 */
type Service struct {
	repo Repository
	now  func() time.Time
}

/**
 * NewService crea el servicio de postulaciones con su repositorio.
 *
 * @param {Repository} repo - Puerto de persistencia de postulaciones.
 * @returns {Service} Servicio listo para usar.
 */
func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

/**
 * Create abre una postulación en estado borrador.
 *
 * @param {Context} ctx - Contexto de la operación.
 * @param {CreateInput} input - Datos de la nueva postulación.
 * @returns {Submission} Postulación creada.
 * @throws {Error} Errores de dominio si los datos no son válidos.
 */
func (s *Service) Create(ctx context.Context, input CreateInput) (*Submission, error) {
	now := s.now()

	code := input.Code
	if code == "" {
		generated, err := GenerateCode(now)
		if err != nil {
			return nil, err
		}
		code = generated
	}

	item, err := NewSubmission(uuid.New(), input.ApplicantID, code, input.Content, now)
	if err != nil {
		return nil, err
	}

	if err := s.repo.Create(ctx, item); err != nil {
		return nil, err
	}

	return item, nil
}

/**
 * Get devuelve una postulación por su identificador.
 *
 * @param {Context} ctx - Contexto de la operación.
 * @param {uuid.UUID} id - Identificador de la postulación.
 * @returns {Submission} Postulación encontrada.
 * @throws {Error} ErrNotFound si la postulación no existe.
 */
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Submission, error) {
	if id == uuid.Nil {
		return nil, ErrInvalidIdentifier
	}

	item, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, ErrNotFound
	}

	return item, nil
}

/**
 * List devuelve la página de postulaciones de un postulante.
 *
 * @param {Context} ctx - Contexto de la operación.
 * @param {uuid.UUID} applicantID - Postulante cuyas postulaciones se listan.
 * @param {PageRequest} page - Paginación solicitada.
 * @returns {Page} Página de postulaciones.
 */
func (s *Service) List(ctx context.Context, applicantID uuid.UUID, page shareddomain.PageRequest) (shareddomain.Page[*Submission], error) {
	if applicantID == uuid.Nil {
		return shareddomain.Page[*Submission]{}, ErrInvalidApplicant
	}

	items, total, err := s.repo.ListByApplicant(ctx, applicantID, page.Limit(), page.Offset())
	if err != nil {
		return shareddomain.Page[*Submission]{}, err
	}

	return shareddomain.NewPage(items, total, page), nil
}

/**
 * ListCursor devuelve la página siguiente usando paginación por cursor. Es la
 * opción para listados grandes: el costo no crece con el desplazamiento y el
 * resultado es estable aunque se inserten postulaciones nuevas.
 *
 * @param {Context} ctx - Contexto de la operación.
 * @param {uuid.UUID} applicantID - Postulante cuyas postulaciones se listan.
 * @param {CursorRequest} request - Límite y cursor de la página anterior.
 * @returns {CursorPage} Página de postulaciones con el cursor siguiente.
 * @throws {Error} ErrInvalidCursor si el cursor no tiene la clave de orden esperada.
 */
func (s *Service) ListCursor(ctx context.Context, applicantID uuid.UUID, request shareddomain.CursorRequest) (shareddomain.CursorPage[*Submission], error) {
	if applicantID == uuid.Nil {
		return shareddomain.CursorPage[*Submission]{}, ErrInvalidApplicant
	}

	var before time.Time
	var beforeID uuid.UUID

	if request.Cursor != nil {
		rawTime, ok := request.Cursor.Value(0)
		if !ok {
			return shareddomain.CursorPage[*Submission]{}, fmt.Errorf("%w: falta la fecha del cursor", shareddomain.ErrInvalidCursor)
		}

		parsedTime, err := time.Parse(time.RFC3339Nano, rawTime)
		if err != nil {
			return shareddomain.CursorPage[*Submission]{}, fmt.Errorf("%w: la fecha del cursor no es válida", shareddomain.ErrInvalidCursor)
		}

		rawID, ok := request.Cursor.Value(1)
		if !ok {
			return shareddomain.CursorPage[*Submission]{}, fmt.Errorf("%w: falta el identificador del cursor", shareddomain.ErrInvalidCursor)
		}

		parsedID, err := uuid.Parse(rawID)
		if err != nil {
			return shareddomain.CursorPage[*Submission]{}, fmt.Errorf("%w: el identificador del cursor no es válido", shareddomain.ErrInvalidCursor)
		}

		before = parsedTime
		beforeID = parsedID
	}

	// Se lee un elemento extra para saber si existe una página siguiente.
	items, err := s.repo.ListByApplicantCursor(ctx, applicantID, before, beforeID, request.Limit+1)
	if err != nil {
		return shareddomain.CursorPage[*Submission]{}, err
	}

	hasMore := len(items) > request.Limit
	if hasMore {
		items = items[:request.Limit]
	}

	nextCursor := ""
	if hasMore && len(items) > 0 {
		last := items[len(items)-1]
		cursor := shareddomain.NewCursor(last.UpdatedAt.UTC().Format(time.RFC3339Nano), last.ID.String())
		nextCursor = cursor.Encode()
	}

	return shareddomain.NewCursorPage(items, nextCursor, hasMore, request.Limit), nil
}

/**
 * Update reemplaza el contenido del formulario de una postulación editable.
 *
 * @param {Context} ctx - Contexto de la operación.
 * @param {uuid.UUID} id - Identificador de la postulación.
 * @param {json.RawMessage} content - Nuevo contenido del formulario.
 * @returns {Submission} Postulación actualizada.
 * @throws {Error} ErrNotFound o ErrNotEditable según el estado del recurso.
 */
func (s *Service) Update(ctx context.Context, id uuid.UUID, content json.RawMessage) (*Submission, error) {
	item, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := item.UpdateContent(content, s.now()); err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, item); err != nil {
		return nil, err
	}

	return item, nil
}

/**
 * Delete elimina una postulación que todavía no fue enviada.
 *
 * @param {Context} ctx - Contexto de la operación.
 * @param {uuid.UUID} id - Identificador de la postulación.
 * @returns {void}
 * @throws {Error} ErrNotFound o ErrNotEditable según el estado del recurso.
 */
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	item, err := s.Get(ctx, id)
	if err != nil {
		return err
	}

	if !item.IsEditable() {
		return fmt.Errorf("%w: la postulación enviada no se puede eliminar", ErrNotEditable)
	}

	return s.repo.Delete(ctx, id)
}
