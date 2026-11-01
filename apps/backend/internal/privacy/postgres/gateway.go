package postgres

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/httpx"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/mail"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/privacy"
	"net/http"
	"strings"
	"time"
)

/** Guard cubre las salidas de datos ya existentes; complementa, nunca sustituye, titularidad y permisos de cada handler. */
func (r *Repository) Guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != "GET" {
			next.ServeHTTP(w, req)
			return
		}
		path := strings.Trim(req.URL.Path, "/")
		parts := strings.Split(path, "/")
		ctx := req.Context()
		var err error
		sub := uuid.Nil
		channel := "datos"
		handled := false
		if strings.HasSuffix(path, "/download") {
			channel = "documentos"
		}
		if strings.HasSuffix(path, "/print.pdf") || strings.HasSuffix(path, "/report.pdf") {
			channel = "reportes"
		}
		if strings.HasPrefix(path, "api/v1/admin/results/lists/") {
			if len(parts) >= 6 {
				listID, e := uuid.Parse(parts[5])
				if e == nil {
					v, e := r.q.GetResultList(ctx, listID)
					if e == nil {
						period := v.CompetitionID
						channel = "exportacion"
						err = r.Period(ctx, period, channel)
						handled = true
					} else {
						err = e
						handled = true
					}
				}
			}
		}
		if !handled && strings.HasPrefix(path, "api/v1/admin/submissions/export.") {
			period, e := uuid.Parse(req.URL.Query().Get("convocatoriaId"))
			if e != nil {
				err = privacy.ErrDenied
			} else {
				err = r.Period(ctx, period, "exportacion")
			}
			handled = true
		}
		if !handled && (path == "api/v1/admin/submissions" || path == "api/v1/admin/submissions/stats") {
			period, e := uuid.Parse(req.URL.Query().Get("convocatoriaId"))
			if e != nil {
				err = privacy.ErrDenied
			} else {
				err = r.Period(ctx, period, "datos")
			}
			handled = true
		}
		if !handled && path == "api/v1/submissions" {
			a, _ := identity.ApplicantFromContext(ctx)
			if a == nil {
				err = privacy.ErrDenied
			} else {
				ids, e := r.q.PrivacyApplicantSubmissions(ctx, a.ID)
				if e != nil {
					err = e
				} else if len(ids) > 10000 {
					err = privacy.ErrDenied
				} else {
					for _, id := range ids {
						if e = r.Allow(ctx, id, "datos"); e != nil {
							err = e
							break
						}
					}
				}
			}
			handled = true
		}
		if !handled {
			for i, p := range parts {
				if p == "submissions" && i+1 < len(parts) {
					if v, e := uuid.Parse(parts[i+1]); e == nil {
						sub = v
						handled = true
						break
					}
				}
			}
			if !handled && strings.Contains(path, "/documents/") {
				for i, p := range parts {
					if p == "documents" && i+1 < len(parts) {
						if v, e := uuid.Parse(parts[i+1]); e == nil {
							sub, err = r.q.DocumentSubmissionForPrivacy(ctx, v)
							handled = true
						}
						break
					}
				}
			}
			if handled && err == nil {
				err = r.Allow(ctx, sub, channel)
			}
		}
		// Los borradores sin sello aún son accesibles para completar su aceptación; nunca saltar el control para un enviado legado.
		if handled && err != nil && sub != uuid.Nil {
			v, e := r.q.GetSubmissionByID(ctx, sub)
			if e == nil && v.Status != "submitted" {
				err = nil
			}
		}
		if err != nil {
			a, _ := identity.ApplicantFromContext(ctx)
			if a != nil && strings.Contains(path, "/admin/") {
				err = audit.Record(ctx, r.audit, audit.Event{ActorID: a.ID, Role: a.Role, SubmissionID: sub, ResourceID: sub, ResourceType: "postulacion", Action: "privacidad.acceso_denegado", CreatedAt: time.Now().UTC()}, err)
			}
			status, code := 409, "privacidad_pendiente"
			if !errors.Is(err, privacy.ErrDenied) {
				status, code = 503, "privacidad_no_disponible"
			}
			w.Header().Set("Cache-Control", "no-store")
			httpx.RespondError(w, req, status, code, "El tratamiento requiere una política vigente; no se elimina evidencia histórica automáticamente.")
			return
		}
		next.ServeHTTP(w, req)
	})
}

/** Sender comprueba la política antes de preparar adjuntos o entregar a SMTP; nunca bloquea OTP que no contiene expediente. */
type Sender struct {
	Gate *Repository
	Next mail.Sender
}

func (s *Sender) Send(ctx context.Context, m mail.Message) error {
	if !strings.HasPrefix(m.Kind, "otp.") {
		resource, e := uuid.Parse(m.ResourceID)
		if e != nil {
			return privacy.ErrDenied
		}
		sub, e := s.Gate.q.MailSubmission(ctx, resource)
		if e != nil {
			return privacy.ErrDenied
		}
		if e = s.Gate.Allow(ctx, sub, "correo"); e != nil {
			return e
		}
	}
	if s.Next == nil {
		return mail.ErrSMTPDisabled
	}
	return s.Next.Send(ctx, m)
}
