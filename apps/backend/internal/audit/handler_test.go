package audit

import (
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type historySession struct {
	identity.SessionRepository
	actor *identity.Applicant
}

func (s historySession) GetWithApplicant(context.Context, string) (*identity.Session, *identity.Applicant, error) {
	return &identity.Session{ExpiresAt: time.Now().Add(time.Hour)}, s.actor, nil
}
func TestHistoryHTTPKeepsOwnAndAdministrativePermissionsSeparate(t *testing.T) {
	owner, id := uuid.New(), uuid.New()
	actor := &identity.Applicant{ID: owner, Role: identity.RoleApplicant}
	m := &historyMemory{owner: &Owner{ApplicantID: owner, Status: "submitted"}}
	r := chi.NewRouter()
	r.Use(identity.RequireSession(identity.NewService(identity.ServiceDependencies{Sessions: historySession{actor: actor}})))
	r.Use(identity.ProtectAdministration)
	RegisterHistoryRoutes(r, NewHistoryService(m, m))
	cases := []struct {
		path    string
		role    string
		actorID uuid.UUID
		token   bool
		want    int
	}{
		{"/api/v1/submissions/" + id.String() + "/events", identity.RoleApplicant, owner, true, 200},
		{"/api/v1/submissions/" + id.String() + "/events", identity.RoleAnalyst, uuid.New(), true, 403},
		{"/api/v1/admin/submissions/" + id.String() + "/events", identity.RoleApplicant, owner, true, 403},
		{"/api/v1/admin/submissions/" + id.String() + "/events", identity.RoleAnalyst, uuid.New(), true, 200},
		{"/api/v1/submissions/" + id.String() + "/events?limit=5&limit=6", identity.RoleApplicant, owner, true, 400},
		{"/api/v1/submissions/" + id.String() + "/events?cursor=invalid", identity.RoleApplicant, owner, true, 400},
		{"/api/v1/submissions/" + id.String() + "/events", identity.RoleApplicant, owner, false, 401},
	}
	for _, tc := range cases {
		actor.ID = tc.actorID
		actor.Role = tc.role
		request := httptest.NewRequest("GET", tc.path, nil)
		if tc.token {
			request.AddCookie(&http.Cookie{Name: "session", Value: "test"})
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, request)
		if w.Code != tc.want {
			t.Fatalf("%s: %d %s", tc.path, w.Code, w.Body.String())
		}
	}
}
