package exam

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/httpx"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/privacy"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/shared/domain"
	"io"
	"net/http"
	"strconv"
	"time"
)

type Handler struct {
	repo    Repository
	auditor audit.Recorder
}

func NewHandler(r Repository, a audit.Recorder) *Handler { return &Handler{r, a} }
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/api/v1/exams/submissions/{id}", h.read(false))
	r.Post("/api/v1/exams/submissions/{id}/start", h.start)
	r.Get("/api/v1/exams/submissions/{id}/questions/{index}", h.question)
	r.Put("/api/v1/exams/submissions/{id}/answers", h.answer)
	r.Post("/api/v1/exams/submissions/{id}/finish", h.finish)
	r.Get("/api/v1/exams/submissions/{id}/report.pdf", h.report(false))
	r.Group(func(r chi.Router) {
		r.Use(identity.RequirePermission(identity.PermissionQuestions))
		r.Post("/api/v1/admin/question-bank/competitions/{period}/sectors/{sector}/versions", h.createBank)
		r.Get("/api/v1/admin/question-bank/competitions/{period}/sectors/{sector}/versions", h.banks)
		r.Get("/api/v1/admin/question-bank/versions/{id}", h.bank)
		r.Post("/api/v1/admin/question-bank/versions/{id}/configurations", h.configure)
		r.Post("/api/v1/admin/question-bank/submissions/{id}/enable", h.enroll)
	})
	r.With(identity.RequirePermission(identity.PermissionGrades)).Get("/api/v1/admin/grades/submissions/{id}/exam", h.read(true))
	r.With(identity.RequirePermission(identity.PermissionGrades)).Get("/api/v1/admin/grades/submissions/{id}/exam/report.pdf", h.report(true))
}
func actor(r *http.Request) *identity.Applicant {
	a, _ := identity.ApplicantFromContext(r.Context())
	return a
}
func id(r *http.Request, key string) (uuid.UUID, error) {
	v, e := uuid.Parse(chi.URLParam(r, key))
	if e != nil || v == uuid.Nil {
		return uuid.Nil, ErrInvalid
	}
	return v, nil
}
func decode(r *http.Request, v interface{}, limit int64) error {
	defer r.Body.Close()
	raw, e := io.ReadAll(io.LimitReader(r.Body, limit+1))
	if e != nil || int64(len(raw)) > limit {
		return ErrInvalid
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(v) != nil || d.Decode(new(json.RawMessage)) != io.EOF {
		return ErrInvalid
	}
	return nil
}
func respond(w http.ResponseWriter, r *http.Request, v interface{}, e error) {
	w.Header().Set("Cache-Control", "no-store")
	if e == nil {
		httpx.Respond(w, r, 200, v)
		return
	}
	status, code := 503, "examen_no_disponible"
	switch {
	case errors.Is(e, ErrForbidden):
		status, code = 403, "examen_prohibido"
	case errors.Is(e, ErrNotFound):
		status, code = 404, "examen_no_encontrado"
	case errors.Is(e, ErrInvalid), errors.Is(e, domain.ErrInvalidCursor):
		status, code = 422, "examen_invalido"
	case errors.Is(e, ErrConflict):
		status, code = 409, "version_examen_conflicto"
	case errors.Is(e, ErrExpired):
		status, code = 409, "examen_cerrado"
	case errors.Is(e, ErrPhase):
		status, code = 409, "examen_no_habilitado"
	case errors.Is(e, ErrExhausted):
		status, code = 409, "asignaciones_examen_agotadas"
	case errors.Is(e, privacy.ErrDenied):
		status, code = 409, "privacidad_pendiente"
	}
	httpx.RespondError(w, r, status, code, "No se pudo completar la operación de evaluación.")
}
func (h *Handler) auditRead(ctx context.Context, a *identity.Applicant, resource uuid.UUID, action string, e error) error {
	if a == nil {
		return ErrForbidden
	}
	return audit.Record(ctx, h.auditor, audit.Event{ActorID: a.ID, Role: a.Role, ResourceID: resource, ResourceType: "evaluacion", Action: action, CreatedAt: time.Now().UTC()}, e)
}
func (h *Handler) createBank(w http.ResponseWriter, r *http.Request) {
	p, e := id(r, "period")
	var input BankInput
	if e != nil || decode(r, &input, 2<<20) != nil {
		respond(w, r, nil, ErrInvalid)
		return
	}
	b, e := h.repo.CreateBank(r.Context(), actor(r), p, chi.URLParam(r, "sector"), input)
	respond(w, r, b, e)
}
func (h *Handler) bank(w http.ResponseWriter, r *http.Request) {
	v, e := id(r, "id")
	if e != nil {
		respond(w, r, nil, e)
		return
	}
	b, e := h.repo.Bank(r.Context(), v)
	e = h.auditRead(r.Context(), actor(r), v, "banco.consultado", e)
	if e != nil {
		b = nil
	}
	respond(w, r, b, e)
}
func (h *Handler) banks(w http.ResponseWriter, r *http.Request) {
	p, e := id(r, "period")
	if e != nil || httpx.ValidateQuery(r, "limit", "cursor") != nil {
		respond(w, r, nil, ErrInvalid)
		return
	}
	q, e := httpx.ParseCursorRequest(r)
	if e != nil {
		respond(w, r, nil, ErrInvalid)
		return
	}
	scope := "exam-banks:" + p.String() + ":" + chi.URLParam(r, "sector")
	before := int64(0)
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
	items, e := h.repo.Banks(r.Context(), p, chi.URLParam(r, "sector"), before, q.Limit+1)
	e = h.auditRead(r.Context(), actor(r), p, "banco.consultado", e)
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
func (h *Handler) configure(w http.ResponseWriter, r *http.Request) {
	bank, e := id(r, "id")
	var input struct {
		ConfigRules
		ExplicitThreshold *int64 `json:"umbral"`
	}
	if e != nil || decode(r, &input, 8192) != nil || input.ExplicitThreshold == nil {
		respond(w, r, nil, ErrInvalid)
		return
	}
	input.ConfigRules.Threshold = *input.ExplicitThreshold
	v, e := h.repo.Configure(r.Context(), actor(r), bank, input.ConfigRules)
	respond(w, r, v, e)
}

func (h *Handler) enroll(w http.ResponseWriter, r *http.Request) {
	sub, e := id(r, "id")
	var input EnrollmentInput
	if e != nil || decode(r, &input, 8192) != nil {
		respond(w, r, nil, ErrInvalid)
		return
	}
	e = h.repo.Enroll(r.Context(), actor(r), sub, input)
	respond(w, r, map[string]string{"fase": "evaluacion_tecnica"}, e)
}
func (h *Handler) start(w http.ResponseWriter, r *http.Request) {
	sub, e := id(r, "id")
	var input struct{}
	if e != nil || decode(r, &input, 1024) != nil {
		respond(w, r, nil, ErrInvalid)
		return
	}
	a, t, e := h.repo.Start(r.Context(), actor(r), sub)
	if e != nil {
		respond(w, r, nil, e)
		return
	}
	respond(w, r, PublicView(a, t), nil)
}
func (h *Handler) read(admin bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sub, e := id(r, "id")
		if e != nil {
			respond(w, r, nil, e)
			return
		}
		a, t, e := h.repo.Read(r.Context(), actor(r), sub, admin)
		if admin {
			e = h.auditRead(r.Context(), actor(r), sub, "examen.consultado", e)
		}
		if e != nil {
			respond(w, r, nil, e)
			return
		}
		respond(w, r, PublicView(a, t), nil)
	}
}
func (h *Handler) question(w http.ResponseWriter, r *http.Request) {
	sub, e := id(r, "id")
	index, e2 := strconv.Atoi(chi.URLParam(r, "index"))
	if e != nil || e2 != nil {
		respond(w, r, nil, ErrInvalid)
		return
	}
	a, _, e := h.repo.Read(r.Context(), actor(r), sub, false)
	if e != nil {
		respond(w, r, nil, e)
		return
	}
	v, e := QuestionAt(a, index)
	respond(w, r, v, e)
}
func (h *Handler) answer(w http.ResponseWriter, r *http.Request) {
	sub, e := id(r, "id")
	var input AnswerInput
	if e != nil || decode(r, &input, 8192) != nil {
		respond(w, r, nil, ErrInvalid)
		return
	}
	a, t, e := h.repo.Answer(r.Context(), actor(r), sub, input)
	if e != nil {
		respond(w, r, nil, e)
		return
	}
	respond(w, r, PublicView(a, t), nil)
}
func (h *Handler) finish(w http.ResponseWriter, r *http.Request) {
	sub, e := id(r, "id")
	var input struct{}
	if e != nil || decode(r, &input, 1024) != nil {
		respond(w, r, nil, ErrInvalid)
		return
	}
	a, t, e := h.repo.Finish(r.Context(), actor(r), sub, false)
	if e != nil {
		respond(w, r, nil, e)
		return
	}
	respond(w, r, PublicView(a, t), nil)
}
func (h *Handler) report(admin bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sub, e := id(r, "id")
		if e != nil {
			respond(w, r, nil, e)
			return
		}
		v, e := h.repo.Report(r.Context(), actor(r), sub, admin)
		if admin {
			e = h.auditRead(r.Context(), actor(r), sub, "examen.reporte_descargado", e)
		}
		if e != nil {
			respond(w, r, nil, e)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", `attachment; filename="examen-`+sub.String()+`.pdf"`)
		w.Header().Set("X-Content-SHA256", v.Hash)
		_, _ = w.Write(v.PDF)
	}
}
