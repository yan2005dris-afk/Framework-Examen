package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/competition"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/evaluation"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/merit"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/db"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/results"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/shared/domain"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"slices"
	"time"
)

type Repository struct {
	q        *sqlcgen.Queries
	tx       *db.TxManager
	audit    audit.WorkflowRecorder
	notifier results.Notifier
}

func NewRepository(q *sqlcgen.Queries, tx *db.TxManager, a audit.WorkflowRecorder, n results.Notifier) *Repository {
	return &Repository{q, tx, a, n}
}
func ts(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }
func uid(id uuid.UUID) pgtype.UUID      { return pgtype.UUID{Bytes: id, Valid: true} }
func mapped(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return results.ErrNotFound
	}
	var p *pgconn.PgError
	if errors.As(err, &p) && p.Code == "23505" {
		return results.ErrConflict
	}
	return err
}
func actor(ctx context.Context, q *sqlcgen.Queries, a *identity.Applicant, publish bool) error {
	if a == nil || !identity.HasPermission(a, identity.PermissionResults) || publish && a.Role != identity.RoleAdministrator {
		return results.ErrForbidden
	}
	rows, err := q.LockApplicantsForRoleChange(ctx, []uuid.UUID{a.ID})
	if err != nil {
		return err
	}
	if len(rows) != 1 || rows[0].Role != a.Role {
		return results.ErrForbidden
	}
	return nil
}
func (r *Repository) record(ctx context.Context, a *identity.Applicant, period uuid.UUID, action string, now time.Time) error {
	if r.audit == nil {
		return audit.ErrUnavailable
	}
	return r.audit.RecordWorkflow(ctx, audit.Event{ActorID: a.ID, Role: a.Role, ResourceID: period, ResourceType: "convocatoria", Action: action, Result: "permitido", CreatedAt: now})
}

