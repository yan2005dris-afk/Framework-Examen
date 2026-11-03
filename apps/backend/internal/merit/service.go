package merit

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"time"
)

type Service struct {
	repo    Repository
	auditor audit.Recorder
	now     func() time.Time
}

func NewService(repo Repository, recorder audit.Recorder) *Service {
	return &Service{repo, recorder, time.Now}
}
func (s *Service) CreateRules(ctx context.Context, actor *identity.Applicant, period uuid.UUID, sector string, definition RuleDefinition) (*RuleSet, error) {
	if !identity.HasPermission(actor, identity.PermissionGrades) {
		return nil, ErrForbidden
	}
	if period == uuid.Nil || !validSector(sector) {
		return nil, ErrInvalid
	}
	raw, _ := json.Marshal(definition)
	if len(raw) > 64<<10 {
		return nil, ErrInvalid
	}
	hash, err := definitionHash(definition)
	if err != nil {
		return nil, ErrInvalid
	}
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	r := &RuleSet{ID: id, CompetitionID: period, Sector: sector, Definition: definition, Hash: hash, ActorID: actor.ID, CreatedAt: s.now().UTC()}
	if err = s.repo.CreateRules(ctx, actor, r); err != nil {
		return nil, err
	}
	return r, nil
}
func (s *Service) ApproveRules(ctx context.Context, actor *identity.Applicant, id uuid.UUID, reference string) (*RuleSet, error) {
	if !identity.HasPermission(actor, identity.PermissionPhases) {
		return nil, ErrForbidden
	}
	if !validText(reference, 2000) {
		return nil, ErrInvalid
	}
	return s.repo.ApproveRules(ctx, actor, id, reference)
}
func (s *Service) Review(ctx context.Context, actor *identity.Applicant, id uuid.UUID, input ReviewInput) (*Case, error) {
	if !identity.HasPermission(actor, identity.PermissionGrades) {
		return nil, ErrForbidden
	}
	return s.repo.MutateCase(ctx, actor, id, func(c *Case) error {
		version := int64(0)
		if c.Review != nil {
			version = c.Review.Version
		}
		if input.Version != version {
			return ErrConflict
		}
		if err := ValidateReview(c, input); err != nil {
			return err
		}
		reviewID, err := uuid.NewV7()
		if err != nil {
			return err
		}
		c.Review = &Review{ID: reviewID, SubmissionID: id, SnapshotHash: c.Snapshot.Hash, Version: version + 1, Input: input, ActorID: actor.ID, CreatedAt: s.now().UTC()}
		return nil
	})
}
func (s *Service) Calculate(ctx context.Context, actor *identity.Applicant, id uuid.UUID, input CalculateInput) (*Case, error) {
	if !identity.HasPermission(actor, identity.PermissionGrades) {
		return nil, ErrForbidden
	}
	if !validText(input.Motive, 4000) {
		return nil, ErrInvalid
	}
	return s.repo.MutateCase(ctx, actor, id, func(c *Case) error {
		version := int64(0)
		if c.Calculation != nil {
			version = c.Calculation.Version
		}
		if version != input.Version || c.Review == nil || c.Review.ID != input.ReviewID {
			return ErrConflict
		}
		result, err := Evaluate(c, c.Review)
		if err != nil {
			return err
		}
		calcID, err := uuid.NewV7()
		if err != nil {
			return err
		}
		c.Calculation = &Calculation{EngineVersion: EngineVersion, ID: calcID, SubmissionID: id, Version: version + 1, ReviewID: c.Review.ID, RuleSetID: c.ActiveRules.ID, RulesHash: c.ActiveRules.Hash, SnapshotHash: c.Snapshot.Hash, Evaluation: result, Motive: input.Motive, ActorID: actor.ID, CreatedAt: s.now().UTC()}
		return nil
	})
}
func (s *Service) Publish(ctx context.Context, actor *identity.Applicant, id uuid.UUID, input PublishInput) (*Case, error) {
	if !identity.HasPermission(actor, identity.PermissionPhases) {
		return nil, ErrForbidden
	}
	if !validText(input.Motive, 4000) || (input.Decision != "avanza" && input.Decision != "excluido") {
		return nil, ErrInvalid
	}
	return s.repo.MutateCase(ctx, actor, id, func(c *Case) error {
		version := int64(0)
		if c.Publication != nil {
			version = c.Publication.Version
			if input.SupersedesID == nil || *input.SupersedesID != c.Publication.ID {
				return ErrConflict
			}
		} else if input.SupersedesID != nil {
			return ErrInvalid
		}
		if input.Version != version || c.Calculation == nil || c.Calculation.SnapshotHash != c.Snapshot.Hash || c.Calculation.EngineVersion != EngineVersion || c.Calculation.ID != input.CalculationID || c.Review == nil || c.Calculation.ReviewID != c.Review.ID || c.ActiveRules == nil || c.Calculation.RuleSetID != c.ActiveRules.ID || c.Calculation.RulesHash != c.ActiveRules.Hash {
			return ErrConflict
		}
		evaluation, err := Evaluate(c, c.Review)
		if err != nil {
			return err
		}
		before, _ := json.Marshal(c.Calculation.Evaluation)
		after, _ := json.Marshal(evaluation)
		if string(before) != string(after) {
			return ErrConflict
		}
		if input.Decision == "avanza" && !evaluation.CanAdvance {
			return ErrEvidence
		}
		pubID, err := uuid.NewV7()
		if err != nil {
			return err
		}
		c.Publication = &Publication{ID: pubID, SubmissionID: id, Version: version + 1, CalculationID: c.Calculation.ID, SupersedesID: input.SupersedesID, Decision: input.Decision, Motive: input.Motive, Evaluation: evaluation, SnapshotHash: c.Snapshot.Hash, RuleSetID: c.ActiveRules.ID, RulesHash: c.ActiveRules.Hash, ActorID: actor.ID, CreatedAt: s.now().UTC()}
		return nil
	})
}
func (s *Service) Read(ctx context.Context, actor *identity.Applicant, id uuid.UUID, admin bool) (c *Case, err error) {
	if actor == nil || actor.ID == uuid.Nil || admin && !identity.HasPermission(actor, identity.PermissionGrades) {
		return nil, ErrForbidden
	}
	if admin {
		defer func() {
			err = audit.Record(ctx, s.auditor, audit.Event{ActorID: actor.ID, Role: actor.Role, Action: "meritos.consultados", ResourceID: id, ResourceType: "postulacion", CreatedAt: s.now().UTC()}, err)
			if err != nil {
				c = nil
			}
		}()
	}
	c, err = s.repo.ReadCase(ctx, id)
	if err != nil {
		return nil, err
	}
	if !admin && c.Snapshot.ApplicantID != actor.ID {
		return nil, ErrForbidden
	}
	return c, nil
}
