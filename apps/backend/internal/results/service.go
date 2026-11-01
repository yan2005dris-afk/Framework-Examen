package results

import (
	"context"
	"encoding/csv"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/shared/domain"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type Service struct {
	repo    Repository
	auditor audit.Recorder
}

func NewService(r Repository, a audit.Recorder) *Service { return &Service{r, a} }
func (s *Service) record(ctx context.Context, a *identity.Applicant, id uuid.UUID, action string, err error) error {
	return audit.Record(ctx, s.auditor, audit.Event{ActorID: a.ID, Role: a.Role, ResourceID: id, ResourceType: "convocatoria", Action: action, CreatedAt: time.Now().UTC()}, err)
}
func (s *Service) Generate(ctx context.Context, a *identity.Applicant, id uuid.UUID, sector string, key uuid.UUID, input GenerateInput) (*List, error) {
	if !identity.HasPermission(a, identity.PermissionResults) {
		return nil, ErrForbidden
	}
	if id == uuid.Nil || key == uuid.Nil || !ValidSector(sector) || input.Rules.Validate() != nil {
		return nil, ErrInvalid
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	return s.repo.Generate(ctx, a, id, sector, key, input)
}
func (s *Service) Publish(ctx context.Context, a *identity.Applicant, id, key uuid.UUID, input PublishInput) (*Publication, error) {
	if !identity.HasPermission(a, identity.PermissionPhases) {
		return nil, ErrForbidden
	}
	if id == uuid.Nil || key == uuid.Nil || input.PreviousVersion < 0 || !text(input.Reference, 2000) || len(input.PrivacyReference) > 2000 {
		return nil, ErrInvalid
	}
	if input.Public && !text(input.PrivacyReference, 2000) {
		return nil, ErrPrivacy
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	return s.repo.Publish(ctx, a, id, key, input)
}
func (s *Service) Get(ctx context.Context, a *identity.Applicant, id uuid.UUID) (v *List, err error) {
	if !identity.HasPermission(a, identity.PermissionResults) {
		return nil, ErrForbidden
	}
	period := uuid.Nil
	defer func() {
		err = s.record(ctx, a, period, "lista.consultada", err)
		if err != nil {
			v = nil
		}
	}()
	v, err = s.repo.Get(ctx, id)
	if v != nil {
		period = v.CompetitionID
	}
	return v, err
}
func (s *Service) Publication(ctx context.Context, a *identity.Applicant, id uuid.UUID) (v *Publication, err error) {
	if !identity.HasPermission(a, identity.PermissionResults) {
		return nil, ErrForbidden
	}
	period := uuid.Nil
	defer func() {
		err = s.record(ctx, a, period, "lista.consultada", err)
		if err != nil {
			v = nil
		}
	}()
	v, err = s.repo.Publication(ctx, id)
	if v != nil {
		period = v.CompetitionID
	}
	return v, err
}
func (s *Service) Lists(ctx context.Context, a *identity.Applicant, id uuid.UUID, sector string, q domain.CursorRequest) (page domain.CursorPage[List], err error) {
	if !identity.HasPermission(a, identity.PermissionResults) {
		return page, ErrForbidden
	}
	defer func() {
		err = s.record(ctx, a, id, "listas.consultadas", err)
		if err != nil {
			page = domain.CursorPage[List]{}
		}
	}()
	if id == uuid.Nil || !ValidSector(sector) {
		return page, ErrInvalid
	}
	q = domain.NewCursorRequest(q.Limit, q.Cursor)
	scope := "result-lists:" + id.String() + ":" + sector
	p, err := domain.ParsePosition(q.Cursor, scope)
	if err != nil {
		return page, err
	}
	items, err := s.repo.Lists(ctx, id, sector, p, q.Limit+1)
	if err != nil {
		return page, err
	}
	more := len(items) > q.Limit
	next := ""
	if more {
		items = items[:q.Limit]
		v := items[len(items)-1]
		next = domain.EncodePosition(scope, v.CreatedAt, v.ID)
	}
	return domain.NewCursorPage(items, next, more, q.Limit), nil
}
func (s *Service) Publications(ctx context.Context, a *identity.Applicant, id uuid.UUID, sector string, q domain.CursorRequest) (page domain.CursorPage[Publication], err error) {
	if !identity.HasPermission(a, identity.PermissionResults) {
		return page, ErrForbidden
	}
	defer func() {
		err = s.record(ctx, a, id, "listas.consultadas", err)
		if err != nil {
			page = domain.CursorPage[Publication]{}
		}
	}()
	if id == uuid.Nil || !ValidSector(sector) {
		return page, ErrInvalid
	}
	q = domain.NewCursorRequest(q.Limit, q.Cursor)
	scope := "result-pubs:" + id.String() + ":" + sector
	p, err := domain.ParsePosition(q.Cursor, scope)
	if err != nil {
		return page, err
	}
	items, err := s.repo.Publications(ctx, id, sector, p, q.Limit+1)
	if err != nil {
		return page, err
	}
	more := len(items) > q.Limit
	next := ""
	if more {
		items = items[:q.Limit]
		v := items[len(items)-1]
		next = domain.EncodePosition(scope, v.CreatedAt, v.ID)
	}
	return domain.NewCursorPage(items, next, more, q.Limit), nil
}

/** Public sólo devuelve una publicación explícitamente pública; el transporte construye una proyección distinta del reporte interno. */
func (s *Service) Public(ctx context.Context, id uuid.UUID, sector string, current bool) (*Publication, error) {
	var v *Publication
	var err error
	if current {
		if !ValidSector(sector) {
			return nil, ErrInvalid
		}
		v, err = s.repo.Current(ctx, id, sector)
	} else {
		v, err = s.repo.Publication(ctx, id)
	}
	if err != nil {
		return nil, err
	}
	if v == nil || !v.Public {
		return nil, ErrNotFound
	}
	return v, nil
}
func cell(v string) string {
	v2 := strings.TrimLeftFunc(v, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) || r == '\ufeff' })
	if v2 != "" && strings.ContainsAny(v2[:1], "=+-@") {
		return "'" + v
	}
	return v
}

