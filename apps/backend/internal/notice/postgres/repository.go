package postgres

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/notice"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/db"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/shared/domain"
	"strings"
	"time"
)

type Repository struct {
	q       *sqlcgen.Queries
	tx      *db.TxManager
	auditor audit.WorkflowRecorder
}

func NewRepository(q *sqlcgen.Queries, tx *db.TxManager, a audit.WorkflowRecorder) *Repository {
	return &Repository{q, tx, a}
}
func mapped(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return notice.ErrNotFound
	}
	return err
}
func ts(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }
func (r *Repository) Owner(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	s, err := r.q.GetSubmissionByID(ctx, id)
	if err != nil {
		return uuid.Nil, mapped(err)
	}
	if s.Status != "submitted" {
		return uuid.Nil, notice.ErrForbidden
	}
	return s.ApplicantID, nil
}
func (r *Repository) List(ctx context.Context, id uuid.UUID, p domain.Position, limit int) ([]notice.Delivery, error) {
	rows, err := r.q.ListSubmissionNotices(ctx, sqlcgen.ListSubmissionNoticesParams{SubmissionID: id, HasCursor: !p.At.IsZero(), BeforeAt: ts(p.At), BeforeID: p.ID, RowLimit: int32(limit)})
	if err != nil {
		return nil, err
	}
	items := []notice.Delivery{}
	for _, v := range rows {
		items = append(items, notice.Delivery{ID: v.ID, Action: v.Action, Template: v.TemplateVersion, Status: v.Status, Attempts: int(v.Attempts), Total: int(v.TotalAttempts), Retries: int(v.RetryCount), ErrorCode: v.LastErrorCode, CreatedAt: v.CreatedAt.Time, UpdatedAt: v.UpdatedAt.Time})
	}
	return items, nil
}
func (r *Repository) History(ctx context.Context, id, mailID uuid.UUID, p domain.Position, limit int) (items []notice.Event, err error) {
	err = r.tx.WithinTx(ctx, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		if _, err := q.LockSubmissionMail(ctx, sqlcgen.LockSubmissionMailParams{SubmissionID: id, MailID: mailID}); err != nil {
			return mapped(err)
		}
		rows, err := q.ListMailDeliveryEvents(ctx, sqlcgen.ListMailDeliveryEventsParams{MailID: mailID, HasCursor: !p.At.IsZero(), BeforeAt: ts(p.At), BeforeID: p.ID, RowLimit: int32(limit)})
		if err != nil {
			return err
		}
		items = []notice.Event{}
		for _, v := range rows {
			items = append(items, notice.Event{ID: v.ID, Status: v.Status, ErrorCode: v.ErrorCode, Attempt: int(v.Attempt), Cycle: int(v.Cycle), At: v.CreatedAt.Time})
		}
		return nil
	})
	return items, err
}

/** Retry conserva el identificador lógico del correo y rechaza OTP, entregados, vencidos y leases activos. */
func (r *Repository) Retry(ctx context.Context, a *identity.Applicant, id, mailID uuid.UUID, input notice.RetryInput) error {
	return r.tx.WithinTx(ctx, func(tx pgx.Tx) error {
		ctx = db.WithTransaction(ctx, tx)
		q := r.q.WithTx(tx)
		actors, err := q.LockApplicantsForRoleChange(ctx, []uuid.UUID{a.ID})
		if err != nil {
			return err
		}
		if len(actors) != 1 || actors[0].Role != identity.RoleAdministrator || a.Role != actors[0].Role {
			return notice.ErrForbidden
		}
		m, err := q.LockSubmissionMail(ctx, sqlcgen.LockSubmissionMailParams{SubmissionID: id, MailID: mailID})
		if err != nil {
			return mapped(err)
		}
		now, err := q.EvaluationNow(ctx)
		if err != nil {
			return err
		}
		if m.Status != "failed" || !now.Time.Before(m.ExpiresAt.Time) || len(m.Ciphertext) == 0 || int(m.RetryCount) != input.PreviousRetries || m.RetryCount >= 3 || m.LastErrorCode == "payload_invalid" || strings.HasPrefix(m.Kind, "otp.") {
			return notice.ErrConflict
		}
		if err = q.RetryStageMail(ctx, mailID); err != nil {
			return err
		}
		reason := strings.TrimSpace(input.Reason)
		if reason == "" {
			return notice.ErrInvalid
		}
		if err = q.RecordMailRetry(ctx, sqlcgen.RecordMailRetryParams{MailID: mailID, Cycle: m.RetryCount + 1, ActorID: pgtype.UUID{Bytes: a.ID, Valid: true}, Reason: &reason}); err != nil {
			return err
		}
		if r.auditor == nil {
			return audit.ErrUnavailable
		}
		return r.auditor.RecordWorkflow(ctx, audit.Event{ActorID: a.ID, Role: a.Role, Action: "notificacion.reintentada", SubmissionID: id, ResourceID: mailID, ResourceType: "notificacion", Result: "permitido", CreatedAt: now.Time})
	})
}