/** calendar bloquea el control común de T18 para que generación/publicación no se mezclen con nuevas fuentes o solicitudes. */
func calendar(ctx context.Context, q *sqlcgen.Queries, period uuid.UUID, sector string) (time.Time, error) {
	c, err := q.LockResultsCalendar(ctx, period)
	if err != nil {
		return time.Time{}, mapped(err)
	}
	clock, err := q.EvaluationNow(ctx)
	if err != nil {
		return time.Time{}, err
	}
	now := clock.Time
	if !c.Approved || !c.Published || !slices.Contains(c.Sectors, sector) || now.Before(c.OpensAt.Time) || !now.Before(c.ClosesAt.Time) {
		return now, results.ErrPhase
	}
	var phases []competition.Phase
	if json.Unmarshal(c.Phases, &phases) != nil {
		return now, results.ErrPhase
	}
	ready, closed := false, false
	for _, p := range phases {
		if p.Code == "resultados" && p.Published && p.Enabled && !now.Before(p.OpensAt) && now.Before(p.ClosesAt) {
			ready = true
		}
		if p.Code == "reconsideracion" && p.Published && !now.Before(p.ClosesAt) {
			closed = true
		}
		if p.Code == "postulacion" && p.Enabled && !now.Before(p.OpensAt) && now.Before(p.ClosesAt) {
			return now, results.ErrPhase
		}
	}
	if !ready || !closed {
		return now, results.ErrPhase
	}
	return now, nil
}
func collect(ctx context.Context, q *sqlcgen.Queries, period uuid.UUID, sector string, rules results.Rules) ([]results.Entry, string, error) {
	budget, err := q.ResultSourceBudget(ctx, sqlcgen.ResultSourceBudgetParams{CompetitionID: period, Sector: sector})
	if err != nil {
		return nil, "", err
	}
	if budget.Total > results.MaxRows || budget.Bytes > results.MaxBytes {
		return nil, "", results.ErrTooLarge
	}
	var policy *evaluation.Policy
	h, err := q.GetEvaluationPolicyStream(ctx, sqlcgen.GetEvaluationPolicyStreamParams{CompetitionID: period, Sector: sector, Kind: "oposicion"})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, "", err
	}
	if err == nil && h.ActiveID.Valid {
		v, err := q.GetEvaluationPolicy(ctx, uuid.UUID(h.ActiveID.Bytes))
		if err != nil {
			return nil, "", err
		}
		policy = &evaluation.Policy{ID: v.ID, CompetitionID: v.CompetitionID, Sector: v.Sector, Kind: v.Kind, Hash: v.DefinitionHash, Approved: v.Approved}
		if err = json.Unmarshal(v.Definition, &policy.Definition); err != nil {
			return nil, "", err
		}
	}
	rows, err := q.ListResultSources(ctx, sqlcgen.ListResultSourcesParams{CompetitionID: period, Sector: sector})
	if err != nil {
		return nil, "", err
	}
	sources := make([]results.Source, 0, len(rows))
	for _, v := range rows {
		snap := &submission.Submission{ID: v.ID, ApplicantID: v.ApplicantID, Code: v.Code, Status: submission.Status(v.Status), Content: v.Content, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt, CompetitionID: &period}
		if v.Hash != nil {
			snap.Hash = *v.Hash
		}
		if v.SubmittedAt.Valid {
			t := v.SubmittedAt.Time
			snap.SubmittedAt = &t
		}
		source := results.Source{Snapshot: snap}
		if string(v.Merit) != "{}" {
			source.Merit = &merit.Publication{}
			if err = json.Unmarshal(v.Merit, source.Merit); err != nil {
				return nil, "", err
			}
		}
		if err = json.Unmarshal(v.Evaluation, &source.State); err != nil {
			return nil, "", err
		}
		if string(v.Theory) != "{}" {
			source.Theory = &evaluation.Result{}
			if err = json.Unmarshal(v.Theory, source.Theory); err != nil {
				return nil, "", err
			}
		}
		if err = json.Unmarshal(v.Appeals, &source.Appeals); err != nil {
			return nil, "", err
		}
		sources = append(sources, source)
	}
	return results.Build(period, sector, rules, policy, sources)
}
func get(ctx context.Context, q *sqlcgen.Queries, id uuid.UUID) (*results.List, error) {
	v, err := q.GetResultList(ctx, id)
	if err != nil {
		return nil, mapped(err)
	}
	list, err := results.DecodeList(v.Payload)
	if err != nil {
		return nil, err
	}
	hash, hashErr := evaluation.Digest(struct {
		Rules   results.Rules
		Entries []results.Entry
	}{list.Rules, list.Entries})
	if hashErr != nil || hash != list.SourcesHash || list.Version != v.Version || list.ID != v.ID || list.SourcesHash != v.SourcesHash || list.CompetitionID != v.CompetitionID || list.Sector != v.Sector {
		return nil, results.ErrInvalid
	}
	return list, nil
}
func publication(v sqlcgen.ResultListPublication) (*results.Publication, error) {
	p := &results.Publication{ID: v.ID, ListID: v.ListID, CompetitionID: v.CompetitionID, Sector: v.Sector, Version: v.Version, Reference: v.ApprovalReference, PrivacyReference: v.PrivacyReference, Public: v.IsPublic, ActorID: v.ActorID, CreatedAt: v.CreatedAt.Time}
	if v.PreviousID.Valid {
		id := uuid.UUID(v.PreviousID.Bytes)
		p.PreviousID = &id
	}
	if err := json.Unmarshal(v.PublicPayload, &p.Names); err != nil {
		return nil, err
	}
	return p, nil
}
func getPublication(ctx context.Context, q *sqlcgen.Queries, id uuid.UUID) (*results.Publication, error) {
	v, err := q.GetResultListPublication(ctx, id)
	if err != nil {
		return nil, mapped(err)
	}
	return publication(v)
}
func replay(ctx context.Context, q *sqlcgen.Queries, a *identity.Applicant, key uuid.UUID, operation, scope, hash string) (uuid.UUID, error) {
	v, err := q.GetResultListCommand(ctx, key)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, nil
	}
	if err != nil {
		return uuid.Nil, err
	}
	if v.ActorID != a.ID || v.Operation != operation || v.Scope != scope || v.RequestHash != hash {
		return uuid.Nil, results.ErrConflict
	}
	return v.ResourceID, nil
}
func command(ctx context.Context, q *sqlcgen.Queries, a *identity.Applicant, key uuid.UUID, operation, scope, hash string, id uuid.UUID) error {
	return mapped(q.CreateResultListCommand(ctx, sqlcgen.CreateResultListCommandParams{ID: key, ActorID: a.ID, Operation: operation, Scope: scope, RequestHash: hash, ResourceID: id}))
}
func (r *Repository) Generate(ctx context.Context, a *identity.Applicant, period uuid.UUID, sector string, key uuid.UUID, input results.GenerateInput) (out *results.List, err error) {
	err = r.tx.WithinTx(ctx, func(tx pgx.Tx) error {
		ctx = db.WithTransaction(ctx, tx)
		q := r.q.WithTx(tx)
		if err := actor(ctx, q, a, false); err != nil {
			return err
		}
		hash, err := evaluation.Digest(input)
		if err != nil {
			return err
		}
		scope := period.String() + ":" + sector
		prior, err := replay(ctx, q, a, key, "generate", scope, hash)
		if err != nil {
			return err
		}
		if prior != uuid.Nil {
			out, err = get(ctx, q, prior)
			return err
		}
		now, err := calendar(ctx, q, period, sector)
		if err != nil {
			return err
		}
		if err = q.EnsureResultListStream(ctx, sqlcgen.EnsureResultListStreamParams{CompetitionID: period, Sector: sector}); err != nil {
			return err
		}
		h, err := q.LockResultListStream(ctx, sqlcgen.LockResultListStreamParams{CompetitionID: period, Sector: sector})
		if err != nil {
			return err
		}
		items, sourceHash, err := collect(ctx, q, period, sector, input.Rules)
		if err != nil {
			return err
		}
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		out = &results.List{ID: id, CompetitionID: period, Sector: sector, Version: h.Version + 1, Rules: input.Rules, SourcesHash: sourceHash, ActorID: a.ID, CreatedAt: now, Entries: items}
		raw, err := results.EncodeList(out)
		if err != nil {
			return err
		}
		if len(raw) > results.MaxBytes {
			return results.ErrTooLarge
		}
		if err = q.CreateResultList(ctx, sqlcgen.CreateResultListParams{ID: id, CompetitionID: period, Sector: sector, Version: out.Version, SourcesHash: sourceHash, Payload: raw, ActorID: a.ID, CreatedAt: ts(now)}); err != nil {
			return err
		}
		if err = q.SetResultListStream(ctx, sqlcgen.SetResultListStreamParams{CompetitionID: period, Sector: sector, Version: out.Version, PublishedVersion: h.PublishedVersion, PublishedID: h.PublishedID}); err != nil {
			return err
		}
		if err = r.record(ctx, a, period, "lista.generada", now); err != nil {
			return err
		}
		return command(ctx, q, a, key, "generate", scope, hash, id)
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
func (r *Repository) Publish(ctx context.Context, a *identity.Applicant, id, key uuid.UUID, input results.PublishInput) (out *results.Publication, err error) {
	err = r.tx.WithinTx(ctx, func(tx pgx.Tx) error {
		ctx = db.WithTransaction(ctx, tx)
		q := r.q.WithTx(tx)
		if err := actor(ctx, q, a, true); err != nil {
			return err
		}
		hash, err := evaluation.Digest(input)
		if err != nil {
			return err
		}
		prior, err := replay(ctx, q, a, key, "publish", id.String(), hash)
		if err != nil {
			return err
		}
		if prior != uuid.Nil {
			out, err = getPublication(ctx, q, prior)
			return err
		}
		list, err := get(ctx, q, id)
		if err != nil {
			return err
		}
		now, err := calendar(ctx, q, list.CompetitionID, list.Sector)
		if err != nil {
			return err
		}
		h, err := q.LockResultListStream(ctx, sqlcgen.LockResultListStreamParams{CompetitionID: list.CompetitionID, Sector: list.Sector})
		if err != nil {
			return err
		}
		if input.PreviousVersion != h.PublishedVersion || (h.PublishedID.Valid && (input.SupersedesID == nil || *input.SupersedesID != uuid.UUID(h.PublishedID.Bytes))) || (!h.PublishedID.Valid && input.SupersedesID != nil) {
			return results.ErrConflict
		}
		_, sourceHash, err := collect(ctx, q, list.CompetitionID, list.Sector, list.Rules)
		if err != nil {
			return err
		}
		if sourceHash != list.SourcesHash {
			return results.ErrConflict
		}
		names := []string{}
		if input.Public {
			names, err = results.PublicNames(list.Entries)
			if err != nil {
				return err
			}
		}
		raw, err := json.Marshal(names)
		if err != nil {
			return err
		}
		pubID, err := uuid.NewV7()
		if err != nil {
			return err
		}
		var previous pgtype.UUID
		if input.SupersedesID != nil {
			previous = uid(*input.SupersedesID)
		}
		if err = q.CreateResultListPublication(ctx, sqlcgen.CreateResultListPublicationParams{ID: pubID, ListID: id, CompetitionID: list.CompetitionID, Sector: list.Sector, Version: h.PublishedVersion + 1, PreviousID: previous, RequestHash: hash, ApprovalReference: input.Reference, PrivacyReference: input.PrivacyReference, IsPublic: input.Public, PublicPayload: raw, ActorID: a.ID, CreatedAt: ts(now)}); err != nil {
			return mapped(err)
		}
		if err = q.SetResultListStream(ctx, sqlcgen.SetResultListStreamParams{CompetitionID: list.CompetitionID, Sector: list.Sector, Version: h.Version, PublishedVersion: h.PublishedVersion + 1, PublishedID: uid(pubID)}); err != nil {
			return err
		}
		if err = r.record(ctx, a, list.CompetitionID, "lista.publicada", now); err != nil {
			return err
		}
		if r.notifier == nil {
			return errors.New("cola de listas no configurada")
		}
		for _, e := range list.Entries {
			if err = r.notifier.Enqueue(ctx, e.SubmissionID, pubID, "lista.publicada"); err != nil {
				return err
			}
		}
		if err = command(ctx, q, a, key, "publish", id.String(), hash, pubID); err != nil {
			return err
		}
		out, err = getPublication(ctx, q, pubID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*results.List, error) {
	return get(ctx, r.q, id)
}
func (r *Repository) Publication(ctx context.Context, id uuid.UUID) (*results.Publication, error) {
	return getPublication(ctx, r.q, id)
}
func (r *Repository) Current(ctx context.Context, id uuid.UUID, sector string) (*results.Publication, error) {
	h, err := r.q.GetResultListStream(ctx, sqlcgen.GetResultListStreamParams{CompetitionID: id, Sector: sector})
	if err != nil {
		return nil, mapped(err)
	}
	if !h.PublishedID.Valid {
		return nil, results.ErrNotFound
	}
	return getPublication(ctx, r.q, uuid.UUID(h.PublishedID.Bytes))
}
func (r *Repository) Lists(ctx context.Context, id uuid.UUID, sector string, p domain.Position, limit int) ([]results.List, error) {
	rows, err := r.q.ListResultLists(ctx, sqlcgen.ListResultListsParams{CompetitionID: id, Sector: sector, HasCursor: !p.At.IsZero(), BeforeAt: ts(p.At), BeforeID: p.ID, RowLimit: int32(limit)})
	if err != nil {
		return nil, err
	}
	items := []results.List{}
	for _, v := range rows {
		items = append(items, results.List{ID: v.ID, CompetitionID: id, Sector: sector, Version: v.Version, SourcesHash: v.SourcesHash, ActorID: v.ActorID, CreatedAt: v.CreatedAt.Time})
	}
	return items, nil
}
func (r *Repository) Publications(ctx context.Context, id uuid.UUID, sector string, p domain.Position, limit int) ([]results.Publication, error) {
	rows, err := r.q.ListResultListPublications(ctx, sqlcgen.ListResultListPublicationsParams{CompetitionID: id, Sector: sector, HasCursor: !p.At.IsZero(), BeforeAt: ts(p.At), BeforeID: p.ID, RowLimit: int32(limit)})
	if err != nil {
		return nil, err
	}
	items := []results.Publication{}
	for _, v := range rows {
		items = append(items, results.Publication{ID: v.ID, ListID: v.ListID, CompetitionID: id, Sector: sector, Version: v.Version, Public: v.IsPublic, ActorID: v.ActorID, CreatedAt: v.CreatedAt.Time})
	}
	return items, nil
}
