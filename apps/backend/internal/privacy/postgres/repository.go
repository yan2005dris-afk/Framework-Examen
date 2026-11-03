package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/evaluation"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/db"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/privacy"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"time"
)

type Repository struct {
	q     *sqlcgen.Queries
	tx    *db.TxManager
	audit audit.Recorder
}

func NewRepository(q *sqlcgen.Queries, tx *db.TxManager, a audit.Recorder) *Repository {
	return &Repository{q, tx, a}
}
func ts(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }
func (r *Repository) queries(ctx context.Context) *sqlcgen.Queries {
	if tx, ok := db.Transaction(ctx); ok {
		return r.q.WithTx(tx)
	}
	return r.q
}
func mapped(e error) error {
	if errors.Is(e, pgx.ErrNoRows) {
		return privacy.ErrNotFound
	}
	return e
}
func decode(v sqlcgen.PrivacyPolicy) (*privacy.Policy, error) {
	p := &privacy.Policy{ID: v.ID, CompetitionID: v.CompetitionID, Version: v.Version, Hash: v.Hash, ActorID: v.ActorID, At: v.CreatedAt.Time}
	if e := json.Unmarshal(v.Definition, &p.Rules); e != nil {
		return nil, e
	}
	h, e := evaluation.Digest(p.Rules)
	if e != nil || h != p.Hash || p.Rules.Validate() != nil {
		return nil, privacy.ErrInvalid
	}
	return p, nil
}
func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*privacy.Policy, error) {
	v, e := r.queries(ctx).GetPrivacyPolicy(ctx, id)
	if e != nil {
		return nil, mapped(e)
	}
	return decode(v)
}
func (r *Repository) Current(ctx context.Context, id uuid.UUID) (*privacy.Policy, error) {
	h, e := r.queries(ctx).GetPrivacyControl(ctx, id)
	if e != nil {
		return nil, mapped(e)
	}
	if !h.ActiveID.Valid {
		return nil, privacy.ErrNotFound
	}
	p, e := r.Get(ctx, uuid.UUID(h.ActiveID.Bytes))
	if e == nil {
		p.ControlVersion = h.Version
		p.Blocked = h.Blocked
	}
	return p, e
}
func (r *Repository) List(ctx context.Context, id uuid.UUID, before int64, limit int) ([]privacy.Policy, error) {
	rows, e := r.queries(ctx).ListPrivacyPolicies(ctx, sqlcgen.ListPrivacyPoliciesParams{CompetitionID: id, BeforeVersion: before, RowLimit: int32(limit)})
	if e != nil {
		return nil, e
	}
	out := []privacy.Policy{}
	for _, v := range rows {
		p, e := decode(v)
		if e != nil {
			return nil, e
		}
		out = append(out, *p)
	}
	return out, nil
}
func admin(ctx context.Context, q *sqlcgen.Queries, a *identity.Applicant) error {
	if !identity.HasPermission(a, identity.PermissionPhases) {
		return privacy.ErrForbidden
	}
	rows, e := q.LockApplicantsForRoleChange(ctx, []uuid.UUID{a.ID})
	if e != nil {
		return e
	}
	if len(rows) != 1 || rows[0].Role != identity.RoleAdministrator {
		return privacy.ErrForbidden
	}
	return nil
}
func (r *Repository) record(ctx context.Context, a *identity.Applicant, id uuid.UUID, action string) error {
	recorder, ok := r.audit.(audit.WorkflowRecorder)
	if !ok {
		return audit.ErrUnavailable
	}
	return recorder.RecordWorkflow(ctx, audit.Event{ActorID: a.ID, Role: a.Role, ResourceID: id, ResourceType: "convocatoria", Action: action, Result: "permitido", CreatedAt: time.Now().UTC()})
}

