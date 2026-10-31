package submission

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"

	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/httpx"
	shareddomain "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/shared/domain"
)

/**
 * Handler expone el contexto de postulaciones por HTTP.
 */
type Handler struct {
	service *Service
}

/**
 * NewHandler crea el handler de postulaciones.
 *
 * @param {Service} service - Servicio del contexto.
 * @returns {Handler} Handler listo para registrar sus rutas.
 */
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

/**
 * RegisterRoutes registra las rutas del contexto sobre el router de la API.
 *
 * @param {chi.Router} router - Router raíz de la API.
 */
func (h *Handler) RegisterRoutes(router chi.Router) {
	router.With(identity.RequirePermission(identity.PermissionReview)).Get("/api/v1/admin/submissions/{id}", h.review)
	router.Route("/api/v1/submissions", func(r chi.Router) {
		r.Get("/form-definition", h.definition)
		r.Post("/", h.create)
		r.Get("/", h.list)
		r.Route("/{id}", func(r chi.Router) {
			r.Use(h.requireOwner)
			r.Get("/", h.get)
			r.Put("/", h.update)
			r.Delete("/", h.delete)
			r.Post("/sections/{n}", h.saveSection)
			r.Post("/submit", h.submit)
		})
	})
}

/** review entrega el expediente sellado por una ruta administrativa separada. */
func (h *Handler) review(w http.ResponseWriter, r *http.Request) {
	actor, _ := identity.ApplicantFromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, r, ErrInvalidIdentifier)
		return
	}
	item, err := h.service.GetForReview(r.Context(), actor, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.Respond(w, r, http.StatusOK, NewResponse(item))
}

/** definition entrega los mismos catálogos y literales usados en la validación.
 * @param {http.Request} request - Petición con sesión; no incluye datos personales.
 */
func (h *Handler) definition(writer http.ResponseWriter, request *http.Request) {
	if _, ok := identity.ApplicantFromContext(request.Context()); !ok {
		httpx.RespondError(writer, request, 401, "no_autenticado", "Debe iniciar sesión.")
		return
	}
	definition, err := FormDefinition()
	if err != nil {
		writeError(writer, request, err)
		return
	}
	httpx.Respond(writer, request, http.StatusOK, definition)
}

/**
 * create abre una postulación en estado borrador.
 *
 * @param {http.ResponseWriter} writer - Respuesta HTTP en construcción.
 * @param {http.Request} request - Petición recibida.
 */
func (h *Handler) create(writer http.ResponseWriter, request *http.Request) {
	actor, ok := identity.ApplicantFromContext(request.Context())
	if !ok {
		httpx.RespondError(writer, request, 401, "no_autenticado", "Debe iniciar sesión.")
		return
	}
	var body CreateRequest
	if err := decodeJSON(request, &body); err != nil {
		httpx.RespondError(writer, request, http.StatusBadRequest, "cuerpo_invalido", "El cuerpo de la petición no es válido.")
		return
	}

	applicantID := actor.ID
	if body.ApplicantID != "" && body.ApplicantID != actor.ID.String() {
		writeError(writer, request, ErrForbidden)
		return
	}

	item, err := h.service.Create(request.Context(), CreateInput{
		CompetitionID: body.CompetitionID,
		ApplicantID:   applicantID,
		Code:          body.Code,
		Content:       body.Content,
	})
	if err != nil {
		writeError(writer, request, err)
		return
	}

	httpx.Respond(writer, request, http.StatusCreated, NewResponse(item))
}

/**
 * list devuelve las postulaciones de un postulante. Acepta paginación por
 * offset (page, pageSize) o por cursor (limit, cursor). El modo cursor se
 * activa cuando llega limit o cursor; en caso contrario se usa offset.
 *
 * @param {http.ResponseWriter} writer - Respuesta HTTP en construcción.
 * @param {http.Request} request - Petición recibida.
 */
func (h *Handler) list(writer http.ResponseWriter, request *http.Request) {
	actor, ok := identity.ApplicantFromContext(request.Context())
	if !ok {
		httpx.RespondError(writer, request, 401, "no_autenticado", "Debe iniciar sesión.")
		return
	}
	applicantID := actor.ID
	if value := request.URL.Query().Get("applicantId"); value != "" && value != actor.ID.String() {
		writeError(writer, request, ErrForbidden)
		return
	}

	query := request.URL.Query()
	if query.Get("cursor") != "" || query.Get("limit") != "" {
		h.listCursor(writer, request, applicantID)
		return
	}

	page, err := httpx.ParsePageRequest(request)
	if err != nil {
		httpx.RespondError(writer, request, http.StatusBadRequest, "paginacion_invalida", "Los parámetros de paginación no son válidos.")
		return
	}

	result, err := h.service.List(request.Context(), applicantID, page)
	if err != nil {
		writeError(writer, request, err)
		return
	}

	items := make([]Response, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, NewResponse(item))
	}

	httpx.Respond(writer, request, http.StatusOK, shareddomain.NewPage(items, result.Total, page))
}

