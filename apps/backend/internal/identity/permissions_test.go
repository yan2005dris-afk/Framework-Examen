package identity

import (
	"context"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type roleSessionStore struct {
	SessionRepository
	actor *Applicant
}

func (s *roleSessionStore) GetWithApplicant(_ context.Context, hash string) (*Session, *Applicant, error) {
	if hash != HashSessionToken("personal") {
		return nil, nil, ErrSessionExpired
	}
	return &Session{ApplicantID: s.actor.ID, ExpiresAt: time.Now().Add(time.Hour)}, s.actor, nil
}

func TestAdministrativePermissionMatrixAndLiveRole(t *testing.T) {
	actor := &Applicant{ID: uuid.New()}
	sessions := &roleSessionStore{actor: actor}
	router := chi.NewRouter()
	router.Use(RequireSession(NewService(ServiceDependencies{Sessions: sessions})))
	router.Use(ProtectAdministration)
	router.HandleFunc("/api/v1/admin/*", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	for _, role := range []string{RoleApplicant, RoleAnalyst, RoleAdministrator, "superadmin", ""} {
		actor.Role = role
		for _, scope := range []string{"submissions", "documents", "phases", "question-bank", "grades", "reconsiderations", "applicants", "unknown"} {
			t.Run(role+"/"+scope, func(t *testing.T) {
				want := 403
				if role == RoleAdministrator && scope != "unknown" || role == RoleAnalyst && (scope == "submissions" || scope == "documents" || scope == "grades" || scope == "reconsiderations") {
					want = 204
				}
				r := httptest.NewRequest("POST", "/api/v1/admin/"+scope+"/operation", nil)
				r.AddCookie(&http.Cookie{Name: "session", Value: "personal"})
				w := httptest.NewRecorder()
				router.ServeHTTP(w, r)
				if w.Code != want {
					t.Fatalf("esperado %d; recibido %d: %s", want, w.Code, w.Body.String())
				}
			})
		}
	}
	for _, token := range []string{"", "expirada"} {
		r := httptest.NewRequest("GET", "/api/v1/admin/submissions/id", nil)
		if token != "" {
			r.AddCookie(&http.Cookie{Name: "session", Value: token})
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != 401 {
			t.Fatalf("sesión inválida admitida: %d", w.Code)
		}
	}
	// La misma cookie pierde privilegios en la siguiente petición tras una revocación.
	actor.Role = RoleApplicant
	r := httptest.NewRequest("GET", "/api/v1/admin/submissions/id", nil)
	r.AddCookie(&http.Cookie{Name: "session", Value: "personal"})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("se conservaron privilegios revocados")
	}
}

func TestRequireRoleDeniesMissingIdentityEmptyAndUnknownPolicies(t *testing.T) {
	for _, roles := range [][]string{nil, {"superadmin"}, {RoleAdministrator}} {
		h := RequireRole(roles...)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
		for _, actor := range []*Applicant{nil, {ID: uuid.Nil, Role: RoleAdministrator}, {ID: uuid.New(), Role: "superadmin"}} {
			r := httptest.NewRequest("GET", "/", nil)
			if actor != nil {
				r = r.WithContext(context.WithValue(r.Context(), applicantContextKey, actor))
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != 401 && w.Code != 403 {
				t.Fatalf("política desconocida admitida: %d", w.Code)
			}
		}
	}
	if HasPermission(&Applicant{ID: uuid.New(), Role: RoleAdministrator}, Permission("desconocido")) {
		t.Fatal("permiso desconocido concedido")
	}
}

type roleWriter struct {
	calls int
	err   error
}

func (s *roleWriter) ChangeRole(_ context.Context, actor, target uuid.UUID, role, reason string) (*Applicant, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	return &Applicant{ID: target, Role: role}, nil
}

func TestRoleManagementCannotEscalateAndValidatesInput(t *testing.T) {
	writer := &roleWriter{}
	service := NewRoleService(writer)
	for _, actor := range []*Applicant{nil, {ID: uuid.New(), Role: RoleApplicant}, {ID: uuid.New(), Role: RoleAnalyst}} {
		if _, err := service.Change(context.Background(), actor, uuid.New(), RoleAdministrator, "Gestión del concurso"); !errors.Is(err, ErrRoleForbidden) {
			t.Fatalf("escalamiento admitido: %v", err)
		}
	}
	admin := &Applicant{ID: uuid.New(), Role: RoleAdministrator}
	for _, change := range [][2]string{{"superadmin", "Asignación"}, {RoleAdministrator, " "}, {RoleAnalyst, strings.Repeat("x", 501)}, {RoleApplicant, "motivo\ninyectado"}} {
		if _, err := service.Change(context.Background(), admin, uuid.New(), change[0], change[1]); !errors.Is(err, ErrInvalidRoleChange) {
			t.Fatalf("entrada inválida aceptada: %v", err)
		}
	}
	if writer.calls != 0 {
		t.Fatal("una entrada rechazada alcanzó la persistencia")
	}
	for _, role := range []string{RoleApplicant, RoleAnalyst, RoleAdministrator} {
		item, err := service.Change(context.Background(), admin, uuid.New(), role, "Asignación autorizada")
		if err != nil || item.Role != role {
			t.Fatalf("cambio válido rechazado: %v", err)
		}
	}
	writer.err = errors.New("fallo de auditoría transaccional")
	if item, err := service.Change(context.Background(), admin, uuid.New(), RoleAnalyst, "Asignación"); item != nil || err == nil {
		t.Fatal("se ocultó un fallo de auditoría")
	}
}

func TestRoleManagementHTTPAuthorizationAndStrictBody(t *testing.T) {
	actor := &Applicant{ID: uuid.New(), Role: RoleAdministrator}
	writer := &roleWriter{}
	router := chi.NewRouter()
	router.Use(RequireSession(NewService(ServiceDependencies{Sessions: &roleSessionStore{actor: actor}})))
	RegisterRoleRoutes(router, NewRoleService(writer))
	for _, tc := range []struct {
		role, body string
		want       int
	}{
		{RoleApplicant, `{"rol":"administrador","motivo":"Asignación"}`, 403},
		{RoleAnalyst, `{"rol":"administrador","motivo":"Asignación"}`, 403},
		{RoleAdministrator, `{"rol":"administrador","motivo":"Asignación","actor":"falso"}`, 400},
		{RoleAdministrator, `{"rol":"analista","motivo":"Asignación"} {}`, 400},
		{RoleAdministrator, `{"rol":"desconocido","motivo":"Asignación"}`, 422},
		{RoleAdministrator, `{"rol":"analista","motivo":"Asignación"}`, 200},
	} {
		actor.Role = tc.role
		r := httptest.NewRequest("PATCH", "/api/v1/admin/applicants/"+uuid.NewString()+"/role", strings.NewReader(tc.body))
		r.AddCookie(&http.Cookie{Name: "session", Value: "personal"})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != tc.want {
			t.Fatalf("esperado %d, recibido %d: %s", tc.want, w.Code, w.Body.String())
		}
	}
	if writer.calls != 1 {
		t.Fatalf("se persistieron %d cambios en lugar de uno", writer.calls)
	}
}
