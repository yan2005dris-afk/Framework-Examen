package review

import (
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/shared/domain"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type roleSession struct {
	identity.SessionRepository
	actor *identity.Applicant
}

func (s roleSession) GetWithApplicant(_ context.Context, hash string) (*identity.Session, *identity.Applicant, error) {
	if hash != identity.HashSessionToken("test-session") {
		return nil, nil, identity.ErrSessionExpired
	}
	return &identity.Session{ExpiresAt: time.Now().Add(time.Hour)}, s.actor, nil
}
func reviewHTTP(t *testing.T) (http.Handler, *identity.Applicant, *memoryReview, *memoryAudit) {
	t.Helper()
	actor := &identity.Applicant{ID: uuid.New(), Role: identity.RoleAnalyst}
	repo := &memoryReview{snapshots: []Snapshot{reviewSnapshot(t)}}
	auditor := &memoryAudit{}
	r := chi.NewRouter()
	r.Use(identity.RequireSession(identity.NewService(identity.ServiceDependencies{Sessions: roleSession{actor: actor}})))
	r.Use(identity.ProtectAdministration)
	// El router real comparte las rutas /submissions/{id} con T3/T6.
	submission.NewHandler(submission.NewService(nil)).RegisterRoutes(r)
	NewHandler(NewService(repo, auditor)).RegisterRoutes(r)
	audit.RegisterHistoryRoutes(r, audit.NewHistoryService(httpHistory{owner: actor.ID}, auditor))
	return r, actor, repo, auditor
}

type httpHistory struct{ owner uuid.UUID }

func (h httpHistory) Owner(context.Context, uuid.UUID) (*audit.Owner, error) {
	return &audit.Owner{ApplicantID: h.owner, Status: "submitted"}, nil
}
func (h httpHistory) History(context.Context, uuid.UUID, domain.Position, int) ([]audit.Entry, error) {
	return []audit.Entry{}, nil
}

func TestHistoryRouteIsNotCapturedBySubmissionSubrouter(t *testing.T) {
	h, actor, _, _ := reviewHTTP(t)
	actor.Role = identity.RoleApplicant
	w := get(h, "/api/v1/submissions/"+uuid.NewString()+"/events", "test-session")
	if w.Code != 200 {
		t.Fatalf("el subrouter interceptó el historial: %d %s", w.Code, w.Body.String())
	}
}
func get(h http.Handler, path, token string) *httptest.ResponseRecorder {
	r := httptest.NewRequest("GET", path, nil)
	if token != "" {
		r.AddCookie(&http.Cookie{Name: "session", Value: token})
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestReviewHTTPContractPermissionsAndNoDynamicRouteCollision(t *testing.T) {
	h, actor, _, auditor := reviewHTTP(t)
	for _, path := range []string{"/api/v1/admin/submissions?sinConvocatoria=true", "/api/v1/admin/submissions/stats?sinConvocatoria=true", "/api/v1/admin/competitions", "/api/v1/admin/submissions/export.xlsx?sinConvocatoria=true", "/api/v1/admin/submissions/export.pdf?sinConvocatoria=true"} {
		for _, tc := range []struct {
			role, token string
			want        int
		}{{identity.RoleAnalyst, "test-session", 200}, {identity.RoleAdministrator, "test-session", 200}, {identity.RoleApplicant, "test-session", 403}, {identity.RoleAnalyst, "", 401}} {
			actor.Role = tc.role
			w := get(h, path, tc.token)
			if w.Code != tc.want {
				t.Fatalf("%s rol=%s: %d %s", path, tc.role, w.Code, w.Body.String())
			}
			if tc.want == 200 && w.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("respuesta administrativa cacheable")
			}
		}
	}
	actor.Role = identity.RoleAnalyst
	auditor.fail = true
	w := get(h, "/api/v1/admin/submissions/export.xlsx?sinConvocatoria=true", "test-session")
	if w.Code != 500 || w.Header().Get("Content-Disposition") != "" {
		t.Fatal("bytes expuestos tras fallo de auditoría", w.Code)
	}
}

func TestReviewHTTPRejectsAmbiguousFiltersAndExportPagination(t *testing.T) {
	h, _, repo, _ := reviewHTTP(t)
	for _, suffix := range []string{"", "?sinConvocatoria=false", "?sinConvocatoria=true&convocatoriaId=" + uuid.NewString(), "?sinConvocatoria=true&sector=otro", "?sinConvocatoria=true&status=draft", "?sinConvocatoria=true&sector=afiliados&sector=jubilados", "?sinConvocatoria=true&desconocido=1", "?sinConvocatoria=true&limit=x", "?sinConvocatoria=true&cursor=mal-cursor", "?sinConvocatoria=true&sector=%zz"} {
		w := get(h, "/api/v1/admin/submissions"+suffix, "test-session")
		if w.Code != 400 {
			t.Fatalf("filtro %s: %d %s", suffix, w.Code, w.Body.String())
		}
	}
	for _, suffix := range []string{"&limit=20", "&cursor=abc", "&sector="} {
		w := get(h, "/api/v1/admin/submissions/export.xlsx?sinConvocatoria=true"+suffix, "test-session")
		if w.Code != 400 {
			t.Fatal("exportación parcialmente paginada", w.Code)
		}
	}
	if repo.calls != 0 {
		t.Fatal("filtros inválidos llegaron a persistencia")
	}
}
