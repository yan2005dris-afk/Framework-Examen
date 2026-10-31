package submission

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	shareddomain "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/shared/domain"
)

/** Notifier es el puerto de confirmación; el correo se emite después del commit. */
type Notifier interface {
	Confirm(context.Context, *Submission) error
}

/** DocumentVerifier comprueba respaldos reales mientras se mantiene el bloqueo del envío. */
type DocumentVerifier interface {
	VerifyReferences(context.Context, uuid.UUID, *Form) error
}

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
	repo      Repository
	now       func() time.Time
	notifier  Notifier
	documents DocumentVerifier
}

/** WithDocuments configura la validación obligatoria de respaldos en la composición.
 * @returns {Service} Servicio configurado antes de atender peticiones.
 */
func (s *Service) WithDocuments(verifier DocumentVerifier) *Service {
	s.documents = verifier
	return s
}

/**
 * NewService crea el servicio de postulaciones con su repositorio.
 *
 * @param {Repository} repo - Puerto de persistencia de postulaciones.
 * @returns {Service} Servicio listo para usar.
 */
func NewService(repo Repository, notifiers ...Notifier) *Service {
	s := &Service{repo: repo, now: time.Now}
	if len(notifiers) > 0 {
		s.notifier = notifiers[0]
	}
	return s
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
	content, err := validateContent(input.Content, false, now)
	if err != nil {
		return nil, err
	}

	code := input.Code
	if code == "" {
		generated, err := GenerateCode(now)
		if err != nil {
			return nil, err
		}
		code = generated
	}

	id, err := newIdentifier()
	if err != nil {
		return nil, err
	}

	item, err := NewSubmission(id, input.ApplicantID, code, content, now)
	if err != nil {
		return nil, err
	}

	if err := s.repo.Create(ctx, item); err != nil {
		return nil, err
	}

	return item, nil
}

/**
 * newIdentifier genera un identificador UUIDv7: los 48 bits iniciales son
 * una marca de tiempo en milisegundos, así que los identificadores nuevos
 * ordenan por orden de creación. Eso mantiene el índice del PRIMARY KEY
 * compacto y hace que el desempate de la paginación por cursor sea
 * consistente con la fecha de creación.
 *
 * La base de datos declara el mismo valor como DEFAULT (uuidv7() de
 * PostgreSQL 18) para los caminos de inserción que no lo fijen.
 *
 * @returns {uuid.UUID} Identificador nuevo.
 * @returns {Error} Si la fuente de aleatoriedad no está disponible.
 */
func newIdentifier() (uuid.UUID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, fmt.Errorf("no se pudo generar el identificador de la postulación: %w", err)
	}

	return id, nil
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
	return s.repo.Mutate(ctx, id, func(ctx context.Context, item *Submission) error {
		if !item.IsEditable() {
			return ErrNotEditable
		}
		now := s.now()
		validated, err := validateContent(content, false, now)
		if err != nil {
			return err
		}
		return item.UpdateContent(validated, now)
	})
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

/**
 * SaveSection mezcla los campos recibidos con la sección existente. Las listas
 * se reemplazan completas y las otras secciones se conservan. El bloqueo de
 * fila evita perder cambios cuando dos peticiones guardan simultáneamente.
 * @param {int} number - Sección entre 1 y 9.
 * @param {json.RawMessage} content - Campos de la sección, sin envoltorio.
 * @returns {Submission, error} Postulación guardada o error de validación.
 */
