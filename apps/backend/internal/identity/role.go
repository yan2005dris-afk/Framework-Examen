package identity

/** Los roles se asignan en el servidor; el registro público siempre crea postulantes. */
const (
	RoleApplicant     = "postulante"
	RoleAnalyst       = "analista"
	RoleAdministrator = "administrador"
)

/** ValidRole limita los roles a los actores definidos por el concurso. */
func ValidRole(role string) bool {
	return role == RoleApplicant || role == RoleAnalyst || role == RoleAdministrator
}
