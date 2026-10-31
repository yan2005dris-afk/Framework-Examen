package competition

import (
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/httpx"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/shared/domain"
	"io"
	"net/http"
	"time"
)

type Handler struct{ service *Service }

func NewHandler(s *Service) *Handler { return &Handler{s} }

/** RegisterPublic publica sólo información aprobada y publicada, sin identidad del operador. */
func (h *Handler) RegisterPublic(r chi.Router) {
	r.Get("/api/v1/competitions", h.listPublic)
	r.Get("/api/v1/competitions/{id}", h.publicDetail)
}
func (h *Handler) RegisterAdministration(r chi.Router) {
	r.With(identity.RequirePermission(identity.PermissionPhases)).Post("/api/v1/admin/competitions", h.create)
	r.With(identity.RequirePermission(identity.PermissionPhases)).Get("/api/v1/admin/competitions/{id}", h.detail)
	r.With(identity.RequirePermission(identity.PermissionPhases)).Put("/api/v1/admin/competitions/{id}/configuration", h.configure)
}
func decodeRequest(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return ErrInvalid
	}
	if d.Decode(new(json.RawMessage)) != io.EOF {
		return ErrInvalid
	}
	return nil
}
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var c Competition
	if decodeRequest(r, &c) != nil || c.ID != uuid.Nil || c.ApprovedAt != nil || c.ApprovedBy != nil {
		writeError(w, r, ErrInvalid)
		return
	}
	id, err := uuid.NewV7()
	if err != nil {
		writeError(w, r, err)
		return
	}
	c.ID = id
	actor, _ := identity.ApplicantFromContext(r.Context())
	if err = h.service.Create(r.Context(), actor, &c); err != nil {
		writeError(w, r, err)
		return
	}
	httpx.Respond(w, r, 201, c)
}
func (h *Handler) detail(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, r, ErrInvalid)
		return
	}
	c, err := h.service.repo.Get(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.Respond(w, r, 200, c)
}
func (h *Handler) configure(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, r, ErrInvalid)
		return
	}
	var config Configuration
	if decodeRequest(r, &config) != nil {
		writeError(w, r, ErrInvalid)
		return
	}
	actor, _ := identity.ApplicantFromContext(r.Context())
	c, err := h.service.Configure(r.Context(), actor, id, config)
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpx.Respond(w, r, 200, c)
}

type publicCompetition struct {
	ID          uuid.UUID `json:"id"`
	Code        string    `json:"codigo"`
	Title       string    `json:"titulo"`
	OpensAt     time.Time `json:"apertura"`
	ClosesAt    time.Time `json:"cierre"`
	Sectors     []string  `json:"sectores"`
	Phases      []Phase   `json:"fases"`
	FormEnabled bool      `json:"ef12Disponible"`
	TimeZone    string    `json:"zonaHoraria"`
	Interval    string    `json:"limites"`
}

func publicView(c Competition) publicCompetition {
	phases := []Phase{}
	for _, p := range c.Phases {
		if p.Published {
			phases = append(phases, p)
		}
	}
	return publicCompetition{c.ID, c.Code, c.Title, c.OpensAt, c.ClosesAt, c.Sectors, phases, c.Writable(time.Now(), ""), TimeZone, "[apertura,cierre)"}
}
func (h *Handler) publicDetail(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, r, ErrNotFound)
		return
	}
	c, err := h.service.repo.Get(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	if !c.Approved || !c.Published {
		writeError(w, r, ErrNotFound)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.Respond(w, r, 200, publicView(*c))
}
func (h *Handler) listPublic(w http.ResponseWriter, r *http.Request) {
	if httpx.ValidateQuery(r, "limit", "cursor") != nil {
		writeError(w, r, ErrInvalid)
		return
	}
	q, err := httpx.ParseCursorRequest(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	before := uuid.Nil
	if q.Cursor != nil {
		marker, ok := q.Cursor.Value(0)
		value, ok2 := q.Cursor.Value(1)
		if !ok || !ok2 || marker != "public-competitions-v1" {
			writeError(w, r, domain.ErrInvalidCursor)
			return
		}
		before, err = uuid.Parse(value)
		if err != nil {
			writeError(w, r, domain.ErrInvalidCursor)
			return
		}
	}
	records, err := h.service.repo.ListPublished(r.Context(), q.Limit+1, before)
	if err != nil {
		writeError(w, r, err)
		return
	}
	more := len(records) > q.Limit
	if more {
		records = records[:q.Limit]
	}
	items := []publicCompetition{}
	for _, c := range records {
		items = append(items, publicView(c))
	}
	cursor := ""
	if more {
		cursor = domain.NewCursor("public-competitions-v1", records[len(records)-1].ID.String()).Encode()
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.Respond(w, r, 200, domain.NewCursorPage(items, cursor, more, q.Limit))
}
func writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrForbidden):
		httpx.RespondError(w, r, 403, "acceso_denegado", ErrForbidden.Error())
	case errors.Is(err, ErrInvalid):
		httpx.RespondError(w, r, 422, "convocatoria_invalida", ErrInvalid.Error())
	case errors.Is(err, ErrNotFound):
		httpx.RespondError(w, r, 404, "convocatoria_no_encontrada", ErrNotFound.Error())
	case errors.Is(err, ErrVersion):
		httpx.RespondError(w, r, 409, "version_convocatoria", ErrVersion.Error())
	case errors.Is(err, ErrDuplicate):
		httpx.RespondError(w, r, 409, "convocatoria_duplicada", ErrDuplicate.Error())
	case errors.Is(err, domain.ErrInvalidCursor):
		httpx.RespondError(w, r, 400, "cursor_invalido", "La paginación no es válida.")
	default:
		httpx.RespondError(w, r, 503, "convocatorias_no_disponibles", "No se pudo completar la operación de convocatorias.")
	}
}
