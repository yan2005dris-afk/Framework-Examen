package notice

import (
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/shared/domain"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type repoStub struct {
	Repository
	owner uuid.UUID
	calls int
}

func (r *repoStub) Owner(context.Context, uuid.UUID) (uuid.UUID, error) { return r.owner, nil }
func (r *repoStub) List(context.Context, uuid.UUID, domain.Position, int) ([]Delivery, error) {
	return []Delivery{{ID: uuid.New(), Status: "failed", ErrorCode: "smtp_error", Total: 5}}, nil
}
func (r *repoStub) Retry(context.Context, *identity.Applicant, uuid.UUID, uuid.UUID, RetryInput) error {
	r.calls++
	return nil
}

type sessionStub struct {
	identity.SessionRepository
	a *identity.Applicant
}

func (s sessionStub) GetWithApplicant(context.Context, string) (*identity.Session, *identity.Applicant, error) {
	return &identity.Session{ExpiresAt: time.Now().Add(time.Hour)}, s.a, nil
}

type auditStub struct{}

func (auditStub) Record(context.Context, audit.Event) error { return nil }
func TestNotificationHTTPRolesAndStrictRetry(t *testing.T) {
	for _, role := range []string{identity.RoleApplicant, identity.RoleAnalyst, identity.RoleAdministrator} {
		t.Run(role, func(t *testing.T) {
			a := &identity.Applicant{ID: uuid.New(), Role: role}
			repo := &repoStub{owner: a.ID}
			r := chi.NewRouter()
			r.Use(identity.RequireSession(identity.NewService(identity.ServiceDependencies{Sessions: sessionStub{a: a}})))
			r.Use(identity.ProtectAdministration)
			NewHandler(NewService(repo, auditStub{})).RegisterRoutes(r)
			sub, m := uuid.New(), uuid.New()
			path := "/api/v1/admin/notifications/submissions/" + sub.String()
			call := func(method, path, body string, cookie bool) *httptest.ResponseRecorder {
				q := httptest.NewRequest(method, path, strings.NewReader(body))
				if cookie {
					q.AddCookie(&http.Cookie{Name: "session", Value: "qa"})
				}
				w := httptest.NewRecorder()
				r.ServeHTTP(w, q)
				return w
			}
			w := call("GET", path, "", false)
			if w.Code != 401 {
				t.Fatal(w.Code)
			}
			w = call("GET", path, "", true)
			want := 200
			if role == identity.RoleApplicant {
				want = 403
			}
			if w.Code != want {
				t.Fatal(w.Code, w.Body.String())
			}
			w = call("POST", path+"/"+m.String()+"/retry", `{"reenviosAnteriores":0,"motivo":"QA","correo":"otro@example.test"}`, true)
			want = 422
			if role != identity.RoleAdministrator {
				want = 403
			}
			if w.Code != want || repo.calls != 0 {
				t.Fatal("aceptó destinatario o rol", w.Code, w.Body.String())
			}
			w = call("GET", "/api/v1/notifications/submissions/"+sub.String(), "", true)
			if w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" {
				t.Fatal(w.Code, w.Body.String())
			}
			if role == identity.RoleAdministrator {
				w = call("POST", path+"/"+m.String()+"/retry", `{"reenviosAnteriores":0,"motivo":"Reintento QA"}`, true)
				if w.Code != 200 || repo.calls != 1 {
					t.Fatal(w.Code, w.Body.String())
				}
			}
		})
	}
}
