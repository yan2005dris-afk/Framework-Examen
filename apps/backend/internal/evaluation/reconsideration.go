package evaluation

import (
	"context"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
)

/** FileAppeal vincula el reclamo al resultado publicado exacto, al calendario aprobado y al sello del titular. */
func (s *Service) FileAppeal(ctx context.Context, a *identity.Applicant, id, key uuid.UUID, input FileAppealInput) (Receipt, error) {
	if a == nil || a.Role != identity.RoleApplicant {
		return Receipt{}, ErrForbidden
	}
	cmd, err := command(key, "reconsideracion.presentada", id, input)
	if err != nil {
		return Receipt{}, err
	}
	return s.withEvidence(ctx, a, id, cmd, input.Evidence, true, func(c *Case, e *Evidence) (Change, error) {
		if c.Snapshot.ApplicantID != a.ID {
			return Change{}, ErrForbidden
		}
		if err := phaseWindow(c, "reconsideracion"); err != nil {
			return Change{}, err
		}
		p := c.AppealPolicy
		if !approved(p) {
			return Change{}, ErrPolicy
		}
		source, ok := c.Sources[input.Phase]
		if !ok || source.ID != input.ResultID {
			return Change{}, ErrConflict
		}
		if !validText(input.Motive, 8000) || len(input.DocumentIDs) > 30 {
			return Change{}, ErrInvalid
		}
		seen := map[uuid.UUID]bool{}
		for _, id := range input.DocumentIDs {
			if _, ok := c.Documents[id]; !ok || seen[id] {
				return Change{}, ErrInvalid
			}
			seen[id] = true
		}
		anchor := source.PublishedAt
		if c.PhaseStarts["reconsideracion"].After(anchor) {
			anchor = c.PhaseStarts["reconsideracion"]
		}
		rules := p.Definition.Reconsideration
		end := c.PhaseEnds["reconsideracion"]
		filing, err := boundedDeadline(anchor, rules.Applicant, rules.Calendar, end)
		if err != nil {
			return Change{}, err
		}
		if c.Now.Before(anchor) || !c.Now.Before(filing) {
			return Change{}, ErrDeadline
		}
		talent, err := boundedDeadline(c.Now, rules.Talent, rules.Calendar, end)
		if err != nil {
			return Change{}, err
		}
		appeal := &Appeal{ID: newID(), SubmissionID: id, Version: 1, Status: "recibida", Original: source, Policy: *p, Motive: input.Motive, Evidence: e, DocumentIDs: input.DocumentIDs, ApplicantID: a.ID, FiledAt: c.Now, WindowEnd: end, FilingDeadline: filing, TalentDeadline: talent}
		change, err := makeChange(appeal.ID, "reconsideracion.presentada", input.Phase, appeal, false)
		change.Appeal = appeal
		change.Evidence = e
		return change, err
	})
}
func (s *Service) Attend(ctx context.Context, a *identity.Applicant, id, appealID, key uuid.UUID, input AttendInput) (Receipt, error) {
	if a == nil || a.Role != identity.RoleAnalyst {
		return Receipt{}, ErrForbidden
	}
	cmd, err := command(key, "reconsideracion.atendida", appealID, input)
	if err != nil {
		return Receipt{}, err
	}
	cmd.AppealID = appealID
	return s.repo.Execute(ctx, a, id, cmd, func(c *Case) (Change, error) {
		appeal := c.Appeal
		if appeal == nil {
			return Change{}, ErrNotFound
		}
		if appeal.Status != "recibida" {
			return Change{}, ErrState
		}
		if appeal.Version != input.PreviousVersion {
			return Change{}, ErrConflict
		}
		if err := phaseWindow(c, "reconsideracion"); err != nil {
			return Change{}, err
		}
		if !c.Now.Before(appeal.TalentDeadline) {
			return Change{}, ErrDeadline
		}
		if !validText(input.Observation, 8000) || (input.Recommendation != "confirmar" && input.Recommendation != "rectificar") {
			return Change{}, ErrInvalid
		}
		end := appeal.WindowEnd
		if c.PhaseEnds["reconsideracion"].Before(end) {
			end = c.PhaseEnds["reconsideracion"]
		}
		rules := appeal.Policy.Definition.Reconsideration
		deadline, err := boundedDeadline(c.Now, rules.Commission, rules.Calendar, end)
		if err != nil {
			return Change{}, err
		}
		appeal.Version++
		appeal.Status = "atendida"
		appeal.Attention = &Attention{ActorID: a.ID, At: c.Now, Observation: input.Observation, Recommendation: input.Recommendation}
		appeal.CommissionDeadline = &deadline
		change, err := makeChange(newID(), "reconsideracion.atendida", appeal.Original.Phase, appeal, false)
		change.Appeal = appeal
		return change, err
	})
}

/** Resolve no acepta nuevos puntajes: una rectificación referencia otra publicación calculada/autorizada del mismo expediente y fase. */
func (s *Service) Resolve(ctx context.Context, a *identity.Applicant, id, appealID, key uuid.UUID, input ResolveInput) (Receipt, error) {
	if err := admin(a); err != nil {
		return Receipt{}, err
	}
	cmd, err := command(key, "reconsideracion.resuelta", appealID, input)
	if err != nil {
		return Receipt{}, err
	}
	cmd.AppealID = appealID
	return s.repo.Execute(ctx, a, id, cmd, func(c *Case) (Change, error) {
		appeal := c.Appeal
		if appeal == nil {
			return Change{}, ErrNotFound
		}
		if appeal.Status != "atendida" {
			return Change{}, ErrState
		}
		if appeal.Version != input.PreviousVersion {
			return Change{}, ErrConflict
		}
		if err := phaseWindow(c, "reconsideracion"); err != nil {
			return Change{}, err
		}
		if appeal.CommissionDeadline == nil || !c.Now.Before(*appeal.CommissionDeadline) {
			return Change{}, ErrDeadline
		}
		if !validText(input.Motive, 8000) {
			return Change{}, ErrInvalid
		}
		current, ok := c.Sources[appeal.Original.Phase]
		if !ok {
			return Change{}, ErrPending
		}
		switch input.Decision {
		case "confirmar":
			if input.ReplacementID != nil || current.ID != appeal.Original.ID {
				return Change{}, ErrConflict
			}
		case "rectificar":
			if input.ReplacementID == nil || *input.ReplacementID != current.ID || current.ID == appeal.Original.ID || current.Version <= appeal.Original.Version || current.SubmissionID != id || current.SnapshotHash != appeal.Original.SnapshotHash {
				return Change{}, ErrConflict
			}
		default:
			return Change{}, ErrInvalid
		}
		resolution := &Resolution{ActorID: a.ID, At: c.Now, Decision: input.Decision, Motive: input.Motive, EffectiveResult: current}
		if current.Score != nil && appeal.Original.Score != nil && current.Maximum != nil && appeal.Original.Maximum != nil && *current.Maximum == *appeal.Original.Maximum {
			delta := *current.Score - *appeal.Original.Score
			resolution.ScoreDelta = &delta
		}
		appeal.Status = "resuelta"
		appeal.Version++
		appeal.Resolution = resolution
		change, err := makeChange(newID(), "reconsideracion.resuelta", appeal.Original.Phase, appeal, true)
		change.Appeal = appeal
		return change, err
	})
}
