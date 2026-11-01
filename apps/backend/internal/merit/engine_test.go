package merit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"strings"
	"testing"
	"time"
)

// Esta matriz sólo ejercita el motor; no es una propuesta de puntajes institucionales.
func fixture(t *testing.T) (*Case, *Review) {
	t.Helper()
	max, threshold := int64(10000), int64(5000)
	def := RuleDefinition{Criteria: []Criterion{{Code: "estudios", Label: "Caso sintético", Metric: "education_count", UnitSize: 1, PointsPerUnit: 6000, MaximumPoints: 10000, Weight: 10000, Required: true}}, Requirements: []Requirement{{Code: "base", Label: "Verificación sintética"}}, MaximumTotal: &max, Threshold: &threshold, Rounding: "inferior", ExperienceAggregation: "unir_intervalos", DayCounting: "dias_transcurridos", TieBreakers: []string{"estudios", "resolucion_manual"}}
	for i := 1; i <= 13; i++ {
		answer := "No"
		if i == 1 {
			answer = "Sí"
		}
		def.Questions = append(def.Questions, QuestionRule{fmt.Sprintf("q%d", i), answer})
	}
	hash, err := definitionHash(def)
	if err != nil {
		t.Fatal(err)
	}
	rules := &RuleSet{ID: uuid.New(), CompetitionID: uuid.New(), Sector: "afiliados", Definition: def, Hash: hash, Approved: true}
	docID := uuid.New().String()
	sha := strings.Repeat("a", 64)
	content := fmt.Sprintf(`{"seccion4":{"sector":"afiliados"},"seccion5":{"titulos":[{"id":"grado-1","nivel":"Tercer Nivel de Grado","documentoPdf":{"id":%q,"sha256":%q}}]},"seccion9":{"q1":"Sí","q2":"No","q3":"No","q4":"No","q5":"No","q6":"No","q7":"No","q7Detalles":{"institucion":"dato legado"},"q8":"No","q9":"No","q10":"No","q11":"No","q12":"No","q13":"No"}}`, docID, sha)
	now := time.Date(2026, 10, 31, 15, 0, 0, 0, time.UTC)
	raw, err := submission.CanonicalContent([]byte(content))
	if err != nil {
		t.Fatal(err)
	}
	item := &submission.Submission{ID: uuid.New(), ApplicantID: uuid.New(), CompetitionID: &rules.CompetitionID, Status: submission.StatusSubmitted, SubmittedAt: &now, Content: raw, Hash: submission.HashContent(raw)}
	input := ReviewInput{RuleSetID: rules.ID, Documents: []RecordReview{{Section: 5, RecordID: "grado-1", DocumentID: docID, Verdict: "valido"}}, Requirements: []RequirementReview{{Code: "base", Satisfied: true}}, Motive: "Revisión de prueba"}
	for i := 1; i <= 13; i++ {
		input.Questions = append(input.Questions, QuestionReview{ID: fmt.Sprintf("q%d", i), Confirmed: true})
	}
	c := &Case{Snapshot: item, ActiveRules: rules, Documents: []Document{{ID: docID, Section: 5, RecordID: "grado-1", SHA256: sha}}}
	review := &Review{ID: uuid.New(), SubmissionID: item.ID, SnapshotHash: item.Hash, Version: 1, Input: input}
	c.Review = review
	return c, review
}
func TestVerifiedScoreAndPositiveFirstQuestion(t *testing.T) {
	c, r := fixture(t)
	v, err := Evaluate(c, r)
	if err != nil {
		t.Fatal(err)
	}
	if v.Total != 6000 || !v.CanAdvance || len(v.Blockers) != 0 || !v.ManualTie {
		t.Fatalf("resultado: %+v", v)
	}
	c.Snapshot.Content = []byte(strings.Replace(string(c.Snapshot.Content), `"q1":"Sí"`, `"q1":"No"`, 1))
	canonical, _ := submission.CanonicalContent(c.Snapshot.Content)
	c.Snapshot.Hash = submission.HashContent(canonical)
	r.SnapshotHash = c.Snapshot.Hash
	v, err = Evaluate(c, r)
	if err != nil || v.CanAdvance {
		t.Fatalf("q1 negativa no debe avanzar: %+v %v", v, err)
	}
}
func TestInvalidRulesFailClosed(t *testing.T) {
	for _, name := range []string{"empty", "weights", "threshold", "first_question", "rounding", "tie", "maximum", "metric"} {
		t.Run(name, func(t *testing.T) {
			c, _ := fixture(t)
			d := c.ActiveRules.Definition
			switch name {
			case "empty":
				d.Criteria = nil
			case "weights":
				d.Criteria[0].Weight = 9999
			case "threshold":
				d.Threshold = nil
			case "first_question":
				d.Questions[0].AcceptedAnswer = "No"
			case "rounding":
				d.Rounding = ""
			case "tie":
				d.TieBreakers = nil
			case "maximum":
				*d.MaximumTotal = 9999
			case "metric":
				d.Criteria[0].Metric = "client_score"
			}
			if !errors.Is(d.Validate(), ErrInvalid) {
				t.Fatal("aceptó matriz incompleta")
			}
		})
	}
}
func TestEvidenceReviewFailures(t *testing.T) {
	for _, name := range []string{"foreign_document", "wrong_sha", "missing_review", "duplicate_question", "unconfirmed", "invalid_document", "requirement", "seal", "rule_hash", "pending"} {
		t.Run(name, func(t *testing.T) {
			c, r := fixture(t)
			wantError := true
			switch name {
			case "foreign_document":
				c.Documents[0].ID = uuid.NewString()
			case "wrong_sha":
				c.Documents[0].SHA256 = strings.Repeat("b", 64)
			case "missing_review":
				r.Input.Documents = nil
			case "duplicate_question":
				r.Input.Questions[1].ID = "q1"
			case "unconfirmed":
				r.Input.Questions[1].Confirmed = false
				r.Input.Questions[1].Observation = "Pendiente de evidencia"
				wantError = false
			case "invalid_document":
				r.Input.Documents[0].Verdict = "invalido"
				r.Input.Documents[0].Observation = "No acredita el grado"
				wantError = false
			case "requirement":
				r.Input.Requirements[0].Satisfied = false
				r.Input.Requirements[0].Observation = "No acreditado"
				wantError = false
			case "seal":
				c.Snapshot.Hash = strings.Repeat("c", 64)
			case "rule_hash":
				c.ActiveRules.Hash = strings.Repeat("d", 64)
			case "pending":
				c.ActiveRules.Approved = false
			}
			v, err := Evaluate(c, r)
			if wantError && err == nil {
				t.Fatal("aceptó evidencia inválida")
			}
			if !wantError && (err != nil || v.CanAdvance) {
				t.Fatalf("no bloqueó avance: %+v %v", v, err)
			}
		})
	}
}
func TestExactScoreRoundingAndCaps(t *testing.T) {
	c := Criterion{UnitSize: 3, PointsPerUnit: 100, MaximumPoints: 100, Weight: 5000}
	for _, tc := range []struct {
		units int64
		round string
		want  int64
	}{{1, "inferior", 16}, {1, "mas_cercano", 17}, {3, "inferior", 50}, {1000000000, "mas_cercano", 50}} {
		if got := score(tc.units, c, tc.round); got != tc.want {
			t.Fatalf("score(%+v)=%d", tc, got)
		}
	}
}
func TestExperiencePoliciesAndSealedCutoff(t *testing.T) {
	c, _ := fixture(t)
	form, err := submission.DecodeForm([]byte(`{"seccion6":{"experiencias":[{"id":"a","tipo":"General","fechaInicio":"2026-10-01","actualmenteLaborando":false,"fechaFin":"2026-10-10"},{"id":"b","tipo":"General","fechaInicio":"2026-10-05","actualmenteLaborando":true}]}}`))
	if err != nil {
		t.Fatal(err)
	}
	// Los nombres de captura son los del contrato EF-12; no se usa la duración declarada.
	form.Experience.Experiences[1].Current = true
	valid := map[string]bool{recordKey(6, "a"): true, recordKey(6, "b"): true}
	d := c.ActiveRules.Definition
	criterion := Criterion{Metric: "experience_days"}
	units, err := metric(form, c.Snapshot, valid, criterion, d)
	if err != nil || units != 30 {
		t.Fatalf("unión: %d %v", units, err)
	}
	d.ExperienceAggregation = "sumar_registros"
	units, err = metric(form, c.Snapshot, valid, criterion, d)
	if err != nil || units != 35 {
		t.Fatalf("suma: %d %v", units, err)
	}
	d.DayCounting = "dias_inclusivos"
	units, err = metric(form, c.Snapshot, valid, criterion, d)
	if err != nil || units != 37 {
		t.Fatalf("inclusivo: %d %v", units, err)
	}
	form.Experience.Experiences[0].EndDate = "2027-01-01"
	if _, err = metric(form, c.Snapshot, valid, criterion, d); !errors.Is(err, ErrEvidence) {
		t.Fatal("aceptó fecha posterior al envío")
	}
}

