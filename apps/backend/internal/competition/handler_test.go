package competition

import (
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type repositoryStub struct {
	item    Competition
	created int
}

func (r *repositoryStub) Create(_ context.Context, c *Competition, _ uuid.UUID) error {
	r.created++
	c.Version = 1
	return nil
}
func (r *repositoryStub) Get(context.Context, uuid.UUID) (*Competition, error) {
	c := r.item
	return &c, nil
}
func (r *repositoryStub) Configure(_ context.Context, _ uuid.UUID, c Configuration, _ uuid.UUID) (*Competition, error) {
	item := r.item
	item.Configuration = c
	return &item, nil
}
func (r *repositoryStub) ListPublished(context.Context, int, uuid.UUID) ([]Competition, error) {
	return []Competition{r.item}, nil
}

type sessionStub struct {
	identity.SessionRepository
	actor *identity.Applicant
}

func (s sessionStub) GetWithApplicant(context.Context, string) (*identity.Session, *identity.Applicant, error) {
	return &identity.Session{ExpiresAt: time.Now().Add(time.Hour)}, s.actor, nil
}
func routerFixture(role string, repo *repositoryStub) http.Handler {
	r := chi.NewRouter()
	h := NewHandler(NewService(repo))
	h.RegisterPublic(r)
	sessions := sessionStub{actor: &identity.Applicant{ID: uuid.New(), Role: role}}
	auth := identity.NewService(identity.ServiceDependencies{Sessions: sessions})
	r.Group(func(r chi.Router) {
		r.Use(identity.RequireSession(auth))
		r.Use(identity.ProtectAdministration)
		h.RegisterAdministration(r)
	})
	return r
}

func TestAdministrationRequiresAdministratorWithoutChangingReviewPermission(t *testing.T) {
	for _, tt := range []struct {
		role string
		want int
	}{{"", 401}, {"postulante", 403}, {"analista", 403}, {"administrador", 201}} {
		t.Run(tt.role, func(t *testing.T) {
			repo := &repositoryStub{}
			handler := routerFixture(tt.role, repo)
			req := httptest.NewRequest("POST", "/api/v1/admin/competitions", strings.NewReader(`{"codigo":"QA","titulo":"Prueba","apertura":"2027-01-01T00:00:00-05:00","cierre":"2027-02-01T00:00:00-05:00","sectores":["afiliados"],"fases":[]}`))
			if tt.role != "" {
				req.AddCookie(&http.Cookie{Name: "session", Value: "token-sintetico"})
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			if w.Code != tt.want {
				t.Fatal(w.Code, w.Body.String())
			}
			if (repo.created == 1) != (tt.want == 201) {
				t.Fatal("persistencia sin permiso")
			}
		})
	}
}
func TestPublicCalendarDoesNotUsePrivateAdministrativeEndpointOrExposeOperators(t *testing.T) {
	c := validCompetition()
	actor := uuid.New()
	c.ApprovedBy = &actor
	c.Phases = append(c.Phases, Phase{Code: "meritos", Title: "BORRADOR-PRIVADO"})
	handler := routerFixture("", &repositoryStub{item: c})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/competitions?limit=20", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), "America/Guayaquil") || strings.Contains(w.Body.String(), "BORRADOR-PRIVADO") || strings.Contains(w.Body.String(), "aprobadoPor") {
		t.Fatal(w.Code, w.Body.String())
	}
}
func TestUnpublishedCompetitionRemainsPrivate(t *testing.T) {
	c := validCompetition()
	c.Published = false
	handler := routerFixture("", &repositoryStub{item: c})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/competitions/"+c.ID.String(), nil))
	if w.Code != 404 {
		t.Fatal(w.Code, w.Body.String())
	}
}