/**
 * listCursor devuelve las postulaciones de un postulante usando paginación por
 * cursor.
 *
 * @param {http.ResponseWriter} writer - Respuesta HTTP en construcción.
 * @param {http.Request} request - Petición recibida.
 * @param {uuid.UUID} applicantID - Postulante cuyas postulaciones se listan.
 */
func (h *Handler) listCursor(writer http.ResponseWriter, request *http.Request, applicantID uuid.UUID) {
	cursorRequest, err := httpx.ParseCursorRequest(request)
	if err != nil {
		httpx.RespondError(writer, request, http.StatusBadRequest, "cursor_invalido", "El cursor de paginación no es válido.")
		return
	}

	result, err := h.service.ListCursor(request.Context(), applicantID, cursorRequest)
	if err != nil {
		writeError(writer, request, err)
		return
	}

	items := make([]Response, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, NewResponse(item))
	}

	httpx.Respond(writer, request, http.StatusOK, shareddomain.NewCursorPage(items, result.NextCursor, result.HasMore, result.Limit))
}

/**
 * get devuelve una postulación por su identificador.
 *
 * @param {http.ResponseWriter} writer - Respuesta HTTP en construcción.
 * @param {http.Request} request - Petición recibida.
 */
func (h *Handler) get(writer http.ResponseWriter, request *http.Request) {
	id, err := uuid.Parse(chi.URLParam(request, "id"))
	if err != nil {
		httpx.RespondError(writer, request, http.StatusBadRequest, "identificador_invalido", "El identificador de la postulación no es válido.")
		return
	}

	item, err := h.service.Get(request.Context(), id)
	if err != nil {
		writeError(writer, request, err)
		return
	}

	httpx.Respond(writer, request, http.StatusOK, NewResponse(item))
}

/**
 * update reemplaza el contenido del formulario de una postulación editable.
 *
 * @param {http.ResponseWriter} writer - Respuesta HTTP en construcción.
 * @param {http.Request} request - Petición recibida.
 */
func (h *Handler) update(writer http.ResponseWriter, request *http.Request) {
	id, err := uuid.Parse(chi.URLParam(request, "id"))
	if err != nil {
		httpx.RespondError(writer, request, http.StatusBadRequest, "identificador_invalido", "El identificador de la postulación no es válido.")
		return
	}

	var body UpdateRequest
	if err := decodeJSON(request, &body); err != nil {
		httpx.RespondError(writer, request, http.StatusBadRequest, "cuerpo_invalido", "El cuerpo de la petición no es válido.")
		return
	}

	item, err := h.service.Update(request.Context(), id, body.Content)
	if err != nil {
		writeError(writer, request, err)
		return
	}

	httpx.Respond(writer, request, http.StatusOK, NewResponse(item))
}

/**
 * delete elimina una postulación que todavía no fue enviada.
 *
 * @param {http.ResponseWriter} writer - Respuesta HTTP en construcción.
 * @param {http.Request} request - Petición recibida.
 */
func (h *Handler) delete(writer http.ResponseWriter, request *http.Request) {
	id, err := uuid.Parse(chi.URLParam(request, "id"))
	if err != nil {
		httpx.RespondError(writer, request, http.StatusBadRequest, "identificador_invalido", "El identificador de la postulación no es válido.")
		return
	}

	if err := h.service.Delete(request.Context(), id); err != nil {
		writeError(writer, request, err)
		return
	}

	writer.WriteHeader(http.StatusNoContent)
}

/**
 * decodeJSON deserializa el cuerpo de la petición rechazando campos
 * desconocidos para detectar errores de tipeo del cliente.
 *
 * @param {http.Request} request - Petición recibida.
 * @param {any} target - Destino de la deserialización.
 * @returns {Error} Error de deserialización si el cuerpo no es válido.
 */
func decodeJSON(request *http.Request, target any) error {
	raw, err := io.ReadAll(request.Body)
	if err != nil {
		return fmt.Errorf("no se pudo leer el cuerpo de la petición: %w", err)
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return io.EOF
	}
	if trimmed[0] != '{' {
		return fmt.Errorf("%w: el cuerpo debe ser un objeto JSON", ErrInvalidContent)
	}

	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("no se pudo leer el cuerpo de la petición: %w", err)
	}
	if err := decoder.Decode(new(json.RawMessage)); err != io.EOF {
		return ErrInvalidContent
	}
	return nil
}