func (s *Service) SaveSection(ctx context.Context, id uuid.UUID, number int, content json.RawMessage) (*Submission, error) {
	if number < 1 || number > 9 {
		return nil, ErrInvalidSection
	}
	if _, err := CanonicalContent(content); err != nil {
		return nil, err
	}
	wrapper := fmt.Sprintf(`{"seccion%d":%s}`, number, string(content))
	if _, err := DecodeForm(json.RawMessage(wrapper)); err != nil {
		return nil, ErrInvalidContent
	}
	return s.repo.Mutate(ctx, id, func(ctx context.Context, item *Submission) error {
		if !item.IsEditable() {
			return ErrNotEditable
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(item.Content, &fields); err != nil || fields == nil {
			return ErrInvalidContent
		}
		key := fmt.Sprintf("seccion%d", number)
		merged, err := mergeSection(fields[key], content)
		if err != nil {
			return err
		}
		fields[key] = merged
		raw, err := json.Marshal(fields)
		if err != nil {
			return ErrInvalidContent
		}
		now := s.now()
		validated, err := validateContent(raw, false, now)
		if err != nil {
			return err
		}
		return item.UpdateContent(validated, now)
	})
}

/**
 * Submit valida el formulario completo y sella exactamente el contenido que
 * se confirma bajo bloqueo de fila. Un fallo SMTP no revierte el envío.
 * @param {Consent} consent - Aceptaciones finales opcionales si ya se guardaron.
 * @returns {Submission, error} Postulación sellada o regla incumplida.
 */
func (s *Service) Submit(ctx context.Context, id uuid.UUID, consent *Consent) (*Submission, error) {
	item, err := s.repo.Mutate(ctx, id, func(ctx context.Context, item *Submission) error {
		if !item.IsEditable() {
			return ErrNotEditable
		}
		form, err := DecodeForm(item.Content)
		if err != nil {
			return err
		}
		if consent != nil {
			form.Consent = consent
		}
		now := s.now()
		if err := ValidateForm(form, true, now); err != nil {
			return err
		}
		if s.documents != nil {
			if err := s.documents.VerifyReferences(ctx, id, form); err != nil {
				return err
			}
		}
		if form.Consent != nil {
			form.Consent.Statements = consentStatements
			form.Consent.Version = "v1.4"
			form.Consent.AcceptedAt = now.UTC().Format(time.RFC3339)
		}
		normalizeDurations(form, now)
		raw, err := json.Marshal(form)
		if err != nil {
			return ErrInvalidContent
		}
		if err := item.UpdateContent(raw, now); err != nil {
			return err
		}
		return item.MarkAsSent(now)
	})
	if err != nil {
		return nil, err
	}
	if s.notifier != nil {
		if err := s.notifier.Confirm(ctx, item); err != nil {
			// No se incluyen destinatario, contenido ni errores SMTP con datos personales.
			slog.WarnContext(ctx, "la postulación se selló pero no se pudo enviar el correo de confirmación")
		}
	}
	return item, nil
}

/** validateContent tipa, valida y recalcula los valores derivados del formulario. */
func validateContent(content json.RawMessage, complete bool, now time.Time) (json.RawMessage, error) {
	form, err := DecodeForm(content)
	if err != nil {
		return nil, err
	}
	if err := ValidateForm(form, complete, now); err != nil {
		return nil, err
	}
	normalizeDurations(form, now)
	raw, err := json.Marshal(form)
	if err != nil {
		return nil, ErrInvalidContent
	}
	return CanonicalContent(raw)
}

/** mergeSection mezcla objetos anidados, reemplaza arrays y rechaza nulos. */
func mergeSection(previous, patch json.RawMessage) (json.RawMessage, error) {
	fields := map[string]json.RawMessage{}
	if len(previous) > 0 {
		if err := json.Unmarshal(previous, &fields); err != nil || fields == nil {
			return nil, ErrInvalidContent
		}
	}
	var changes map[string]json.RawMessage
	if err := json.Unmarshal(patch, &changes); err != nil || changes == nil {
		return nil, ErrInvalidContent
	}
	for key, value := range changes {
		if string(value) == "null" {
			delete(fields, key)
			continue
		}
		if len(value) > 0 && value[0] == '{' {
			merged, err := mergeSection(fields[key], value)
			if err != nil {
				return nil, err
			}
			fields[key] = merged
		} else {
			fields[key] = value
		}
	}
	result, err := json.Marshal(fields)
	if err != nil {
		return nil, ErrInvalidContent
	}
	return result, nil
}
