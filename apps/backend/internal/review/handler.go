package review

import (
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/httpx"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/shared/domain"
	"net/http"
)

/** Handler expone sólo lecturas administrativas y exportaciones privadas de EF-13. */
type Handler struct{ service *Service }

/** NewHandler recibe los casos de uso del contexto de revisión. */
func NewHandler(service *Service) *Handler { return &Handler{service: service} }

/** RegisterRoutes utiliza el prefijo administrativo ya protegido por T6. */
func (h *Handler) RegisterRoutes(router chi.Router) {
	router.With(identity.RequirePermission(identity.PermissionReview)).Get("/api/v1/admin/competitions", h.competitions)
	router.With(identity.RequirePermission(identity.PermissionReview)).Get("/api/v1/admin/submissions", h.list)
	router.With(identity.RequirePermission(identity.PermissionReview)).Get("/api/v1/admin/submissions/stats", h.stats)
	router.With(identity.RequirePermission(identity.PermissionReview)).Get("/api/v1/admin/submissions/export.{format}", h.export)
}

func filters(r *http.Request, paginated bool) (Filter, error) {
	keys := []string{"convocatoriaId", "sinConvocatoria", "sector", "status"}
	if paginated {
		keys = append(keys, "limit", "cursor")
	}
	if err := httpx.ValidateQuery(r, keys...); err != nil {
		return Filter{}, ErrInvalidFilter
	}
	q := r.URL.Query()
	f := Filter{Sector: q.Get("sector"), Status: q.Get("status")}
	if raw := q.Get("convocatoriaId"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return f, ErrInvalidFilter
		}
		f.CompetitionID = &id
	}
	if raw := q.Get("sinConvocatoria"); raw != "" {
		if raw != "true" {
			return f, ErrInvalidFilter
		}
		f.Legacy = true
	}
	return f, ValidateFilter(f)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	f, err := filters(r, true)
	if err != nil {
		writeError(w, r, err)
		return
	}
	q, err := httpx.ParseCursorRequest(r)
	if err != nil {
		writeError(w, r, domain.ErrInvalidCursor)
		return
	}
	actor, _ := identity.ApplicantFromContext(r.Context())
	result, err := h.service.List(r.Context(), actor, f, q)
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpx.Respond(w, r, 200, result)
}
func (h *Handler) stats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	f, err := filters(r, false)
	if err != nil {
		writeError(w, r, err)
		return
	}
	actor, _ := identity.ApplicantFromContext(r.Context())
	result, err := h.service.Counts(r.Context(), actor, f)
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpx.Respond(w, r, 200, result)
}
func (h *Handler) competitions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if err := httpx.ValidateQuery(r, "limit", "cursor"); err != nil {
		writeError(w, r, ErrInvalidFilter)
		return
	}
	q, err := httpx.ParseCursorRequest(r)
	if err != nil {
		writeError(w, r, domain.ErrInvalidCursor)
		return
	}
	actor, _ := identity.ApplicantFromContext(r.Context())
	result, err := h.service.Competitions(r.Context(), actor, q)
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpx.Respond(w, r, 200, result)
}
func (h *Handler) export(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	f, err := filters(r, false)
	if err != nil {
		writeError(w, r, err)
		return
	}
	format := chi.URLParam(r, "format")
	if format != "xlsx" && format != "pdf" {
		writeError(w, r, ErrInvalidFilter)
		return
	}
	actor, _ := identity.ApplicantFromContext(r.Context())
	data, err := h.service.Export(r.Context(), actor, f, format)
	if err != nil {
		writeError(w, r, err)
		return
	}
	contentType := "application/pdf"
	if format == "xlsx" {
		contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", `attachment; filename="matriz-ef13.`+format+`"`)
	w.WriteHeader(200)
	_, _ = w.Write(data)
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrForbidden):
		httpx.RespondError(w, r, 403, "acceso_denegado", ErrForbidden.Error())
	case errors.Is(err, ErrInvalidFilter):
		httpx.RespondError(w, r, 400, "filtro_revision_invalido", ErrInvalidFilter.Error())
	case errors.Is(err, domain.ErrInvalidCursor):
		httpx.RespondError(w, r, 400, "cursor_invalido", "La paginación no es válida para estos filtros.")
	case errors.Is(err, ErrCompetitionNotFound):
		httpx.RespondError(w, r, 404, "convocatoria_no_encontrada", ErrCompetitionNotFound.Error())
	case errors.Is(err, ErrExportTooLarge):
		httpx.RespondError(w, r, 413, "exportacion_demasiado_grande", ErrExportTooLarge.Error())
	case errors.Is(err, ErrSnapshot):
		httpx.RespondError(w, r, 409, "expediente_no_integro", ErrSnapshot.Error())
	default:
		httpx.RespondError(w, r, 500, "revision_no_disponible", "No se pudo completar la consulta de revisión.")
	}
}
