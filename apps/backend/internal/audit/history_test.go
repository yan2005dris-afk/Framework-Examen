package audit

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/shared/domain"
	"testing"
	"time"
)

type historyMemory struct {
	owner    *Owner
	entries  []Entry
	calls    int
	limit    int
	position domain.Position
	fail     bool
	audit    []Event
}

func (m *historyMemory) Owner(context.Context, uuid.UUID) (*Owner, error) { return m.owner, nil }
func (m *historyMemory) History(_ context.Context, _ uuid.UUID, p domain.Position, limit int) ([]Entry, error) {
	m.calls++
	m.limit = limit
	m.position = p
	return m.entries, nil
}
func (m *historyMemory) Record(_ context.Context, e Event) error {
	m.audit = append(m.audit, e)
	if m.fail {
		return errors.New("fallo")
	}
	return nil
}

func TestHistoryOwnershipPermissionAndAuditFailure(t *testing.T) {
	owner, id := uuid.New(), uuid.New()
	for _, tc := range []struct {
		name   string
		actor  *identity.Applicant
		admin  bool
		status string
		fail   bool
		want   error
		calls  int
	}{
		{"propietario", &identity.Applicant{ID: owner, Role: identity.RoleApplicant}, false, "draft", false, nil, 1},
		{"propietario eliminado", &identity.Applicant{ID: owner, Role: identity.RoleApplicant}, false, "deleted", false, nil, 1},
		{"tercero", &identity.Applicant{ID: uuid.New(), Role: identity.RoleApplicant}, false, "submitted", false, ErrForbidden, 0},
		{"analista no eleva ruta propia", &identity.Applicant{ID: uuid.New(), Role: identity.RoleAnalyst}, false, "submitted", false, ErrForbidden, 0},
		{"analista expediente", &identity.Applicant{ID: uuid.New(), Role: identity.RoleAnalyst}, true, "submitted", false, nil, 1},
		{"administrador expediente", &identity.Applicant{ID: uuid.New(), Role: identity.RoleAdministrator}, true, "submitted", false, nil, 1},
		{"borrador privado", &identity.Applicant{ID: uuid.New(), Role: identity.RoleAnalyst}, true, "draft", false, ErrForbidden, 0},
		{"postulante no administrador", &identity.Applicant{ID: owner, Role: identity.RoleApplicant}, true, "submitted", false, ErrForbidden, 0},
		{"sin sesión", nil, true, "submitted", false, ErrForbidden, 0},
		{"sin auditoría", &identity.Applicant{ID: uuid.New(), Role: identity.RoleAnalyst}, true, "submitted", true, ErrUnavailable, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := &historyMemory{owner: &Owner{ApplicantID: owner, Status: tc.status}, fail: tc.fail, entries: []Entry{{ID: uuid.New(), Action: "guardada"}}}
			result, err := NewHistoryService(m, m).List(context.Background(), tc.actor, id, domain.NewCursorRequest(20, nil), tc.admin)
			if !errors.Is(err, tc.want) || m.calls != tc.calls {
				t.Fatalf("error=%v consultas=%d", err, m.calls)
			}
			if err != nil && len(result.Items) != 0 {
				t.Fatal("se entregó historial tras un rechazo")
			}
			if tc.admin && tc.calls == 1 && len(m.audit) != 1 {
				t.Fatal("lectura sin auditoría")
			}
		})
	}
}

func TestHistoryStableBoundCursorAndLimit(t *testing.T) {
	owner, id := uuid.New(), uuid.New()
	now := time.Now().UTC()
	m := &historyMemory{owner: &Owner{ApplicantID: owner, Status: "submitted"}, entries: []Entry{{ID: uuid.New(), CreatedAt: now}, {ID: uuid.New(), CreatedAt: now}}}
	s := NewHistoryService(m, m)
	actor := &identity.Applicant{ID: owner}
	page, err := s.List(context.Background(), actor, id, domain.NewCursorRequest(1, nil), false)
	if err != nil || !page.HasMore || len(page.Items) != 1 || m.limit != 2 {
		t.Fatalf("página %+v %v", page, err)
	}
	cursor, err := domain.DecodeCursor(page.NextCursor)
	if err != nil {
		t.Fatal(err)
	}
	m.entries = nil
	if _, err = s.List(context.Background(), actor, id, domain.NewCursorRequest(1, &cursor), false); err != nil {
		t.Fatal(err)
	}
	if !m.position.At.Equal(now) || m.position.ID != page.Items[0].ID {
		t.Fatal("se perdió el desempate UUID")
	}
	if _, err = s.List(context.Background(), actor, uuid.New(), domain.NewCursorRequest(1, &cursor), false); !errors.Is(err, domain.ErrInvalidCursor) {
		t.Fatal("cursor de otro expediente admitido", err)
	}
	if _, err = s.List(context.Background(), actor, id, domain.CursorRequest{Limit: 1000}, false); err != nil || m.limit != 101 {
		t.Fatal("no se normalizó el límite", err)
	}
}

func TestWorkflowMetadataIsClosedAndContainsNoPayload(t *testing.T) {
	e := Event{ActorID: uuid.New(), Role: identity.RoleAnalyst, Action: "meritos.calificados", ResourceID: uuid.New(), ResourceType: "postulacion", Result: "permitido", CreatedAt: time.Now()}
	if !ValidWorkflow(e) {
		t.Fatal("evento válido rechazado")
	}
	for _, change := range []func(*Event){func(e *Event) { e.Action = "accion.arbitraria" }, func(e *Event) { e.Role = "root" }, func(e *Event) { e.Result = "correo@example.test" }, func(e *Event) { e.ResourceType = "nombre completo" }, func(e *Event) { e.ActorID = uuid.Nil }} {
		invalid := e
		change(&invalid)
		if ValidWorkflow(invalid) {
			t.Fatal("evento inválido aceptado")
		}
	}
	e.ActorID = uuid.Nil
	e.Role = "sistema"
	if !ValidWorkflow(e) {
		t.Fatal("evento automático rechazado")
	}
}
