package notice

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/httpx"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/shared/domain"
	"io"
	"net/http"
)

type Handler struct{ s *Service }

func NewHandler(s *Service) *Handler { return &Handler{s} }
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/api/v1/notifications/submissions/{id}", h.list(false))
	r.Get("/api/v1/notifications/submissions/{id}/{mail}/history", h.history(false))
	r.Group(func(r chi.Router) {
		r.Use(identity.RequirePermission(identity.PermissionNotifications))
		r.Get("/api/v1/admin/notifications/submissions/{id}", h.list(true))
		r.Get("/api/v1/admin/notifications/submissions/{id}/{mail}/history", h.history(true))
		r.With(identity.RequirePermission(identity.PermissionPhases)).Post("/api/v1/admin/notifications/submissions/{id}/{mail}/retry", h.retry)
	})
}
func ids(r *http.Request) (uuid.UUID, uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, uuid.Nil, ErrInvalid
	}
	m := uuid.Nil
	if v := chi.URLParam(r, "mail"); v != "" {
		m, err = uuid.Parse(v)
		if err != nil || m == uuid.Nil {
			return id, m, ErrInvalid
		}
	}
	return id, m, nil
}
func user(r *http.Request) *identity.Applicant {
	a, _ := identity.ApplicantFromContext(r.Context())
	return a
}
func respond(w http.ResponseWriter, r *http.Request, v interface{}, err error) {
	w.Header().Set("Cache-Control", "no-store")
	if err == nil {
		httpx.Respond(w, r, 200, v)
		return
	}
	status, code := 503, "notificacion_no_disponible"
	switch {
	case errors.Is(err, ErrForbidden):
		status, code = 403, "notificacion_prohibida"
	case errors.Is(err, ErrNotFound):
		status, code = 404, "notificacion_no_encontrada"
	case errors.Is(err, ErrConflict):
		status, code = 409, "reintento_no_admitido"
	case errors.Is(err, ErrInvalid), errors.Is(err, domain.ErrInvalidCursor):
		status, code = 422, "notificacion_invalida"
	}
	httpx.RespondError(w, r, status, code, "No se pudo completar la operación de notificación.")
}
func parse(r *http.Request) (domain.CursorRequest, error) {
	if httpx.ValidateQuery(r, "limit", "cursor") != nil {
		return domain.CursorRequest{}, ErrInvalid
	}
	v, err := httpx.ParseCursorRequest(r)
	if err != nil {
		return v, ErrInvalid
	}
	return v, nil
}
func (h *Handler) list(admin bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, _, err := ids(r)
		if err != nil {
			respond(w, r, nil, err)
			return
		}
		q, err := parse(r)
		if err != nil {
			respond(w, r, nil, err)
			return
		}
		v, err := h.s.List(r.Context(), user(r), id, q, admin)
		respond(w, r, v, err)
	}
}
func (h *Handler) history(admin bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, m, err := ids(r)
		if err != nil {
			respond(w, r, nil, err)
			return
		}
		q, err := parse(r)
		if err != nil {
			respond(w, r, nil, err)
			return
		}
		v, err := h.s.History(r.Context(), user(r), id, m, q, admin)
		respond(w, r, v, err)
	}
}
func (h *Handler) retry(w http.ResponseWriter, r *http.Request) {
	id, m, err := ids(r)
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	defer r.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(r.Body, 8193))
	if err != nil || len(raw) > 8192 {
		respond(w, r, nil, ErrInvalid)
		return
	}
	var input RetryInput
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(&input) != nil || d.Decode(new(json.RawMessage)) != io.EOF {
		respond(w, r, nil, ErrInvalid)
		return
	}
	err = h.s.Retry(r.Context(), user(r), id, m, input)
	respond(w, r, map[string]string{"estado": "pending"}, err)
}