type memoryRepo struct {
	Repository
	c *Case
}

func (m *memoryRepo) MutateCase(_ context.Context, _ *identity.Applicant, _ uuid.UUID, f func(*Case) error) (*Case, error) {
	raw, _ := json.Marshal(m.c)
	var next Case
	_ = json.Unmarshal(raw, &next)
	next.Snapshot = m.c.Snapshot
	next.Documents = m.c.Documents
	if err := f(&next); err != nil {
		return nil, err
	}
	m.c = &next
	return m.c, nil
}
func (m *memoryRepo) ReadCase(context.Context, uuid.UUID) (*Case, error) { return m.c, nil }
func TestExplicitPublicationAndCorrection(t *testing.T) {
	c, r := fixture(t)
	repo := &memoryRepo{c: c}
	s := NewService(repo, nil)
	admin := &identity.Applicant{ID: uuid.New(), Role: identity.RoleAdministrator}
	analyst := &identity.Applicant{ID: uuid.New(), Role: identity.RoleAnalyst}
	id := c.Snapshot.ID
	calculated, err := s.Calculate(context.Background(), analyst, id, CalculateInput{ReviewID: r.ID, Motive: "Cálculo inicial"})
	if err != nil {
		t.Fatal(err)
	}
	input := PublishInput{CalculationID: calculated.Calculation.ID, Decision: "avanza", Motive: "Resolución sintética"}
	if _, err = s.Publish(context.Background(), analyst, id, input); !errors.Is(err, ErrForbidden) {
		t.Fatal("analista publicó")
	}
	published, err := s.Publish(context.Background(), admin, id, input)
	if err != nil {
		t.Fatal(err)
	}
	old := *published.Publication
	if _, err = s.Publish(context.Background(), admin, id, input); !errors.Is(err, ErrConflict) {
		t.Fatal("sobrescritura silenciosa")
	}
	repo.c.Review.Input.Motive = "Nueva revisión"
	repo.c.Review.ID = uuid.New()
	_, err = s.Calculate(context.Background(), analyst, id, CalculateInput{ReviewID: repo.c.Review.ID, Version: 1, Motive: "Recalcular explícitamente"})
	if err != nil {
		t.Fatal(err)
	}
	if repo.c.Publication.ID != old.ID {
		t.Fatal("recalcular sustituyó publicación")
	}
	input.CalculationID = repo.c.Calculation.ID
	input.Version = 1
	input.SupersedesID = &old.ID
	input.Decision = "excluido"
	input.Motive = "Corrección explícita de la resolución"
	corrected, err := s.Publish(context.Background(), admin, id, input)
	if err != nil || corrected.Publication.Version != 2 || *corrected.Publication.SupersedesID != old.ID {
		t.Fatalf("corrección: %v", err)
	}
}

