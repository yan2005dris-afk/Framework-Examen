package evaluation

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
	"slices"
	"strconv"
	"time"
)

type Handler struct{ service *Service }

func NewHandler(s *Service) *Handler { return &Handler{s} }
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/api/v1/evaluations/submissions/{id}", h.read(false))
	r.Get("/api/v1/evaluations/submissions/{id}/history", h.history(false))
	r.Get("/api/v1/evaluations/submissions/{id}/evidence/{evidence}/download", h.download(false))
	r.Get("/api/v1/reconsiderations/submissions/{id}", h.appeals(false))
	r.Post("/api/v1/reconsiderations/submissions/{id}", h.file)
	r.Get("/api/v1/reconsiderations/submissions/{id}/{appeal}", h.read(false))
	r.Group(func(r chi.Router) {
		r.Use(identity.RequirePermission(identity.PermissionGrades))
		r.Post("/api/v1/admin/grades/policies", h.createPolicy)
		r.Get("/api/v1/admin/grades/policies/{policy}", h.policy)
		r.Get("/api/v1/admin/grades/competitions/{period}/sectors/{sector}/policies/{kind}", h.policies)
		r.With(identity.RequirePermission(identity.PermissionPhases)).Post("/api/v1/admin/grades/policies/{policy}/approve", h.approve)
		r.Get("/api/v1/admin/grades/submissions/{id}", h.read(true))
		r.Get("/api/v1/admin/grades/submissions/{id}/history", h.history(true))
		r.Get("/api/v1/admin/grades/submissions/{id}/evidence/{evidence}/download", h.download(true))
		r.Post("/api/v1/admin/grades/submissions/{id}/psychometric", h.psychometric)
		r.With(identity.RequirePermission(identity.PermissionPhases)).Post("/api/v1/admin/grades/submissions/{id}/psychometric/publish", h.publishPsychometric)
		r.With(identity.RequirePermission(identity.PermissionPhases)).Post("/api/v1/admin/grades/submissions/{id}/consolidate", h.consolidate)
	})
	r.Group(func(r chi.Router) {
		r.Use(identity.RequirePermission(identity.PermissionReconsiderations))
		r.Get("/api/v1/admin/reconsiderations/submissions/{id}", h.appeals(true))
		r.Get("/api/v1/admin/reconsiderations/submissions/{id}/{appeal}", h.read(true))
		r.Post("/api/v1/admin/reconsiderations/submissions/{id}/{appeal}/attend", h.attend)
		r.With(identity.RequirePermission(identity.PermissionPhases)).Post("/api/v1/admin/reconsiderations/submissions/{id}/{appeal}/resolve", h.resolve)
	})
}
func actor(r *http.Request) *identity.Applicant {
	a, _ := identity.ApplicantFromContext(r.Context())
	return a
}
func identifier(r *http.Request, name string) (uuid.UUID, error) {
	v, err := uuid.Parse(chi.URLParam(r, name))
	if err != nil || v == uuid.Nil {
		return uuid.Nil, ErrInvalid
	}
	return v, nil
}
func decode(r *http.Request, v interface{}, maxBytes int64) error {
	defer r.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(r.Body, maxBytes+1))
	if err != nil || int64(len(raw)) > maxBytes {
		return ErrInvalid
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(v) != nil || d.Decode(new(json.RawMessage)) != io.EOF {
		return ErrInvalid
	}
	return nil
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
func respond(w http.ResponseWriter, r *http.Request, status int, v interface{}, err error) {
	w.Header().Set("Cache-Control", "no-store")
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpx.Respond(w, r, status, v)
}
func writeError(w http.ResponseWriter, r *http.Request, err error) {
	status, code, message := 503, "evaluacion_no_disponible", "No se pudo confirmar la operación ni su auditoría/notificación."
	switch {
	case errors.Is(err, ErrForbidden):
		status, code, message = 403, "evaluacion_prohibida", ErrForbidden.Error()
	case errors.Is(err, ErrNotFound):
		status, code, message = 404, "evaluacion_no_encontrada", ErrNotFound.Error()
	case errors.Is(err, ErrPolicy):
		status, code, message = 409, "reglas_evaluacion_pendientes", ErrPolicy.Error()
	case errors.Is(err, ErrConflict):
		status, code, message = 409, "version_evaluacion_conflicto", ErrConflict.Error()
	case errors.Is(err, ErrDeadline):
		status, code, message = 409, "plazo_reconsideracion_vencido", ErrDeadline.Error()
	case errors.Is(err, ErrPhase):
		status, code, message = 409, "fase_evaluacion_cerrada", ErrPhase.Error()
	case errors.Is(err, ErrState):
		status, code, message = 409, "estado_reconsideracion_invalido", ErrState.Error()
	case errors.Is(err, ErrPending):
		status, code, message = 409, "resultado_definitivo_pendiente", ErrPending.Error()
	case errors.Is(err, ErrQuota):
		status, code, message = 413, "cuota_respaldos_excedida", ErrQuota.Error()
	case errors.Is(err, ErrInvalid), errors.Is(err, submission.ErrInvalidSeal), errors.Is(err, domain.ErrInvalidCursor):
		status, code, message = 422, "evaluacion_invalida", ErrInvalid.Error()
	}
	httpx.RespondError(w, r, status, code, message)
}
func receipt(w http.ResponseWriter, r *http.Request, value Receipt, err error) {
	status := 201
	if !value.Applied {
		status = 200
	}
	respond(w, r, status, value.Data, err)
}
func (h *Handler) read(admin bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := identifier(r, "id")
		if err != nil {
			writeError(w, r, err)
			return
		}
		appeal := uuid.Nil
		if chi.URLParam(r, "appeal") != "" {
			appeal, err = identifier(r, "appeal")
			if err != nil {
				writeError(w, r, err)
				return
			}
		}
		c, err := h.service.Read(r.Context(), actor(r), id, appeal, admin)
		respond(w, r, 200, c, err)
	}
}
func (h *Handler) createPolicy(w http.ResponseWriter, r *http.Request) {
	var input struct {
		CompetitionID uuid.UUID  `json:"convocatoriaId"`
		Sector        string     `json:"sector"`
		Kind          string     `json:"tipo"`
		Definition    Definition `json:"definicion"`
	}
	if decode(r, &input, 128<<10) != nil {
		writeError(w, r, ErrInvalid)
		return
	}
	p, err := h.service.CreatePolicy(r.Context(), actor(r), input.CompetitionID, input.Sector, input.Kind, input.Definition)
	respond(w, r, 201, p, err)
}
func (h *Handler) approve(w http.ResponseWriter, r *http.Request) {
	id, err := identifier(r, "policy")
	var input struct {
		Reference string `json:"referenciaAprobacion"`
	}
	if err != nil || decode(r, &input, 8192) != nil {
		writeError(w, r, ErrInvalid)
		return
	}
	p, err := h.service.ApprovePolicy(r.Context(), actor(r), id, input.Reference)
	respond(w, r, 200, p, err)
}
func (h *Handler) policyAudit(r *http.Request, id uuid.UUID, err error) error {
	a := actor(r)
	if !identity.HasPermission(a, identity.PermissionGrades) {
		return ErrForbidden
	}
	return audit.Record(r.Context(), h.service.auditor, audit.Event{ActorID: a.ID, Role: a.Role, Action: "evaluacion.reglas_consultadas", ResourceID: id, ResourceType: "convocatoria", CreatedAt: time.Now().UTC()}, err)
}
func (h *Handler) policy(w http.ResponseWriter, r *http.Request) {
	id, err := identifier(r, "policy")
	if err != nil {
		writeError(w, r, err)
		return
	}
	p, err := h.service.repo.GetPolicy(r.Context(), id)
	resource := id
	if p != nil {
		resource = p.CompetitionID
	}
	err = h.policyAudit(r, resource, err)
	respond(w, r, 200, p, err)
}
func (h *Handler) policies(w http.ResponseWriter, r *http.Request) {
	id, err := identifier(r, "period")
	sector, kind := chi.URLParam(r, "sector"), chi.URLParam(r, "kind")
	if err != nil || !slices.Contains([]string{"afiliados", "jubilados", "empleadores"}, sector) || !slices.Contains([]string{"oposicion", "reconsideracion"}, kind) || httpx.ValidateQuery(r, "limit", "cursor") != nil {
		writeError(w, r, ErrInvalid)
		return
	}
	request, err := httpx.ParseCursorRequest(r)
	if err != nil {
		writeError(w, r, ErrInvalid)
		return
	}
	before := int64(0)
	if request.Cursor != nil {
		scope, ok := request.Cursor.Value(0)
		value, ok2 := request.Cursor.Value(1)
		_, extra := request.Cursor.Value(2)
		if !ok || !ok2 || extra || scope != "evaluation-policies:"+id.String()+":"+sector+":"+kind {
			writeError(w, r, ErrInvalid)
			return
		}
		before, err = strconv.ParseInt(value, 10, 64)
		if err != nil || before < 1 {
			writeError(w, r, ErrInvalid)
			return
		}
	}
	items, err := h.service.repo.ListPolicies(r.Context(), id, sector, kind, before, request.Limit+1)
	err = h.policyAudit(r, id, err)
	if err != nil {
		writeError(w, r, err)
		return
	}
	more := len(items) > request.Limit
	next := ""
	if more {
		items = items[:request.Limit]
		next = domain.NewCursor("evaluation-policies:"+id.String()+":"+sector+":"+kind, strconv.FormatInt(items[len(items)-1].Version, 10)).Encode()
	}
	respond(w, r, 200, domain.NewCursorPage(items, next, more, request.Limit), nil)
}
func (h *Handler) psychometric(w http.ResponseWriter, r *http.Request) {
	id, err := identifier(r, "id")
	k, keyErr := key(r)
	var input PsychometricInput
	if err != nil || keyErr != nil || decode(r, &input, 4<<20) != nil {
		writeError(w, r, ErrInvalid)
		return
	}
	v, err := h.service.RecordPsychometric(r.Context(), actor(r), id, k, input)
	receipt(w, r, v, err)
}
func (h *Handler) publishPsychometric(w http.ResponseWriter, r *http.Request) {
	id, err := identifier(r, "id")
	k, keyErr := key(r)
	var input PublishInput
	if err != nil || keyErr != nil || decode(r, &input, 128<<10) != nil {
		writeError(w, r, ErrInvalid)
		return
	}
	v, err := h.service.PublishPsychometric(r.Context(), actor(r), id, k, input)
	receipt(w, r, v, err)
}
func (h *Handler) consolidate(w http.ResponseWriter, r *http.Request) {
	id, err := identifier(r, "id")
	k, keyErr := key(r)
	var input PublishInput
	if err != nil || keyErr != nil || decode(r, &input, 128<<10) != nil {
		writeError(w, r, ErrInvalid)
		return
	}
	v, err := h.service.Consolidate(r.Context(), actor(r), id, k, input)
	receipt(w, r, v, err)
}
func (h *Handler) file(w http.ResponseWriter, r *http.Request) {
	id, err := identifier(r, "id")
	k, keyErr := key(r)
	var input FileAppealInput
	if err != nil || keyErr != nil || decode(r, &input, 4<<20) != nil {
		writeError(w, r, ErrInvalid)
		return
	}
	v, err := h.service.FileAppeal(r.Context(), actor(r), id, k, input)
	receipt(w, r, v, err)
}
func (h *Handler) attend(w http.ResponseWriter, r *http.Request) {
	id, err := identifier(r, "id")
	appeal, appealErr := identifier(r, "appeal")
	k, keyErr := key(r)
	var input AttendInput
	if err != nil || appealErr != nil || keyErr != nil || decode(r, &input, 128<<10) != nil {
		writeError(w, r, ErrInvalid)
		return
	}
	v, err := h.service.Attend(r.Context(), actor(r), id, appeal, k, input)
	receipt(w, r, v, err)
}
func (h *Handler) resolve(w http.ResponseWriter, r *http.Request) {
	id, err := identifier(r, "id")
	appeal, appealErr := identifier(r, "appeal")
	k, keyErr := key(r)
	var input ResolveInput
	if err != nil || appealErr != nil || keyErr != nil || decode(r, &input, 128<<10) != nil {
		writeError(w, r, ErrInvalid)
		return
	}
	v, err := h.service.Resolve(r.Context(), actor(r), id, appeal, k, input)
	receipt(w, r, v, err)
}
func pageRequest(r *http.Request, id uuid.UUID, kind string) (domain.CursorRequest, domain.Position, error) {
	if httpx.ValidateQuery(r, "limit", "cursor") != nil {
		return domain.CursorRequest{}, domain.Position{}, ErrInvalid
	}
	request, err := httpx.ParseCursorRequest(r)
	if err != nil {
		return request, domain.Position{}, err
	}
	position, err := domain.ParsePosition(request.Cursor, "evaluation:"+kind+":"+id.String())
	return request, position, err
}
func (h *Handler) appeals(admin bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := identifier(r, "id")
		if err != nil {
			writeError(w, r, err)
			return
		}
		if _, err = h.service.Read(r.Context(), actor(r), id, uuid.Nil, admin); err != nil {
			writeError(w, r, err)
			return
		}
		request, position, err := pageRequest(r, id, "appeals")
		if err != nil {
			writeError(w, r, err)
			return
		}
		items, err := h.service.repo.ListAppeals(r.Context(), id, position, request.Limit+1)
		if err != nil {
			writeError(w, r, err)
			return
		}
		more := len(items) > request.Limit
		next := ""
		if more {
			items = items[:request.Limit]
			last := items[len(items)-1]
			next = domain.EncodePosition("evaluation:appeals:"+id.String(), last.FiledAt, last.ID)
		}
		respond(w, r, 200, domain.NewCursorPage(items, next, more, request.Limit), nil)
	}
}
func (h *Handler) history(admin bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := identifier(r, "id")
		if err != nil {
			writeError(w, r, err)
			return
		}
		if _, err = h.service.Read(r.Context(), actor(r), id, uuid.Nil, admin); err != nil {
			writeError(w, r, err)
			return
		}
		request, position, err := pageRequest(r, id, "history")
		if err != nil {
			writeError(w, r, err)
			return
		}
		items, err := h.service.repo.ListHistory(r.Context(), id, position, request.Limit+1)
		if err != nil {
			writeError(w, r, err)
			return
		}
		if !admin {
			for i := range items {
				if items[i].Action == "psicotecnica.registrada" {
					items[i].Data = nil
				}
			}
		}
		more := len(items) > request.Limit
		next := ""
		if more {
			items = items[:request.Limit]
			last := items[len(items)-1]
			next = domain.EncodePosition("evaluation:history:"+id.String(), last.At, last.ID)
		}
		respond(w, r, 200, domain.NewCursorPage(items, next, more, request.Limit), nil)
	}
}
func (h *Handler) download(admin bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := identifier(r, "id")
		evidence, evidenceErr := identifier(r, "evidence")
		if err != nil || evidenceErr != nil {
			writeError(w, r, ErrInvalid)
			return
		}
		url, err := h.service.Download(r.Context(), actor(r), id, evidence, admin)
		respond(w, r, 200, struct {
			URL       string    `json:"url"`
			ExpiresAt time.Time `json:"expiraEn"`
		}{url, time.Now().UTC().Add(5 * time.Minute)}, err)
	}
}
