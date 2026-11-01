package results

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
	"strconv"
)

type Handler struct{ s *Service }

func NewHandler(s *Service) *Handler { return &Handler{s} }
func (h *Handler) RegisterPublic(r chi.Router) {
	r.Get("/api/v1/public/results/competitions/{period}/sectors/{sector}", h.public(true))
	r.Get("/api/v1/public/results/publications/{id}", h.public(false))
}
func (h *Handler) RegisterAdministration(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(identity.RequirePermission(identity.PermissionResults))
		r.Post("/api/v1/admin/results/competitions/{period}/sectors/{sector}/lists", h.generate)
		r.Get("/api/v1/admin/results/competitions/{period}/sectors/{sector}/lists", h.lists(false))
		r.Get("/api/v1/admin/results/competitions/{period}/sectors/{sector}/publications", h.lists(true))
		r.Get("/api/v1/admin/results/lists/{id}", h.get)
		r.Get("/api/v1/admin/results/lists/{id}/entries", h.entries)
		r.Get("/api/v1/admin/results/lists/{id}/export.csv", h.export)
		r.Get("/api/v1/admin/results/publications/{id}", h.publication)
		r.With(identity.RequirePermission(identity.PermissionPhases)).Post("/api/v1/admin/results/lists/{id}/publish", h.publish)
	})
}
func user(r *http.Request) *identity.Applicant {
	a, _ := identity.ApplicantFromContext(r.Context())
	return a
}
func identifier(r *http.Request, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, name))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, ErrInvalid
	}
	return id, nil
}
func key(r *http.Request) (uuid.UUID, error) {
	if len(r.Header.Values("Idempotency-Key")) != 1 {
		return uuid.Nil, ErrInvalid
	}
	id, err := uuid.Parse(r.Header.Get("Idempotency-Key"))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, ErrInvalid
	}
	return id, nil
}
func decode(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(r.Body, 8193))
	if err != nil || len(raw) > 8192 {
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
	status, code := 503, "lista_no_disponible"
	switch {
	case errors.Is(err, ErrForbidden):
		status, code = 403, "lista_prohibida"
	case errors.Is(err, ErrNotFound):
		status, code = 404, "lista_no_encontrada"
	case errors.Is(err, ErrPending):
		status, code = 409, "fuentes_lista_pendientes"
	case errors.Is(err, ErrConflict):
		status, code = 409, "version_lista_conflicto"
	case errors.Is(err, ErrPhase):
		status, code = 409, "fase_lista_cerrada"
	case errors.Is(err, ErrPrivacy):
		status, code = 409, "publicacion_nombre_no_autorizada"
	case errors.Is(err, ErrTooLarge):
		status, code = 413, "lista_demasiado_grande"
	case errors.Is(err, ErrInvalid), errors.Is(err, domain.ErrInvalidCursor):
		status, code = 422, "lista_invalida"
	}
	httpx.RespondError(w, r, status, code, "No fue posible completar la operación de lista de resultados.")
}
func page(r *http.Request) (domain.CursorRequest, error) {
	if httpx.ValidateQuery(r, "limit", "cursor") != nil {
		return domain.CursorRequest{}, ErrInvalid
	}
	q, err := httpx.ParseCursorRequest(r)
	if err != nil {
		return q, ErrInvalid
	}
	return q, nil
}
func ordinal(q domain.CursorRequest, scope string, total int) (int, error) {
	if q.Cursor == nil {
		return 0, nil
	}
	s, ok := q.Cursor.Value(0)
	v, ok2 := q.Cursor.Value(1)
	_, extra := q.Cursor.Value(2)
	n, err := strconv.Atoi(v)
	if !ok || !ok2 || extra || s != scope || err != nil || n < 1 || n >= total {
		return 0, ErrInvalid
	}
	return n, nil
}
func arrayPage[T any](items []T, q domain.CursorRequest, scope string) (domain.CursorPage[T], error) {
	start, err := ordinal(q, scope, len(items))
	if err != nil {
		return domain.CursorPage[T]{}, err
	}
	end := min(len(items), start+q.Limit)
	more := end < len(items)
	next := ""
	if more {
		next = domain.NewCursor(scope, strconv.Itoa(end)).Encode()
	}
	return domain.NewCursorPage(items[start:end], next, more, q.Limit), nil
}
func (h *Handler) generate(w http.ResponseWriter, r *http.Request) {
	id, err := identifier(r, "period")
	k, e := key(r)
	var input GenerateInput
	if err != nil || e != nil || decode(r, &input) != nil {
		respond(w, r, nil, ErrInvalid)
		return
	}
	v, err := h.s.Generate(r.Context(), user(r), id, chi.URLParam(r, "sector"), k, input)
	respond(w, r, v, err)
}
func (h *Handler) publish(w http.ResponseWriter, r *http.Request) {
	id, err := identifier(r, "id")
	k, e := key(r)
	var input PublishInput
	if err != nil || e != nil || decode(r, &input) != nil {
		respond(w, r, nil, ErrInvalid)
		return
	}
	v, err := h.s.Publish(r.Context(), user(r), id, k, input)
	respond(w, r, v, err)
}
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := identifier(r, "id")
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	v, err := h.s.Get(r.Context(), user(r), id)
	respond(w, r, v, err)
}
func (h *Handler) entries(w http.ResponseWriter, r *http.Request) {
	id, err := identifier(r, "id")
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	q, err := page(r)
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	v, err := h.s.Get(r.Context(), user(r), id)
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	p, err := arrayPage(v.Entries, q, "result-entries:"+id.String())
	respond(w, r, p, err)
}
func (h *Handler) publication(w http.ResponseWriter, r *http.Request) {
	id, err := identifier(r, "id")
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	v, err := h.s.Publication(r.Context(), user(r), id)
	respond(w, r, v, err)
}
func (h *Handler) lists(pubs bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := identifier(r, "period")
		if err != nil {
			respond(w, r, nil, err)
			return
		}
		q, err := page(r)
		if err != nil {
			respond(w, r, nil, err)
			return
		}
		if pubs {
			v, err := h.s.Publications(r.Context(), user(r), id, chi.URLParam(r, "sector"), q)
			respond(w, r, v, err)
		} else {
			v, err := h.s.Lists(r.Context(), user(r), id, chi.URLParam(r, "sector"), q)
			respond(w, r, v, err)
		}
	}
}
func (h *Handler) public(current bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		field := "id"
		if current {
			field = "period"
		}
		id, err := identifier(r, field)
		if err != nil {
			respond(w, r, nil, err)
			return
		}
		q, err := page(r)
		if err != nil {
			respond(w, r, nil, err)
			return
		}
		v, err := h.s.Public(r.Context(), id, chi.URLParam(r, "sector"), current)
		if err != nil {
			respond(w, r, nil, err)
			return
		}
		p, err := arrayPage(v.Names, q, "public-result-names:"+v.ID.String())
		if err != nil {
			respond(w, r, nil, err)
			return
		}
		respond(w, r, map[string]interface{}{"id": v.ID, "convocatoriaId": v.CompetitionID, "sector": v.Sector, "version": v.Version, "publicadaEn": v.CreatedAt, "orden": "alfabetico_nombres_sin_puntajes", "nombres": p}, nil)
	}
}
func (h *Handler) export(w http.ResponseWriter, r *http.Request) {
	id, err := identifier(r, "id")
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	if httpx.ValidateQuery(r) != nil {
		respond(w, r, nil, ErrInvalid)
		return
	}
	data, err := h.s.Export(r.Context(), user(r), id)
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="lista-`+id.String()+`.csv"`)
	w.WriteHeader(200)
	_, _ = w.Write(data)
}