func (r *Repository) Publish(ctx context.Context, a *identity.Applicant, period uuid.UUID, input privacy.PublishInput) (out *privacy.Policy, err error) {
	if input.Rules.Validate() != nil || input.PreviousVersion < 0 || period == uuid.Nil {
		return nil, privacy.ErrInvalid
	}
	err = r.tx.WithinTx(ctx, func(tx pgx.Tx) error {
		ctx = db.WithTransaction(ctx, tx)
		q := r.queries(ctx)
		if e := admin(ctx, q, a); e != nil {
			return e
		}
		if _, e := q.GetMeritCalendar(ctx, period); e != nil {
			return mapped(e)
		}
		if e := q.EnsurePrivacyControl(ctx, period); e != nil {
			return e
		}
		h, e := q.LockPrivacyControl(ctx, period)
		if e != nil {
			return e
		}
		if h.Version != input.PreviousVersion {
			return privacy.ErrConflict
		}
		id, e := uuid.NewV7()
		if e != nil {
			return e
		}
		hash, e := evaluation.Digest(input.Rules)
		if e != nil {
			return e
		}
		now, e := q.EvaluationNow(ctx)
		if e != nil {
			return e
		}
		raw, e := json.Marshal(input.Rules)
		if e != nil {
			return e
		}
		if e = q.CreatePrivacyPolicy(ctx, sqlcgen.CreatePrivacyPolicyParams{ID: id, CompetitionID: period, Version: h.Version + 1, Definition: raw, Hash: hash, ActorID: a.ID, CreatedAt: now}); e != nil {
			return e
		}
		if e = q.SetPrivacyControl(ctx, sqlcgen.SetPrivacyControlParams{CompetitionID: period, Version: h.Version + 1, ActiveID: pgtype.UUID{Bytes: id, Valid: true}, Blocked: h.Blocked}); e != nil {
			return e
		}
		if e = r.record(ctx, a, period, "privacidad.publicada"); e != nil {
			return e
		}
		out, e = r.Current(ctx, period)
		return e
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
func (r *Repository) Block(ctx context.Context, a *identity.Applicant, period uuid.UUID, input privacy.BlockInput) error {
	if len(input.Reason) == 0 || len(input.Reason) > 2000 {
		return privacy.ErrInvalid
	}
	return r.tx.WithinTx(ctx, func(tx pgx.Tx) error {
		ctx = db.WithTransaction(ctx, tx)
		q := r.queries(ctx)
		if e := admin(ctx, q, a); e != nil {
			return e
		}
		h, e := q.LockPrivacyControl(ctx, period)
		if e != nil {
			return mapped(e)
		}
		if input.PreviousVersion != h.Version {
			return privacy.ErrConflict
		}
		id, e := uuid.NewV7()
		if e != nil {
			return e
		}
		now, e := q.EvaluationNow(ctx)
		if e != nil {
			return e
		}
		if e = q.CreatePrivacyDecision(ctx, sqlcgen.CreatePrivacyDecisionParams{ID: id, CompetitionID: period, Blocked: input.Blocked, Reason: input.Reason, ActorID: a.ID, CreatedAt: now}); e != nil {
			return e
		}
		if e = q.SetPrivacyControl(ctx, sqlcgen.SetPrivacyControlParams{CompetitionID: period, Version: h.Version + 1, ActiveID: h.ActiveID, Blocked: input.Blocked}); e != nil {
			return e
		}
		return r.record(ctx, a, period, "privacidad.restringida")
	})
}

/** Bind exige aceptación expresa de la política actual; se guarda dentro del commit del envío sin tocar los cinco literales. */
func (r *Repository) Bind(ctx context.Context, s *submission.Submission, c *submission.Consent) error {
	if _, ok := db.Transaction(ctx); !ok {
		return audit.ErrTransactionRequired
	}
	if s == nil || s.CompetitionID == nil || s.SubmittedAt == nil || c == nil {
		return privacy.ErrDenied
	}
	if submission.ValidateSeal(s) != nil || !c.DataProcessing || !c.Truthfulness || !c.CompetitionRules || !c.NamePublication || !c.PhysicalDelivery {
		return privacy.ErrDenied
	}
	form, e := submission.DecodeForm(s.Content)
	if e != nil || form.Consent == nil {
		return privacy.ErrDenied
	}
	given, e := evaluation.Digest(c)
	if e != nil {
		return privacy.ErrDenied
	}
	sealed, e := evaluation.Digest(form.Consent)
	if e != nil || given != sealed {
		return privacy.ErrDenied
	}
	q := r.queries(ctx)
	h, e := q.GetPrivacyControl(ctx, *s.CompetitionID)
	if e != nil || h.Blocked || !h.ActiveID.Valid {
		return privacy.ErrDenied
	}
	p, e := r.Get(ctx, uuid.UUID(h.ActiveID.Bytes))
	if e != nil {
		return e
	}
	if c.PrivacyPolicyID != p.ID.String() || c.PrivacyPolicyHash != p.Hash {
		return privacy.ErrDenied
	}
	return q.BindPrivacy(ctx, sqlcgen.BindPrivacyParams{SubmissionID: s.ID, PolicyID: p.ID, PolicyHash: p.Hash, AcceptedAt: ts(*s.SubmittedAt), RetentionReviewAt: ts(s.SubmittedAt.AddDate(0, 0, p.Rules.ReviewDays))})
}
func (r *Repository) Allow(ctx context.Context, id uuid.UUID, channel string) error {
	q := r.queries(ctx)
	b, e := q.GetBoundPrivacy(ctx, id)
	if errors.Is(e, pgx.ErrNoRows) {
		return privacy.ErrDenied
	}
	if e != nil {
		return e
	}
	if b.Blocked {
		return privacy.ErrDenied
	}
	var rules privacy.Rules
	if json.Unmarshal(b.Definition, &rules) != nil {
		return privacy.ErrDenied
	}
	hash, e := evaluation.Digest(rules)
	if e != nil || hash != b.PolicyHash {
		return privacy.ErrDenied
	}
	now, e := q.EvaluationNow(ctx)
	if e != nil {
		return e
	}
	instant := now.Time
	if channel == "documentos" {
		instant = instant.Add(5 * time.Minute)
	}
	return rules.Allow(b.AcceptedAt.Time, instant, channel)
}
func (r *Repository) Period(ctx context.Context, id uuid.UUID, channel string) error {
	q := r.queries(ctx)
	rows, e := q.PrivacyPeriodSubmissions(ctx, id)
	if e != nil {
		return e
	}
	if len(rows) > 10000 {
		return privacy.ErrDenied
	}
	h, e := q.GetPrivacyControl(ctx, id)
	if e != nil || h.Blocked || !h.ActiveID.Valid {
		return privacy.ErrDenied
	}
	for _, sub := range rows {
		if e = r.Allow(ctx, sub, channel); e != nil {
			return e
		}
	}
	return nil
}
func (r *Repository) Reviews(ctx context.Context, id uuid.UUID, at time.Time, after uuid.UUID, limit int) ([]privacy.Review, error) {
	rows, e := r.q.PrivacyReviewDue(ctx, sqlcgen.PrivacyReviewDueParams{CompetitionID: pgtype.UUID{Bytes: id, Valid: true}, HasCursor: !at.IsZero(), AfterAt: ts(at), AfterID: after, RowLimit: int32(limit)})
	if e != nil {
		return nil, e
	}
	out := []privacy.Review{}
	for _, v := range rows {
		out = append(out, privacy.Review{SubmissionID: v.SubmissionID, PolicyID: v.PolicyID, Due: v.RetentionReviewAt.Time})
	}
	return out, nil
}
