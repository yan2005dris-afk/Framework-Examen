package evaluation

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/document"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"slices"
	"time"
)

type Service struct {
	repo    Repository
	storage document.Storage
	auditor audit.Recorder
}

func NewService(repo Repository, storage document.Storage, recorder audit.Recorder) *Service {
	return &Service{repo, storage, recorder}
}
func newID() uuid.UUID { id, _ := uuid.NewV7(); return id }
func permitted(a *identity.Applicant, p identity.Permission) error {
	if !identity.HasPermission(a, p) {
		return ErrForbidden
	}
	return nil
}
func admin(a *identity.Applicant) error { return permitted(a, identity.PermissionPhases) }
func command(key uuid.UUID, operation string, id uuid.UUID, input interface{}) (Command, error) {
	if key == uuid.Nil {
		return Command{}, ErrInvalid
	}
	hash, err := Digest(input)
	if err != nil {
		return Command{}, ErrInvalid
	}
	return Command{Key: key, Operation: operation, ResourceID: id, Hash: hash}, nil
}
func makeChange(id uuid.UUID, action, phase string, v interface{}, notify bool) (Change, error) {
	raw, err := json.Marshal(v)
	return Change{ID: id, Action: action, Phase: phase, Record: raw, Notify: notify}, err
}
func (s *Service) CreatePolicy(ctx context.Context, a *identity.Applicant, period uuid.UUID, sector, kind string, def Definition) (*Policy, error) {
	if err := permitted(a, identity.PermissionGrades); err != nil {
		return nil, err
	}
	if period == uuid.Nil || !slices.Contains([]string{"afiliados", "jubilados", "empleadores"}, sector) || !slices.Contains([]string{"oposicion", "reconsideracion"}, kind) {
		return nil, ErrInvalid
	}
	hash, err := Digest(def)
	if err != nil {
		return nil, ErrInvalid
	}
	p := &Policy{ID: newID(), CompetitionID: period, Sector: sector, Kind: kind, Definition: def, Hash: hash, ActorID: a.ID, CreatedAt: time.Now().UTC()}
	if p.ID == uuid.Nil {
		return nil, ErrUnavailable
	}
	raw, _ := json.Marshal(def)
	if len(raw) > 128<<10 {
		return nil, ErrInvalid
	}
	if err = s.repo.CreatePolicy(ctx, a, p); err != nil {
		return nil, err
	}
	return p, nil
}
func (s *Service) ApprovePolicy(ctx context.Context, a *identity.Applicant, id uuid.UUID, reference string) (*Policy, error) {
	if err := admin(a); err != nil {
		return nil, err
	}
	if !validText(reference, 2000) {
		return nil, ErrInvalid
	}
	return s.repo.ApprovePolicy(ctx, a, id, reference)
}
func approved(p *Policy) bool { return p != nil && p.Approved && p.Validate() == nil }
func (s *Service) RecordPsychometric(ctx context.Context, a *identity.Applicant, id, key uuid.UUID, input PsychometricInput) (Receipt, error) {
	if err := permitted(a, identity.PermissionGrades); err != nil {
		return Receipt{}, err
	}
	cmd, err := command(key, "psicotecnica.registrada", id, input)
	if err != nil {
		return Receipt{}, err
	}
	return s.withEvidence(ctx, a, id, cmd, &input.Evidence, false, func(c *Case, e *Evidence) (Change, error) {
		if err := phaseWindow(c, "oposicion_psicotecnica"); err != nil {
			return Change{}, err
		}
		p := c.OppositionPolicy
		if !approved(p) {
			return Change{}, ErrPolicy
		}
		if input.PolicyID != p.ID {
			return Change{}, ErrConflict
		}
		r := p.Definition.Opposition
		version := int64(0)
		if c.State.Psychometric != nil {
			version = c.State.Psychometric.Version
		}
		if input.PreviousVersion != version {
			return Change{}, ErrConflict
		}
		if !validText(input.Motive, 4000) || input.EvaluatedAt.Before(*c.Snapshot.SubmittedAt) || input.EvaluatedAt.After(c.Now) || !slices.Contains(r.Conclusions, input.Conclusion) {
			return Change{}, ErrInvalid
		}
		if r.PsychometricMode == "puntaje" {
			if input.Score == nil || *input.Score < 0 || *input.Score > *r.PsychometricMaximum {
				return Change{}, ErrInvalid
			}
		} else if input.Score != nil {
			return Change{}, ErrInvalid
		}
		record := &Psychometric{ID: newID(), SubmissionID: id, Version: version + 1, PolicyID: p.ID, PolicyHash: p.Hash, SnapshotHash: c.Snapshot.Hash, Instrument: r.Instrument, InstrumentVersion: r.InstrumentVersion, Score: input.Score, Maximum: r.PsychometricMaximum, Conclusion: input.Conclusion, EvaluatedAt: input.EvaluatedAt, Evidence: *e, Motive: input.Motive, ActorID: a.ID, CreatedAt: c.Now}
		c.State.Psychometric = record
		change, err := makeChange(record.ID, "psicotecnica.registrada", "oposicion_psicotecnica", record, false)
		change.Evidence = e
		return change, err
	})
}
func validatePublication(c *Case, phase string, input PublishInput) error {
	old, exists := c.State.Results[phase]
	version := int64(0)
	if exists {
		version = old.Version
		if input.SupersedesID == nil || *input.SupersedesID != old.ID {
			return ErrConflict
		}
	} else if input.SupersedesID != nil {
		return ErrInvalid
	}
	if input.PreviousVersion != version || !validText(input.Motive, 4000) {
		return ErrConflict
	}
	return nil
}
func (s *Service) PublishPsychometric(ctx context.Context, a *identity.Applicant, id, key uuid.UUID, input PublishInput) (Receipt, error) {
	if err := admin(a); err != nil {
		return Receipt{}, err
	}
	cmd, err := command(key, "psicotecnica.publicada", id, input)
	if err != nil {
		return Receipt{}, err
	}
	return s.repo.Execute(ctx, a, id, cmd, func(c *Case) (Change, error) {
		if err := phaseWindow(c, "oposicion_psicotecnica"); err != nil {
			return Change{}, err
		}
		p := c.OppositionPolicy
		record := c.State.Psychometric
		if !approved(p) {
			return Change{}, ErrPolicy
		}
		if record == nil || record.ID != input.SourceID || input.PolicyID != p.ID || record.PolicyID != p.ID || record.PolicyHash != p.Hash {
			return Change{}, ErrConflict
		}
		if err := validatePublication(c, "oposicion_psicotecnica", input); err != nil {
			return Change{}, err
		}
		value := Result{ID: newID(), SubmissionID: id, Phase: "oposicion_psicotecnica", Version: input.PreviousVersion + 1, SupersedesID: input.SupersedesID, SourceID: record.ID, PolicyID: p.ID, PolicyHash: p.Hash, SnapshotHash: c.Snapshot.Hash, Score: record.Score, Maximum: record.Maximum, CanAdvance: p.Definition.Opposition.AcceptsPsychometric(record.Score, record.Conclusion), Motive: input.Motive, ActorID: a.ID, PublishedAt: c.Now}
		c.State.Results[value.Phase] = value
		return makeChange(value.ID, "psicotecnica.publicada", value.Phase, value, true)
	})
}
func (s *Service) Consolidate(ctx context.Context, a *identity.Applicant, id, key uuid.UUID, input PublishInput) (Receipt, error) {
	if err := admin(a); err != nil {
		return Receipt{}, err
	}
	cmd, err := command(key, "expediente.consolidado", id, input)
	if err != nil {
		return Receipt{}, err
	}
	return s.repo.Execute(ctx, a, id, cmd, func(c *Case) (Change, error) {
		if err := phaseWindow(c, "resultados"); err != nil {
			return Change{}, err
		}
		p := c.OppositionPolicy
		if !approved(p) {
			return Change{}, ErrPolicy
		}
		if input.PolicyID != p.ID || input.SourceID != uuid.Nil {
			return Change{}, ErrConflict
		}
		if err := validatePublication(c, "resultados", input); err != nil {
			return Change{}, err
		}
		if len(input.ExpectedSources) != 3 {
			return Change{}, ErrInvalid
		}
		for _, phase := range []string{"meritos", "oposicion_teorica", "oposicion_psicotecnica"} {
			if c.Sources[phase].ID == uuid.Nil {
				return Change{}, ErrPending
			}
			if input.ExpectedSources[phase] != c.Sources[phase].ID {
				return Change{}, ErrConflict
			}
		}
		parts, total, advance, err := Consolidate(c, *p.Definition.Opposition)
		if err != nil {
			return Change{}, err
		}
		maximum := p.Definition.Opposition.TotalMaximum
		result := Result{ID: newID(), SubmissionID: id, Phase: "resultados", Version: input.PreviousVersion + 1, SupersedesID: input.SupersedesID, PolicyID: p.ID, PolicyHash: p.Hash, SnapshotHash: c.Snapshot.Hash, Score: &total, Maximum: &maximum, CanAdvance: advance, Motive: input.Motive, ActorID: a.ID, PublishedAt: c.Now, EngineVersion: EngineVersion, Breakdown: parts}
		c.State.Results[result.Phase] = result
		return makeChange(result.ID, "expediente.consolidado", result.Phase, result, true)
	})
}

/** Read no expone registros psicotécnicos sin publicar al titular ni permite leer expedientes ajenos. */
func (s *Service) Read(ctx context.Context, a *identity.Applicant, id, appealID uuid.UUID, administrative bool) (c *Case, err error) {
	if a == nil || a.ID == uuid.Nil {
		return nil, ErrForbidden
	}
	if administrative {
		if err = permitted(a, identity.PermissionReview); err != nil {
			return nil, err
		}
		defer func() {
			err = audit.Record(ctx, s.auditor, audit.Event{ActorID: a.ID, Role: a.Role, SubmissionID: id, Action: "evaluacion.consultada", ResourceID: id, ResourceType: "postulacion", CreatedAt: time.Now().UTC()}, err)
			if err != nil {
				c = nil
			}
		}()
	}
	c, err = s.repo.ReadCase(ctx, id, appealID)
	if err != nil {
		return nil, err
	}
	if !administrative {
		if c.Snapshot.ApplicantID != a.ID {
			return nil, ErrForbidden
		}
		c.State.Psychometric = nil
		c.OppositionPolicy = nil
		c.AppealPolicy = nil
	}
	return c, nil
}
