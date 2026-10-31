package review

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/shared/domain"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"io"
	"strings"
	"testing"
	"time"
)

type memoryReview struct {
	snapshots []Snapshot
	counts    []Count
	period    *Competition
	fail      error
	calls     int
	filter    Filter
	position  domain.Position
	limit     int
}

func (m *memoryReview) Competition(context.Context, uuid.UUID) (*Competition, error) {
	return m.period, m.fail
}
func (m *memoryReview) Competitions(context.Context, domain.Position, int) ([]Competition, error) {
	if m.period == nil {
		return nil, m.fail
	}
	return []Competition{*m.period}, m.fail
}
func (m *memoryReview) List(_ context.Context, f Filter, p domain.Position, limit int) ([]Snapshot, error) {
	m.calls++
	m.filter = f
	m.position = p
	m.limit = limit
	return m.snapshots, m.fail
}
func (m *memoryReview) Export(ctx context.Context, f Filter) ([]Snapshot, error) {
	return m.List(ctx, f, domain.Position{}, MaxExportRows+1)
}
func (m *memoryReview) Counts(_ context.Context, f Filter) ([]Count, error) {
	m.calls++
	m.filter = f
	return m.counts, m.fail
}

type memoryAudit struct {
	events []audit.Event
	fail   bool
}

func (m *memoryAudit) Record(_ context.Context, e audit.Event) error {
	m.events = append(m.events, e)
	if m.fail {
		return errors.New("fallo")
	}
	return nil
}

func reviewSnapshot(t *testing.T) Snapshot {
	t.Helper()
	now := time.Date(2026, 10, 30, 18, 0, 0, 0, time.UTC)
	f := submission.Form{Personal: &submission.PersonalSection{FirstName: "María Ñúñez", LastName: "Prueba"}, Position: &submission.PositionSection{Sector: "afiliados"},
		Education: &submission.EducationSection{Degrees: []submission.Degree{{Level: "Cuarto Nivel", Title: "Economía"}}},
		Experience: &submission.ExperienceSection{Experiences: []submission.Experience{{ID: "exp-1", Type: "Dirección", Position: "Directora", StartDate: "2020-01-01", Current: true,
			Duration: &submission.Duration{Years: 6, Months: 9, Days: 29, Text: "6 años, 9 meses, 29 días"}}}}}
	data, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	data, err = submission.CanonicalContent(data)
	if err != nil {
		t.Fatal(err)
	}
	return Snapshot{ID: uuid.New(), Code: "EF12-TEST", CompetitionCode: "SIN-CONVOCATORIA", Status: "submitted", SubmittedAt: now, CreatedAt: now, Content: data, Hash: submission.HashContent(data)}
}

func TestReviewFilterDoesNotInferOrMixPeriods(t *testing.T) {
	id := uuid.New()
	for _, tc := range []struct {
		f     Filter
		valid bool
	}{{Filter{}, false}, {Filter{Legacy: true}, true}, {Filter{CompetitionID: &id}, true},
		{Filter{CompetitionID: &id, Legacy: true}, false}, {Filter{Legacy: true, Status: "draft"}, false}, {Filter{Legacy: true, Sector: "inventado"}, false},
		{Filter{Legacy: true, Sector: "jubilados", Status: "submitted"}, true}} {
		if (ValidateFilter(tc.f) == nil) != tc.valid {
			t.Fatalf("filtro: %+v", tc.f)
		}
	}
}

