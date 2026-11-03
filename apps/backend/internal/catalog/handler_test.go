package catalog_test

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
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/catalog"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/cache"
)

type sessionReader struct {
	identity.SessionRepository
	actor *identity.Applicant
}

func (s sessionReader) GetWithApplicant(_ context.Context, hash string) (*identity.Session, *identity.Applicant, error) {
	if hash != identity.HashSessionToken("vigente") {
		return nil, nil, identity.ErrSessionExpired
	}
	return &identity.Session{ApplicantID: s.actor.ID, ExpiresAt: time.Now().Add(time.Hour)}, s.actor, nil
}

func TestCatalogHTTPAuthenticationFiltersAndErrorContract(t *testing.T) {
	repo, cached := &catalogRepository{}, newCache()
	actor := &identity.Applicant{ID: uuid.New(), Role: identity.RoleApplicant}
	router := chi.NewRouter()
	router.Use(identity.RequireSession(identity.NewService(identity.ServiceDependencies{Sessions: sessionReader{actor: actor}})))
	catalog.NewHandler(catalog.NewService(repo, cache.NewCatalogStore(cached))).RegisterRoutes(router)
	request := func(path, token string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", path, nil)
		if token != "" {
			r.AddCookie(&http.Cookie{Name: "session", Value: token})
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}
	for _, token := range []string{"", "expirada"} {
		if w := request("/api/v1/catalogs/countries", token); w.Code != 401 {
			t.Fatal("catálogos accesibles sin sesión")
		}
	}
	for _, role := range []string{identity.RoleApplicant, identity.RoleAnalyst, identity.RoleAdministrator} {
		actor.Role = role
		for _, path := range []string{"countries", "provinces?country=EC", "cities?country=Ecuador&province=Pichincha", "options/instituciones", "provinces/Ecuador", "cities/Ecuador/Pichincha"} {
			w := request("/api/v1/catalogs/"+path, "vigente")
			field := "paises"
			if strings.HasPrefix(path, "provinces") {
				field = "provincias"
			} else if strings.HasPrefix(path, "cities") {
				field = "cantones"
			} else if strings.HasPrefix(path, "options") {
				field = "opciones"
			}
			var values map[string][]string
			if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &values) != nil || len(values[field]) == 0 {
				t.Fatalf("respuesta incompatible con el cliente: %d %s", w.Code, w.Body.String())
			}
		}
	}
	for _, path := range []string{
		"provinces", "provinces?country=", "cities?country=EC", "cities?country=EC&province=Quito",
		"provinces?country=Atlantis", "provinces?country=EC&country=Argentina",
		"cities?country=EC&province=Pichincha&province=Azuay", "provinces?country=EC&otro=1",
		"countries?country=EC", "options/instituciones?otro=1",
		"provinces/Ecuador?country=Argentina", "cities/Ecuador/Pichincha?province=Azuay",
	} {
		if w := request("/api/v1/catalogs/"+path, "vigente"); w.Code != 400 || !strings.Contains(w.Body.String(), "filtro_catalogo_invalido") {
			t.Fatalf("filtro inválido admitido: %s %d %s", path, w.Code, w.Body.String())
		}
	}
	if w := request("/api/v1/catalogs/options/no-existe", "vigente"); w.Code != 404 || !strings.Contains(w.Body.String(), "catalogo_no_encontrado") {
		t.Fatal("catálogo desconocido admitido")
	}
	if w := request("/api/v1/catalogs/provinces?country=Argentina", "vigente"); w.Code != 200 || strings.TrimSpace(w.Body.String()) != `{"provincias":[]}` {
		t.Fatal("geografía extranjera debe permitir texto libre")
	}
	actor.Role = "superadmin"
	if w := request("/api/v1/catalogs/countries", "vigente"); w.Code != 403 {
		t.Fatal("rol desconocido admitido")
	}
	actor.Role = identity.RoleApplicant
	repo.fail, cached.failGet = true, true
	if w := request("/api/v1/catalogs/countries", "vigente"); w.Code != 503 || !strings.Contains(w.Body.String(), "catalogo_no_disponible") || strings.Contains(w.Body.String(), "base de datos") {
		t.Fatal("error de fuente debe ser controlado y no filtrarse")
	}
}

func TestCatalogHandlerRequiresIdentityEvenWithoutOuterRouter(t *testing.T) {
	router := chi.NewRouter()
	catalog.NewHandler(catalog.NewService(&catalogRepository{}, nil)).RegisterRoutes(router)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/catalogs/countries", nil))
	if w.Code != 401 {
		t.Fatal("el contexto permitió uso sin autenticación")
	}
}
