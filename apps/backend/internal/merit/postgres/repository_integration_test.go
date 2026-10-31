package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit"
	auditpostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit/postgres"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/competition"
	competitionpostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/competition/postgres"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/merit"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/db"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/testutil"
	"strings"
	"testing"
	"time"
)

type brokenAudit struct{}

func (brokenAudit) RecordWorkflow(context.Context, audit.Event) error { return audit.ErrUnavailable }

func TestPostgresMeritLifecycleAndRollback(t *testing.T) {
	pool := testutil.Postgres(t, "MERIT_TEST_DATABASE_URL")
	ctx := context.Background()
	q := sqlcgen.New(pool)
	tx := db.NewTxManager(pool)
	recorder := auditpostgres.NewRepository(q)
	admin := &identity.Applicant{ID: uuid.New(), Role: identity.RoleAdministrator}
	analyst := &identity.Applicant{ID: uuid.New(), Role: identity.RoleAnalyst}
	owner := &identity.Applicant{ID: uuid.New(), Role: identity.RoleApplicant}
	for i, a := range []*identity.Applicant{admin, analyst, owner} {
		if _, err := pool.Exec(ctx, `INSERT INTO applicants(id,cedula,nombres,apellidos,correo,role) VALUES($1,$2,'Persona','QA',$3,$4)`, a.ID, fmt.Sprintf("000000000%d", i+1), a.ID.String()+"@example.test", a.Role); err != nil {
			t.Fatal(err)
		}
	}
	start, end := time.Now().Add(-time.Hour).UTC(), time.Now().Add(time.Hour).UTC()
	period := &competition.Competition{ID: uuid.New(), Code: "MERITOS-QA-NO-OFICIAL", Title: "Prueba aislada", OpensAt: start, ClosesAt: end, Configuration: competition.Configuration{Sectors: []string{"afiliados"}, Phases: []competition.Phase{}}}
	calendar := competition.NewService(competitionpostgres.NewRepository(q, tx, recorder))
	if err := calendar.Create(ctx, admin, period); err != nil {
		t.Fatal(err)
	}
	config := competition.Configuration{Version: 1, Approved: true, Published: true, FormEnabled: true, Sectors: []string{"afiliados"}, Phases: []competition.Phase{{Code: "postulacion", Title: "QA", OpensAt: start, ClosesAt: end, Published: true, Enabled: true}}}
	if _, err := calendar.Configure(ctx, admin, period.ID, config); err != nil {
		t.Fatal(err)
	}
	id, docID := uuid.New(), uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)
	sha := strings.Repeat("a", 64)
	raw := []byte(fmt.Sprintf(`{"seccion4":{"sector":"afiliados"},"seccion5":{"titulos":[{"id":"grado-1","documentoPdf":{"id":%q,"sha256":%q}}]},"seccion9":{"q1":"Sí","q2":"No","q3":"No","q4":"No","q5":"No","q6":"No","q7":"No","q8":"No","q9":"No","q10":"No","q11":"No","q12":"No","q13":"No"}}`, docID.String(), sha))
	canonical, err := submission.CanonicalContent(raw)
	if err != nil {
		t.Fatal(err)
	}
	hash := submission.HashContent(canonical)
	if _, err = pool.Exec(ctx, `INSERT INTO submissions(id,applicant_id,competition_id,code,status,content,created_at,updated_at) VALUES($1,$2,$3,'QA-SELLADO','draft',$4,$5,$5)`, id, owner.ID, period.ID, canonical, now); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO documents(id,submission_id,section,record_id,name,mime,size_bytes,sha256,storage_key) VALUES($1,$2,5,'grado-1','qa.pdf','application/pdf',10,$3,'qa-sintetico')`, docID, id, sha); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE submissions SET status='submitted',hash=$2,submitted_at=$3 WHERE id=$1`, id, hash, now); err != nil {
		t.Fatal(err)
	}
	config.Version = 2
	config.FormEnabled = false
	config.Phases = []competition.Phase{{Code: "meritos", Title: "QA méritos", OpensAt: start, ClosesAt: end, Published: true, Enabled: true}}
	if _, err = calendar.Configure(ctx, admin, period.ID, config); err != nil {
		t.Fatal(err)
	}
	repo := NewRepository(q, tx, recorder)
	service := merit.NewService(repo, recorder)
	max, threshold := int64(100), int64(50)
	def := merit.RuleDefinition{Criteria: []merit.Criterion{{Code: "grado", Label: "Sólo prueba", Metric: "education_count", UnitSize: 1, PointsPerUnit: 100, MaximumPoints: 100, Weight: 10000, Required: true}}, Requirements: []merit.Requirement{{Code: "base", Label: "Sólo prueba"}}, MaximumTotal: &max, Threshold: &threshold, Rounding: "inferior", ExperienceAggregation: "unir_intervalos", DayCounting: "dias_transcurridos", TieBreakers: []string{"grado", "resolucion_manual"}}
	for n := 1; n <= 13; n++ {
		answer := "No"
		if n == 1 {
			answer = "Sí"
		}
		def.Questions = append(def.Questions, merit.QuestionRule{ID: fmt.Sprintf("q%d", n), AcceptedAnswer: answer})
	}
	rules, err := service.CreateRules(ctx, analyst, period.ID, "afiliados", def)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.ApproveRules(ctx, analyst, rules.ID, "QA"); !errors.Is(err, merit.ErrForbidden) {
		t.Fatal("analista aprobó reglas", err)
	}
	input := merit.ReviewInput{RuleSetID: rules.ID, Motive: "Verificación QA", Documents: []merit.RecordReview{{Section: 5, RecordID: "grado-1", DocumentID: docID.String(), Verdict: "valido"}}, Requirements: []merit.RequirementReview{{Code: "base", Satisfied: true}}}
	for n := 1; n <= 13; n++ {
		input.Questions = append(input.Questions, merit.QuestionReview{ID: fmt.Sprintf("q%d", n), Confirmed: true})
	}
	if _, err = service.Review(ctx, analyst, id, input); !errors.Is(err, merit.ErrRulesPending) {
		t.Fatal("revisó con matriz sin aprobar", err)
	}
	rules, err = service.ApproveRules(ctx, admin, rules.ID, "Acta sintética QA, no institucional")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE merit_rule_sets SET definition='{}' WHERE id=$1`, rules.ID); err == nil {
		t.Fatal("matriz aprobada mutable")
	}
	failing := merit.NewService(NewRepository(q, tx, brokenAudit{}), recorder)
	if _, err = failing.Review(ctx, analyst, id, input); !errors.Is(err, audit.ErrUnavailable) {
		t.Fatal("fallo de auditoría ignorado", err)
	}
	var count int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM merit_reviews`).Scan(&count); err != nil || count != 0 {
		t.Fatal("revisión sin auditoría confirmada", count, err)
	}
	type answer struct {
		c   *merit.Case
		err error
	}
	answers := make(chan answer, 2)
	for range 2 {
		go func() { c, err := service.Review(ctx, analyst, id, input); answers <- answer{c, err} }()
	}
	var reviewed *merit.Case
	successes, conflicts := 0, 0
	for range 2 {
		a := <-answers
		if a.err == nil {
			successes++
			reviewed = a.c
		} else if errors.Is(a.err, merit.ErrConflict) {
			conflicts++
		} else {
			t.Fatal(a.err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatal("revisiones concurrentes", successes, conflicts)
	}
	if _, err = service.Review(ctx, analyst, id, input); !errors.Is(err, merit.ErrConflict) {
		t.Fatal("versión obsoleta aceptada", err)
	}
	calc, err := service.Calculate(ctx, analyst, id, merit.CalculateInput{ReviewID: reviewed.Review.ID, Motive: "Cálculo QA"})
	if err != nil || calc.Calculation.Evaluation.Total != 100 {
		t.Fatal("cálculo", err)
	}
	publication := merit.PublishInput{CalculationID: calc.Calculation.ID, Decision: "avanza", Motive: "Resolución QA"}
	published, err := service.Publish(ctx, admin, id, publication)
	if err != nil {
		t.Fatal("publicación", err)
	}
	oldID := published.Publication.ID
	if _, err = pool.Exec(ctx, `DELETE FROM merit_publications WHERE id=$1`, oldID); err == nil {
		t.Fatal("publicación eliminable")
	}
	loaded, err := service.Read(ctx, owner, id, false)
	if err != nil || loaded.Publication.ID != oldID || loaded.Snapshot.Hash != hash {
		t.Fatal("lectura propia", err)
	}
	if _, err = service.Read(ctx, analyst, id, false); !errors.Is(err, merit.ErrForbidden) {
		t.Fatal("lectura de otro titular", err)
	}
	input.Version = 1
	input.Motive = "Corrección QA"
	reviewed, err = service.Review(ctx, analyst, id, input)
	if err != nil {
		t.Fatal(err)
	}
	if reviewed.Publication.ID != oldID {
		t.Fatal("revisión reemplazó resolución")
	}
	calc, err = service.Calculate(ctx, analyst, id, merit.CalculateInput{Version: 1, ReviewID: reviewed.Review.ID, Motive: "Nuevo cálculo QA"})
	if err != nil {
		t.Fatal(err)
	}
	publication.CalculationID = calc.Calculation.ID
	if _, err = service.Publish(ctx, admin, id, publication); !errors.Is(err, merit.ErrConflict) {
		t.Fatal("reemplazó publicación sin referencia", err)
	}
	publication.Version = 1
	publication.SupersedesID = &oldID
	publication.Decision = "excluido"
	publication.Motive = "Corrección explícita QA"
	if _, err = service.Publish(ctx, admin, id, publication); err != nil {
		t.Fatal(err)
	}
	// Una nueva matriz invalida el cálculo pendiente, pero no altera la resolución ya publicada.
	next, err := service.CreateRules(ctx, analyst, period.ID, "afiliados", def)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.ApproveRules(ctx, admin, next.ID, "Versión QA 2"); err != nil {
		t.Fatal(err)
	}
	if _, err = service.Calculate(ctx, analyst, id, merit.CalculateInput{Version: 2, ReviewID: reviewed.Review.ID, Motive: "Reglas obsoletas"}); !errors.Is(err, merit.ErrConflict) {
		t.Fatal("calculó contra reglas anteriores", err)
	}
	// Cerrar méritos no depende de que la ventana de postulación esté habilitada.
	config.Version = 3
	config.Phases[0].Enabled = false
	if _, err = calendar.Configure(ctx, admin, period.ID, config); err != nil {
		t.Fatal(err)
	}
	input.Version = 2
	input.RuleSetID = next.ID
	if _, err = service.Review(ctx, analyst, id, input); !errors.Is(err, merit.ErrPhase) {
		t.Fatal("escribió con méritos cerrados", err)
	}
	if _, err = pool.Exec(ctx, `UPDATE applicants SET role='postulante' WHERE id=$1`, analyst.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = service.CreateRules(ctx, analyst, period.ID, "afiliados", def); !errors.Is(err, merit.ErrForbidden) {
		t.Fatal("rol revocado aceptado", err)
	}
	var stored []byte
	if err = pool.QueryRow(ctx, `SELECT content FROM submissions WHERE id=$1`, id).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	storedCanonical, _ := submission.CanonicalContent(stored)
	if string(storedCanonical) != string(canonical) {
		t.Fatal("méritos alteró snapshot")
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM workflow_events WHERE resource_id=$1 AND submission_id=$1 AND action='resultado.publicado'`, id).Scan(&count); err != nil || count != 2 {
		t.Fatal("publicaciones sin eventos", count, err)
	}
	var saved merit.Publication
	payload, err := q.GetMeritPublication(ctx, sqlcgen.GetMeritPublicationParams{ID: oldID, SubmissionID: id})
	if err != nil || json.Unmarshal(payload, &saved) != nil || saved.Decision != "avanza" {
		t.Fatal("perdió historia")
	}
}
