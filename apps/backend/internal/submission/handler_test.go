package submission

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
)

type sessionReader struct {
	identity.SessionRepository
	owner uuid.UUID
}

func (r sessionReader) GetWithApplicant(_ context.Context, hash string) (*identity.Session, *identity.Applicant, error) {
	id := r.owner
	if hash == identity.HashSessionToken("otra-sesion") {
		id = uuid.New()
	} else if hash != identity.HashSessionToken("sesion-prueba") {
		return nil, nil, identity.ErrSessionExpired
	}
	return &identity.Session{ApplicantID: id, ExpiresAt: time.Now().Add(time.Hour)}, &identity.Applicant{ID: id, IDCard: "1710034008"}, nil
}

func httpFixture(t *testing.T, full bool) (http.Handler, *memoryRepository) {
	t.Helper()
	s, repo, _ := serviceFixture(t, full)
	auth := identity.NewService(identity.ServiceDependencies{Sessions: sessionReader{owner: repo.item.ApplicantID}})
	router := chi.NewRouter()
	router.Use(identity.RequireSession(auth))
	NewHandler(s).RegisterRoutes(router)
	return router, repo
}
func performRequest(handler http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	if token != "" {
		r.AddCookie(&http.Cookie{Name: "session", Value: token})
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	return w
}

func TestSubmissionHTTPAuthorization(t *testing.T) {
	handler, repo := httpFixture(t, true)
	base := "/api/v1/submissions/" + repo.item.ID.String()
	for _, methodPath := range [][2]string{{"GET", base}, {"PUT", base}, {"DELETE", base}, {"POST", base + "/sections/1"}, {"POST", base + "/submit"}} {
		for _, tokenStatus := range []struct {
			token  string
			status int
		}{{"", 401}, {"otra-sesion", 403}} {
			w := performRequest(handler, methodPath[0], methodPath[1], `{}`, tokenStatus.token)
			if w.Code != tokenStatus.status {
				t.Fatalf("%s %s: esperado %d, obtenido %d: %s", methodPath[0], methodPath[1], tokenStatus.status, w.Code, w.Body)
			}
		}
	}
	for _, path := range []string{"/api/v1/submissions/", "/api/v1/submissions"} {
		w := performRequest(handler, "POST", path, `{"applicantId":"`+uuid.NewString()+`","content":{}}`, "sesion-prueba")
		if w.Code != 403 {
			t.Fatalf("titularidad de creación: %d", w.Code)
		}
	}
	w := performRequest(handler, "GET", "/api/v1/submissions?applicantId="+uuid.NewString(), "", "sesion-prueba")
	if w.Code != 403 {
		t.Fatalf("listado de tercero: %d", w.Code)
	}
}

func TestSubmissionHTTPSaveAndSubmit(t *testing.T) {
	handler, repo := httpFixture(t, true)
	base := "/api/v1/submissions/" + repo.item.ID.String()
	w := performRequest(handler, "POST", base+"/sections/1", `{"nombres":"Persona actualizada"}`, "sesion-prueba")
	if w.Code != 200 {
		t.Fatalf("guardar: %d %s", w.Code, w.Body)
	}
	var saved Response
	if err := json.Unmarshal(w.Body.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Progress.Completed != 9 || saved.Progress.Percentage != 100 {
		t.Fatalf("avance: %+v", saved.Progress)
	}
	w = performRequest(handler, "POST", base+"/submit", `{}`, "sesion-prueba")
	if w.Code != 200 {
		t.Fatalf("enviar: %d %s", w.Code, w.Body)
	}
	var sealed Response
	if err := json.Unmarshal(w.Body.Bytes(), &sealed); err != nil {
		t.Fatal(err)
	}
	if sealed.Status != StatusSubmitted || len(sealed.Hash) != 64 || sealed.SubmittedAt == nil {
		t.Fatal("respuesta sin sello")
	}
	for _, mp := range [][2]string{{"PUT", base}, {"DELETE", base}, {"POST", base + "/sections/1"}, {"POST", base + "/submit"}} {
		w := performRequest(handler, mp[0], mp[1], `{"content":{}}`, "sesion-prueba")
		if mp[0] == "POST" {
			body := `{}`
			w = performRequest(handler, mp[0], mp[1], body, "sesion-prueba")
		}
		if w.Code != 409 {
			t.Fatalf("bloqueo %s %s: %d %s", mp[0], mp[1], w.Code, w.Body)
		}
	}
}

func TestSubmissionHTTPValidation(t *testing.T) {
	handler, repo := httpFixture(t, false)
	base := "/api/v1/submissions/" + repo.item.ID.String()
	for _, test := range []struct {
		path, body string
		status     int
		code       string
	}{
		{base + "/sections/0", `{}`, 400, "seccion_invalida"},
		{base + "/sections/no", `{}`, 400, "seccion_invalida"},
		{base + "/sections/1", `{"nombres":`, 400, "cuerpo_invalido"},
		{base + "/sections/1", `{} {}`, 400, "cuerpo_invalido"},
		{base + "/sections/1", `{"campoDesconocido":true}`, 400, "contenido_invalido"},
		{base + "/sections/1", `{"nacionalidad":"Extranjera"}`, 422, "regla_de_negocio"},
		{base + "/sections/2", `{"pais":"Ecuador","provincia":"Pichincha","canton":"Guayaquil"}`, 422, "regla_de_negocio"},
		{base + "/submit", `{}`, 422, "regla_de_negocio"},
		{base + "/submit", `{"desconocido":true}`, 400, "cuerpo_invalido"},
		{base + "/submit", `null`, 400, "cuerpo_invalido"},
		{base + "/sections/1", `{"cedula":"1710034016"}`, 403, "acceso_denegado"},
	} {
		w := performRequest(handler, "POST", test.path, test.body, "sesion-prueba")
		if w.Code != test.status || !strings.Contains(w.Body.String(), test.code) {
			t.Fatalf("%s: esperado %d %s, obtenido %d %s", test.path, test.status, test.code, w.Code, w.Body)
		}
	}
}
