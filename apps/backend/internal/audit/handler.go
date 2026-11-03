package audit

import (
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/httpx"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/shared/domain"
	"net/http"
)

/** RegisterHistoryRoutes monta consultas propias y administrativas separadas. */
func RegisterHistoryRoutes(router chi.Router, service *HistoryService) {
	router.Get("/api/v1/submissions/{id}/events", historyHandler(service, false))
	router.With(identity.RequirePermission(identity.PermissionReview)).Get("/api/v1/admin/submissions/{id}/events", historyHandler(service, true))
}

func historyHandler(service *HistoryService, admin bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		actor, ok := identity.ApplicantFromContext(r.Context())
		if !ok {
			httpx.RespondError(w, r, 401, "no_autenticado", "Debe iniciar sesión.")
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil || id == uuid.Nil {
			httpx.RespondError(w, r, 400, "identificador_invalido", "El identificador no es válido.")
			return
		}
		if err := httpx.ValidateQuery(r, "limit", "cursor"); err != nil {
			httpx.RespondError(w, r, 400, "paginacion_invalida", "Los parámetros de historial no son válidos.")
			return
		}
		request, err := httpx.ParseCursorRequest(r)
		if err != nil {
			httpx.RespondError(w, r, 400, "cursor_invalido", "La paginación no es válida.")
			return
		}
		page, err := service.List(r.Context(), actor, id, request, admin)
		switch {
		case errors.Is(err, ErrForbidden):
			httpx.RespondError(w, r, 403, "acceso_denegado", ErrForbidden.Error())
		case errors.Is(err, ErrNotFound):
			httpx.RespondError(w, r, 404, "postulacion_no_encontrada", ErrNotFound.Error())
		case err != nil:
			// Un cursor bien codificado también puede pertenecer a otro expediente.
			if errors.Is(err, domain.ErrInvalidCursor) {
				httpx.RespondError(w, r, 400, "cursor_invalido", "La paginación no es válida.")
			} else {
				httpx.RespondError(w, r, 500, "historial_no_disponible", "No se pudo consultar el historial.")
			}
		default:
			httpx.Respond(w, r, 200, page)
		}
	}
}
