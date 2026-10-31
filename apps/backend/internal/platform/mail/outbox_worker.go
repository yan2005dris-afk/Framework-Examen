package mail

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"log/slog"
	"time"
)

/** Dispatch reclama con lease para varios workers y reintenta sin copiar errores SMTP sensibles. */
func (o *Outbox) Dispatch(ctx context.Context, sender Sender, active func(context.Context, string, string) (bool, error)) error {
	if o == nil || o.queries == nil {
		return ErrQueueUnavailable
	}
	if err := o.queries.ExpireMail(ctx); err != nil {
		return ErrQueueUnavailable
	}
	if o.box == nil || sender == nil {
		return ErrQueueUnavailable
	}
	lease, err := uuid.NewRandom()
	if err != nil {
		return ErrQueueUnavailable
	}
	jobs, err := o.queries.ClaimMail(ctx, pgtype.UUID{Bytes: lease, Valid: true})
	if err != nil {
		return ErrQueueUnavailable
	}
	for _, job := range jobs {
		if err := ctx.Err(); err != nil {
			return err
		}
		status, code := "smtp_accepted", ""
		var message Message
		if len(job.Ciphertext) < o.box.NonceSize() {
			status, code = "failed", "payload_invalid"
		} else {
			raw, err := o.box.Open(nil, job.Ciphertext[:o.box.NonceSize()], job.Ciphertext[o.box.NonceSize():], []byte(job.ID.String()))
			if err != nil || json.Unmarshal(raw, &message) != nil {
				status, code = "failed", "payload_invalid"
			}
		}
		deliveryCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		if status == "smtp_accepted" && message.Suppressed {
			status, code = "superseded", "not_applicable"
		}
		if status == "smtp_accepted" && message.ChallengeID != "" {
			valid := false
			var err error
			if active == nil {
				err = ErrQueueUnavailable
			} else {
				valid, err = active(deliveryCtx, message.ChallengeKey, message.ChallengeID)
			}
			if err != nil {
				status, code = "pending", "security_unavailable"
			} else if !valid {
				status, code = "superseded", "challenge_inactive"
			}
		}
		if status == "smtp_accepted" {
			if !time.Now().Before(job.ExpiresAt.Time) {
				status, code = "expired", "expired"
			} else if err := sender.Send(deliveryCtx, message); err != nil {
				status, code = "pending", "smtp_error"
			}
		}
		cancel()
		if status == "pending" && job.Attempts >= 5 {
			status = "failed"
		}
		// Un worker viejo no puede confirmar un lease nuevo; los demás trabajos quedan reintentables al expirar su lease.
		finishCtx, finishCancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err := o.queries.FinishMail(finishCtx, sqlcgen.FinishMailParams{ID: job.ID, LeaseID: pgtype.UUID{Bytes: lease, Valid: true}, Status: status, ErrorCode: code,
			NextAttemptAt: pgtype.Timestamptz{Time: time.Now().Add(time.Duration(1<<uint(job.Attempts)) * 5 * time.Second), Valid: true}})
		finishCancel()
		if err != nil {
			return ErrQueueUnavailable
		}
	}
	return nil
}

/** Run entrega en segundo plano; apagar el proceso no pierde trabajos confirmados. */
func (o *Outbox) Run(ctx context.Context, sender Sender, active func(context.Context, string, string) (bool, error)) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		if err := o.Dispatch(ctx, sender, active); err != nil && ctx.Err() == nil {
			slog.WarnContext(ctx, "no se pudo procesar la cola de correo")
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