func TestSupportedMetricsIgnoreUnverifiedRecords(t *testing.T) {
	c, _ := fixture(t)
	form := &submission.Form{
		Education:    &submission.EducationSection{Degrees: []submission.Degree{{ID: "e1", Level: "Tercer Nivel de Grado"}, {ID: "e2", Level: "Cuarto Nivel - Maestría Académica"}}},
		Publications: &submission.PublicationsSection{Publications: []submission.Publication{{ID: "p1", Type: "Publicación"}, {ID: "p2", Type: "Trabajo de investigación"}}},
		Training:     &submission.TrainingSection{Events: []submission.Training{{ID: "t1", Type: "Curso", Hours: 20}, {ID: "t2", Type: "Curso", Hours: 999}}},
	}
	valid := map[string]bool{recordKey(5, "e1"): true, recordKey(5, "e2"): true, recordKey(7, "p1"): true, recordKey(8, "t1"): true}
	for _, tc := range []struct {
		metric, filter string
		want           int64
	}{{"education_count", "", 2}, {"education_count", "Tercer Nivel de Grado", 1}, {"publication_count", "", 1}, {"publication_count", "Trabajo de investigación", 0}, {"training_hours", "Curso", 20}} {
		units, err := metric(form, c.Snapshot, valid, Criterion{Metric: tc.metric, Filter: tc.filter}, c.ActiveRules.Definition)
		if err != nil || units != tc.want {
			t.Fatal(tc, units, err)
		}
	}
	form.Training.Events[0].Hours = -1
	if _, err := metric(form, c.Snapshot, valid, Criterion{Metric: "training_hours"}, c.ActiveRules.Definition); !errors.Is(err, ErrEvidence) {
		t.Fatal("horas inválidas", err)
	}
}

