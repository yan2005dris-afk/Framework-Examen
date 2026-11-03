package review

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/shared/domain"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"time"
)

const RefreshSeconds = 30
const MaxExportRows = 10000
const MaxExportBytes = 32 * 1024 * 1024
const MatrixVersion = "EF13-TECNICA-v1"

/** Service conserva la autorización y la auditoría incluso fuera del adaptador HTTP. */
type Service struct {
	repo    Repository
	auditor audit.Recorder
	now     func() time.Time
}

/** NewService conecta fuentes persistentes sin dependencias de transporte ni almacenamiento. */
func NewService(repo Repository, auditor audit.Recorder) *Service {
	return &Service{repo: repo, auditor: auditor, now: time.Now}
}

/** ValidateFilter no admite borradores, períodos implícitos ni sectores inventados. */
func ValidateFilter(f Filter) error {
	if (f.CompetitionID == nil && !f.Legacy) || (f.CompetitionID != nil && f.Legacy) {
		return ErrInvalidFilter
	}
	if f.CompetitionID != nil && *f.CompetitionID == uuid.Nil {
		return ErrInvalidFilter
	}
	if f.Sector != "" && f.Sector != "afiliados" && f.Sector != "jubilados" && f.Sector != "empleadores" {
		return ErrInvalidFilter
	}
	if f.Status != "" && f.Status != "submitted" {
		return ErrInvalidFilter
	}
	return nil
}

func filterScope(f Filter) string {
	raw, _ := json.Marshal(f)
	sum := sha256.Sum256(raw)
	return "revision:" + hex.EncodeToString(sum[:])
}
func (s *Service) validate(ctx context.Context, f Filter) error {
	if err := ValidateFilter(f); err != nil {
		return err
	}
	if f.CompetitionID != nil {
		c, err := s.repo.Competition(ctx, *f.CompetitionID)
		if err != nil {
			return err
		}
		if c == nil {
			return ErrCompetitionNotFound
		}
	}
	return nil
}
func (s *Service) record(ctx context.Context, a *identity.Applicant, f Filter, action string, result error) error {
	id := uuid.Nil
	if f.CompetitionID != nil {
		id = *f.CompetitionID
	}
	return audit.Record(ctx, s.auditor, audit.Event{ActorID: a.ID, Role: a.Role, Action: action, ResourceID: id, ResourceType: "convocatoria", CreatedAt: s.now().UTC()}, result)
}

/** List devuelve todas las páginas de la nómina sin exponer contactos ni PDFs en listados. */
func (s *Service) List(ctx context.Context, a *identity.Applicant, f Filter, q domain.CursorRequest) (page domain.CursorPage[Applicant], err error) {
	if !identity.HasPermission(a, identity.PermissionReview) {
		return page, ErrForbidden
	}
	defer func() {
		err = s.record(ctx, a, f, "nomina.consultada", err)
		if err != nil {
			page = domain.CursorPage[Applicant]{}
		}
	}()
	if err = s.validate(ctx, f); err != nil {
		return page, err
	}
	q = domain.NewCursorRequest(q.Limit, q.Cursor)
	scope := filterScope(f)
	position, err := domain.ParsePosition(q.Cursor, scope)
	if err != nil {
		return page, err
	}
	snapshots, err := s.repo.List(ctx, f, position, q.Limit+1)
	if err != nil {
		return page, err
	}
	more := len(snapshots) > q.Limit
	if more {
		snapshots = snapshots[:q.Limit]
	}
	items, err := project(snapshots)
	if err != nil {
		return page, err
	}
	next := ""
	if more {
		last := snapshots[len(snapshots)-1]
		next = domain.EncodePosition(scope, last.CreatedAt, last.ID)
	}
	return domain.NewCursorPage(items, next, more, q.Limit), nil
}

/** Counts consulta PostgreSQL en cada petición; un fallo no se disfraza como cero postulantes. */
func (s *Service) Counts(ctx context.Context, a *identity.Applicant, f Filter) (result Stats, err error) {
	if !identity.HasPermission(a, identity.PermissionReview) {
		return result, ErrForbidden
	}
	defer func() {
		err = s.record(ctx, a, f, "conteos.consultados", err)
		if err != nil {
			result = Stats{}
		}
	}()
	if err = s.validate(ctx, f); err != nil {
		return result, err
	}
	counts, err := s.repo.Counts(ctx, f)
	if err != nil {
		return result, err
	}
	if counts == nil {
		counts = []Count{}
	}
	return Stats{Filter: f, AsOf: s.now().UTC(), RefreshAfterSeconds: RefreshSeconds, Counts: counts}, nil
}

