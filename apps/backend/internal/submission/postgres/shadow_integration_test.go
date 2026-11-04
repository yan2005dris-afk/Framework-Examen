package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
)

// accountReader devuelve la cuenta del entorno de integración (misma cédula y correo que form_cases.json).
type accountReader struct{ applicant *identity.Applicant }

func (a accountReader) GetByID(context.Context, uuid.UUID) (*identity.Applicant, error) {
	return a.applicant, nil
}

// capturingRepository guarda el formulario que el servicio calculó dentro de la
// última mutación: el oráculo independiente de lo que después se lee de las tablas.
type capturingRepository struct {
	*Repository
	content json.RawMessage
}

func (c *capturingRepository) Mutate(ctx context.Context, id uuid.UUID, change func(context.Context, *submission.Submission) error) (*submission.Submission, error) {
	return c.Repository.Mutate(ctx, id, func(ctx context.Context, item *submission.Submission) error {
		if err := change(ctx, item); err != nil {
			return err
		}
		c.content = append(json.RawMessage(nil), item.Content...)
		return nil
	})
}

// assertShadow compara el formulario armado desde las tablas con el que calculó el servicio.
func assertShadow(t *testing.T, capture *capturingRepository, id uuid.UUID, step string) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()
	repo := capture.Repository
	stored, err := submission.DecodeForm(capture.content)
	if err != nil {
		t.Fatalf("%s: formulario del servicio: %v", step, err)
	}
	submission.NormalizeDurations(stored, now)
	loaded, err := repo.loadForm(ctx, id, now)
	if err != nil {
		t.Fatalf("%s: tablas: %v", step, err)
	}
	if want, got := canonicalForm(t, stored), canonicalForm(t, loaded); want != got {
		t.Fatalf("%s: las tablas difieren de content:\ncontent %s\ntablas  %s", step, want, got)
	}
}

// sectionOf extrae una sección del formulario base de los casos compartidos.
func sectionOf(t *testing.T, base json.RawMessage, number int) json.RawMessage {
	t.Helper()
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(base, &fields); err != nil {
		t.Fatal(err)
	}
	return fields[fmt.Sprintf("seccion%d", number)]
}

// savedAt devuelve la marca de guardado de una sección para verificar que no se reescribió.
func savedAt(t *testing.T, pool *pgxpool.Pool, id uuid.UUID, section int) time.Time {
	t.Helper()
	var at time.Time
	if err := pool.QueryRow(context.Background(), `SELECT saved_at FROM submission_sections WHERE submission_id=$1 AND section=$2`, id, section).Scan(&at); err != nil {
		t.Fatalf("sección %d: %v", section, err)
	}
	return at
}

