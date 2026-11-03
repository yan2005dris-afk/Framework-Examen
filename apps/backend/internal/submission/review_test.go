package submission

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"testing"
)

type reviewAudit struct {
	events []audit.Event
	fail   bool
}

func (a *reviewAudit) Record(_ context.Context, event audit.Event) error {
	if a.fail {
		return errors.New("auditoría no disponible")
	}
	a.events = append(a.events, event)
	return nil
}

func TestAdministrativeReviewIsAuditedAndDoesNotChangeSealedSnapshot(t *testing.T) {
	s, repo, _ := serviceFixture(t, true)
	item, err := s.Submit(context.Background(), repo.item.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	before := string(item.Content)
	auditor := &reviewAudit{}
	s.WithAudit(auditor)
	for _, role := range []string{identity.RoleAnalyst, identity.RoleAdministrator} {
		actor := &identity.Applicant{ID: uuid.New(), Role: role}
		got, err := s.GetForReview(context.Background(), actor, item.ID)
		if err != nil || got.Hash != item.Hash || string(got.Content) != before {
			t.Fatalf("snapshot alterado o rechazado: %v", err)
		}
		event := auditor.events[len(auditor.events)-1]
		if event.ActorID != actor.ID || event.ResourceID != item.ID || event.Result != "permitido" || event.CreatedAt.IsZero() {
			t.Fatalf("auditoría incompleta: %+v", event)
		}
	}
	for _, actor := range []*identity.Applicant{nil, {ID: item.ApplicantID, Role: identity.RoleApplicant}, {ID: uuid.New(), Role: "superadmin"}} {
		if got, err := s.GetForReview(context.Background(), actor, item.ID); got != nil || !errors.Is(err, ErrForbidden) {
			t.Fatalf("lectura no autorizada: %v", err)
		}
	}
	actor := &identity.Applicant{ID: uuid.New(), Role: identity.RoleAnalyst}
	if _, err := s.GetForReview(context.Background(), actor, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("recurso ausente: %v", err)
	}
	if auditor.events[len(auditor.events)-1].Result != "rechazado" {
		t.Fatal("falta auditoría del acceso fallido")
	}
	auditor.fail = true
	if got, err := s.GetForReview(context.Background(), actor, item.ID); got != nil || !errors.Is(err, audit.ErrUnavailable) {
		t.Fatal("datos expuestos sin auditar")
	}
	s.WithAudit(nil)
	if got, err := s.GetForReview(context.Background(), actor, item.ID); got != nil || !errors.Is(err, audit.ErrUnavailable) {
		t.Fatal("auditoría ausente admitida")
	}
}

func TestAdministrativeReviewDoesNotExposeDrafts(t *testing.T) {
	s, repo, _ := serviceFixture(t, false)
	auditor := &reviewAudit{}
	s.WithAudit(auditor)
	actor := &identity.Applicant{ID: uuid.New(), Role: identity.RoleAdministrator}
	if got, err := s.GetForReview(context.Background(), actor, repo.item.ID); got != nil || !errors.Is(err, ErrForbidden) {
		t.Fatalf("borrador expuesto: %v", err)
	}
	if len(auditor.events) != 1 || auditor.events[0].Result != "rechazado" {
		t.Fatal("rechazo sin auditoría")
	}
}
