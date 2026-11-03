package notification

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/mail"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"testing"
	"time"
)

type printStub struct {
	hash string
	err  error
}

func (p *printStub) Render(_ context.Context, s *submission.Submission) ([]byte, error) {
	p.hash = s.Hash
	return []byte("%PDF-sellado"), p.err
}
func TestWorkerPreparesSealedSnapshotAndFailsBeforeSMTP(t *testing.T) {
	snapshot := sealedFixture(time.Now())
	raw, _ := json.Marshal(PrintEnvelope{Version: "ef12-print-v1", Snapshot: *snapshot})
	message := mail.Message{Kind: "submission.confirmation", ResourceID: snapshot.ID.String(), RenderPayload: raw}
	smtp := &sender{}
	printer := &printStub{}
	worker := &PrintingSender{Next: smtp, Printer: printer}
	if err := worker.Send(context.Background(), message); err != nil {
		t.Fatal(err)
	}
	if len(smtp.message.Attachments) != 1 || len(smtp.message.RenderPayload) != 0 || printer.hash != snapshot.Hash {
		t.Fatal("no preparó el snapshot")
	}
	smtp.message = mail.Message{}
	printer.err = errors.New("falla de impresión")
	if err := worker.Send(context.Background(), message); !errors.Is(err, mail.ErrUnpreparedMessage) || len(smtp.message.Attachments) > 0 {
		t.Fatal("envió correo sin copia", err)
	}
	printer.err = nil
	snapshot.Hash = "alterado"
	message.RenderPayload, _ = json.Marshal(PrintEnvelope{Version: "ef12-print-v1", Snapshot: *snapshot})
	if err := worker.Send(context.Background(), message); !errors.Is(err, mail.ErrUnpreparedMessage) {
		t.Fatal("aceptó contenido alterado")
	}
}