// TestPostgresFormDualWriteShadowsContent recorre el ciclo completo del borrador
// con el servicio real y exige que las tablas reproduzcan `content` en cada paso.
func TestPostgresFormDualWriteShadowsContent(t *testing.T) {
	repo, item, pool := integrationEnvironment(t)
	ctx := context.Background()
	card, err := identity.NewIDCard("1710034008")
	if err != nil {
		t.Fatal(err)
	}
	capture := &capturingRepository{Repository: repo}
	service := submission.NewService(capture).WithApplicants(accountReader{&identity.Applicant{
		ID: item.ApplicantID, IDCard: card, FirstName: "Persona", LastName: "Prueba", Email: "prueba@example.test",
	}})
	_, cases := sharedCases(t)
	base := cases["Formulario completo sin firma digital"]

	for number := 1; number <= 9; number++ {
		if _, err := service.SaveSection(ctx, item.ID, number, sectionOf(t, base, number)); err != nil {
			t.Fatalf("guardar sección %d: %v", number, err)
		}
		assertShadow(t, capture, item.ID, fmt.Sprintf("sección %d", number))
	}

	// Guardar una sección no reescribe las demás.
	before := savedAt(t, pool, item.ID, 5)
	time.Sleep(10 * time.Millisecond)
	if _, err := service.SaveSection(ctx, item.ID, 1, json.RawMessage(`{"estadoCivil":"Casado/a","nombreConyuge":"Persona Pareja","cedulaConyuge":"1710034016"}`)); err != nil {
		t.Fatal(err)
	}
	assertShadow(t, capture, item.ID, "pareja")
	if !savedAt(t, pool, item.ID, 5).Equal(before) {
		t.Fatal("guardar la sección 1 reescribió la sección 5")
	}

	degree := insertDocument(t, pool, item.ID, 5, "titulo-1")
	var doc struct {
		Name      string
		Size      int64
		SHA       string
		CreatedAt time.Time
	}
	if err := pool.QueryRow(ctx, `SELECT name,size_bytes,sha256,created_at FROM documents WHERE id=$1`, degree).Scan(&doc.Name, &doc.Size, &doc.SHA, &doc.CreatedAt); err != nil {
		t.Fatal(err)
	}
	reference, _ := json.Marshal(submission.NewDocumentReference(degree, doc.Name, doc.Size, doc.SHA, doc.CreatedAt))

	steps := []struct {
		name    string
		section int
		patch   string
	}{
		{"familiar", 3, `{"tieneFamiliaresIessBiess":"Sí","familiares":[{"id":"familiar-1","nombres":"Persona Prueba","parentesco":"padre","institucion":"IESS","area":"Área","cargo":"Cargo"}]}`},
		{"país extranjero", 2, `{"pais":"España","provincia":"Madrid","canton":"Alcalá de Henares"}`},
		{"vuelta a Ecuador", 2, `{"pais":"Ecuador","provincia":"Azuay","canton":"Cuenca"}`},
		{"respaldo del título", 5, `{"titulos":[{"id":"titulo-1","nivel":"Tercer Nivel de Grado","institucion":"Universidad de prueba","tipo":"Nacional","titulo":"Economista","area":"Economía","registroSenescyt":"PRUEBA","fechaRegistroSenescyt":"2015-01-01","documentoPdf":` + string(reference) + `}]}`},
		{"empleo vigente", 6, `{"experiencias":[{"id":"experiencia-1","tipo":"General","cargo":"Analista","institucion":"Entidad","fechaInicio":"2020-01-01","fechaFin":"2021-03-15","actividades":"Finanzas","descripcionFunciones":"Funciones"},{"id":"experiencia-2","tipo":"Dirección","cargo":"Directora","institucion":"Banco","fechaInicio":"2021-04-01","actualmenteLaborando":true,"actividades":"Economía","descripcionFunciones":"Dirección"}]}`},
		{"capacitación", 8, `{"capacitaciones":[{"id":"curso-1","tipo":"Curso","nombre":"Riesgos","institucion":"IAEN","fechaInicio":"2022-03-01","fechaFin":"2022-03-05","horas":40}]}`},
		{"remoción declarada", 9, `{"q7":"Sí","q7Detalles":{"institucion":"Superintendencia","cargo":"Vocal","fechaFin":"2021-06-30","motivo":"Reestructuración"}}`},
		{"remoción retirada", 9, `{"q7":"No","q7Detalles":null}`},
	}
	for _, step := range steps {
		if _, err := service.SaveSection(ctx, item.ID, step.section, json.RawMessage(step.patch)); err != nil {
			t.Fatalf("%s: %v", step.name, err)
		}
		assertShadow(t, capture, item.ID, step.name)
	}

	// Una referencia a un PDF ajeno no bloquea el borrador: se descarta en las tablas.
	if _, err := service.SaveSection(ctx, item.ID, 7, json.RawMessage(`{"cuentaConPublicaciones":"Sí","publicaciones":[{"id":"pub-1","titulo":"T","tipo":"Publicación","medio":"M","fechaPublicacion":"2020-01-01","relacionCampo":"Sí","documentoPdf":{"id":"`+uuid.NewString()+`","nombre":"x.pdf","tamano":"1"}}]}`)); err != nil {
		t.Fatalf("referencia ajena: %v", err)
	}
	var linked *uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT document_id FROM submission_publications WHERE submission_id=$1`, item.ID).Scan(&linked); err != nil || linked != nil {
		t.Fatalf("la referencia ajena no debe vincularse: %v %v", linked, err)
	}
	if _, err := service.SaveSection(ctx, item.ID, 7, sectionOf(t, base, 7)); err != nil {
		t.Fatal(err)
	}
	assertShadow(t, capture, item.ID, "publicaciones sin referencia")

	// El sello escribe las tablas antes de cambiar el estado y quedan iguales al snapshot.
	sealed, err := service.Submit(ctx, item.ID, &submission.Consent{DataProcessing: true, Truthfulness: true, CompetitionRules: true, NamePublication: true, PhysicalDelivery: true})
	if err != nil {
		t.Fatalf("enviar: %v", err)
	}
	// El hash se calcula sobre el JSON canónico; la base devuelve el jsonb con su propio formato.
	canonical, err := submission.CanonicalContent(sealed.Content)
	if err != nil || sealed.Status != submission.StatusSubmitted || sealed.Hash != submission.HashContent(canonical) {
		t.Fatalf("sello inesperado: %s %s %v", sealed.Status, sealed.Hash, err)
	}
	assertShadow(t, capture, item.ID, "sello")
	// El snapshot guardado es exactamente el formulario que selló el servicio.
	if sealedForm, _ := submission.CanonicalContent(capture.content); string(sealedForm) != string(canonical) {
		t.Fatal("el snapshot sellado difiere del formulario calculado por el servicio")
	}
	var accepted *time.Time
	if err := pool.QueryRow(ctx, `SELECT accepted_at FROM submission_consents WHERE submission_id=$1`, item.ID).Scan(&accepted); err != nil || accepted == nil {
		t.Fatalf("la aceptación sellada debe quedar en la tabla: %v %v", accepted, err)
	}
}
