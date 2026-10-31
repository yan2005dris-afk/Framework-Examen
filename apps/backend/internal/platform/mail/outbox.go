package mail

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/db"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"log/slog"
	"time"
)

var ErrQueueUnavailable = errors.New("no se pudo registrar la notificación")

/** DeliveryState informa aceptación por SMTP, nunca certifica llegada al buzón del destinatario. */
type DeliveryState struct {
	Status    string    `json:"estado"`
	Attempts  int       `json:"intentos"`
	ErrorCode string    `json:"codigoError,omitempty"`
	UpdatedAt time.Time `json:"actualizadoEn"`
}

/** Outbox cifra cargas con AES-GCM y confirma la cola antes de afirmar recepción del envío. */
type Outbox struct {
	queries *sqlcgen.Queries
	box     cipher.AEAD
}

/** NewOutbox requiere una clave estable base64 de 32 bytes; nunca genera una clave efímera de producción. */
func NewOutbox(queries *sqlcgen.Queries, key string) (*Outbox, error) {
	o := &Outbox{queries: queries}
	raw, err := base64.StdEncoding.DecodeString(key)
	if err != nil || len(raw) != 32 {
		return o, ErrQueueUnavailable
	}
	block, err := aes.NewCipher(raw)
	if err != nil {
		return o, ErrQueueUnavailable
	}
	o.box, err = cipher.NewGCM(block)
	if err != nil {
		return o, ErrQueueUnavailable
	}
	return o, nil
}
func (o *Outbox) executor(ctx context.Context) *sqlcgen.Queries {
	if tx, ok := db.Transaction(ctx); ok {
		return o.queries.WithTx(tx)
	}
	return o.queries
}

/** Ready valida esquema y clave sin consultar existencia de una cuenta. */
func (o *Outbox) Ready(ctx context.Context) error {
	if o == nil || o.queries == nil || o.box == nil {
		return ErrQueueUnavailable
	}
	_, err := o.executor(ctx).MailOutboxReady(ctx)
	if err != nil {
		return ErrQueueUnavailable
	}
	return nil
}

/** Send es una aceptación durable en cola, no una llamada SMTP ni confirmación de entrega. */
func (o *Outbox) Send(ctx context.Context, message Message) error {
	if o == nil || o.box == nil || o.queries == nil {
		return ErrQueueUnavailable
	}
	if message.To == "" && !message.Suppressed {
		return ErrEmptyRecipient
	}
	if message.Kind == "" {
		message.Kind = "submission.confirmation"
	}
	if message.ExpiresAt.IsZero() {
		message.ExpiresAt = time.Now().UTC().Add(7 * 24 * time.Hour)
	}
	raw, err := json.Marshal(message)
	if err != nil {
		return ErrQueueUnavailable
	}
	nonce := make([]byte, o.box.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return ErrQueueUnavailable
	}
	id, err := uuid.NewV7()
	if err != nil {
		return ErrQueueUnavailable
	}
	encrypted := o.box.Seal(nonce, nonce, raw, []byte(id.String()))
	// El identificador de deduplicación tampoco contiene destinatario ni OTP legibles.
	sum := sha256.Sum256(raw)
	var resource pgtype.UUID
	if message.ResourceID != "" {
		parsed, err := uuid.Parse(message.ResourceID)
		if err != nil {
			return ErrQueueUnavailable
		}
		resource = pgtype.UUID{Bytes: parsed, Valid: true}
	}
	err = o.executor(ctx).EnqueueMail(ctx, sqlcgen.EnqueueMailParams{ID: id, DedupKey: hex.EncodeToString(sum[:]), Kind: message.Kind, ResourceID: resource,
		Ciphertext: encrypted, ExpiresAt: pgtype.Timestamptz{Time: message.ExpiresAt, Valid: true}})
	if err != nil {
		return ErrQueueUnavailable
	}
	return nil
}

/** State devuelve exclusivamente metadatos; la capa de dominio impone titularidad antes de llamarlo. */
func (o *Outbox) State(ctx context.Context, resource uuid.UUID) (DeliveryState, error) {
	row, err := o.queries.MailStatusByResource(ctx, pgtype.UUID{Bytes: resource, Valid: true})
	if err != nil {
		return DeliveryState{}, err
	}
	return DeliveryState{Status: row.Status, Attempts: int(row.Attempts), ErrorCode: row.LastErrorCode, UpdatedAt: row.UpdatedAt.Time}, nil
}

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
