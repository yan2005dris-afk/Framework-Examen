package postgres

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit"
	auditpostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit/postgres"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/competition"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/notice"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/db"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/mail"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/shared/domain"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/testutil"
	"strings"
	"testing"
	"time"
)

type auditFailure struct{}

func (auditFailure) RecordWorkflow(context.Context, audit.Event) error { return audit.ErrUnavailable }
func (auditFailure) Record(context.Context, audit.Event) error         { return audit.ErrUnavailable }

type smtpFailure struct {
	fail bool
	ids  []string
}

func (s *smtpFailure) Send(_ context.Context, m mail.Message) error {
	s.ids = append(s.ids, m.ID)
	if s.fail {
		return errors.New("SMTP PRIVADO contacto@example.test OTP 123456")
	}
	return nil
}
func TestStageNoticeTransactionalDeliveryAndAuthorizedRetry(t *testing.T) {
	pool := testutil.Postgres(t, "NOTICE_TEST_DATABASE_URL")
	ctx := context.Background()
	q := sqlcgen.New(pool)
	tx := db.NewTxManager(pool)
	auditor := auditpostgres.NewRepository(q)
	owner := &identity.Applicant{ID: uuid.New(), Role: identity.RoleApplicant}
	admin := &identity.Applicant{ID: uuid.New(), Role: identity.RoleAdministrator}
	for i, a := range []*identity.Applicant{owner, admin} {
		if _, err := pool.Exec(ctx, `INSERT INTO applicants(id,cedula,nombres,apellidos,correo,role) VALUES($1,$2,'QA','Prueba',$3,$4)`, a.ID, strings.Repeat("0", 9)+string(rune('1'+i)), a.ID.String()+"@example.test", a.Role); err != nil {
			t.Fatal(err)
		}
	}
	period := uuid.New()
	phases, _ := json.Marshal([]competition.Phase{{Code: "postulacion", Title: "QA", Enabled: true, Published: true, OpensAt: time.Now().Add(-time.Hour), ClosesAt: time.Now().Add(time.Hour)}})
	if _, err := pool.Exec(ctx, `INSERT INTO competitions(id,code,title,opens_at,closes_at) VALUES($1,'QA-NOTICE','QA',now()-interval '1 hour',now()+interval '1 hour')`, period); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO competition_controls(competition_id,approved,published,form_enabled,sectors,phases) VALUES($1,true,true,true,ARRAY['afiliados'],$2)`, period, phases); err != nil {
		t.Fatal(err)
	}
	sub, source := uuid.New(), uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO submissions(id,applicant_id,competition_id,code,status,content) VALUES($1,$2,$3,'QA-AVISO','draft','{}')`, sub, owner.ID, period); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE submissions SET status='submitted',hash=repeat('a',64),submitted_at=now() WHERE id=$1`, sub); err != nil {
		t.Fatal(err)
	}
	outbox, err := mail.NewOutbox(q, base64.StdEncoding.EncodeToString(make([]byte, 32)))
	if err != nil {
		t.Fatal(err)
	}
	queue := &Queue{Outbox: outbox, Queries: q}
	enqueue := func(ctx context.Context) error { return queue.Enqueue(ctx, sub, source, "resultado.publicado") }
	if err = enqueue(ctx); !errors.Is(err, notice.ErrUnavailable) {
		t.Fatal("aviso fuera de transacción", err)
	}
	sentinel := errors.New("rollback QA")
	err = tx.WithinTx(ctx, func(t pgx.Tx) error {
		if err := enqueue(db.WithTransaction(ctx, t)); err != nil {
			return err
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatal(err)
	}
	var count int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM stage_notices`).Scan(&count); err != nil || count != 0 {
		t.Fatal("sobrevivió rollback", err)
	}
	for range 2 {
		if err = tx.WithinTx(ctx, func(tx pgx.Tx) error { return enqueue(db.WithTransaction(ctx, tx)) }); err != nil {
			t.Fatal(err)
		}
	}
	repo := NewRepository(q, tx, auditor)
	s := notice.NewService(repo, auditor)
	page, err := s.List(ctx, owner, sub, domain.NewCursorRequest(100, nil), false)
	if err != nil || len(page.Items) != 1 {
		t.Fatal("aviso duplicado o ausente", err)
	}
	mailID := page.Items[0].ID
	var ciphertext []byte
	if err = pool.QueryRow(ctx, `SELECT ciphertext FROM mail_outbox WHERE id=$1`, mailID).Scan(&ciphertext); err != nil || bytes.Contains(ciphertext, []byte("example.test")) {
		t.Fatal("carga no cifrada", err)
	}
	sender := &smtpFailure{fail: true}
	for range 5 {
		if _, err = pool.Exec(ctx, `UPDATE mail_outbox SET next_attempt_at=now() WHERE id=$1`, mailID); err != nil {
			t.Fatal(err)
		}
		if err = outbox.Dispatch(ctx, sender, nil); err != nil {
			t.Fatal(err)
		}
	}
	page, err = s.List(ctx, owner, sub, domain.NewCursorRequest(100, nil), false)
	if err != nil || page.Items[0].Status != "failed" || page.Items[0].Total != 5 || page.Items[0].ErrorCode != "smtp_error" {
		t.Fatal("historial de fallo incorrecto", page, err)
	}
	input := notice.RetryInput{Reason: "Reintento operativo QA"}
	if err = s.Retry(ctx, owner, sub, mailID, input); !errors.Is(err, notice.ErrForbidden) {
		t.Fatal("postulante reintentó", err)
	}
	failed := notice.NewService(NewRepository(q, tx, auditFailure{}), auditor)
	if err = failed.Retry(ctx, admin, sub, mailID, input); !errors.Is(err, audit.ErrUnavailable) {
		t.Fatal("reintento sin auditoría", err)
	}
	if err = s.Retry(ctx, admin, sub, mailID, input); err != nil {
		t.Fatal(err)
	}
	if err = s.Retry(ctx, admin, sub, mailID, input); !errors.Is(err, notice.ErrConflict) {
		t.Fatal("duplicó reintento", err)
	}
	sender.fail = false
	if err = outbox.Dispatch(ctx, sender, nil); err != nil {
		t.Fatal(err)
	}
	for _, id := range sender.ids {
		if id != mailID.String() {
			t.Fatal("cambió Message-ID")
		}
	}
	page, err = s.List(ctx, owner, sub, domain.NewCursorRequest(100, nil), false)
	if err != nil || page.Items[0].Status != "smtp_accepted" || page.Items[0].Total != 6 || page.Items[0].Retries != 1 {
		t.Fatal(page, err)
	}
	if err = s.Retry(ctx, admin, sub, mailID, notice.RetryInput{PreviousRetries: 1, Reason: "QA"}); !errors.Is(err, notice.ErrConflict) {
		t.Fatal("reenvió correo aceptado", err)
	}
	history, err := s.History(ctx, owner, sub, mailID, domain.NewCursorRequest(100, nil), false)
	if err != nil || len(history.Items) != 13 {
		t.Fatal("intentos sin historial", len(history.Items), err)
	}
	if _, err = pool.Exec(ctx, `DELETE FROM mail_delivery_events WHERE mail_id=$1`, mailID); err == nil {
		t.Fatal("historial eliminable")
	}
	if err = pool.QueryRow(ctx, `SELECT ciphertext FROM mail_outbox WHERE id=$1`, mailID).Scan(&ciphertext); err != nil || len(ciphertext) != 0 {
		t.Fatal("carga personal retenida", err)
	}
	badRead := notice.NewService(repo, auditFailure{})
	if v, err := badRead.List(ctx, admin, sub, domain.NewCursorRequest(100, nil), true); !errors.Is(err, audit.ErrUnavailable) || len(v.Items) != 0 {
		t.Fatal("lectura administrativa sin auditoría", err)
	}
	stranger := &identity.Applicant{ID: uuid.New(), Role: identity.RoleApplicant}
	if _, err = s.List(ctx, stranger, sub, domain.NewCursorRequest(100, nil), false); !errors.Is(err, notice.ErrForbidden) {
		t.Fatal("leyó avisos de otro", err)
	}
	// Un segundo aviso fallido se elimina al vencer; no se retiene el cifrado para siempre.
	source = uuid.New()
	if err = tx.WithinTx(ctx, func(tx pgx.Tx) error { return enqueue(db.WithTransaction(ctx, tx)) }); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE mail_outbox SET status='failed',expires_at=now()-interval '1 second' WHERE status='pending'`); err != nil {
		t.Fatal(err)
	}
	if err = outbox.Dispatch(ctx, sender, nil); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM mail_outbox WHERE status='expired' AND octet_length(ciphertext)=0`).Scan(&count); err != nil || count != 1 {
		t.Fatal("carga vencida retenida", count, err)
	}
}
