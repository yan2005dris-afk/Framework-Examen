package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/testutil"
)

// TestPostgresCreateWritesFormToTables cubre el error de la fase 5: `create` no debe
// perder el formulario inicial al recibir la fila con `content` NULL.
func TestPostgresCreateWritesFormToTables(t *testing.T) {
	repo, item, pool := integrationEnvironment(t)
	ctx := context.Background()
	_, cases := sharedCases(t)
	base := cases["Formulario completo sin firma digital"]

	draft, err := submission.NewSubmission(uuid.New(), item.ApplicantID, "CREAR-"+uuid.NewString()[:8], base, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	// Otra convocatoria: la misma cuenta sólo puede tener una postulación por período.
	competition := testutil.OpenCompetition(t, pool)
	draft.CompetitionID = &competition
	if err := repo.Create(ctx, draft); err != nil {
		t.Fatalf("crear con formulario: %v", err)
	}

	var sections int
	var hasContent bool
	if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM submission_sections WHERE submission_id=$1), content IS NOT NULL FROM submissions WHERE id=$1`, draft.ID).Scan(&sections, &hasContent); err != nil {
		t.Fatal(err)
	}
	if sections != 9 || hasContent {
		t.Fatalf("el borrador debe quedar en las 9 secciones y sin content: secciones=%d content=%v", sections, hasContent)
	}
	loaded, err := repo.FindByID(ctx, draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	want, err := repo.expectedForm(ctx, draft.ID, base, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := submission.DecodeForm(loaded.Content); canonicalText(got) != canonicalText(want) {
		t.Fatalf("el formulario creado no se lee igual:\n%s\n%s", loaded.Content, canonicalText(want))
	}
}

// closingAssessor simula la respuesta del Historial Laboral que cierra la postulación.
type closingAssessor struct{}

func (closingAssessor) Assess(_ context.Context, item *submission.Submission, form *submission.Form) (*submission.EligibilityAssessment, error) {
	sector := ""
	if form.Position != nil {
		sector = form.Position.Sector
	}
	competition := uuid.Nil
	if item.CompetitionID != nil {
		competition = *item.CompetitionID
	}
	return &submission.EligibilityAssessment{ID: uuid.New(), ApplicantID: item.ApplicantID, CompetitionID: competition, Sector: sector,
		Eligible: false, Source: "simulacion_prueba", CheckedAt: time.Now().UTC(), Message: "no mantiene la calidad de afiliado al IESS"}, nil
}

// TestPostgresClosureKeepsFormSnapshot exige que un cierre por elegibilidad congele
// en `content` el formulario previo al cierre, sin aceptar el cambio rechazado.
func TestPostgresClosureKeepsFormSnapshot(t *testing.T) {
	repo, item, pool := integrationEnvironment(t)
	ctx := context.Background()
	card, err := identity.NewIDCard("1710034008")
	if err != nil {
		t.Fatal(err)
	}
	account := accountReader{&identity.Applicant{ID: item.ApplicantID, IDCard: card, FirstName: "Persona", LastName: "Prueba", Email: "prueba@example.test"}}
	_, cases := sharedCases(t)
	base := cases["Formulario completo sin firma digital"]

	open := submission.NewService(repo).WithApplicants(account)
	for _, number := range []int{1, 2, 3} {
		if _, err := open.SaveSection(ctx, item.ID, number, sectionOf(t, base, number)); err != nil {
			t.Fatalf("sección %d: %v", number, err)
		}
	}
	before, err := repo.formContent(ctx, item.ID)
	if err != nil {
		t.Fatal(err)
	}

	closing := submission.NewService(repo).WithApplicants(account).WithEligibility(closingAssessor{})
	if _, err := closing.SaveSection(ctx, item.ID, 4, sectionOf(t, base, 4)); !errors.Is(err, submission.ErrEligibilityClosed) {
		t.Fatalf("se esperaba el cierre por elegibilidad: %v", err)
	}

	var status string
	var snapshot json.RawMessage
	if err := pool.QueryRow(ctx, `SELECT status, content FROM submissions WHERE id=$1`, item.ID).Scan(&status, &snapshot); err != nil {
		t.Fatal(err)
	}
	if status != string(submission.StatusClosed) || snapshot == nil {
		t.Fatalf("el expediente cerrado debe conservar su snapshot: estado=%s", status)
	}
	canonical, err := submission.CanonicalContent(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if string(canonical) != string(before) {
		t.Fatalf("el snapshot del cierre debe ser el formulario previo:\n%s\n%s", canonical, before)
	}
	// La sección 4 rechazada no llegó a las tablas y el expediente cerrado ya no admite cambios.
	var rejected int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM submission_sections WHERE submission_id=$1 AND section=4`, item.ID).Scan(&rejected); err != nil || rejected != 0 {
		t.Fatalf("la sección rechazada no debe guardarse: %d %v", rejected, err)
	}
	if _, err := open.SaveSection(ctx, item.ID, 5, sectionOf(t, base, 5)); err == nil {
		t.Fatal("un expediente cerrado no debe admitir más cambios")
	}
}
