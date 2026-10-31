package mail

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/db"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/testutil"
	"net"
	"testing"
	"time"
)

type retrySender struct {
	calls int
	fail  bool
}

func (s *retrySender) Send(context.Context, Message) error {
	s.calls++
	if s.fail {
		return errors.New("fallo SMTP con datos privados que no deben persistir")
	}
	return nil
}

func TestPostgresOutboxIsTransactionalRetryableAndExpiresOTP(t *testing.T) {
	pool := testutil.Postgres(t, "MAIL_TEST_DATABASE_URL")
	ctx := context.Background()
	queue, err := NewOutbox(sqlcgen.New(pool), testKey())
	if err != nil {
		t.Fatal(err)
	}
	id := uuid.New()
	message := Message{Kind: "submission.confirmation", To: "a@example.test", Body: "Formulario sellado de prueba", ResourceID: id.String()}
	sentinel := errors.New("revertir todo")
	err = db.NewTxManager(pool).WithinTx(ctx, func(tx pgx.Tx) error {
		if err := queue.Send(db.WithTransaction(ctx, tx), message); err != nil {
			return err
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatal(err)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM mail_outbox`).Scan(&count); err != nil || count != 0 {
		t.Fatal("cola sobrevivió al rollback", err)
	}
	if err := queue.Send(ctx, message); err != nil {
		t.Fatal(err)
	}
	sender := &retrySender{fail: true}
	if err := queue.Dispatch(ctx, sender, nil); err != nil {
		t.Fatal(err)
	}
	state, err := queue.State(ctx, id)
	if err != nil || state.Status != "pending" || state.Attempts != 1 || state.ErrorCode != "smtp_error" {
		t.Fatalf("estado tras fallo: %+v %v", state, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE mail_outbox SET next_attempt_at=now() WHERE resource_id=$1`, id); err != nil {
		t.Fatal(err)
	}
	sender.fail = false
	if err := queue.Dispatch(ctx, sender, nil); err != nil {
		t.Fatal(err)
	}
	state, err = queue.State(ctx, id)
	if err != nil || state.Status != "smtp_accepted" || state.Attempts != 2 {
		t.Fatal(state, err)
	}
	var size int
	if err := pool.QueryRow(ctx, `SELECT octet_length(ciphertext) FROM mail_outbox WHERE resource_id=$1`, id).Scan(&size); err != nil || size != 0 {
		t.Fatal("carga personal retenida tras aceptación", err)
	}
	calls := sender.calls
	if err := queue.Send(ctx, Message{Kind: "otp.registro", To: "a@example.test", Body: "123456", ExpiresAt: time.Now().Add(-time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if err := queue.Send(ctx, Message{Kind: "otp.registro", To: "a@example.test", Body: "654321", ChallengeKey: "clave", ChallengeID: "anterior", ExpiresAt: time.Now().Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if err := queue.Dispatch(ctx, sender, func(context.Context, string, string) (bool, error) { return false, nil }); err != nil {
		t.Fatal(err)
	}
	if sender.calls != calls {
		t.Fatal("se entregó OTP vencido o reemplazado")
	}
}

func TestPostgresOutboxSMTPRealTestServer(t *testing.T) {
	pool := testutil.Postgres(t, "MAIL_TEST_DATABASE_URL")
	ctx := context.Background()
	queue, err := NewOutbox(sqlcgen.New(pool), testKey())
	if err != nil {
		t.Fatal(err)
	}
	server := startFakeSMTP(t)
	sender := NewSMTPSender(Config{Host: "127.0.0.1", Port: server.listener.Addr().(*net.TCPAddr).Port, From: "prueba@example.test", Timeout: 2 * time.Second})
	if err := queue.Send(ctx, Message{To: "a@example.test", Body: "Código de prueba 123456"}); err != nil {
		t.Fatal(err)
	}
	if err := queue.Dispatch(ctx, sender, nil); err != nil {
		t.Fatal(err)
	}
	select {
	case <-server.messages:
	case <-time.After(3 * time.Second):
		t.Fatal("correo en cola no llegó al servidor SMTP de prueba")
	}
}
