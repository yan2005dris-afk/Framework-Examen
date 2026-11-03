package evaluation

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/shared/domain"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type memoryRepo struct {
	Repository
	c     *Case
	calls int
}

func (r *memoryRepo) ReadCase(context.Context, uuid.UUID, uuid.UUID) (*Case, error) {
	copy := *r.c
	copy.State = r.c.State
	return &copy, nil
}
func (r *memoryRepo) ListHistory(context.Context, uuid.UUID, domain.Position, int) ([]HistoryEntry, error) {
	return []HistoryEntry{{ID: uuid.New(), Action: "psicotecnica.registrada", Data: json.RawMessage(`{"motivo":"HISTORIAL-PRIVADO"}`)}, {ID: uuid.New(), Action: "psicotecnica.publicada", Data: json.RawMessage(`{"version":1}`)}}, nil
}
func (r *memoryRepo) Execute(_ context.Context, _ *identity.Applicant, _ uuid.UUID, _ Command, f func(*Case) (Change, error)) (Receipt, error) {
	r.calls++
	change, err := f(r.c)
	return Receipt{Data: change.Record, Applied: true}, err
}

type sessionStub struct {
	identity.SessionRepository
	actor *identity.Applicant
}

func (s sessionStub) GetWithApplicant(context.Context, string) (*identity.Session, *identity.Applicant, error) {
	return &identity.Session{ExpiresAt: time.Now().Add(time.Hour)}, s.actor, nil
}

type auditStub struct{ fail bool }