/** Export conserva la versión histórica y las claves de enlace con T9, no ejecuta una nueva calificación. */
func (s *Service) Export(ctx context.Context, a *identity.Applicant, id uuid.UUID) (data []byte, err error) {
	if !identity.HasPermission(a, identity.PermissionResults) {
		return nil, ErrForbidden
	}
	v, err := s.repo.Get(ctx, id)
	period := uuid.Nil
	if v != nil {
		period = v.CompetitionID
	}
	defer func() {
		err = s.record(ctx, a, period, "lista.exportada.csv", err)
		if err != nil {
			data = nil
		}
	}()
	if err != nil {
		return nil, err
	}
	var b strings.Builder
	w := csv.NewWriter(&b)
	if err = w.Write([]string{"version_lista", "convocatoria_id", "sector", "postulacion_id", "codigo", "hash_envio", "apellidos", "nombres", "elegible", "puesto", "empate", "resultado_meritos_id", "resultado_teorico_id", "resultado_psicotecnico_id", "resultado_consolidado_id", "puntaje_meritos", "maximo_meritos", "puntaje_teorico", "maximo_teorico", "puntaje_psicotecnico", "maximo_psicotecnico", "puntaje_consolidado", "maximo_consolidado", "hash_fuentes"}); err != nil {
		return nil, err
	}
	for _, e := range v.Entries {
		if err = ctx.Err(); err != nil {
			return nil, err
		}
		row := []string{strconv.FormatInt(v.Version, 10), v.CompetitionID.String(), v.Sector, e.SubmissionID.String(), e.Code, e.Hash, e.LastName, e.FirstName, strconv.FormatBool(e.Eligible), strconv.Itoa(e.Rank), strconv.FormatBool(e.Tied)}
		for _, phase := range []string{"meritos", "oposicion_teorica", "oposicion_psicotecnica", "resultados"} {
			id := e.Results[phase].ID
			s := ""
			if id != uuid.Nil {
				s = id.String()
			}
			row = append(row, s)
		}

		for _, phase := range []string{"meritos", "oposicion_teorica", "oposicion_psicotecnica", "resultados"} {
			source := e.Results[phase]
			score, maximum := "", ""
			if source.Score != nil {
				score = strconv.FormatInt(*source.Score, 10)
			}
			if source.Maximum != nil {
				maximum = strconv.FormatInt(*source.Maximum, 10)
			}
			row = append(row, score, maximum)
		}
		row = append(row, v.SourcesHash)
		for i := range row {
			row[i] = cell(row[i])
		}
		if err = w.Write(row); err != nil {
			return nil, err
		}
	}
	w.Flush()
	if err = w.Error(); err != nil {
		return nil, err
	}
	if b.Len() > MaxBytes {
		return nil, ErrTooLarge
	}
	return []byte(b.String()), nil
}
