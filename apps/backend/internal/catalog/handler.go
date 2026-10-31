package catalog

import (
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/httpx"
	"net/http"
)

/** Handler sirve listas de nombres con las claves que consume el cliente existente. */
type Handler struct{ service *Service }

/** NewHandler conecta el servicio de catálogos con HTTP. */
func NewHandler(service *Service) *Handler { return &Handler{service: service} }

/** RegisterRoutes registra catálogos para cualquiera de los tres roles autenticados. */
func (h *Handler) RegisterRoutes(router chi.Router) {
	router.Route("/api/v1/catalogs", func(r chi.Router) {
		r.Use(identity.RequireRole(identity.RoleApplicant, identity.RoleAnalyst, identity.RoleAdministrator))
		r.Get("/countries", h.countries)
		r.Get("/provinces", h.provinces)
		r.Get("/provinces/{country}", h.provincesByPath)
		r.Get("/cities", h.cities)
		r.Get("/cities/{country}/{province}", h.citiesByPath)
		r.Get("/options/{name}", h.options)
	})
}

func (h *Handler) countries(w http.ResponseWriter, r *http.Request) {
	if len(r.URL.Query()) != 0 {
		respond(w, r, "paises", nil, ErrInvalidFilter)
		return
	}
	items, err := h.service.Countries(r.Context())
	respond(w, r, "paises", items, err)
}
func (h *Handler) provinces(w http.ResponseWriter, r *http.Request) {
	country, _, err := geographyFilter(r.URL.Query(), false)
	if err != nil {
		respond(w, r, "provincias", nil, err)
		return
	}
	items, err := h.service.Provinces(r.Context(), country)
	respond(w, r, "provincias", items, err)
}
func (h *Handler) cities(w http.ResponseWriter, r *http.Request) {
	country, province, err := geographyFilter(r.URL.Query(), true)
	if err != nil {
		respond(w, r, "cantones", nil, err)
		return
	}
	items, err := h.service.Cities(r.Context(), country, province)
	respond(w, r, "cantones", items, err)
}

func (h *Handler) provincesByPath(w http.ResponseWriter, r *http.Request) {
	if len(r.URL.Query()) != 0 {
		respond(w, r, "provincias", nil, ErrInvalidFilter)
		return
	}
	items, err := h.service.Provinces(r.Context(), chi.URLParam(r, "country"))
	respond(w, r, "provincias", items, err)
}

func (h *Handler) citiesByPath(w http.ResponseWriter, r *http.Request) {
	if len(r.URL.Query()) != 0 {
		respond(w, r, "cantones", nil, ErrInvalidFilter)
		return
	}
	items, err := h.service.Cities(r.Context(), chi.URLParam(r, "country"), chi.URLParam(r, "province"))
	respond(w, r, "cantones", items, err)
}
func (h *Handler) options(w http.ResponseWriter, r *http.Request) {
	if len(r.URL.Query()) != 0 {
		respond(w, r, "opciones", nil, ErrInvalidFilter)
		return
	}
	items, err := h.service.Options(r.Context(), chi.URLParam(r, "name"))
	respond(w, r, "opciones", items, err)
}
func respond(w http.ResponseWriter, r *http.Request, field string, items []string, err error) {
	switch {
	case errors.Is(err, ErrInvalidFilter):
		httpx.RespondError(w, r, 400, "filtro_catalogo_invalido", ErrInvalidFilter.Error())
	case errors.Is(err, ErrUnknownCatalog):
		httpx.RespondError(w, r, 404, "catalogo_no_encontrado", ErrUnknownCatalog.Error())
	case err != nil:
		httpx.RespondError(w, r, 503, "catalogo_no_disponible", "No se pudo consultar el catálogo.")
	default:
		httpx.Respond(w, r, 200, map[string][]string{field: items})
	}
}
