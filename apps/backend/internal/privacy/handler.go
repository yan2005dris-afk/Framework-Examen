package privacy

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/httpx"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/shared/domain"
	"io"
	"net/http"
	"strconv"
	"time"
)

type Handler struct {
	repo  Repository
	audit audit.Recorder
}

func NewHandler(r Repository, a audit.Recorder) *Handler { return &Handler{r, a} }
func (h *Handler) RegisterPublic(r chi.Router) {
	r.Get("/api/v1/public/privacy/competitions/{id}", h.current)
	r.Get("/api/v1/public/privacy/policies/{id}", h.get)
}
func (h *Handler) RegisterAdministration(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(identity.RequirePermission(identity.PermissionPhases))
		r.Post("/api/v1/admin/privacy/competitions/{id}/policies", h.publish)
		r.Post("/api/v1/admin/privacy/competitions/{id}/restriction", h.block)
		r.Get("/api/v1/admin/privacy/competitions/{id}/policies", h.list)
		r.Get("/api/v1/admin/privacy/competitions/{id}/retention-review", h.reviews)
	})
}
func identifier(r *http.Request) (uuid.UUID, error) {
	v, e := uuid.Parse(chi.URLParam(r, "id"))
	if e != nil || v == uuid.Nil {
		return uuid.Nil, ErrInvalid
	}
	return v, nil
}
func actor(r *http.Request) *identity.Applicant {
	a, _ := identity.ApplicantFromContext(r.Context())
	return a
}
func decode(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	raw, e := io.ReadAll(io.LimitReader(r.Body, 160001))
	if e != nil || len(raw) > 160000 {
		return ErrInvalid
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(v) != nil || d.Decode(new(json.RawMessage)) != io.EOF {
		return ErrInvalid
	}
	return nil
}
func respond(w http.ResponseWriter, r *http.Request, v interface{}, err error) {
	w.Header().Set("Cache-Control", "no-store")
	if err == nil {
		httpx.Respond(w, r, 200, v)
		return
	}
	status, code := 503, "privacidad_no_disponible"
	switch {
	case errors.Is(err, ErrForbidden):
		status, code = 403, "privacidad_prohibida"
	case errors.Is(err, ErrInvalid), errors.Is(err, domain.ErrInvalidCursor):
		status, code = 422, "privacidad_invalida"
	case errors.Is(err, ErrConflict):
		status, code = 409, "version_privacidad_conflicto"
	case errors.Is(err, ErrNotFound):
		status, code = 404, "politica_no_encontrada"
	case errors.Is(err, ErrDenied):
		status, code = 409, "privacidad_pendiente"
	}
	httpx.RespondError(w, r, status, code, "No se pudo confirmar la política o autorización de tratamiento.")
}
func (h *Handler) current(w http.ResponseWriter, r *http.Request) {
	id, e := identifier(r)
	if e != nil {
		respond(w, r, nil, e)
		return
	}
	p, e := h.repo.Current(r.Context(), id)
	respond(w, r, p, e)
}
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, e := identifier(r)
	if e != nil {
		respond(w, r, nil, e)
		return
	}
	p, e := h.repo.Get(r.Context(), id)
	respond(w, r, p, e)
}
func (h *Handler) publish(w http.ResponseWriter, r *http.Request) {
	id, e := identifier(r)
	var input PublishInput
	if e != nil || decode(r, &input) != nil {
		respond(w, r, nil, ErrInvalid)
		return
	}
	p, e := h.repo.Publish(r.Context(), actor(r), id, input)
	respond(w, r, p, e)
}
func (h *Handler) block(w http.ResponseWriter, r *http.Request) {
	id, e := identifier(r)
	var input BlockInput
	if e != nil || decode(r, &input) != nil {
		respond(w, r, nil, ErrInvalid)
		return
	}
	e = h.repo.Block(r.Context(), actor(r), id, input)
	respond(w, r, map[string]bool{"restringido": input.Blocked}, e)
}
func (h *Handler) readAudit(r *http.Request, id uuid.UUID, e error) error {
	a := actor(r)
	if !identity.HasPermission(a, identity.PermissionPhases) {
		return ErrForbidden
	}
	return audit.Record(r.Context(), h.audit, audit.Event{ActorID: a.ID, Role: a.Role, ResourceID: id, ResourceType: "convocatoria", Action: "privacidad.consultada", CreatedAt: time.Now().UTC()}, e)
}
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	id, e := identifier(r)
	if e != nil || httpx.ValidateQuery(r, "limit", "cursor") != nil {
		respond(w, r, nil, ErrInvalid)
		return
	}
	q, e := httpx.ParseCursorRequest(r)
	if e != nil {
		respond(w, r, nil, ErrInvalid)
		return
	}
	before := int64(0)
	scope := "privacy-policies:" + id.String()
	if q.Cursor != nil {
		s, ok := q.Cursor.Value(0)
		v, ok2 := q.Cursor.Value(1)
		_, extra := q.Cursor.Value(2)
		before, e = strconv.ParseInt(v, 10, 64)
		if !ok || !ok2 || extra || s != scope || e != nil || before < 1 {
			respond(w, r, nil, ErrInvalid)
			return
		}
	}
	items, e := h.repo.List(r.Context(), id, before, q.Limit+1)
	e = h.readAudit(r, id, e)
	if e != nil {
		respond(w, r, nil, e)
		return
	}
	more := len(items) > q.Limit
	next := ""
	if more {
		items = items[:q.Limit]
		next = domain.NewCursor(scope, strconv.FormatInt(items[len(items)-1].Version, 10)).Encode()
	}
	respond(w, r, domain.NewCursorPage(items, next, more, q.Limit), nil)
}
func (h *Handler) reviews(w http.ResponseWriter, r *http.Request) {
	id, e := identifier(r)
	if e != nil || httpx.ValidateQuery(r, "limit", "cursor") != nil {
		respond(w, r, nil, ErrInvalid)
		return
	}
	q, e := httpx.ParseCursorRequest(r)
	if e != nil {
		respond(w, r, nil, ErrInvalid)
		return
	}
	scope := "privacy-review:" + id.String()
	p, e := domain.ParsePosition(q.Cursor, scope)
	if e != nil {
		respond(w, r, nil, e)
		return
	}
	items, e := h.repo.Reviews(r.Context(), id, p.At, p.ID, q.Limit+1)
	e = h.readAudit(r, id, e)
	if e != nil {
		respond(w, r, nil, e)
		return
	}
	more := len(items) > q.Limit
	next := ""
	if more {
		items = items[:q.Limit]
		v := items[len(items)-1]
		next = domain.EncodePosition(scope, v.Due, v.SubmissionID)
	}
	respond(w, r, domain.NewCursorPage(items, next, more, q.Limit), nil)
}
