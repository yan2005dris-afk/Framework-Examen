package results

import (
	"context"
	"encoding/json"
	"errors"
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
	list  *List
	pub   *Publication
	calls int
}

func (r *repoStub) Get(context.Context, uuid.UUID) (*List, error)                { return r.list, nil }
func (r *repoStub) Publication(context.Context, uuid.UUID) (*Publication, error) { return r.pub, nil }
func (r *repoStub) Current(context.Context, uuid.UUID, string) (*Publication, error) {
	return r.pub, nil
}
func (r *repoStub) Publish(context.Context, *identity.Applicant, uuid.UUID, uuid.UUID, PublishInput) (*Publication, error) {
	r.calls++
	return r.pub, nil
}

type sessionStub struct {
	identity.SessionRepository
	a *identity.Applicant
}

func (s sessionStub) GetWithApplicant(context.Context, string) (*identity.Session, *identity.Applicant, error) {
	return &identity.Session{ExpiresAt: time.Now().Add(time.Hour)}, s.a, nil
}

type auditStub struct{ fail bool }

func (a auditStub) Record(context.Context, audit.Event) error {
	if a.fail {
		return audit.ErrUnavailable
	}
	return nil
}
func TestResultsHTTPRolesPrivacyAndScopedPages(t *testing.T) {
	for _, role := range []string{identity.RoleApplicant, identity.RoleAnalyst, identity.RoleAdministrator} {
		t.Run(role, func(t *testing.T) {
			a := &identity.Applicant{ID: uuid.New(), Role: role}
			id := uuid.New()
			repo := &repoStub{list: &List{ID: id, CompetitionID: uuid.New(), Entries: []Entry{{Code: "INTERNO-1"}, {Code: "INTERNO-2"}, {Code: "INTERNO-3"}}}, pub: &Publication{ID: uuid.New(), Public: true, Names: []string{"Apellido Uno", "Apellido Dos", "Apellido Tres"}}}
			h := NewHandler(NewService(repo, auditStub{}))
			r := chi.NewRouter()
			h.RegisterPublic(r)
			r.Group(func(r chi.Router) {
				r.Use(identity.RequireSession(identity.NewService(identity.ServiceDependencies{Sessions: sessionStub{a: a}})))
				r.Use(identity.ProtectAdministration)
				h.RegisterAdministration(r)
			})
			call := func(method, path, body string, cookie bool) *httptest.ResponseRecorder {
				q := httptest.NewRequest(method, path, strings.NewReader(body))
				q.Header.Set("Idempotency-Key", uuid.NewString())
				if cookie {
					q.AddCookie(&http.Cookie{Name: "session", Value: "qa"})
				}
				w := httptest.NewRecorder()
				r.ServeHTTP(w, q)
				return w
			}
			path := "/api/v1/admin/results/lists/" + id.String()
			if w := call("GET", path, "", false); w.Code != 401 {
				t.Fatal(w.Code)
			}
			w := call("GET", path, "", true)
			want := 200
			if role == identity.RoleApplicant {
				want = 403
			}
			if w.Code != want || strings.Contains(w.Body.String(), "INTERNO-") {
				t.Fatal("metadatos expusieron entradas", w.Code, w.Body.String())
			}
			w = call("POST", path+"/publish", `{"referenciaAprobacion":"QA","nota":100}`, true)
			want = 422
			if role != identity.RoleAdministrator {
				want = 403
			}
			if w.Code != want || repo.calls != 0 {
				t.Fatal(w.Code, w.Body.String())
			}
			publicPath := "/api/v1/public/results/publications/" + repo.pub.ID.String()
			w = call("GET", publicPath+"?limit=2", "", false)
			if w.Code != 200 || strings.Contains(w.Body.String(), "INTERNO") || w.Header().Get("Cache-Control") != "no-store" {
				t.Fatal(w.Code, w.Body.String())
			}
			var body struct {
				Names domain.CursorPage[string] `json:"nombres"`
			}
			if json.Unmarshal(w.Body.Bytes(), &body) != nil || len(body.Names.Items) != 2 || body.Names.NextCursor == "" {
				t.Fatal(w.Body.String())
			}
			w = call("GET", publicPath+"?limit=2&cursor="+body.Names.NextCursor, "", false)
			if w.Code != 200 || !strings.Contains(w.Body.String(), "Apellido Tres") {
				t.Fatal("paginación pública", w.Code, w.Body.String())
			}
			bad := domain.NewCursor("public-result-names:"+uuid.NewString(), "1").Encode()
			w = call("GET", publicPath+"?cursor="+bad, "", false)
			if w.Code != 422 {
				t.Fatal("cursor ajeno aceptado", w.Code)
			}
			repo.pub.Public = false
			w = call("GET", publicPath, "", false)
			if w.Code != 404 {
				t.Fatal("publicación interna expuesta", w.Code)
			}
		})
	}
}
func TestResultsApprovalAuditAndCSVFormulaProtection(t *testing.T) {
	admin := &identity.Applicant{ID: uuid.New(), Role: identity.RoleAdministrator}
	repo := &repoStub{list: &List{ID: uuid.New()}}
	s := NewService(repo, auditStub{fail: true})
	if v, err := s.Get(context.Background(), admin, repo.list.ID); !errors.Is(err, audit.ErrUnavailable) || v != nil {
		t.Fatal("lectura no falla cerrada", err)
	}
	s = NewService(repo, auditStub{})
	if _, err := s.Publish(context.Background(), admin, repo.list.ID, uuid.New(), PublishInput{Reference: "QA", Public: true}); !errors.Is(err, ErrPrivacy) || repo.calls != 0 {
		t.Fatal("publicó sin referencia privacidad", err)
	}
	for _, v := range []string{"=SUM(1)", "\t@CMD", " -FORMULA", "+1", "\ufeff=FORMULA", "\u00a0@FORMULA"} {
		if cell(v) != "'"+v {
			t.Fatal("fórmula no neutralizada", v)
		}
	}
}
