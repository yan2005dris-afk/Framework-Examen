package document

import (
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/httpx"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"io"
	"log/slog"
	"net/http"
	"strconv"
)

/** Handler recibe multipart acotado y responde con el envelope de la API. */
type Handler struct{ service *Service }

/** NewHandler conecta los casos de uso con HTTP.
 * @returns {Handler} Adaptador para registrar sobre un router autenticado.
 */
func NewHandler(service *Service) *Handler { return &Handler{service: service} }

/** RegisterRoutes registra endpoints propios sin sobrescribir el subrouter de postulaciones.
 * @param {chi.Router} router - Grupo protegido por RequireSession.
 */
func (h *Handler) RegisterRoutes(router chi.Router) {
	router.Post("/api/v1/submissions/{id}/documents", h.upload)
	router.Get("/api/v1/submissions/{id}/documents", h.list)
	router.Delete("/api/v1/documents/{id}", h.delete)
	router.Get("/api/v1/documents/{id}/download", h.download)
}

func requestIdentity(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	actor, ok := identity.ApplicantFromContext(r.Context())
	if !ok {
		httpx.RespondError(w, r, 401, "no_autenticado", "Debe iniciar sesión.")
		return uuid.Nil, uuid.Nil, false
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.RespondError(w, r, 400, "identificador_invalido", "El identificador no es válido.")
		return uuid.Nil, uuid.Nil, false
	}
	return actor.ID, id, true
}

func (h *Handler) upload(w http.ResponseWriter, r *http.Request) {
	actor, id, ok := requestIdentity(w, r)
	if !ok {
		return
	}
	if _, err := h.service.Authorize(r.Context(), actor, id, true); err != nil {
		writeError(w, r, err)
		return
	}
	// Un único archivo más margen para campos y cabeceras; también cubre cuerpos chunked.
	r.Body = http.MaxBytesReader(w, r.Body, MaxOrganizationBytes+64*1024)
	if err := r.ParseMultipartForm(MaxOrganizationBytes + 64*1024); err != nil {
		if r.MultipartForm != nil {
			defer r.MultipartForm.RemoveAll()
		}
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeError(w, r, err)
		} else {
			httpx.RespondError(w, r, 400, "cuerpo_invalido", "El cuerpo multipart no es válido.")
		}
		return
	}
	defer r.MultipartForm.RemoveAll()
	for key, values := range r.MultipartForm.Value {
		if (key != "section" && key != "recordId") || len(values) != 1 {
			httpx.RespondError(w, r, 400, "cuerpo_invalido", "Los campos multipart no son válidos.")
			return
		}
	}
	if len(r.MultipartForm.File) != 1 || len(r.MultipartForm.File["file"]) != 1 {
		httpx.RespondError(w, r, 400, "cuerpo_invalido", "Debe adjuntar un único archivo en el campo file.")
		return
	}
	sectionValues := r.MultipartForm.Value["section"]
	if len(sectionValues) != 1 {
		writeError(w, r, ErrInvalidSection)
		return
	}
	section, err := strconv.Atoi(sectionValues[0])
	if err != nil {
		writeError(w, r, ErrInvalidSection)
		return
	}
	limit, err := LimitForSection(section)
	if err != nil {
		writeError(w, r, err)
		return
	}
	header := r.MultipartForm.File["file"][0]
	if header.Size > limit {
		writeError(w, r, ErrFileTooLarge)
		return
	}
	file, err := header.Open()
	if err != nil {
		writeError(w, r, err)
		return
	}
	defer file.Close()
	recordID := ""
	if values := r.MultipartForm.Value["recordId"]; len(values) == 1 {
		recordID = values[0]
	}
	d, err := h.service.Upload(r.Context(), UploadInput{ApplicantID: actor, SubmissionID: id, Section: section, RecordID: recordID, Name: header.Filename, MIME: header.Header.Get("Content-Type"), Content: file})
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpx.Respond(w, r, http.StatusCreated, d)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	actor, id, ok := requestIdentity(w, r)
	if !ok {
		return
	}
	docs, err := h.service.List(r.Context(), actor, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.Respond(w, r, 200, docs)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	actor, id, ok := requestIdentity(w, r)
	if !ok {
		return
	}
	if err := h.service.Delete(r.Context(), actor, id); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) download(w http.ResponseWriter, r *http.Request) {
	actor, id, ok := requestIdentity(w, r)
	if !ok {
		return
	}
	result, err := h.service.Download(r.Context(), actor, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.Respond(w, r, 200, result)
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	var maxErr *http.MaxBytesError
	switch {
	case errors.Is(err, submission.ErrNotFound):
		httpx.RespondError(w, r, 404, "postulacion_no_encontrada", submission.ErrNotFound.Error())
	case errors.Is(err, submission.ErrForbidden):
		httpx.RespondError(w, r, 403, "acceso_denegado", "La postulación pertenece a otro postulante.")
	case errors.Is(err, submission.ErrNotEditable):
		httpx.RespondError(w, r, 409, "postulacion_no_editable", submission.ErrNotEditable.Error())
	case errors.Is(err, ErrNotFound):
		httpx.RespondError(w, r, 404, "documento_no_encontrado", ErrNotFound.Error())
	case errors.Is(err, ErrDuplicate):
		httpx.RespondError(w, r, 409, "documento_duplicado", ErrDuplicate.Error())
	case errors.Is(err, ErrUnsupportedType):
		httpx.RespondError(w, r, 415, "tipo_no_soportado", ErrUnsupportedType.Error())
	case errors.As(err, &maxErr), errors.Is(err, ErrFileTooLarge):
		httpx.RespondError(w, r, 413, "archivo_muy_grande", ErrFileTooLarge.Error())
	case errors.Is(err, ErrQuotaExceeded):
		httpx.RespondError(w, r, 413, "cuota_documentos_excedida", ErrQuotaExceeded.Error())
	case errors.Is(err, ErrInvalidSection):
		httpx.RespondError(w, r, 400, "seccion_invalida", ErrInvalidSection.Error())
	case errors.Is(err, ErrInvalidRecord):
		httpx.RespondError(w, r, 422, "registro_invalido", ErrInvalidRecord.Error())
	case errors.Is(err, ErrInvalidName):
		httpx.RespondError(w, r, 400, "nombre_invalido", ErrInvalidName.Error())
	case errors.Is(err, io.EOF), errors.Is(err, io.ErrUnexpectedEOF), errors.Is(err, http.ErrNotMultipart):
		httpx.RespondError(w, r, 400, "cuerpo_invalido", "El cuerpo multipart no es válido.")
	default:
		slog.Error("falló una operación de documentos")
		httpx.RespondError(w, r, 500, "error_interno", "No se pudo procesar el documento.")
	}
}