func (a auditStub) Record(context.Context, audit.Event) error {
	if a.fail {
		return audit.ErrUnavailable
	}
	return nil
}
func httpFixture(role string, fail bool) (http.Handler, *memoryRepo, *identity.Applicant) {
	a := &identity.Applicant{ID: uuid.New(), Role: role}
	repo := &memoryRepo{c: &Case{Snapshot: &submission.Submission{ID: uuid.New(), ApplicantID: a.ID}, State: State{Psychometric: &Psychometric{Motive: "REGISTRO-PRIVADO"}}, Sources: map[string]Result{}, PhaseStarts: map[string]time.Time{}, PhaseEnds: map[string]time.Time{}}}
	r := chi.NewRouter()
	r.Use(identity.RequireSession(identity.NewService(identity.ServiceDependencies{Sessions: sessionStub{actor: a}})))
	r.Use(identity.ProtectAdministration)
	NewHandler(NewService(repo, nil, auditStub{fail})).RegisterRoutes(r)
	return r, repo, a
}
func request(h http.Handler, method, path, body string, k uuid.UUID, cookie bool) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if cookie {
		r.AddCookie(&http.Cookie{Name: "session", Value: "qa"})
	}
	if k != uuid.Nil {
		r.Header.Set("Idempotency-Key", k.String())
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func TestEvaluationHTTPRolesOwnershipAndAudit(t *testing.T) {
	for _, role := range []string{identity.RoleApplicant, identity.RoleAnalyst, identity.RoleAdministrator} {
		t.Run(role, func(t *testing.T) {
			h, repo, _ := httpFixture(role, false)
			id := repo.c.Snapshot.ID.String()
			w := request(h, "GET", "/api/v1/admin/grades/submissions/"+id, "", uuid.Nil, true)
			want := 200
			if role == identity.RoleApplicant {
				want = 403
			}
			if w.Code != want {
				t.Fatal(w.Code, w.Body.String())
			}
			for _, route := range []string{"psychometric/publish", "consolidate"} {
				w = request(h, "POST", "/api/v1/admin/grades/submissions/"+id+"/"+route, `{"puntajeTeorico":100}`, uuid.New(), true)
				want = 422
				if role != identity.RoleAdministrator {
					want = 403
				}
				if w.Code != want {
					t.Fatal(route, w.Code, w.Body.String())
				}
			}
			w = request(h, "GET", "/api/v1/evaluations/submissions/"+id, "", uuid.Nil, true)
			if w.Code != 200 || strings.Contains(w.Body.String(), "REGISTRO-PRIVADO") {
				t.Fatal("registro no publicado expuesto", w.Body.String())
			}
			w = request(h, "GET", "/api/v1/evaluations/submissions/"+id+"/history", "", uuid.Nil, true)
			if w.Code != 200 || strings.Contains(w.Body.String(), "HISTORIAL-PRIVADO") || !strings.Contains(w.Body.String(), `"version":1`) {
				t.Fatal("historial propio filtrado incorrectamente", w.Code, w.Body.String())
			}
			if role != identity.RoleApplicant {
				w = request(h, "GET", "/api/v1/admin/grades/submissions/"+id+"/history", "", uuid.Nil, true)
				if w.Code != 200 || !strings.Contains(w.Body.String(), "HISTORIAL-PRIVADO") {
					t.Fatal("falta detalle histórico autorizado", w.Code, w.Body.String())
				}
			}
		})
	}
	h, repo, _ := httpFixture(identity.RoleAdministrator, true)
	w := request(h, "GET", "/api/v1/admin/grades/submissions/"+repo.c.Snapshot.ID.String(), "", uuid.Nil, true)
	if w.Code != 503 || strings.Contains(w.Body.String(), "REGISTRO-PRIVADO") {
		t.Fatal("lectura sin auditoría", w.Code)
	}
	h, repo, _ = httpFixture(identity.RoleApplicant, false)
	repo.c.Snapshot.ApplicantID = uuid.New()
	w = request(h, "GET", "/api/v1/evaluations/submissions/"+repo.c.Snapshot.ID.String(), "", uuid.Nil, true)
	if w.Code != 403 {
		t.Fatal("lectura de otro titular", w.Code)
	}
}
func TestEvaluationHTTPStrictBodiesAndNoEditableTheory(t *testing.T) {
	h, repo, _ := httpFixture(identity.RoleAdministrator, false)
	path := "/api/v1/admin/grades/submissions/" + repo.c.Snapshot.ID.String()
	for _, body := range []string{`{"notaTeorica":100}`, `{} {}`, `{}` + strings.Repeat(" ", 129<<10)} {
		w := request(h, "POST", path+"/consolidate", body, uuid.New(), true)
		if w.Code != 422 || repo.calls != 0 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	w := request(h, "POST", path+"/consolidate", `{}`, uuid.Nil, true)
	if w.Code != 422 || repo.calls != 0 {
		t.Fatal("sin clave de reintento", w.Code)
	}
	w = request(h, "POST", path+"/theoretical", `{"puntaje":100}`, uuid.New(), true)
	if w.Code != 404 {
		t.Fatal("existe ruta para nota teórica libre", w.Code)
	}
	w = request(h, "GET", path, "", uuid.Nil, false)
	if w.Code != 401 {
		t.Fatal(w.Code)
	}
}
func TestReconsiderationActorAndDeadlineStateMachine(t *testing.T) {
	now := time.Date(2026, 10, 31, 15, 0, 0, 0, time.UTC)
	term := Term{Quantity: 60, Unit: "minutos_transcurridos"}
	policy := Policy{Definition: Definition{Reconsideration: &AppealRules{Applicant: term, Talent: term, Commission: term, Calendar: syntheticCalendar()}}}
	id, owner := uuid.New(), uuid.New()
	source := Result{ID: uuid.New(), SubmissionID: id, Phase: "meritos", Version: 1, SnapshotHash: "sello"}
	deadline := now.Add(time.Hour)
	appeal := &Appeal{ID: uuid.New(), SubmissionID: id, Version: 1, Status: "recibida", Original: source, Policy: policy, ApplicantID: owner, TalentDeadline: deadline, WindowEnd: now.Add(4 * time.Hour)}
	c := &Case{Snapshot: &submission.Submission{ID: id, Hash: "sello"}, Now: now, Appeal: appeal, PhaseStarts: map[string]time.Time{"reconsideracion": now.Add(-time.Hour)}, PhaseEnds: map[string]time.Time{"reconsideracion": now.Add(4 * time.Hour)}, Sources: map[string]Result{"meritos": source}}
	repo := &memoryRepo{c: c}
	s := NewService(repo, nil, nil)
	analyst := &identity.Applicant{ID: uuid.New(), Role: identity.RoleAnalyst}
	admin := &identity.Applicant{ID: uuid.New(), Role: identity.RoleAdministrator}
	input := AttendInput{PreviousVersion: 1, Observation: "Observación QA", Recommendation: "confirmar"}
	c.Now = deadline
	if _, err := s.Attend(context.Background(), analyst, id, appeal.ID, uuid.New(), input); !errors.Is(err, ErrDeadline) {
		t.Fatal("atención tardía aceptada", err)
	}
	c.Now = now
	if _, err := s.Resolve(context.Background(), admin, id, appeal.ID, uuid.New(), ResolveInput{PreviousVersion: 1, Decision: "confirmar", Motive: "QA"}); !errors.Is(err, ErrState) {
		t.Fatal("Comisión omitió TH", err)
	}
	if _, err := s.Attend(context.Background(), analyst, id, appeal.ID, uuid.New(), input); err != nil {
		t.Fatal(err)
	}
	if appeal.Status != "atendida" || appeal.CommissionDeadline == nil {
		t.Fatal("no transitó a atención")
	}
	resolution := ResolveInput{PreviousVersion: 2, Decision: "rectificar", Motive: "Resolución QA"}
	fake := uuid.New()
	resolution.ReplacementID = &fake
	if _, err := s.Resolve(context.Background(), admin, id, appeal.ID, uuid.New(), resolution); !errors.Is(err, ErrConflict) {
		t.Fatal("aceptó rectificación no publicada", err)
	}
	corrected := source
	corrected.ID = uuid.New()
	corrected.Version = 2
	score, maximum := int64(90), int64(100)
	originalScore := int64(80)
	source.Score = &originalScore
	source.Maximum = &maximum
	appeal.Original = source
	corrected.Score = &score
	corrected.Maximum = &maximum
	corrected.CanAdvance = true
	c.Sources["meritos"] = corrected
	resolution.ReplacementID = &corrected.ID
	c.Now = *appeal.CommissionDeadline
	if _, err := s.Resolve(context.Background(), admin, id, appeal.ID, uuid.New(), resolution); !errors.Is(err, ErrDeadline) {
		t.Fatal("resolución tardía aceptada", err)
	}
	c.Now = now
	if _, err := s.Resolve(context.Background(), admin, id, appeal.ID, uuid.New(), resolution); err != nil {
		t.Fatal(err)
	}
	if appeal.Status != "resuelta" || appeal.Resolution.ScoreDelta == nil || *appeal.Resolution.ScoreDelta != 10 || appeal.Original.ID != source.ID {
		t.Fatal("efecto/original incorrecto")
	}
}
