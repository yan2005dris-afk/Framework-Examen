package notification

import (
	"context"
	"encoding/base64"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/db"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/mail"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/testutil"
	"testing"
	"time"
)

func TestPostgresQueuedPrintRetriesAfterPreparationFailure(t *testing.T) {
	pool := testutil.Postgres(t, "MAIL_TEST_DATABASE_URL")
	ctx := context.Background()
	outbox, err := mail.NewOutbox(sqlcgen.New(pool), base64.StdEncoding.EncodeToString(make([]byte, 32)))
	if err != nil {
		t.Fatal(err)
	}
	snapshot := sealedFixture(time.Now().UTC())
	queue := &QueuedConfirmation{Outbox: outbox, Applicants: applicantReader{applicant: &identity.Applicant{ID: snapshot.ApplicantID, Email: "registrado@example.test"}}}
	tx := db.NewTxManager(pool)
	sentinel := errors.New("rollback QA")
	err = tx.WithinTx(ctx, func(tx pgx.Tx) error {
		if err := queue.Enqueue(db.WithTransaction(ctx, tx), snapshot); err != nil {
			return err
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatal(err)
	}
	var count int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM mail_outbox`).Scan(&count); err != nil || count != 0 {
		t.Fatal("correo sobrevivió rollback", count, err)
	}
	if err = tx.WithinTx(ctx, func(tx pgx.Tx) error { return queue.Enqueue(db.WithTransaction(ctx, tx), snapshot) }); err != nil {
		t.Fatal(err)
	}
	smtp := &sender{}
	printer := &printStub{err: errors.New("fallo temporal")}
	worker := &PrintingSender{Next: smtp, Printer: printer}
	beforeHash := snapshot.Hash
	if err = outbox.Dispatch(ctx, worker, nil); err != nil {
		t.Fatal(err)
	}
	state, err := queue.State(ctx, snapshot.ID)
	if err != nil || state.Status != "pending" || state.ErrorCode != "document_preparation_error" || len(smtp.message.Attachments) > 0 {
		t.Fatal("preparación no reintentable", state, err)
	}
	if snapshot.Hash != beforeHash {
		t.Fatal("fallo modificó sello")
	}
	if _, err = pool.Exec(ctx, `UPDATE mail_outbox SET next_attempt_at=now()`); err != nil {
		t.Fatal(err)
	}
	printer.err = nil
	if err = outbox.Dispatch(ctx, worker, nil); err != nil {
		t.Fatal(err)
	}
	state, err = queue.State(ctx, snapshot.ID)
	if err != nil || state.Status != "smtp_accepted" || state.Attempts != 2 || len(smtp.message.Attachments) != 1 || smtp.message.To != "registrado@example.test" || printer.hash != beforeHash {
		t.Fatal("entrega incorrecta", state, err)
	}
	if err = pool.QueryRow(ctx, `SELECT octet_length(ciphertext) FROM mail_outbox`).Scan(&count); err != nil || count != 0 {
		t.Fatal("carga retenida tras entrega", count, err)
	}
}
