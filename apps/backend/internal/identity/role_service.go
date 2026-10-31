package identity

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	ErrRoleForbidden     = errors.New("su rol no permite administrar permisos")
	ErrInvalidRoleChange = errors.New("indique un rol válido y un motivo de hasta 500 caracteres")
)

/** RoleRepository confirma el cambio y su auditoría dentro de una transacción. */
type RoleRepository interface {
	ChangeRole(context.Context, uuid.UUID, uuid.UUID, string, string) (*Applicant, error)
}

/** RoleService administra roles sin introducirlos en el registro público. */
type RoleService struct{ repo RoleRepository }

/** NewRoleService recibe la persistencia transaccional de permisos. */
func NewRoleService(repo RoleRepository) *RoleService { return &RoleService{repo: repo} }

/** ValidRoleChange comprueba la entrada tanto de HTTP como de la herramienta operativa. */
func ValidRoleChange(role, reason string) bool {
	return ValidRole(role) && strings.TrimSpace(reason) != "" && utf8.ValidString(reason) &&
		utf8.RuneCountInString(reason) <= 500 && strings.IndexFunc(reason, unicode.IsControl) < 0
}

/** Change exige administrador y delega la comprobación vigente y auditoría al repositorio. */
func (s *RoleService) Change(ctx context.Context, actor *Applicant, target uuid.UUID, role, reason string) (*Applicant, error) {
	if !HasPermission(actor, PermissionRoles) {
		return nil, ErrRoleForbidden
	}
	if target == uuid.Nil || !ValidRoleChange(role, reason) {
		return nil, ErrInvalidRoleChange
	}
	return s.repo.ChangeRole(ctx, actor.ID, target, role, strings.TrimSpace(reason))
}
