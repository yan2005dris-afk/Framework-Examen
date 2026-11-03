package mail

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"strings"
	"testing"
	"time"
)

type captureOutbox struct {
	sqlcgen.DBTX
	args []interface{}
	fail bool
}

func (c *captureOutbox) Exec(_ context.Context, _ string, args ...interface{}) (pgconn.CommandTag, error) {
	c.args = args
	if c.fail {
		return pgconn.CommandTag{}, errors.New("fallo privado")
	}
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}
func testKey() string { return base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32)) }

func TestOutboxEncryptsPersonalDataAndBindsCiphertextToJob(t *testing.T) {
	capture := &captureOutbox{}
	queue, err := NewOutbox(sqlcgen.New(capture), testKey())
	if err != nil {
		t.Fatal(err)
	}
	message := Message{Kind: "otp.registro", To: "persona@example.test", Body: "Código 123456", ExpiresAt: time.Now().Add(time.Minute)}
	if err := queue.Send(context.Background(), message); err != nil {
		t.Fatal(err)
	}
	encrypted := capture.args[4].([]byte)
	for _, secret := range []string{"123456", "persona@example.test"} {
		if bytes.Contains(encrypted, []byte(secret)) {
			t.Fatal("dato personal en texto claro")
		}
	}
	jobID := capture.args[0].(uuid.UUID)
	raw, err := queue.box.Open(nil, encrypted[:queue.box.NonceSize()], encrypted[queue.box.NonceSize():], []byte(jobID.String()))
	if err != nil || !strings.Contains(string(raw), message.To) {
		t.Fatal("no se pudo recuperar la carga cifrada", err)
	}
	if _, err := queue.box.Open(nil, encrypted[:queue.box.NonceSize()], encrypted[queue.box.NonceSize():], []byte(uuid.NewString())); err == nil {
		t.Fatal("se pudo mover la carga a otro trabajo")
	}
	encrypted[len(encrypted)-1] ^= 1
	if _, err := queue.box.Open(nil, encrypted[:queue.box.NonceSize()], encrypted[queue.box.NonceSize():], []byte(jobID.String())); err == nil {
		t.Fatal("carga alterada aceptada")
	}
}

func TestOutboxRequiresStableKeyAndPropagatesPersistenceFailure(t *testing.T) {
	capture := &captureOutbox{}
	for _, key := range []string{"", "no-base64", base64.StdEncoding.EncodeToString([]byte("corta"))} {
		queue, err := NewOutbox(sqlcgen.New(capture), key)
		if !errors.Is(err, ErrQueueUnavailable) {
			t.Fatal("clave inválida aceptada")
		}
		if err := queue.Send(context.Background(), Message{To: "a@example.test"}); !errors.Is(err, ErrQueueUnavailable) {
			t.Fatal("envío aceptado sin clave")
		}
	}
	queue, err := NewOutbox(sqlcgen.New(capture), testKey())
	if err != nil {
		t.Fatal(err)
	}
	capture.fail = true
	if err := queue.Send(context.Background(), Message{To: "a@example.test"}); !errors.Is(err, ErrQueueUnavailable) {
		t.Fatal("fallo de persistencia descartado")
	}
}
