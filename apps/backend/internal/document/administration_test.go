package document

import (
	"context"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type administrativeAudit struct {
	events []audit.Event
	fail   bool
}

func (a *administrativeAudit) Record(_ context.Context, event audit.Event) error {
	if a.fail {
		return errors.New("auditoría no disponible")
	}
	a.events = append(a.events, event)
	return nil
}

type administrativeSessions struct {
	identity.SessionRepository
	actor *identity.Applicant
}

func (s administrativeSessions) GetWithApplicant(_ context.Context, hash string) (*identity.Session, *identity.Applicant, error) {
	if hash != identity.HashSessionToken("personal") {
		return nil, nil, identity.ErrSessionExpired
	}
	return &identity.Session{ApplicantID: s.actor.ID, ExpiresAt: time.Now().Add(time.Hour)}, s.actor, nil
}

func TestAdministrativeDocumentsHTTPPreservesOwnerRoutesAndAudits(t *testing.T) {
	s, repo, _, input := fixture(t)
	d, err := s.Upload(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	repo.item.Status = submission.StatusSubmitted
	repo.item.SubmittedAt = &now
	repo.item.Hash = "sello"
	auditor := &administrativeAudit{}
	s.WithAudit(auditor)
	actor := &identity.Applicant{ID: uuid.New(), Role: identity.RoleAnalyst}
	router := chi.NewRouter()
	router.Use(identity.RequireSession(identity.NewService(identity.ServiceDependencies{Sessions: administrativeSessions{actor: actor}})))
	router.Use(identity.ProtectAdministration)
	submissions := submission.NewService(&administrativeSubmissionReader{item: repo.item}).WithAudit(auditor)
	submission.NewHandler(submissions).RegisterRoutes(router)
	NewHandler(s).RegisterRoutes(router)
	request := func(method, path string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(`{}`))
		r.AddCookie(&http.Cookie{Name: "session", Value: "personal"})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}
	for _, role := range []string{identity.RoleAnalyst, identity.RoleAdministrator} {
		actor.Role = role
		for _, path := range []string{"/api/v1/admin/submissions/" + repo.item.ID.String(), "/api/v1/admin/submissions/" + repo.item.ID.String() + "/documents", "/api/v1/admin/documents/" + d.ID.String() + "/download"} {
			w := request("GET", path)
			if w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("lectura autorizada rechazada: %d %s", w.Code, w.Body.String())
			}
			if strings.Contains(w.Body.String(), d.StorageKey) {
				t.Fatal("clave privada expuesta")
			}
		}
		for _, pair := range [][2]string{
			{"GET", "/api/v1/submissions/" + repo.item.ID.String()},
			{"PUT", "/api/v1/submissions/" + repo.item.ID.String()},
			{"DELETE", "/api/v1/submissions/" + repo.item.ID.String()},
			{"POST", "/api/v1/submissions/" + repo.item.ID.String() + "/submit"},
			{"POST", "/api/v1/submissions/" + repo.item.ID.String() + "/sections/1"},
			{"GET", "/api/v1/documents/" + d.ID.String() + "/download"},
			{"DELETE", "/api/v1/documents/" + d.ID.String()},
			{"GET", "/api/v1/submissions/" + repo.item.ID.String() + "/documents"},
			{"POST", "/api/v1/submissions/" + repo.item.ID.String() + "/documents"},
		} {
			if w := request(pair[0], pair[1]); w.Code != 403 {
				t.Fatalf("ruta del propietario permitió acceso administrativo: %s %s: %d", pair[0], pair[1], w.Code)
			}
		}
	}
	if len(auditor.events) != 6 {
		t.Fatalf("esperados seis eventos, recibidos %d", len(auditor.events))
	}
	for _, event := range auditor.events {
		if event.ActorID != actor.ID || event.Result != "permitido" {
			t.Fatalf("evento incompleto: %+v", event)
		}
	}
	actor.Role = identity.RoleApplicant
	if w := request("GET", "/api/v1/admin/documents/"+d.ID.String()+"/download"); w.Code != 403 {
		t.Fatal("postulante pudo usar descarga administrativa")
	}
	actor.Role = identity.RoleAdministrator
	auditor.fail = true
	if w := request("GET", "/api/v1/admin/documents/"+d.ID.String()+"/download"); w.Code != 500 || strings.Contains(w.Body.String(), "https://") {
		t.Fatal("URL emitida sin auditoría")
	}
	auditor.fail = false
	repo.item.Status = submission.StatusDraft
	if w := request("GET", "/api/v1/admin/documents/"+d.ID.String()+"/download"); w.Code != 403 {
		t.Fatal("certificado de borrador expuesto")
	}
	if auditor.events[len(auditor.events)-1].Result != "rechazado" {
		t.Fatal("falta auditoría de rechazo")
	}
}

type administrativeSubmissionReader struct {
	submission.Repository
	item *submission.Submission
}

func (r *administrativeSubmissionReader) FindByID(context.Context, uuid.UUID) (*submission.Submission, error) {
	return r.item, nil
}

func TestAdministrativeDownloadsRequireRoleAndAudit(t *testing.T) {
	s, repo, _, input := fixture(t)
	d, err := s.Upload(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	actor := &identity.Applicant{ID: uuid.New(), Role: identity.RoleApplicant}
	if result, err := s.DownloadForReview(context.Background(), actor, d.ID); !errors.Is(err, submission.ErrForbidden) || result.URL != "" {
		t.Fatal("servicio permitió escalamiento")
	}
	actor.Role = identity.RoleAnalyst
	now := time.Now()
	repo.item.Status = submission.StatusSubmitted
	repo.item.SubmittedAt = &now
	repo.item.Hash = "sello"
	if result, err := s.DownloadForReview(context.Background(), actor, d.ID); !errors.Is(err, audit.ErrUnavailable) || result.URL != "" {
		t.Fatal("servicio permitió descarga sin auditoría")
	}
	auditor := &administrativeAudit{}
	s.WithAudit(auditor)
	if result, err := s.DownloadForReview(context.Background(), actor, uuid.New()); !errors.Is(err, ErrNotFound) || result.URL != "" {
		t.Fatal("documento inexistente admitido")
	}
	result, err := s.DownloadForReview(context.Background(), actor, d.ID)
	if err != nil || result.URL == "" || time.Until(result.ExpiresAt) > DownloadTTL || time.Until(result.ExpiresAt) < DownloadTTL-time.Second {
		t.Fatalf("duración incorrecta: %+v %v", result, err)
	}
}