func TestReviewPermissionAuditFailureAndClosedPeriod(t *testing.T) {
	row := reviewSnapshot(t)
	periodID := uuid.New()
	for _, role := range []string{identity.RoleApplicant, identity.RoleAnalyst, identity.RoleAdministrator, "superadmin"} {
		t.Run(role, func(t *testing.T) {
			repo := &memoryReview{snapshots: []Snapshot{row}, period: &Competition{ID: periodID, ClosesAt: row.SubmittedAt.Add(-time.Hour)}}
			auditor := &memoryAudit{}
			s := NewService(repo, auditor)
			actor := &identity.Applicant{ID: uuid.New(), Role: role}
			page, err := s.List(context.Background(), actor, Filter{CompetitionID: &periodID}, domain.NewCursorRequest(20, nil))
			allowed := role == identity.RoleAnalyst || role == identity.RoleAdministrator
			if allowed {
				if err != nil || len(page.Items) != 1 || len(auditor.events) != 1 {
					t.Fatalf("histórico rechazado: %v", err)
				}
			} else if !errors.Is(err, ErrForbidden) || repo.calls != 0 {
				t.Fatal("se consultó sin permiso")
			}
			if allowed {
				auditor.fail = true
				page, err = s.List(context.Background(), actor, Filter{Legacy: true}, domain.NewCursorRequest(20, nil))
				if !errors.Is(err, audit.ErrUnavailable) || len(page.Items) != 0 {
					t.Fatal("datos sin auditoría")
				}
			}
		})
	}
}

func TestReviewCursorIsBoundToFiltersAndSnapshotDurationsNeverAge(t *testing.T) {
	row := reviewSnapshot(t)
	other := row
	other.ID = uuid.New()
	repo := &memoryReview{snapshots: []Snapshot{row, other}}
	s := NewService(repo, &memoryAudit{})
	s.now = func() time.Time { return row.SubmittedAt.AddDate(10, 0, 0) }
	actor := &identity.Applicant{ID: uuid.New(), Role: identity.RoleAnalyst}
	filter := Filter{Legacy: true}
	page, err := s.List(context.Background(), actor, filter, domain.NewCursorRequest(1, nil))
	if err != nil || !page.HasMore || repo.limit != 2 {
		t.Fatal(page, err)
	}
	if page.Items[0].Experiences[0].Duration.Years != 6 {
		t.Fatal("la experiencia creció después del sello")
	}
	cursor, err := domain.DecodeCursor(page.NextCursor)
	if err != nil {
		t.Fatal(err)
	}
	filter.Sector = "jubilados"
	if _, err = s.List(context.Background(), actor, filter, domain.NewCursorRequest(1, &cursor)); !errors.Is(err, domain.ErrInvalidCursor) {
		t.Fatal("cursor de otro filtro admitido", err)
	}
	row.Content = []byte(`{}`)
	repo.snapshots = []Snapshot{row}
	if _, err = s.List(context.Background(), actor, Filter{Legacy: true}, domain.NewCursorRequest(20, nil)); !errors.Is(err, ErrSnapshot) {
		t.Fatal("snapshot alterado admitido", err)
	}
}

func TestCountsAreFreshNotCachedAndFailuresAreNotZero(t *testing.T) {
	repo := &memoryReview{counts: []Count{{Sector: "afiliados", Applicants: 1}}}
	s := NewService(repo, &memoryAudit{})
	a := &identity.Applicant{ID: uuid.New(), Role: identity.RoleAnalyst}
	first, err := s.Counts(context.Background(), a, Filter{Legacy: true})
	if err != nil || first.RefreshAfterSeconds != 30 {
		t.Fatal(err)
	}
	repo.counts[0].Applicants = 2
	second, err := s.Counts(context.Background(), a, Filter{Legacy: true})
	if err != nil || second.Counts[0].Applicants != 2 || repo.calls != 2 {
		t.Fatal("conteo obsoleto")
	}
	repo.fail = errors.New("base no disponible")
	if _, err = s.Counts(context.Background(), a, Filter{Legacy: true}); err == nil {
		t.Fatal("fallo disfrazado de cero")
	}
}

