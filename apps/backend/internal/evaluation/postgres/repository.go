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
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/shared/domain"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"slices"
	"time"
)

type Repository struct {
	queries  *sqlcgen.Queries
	tx       *db.TxManager
	auditor  audit.WorkflowRecorder
	notifier evaluation.Notifier
}

func NewRepository(q *sqlcgen.Queries, tx *db.TxManager, a audit.WorkflowRecorder, n evaluation.Notifier) *Repository {
	return &Repository{q, tx, a, n}
}
func timestamp(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }
func mapError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return evaluation.ErrNotFound
	}
	var constraint *pgconn.PgError
	if errors.As(err, &constraint) && constraint.Code == "23505" {
		return evaluation.ErrConflict
	}
	return err
}

/** lockActor comparte orden UUID con cambios de rol y bloquea al titular para serializar la cuota con T4. */
func lockActor(ctx context.Context, q *sqlcgen.Queries, a *identity.Applicant, owner uuid.UUID, operation string) error {
	if a == nil || a.ID == uuid.Nil {
		return evaluation.ErrForbidden
	}
	ids := []uuid.UUID{a.ID}
	if owner != uuid.Nil && owner != a.ID {
		ids = append(ids, owner)
	}
	rows, err := q.LockApplicantsForRoleChange(ctx, ids)
	if err != nil {
		return err
	}
	found := false
	for _, row := range rows {
		if row.ID == a.ID && row.Role == a.Role {
			found = true
		}
	}
	if !found {
		return evaluation.ErrForbidden
	}
	switch operation {
	case "reconsideracion.presentada":
		if a.Role != identity.RoleApplicant || a.ID != owner {
			return evaluation.ErrForbidden
		}
	case "reconsideracion.atendida":
		if a.Role != identity.RoleAnalyst {
			return evaluation.ErrForbidden
		}
	case "evaluacion.reglas_creadas", "psicotecnica.registrada":
		if !identity.HasPermission(a, identity.PermissionGrades) {
			return evaluation.ErrForbidden
		}
	default:
		if a.Role != identity.RoleAdministrator {
			return evaluation.ErrForbidden
		}
	}
	return nil
}
func (r *Repository) record(ctx context.Context, a *identity.Applicant, id uuid.UUID, kind, action string) error {
	if r.auditor == nil {
		return audit.ErrUnavailable
	}
	sid := uuid.Nil
	if kind == "postulacion" {
		sid = id
	}
	return r.auditor.RecordWorkflow(ctx, audit.Event{ActorID: a.ID, Role: a.Role, SubmissionID: sid, ResourceID: id, ResourceType: kind, Action: action, Result: "permitido", CreatedAt: time.Now().UTC()})
}
func decodePolicy(row sqlcgen.EvaluationPolicy) (*evaluation.Policy, error) {
	p := &evaluation.Policy{ID: row.ID, CompetitionID: row.CompetitionID, Sector: row.Sector, Kind: row.Kind, Version: row.Version, Hash: row.DefinitionHash, Approved: row.Approved, ActorID: row.ActorID, CreatedAt: row.CreatedAt.Time}
	if row.ApprovalReference != nil {
		p.Reference = *row.ApprovalReference
	}
	if row.ApprovedBy.Valid {
		id := uuid.UUID(row.ApprovedBy.Bytes)
		p.ApprovedBy = &id
	}
	if row.ApprovedAt.Valid {
		t := row.ApprovedAt.Time
		p.ApprovedAt = &t
	}
	err := json.Unmarshal(row.Definition, &p.Definition)
	return p, err
}
func getPolicy(ctx context.Context, q *sqlcgen.Queries, id uuid.UUID) (*evaluation.Policy, error) {
	row, err := q.GetEvaluationPolicy(ctx, id)
	if err != nil {
		return nil, mapError(err)
	}
	return decodePolicy(row)
}
func (r *Repository) GetPolicy(ctx context.Context, id uuid.UUID) (*evaluation.Policy, error) {
	return getPolicy(ctx, r.queries, id)
}
func (r *Repository) ListPolicies(ctx context.Context, id uuid.UUID, sector, kind string, before int64, limit int) ([]evaluation.Policy, error) {
	rows, err := r.queries.ListEvaluationPolicies(ctx, sqlcgen.ListEvaluationPoliciesParams{CompetitionID: id, Sector: sector, Kind: kind, BeforeVersion: before, RowLimit: int32(limit)})
	if err != nil {
		return nil, err
	}
	items := []evaluation.Policy{}
	for _, row := range rows {
		p, err := decodePolicy(row)
		if err != nil {
			return nil, err
		}
		items = append(items, *p)
	}
	return items, nil
}
func (r *Repository) CreatePolicy(ctx context.Context, a *identity.Applicant, p *evaluation.Policy) error {
	return r.tx.WithinTx(ctx, func(tx pgx.Tx) error {
		ctx = db.WithTransaction(ctx, tx)
		q := r.queries.WithTx(tx)
		if err := lockActor(ctx, q, a, uuid.Nil, "evaluacion.reglas_creadas"); err != nil {
			return err
		}
		calendar, err := q.GetMeritCalendar(ctx, p.CompetitionID)
		if err != nil {
			return mapError(err)
		}
		if !slices.Contains(calendar.Sectors, p.Sector) {
			return evaluation.ErrInvalid
		}
		if err = q.EnsureEvaluationPolicyStream(ctx, sqlcgen.EnsureEvaluationPolicyStreamParams{CompetitionID: p.CompetitionID, Sector: p.Sector, Kind: p.Kind}); err != nil {
			return err
		}
		h, err := q.LockEvaluationPolicyStream(ctx, sqlcgen.LockEvaluationPolicyStreamParams{CompetitionID: p.CompetitionID, Sector: p.Sector, Kind: p.Kind})
		if err != nil {
			return err
		}
		p.Version = h.Version + 1
		raw, err := json.Marshal(p.Definition)
		if err != nil {
			return err
		}
		if err = q.CreateEvaluationPolicy(ctx, sqlcgen.CreateEvaluationPolicyParams{ID: p.ID, CompetitionID: p.CompetitionID, Sector: p.Sector, Kind: p.Kind, Version: p.Version, Definition: raw, DefinitionHash: p.Hash, ActorID: a.ID, CreatedAt: timestamp(p.CreatedAt)}); err != nil {
			return err
		}
		if err = q.SetEvaluationPolicyStream(ctx, sqlcgen.SetEvaluationPolicyStreamParams{CompetitionID: p.CompetitionID, Sector: p.Sector, Kind: p.Kind, Version: p.Version, ActiveID: h.ActiveID}); err != nil {
			return err
		}
		return r.record(ctx, a, p.CompetitionID, "convocatoria", "evaluacion.reglas_creadas")
	})
}
func (r *Repository) ApprovePolicy(ctx context.Context, a *identity.Applicant, id uuid.UUID, reference string) (result *evaluation.Policy, err error) {
	err = r.tx.WithinTx(ctx, func(tx pgx.Tx) error {
		ctx = db.WithTransaction(ctx, tx)
		q := r.queries.WithTx(tx)
		if err := lockActor(ctx, q, a, uuid.Nil, "evaluacion.reglas_aprobadas"); err != nil {
			return err
		}
		p, err := getPolicy(ctx, q, id)
		if err != nil {
			return err
		}
		h, err := q.LockEvaluationPolicyStream(ctx, sqlcgen.LockEvaluationPolicyStreamParams{CompetitionID: p.CompetitionID, Sector: p.Sector, Kind: p.Kind})
		if err != nil {
			return err
		}
		if p.Approved {
			return evaluation.ErrConflict
		}
		if err = p.Validate(); err != nil {
			return err
		}
		if h.ActiveID.Valid {
			active, err := getPolicy(ctx, q, uuid.UUID(h.ActiveID.Bytes))
			if err != nil {
				return err
			}
			if active.Version >= p.Version {
				return evaluation.ErrConflict
			}
		}
		if err = q.ApproveEvaluationPolicy(ctx, sqlcgen.ApproveEvaluationPolicyParams{ID: id, ApprovalReference: &reference, ApprovedBy: pgtype.UUID{Bytes: a.ID, Valid: true}}); err != nil {
			return err
		}
		if err = q.SetEvaluationPolicyStream(ctx, sqlcgen.SetEvaluationPolicyStreamParams{CompetitionID: p.CompetitionID, Sector: p.Sector, Kind: p.Kind, Version: h.Version, ActiveID: pgtype.UUID{Bytes: id, Valid: true}}); err != nil {
			return err
		}
		if err = r.record(ctx, a, p.CompetitionID, "convocatoria", "evaluacion.reglas_aprobadas"); err != nil {
			return err
		}
		result, err = getPolicy(ctx, q, id)
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
func loadCase(ctx context.Context, q *sqlcgen.Queries, id, appealID uuid.UUID, write bool) (*evaluation.Case, error) {
	row, err := q.GetSubmissionByID(ctx, id)
	if err != nil {
		return nil, mapError(err)
	}
	snapshot := &submission.Submission{ID: row.ID, ApplicantID: row.ApplicantID, Code: row.Code, Content: row.Content, Status: submission.Status(row.Status), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
	if row.Hash != nil {
		snapshot.Hash = *row.Hash
	}
	if row.SubmittedAt.Valid {
		t := row.SubmittedAt.Time
		snapshot.SubmittedAt = &t
	}
	if row.CompetitionID.Valid {
		period := uuid.UUID(row.CompetitionID.Bytes)
		snapshot.CompetitionID = &period
	}
	if err = submission.ValidateSeal(snapshot); err != nil {
		return nil, err
	}
	form, err := submission.DecodeForm(snapshot.Content)
	if err != nil || form.Position == nil {
		return nil, evaluation.ErrInvalid
	}
	c := &evaluation.Case{Snapshot: snapshot, Sector: form.Position.Sector, State: evaluation.State{Results: map[string]evaluation.Result{}}, Sources: map[string]evaluation.Result{}, PhaseStarts: map[string]time.Time{}, PhaseEnds: map[string]time.Time{}, Documents: map[uuid.UUID]string{}}
	if snapshot.CompetitionID != nil {
		period := *snapshot.CompetitionID
		cal, err := q.GetMeritCalendar(ctx, period)
		if err != nil {
			return nil, err
		}
		var phases []competition.Phase
		if err = json.Unmarshal(cal.Phases, &phases); err != nil {
			return nil, err
		}
		if cal.Approved && cal.Published && slices.Contains(cal.Sectors, c.Sector) {
			for _, p := range phases {
				if p.Enabled && p.Published {
					start, end := p.OpensAt, p.ClosesAt
					if start.Before(cal.OpensAt.Time) {
						start = cal.OpensAt.Time
					}
					if end.After(cal.ClosesAt.Time) {
						end = cal.ClosesAt.Time
					}
					c.PhaseStarts[p.Code] = start
					c.PhaseEnds[p.Code] = end
				}
			}
		}
		for _, kind := range []string{"oposicion", "reconsideracion"} {
			head, err := q.GetEvaluationPolicyStream(ctx, sqlcgen.GetEvaluationPolicyStreamParams{CompetitionID: period, Sector: c.Sector, Kind: kind})
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return nil, err
			}
			if err == nil && head.ActiveID.Valid {
				p, err := getPolicy(ctx, q, uuid.UUID(head.ActiveID.Bytes))
				if err != nil {
					return nil, err
				}
				if kind == "oposicion" {
					c.OppositionPolicy = p
				} else {
					c.AppealPolicy = p
				}
			}
		}
	}
	meritHead, err := q.GetMeritCase(ctx, id)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if err == nil && meritHead.PublicationID.Valid {
		raw, err := q.GetMeritPublication(ctx, sqlcgen.GetMeritPublicationParams{ID: uuid.UUID(meritHead.PublicationID.Bytes), SubmissionID: id})
		if err != nil {
			return nil, err
		}
		var p merit.Publication
		if err = json.Unmarshal(raw, &p); err != nil {
			return nil, err
		}
		if p.SubmissionID != id || p.SnapshotHash != snapshot.Hash || p.ID != uuid.UUID(meritHead.PublicationID.Bytes) {
			return nil, evaluation.ErrInvalid
		}
		total, maximum := p.Evaluation.Total, p.Evaluation.Maximum
		c.Sources["meritos"] = evaluation.Result{ID: p.ID, SubmissionID: id, Phase: "meritos", Version: p.Version, SupersedesID: p.SupersedesID, SourceID: p.CalculationID, PolicyID: p.RuleSetID, PolicyHash: p.RulesHash, SnapshotHash: p.SnapshotHash, Score: &total, Maximum: &maximum, CanAdvance: p.Decision == "avanza", Motive: p.Motive, ActorID: p.ActorID, PublishedAt: p.CreatedAt}
	}
	var head sqlcgen.EvaluationHead
	if write {
		if err = q.EnsureEvaluationHead(ctx, id); err != nil {
			return nil, err
		}
		head, err = q.LockEvaluationHead(ctx, id)
	} else {
		head, err = q.GetEvaluationHead(ctx, id)
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if err == nil {
		if err = json.Unmarshal(head.State, &c.State); err != nil {
			return nil, err
		}
	}
	if c.State.Results == nil {
		c.State.Results = map[string]evaluation.Result{}
	}
	for phase, result := range c.State.Results {
		if result.SubmissionID != id || result.SnapshotHash != snapshot.Hash || result.Phase != phase {
			return nil, evaluation.ErrInvalid
		}
		c.Sources[phase] = result
	}
	theory, err := q.GetFinalTheoryResult(ctx, id)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if err == nil {
		var result evaluation.Result
		if err = json.Unmarshal(theory, &result); err != nil {
			return nil, err
		}
		if result.SubmissionID != id || result.Phase != "oposicion_teorica" || result.SnapshotHash != snapshot.Hash {
			return nil, evaluation.ErrInvalid
		}
		c.Sources[result.Phase] = result
	}
	docs, err := q.ListDocuments(ctx, id)
	if err != nil {
		return nil, err
	}
	for _, d := range docs {
		c.Documents[d.ID] = d.Sha256
	}
	if appealID != uuid.Nil {
		raw, err := q.GetReconsideration(ctx, sqlcgen.GetReconsiderationParams{ID: appealID, SubmissionID: id})
		if err != nil {
			return nil, mapError(err)
		}
		if err = json.Unmarshal(raw, &c.Appeal); err != nil {
			return nil, err
		}
		if c.Appeal == nil || c.Appeal.ID != appealID || c.Appeal.SubmissionID != id {
			return nil, evaluation.ErrInvalid
		}
	}
	now, err := q.EvaluationNow(ctx)
	if err != nil {
		return nil, err
	}
	c.Now = now.Time
	return c, nil
}
func (r *Repository) ReadCase(ctx context.Context, id, appealID uuid.UUID) (result *evaluation.Case, err error) {
	err = r.tx.WithinTx(ctx, func(tx pgx.Tx) error {
		result, err = loadCase(ctx, r.queries.WithTx(tx), id, appealID, false)
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

/** Execute serializa el titular, los resultados fuente, la versión y el reloj; un reintento idéntico recupera el recibo antes del control de plazo. */
func (r *Repository) Execute(ctx context.Context, a *identity.Applicant, id uuid.UUID, cmd evaluation.Command, mutate func(*evaluation.Case) (evaluation.Change, error)) (receipt evaluation.Receipt, err error) {
	err = r.tx.WithinTx(ctx, func(tx pgx.Tx) error {
		ctx = db.WithTransaction(ctx, tx)
		q := r.queries.WithTx(tx)
		source, err := q.GetSubmissionByID(ctx, id)
		if err != nil {
			return mapError(err)
		}
		if err = lockActor(ctx, q, a, source.ApplicantID, cmd.Operation); err != nil {
			return err
		}
		previous, err := q.GetEvaluationCommand(ctx, cmd.Key)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if err == nil {
			if previous.ActorID != a.ID || previous.SubmissionID != id || previous.Operation != cmd.Operation || previous.ResourceID != cmd.ResourceID || previous.RequestHash != cmd.Hash {
				return evaluation.ErrConflict
			}
			canonical, canonicalErr := submission.CanonicalContent(previous.Response)
			if canonicalErr != nil {
				return canonicalErr
			}
			receipt = evaluation.Receipt{Data: canonical, Applied: false}
			return nil
		}
		c, err := loadCase(ctx, q, id, cmd.AppealID, true)
		if err != nil {
			return err
		}
		change, err := mutate(c)
		if err != nil {
			return err
		}
		if change.Action != cmd.Operation || change.ID == uuid.Nil {
			return evaluation.ErrInvalid
		}
		if e := change.Evidence; e != nil {
			if e.ApplicantUpload {
				total, err := q.TotalDocumentBytes(ctx, source.ApplicantID)
				if err != nil {
					return err
				}
				if total+e.Size > 50<<20 {
					return evaluation.ErrQuota
				}
			}
			if err = q.CreateEvaluationEvidence(ctx, sqlcgen.CreateEvaluationEvidenceParams{ID: e.ID, SubmissionID: id, ApplicantID: source.ApplicantID, ActorID: a.ID, ApplicantUpload: e.ApplicantUpload, Name: e.Name, SizeBytes: e.Size, Sha256: e.SHA256, StorageKey: e.StorageKey, CreatedAt: timestamp(c.Now)}); err != nil {
				return err
			}
			if err = q.CompleteEvaluationUpload(ctx, e.ID); err != nil {
				return err
			}
		}
		if err = q.CreateEvaluationRecord(ctx, sqlcgen.CreateEvaluationRecordParams{ID: change.ID, SubmissionID: id, Action: change.Action, Phase: change.Phase, Payload: change.Record, CreatedAt: timestamp(c.Now)}); err != nil {
			return mapError(err)
		}
		if appeal := change.Appeal; appeal != nil {
			raw, err := json.Marshal(appeal)
			if err != nil {
				return err
			}
			if appeal.Version == 1 {
				err = q.CreateReconsideration(ctx, sqlcgen.CreateReconsiderationParams{ID: appeal.ID, SubmissionID: id, Phase: appeal.Original.Phase, ResultID: appeal.Original.ID, PolicyID: appeal.Policy.ID, Status: appeal.Status, Version: appeal.Version, Payload: raw, CreatedAt: timestamp(appeal.FiledAt)})
			} else {
				err = q.UpdateReconsideration(ctx, sqlcgen.UpdateReconsiderationParams{ID: appeal.ID, SubmissionID: id, Status: appeal.Status, Version: appeal.Version, Payload: raw})
			}
			if err != nil {
				return mapError(err)
			}
		}
		raw, err := json.Marshal(c.State)
		if err != nil {
			return err
		}
		if err = q.SetEvaluationHead(ctx, sqlcgen.SetEvaluationHeadParams{SubmissionID: id, State: raw}); err != nil {
			return err
		}
		if err = r.record(ctx, a, id, "postulacion", change.Action); err != nil {
			return err
		}
		if change.Notify {
			if r.notifier == nil {
				return evaluation.ErrUnavailable
			}
			if err = r.notifier.Enqueue(ctx, id, change.ID, change.Action); err != nil {
				return evaluation.ErrUnavailable
			}
		}
		canonical, canonicalErr := submission.CanonicalContent(change.Record)
		if canonicalErr != nil {
			return canonicalErr
		}
		if err = q.CreateEvaluationCommand(ctx, sqlcgen.CreateEvaluationCommandParams{ID: cmd.Key, ActorID: a.ID, SubmissionID: id, Operation: cmd.Operation, ResourceID: cmd.ResourceID, RequestHash: cmd.Hash, Response: canonical}); err != nil {
			return mapError(err)
		}
		receipt = evaluation.Receipt{Data: canonical, Applied: true}
		return nil
	})
	if err != nil {
		return evaluation.Receipt{}, err
	}
	return receipt, nil
}
func (r *Repository) ListAppeals(ctx context.Context, id uuid.UUID, p domain.Position, limit int) ([]evaluation.Appeal, error) {
	rows, err := r.queries.ListReconsiderations(ctx, sqlcgen.ListReconsiderationsParams{SubmissionID: id, HasCursor: !p.At.IsZero(), BeforeAt: timestamp(p.At), BeforeID: p.ID, RowLimit: int32(limit)})
	if err != nil {
		return nil, err
	}
	items := []evaluation.Appeal{}
	for _, raw := range rows {
		var a evaluation.Appeal
		if err = json.Unmarshal(raw, &a); err != nil {
			return nil, err
		}
		items = append(items, a)
	}
	return items, nil
}
func (r *Repository) ListHistory(ctx context.Context, id uuid.UUID, p domain.Position, limit int) ([]evaluation.HistoryEntry, error) {
	rows, err := r.queries.ListEvaluationHistory(ctx, sqlcgen.ListEvaluationHistoryParams{SubmissionID: id, HasCursor: !p.At.IsZero(), BeforeAt: timestamp(p.At), BeforeID: p.ID, RowLimit: int32(limit)})
	if err != nil {
		return nil, err
	}
	items := []evaluation.HistoryEntry{}
	for _, row := range rows {
		items = append(items, evaluation.HistoryEntry{ID: row.ID, Action: row.Action, At: row.CreatedAt.Time, Data: row.Payload})
	}
	return items, nil
}
func (r *Repository) GetEvidence(ctx context.Context, id uuid.UUID) (*evaluation.Evidence, error) {
	e, err := r.queries.GetEvaluationEvidence(ctx, id)
	if err != nil {
		return nil, mapError(err)
	}
	return &evaluation.Evidence{ID: e.ID, SubmissionID: e.SubmissionID, ApplicantID: e.ApplicantID, ActorID: e.ActorID, ApplicantUpload: e.ApplicantUpload, Name: e.Name, Size: e.SizeBytes, SHA256: e.Sha256, StorageKey: e.StorageKey, CreatedAt: e.CreatedAt.Time, Published: e.Published}, nil
}
func (r *Repository) BeginUpload(ctx context.Context, e *evaluation.Evidence) error {
	return r.queries.BeginEvaluationUpload(ctx, sqlcgen.BeginEvaluationUploadParams{ID: e.ID, SubmissionID: e.SubmissionID, StorageKey: e.StorageKey})
}
func (r *Repository) CompleteUpload(ctx context.Context, id uuid.UUID) error {
	return r.queries.CompleteEvaluationUpload(ctx, id)
}
func (r *Repository) ExpiredUploads(ctx context.Context) ([]evaluation.Evidence, error) {
	rows, err := r.queries.ExpiredEvaluationUploads(ctx)
	if err != nil {
		return nil, err
	}
	items := []evaluation.Evidence{}
	for _, row := range rows {
		items = append(items, evaluation.Evidence{ID: row.ID, StorageKey: row.StorageKey})
	}
	return items, nil
}
