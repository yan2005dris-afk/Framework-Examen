package competition

import (
	"context"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
)

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }
func (s *Service) Create(ctx context.Context, actor *identity.Applicant, c *Competition) error {
	if !identity.HasPermission(actor, identity.PermissionPhases) {
		return ErrForbidden
	}
	if c.Approved || c.Published || c.FormEnabled || c.Version != 0 {
		return ErrInvalid
	}
	if err := c.Validate(); err != nil {
		return err
	}
	return s.repo.Create(ctx, c, actor.ID)
}
func (s *Service) Configure(ctx context.Context, actor *identity.Applicant, id uuid.UUID, config Configuration) (*Competition, error) {
	if !identity.HasPermission(actor, identity.PermissionPhases) {
		return nil, ErrForbidden
	}
	if config.Version < 1 {
		return nil, ErrInvalid
	}
	return s.repo.Configure(ctx, id, config, actor.ID)
}