/** Competitions pagina períodos activos y cerrados, sin convertirlos en fases aprobadas. */
func (s *Service) Competitions(ctx context.Context, a *identity.Applicant, q domain.CursorRequest) (page domain.CursorPage[Competition], err error) {
	if !identity.HasPermission(a, identity.PermissionReview) {
		return page, ErrForbidden
	}
	defer func() {
		err = s.record(ctx, a, Filter{}, "convocatorias.consultadas", err)
		if err != nil {
			page = domain.CursorPage[Competition]{}
		}
	}()
	q = domain.NewCursorRequest(q.Limit, q.Cursor)
	p, err := domain.ParsePosition(q.Cursor, "convocatorias")
	if err != nil {
		return page, err
	}
	items, err := s.repo.Competitions(ctx, p, q.Limit+1)
	if err != nil {
		return page, err
	}
	more := len(items) > q.Limit
	next := ""
	if more {
		items = items[:q.Limit]
		last := items[len(items)-1]
		next = domain.EncodePosition("convocatorias", last.CreatedAt, last.ID)
	}
	for i := range items {
		items[i].Closed = !s.now().Before(items[i].ClosesAt)
	}
	return domain.NewCursorPage(items, next, more, q.Limit), nil
}

/** Export materializa una selección completa y audita antes de entregar bytes; nunca trunca en silencio. */
func (s *Service) Export(ctx context.Context, a *identity.Applicant, f Filter, format string) (data []byte, err error) {
	if !identity.HasPermission(a, identity.PermissionReview) {
		return nil, ErrForbidden
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	defer func() {
		action := "matriz.exportacion_rechazada"
		if format == "xlsx" || format == "pdf" {
			action = "matriz.exportada." + format
		}
		err = s.record(ctx, a, f, action, err)
		if err != nil {
			data = nil
		}
	}()
	if format != "xlsx" && format != "pdf" {
		return nil, ErrInvalidFilter
	}
	if err = s.validate(ctx, f); err != nil {
		return nil, err
	}
	snapshots, err := s.repo.Export(ctx, f)
	if err != nil {
		return nil, err
	}
	if len(snapshots) > MaxExportRows {
		return nil, ErrExportTooLarge
	}
	items, err := project(snapshots)
	if err != nil {
		return nil, err
	}
	return renderMatrix(ctx, items, f, s.now().UTC(), format)
}

func project(snapshots []Snapshot) ([]Applicant, error) {
	items := make([]Applicant, 0, len(snapshots))
	for _, row := range snapshots {
		canonical, err := submission.CanonicalContent(row.Content)
		if err != nil || submission.HashContent(canonical) != row.Hash || row.Status != "submitted" || row.SubmittedAt.IsZero() {
			return nil, ErrSnapshot
		}
		form, err := submission.DecodeForm(row.Content)
		if err != nil {
			return nil, ErrSnapshot
		}
		p := Applicant{ID: row.ID, Code: row.Code, Competition: row.CompetitionCode, Status: row.Status,
			Hash: row.Hash, SubmittedAt: row.SubmittedAt, CreatedAt: row.CreatedAt, Education: []Education{}, Experiences: []Experience{}}
		if form.Personal != nil {
			p.FirstName = form.Personal.FirstName
			p.LastName = form.Personal.LastName
		}
		if form.Position != nil {
			p.Sector = form.Position.Sector
		}
		if form.Education != nil {
			for _, e := range form.Education.Degrees {
				p.Education = append(p.Education, Education{Level: e.Level, Title: e.Title})
			}
		}
		if form.Experience != nil {
			for _, e := range form.Experience.Experiences {
				duration := e.Duration
				// Compatibilidad con sellos antiguos: calcular al día del envío, jamás al día de consulta.
				if duration == nil {
					start, err := time.Parse(time.DateOnly, e.StartDate)
					if err != nil {
						return nil, ErrSnapshot
					}
					end, err := time.Parse(time.DateOnly, e.EndDate)
					if e.Current {
						ec := time.FixedZone("America/Guayaquil", -5*60*60)
						d := row.SubmittedAt.In(ec)
						end = time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
						err = nil
					}
					if err != nil || end.Before(start) {
						return nil, ErrSnapshot
					}
					value := submission.CalculateDuration(start, end)
					duration = &value
				}
				doc := ""
				if e.Document != nil {
					doc = e.Document.ID
				}
				p.Experiences = append(p.Experiences, Experience{ID: e.ID, Type: e.Type, Position: e.Position, Duration: duration, DocumentID: doc})
			}
		}
		if form.Publications != nil {
			p.Publications = len(form.Publications.Publications)
		}
		if form.Training != nil {
			for _, e := range form.Training.Events {
				p.TrainingHours += e.Hours
			}
		}
		items = append(items, p)
	}
	return items, nil
}

/** MatrixDescription distingue esta propuesta de una calificación o lista oficial de elegibles. */
func MatrixDescription() string {
	return fmt.Sprintf("%s · Propuesta técnica interna; sin puntajes ni orden de méritos", MatrixVersion)
}