func TestChecklistOnlyScoresConfirmedRequirementsAndDisqualifications(t *testing.T) {
	for _, blocked := range []string{"", "requirement", "question"} {
		t.Run(blocked, func(t *testing.T) {
			c, r := fixture(t)
			def := &c.ActiveRules.Definition
			def.Criteria[0].Metric = "requirements_approved"
			def.Criteria[0].PointsPerUnit = 10000
			c.ActiveRules.Hash, _ = definitionHash(*def)
			if blocked == "requirement" {
				r.Input.Requirements[0].Satisfied = false
				r.Input.Requirements[0].Observation = "Sintético: no cumple"
			}
			if blocked == "question" {
				r.Input.Questions[0].Confirmed = false
				r.Input.Questions[0].Observation = "Sintético: pendiente"
			}
			value, err := Evaluate(c, r)
			if err != nil {
				t.Fatal(err)
			}
			want := int64(10000)
			if blocked != "" {
				want = 0
			}
			if value.Total != want {
				t.Fatalf("checklist=%d, esperado=%d", value.Total, want)
			}
		})
	}
}

func TestCompleteExperienceUnitsDoNotPayFractionalYears(t *testing.T) {
	c := Criterion{UnitSize: 365, PointsPerUnit: 2500, MaximumPoints: 10000, Weight: 3000, WholeUnits: true}
	for days, want := range map[int64]int64{0: 0, 364: 0, 365: 750, 729: 750, 730: 1500, 1461: 3000} {
		if got := score(days, c, "mas_cercano"); got != want {
			t.Fatalf("días=%d, puntaje=%d, esperado=%d", days, got, want)
		}
	}
	c.WholeUnits = false
	if got := score(364, c, "mas_cercano"); got == 0 {
		t.Fatal("se alteró el comportamiento histórico proporcional")
	}
}
