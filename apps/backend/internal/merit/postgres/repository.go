package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/competition"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/merit"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/db"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
)

type Repository struct {
	queries  *sqlcgen.Queries
	tx       *db.TxManager
	audit    audit.WorkflowRecorder
	notifier merit.Notifier
}

func NewRepository(q *sqlcgen.Queries, tx *db.TxManager, a audit.WorkflowRecorder) *Repository {
	return &Repository{queries: q, tx: tx, audit: a}
}

func (r *Repository) WithNotifier(n merit.Notifier) *Repository { r.notifier = n; return r }

/** verifyActor serializa la revocación de permisos con cada escritura de méritos. */
func verifyActor(ctx context.Context, q *sqlcgen.Queries, a *identity.Applicant, admin bool) error {
	if a == nil {
		return merit.ErrForbidden
	}
	rows, err := q.LockApplicantsForRoleChange(ctx, []uuid.UUID{a.ID})
	if err != nil {
		return err
	}
	if len(rows) != 1 || rows[0].Role != a.Role || !identity.HasPermission(a, identity.PermissionGrades) || admin && rows[0].Role != "administrador" {
		return merit.ErrForbidden
	}
	return nil
}
func (r *Repository) record(ctx context.Context, a *identity.Applicant, id uuid.UUID, kind, action string) error {
	if r.audit == nil {
		return audit.ErrUnavailable
	}
	submissionID := uuid.Nil
	if kind == "postulacion" {
		submissionID = id
	}
	return r.audit.RecordWorkflow(ctx, audit.Event{SubmissionID: submissionID, ActorID: a.ID, Role: a.Role, ResourceID: id, ResourceType: kind, Action: action, Result: "permitido", CreatedAt: time.Now().UTC()})
}
func decodeRules(row sqlcgen.MeritRuleSet) (*merit.RuleSet, error) {
	v := &merit.RuleSet{ID: row.ID, CompetitionID: row.CompetitionID, Sector: row.Sector, Version: row.Version, Hash: row.DefinitionHash, Approved: row.Approved, ActorID: row.ActorID, CreatedAt: row.CreatedAt.Time}
	if row.ApprovedBy.Valid {
		id := uuid.UUID(row.ApprovedBy.Bytes)
		v.ApprovedBy = &id
	}
	if row.ApprovedAt.Valid {
		t := row.ApprovedAt.Time
		v.ApprovedAt = &t
	}
	if row.ApprovalReference != nil {
		v.ApprovalReference = *row.ApprovalReference
	}
	err := json.Unmarshal(row.Definition, &v.Definition)
	return v, err
}
func getRules(ctx context.Context, q *sqlcgen.Queries, id uuid.UUID) (*merit.RuleSet, error) {
	row, err := q.GetMeritRules(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, merit.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return decodeRules(row)
}
func (r *Repository) GetRules(ctx context.Context, id uuid.UUID) (*merit.RuleSet, error) {
	return getRules(ctx, r.queries, id)
}
func (r *Repository) ListRules(ctx context.Context, period uuid.UUID, sector string, before int64, limit int) ([]merit.RuleSet, error) {
	rows, err := r.queries.ListMeritRules(ctx, sqlcgen.ListMeritRulesParams{CompetitionID: period, Sector: sector, BeforeVersion: before, RowLimit: int32(limit)})
	if err != nil {
		return nil, err
	}
	items := make([]merit.RuleSet, 0, len(rows))
	for _, row := range rows {
		v, err := decodeRules(row)
		if err != nil {
			return nil, err
		}
		items = append(items, *v)
	}
	return items, nil
}
func (r *Repository) CreateRules(ctx context.Context, a *identity.Applicant, v *merit.RuleSet) error {
	return r.tx.WithinTx(ctx, func(tx pgx.Tx) error {
		ctx = db.WithTransaction(ctx, tx)
		q := r.queries.WithTx(tx)
		if err := verifyActor(ctx, q, a, false); err != nil {
			return err
		}
		c, err := q.GetMeritCalendar(ctx, v.CompetitionID)
		if errors.Is(err, pgx.ErrNoRows) {
			return merit.ErrNotFound
		}
		if err != nil {
			return err
		}
		if !slices.Contains(c.Sectors, v.Sector) {
			return merit.ErrInvalid
		}
		if err = q.EnsureMeritRuleStream(ctx, sqlcgen.EnsureMeritRuleStreamParams{CompetitionID: v.CompetitionID, Sector: v.Sector}); err != nil {
			return err
		}
		head, err := q.LockMeritRuleStream(ctx, sqlcgen.LockMeritRuleStreamParams{CompetitionID: v.CompetitionID, Sector: v.Sector})
		if err != nil {
			return err
		}
		v.Version = head.Version + 1
		raw, err := json.Marshal(v.Definition)
		if err != nil {
			return err
		}
		if err = q.CreateMeritRules(ctx, sqlcgen.CreateMeritRulesParams{ID: v.ID, CompetitionID: v.CompetitionID, Sector: v.Sector, Version: v.Version, Definition: raw, DefinitionHash: v.Hash, ActorID: a.ID, CreatedAt: pgtype.Timestamptz{Time: v.CreatedAt, Valid: true}}); err != nil {
			return err
		}
		if err = q.SetMeritRuleStream(ctx, sqlcgen.SetMeritRuleStreamParams{CompetitionID: v.CompetitionID, Sector: v.Sector, Version: v.Version, ActiveRuleID: head.ActiveRuleID}); err != nil {
			return err
		}
		return r.record(ctx, a, v.CompetitionID, "convocatoria", "meritos.reglas_creadas")
	})
}
func (r *Repository) ApproveRules(ctx context.Context, a *identity.Applicant, id uuid.UUID, reference string) (result *merit.RuleSet, err error) {
	err = r.tx.WithinTx(ctx, func(tx pgx.Tx) error {
		ctx = db.WithTransaction(ctx, tx)
		q := r.queries.WithTx(tx)
		if err := verifyActor(ctx, q, a, true); err != nil {
			return err
		}
		v, err := getRules(ctx, q, id)
		if err != nil {
			return err
		}
		head, err := q.LockMeritRuleStream(ctx, sqlcgen.LockMeritRuleStreamParams{CompetitionID: v.CompetitionID, Sector: v.Sector})
		if err != nil {
			return err
		}
		if v.Approved {
			return merit.ErrConflict
		}
		if err = v.Validate(); err != nil {
			return err
		}
		if head.ActiveRuleID.Valid {
			active, err := getRules(ctx, q, uuid.UUID(head.ActiveRuleID.Bytes))
			if err != nil {
				return err
			}
			if active.Version >= v.Version {
				return merit.ErrConflict
			}
		}
		if err = q.ApproveMeritRules(ctx, sqlcgen.ApproveMeritRulesParams{ID: id, ApprovalReference: &reference, ApprovedBy: pgtype.UUID{Bytes: a.ID, Valid: true}}); err != nil {
			return err
		}
		if err = q.SetMeritRuleStream(ctx, sqlcgen.SetMeritRuleStreamParams{CompetitionID: v.CompetitionID, Sector: v.Sector, Version: head.Version, ActiveRuleID: pgtype.UUID{Bytes: id, Valid: true}}); err != nil {
			return err
		}
		if err = r.record(ctx, a, v.CompetitionID, "convocatoria", "meritos.reglas_aprobadas"); err != nil {
			return err
		}
		result, err = getRules(ctx, q, id)
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
func loadSource(ctx context.Context, q *sqlcgen.Queries, id uuid.UUID) (*merit.Case, string, error) {
	row, err := q.GetSubmissionByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", merit.ErrNotFound
	}
	if err != nil {
		return nil, "", err
	}
	item := &submission.Submission{ID: row.ID, ApplicantID: row.ApplicantID, Code: row.Code, Status: submission.Status(row.Status), Content: row.Content, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
	if row.Hash != nil {
		item.Hash = *row.Hash
	}
	if row.SubmittedAt.Valid {
		t := row.SubmittedAt.Time
		item.SubmittedAt = &t
	}
	if row.CompetitionID.Valid {
		id := uuid.UUID(row.CompetitionID.Bytes)
		item.CompetitionID = &id
	}
	if err = submission.ValidateSeal(item); err != nil {
		return nil, "", err
	}
	form, err := submission.DecodeForm(item.Content)
	if err != nil || form.Position == nil {
		return nil, "", merit.ErrEvidence
	}
	c := &merit.Case{Snapshot: item, Documents: []merit.Document{}}
	docs, err := q.ListDocuments(ctx, id)
	if err != nil {
		return nil, "", err
	}
	for _, d := range docs {
		c.Documents = append(c.Documents, merit.Document{ID: d.ID.String(), Section: int(d.Section), RecordID: d.RecordID, SHA256: d.Sha256})
	}
	return c, form.Position.Sector, nil
}
func loadHead(ctx context.Context, q *sqlcgen.Queries, c *merit.Case, h sqlcgen.MeritCaseStream) error {
	if h.ReviewID.Valid {
		raw, err := q.GetMeritReview(ctx, sqlcgen.GetMeritReviewParams{ID: uuid.UUID(h.ReviewID.Bytes), SubmissionID: h.SubmissionID})
		if err != nil {
			return err
		}
		if err = json.Unmarshal(raw, &c.Review); err != nil {
			return err
		}
		if c.Review == nil || c.Review.ID != uuid.UUID(h.ReviewID.Bytes) || c.Review.SubmissionID != h.SubmissionID {
			return merit.ErrEvidence
		}
	}
	if h.CalculationID.Valid {
		raw, err := q.GetMeritCalculation(ctx, sqlcgen.GetMeritCalculationParams{ID: uuid.UUID(h.CalculationID.Bytes), SubmissionID: h.SubmissionID})
		if err != nil {
			return err
		}
		if err = json.Unmarshal(raw, &c.Calculation); err != nil {
			return err
		}
		if c.Calculation == nil || c.Calculation.ID != uuid.UUID(h.CalculationID.Bytes) || c.Calculation.SubmissionID != h.SubmissionID {
			return merit.ErrEvidence
		}
	}
	if h.PublicationID.Valid {
		raw, err := q.GetMeritPublication(ctx, sqlcgen.GetMeritPublicationParams{ID: uuid.UUID(h.PublicationID.Bytes), SubmissionID: h.SubmissionID})
		if err != nil {
			return err
		}
		if err = json.Unmarshal(raw, &c.Publication); err != nil {
			return err
		}
		if c.Publication == nil || c.Publication.ID != uuid.UUID(h.PublicationID.Bytes) || c.Publication.SubmissionID != h.SubmissionID {
			return merit.ErrEvidence
		}
	}
	return nil
}
func activeRules(ctx context.Context, q *sqlcgen.Queries, c *merit.Case, sector string) error {
	if c.Snapshot.CompetitionID == nil {
		return nil
	}
	head, err := q.GetMeritRuleStream(ctx, sqlcgen.GetMeritRuleStreamParams{CompetitionID: *c.Snapshot.CompetitionID, Sector: sector})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if head.ActiveRuleID.Valid {
		c.ActiveRules, err = getRules(ctx, q, uuid.UUID(head.ActiveRuleID.Bytes))
	}
	return err
}
func (r *Repository) ReadCase(ctx context.Context, id uuid.UUID) (result *merit.Case, err error) {
	err = r.tx.WithinTx(ctx, func(tx pgx.Tx) error {
		q := r.queries.WithTx(tx)
		c, sector, err := loadSource(ctx, q, id)
		if err != nil {
			return err
		}
		if err = activeRules(ctx, q, c, sector); err != nil {
			return err
		}
		head, err := q.GetMeritCase(ctx, id)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if err == nil {
			if err = loadHead(ctx, q, c, head); err != nil {
				return err
			}
		}
		result = c
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
func checkPhase(row sqlcgen.GetMeritCalendarRow, sector string) error {
	now := row.CheckedAt.Time
	if !row.Approved || !row.Published || !slices.Contains(row.Sectors, sector) || now.Before(row.OpensAt.Time) || !now.Before(row.ClosesAt.Time) {
		return merit.ErrPhase
	}
	var phases []competition.Phase
	if json.Unmarshal(row.Phases, &phases) != nil {
		return merit.ErrPhase
	}
	for _, p := range phases {
		if p.Code == "meritos" && p.Enabled && p.Published && !now.Before(p.OpensAt) && now.Before(p.ClosesAt) {
			return nil
		}
	}
	return merit.ErrPhase
}

/** MutateCase guarda una única revisión, cálculo o publicación junto a su evento auditable. */
func (r *Repository) MutateCase(ctx context.Context, a *identity.Applicant, id uuid.UUID, change func(*merit.Case) error) (result *merit.Case, err error) {
	err = r.tx.WithinTx(ctx, func(tx pgx.Tx) error {
		ctx = db.WithTransaction(ctx, tx)
		q := r.queries.WithTx(tx)
		if err := verifyActor(ctx, q, a, false); err != nil {
			return err
		}
		c, sector, err := loadSource(ctx, q, id)
		if err != nil {
			return err
		}
		if c.Snapshot.CompetitionID == nil {
			return merit.ErrPhase
		}
		calendar, err := q.GetMeritCalendar(ctx, *c.Snapshot.CompetitionID)
		if err != nil {
			return err
		}
		if err = checkPhase(calendar, sector); err != nil {
			return err
		}
		if err = activeRules(ctx, q, c, sector); err != nil {
			return err
		}
		if err = q.EnsureMeritCase(ctx, id); err != nil {
			return err
		}
		head, err := q.LockMeritCase(ctx, id)
		if err != nil {
			return err
		}
		if err = loadHead(ctx, q, c, head); err != nil {
			return err
		}
		previousReview, previousCalc, previousPub := c.Review, c.Calculation, c.Publication
		if err = change(c); err != nil {
			return err
		}
		action := ""
		changes := 0
		if c.Review != previousReview {
			changes++
			action = "meritos.revisados"
			raw, err := json.Marshal(c.Review)
			if err != nil {
				return err
			}
			if c.Review == nil {
				return merit.ErrInvalid
			}
			if err = q.CreateMeritReview(ctx, sqlcgen.CreateMeritReviewParams{ID: c.Review.ID, SubmissionID: id, Version: c.Review.Version, RuleSetID: c.Review.Input.RuleSetID, Payload: raw}); err != nil {
				return err
			}
			head.ReviewID = pgtype.UUID{Bytes: c.Review.ID, Valid: true}
		}
		if c.Calculation != previousCalc {
			changes++
			action = "meritos.calificados"
			raw, err := json.Marshal(c.Calculation)
			if err != nil {
				return err
			}
			if c.Calculation == nil {
				return merit.ErrInvalid
			}
			if err = q.CreateMeritCalculation(ctx, sqlcgen.CreateMeritCalculationParams{ID: c.Calculation.ID, SubmissionID: id, Version: c.Calculation.Version, ReviewID: c.Calculation.ReviewID, RuleSetID: c.Calculation.RuleSetID, Payload: raw}); err != nil {
				return err
			}
			head.CalculationID = pgtype.UUID{Bytes: c.Calculation.ID, Valid: true}
		}
		if c.Publication != previousPub {
			changes++
			action = "resultado.publicado"
			if a.Role != "administrador" {
				return merit.ErrForbidden
			}
			raw, err := json.Marshal(c.Publication)
			if err != nil {
				return err
			}
			if c.Publication == nil {
				return merit.ErrInvalid
			}
			sup := pgtype.UUID{}
			if c.Publication.SupersedesID != nil {
				sup = pgtype.UUID{Bytes: *c.Publication.SupersedesID, Valid: true}
			}
			if err = q.CreateMeritPublication(ctx, sqlcgen.CreateMeritPublicationParams{ID: c.Publication.ID, SubmissionID: id, Version: c.Publication.Version, CalculationID: c.Publication.CalculationID, SupersedesID: sup, Decision: c.Publication.Decision, Payload: raw}); err != nil {
				return err
			}
			head.PublicationID = pgtype.UUID{Bytes: c.Publication.ID, Valid: true}
		}
		if changes != 1 {
			return merit.ErrInvalid
		}
		if err = q.SetMeritCase(ctx, sqlcgen.SetMeritCaseParams{SubmissionID: id, ReviewID: head.ReviewID, CalculationID: head.CalculationID, PublicationID: head.PublicationID}); err != nil {
			return err
		}
		if err = r.record(ctx, a, id, "postulacion", action); err != nil {
			return err
		}
		if c.Publication != previousPub {
			if r.notifier == nil {
				return errors.New("cola de notificaciones de méritos no configurada")
			}
			if err = r.notifier.Enqueue(ctx, id, c.Publication.ID, "resultado.publicado"); err != nil {
				return err
			}
		}
		result = c
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