func TestLegacyDurationUsesSealDateInEcuadorNotConsultationDate(t *testing.T) {
	row := reviewSnapshot(t)
	row.SubmittedAt = time.Date(2026, 10, 31, 2, 0, 0, 0, time.UTC)
	form, err := submission.DecodeForm(row.Content)
	if err != nil {
		t.Fatal(err)
	}
	form.Experience.Experiences[0].Duration = nil
	form.Experience.Experiences[0].StartDate = "2026-10-30"
	raw, err := json.Marshal(form)
	if err != nil {
		t.Fatal(err)
	}
	row.Content, err = submission.CanonicalContent(raw)
	if err != nil {
		t.Fatal(err)
	}
	row.Hash = submission.HashContent(row.Content)
	items, err := project([]Snapshot{row})
	if err != nil {
		t.Fatal(err)
	}
	d := items[0].Experiences[0].Duration
	if d.Years != 0 || d.Months != 0 || d.Days != 0 {
		t.Fatalf("fecha de corte histórica incorrecta: %+v", d)
	}
}

func TestExportsUseSameFiltersAndFailClosed(t *testing.T) {
	row := reviewSnapshot(t)
	repo := &memoryReview{snapshots: []Snapshot{row}}
	auditor := &memoryAudit{}
	s := NewService(repo, auditor)
	a := &identity.Applicant{ID: uuid.New(), Role: identity.RoleAnalyst}
	f := Filter{Legacy: true, Sector: "afiliados"}
	for _, format := range []string{"xlsx", "pdf"} {
		data, err := s.Export(context.Background(), a, f, format)
		if err != nil || len(data) == 0 || repo.filter != f || repo.limit != MaxExportRows+1 {
			t.Fatalf("%s: %v", format, err)
		}
		if format == "pdf" && !bytes.HasPrefix(data, []byte("%PDF-")) {
			t.Fatal("no es PDF")
		}
		if format == "xlsx" {
			validateWorkbook(t, data)
		}
	}
	auditor.fail = true
	if data, err := s.Export(context.Background(), a, f, "pdf"); !errors.Is(err, audit.ErrUnavailable) || len(data) != 0 {
		t.Fatal("exportación sin auditoría")
	}
	auditor.fail = false
	repo.snapshots = make([]Snapshot, MaxExportRows+1)
	if _, err := s.Export(context.Background(), a, f, "xlsx"); !errors.Is(err, ErrExportTooLarge) {
		t.Fatal("exportación truncada o ilimitada")
	}
}

func validateWorkbook(t *testing.T, data []byte) {
	t.Helper()
	book, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, file := range book.File {
		stream, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(stream)
		_ = stream.Close()
		if err != nil {
			t.Fatal(err)
		}
		decoder := xml.NewDecoder(bytes.NewReader(body))
		for {
			_, err := decoder.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("XML %s: %v", file.Name, err)
			}
		}
		if strings.Contains(string(body), "María Ñúñez") {
			found = true
		}
		if strings.Contains(string(body), "<f>") {
			t.Fatal("fórmula ejecutable")
		}
	}
	if !found {
		t.Fatal("se perdieron tildes o nombres")
	}
}

func TestExportsEscapeFormulaXMLAndLongPDFRows(t *testing.T) {
	row := reviewSnapshot(t)
	items, err := project([]Snapshot{row})
	if err != nil {
		t.Fatal(err)
	}
	items[0].FirstName = `=HYPERLINK("https://example.test") <&> Ñ`
	for i := 0; i < 160; i++ {
		items[0].Experiences = append(items[0].Experiences, Experience{ID: strings.Repeat("registro", 8), Type: "Dirección", Position: "Dirección económica", Duration: &submission.Duration{Years: 1}})
	}
	data, err := renderXLSX(context.Background(), items, []string{MatrixDescription()})
	if err != nil {
		t.Fatal(err)
	}
	book, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range book.File {
		if file.Name == "xl/worksheets/sheet1.xml" {
			r, _ := file.Open()
			body, _ := io.ReadAll(r)
			_ = r.Close()
			if !bytes.Contains(body, []byte("=HYPERLINK")) || bytes.Contains(body, []byte("<f>")) {
				t.Fatal("texto convertido a fórmula")
			}
		}
	}
	pdf, err := renderPDF(context.Background(), items, []string{MatrixDescription()}, time.Now())
	if err != nil || !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatal("fila larga: ", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := renderMatrix(ctx, items, Filter{Legacy: true}, time.Now(), "pdf"); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelación ignorada")
	}
}
