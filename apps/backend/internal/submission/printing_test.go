package submission

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"testing"
)

type printerStub struct{ calls int }

func (p *printerStub) Render(context.Context, *Submission) ([]byte, error) {
	p.calls++
	return []byte("%PDF-test"), nil
}
func TestPrintFinalAuthorizationSealAndAudit(t *testing.T) {
	s, repo, _ := serviceFixture(t, true)
	item, err := s.Submit(context.Background(), repo.item.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	p := &printerStub{}
	a := &reviewAudit{}
	s.WithPrinter(p).WithAudit(a)
	owner := &identity.Applicant{ID: item.ApplicantID, Role: identity.RoleApplicant}
	admin := &identity.Applicant{ID: uuid.New(), Role: identity.RoleAnalyst}
	if data, err := s.PrintFinal(context.Background(), owner, item.ID, false); err != nil || len(data) == 0 {
		t.Fatal(err)
	}
	if _, err = s.PrintFinal(context.Background(), &identity.Applicant{ID: uuid.New(), Role: identity.RoleAdministrator}, item.ID, false); !errors.Is(err, ErrForbidden) {
		t.Fatal("ruta propia elevó permisos")
	}
	if _, err = s.PrintFinal(context.Background(), owner, item.ID, true); !errors.Is(err, ErrForbidden) {
		t.Fatal("postulante accedió a impresión administrativa")
	}
	if _, err = s.PrintFinal(context.Background(), admin, item.ID, true); err != nil || len(a.events) != 1 {
		t.Fatal(err)
	}
	a.fail = true
	if data, err := s.PrintFinal(context.Background(), admin, item.ID, true); data != nil || !errors.Is(err, audit.ErrUnavailable) {
		t.Fatal("PDF expuesto sin auditoría")
	}
	a.fail = false
	repo.item.Hash = "alterado"
	before := p.calls
	if _, err = s.PrintFinal(context.Background(), owner, item.ID, false); !errors.Is(err, ErrInvalidSeal) || p.calls != before {
		t.Fatal("imprimió contenido alterado")
	}
	repo.item.Status = StatusDraft
	if err := ValidateSeal(repo.item); !errors.Is(err, ErrInvalidSeal) {
		t.Fatal("aceptó borrador")
	}
}
