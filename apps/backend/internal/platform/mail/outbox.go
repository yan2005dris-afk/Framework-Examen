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
