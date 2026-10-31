package notification

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/mail"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
)

type applicantReader struct {
	applicant *identity.Applicant
	err       error
}

func (r applicantReader) GetByID(context.Context, uuid.UUID) (*identity.Applicant, error) {
	return r.applicant, r.err
}

type sender struct {
	message mail.Message
	err     error
}

func (s *sender) Send(_ context.Context, m mail.Message) error { s.message = m; return s.err }

func TestConfirmationUsesRegisteredAddressAndSealedContent(t *testing.T) {
	id := uuid.New()
	now := time.Now().UTC()
	item := &submission.Submission{ID: uuid.New(), ApplicantID: id, Code: "EF12-2026-PRUEBA", Status: submission.StatusSubmitted, SubmittedAt: &now, Hash: strings.Repeat("a", 64), Content: json.RawMessage(`{"seccion2":{"correo1":"otro@example.test"}}`)}
	canonical, _ := submission.CanonicalContent(item.Content)
	item.Hash = submission.HashContent(canonical)
	smtp := &sender{}
	c := Confirmation{Sender: smtp, Applicants: applicantReader{applicant: &identity.Applicant{ID: id, Email: "registrado@example.test"}}}
	if err := c.Confirm(context.Background(), item); err != nil {
		t.Fatal(err)
	}
	if smtp.message.To != "registrado@example.test" {
		t.Fatal("el cliente eligió el destinatario")
	}
	for _, expected := range []string{item.Hash, item.Code, string(item.Content), now.Format(time.RFC3339)} {
		if !strings.Contains(smtp.message.Body, expected) {
			t.Fatalf("confirmación incompleta: falta %s", expected)
		}
	}
}

func TestConfirmationPropagatesDeliveryFailure(t *testing.T) {
	now := time.Now()
	failure := errors.New("fallo SMTP de prueba")
	smtp := &sender{err: failure}
	c := Confirmation{Sender: smtp, Applicants: applicantReader{applicant: &identity.Applicant{Email: "prueba@example.test"}}}
	if err := c.Confirm(context.Background(), sealedFixture(now)); !errors.Is(err, failure) {
		t.Fatalf("fallo de entrega perdido: %v", err)
	}
}

func sealedFixture(now time.Time) *submission.Submission {
	raw := json.RawMessage(`{}`)
	return &submission.Submission{ID: uuid.New(), ApplicantID: uuid.New(), Status: submission.StatusSubmitted, SubmittedAt: &now, Content: raw, Hash: submission.HashContent(raw)}
}
