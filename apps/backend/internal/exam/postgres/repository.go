package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/competition"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/evaluation"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/exam"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/merit"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/db"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"slices"
	"time"
)

type Repository struct {
	q        *sqlcgen.Queries
	tx       *db.TxManager
	audit    audit.WorkflowRecorder
	final    exam.Finalizer
	renderer exam.Renderer
	gate     exam.Gate
}

func NewRepository(q *sqlcgen.Queries, tx *db.TxManager, a audit.WorkflowRecorder, f exam.Finalizer, renderer exam.Renderer, g exam.Gate) *Repository {
	return &Repository{q, tx, a, f, renderer, g}
}
func ts(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }
func mapped(e error) error {
	if errors.Is(e, pgx.ErrNoRows) {
		return exam.ErrNotFound
	}
	var p *pgconn.PgError
	if errors.As(e, &p) && p.Code == "23505" {
		return exam.ErrConflict
	}
	return e
}
func now(ctx context.Context, q *sqlcgen.Queries) (time.Time, error) {
	v, e := q.EvaluationNow(ctx)
	return v.Time, e
}
func admin(ctx context.Context, q *sqlcgen.Queries, a *identity.Applicant) error {
	if !identity.HasPermission(a, identity.PermissionQuestions) {
		return exam.ErrForbidden
	}
	rows, e := q.LockApplicantsForRoleChange(ctx, []uuid.UUID{a.ID})
	if e != nil {
		return e
	}
	if len(rows) != 1 || rows[0].Role != identity.RoleAdministrator {
		return exam.ErrForbidden
	}
	return nil
}
func (r *Repository) record(ctx context.Context, a *identity.Applicant, sub uuid.UUID, action string, t time.Time) error {
	if r.audit == nil {
		return audit.ErrUnavailable
	}
	role := "sistema"
	id := uuid.Nil
	if a != nil {
		role = a.Role
		id = a.ID
	}
	return r.audit.RecordWorkflow(ctx, audit.Event{ActorID: id, Role: role, SubmissionID: sub, ResourceID: sub, ResourceType: "postulacion", Action: action, Result: "permitido", CreatedAt: t})
}
func bank(ctx context.Context, q *sqlcgen.Queries, id uuid.UUID) (*exam.Bank, error) {
	v, e := q.GetExamBank(ctx, id)
	if e != nil {
		return nil, mapped(e)
	}
	b := &exam.Bank{ID: v.ID, CompetitionID: v.CompetitionID, Sector: v.Sector, Version: v.Version, Hash: v.Hash, ActorID: v.ActorID, At: v.CreatedAt.Time}
	if e = json.Unmarshal(v.Definition, &b.Input); e != nil {
		return nil, e
	}
	h, e := evaluation.Digest(b.Input)
	if e != nil || h != b.Hash || exam.ValidateBank(b.Input) != nil {
		return nil, exam.ErrInvalid
	}
	return b, nil
}
func config(ctx context.Context, q *sqlcgen.Queries, id uuid.UUID) (*exam.Config, error) {
	v, e := q.GetExamConfig(ctx, id)
	if e != nil {
		return nil, mapped(e)
	}
	c := &exam.Config{ID: v.ID, BankID: v.BankID, CompetitionID: v.CompetitionID, Sector: v.Sector, Hash: v.Hash, ActorID: v.ActorID, At: v.CreatedAt.Time}
	if e = json.Unmarshal(v.Definition, &c.Rules); e != nil {
		return nil, e
	}
	h, e := evaluation.Digest(c.Rules)
	if e != nil || h != c.Hash {
		return nil, exam.ErrInvalid
	}
	return c, nil
}
func (r *Repository) CreateBank(ctx context.Context, a *identity.Applicant, period uuid.UUID, sector string, input exam.BankInput) (out *exam.Bank, err error) {
	if exam.ValidateBank(input) != nil || !slices.Contains([]string{"afiliados", "jubilados", "empleadores"}, sector) {
		return nil, exam.ErrInvalid
	}
	err = r.tx.WithinTx(ctx, func(tx pgx.Tx) error {
		ctx = db.WithTransaction(ctx, tx)
		q := r.q.WithTx(tx)
		if e := admin(ctx, q, a); e != nil {
			return e
		}
		cal, e := q.GetMeritCalendar(ctx, period)
		if e != nil {
			return mapped(e)
		}
		if !slices.Contains(cal.Sectors, sector) {
			return exam.ErrInvalid
		}
		if e = q.EnsureExamBankStream(ctx, sqlcgen.EnsureExamBankStreamParams{CompetitionID: period, Sector: sector}); e != nil {
			return e
		}
		h, e := q.LockExamBankStream(ctx, sqlcgen.LockExamBankStreamParams{CompetitionID: period, Sector: sector})
		if e != nil {
			return e
		}
		if h.Version != input.PreviousVersion {
			return exam.ErrConflict
		}
		id, e := uuid.NewV7()
		if e != nil {
			return e
		}
		hash, e := evaluation.Digest(input)
		if e != nil {
			return e
		}
		t, e := now(ctx, q)
		if e != nil {
			return e
		}
		raw, _ := json.Marshal(input)
		if e = q.CreateExamBank(ctx, sqlcgen.CreateExamBankParams{ID: id, CompetitionID: period, Sector: sector, Version: h.Version + 1, Definition: raw, Hash: hash, ActorID: a.ID, CreatedAt: ts(t)}); e != nil {
			return e
		}
		if e = q.SetExamBankStream(ctx, sqlcgen.SetExamBankStreamParams{CompetitionID: period, Sector: sector, Version: h.Version + 1}); e != nil {
			return e
		}
		if r.audit == nil {
			return audit.ErrUnavailable
		}
		if e = r.audit.RecordWorkflow(ctx, audit.Event{ActorID: a.ID, Role: a.Role, ResourceID: period, ResourceType: "convocatoria", Action: "banco.versionado", Result: "permitido", CreatedAt: t}); e != nil {
			return e
		}
		out, e = bank(ctx, q, id)
		return e
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
func (r *Repository) Bank(ctx context.Context, id uuid.UUID) (*exam.Bank, error) {
	return bank(ctx, r.q, id)
}
func (r *Repository) Banks(ctx context.Context, id uuid.UUID, sector string, before int64, limit int) ([]exam.Bank, error) {
	rows, e := r.q.ListExamBanks(ctx, sqlcgen.ListExamBanksParams{CompetitionID: id, Sector: sector, BeforeVersion: before, RowLimit: int32(limit)})
	if e != nil {
		return nil, e
	}
	out := []exam.Bank{}
	for _, v := range rows {
		out = append(out, exam.Bank{ID: v.ID, CompetitionID: id, Sector: sector, Version: v.Version, Hash: v.Hash, ActorID: v.ActorID, At: v.CreatedAt.Time})
	}
	return out, nil
}
func (r *Repository) Configure(ctx context.Context, a *identity.Applicant, bankID uuid.UUID, rules exam.ConfigRules) (out *exam.Config, err error) {
	rules.OpensAt = rules.OpensAt.UTC().Truncate(time.Microsecond)
	rules.ClosesAt = rules.ClosesAt.UTC().Truncate(time.Microsecond)
	err = r.tx.WithinTx(ctx, func(tx pgx.Tx) error {
		ctx = db.WithTransaction(ctx, tx)
		q := r.q.WithTx(tx)
		if e := admin(ctx, q, a); e != nil {
			return e
		}
		b, e := bank(ctx, q, bankID)
		if e != nil {
			return e
		}
		if rules.Validate(len(b.Input.Questions)) != nil {
			return exam.ErrInvalid
		}
		c, e := q.GetMeritCalendar(ctx, b.CompetitionID)
		if e != nil {
			return e
		}
		if rules.OpensAt.Before(c.OpensAt.Time) || rules.ClosesAt.After(c.ClosesAt.Time) {
			return exam.ErrInvalid
		}
		id, e := uuid.NewV7()
		if e != nil {
			return e
		}
		hash, e := evaluation.Digest(rules)
		if e != nil {
			return e
		}
		t, e := now(ctx, q)
		if e != nil {
			return e
		}
		raw, _ := json.Marshal(rules)
		if e = q.CreateExamConfig(ctx, sqlcgen.CreateExamConfigParams{ID: id, BankID: b.ID, CompetitionID: b.CompetitionID, Sector: b.Sector, Definition: raw, Hash: hash, ActorID: a.ID, CreatedAt: ts(t)}); e != nil {
			return e
		}
		if r.audit == nil {
			return audit.ErrUnavailable
		}
		if e = r.audit.RecordWorkflow(ctx, audit.Event{ActorID: a.ID, Role: a.Role, ResourceID: b.CompetitionID, ResourceType: "convocatoria", Action: "examen.configurado", Result: "permitido", CreatedAt: t}); e != nil {
			return e
		}
		out, e = config(ctx, q, id)
		return e
	})
	if err != nil {
		return nil, mapped(err)
	}
	return out, nil
}
func source(ctx context.Context, q *sqlcgen.Queries, id uuid.UUID) (*submission.Submission, string, error) {
	v, e := q.GetSubmissionByID(ctx, id)
	if e != nil {
		return nil, "", mapped(e)
	}
	s := &submission.Submission{ID: v.ID, ApplicantID: v.ApplicantID, Code: v.Code, Status: submission.Status(v.Status), Content: v.Content}
	if v.Hash != nil {
		s.Hash = *v.Hash
	}
	if v.SubmittedAt.Valid {
		t := v.SubmittedAt.Time
		s.SubmittedAt = &t
	}
	if v.CompetitionID.Valid {
		id := uuid.UUID(v.CompetitionID.Bytes)
		s.CompetitionID = &id
	}
	if e = submission.ValidateSeal(s); e != nil {
		return nil, "", exam.ErrInvalid
	}
	form, e := submission.DecodeForm(v.Content)
	if e != nil || form.Position == nil || s.CompetitionID == nil {
		return nil, "", exam.ErrInvalid
	}
	return s, form.Position.Sector, nil
}
func lockOwner(ctx context.Context, q *sqlcgen.Queries, a *identity.Applicant, s *submission.Submission, administrative, system bool) error {
	ids := []uuid.UUID{s.ApplicantID}
	if !system {
		if a == nil {
			return exam.ErrForbidden
		}
		if a.ID != s.ApplicantID {
			ids = append(ids, a.ID)
		}
	}
	rows, e := q.LockApplicantsForRoleChange(ctx, ids)
	if e != nil {
		return e
	}
	if system {
		return nil
	}
	for _, v := range rows {
		if v.ID == a.ID && v.Role == a.Role {
			if administrative && identity.HasPermission(a, identity.PermissionGrades) || !administrative && a.ID == s.ApplicantID && a.Role == identity.RoleApplicant {
				return nil
			}
		}
	}
	return exam.ErrForbidden
}
func calendar(ctx context.Context, q *sqlcgen.Queries, c *exam.Config, sector string, t time.Time, active bool) error {
	v, e := q.GetMeritCalendar(ctx, c.CompetitionID)
	if e != nil {
		return e
	}
	if !active {
		return nil
	}
	if c.Sector != sector || !v.Approved || !v.Published || !slices.Contains(v.Sectors, sector) || t.Before(c.Rules.OpensAt) || !t.Before(c.Rules.ClosesAt) || t.Before(v.OpensAt.Time) || !t.Before(v.ClosesAt.Time) {
		return exam.ErrPhase
	}
	var phases []competition.Phase
	if json.Unmarshal(v.Phases, &phases) != nil {
		return exam.ErrPhase
	}
	for _, p := range phases {
		if p.Code == "oposicion_teorica" && p.Enabled && p.Published && !t.Before(p.OpensAt) && t.Before(p.ClosesAt) {
			return nil
		}
	}
	return exam.ErrPhase
}
func meritSource(ctx context.Context, q *sqlcgen.Queries, s *submission.Submission, expected uuid.UUID) error {
	h, e := q.GetMeritCase(ctx, s.ID)
	if e != nil {
		return exam.ErrPhase
	}
	if !h.PublicationID.Valid || uuid.UUID(h.PublicationID.Bytes) != expected {
		return exam.ErrConflict
	}
	raw, e := q.GetMeritPublication(ctx, sqlcgen.GetMeritPublicationParams{ID: expected, SubmissionID: s.ID})
	if e != nil {
		return e
	}
	var m merit.Publication
	if json.Unmarshal(raw, &m) != nil || m.SubmissionID != s.ID || m.SnapshotHash != s.Hash || m.Decision != "avanza" {
		return exam.ErrPhase
	}
	return nil
}
func (r *Repository) Enroll(ctx context.Context, a *identity.Applicant, id uuid.UUID, input exam.EnrollmentInput) error {
	return r.tx.WithinTx(ctx, func(tx pgx.Tx) error {
		ctx = db.WithTransaction(ctx, tx)
		q := r.q.WithTx(tx)
		s, sector, e := source(ctx, q, id)
		if e != nil {
			return e
		}
		if e = lockOwner(ctx, q, a, s, true, false); e != nil {
			return e
		}
		if e = admin(ctx, q, a); e != nil {
			return e
		}
		c, e := config(ctx, q, input.ConfigID)
		if e != nil {
			return e
		}
		if c.CompetitionID != *s.CompetitionID || c.Sector != sector {
			return exam.ErrInvalid
		}
		if e = calendar(ctx, q, c, sector, time.Time{}, false); e != nil {
			return e
		}
		if e = meritSource(ctx, q, s, input.MeritID); e != nil {
			return e
		}
		prior, e := q.GetExamEnrollment(ctx, id)
		if e == nil {
			if prior.ConfigID == input.ConfigID && prior.MeritID == input.MeritID {
				return nil
			}
			return exam.ErrConflict
		}
		if !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		t, e := now(ctx, q)
		if e != nil {
			return e
		}
		if !t.Before(c.Rules.ClosesAt) {
			return exam.ErrPhase
		}
		if e = q.CreateExamEnrollment(ctx, sqlcgen.CreateExamEnrollmentParams{SubmissionID: id, ConfigID: input.ConfigID, MeritID: input.MeritID, ActorID: a.ID, CreatedAt: ts(t)}); e != nil {
			return mapped(e)
		}
		return r.record(ctx, a, id, "evaluacion.habilitada", t)
	})
}
func decodeAttempt(v sqlcgen.ExamAttempt) (*exam.Attempt, error) {
	var a exam.Attempt
	if e := json.Unmarshal(v.Payload, &a); e != nil {
		return nil, e
	}
	if a.ID != v.ID || a.SubmissionID != v.SubmissionID || a.Version != v.Version || a.Status != v.Status || !a.ExpiresAt.Equal(v.ExpiresAt.Time) {
		return nil, exam.ErrInvalid
	}
	return &a, nil
}
func save(ctx context.Context, q *sqlcgen.Queries, a *exam.Attempt, t time.Time, create bool, assignment string) error {
	raw, e := json.Marshal(a)
	if e != nil {
		return e
	}
	if create {
		e = q.CreateExamAttempt(ctx, sqlcgen.CreateExamAttemptParams{ID: a.ID, SubmissionID: a.SubmissionID, ConfigID: a.Config.ID, AssignmentHash: assignment, Payload: raw, Version: a.Version, Status: a.Status, ExpiresAt: ts(a.ExpiresAt), CreatedAt: ts(a.StartedAt)})
	} else {
		e = q.UpdateExamAttempt(ctx, sqlcgen.UpdateExamAttemptParams{ID: a.ID, Payload: raw, Version: a.Version, Status: a.Status})
	}
	if e != nil {
		return mapped(e)
	}
	event, e := json.Marshal(struct {
		Version int64              `json:"version"`
		Status  string             `json:"estado"`
		Answers map[string]string  `json:"respuestas"`
		Result  *evaluation.Result `json:"resultado,omitempty"`
	}{a.Version, a.Status, a.Answers, a.Result})
	if e != nil {
		return e
	}
	return q.CreateExamEvent(ctx, sqlcgen.CreateExamEventParams{AttemptID: a.ID, Version: a.Version, Payload: event, CreatedAt: ts(t)})
}
func (r *Repository) Start(ctx context.Context, a *identity.Applicant, id uuid.UUID) (out *exam.Attempt, clock time.Time, err error) {
	err = r.tx.WithinTx(ctx, func(tx pgx.Tx) error {
		ctx = db.WithTransaction(ctx, tx)
		q := r.q.WithTx(tx)
		s, sector, e := source(ctx, q, id)
		if e != nil {
			return e
		}
		if e = lockOwner(ctx, q, a, s, false, false); e != nil {
			return e
		}
		en, e := q.GetExamEnrollment(ctx, id)
		if e != nil {
			return mapped(e)
		}
		c, e := config(ctx, q, en.ConfigID)
		if e != nil {
			return e
		}
		if e = calendar(ctx, q, c, sector, time.Time{}, false); e != nil {
			return e
		}
		existing, e := q.LockExamAttempt(ctx, id)
		if e == nil {
			out, e = decodeAttempt(existing)
			if e != nil {
				return e
			}
			clock, e = now(ctx, q)
			if e != nil {
				return e
			}
			if out.Status == "active" && !clock.Before(out.ExpiresAt) {
				return r.close(ctx, q, out, clock, "vencimiento")
			}
			return nil
		}
		if !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		if e = meritSource(ctx, q, s, en.MeritID); e != nil {
			return e
		}
		if r.gate != nil {
			if e = r.gate.Allow(ctx, id, "datos"); e != nil {
				return e
			}
		}
		if e = q.LockExamConfig(ctx, c.ID); e != nil {
			return e
		}
		clock, e = now(ctx, q)
		if e != nil {
			return e
		}
		if e = calendar(ctx, q, c, sector, clock, true); e != nil {
			return e
		}
		if clock.Add(time.Duration(c.Rules.DurationSeconds) * time.Second).After(c.Rules.ClosesAt) {
			return exam.ErrPhase
		}
		phaseCalendar, e := q.GetMeritCalendar(ctx, c.CompetitionID)
		if e != nil {
			return e
		}
		var phases []competition.Phase
		if json.Unmarshal(phaseCalendar.Phases, &phases) != nil {
			return exam.ErrPhase
		}
		for _, phase := range phases {
			if phase.Code == "oposicion_teorica" && clock.Add(time.Duration(c.Rules.DurationSeconds)*time.Second).After(phase.ClosesAt) {
				return exam.ErrPhase
			}
		}
		b, e := bank(ctx, q, c.BankID)
		if e != nil {
			return e
		}
		usedRows, e := q.ExamAssignments(ctx, c.ID)
		if e != nil {
			return e
		}
		used := map[string]bool{}
		for _, h := range usedRows {
			used[h] = true
		}
		questions, hash, e := exam.Assign(b.Input.Questions, c.Rules.Count, used)
		if e != nil {
			return e
		}
		attemptID, e := uuid.NewV7()
		if e != nil {
			return e
		}
		out = &exam.Attempt{ID: attemptID, SubmissionID: id, Config: *c, SnapshotHash: s.Hash, BankHash: b.Hash, MeritID: en.MeritID, Questions: questions, Answers: map[string]string{}, Version: 1, Status: "active", StartedAt: clock, ExpiresAt: clock.Add(time.Duration(c.Rules.DurationSeconds) * time.Second)}
		if e = save(ctx, q, out, clock, true, hash); e != nil {
			return e
		}
		return r.record(ctx, a, id, "evaluacion.iniciada", clock)
	})
	if err != nil {
		return nil, clock, err
	}
	return out, clock, nil
}

/** close guarda PDF, nota, evento y notificación antes del commit; un fallo revierte todo y queda reintentable. */
func (r *Repository) close(ctx context.Context, q *sqlcgen.Queries, a *exam.Attempt, t time.Time, reason string) error {
	if a.Status == "finished" {
		return nil
	}
	result, count, e := exam.Grade(a, t, reason)
	if e != nil {
		return e
	}
	a.Version++
	a.Status = "finished"
	a.FinishedAt = &t
	a.Result = result
	a.CorrectCount = count
	a.Reason = reason
	if r.renderer == nil || r.final == nil {
		return exam.ErrInvalid
	}
	pdf, e := r.renderer.Render(ctx, a)
	if e != nil {
		return e
	}
	sum := sha256.Sum256(pdf)
	if e = q.CreateExamReport(ctx, sqlcgen.CreateExamReportParams{AttemptID: a.ID, SubmissionID: a.SubmissionID, ResultID: result.ID, Pdf: pdf, Sha256: hex.EncodeToString(sum[:]), CreatedAt: ts(t)}); e != nil {
		return e
	}
	if e = r.final.RecordFinalTheory(ctx, *result); e != nil {
		return e
	}
	return save(ctx, q, a, t, false, "")
}
func (r *Repository) mutate(ctx context.Context, a *identity.Applicant, id uuid.UUID, administrative, system bool, operation string, input *exam.AnswerInput) (out *exam.Attempt, clock time.Time, err error) {
	late := false
	err = r.tx.WithinTx(ctx, func(tx pgx.Tx) error {
		ctx = db.WithTransaction(ctx, tx)
		q := r.q.WithTx(tx)
		s, sector, e := source(ctx, q, id)
		if e != nil {
			return e
		}
		if e = lockOwner(ctx, q, a, s, administrative, system); e != nil {
			return e
		}
		en, e := q.GetExamEnrollment(ctx, id)
		if e != nil {
			return mapped(e)
		}
		c, e := config(ctx, q, en.ConfigID)
		if e != nil {
			return e
		}
		if e = calendar(ctx, q, c, sector, time.Time{}, false); e != nil {
			return e
		}
		row, e := q.LockExamAttempt(ctx, id)
		if e != nil {
			return mapped(e)
		}
		out, e = decodeAttempt(row)
		if e != nil {
			return e
		}
		clock, e = now(ctx, q)
		if e != nil {
			return e
		}
		if out.Status == "active" && !clock.Before(out.ExpiresAt) {
			if e = r.close(ctx, q, out, clock, "vencimiento"); e != nil {
				return e
			}
			late = operation == "answer"
			return nil
		}
		if operation == "finish" {
			return r.close(ctx, q, out, clock, "finalizacion_manual")
		}
		if operation != "answer" {
			if out.Status == "active" && !system {
				if e = calendar(ctx, q, c, sector, clock, true); e != nil {
					return e
				}
				if e = meritSource(ctx, q, s, out.MeritID); e != nil {
					return e
				}
				if r.gate != nil {
					if e = r.gate.Allow(ctx, id, "datos"); e != nil {
						return e
					}
				}
			}
			return nil
		}
		if out.Status != "active" {
			late = true
			return nil
		}
		if r.gate != nil {
			if e = r.gate.Allow(ctx, id, "datos"); e != nil {
				return e
			}
		}
		if e = calendar(ctx, q, c, sector, clock, true); e != nil {
			return e
		}
		if e = meritSource(ctx, q, s, out.MeritID); e != nil {
			return e
		}
		found := false
		for _, question := range out.Questions {
			if question.ID == input.QuestionID {
				for _, option := range question.Options {
					if option.ID == input.OptionID {
						found = true
					}
				}
			}
		}
		if !found {
			return exam.ErrInvalid
		}
		if out.Answers[input.QuestionID] == input.OptionID {
			return nil
		}
		if out.Version != input.PreviousVersion {
			return exam.ErrConflict
		}
		out.Answers[input.QuestionID] = input.OptionID
		out.Version++
		return save(ctx, q, out, clock, false, "")
	})
	if err != nil {
		return nil, clock, err
	}
	if late {
		return out, clock, exam.ErrExpired
	}
	return out, clock, nil
}
func (r *Repository) Read(ctx context.Context, a *identity.Applicant, id uuid.UUID, admin bool) (*exam.Attempt, time.Time, error) {
	return r.mutate(ctx, a, id, admin, false, "read", nil)
}
func (r *Repository) Answer(ctx context.Context, a *identity.Applicant, id uuid.UUID, input exam.AnswerInput) (*exam.Attempt, time.Time, error) {
	return r.mutate(ctx, a, id, false, false, "answer", &input)
}
func (r *Repository) Finish(ctx context.Context, a *identity.Applicant, id uuid.UUID, system bool) (*exam.Attempt, time.Time, error) {
	return r.mutate(ctx, a, id, false, system, "finish", nil)
}
func (r *Repository) Report(ctx context.Context, a *identity.Applicant, id uuid.UUID, admin bool) (exam.Report, error) {
	attempt, _, e := r.Read(ctx, a, id, admin)
	if e != nil {
		return exam.Report{}, e
	}
	if attempt.Status != "finished" {
		return exam.Report{}, exam.ErrConflict
	}
	if r.gate != nil {
		if e = r.gate.Allow(ctx, id, "reportes"); e != nil {
			return exam.Report{}, e
		}
	}
	v, e := r.q.GetExamReport(ctx, id)
	if e != nil {
		return exam.Report{}, mapped(e)
	}
	return exam.Report{PDF: v.Pdf, Hash: v.Sha256}, nil
}
func (r *Repository) Expire(ctx context.Context) error {
	rows, e := r.q.DueExamAttempts(ctx)
	if e != nil {
		return e
	}
	var first error
	for _, id := range rows {
		if _, _, e = r.Finish(ctx, nil, id, true); e != nil && first == nil {
			first = e
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
	return first
}
