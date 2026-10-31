package identity

import (
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/httpx"
	"io"
	"net/http"
)

/** RegisterRoleRoutes publica gestión de roles exclusivamente para administradores autenticados. */
func RegisterRoleRoutes(router chi.Router, service *RoleService) {
	router.With(RequirePermission(PermissionRoles)).Patch("/api/v1/admin/applicants/{id}/role", func(w http.ResponseWriter, r *http.Request) {
		target, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			httpx.RespondError(w, r, 400, "identificador_invalido", "El identificador no es válido.")
			return
		}
		var input struct {
			Role   string `json:"rol"`
			Reason string `json:"motivo"`
		}
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil {
			httpx.RespondError(w, r, 400, "cuerpo_invalido", "El cuerpo de la petición no es válido.")
			return
		}
		var extra json.RawMessage
		if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
			httpx.RespondError(w, r, 400, "cuerpo_invalido", "El cuerpo de la petición no es válido.")
			return
		}
		actor, _ := ApplicantFromContext(r.Context())
		applicant, err := service.Change(r.Context(), actor, target, input.Role, input.Reason)
		switch {
		case errors.Is(err, ErrRoleForbidden):
			httpx.RespondError(w, r, 403, "acceso_denegado", ErrRoleForbidden.Error())
		case errors.Is(err, ErrInvalidRoleChange):
			httpx.RespondError(w, r, 422, "cambio_rol_invalido", ErrInvalidRoleChange.Error())
		case errors.Is(err, ErrApplicantNotFound):
			httpx.RespondError(w, r, 404, "postulante_no_encontrado", "La cuenta no existe.")
		case err != nil:
			httpx.RespondError(w, r, 500, "error_interno", "No se pudo actualizar el rol.")
		default:
			w.Header().Set("Cache-Control", "no-store")
			httpx.Respond(w, r, 200, ToResponse(applicant))
		}
	})
}
