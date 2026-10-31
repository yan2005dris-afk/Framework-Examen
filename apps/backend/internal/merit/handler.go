package merit

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
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"io"
	"net/http"
	"strconv"
)

type Handler struct{ service *Service }

func NewHandler(s *Service) *Handler { return &Handler{s} }
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/api/v1/merits/submissions/{id}", h.owner)
	r.Group(func(r chi.Router) {
		r.Use(identity.RequirePermission(identity.PermissionGrades))
		r.Post("/api/v1/admin/merits/rules", h.createRules)
		r.Get("/api/v1/admin/merits/rules/{id}", h.rules)
		r.With(identity.RequirePermission(identity.PermissionPhases)).Post("/api/v1/admin/merits/rules/{id}/approve", h.approve)
		r.Get("/api/v1/admin/merits/competitions/{id}/sectors/{sector}/rules", h.listRules)
		r.Get("/api/v1/admin/merits/submissions/{id}", h.read)
		r.Post("/api/v1/admin/merits/submissions/{id}/review", h.review)
		r.Post("/api/v1/admin/merits/submissions/{id}/calculate", h.calculate)
		r.With(identity.RequirePermission(identity.PermissionPhases)).Post("/api/v1/admin/merits/submissions/{id}/publish", h.publish)
	})
}
func decodeRequest(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(r.Body, (128<<10)+1))
	if err != nil || len(raw) > 128<<10 {
		return ErrInvalid
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(v) != nil || d.Decode(new(json.RawMessage)) != io.EOF {
		return ErrInvalid
	}
	return nil
}
func requestID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, ErrInvalid
	}
	return id, nil
}
func actor(r *http.Request) *identity.Applicant {
	a, _ := identity.ApplicantFromContext(r.Context())
	return a
}
func respond(w http.ResponseWriter, r *http.Request, status int, v interface{}, err error) {
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.Respond(w, r, status, v)
}
func writeError(w http.ResponseWriter, r *http.Request, err error) {
	status, code, message := 503, "meritos_no_disponibles", "No fue posible completar la operación de méritos."
	switch {
	case errors.Is(err, ErrForbidden):
		status, code, message = 403, "meritos_prohibidos", "No tiene permiso para esta operación."
	case errors.Is(err, ErrNotFound):
		status, code, message = 404, "meritos_no_encontrados", "No se encontró el recurso."
	case errors.Is(err, ErrRulesPending):
		status, code, message = 409, "reglas_meritos_pendientes", "Se requiere una matriz de méritos aprobada."
	case errors.Is(err, ErrConflict):
		status, code, message = 409, "version_meritos_conflicto", "El expediente cambió; vuelva a consultarlo."
	case errors.Is(err, ErrPhase):
		status, code, message = 409, "fase_meritos_cerrada", "La fase de méritos no está habilitada."
	case errors.Is(err, ErrEvidence), errors.Is(err, submission.ErrInvalidSeal):
		status, code, message = 422, "evidencia_meritos_invalida", "La evidencia o el sello no permite esta operación."
	case errors.Is(err, ErrInvalid), errors.Is(err, domain.ErrInvalidCursor):
		status, code, message = 422, "meritos_invalidos", "Revise los datos y las reglas de méritos."
	}
	httpx.RespondError(w, r, status, code, message)
}
func (h *Handler) createRules(w http.ResponseWriter, r *http.Request) {
	var input struct {
		CompetitionID uuid.UUID      `json:"convocatoriaId"`
		Sector        string         `json:"sector"`
		Definition    RuleDefinition `json:"definicion"`
	}
	if err := decodeRequest(r, &input); err != nil {
		writeError(w, r, err)
		return
	}
	v, err := h.service.CreateRules(r.Context(), actor(r), input.CompetitionID, input.Sector, input.Definition)
	respond(w, r, 201, v, err)
}
func (h *Handler) approve(w http.ResponseWriter, r *http.Request) {
	id, err := requestID(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var input struct {
		Reference string `json:"referenciaAprobacion"`
	}
	if decodeRequest(r, &input) != nil {
		writeError(w, r, ErrInvalid)
		return
	}
	v, err := h.service.ApproveRules(r.Context(), actor(r), id, input.Reference)
	respond(w, r, 200, v, err)
}
func (h *Handler) auditRules(r *http.Request, id uuid.UUID, err error) error {
	a := actor(r)
	if !identity.HasPermission(a, identity.PermissionGrades) {
		return ErrForbidden
	}
	return audit.Record(r.Context(), h.service.auditor, audit.Event{ActorID: a.ID, Role: a.Role, Action: "reglas_meritos.consultadas", ResourceID: id, ResourceType: "convocatoria", CreatedAt: h.service.now().UTC()}, err)
}
func (h *Handler) rules(w http.ResponseWriter, r *http.Request) {
	id, err := requestID(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	v, err := h.service.repo.GetRules(r.Context(), id)
	resource := id
	if v != nil {
		resource = v.CompetitionID
	}
	err = h.auditRules(r, resource, err)
	respond(w, r, 200, v, err)
}
func (h *Handler) listRules(w http.ResponseWriter, r *http.Request) {
	id, err := requestID(r)
	sector := chi.URLParam(r, "sector")
	if err != nil || !validSector(sector) || httpx.ValidateQuery(r, "limit", "cursor") != nil {
		writeError(w, r, ErrInvalid)
		return
	}
	q, err := httpx.ParseCursorRequest(r)
	if err != nil {
		writeError(w, r, ErrInvalid)
		return
	}
	before := int64(0)
	if q.Cursor != nil {
		marker, ok := q.Cursor.Value(0)
		period, ok2 := q.Cursor.Value(1)
		scope, ok3 := q.Cursor.Value(2)
		value, ok4 := q.Cursor.Value(3)
		if !ok || !ok2 || !ok3 || !ok4 || marker != "merit-rules-v1" || period != id.String() || scope != sector {
			writeError(w, r, ErrInvalid)
			return
		}
		before, err = strconv.ParseInt(value, 10, 64)
		if err != nil || before <= 0 {
			writeError(w, r, ErrInvalid)
			return
		}
	}
	items, err := h.service.repo.ListRules(r.Context(), id, sector, before, q.Limit+1)
	err = h.auditRules(r, id, err)
	if err != nil {
		writeError(w, r, err)
		return
	}
	hasMore := len(items) > q.Limit
	if hasMore {
		items = items[:q.Limit]
	}
	next := ""
	if hasMore {
		next = domain.NewCursor("merit-rules-v1", id.String(), sector, strconv.FormatInt(items[len(items)-1].Version, 10)).Encode()
	}
	respond(w, r, 200, domain.NewCursorPage(items, next, hasMore, q.Limit), nil)
}
func (h *Handler) read(w http.ResponseWriter, r *http.Request) {
	id, err := requestID(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	v, err := h.service.Read(r.Context(), actor(r), id, true)
	respond(w, r, 200, v, err)
}
func (h *Handler) owner(w http.ResponseWriter, r *http.Request) {
	id, err := requestID(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	v, err := h.service.Read(r.Context(), actor(r), id, false)
	if err != nil {
		writeError(w, r, err)
		return
	}
	respond(w, r, 200, map[string]interface{}{"publicado": v.Publication != nil, "habilitadoFaseTecnica": v.Publication != nil && v.Publication.Decision == "avanza", "resultado": v.Publication}, nil)
}
func (h *Handler) review(w http.ResponseWriter, r *http.Request) {
	id, err := requestID(r)
	var input ReviewInput
	if err != nil || decodeRequest(r, &input) != nil {
		writeError(w, r, ErrInvalid)
		return
	}
	v, err := h.service.Review(r.Context(), actor(r), id, input)
	respond(w, r, 201, v, err)
}
func (h *Handler) calculate(w http.ResponseWriter, r *http.Request) {
	id, err := requestID(r)
	var input CalculateInput
	if err != nil || decodeRequest(r, &input) != nil {
		writeError(w, r, ErrInvalid)
		return
	}
	v, err := h.service.Calculate(r.Context(), actor(r), id, input)
	respond(w, r, 201, v, err)
}
func (h *Handler) publish(w http.ResponseWriter, r *http.Request) {
	id, err := requestID(r)
	var input PublishInput
	if err != nil || decodeRequest(r, &input) != nil {
		writeError(w, r, ErrInvalid)
		return
	}
	v, err := h.service.Publish(r.Context(), actor(r), id, input)
	respond(w, r, 201, v, err)
}