/**
 * writeError traduce los errores del dominio a respuestas HTTP.
 *
 * @param {http.ResponseWriter} writer - Respuesta HTTP en construcción.
 * @param {http.Request} request - Petición que originó el error.
 * @param {Error} err - Error producido por el caso de uso.
 */
func writeError(writer http.ResponseWriter, request *http.Request, err error) {
	switch {
	case errors.Is(err, ErrInvalidCompetition):
		httpx.RespondError(writer, request, 409, "convocatoria_no_disponible", ErrInvalidCompetition.Error())
	case errors.Is(err, ErrForbidden):
		httpx.RespondError(writer, request, http.StatusForbidden, "acceso_denegado", "No tiene permiso para acceder a esta postulación.")
	case errors.Is(err, ErrBusinessRule):
		httpx.RespondError(writer, request, http.StatusUnprocessableEntity, "regla_de_negocio", err.Error())
	case errors.Is(err, ErrInvalidSection):
		httpx.RespondError(writer, request, http.StatusBadRequest, "seccion_invalida", ErrInvalidSection.Error())
	case errors.Is(err, ErrNotFound):
		httpx.RespondError(writer, request, http.StatusNotFound, "postulacion_no_encontrada", "La postulación no existe.")
	case errors.Is(err, ErrNotEditable):
		httpx.RespondError(writer, request, http.StatusConflict, "postulacion_no_editable", "La postulación ya fue enviada y no admite cambios.")
	case errors.Is(err, ErrDuplicateCode):
		httpx.RespondError(writer, request, http.StatusConflict, "codigo_duplicado", "Ya existe una postulación con ese código.")
	case errors.Is(err, ErrInvalidContent):
		httpx.RespondError(writer, request, http.StatusBadRequest, "contenido_invalido", "El contenido de la postulación no es válido.")
	case errors.Is(err, ErrInvalidApplicant):
		httpx.RespondError(writer, request, http.StatusBadRequest, "postulante_invalido", "El postulante indicado no es válido.")
	case errors.Is(err, ErrInvalidIdentifier):
		httpx.RespondError(writer, request, http.StatusBadRequest, "identificador_invalido", "El identificador de la postulación no es válido.")
	case errors.Is(err, shareddomain.ErrInvalidCursor):
		httpx.RespondError(writer, request, http.StatusBadRequest, "cursor_invalido", "El cursor de paginación no es válido.")
	default:
		slog.ErrorContext(request.Context(), "error no controlado en postulaciones", "error", err)
		httpx.RespondError(writer, request, http.StatusInternalServerError, "error_interno", "Ocurrió un error al procesar la solicitud.")
	}
}

/** requireOwner comprueba la sesión y la titularidad antes de acceder al recurso. */
func (h *Handler) requireOwner(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor, ok := identity.ApplicantFromContext(r.Context())
		if !ok {
			httpx.RespondError(w, r, 401, "no_autenticado", "Debe iniciar sesión.")
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			writeError(w, r, ErrInvalidIdentifier)
			return
		}
		item, err := h.service.Get(r.Context(), id)
		if err != nil {
			writeError(w, r, err)
			return
		}
		if item.ApplicantID != actor.ID {
			writeError(w, r, ErrForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

/** saveSection guarda campos de una sección y devuelve el avance recalculado. */
func (h *Handler) saveSection(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	n, err := strconv.Atoi(chi.URLParam(r, "n"))
	if err != nil {
		writeError(w, r, ErrInvalidSection)
		return
	}
	var content json.RawMessage
	if err := decodeJSON(r, &content); err != nil {
		httpx.RespondError(w, r, 400, "cuerpo_invalido", "El cuerpo de la petición no es válido.")
		return
	}
	actor, _ := identity.ApplicantFromContext(r.Context())
	if n == 1 && actor != nil && actor.IDCard.String() != "" {
		var sec PersonalSection
		if err := json.Unmarshal(content, &sec); err == nil && sec.IDCard != "" && sec.IDCard != actor.IDCard.String() {
			writeError(w, r, ErrForbidden)
			return
		}
	}
	item, err := h.service.SaveSection(r.Context(), id, n, content)
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpx.Respond(w, r, http.StatusOK, NewResponse(item))
}

/** submit recibe las aceptaciones finales y devuelve el formulario sellado. */
func (h *Handler) submit(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	var body SubmitRequest
	if err := decodeJSON(r, &body); err != nil && !errors.Is(err, io.EOF) {
		httpx.RespondError(w, r, 400, "cuerpo_invalido", "El cuerpo de la petición no es válido.")
		return
	}
	item, err := h.service.Submit(r.Context(), id, body.Consent)
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpx.Respond(w, r, http.StatusOK, NewResponse(item))
}
