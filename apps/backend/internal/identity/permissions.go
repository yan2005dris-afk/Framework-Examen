package identity

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/httpx"
)

/** Permission identifica operaciones administrativas; lo desconocido se deniega. */
type Permission string

const (
	PermissionReview           Permission = "revision.consultar"
	PermissionDocuments        Permission = "documentos.consultar"
	PermissionPhases           Permission = "fases.gestionar"
	PermissionQuestions        Permission = "preguntas.gestionar"
	PermissionGrades           Permission = "calificaciones.gestionar"
	PermissionReconsiderations Permission = "reconsideraciones.gestionar"
	PermissionRoles            Permission = "roles.gestionar"
)

/** RolesForPermission define una política explícita, sin privilegios implícitos. */
func RolesForPermission(permission Permission) []string {
	switch permission {
	case PermissionReview, PermissionDocuments, PermissionGrades, PermissionReconsiderations:
		return []string{RoleAnalyst, RoleAdministrator}
	case PermissionPhases, PermissionQuestions, PermissionRoles:
		return []string{RoleAdministrator}
	default:
		return nil
	}
}

/** HasPermission aplica la misma política en los casos de uso y en HTTP. */
func HasPermission(actor *Applicant, permission Permission) bool {
	if actor == nil || actor.ID == uuid.Nil {
		return false
	}
	for _, role := range RolesForPermission(permission) {
		if actor.Role == role {
			return true
		}
	}
	return false
}

/** RequireRole exige sesión resuelta y uno de los roles explícitamente permitidos.
 * @param {string[]} roles - Lista cerrada; una lista vacía deniega el acceso.
 * @returns {func(http.Handler) http.Handler} Middleware con errores JSON 401/403.
 */
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := append([]string(nil), roles...)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actor, ok := ApplicantFromContext(r.Context())
			if !ok || actor == nil || actor.ID == uuid.Nil {
				httpx.RespondError(w, r, 401, "no_autenticado", "Debe iniciar sesión.")
				return
			}
			for _, role := range allowed {
				if ValidRole(role) && actor.Role == role {
					next.ServeHTTP(w, r)
					return
				}
			}
			httpx.RespondError(w, r, 403, "acceso_denegado", "Su rol no permite esta operación.")
		})
	}
}

/** RequirePermission convierte la política de una operación en middleware reutilizable. */
func RequirePermission(permission Permission) func(http.Handler) http.Handler {
	return RequireRole(RolesForPermission(permission)...)
}

/** ProtectAdministration protege incluso rutas de módulos que se incorporen después.
 * @returns {http.Handler} Acceso restringido por operación bajo /api/v1/admin/.
 */
func ProtectAdministration(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/v1/admin/") && r.URL.Path != "/api/v1/admin" {
			next.ServeHTTP(w, r)
			return
		}
		segment := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/admin/"), "/")[0]
		permission := map[string]Permission{
			"submissions": PermissionReview, "documents": PermissionDocuments,
			"phases": PermissionPhases, "question-bank": PermissionQuestions,
			"grades": PermissionGrades, "reconsiderations": PermissionReconsiderations,
			"applicants": PermissionRoles,
		}[segment]
		RequirePermission(permission)(next).ServeHTTP(w, r)
	})
}
