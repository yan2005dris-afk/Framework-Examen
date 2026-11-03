package merit

import (
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type sessionStub struct {
	identity.SessionRepository
	a *identity.Applicant
}

func (s sessionStub) GetWithApplicant(context.Context, string) (*identity.Session, *identity.Applicant, error) {
	return &identity.Session{ExpiresAt: time.Now().Add(time.Hour)}, s.a, nil
}

type accessRecorder struct{ fail bool }

func (a accessRecorder) Record(context.Context, audit.Event) error {
	if a.fail {
		return audit.ErrUnavailable
	}
	return nil
}

type httpRepo struct{ memoryRepo }

func (r *httpRepo) GetRules(context.Context, uuid.UUID) (*RuleSet, error) {
	return r.c.ActiveRules, nil
}
func (r *httpRepo) ListRules(context.Context, uuid.UUID, string, int64, int) ([]RuleSet, error) {
	return []RuleSet{*r.c.ActiveRules}, nil
}
func (r *httpRepo) CreateRules(context.Context, *identity.Applicant, *RuleSet) error { return nil }
func meritHTTP(t *testing.T, a *identity.Applicant, fail bool) (http.Handler, *httpRepo) {
	t.Helper()
	c, _ := fixture(t)
	repo := &httpRepo{memoryRepo: memoryRepo{c: c}}
	router := chi.NewRouter()
	auth := identity.NewService(identity.ServiceDependencies{Sessions: sessionStub{a: a}})
	router.Use(identity.RequireSession(auth))
	router.Use(identity.ProtectAdministration)
	NewHandler(NewService(repo, accessRecorder{fail})).RegisterRoutes(router)
	return router, repo
}
func request(h http.Handler, method, path, body string, cookie bool) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if cookie {
		r.AddCookie(&http.Cookie{Name: "session", Value: "qa"})
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func TestMeritHTTPPermissionsAndStrictInputs(t *testing.T) {
	for _, role := range []string{identity.RoleApplicant, identity.RoleAnalyst, identity.RoleAdministrator} {
		t.Run(role, func(t *testing.T) {
			a := &identity.Applicant{ID: uuid.New(), Role: role}
			h, repo := meritHTTP(t, a, false)
			id := repo.c.Snapshot.ID.String()
			paths := []string{"/api/v1/admin/merits/submissions/" + id, "/api/v1/admin/merits/rules/" + repo.c.ActiveRules.ID.String()}
			for _, p := range paths {
				w := request(h, "GET", p, "", true)
				want := 200
				if role == identity.RoleApplicant {
					want = 403
				}
				if w.Code != want {
					t.Fatal(w.Code, w.Body.String())
				}
			}
			for _, op := range []string{"review", "calculate", "publish"} {
				w := request(h, "POST", "/api/v1/admin/merits/submissions/"+id+"/"+op, `{"puntajeCliente":100}`, true)
				want := 422
				if role == identity.RoleApplicant || op == "publish" && role != identity.RoleAdministrator {
					want = 403
				}
				if w.Code != want {
					t.Fatal(op, w.Code, w.Body.String())
				}
			}
		})
	}
	a := &identity.Applicant{ID: uuid.New(), Role: identity.RoleAnalyst}
	h, repo := meritHTTP(t, a, true)
	w := request(h, "GET", "/api/v1/admin/merits/submissions/"+repo.c.Snapshot.ID.String(), "", true)
	if w.Code != 503 || strings.Contains(w.Body.String(), repo.c.ActiveRules.Hash) {
		t.Fatal("lectura sin auditoría", w.Code)
	}
	w = request(h, "GET", "/api/v1/admin/merits/rules/"+repo.c.ActiveRules.ID.String(), "", false)
	if w.Code != 401 {
		t.Fatal(w.Code)
	}
}
func TestOwnerOnlySeesExplicitlyPublishedOutcome(t *testing.T) {
	a := &identity.Applicant{ID: uuid.New(), Role: identity.RoleApplicant}
	h, repo := meritHTTP(t, a, false)
	path := "/api/v1/merits/submissions/" + repo.c.Snapshot.ID.String()
	w := request(h, "GET", path, "", true)
	if w.Code != 403 {
		t.Fatal("acceso de tercero", w.Code)
	}
	repo.c.Snapshot.ApplicantID = a.ID
	w = request(h, "GET", path, "", true)
	if w.Code != 200 || strings.Contains(w.Body.String(), "reglasActivas") || !strings.Contains(w.Body.String(), `"habilitadoFaseTecnica":false`) {
		t.Fatal(w.Code, w.Body.String())
	}
	repo.c.Publication = &Publication{ID: uuid.New(), Decision: "avanza"}
	w = request(h, "GET", path, "", true)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"habilitadoFaseTecnica":true`) {
		t.Fatal(w.Code, w.Body.String())
	}
	repo.c.Publication.Decision = "excluido"
	w = request(h, "GET", path, "", true)
	if !strings.Contains(w.Body.String(), `"habilitadoFaseTecnica":false`) {
		t.Fatal(w.Body.String())
	}
}

func TestDecodeRejectsOversizeAndTrailingBodies(t *testing.T) {
	for _, body := range []string{`{}` + strings.Repeat(" ", 129<<10), `{} {}`, `{"desconocido":1}`} {
		r := httptest.NewRequest("POST", "/", strings.NewReader(body))
		var input CalculateInput
		if err := decodeRequest(r, &input); err == nil {
			t.Fatal("aceptó cuerpo fuera de contrato")
		}
	}
}
