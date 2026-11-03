package postgres

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit"
	auditpostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit/postgres"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/competition"
	competitionpostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/competition/postgres"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/evaluation"
	evaluationnotification "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/evaluation/notification"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/merit"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/db"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/mail"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/shared/domain"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/testutil"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

type objectStore struct {
	mu      sync.Mutex
	objects map[string][]byte
}

func (s *objectStore) Save(_ context.Context, key string, r io.Reader) error {
	raw, err := io.ReadAll(r)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.objects[key] = raw
	return err
}
func (s *objectStore) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.objects, key)
	return nil
}
func (s *objectStore) PresignGet(context.Context, string, string, time.Duration) (string, error) {
	return "https://objects.example.test/firma-temporal", nil
}

type brokenNotifier struct{}

func (brokenNotifier) Enqueue(context.Context, uuid.UUID, uuid.UUID, string) error {
	return errors.New("cola QA no disponible")
}

type brokenAudit struct{}

func (brokenAudit) RecordWorkflow(context.Context, audit.Event) error { return audit.ErrUnavailable }

type fixture struct {
	pool                   *pgxpool.Pool
	q                      *sqlcgen.Queries
	tx                     *db.TxManager
	repo                   *Repository
	s                      *evaluation.Service
	admin, analyst, owner  *identity.Applicant
	period, id, documentID uuid.UUID
	hash                   string
	calendar               *competition.Service
	config                 competition.Configuration
	recorder               *auditpostgres.Repository
	objects                *objectStore
	merit                  merit.Publication
}

func makeFixture(t *testing.T, quota int64) *fixture {
	t.Helper()
	ctx := context.Background()
	pool := testutil.Postgres(t, "EVALUATION_TEST_DATABASE_URL")
	q := sqlcgen.New(pool)
	tx := db.NewTxManager(pool)
	recorder := auditpostgres.NewRepository(q)
	f := &fixture{pool: pool, q: q, tx: tx, admin: &identity.Applicant{ID: uuid.New(), Role: identity.RoleAdministrator}, analyst: &identity.Applicant{ID: uuid.New(), Role: identity.RoleAnalyst}, owner: &identity.Applicant{ID: uuid.New(), Role: identity.RoleApplicant}, period: uuid.New(), id: uuid.New(), documentID: uuid.New(), recorder: recorder, objects: &objectStore{objects: map[string][]byte{}}}
	for i, a := range []*identity.Applicant{f.admin, f.analyst, f.owner} {
		if _, err := pool.Exec(ctx, `INSERT INTO applicants(id,cedula,nombres,apellidos,correo,role) VALUES($1,$2,'Persona','QA',$3,$4)`, a.ID, fmt.Sprintf("000000000%d", i+1), a.ID.String()+"@example.test", a.Role); err != nil {
			t.Fatal(err)
		}
	}
	start, end := time.Now().UTC().Add(-2*time.Hour), time.Now().UTC().Add(48*time.Hour)
	f.calendar = competition.NewService(competitionpostgres.NewRepository(q, tx, recorder))
	period := &competition.Competition{ID: f.period, Code: "T21-T22-QA-NO-OFICIAL", Title: "Período sintético", OpensAt: start, ClosesAt: end, Configuration: competition.Configuration{Sectors: []string{"afiliados"}, Phases: []competition.Phase{}}}
	if err := f.calendar.Create(ctx, f.admin, period); err != nil {
		t.Fatal(err)
	}
	f.config = competition.Configuration{Version: 1, Approved: true, Published: true, FormEnabled: true, Sectors: []string{"afiliados"}, Phases: []competition.Phase{}}
	for _, code := range []string{"postulacion", "meritos", "oposicion_teorica", "oposicion_psicotecnica", "reconsideracion", "resultados"} {
		f.config.Phases = append(f.config.Phases, competition.Phase{Code: code, Title: "Fase QA " + code, OpensAt: start, ClosesAt: end, Published: true, Enabled: true})
	}
	current, err := f.calendar.Configure(ctx, f.admin, f.period, f.config)
	if err != nil {
		t.Fatal(err)
	}
	f.config = current.Configuration
	raw, _ := submission.CanonicalContent([]byte(`{"seccion4":{"sector":"afiliados"}}`))
	f.hash = submission.HashContent(raw)
	now := time.Now().UTC().Add(-time.Hour)
	if _, err = pool.Exec(ctx, `INSERT INTO submissions(id,applicant_id,competition_id,code,status,content,created_at,updated_at) VALUES($1,$2,$3,'QA-EXPEDIENTE','draft',$4,$5,$5)`, f.id, f.owner.ID, f.period, raw, now); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO documents(id,submission_id,section,record_id,name,mime,size_bytes,sha256,storage_key) VALUES($1,$2,5,'registro-qa','original.pdf','application/pdf',$3,repeat('a',64),'original-qa')`, f.documentID, f.id, min(max(quota, 1), 2<<20)); err != nil {
		t.Fatal(err)
	}
	for remaining, i := quota-(2<<20), 1; remaining > 0; i++ {
		size := min(remaining, int64(2<<20))
		if _, err = pool.Exec(ctx, `INSERT INTO documents(id,submission_id,section,record_id,name,mime,size_bytes,sha256,storage_key) VALUES($1,$2,5,$3,'original.pdf','application/pdf',$4,repeat('a',64),$5)`, uuid.New(), f.id, fmt.Sprintf("original-%d", i), size, fmt.Sprintf("objeto-%d", i)); err != nil {
			t.Fatal(err)
		}
		remaining -= size
	}
	if _, err = pool.Exec(ctx, `UPDATE submissions SET status='submitted',hash=$2,submitted_at=$3 WHERE id=$1`, f.id, f.hash, now); err != nil {
		t.Fatal(err)
	}
	rule, review, calc := uuid.New(), uuid.New(), uuid.New()
	f.merit = merit.Publication{ID: uuid.New(), SubmissionID: f.id, Version: 1, CalculationID: calc, Decision: "avanza", Motive: "Fuente T20 sintética", SnapshotHash: f.hash, RuleSetID: rule, RulesHash: strings.Repeat("b", 64), ActorID: f.admin.ID, CreatedAt: time.Now().UTC().Add(-5 * time.Minute), Evaluation: merit.Evaluation{Total: 80, Maximum: 100, CanAdvance: true}}
	payload, _ := json.Marshal(f.merit)
	// Fixture de un productor T20 ya publicado. Su motor se prueba en su contexto, no se simula un puntaje HTTP.
	statements := []struct {
		sql  string
		args []interface{}
	}{{`INSERT INTO merit_rule_sets(id,competition_id,sector,version,definition,definition_hash,actor_id,created_at) VALUES($1,$2,'afiliados',1,'{}',repeat('b',64),$3,now())`, []interface{}{rule, f.period, f.admin.ID}}, {`INSERT INTO merit_reviews(id,submission_id,version,rule_set_id,payload) VALUES($1,$2,1,$3,'{}')`, []interface{}{review, f.id, rule}}, {`INSERT INTO merit_calculations(id,submission_id,version,review_id,rule_set_id,payload) VALUES($1,$2,1,$3,$4,'{}')`, []interface{}{calc, f.id, review, rule}}, {`INSERT INTO merit_publications(id,submission_id,version,calculation_id,decision,payload) VALUES($1,$2,1,$3,'avanza',$4)`, []interface{}{f.merit.ID, f.id, calc, payload}}, {`INSERT INTO merit_case_streams(submission_id,review_id,calculation_id,publication_id) VALUES($1,$2,$3,$4)`, []interface{}{f.id, review, calc, f.merit.ID}}}
	for _, s := range statements {
		if _, err = pool.Exec(ctx, s.sql, s.args...); err != nil {
			t.Fatal(err)
		}
	}
	outbox, err := mail.NewOutbox(q, base64.StdEncoding.EncodeToString(make([]byte, 32)))
	if err != nil {
		t.Fatal(err)
	}
	f.repo = NewRepository(q, tx, recorder, &evaluationnotification.Outbox{Queue: outbox, Queries: q})
	f.s = evaluation.NewService(f.repo, f.objects, recorder)
	return f
}
func (f *fixture) approve(t *testing.T, kind string) *evaluation.Policy {
	t.Helper()
	maximum := int64(100)
	definition := evaluation.Definition{}
	if kind == "oposicion" {
		definition.Opposition = &evaluation.OppositionRules{Instrument: "QA NO OFICIAL", InstrumentVersion: "QA-1", PsychometricMode: "puntaje", PsychometricMaximum: &maximum, Conclusions: []string{"acredita", "no_acredita"}, AcceptedConclusions: []string{"acredita"}, TheoryMaximum: 100, MeritWeight: 4000, TheoryWeight: 4000, PsychometricWeight: 2000, TotalMaximum: 10000, Threshold: 6000, Rounding: "inferior"}
	} else {
		term := evaluation.Term{Quantity: 240, Unit: "minutos_transcurridos"}
		definition.Reconsideration = &evaluation.AppealRules{Applicant: term, Talent: term, Commission: term, Calendar: evaluation.Calendar{TimeZone: "America/Guayaquil", EndPolicy: "exigir_termino_completo"}}
	}
	p, err := f.s.CreatePolicy(context.Background(), f.analyst, f.period, "afiliados", kind, definition)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.s.ApprovePolicy(context.Background(), f.analyst, p.ID, "QA"); !errors.Is(err, evaluation.ErrForbidden) {
		t.Fatal("analista aprobó reglas", err)
	}
	p, err = f.s.ApprovePolicy(context.Background(), f.admin, p.ID, "Acta sintética QA, no institucional")
	if err != nil {
		t.Fatal(err)
	}
	return p
}
func pdf() evaluation.EvidenceInput {
	return evaluation.EvidenceInput{Name: "informe-qa.pdf", PDF: []byte("%PDF-1.4\ncontenido exclusivamente de prueba\n%%EOF\n")}
}
func TestPostgresOppositionConsolidationAndReconsideration(t *testing.T) {
	f := makeFixture(t, 1)
	ctx := context.Background()
	op := f.approve(t, "oposicion")
	appealPolicy := f.approve(t, "reconsideracion")
	_ = appealPolicy
	if _, err := f.pool.Exec(ctx, `UPDATE evaluation_policies SET definition='{}' WHERE id=$1`, op.ID); err == nil {
		t.Fatal("política aprobada mutable")
	}
	score := int64(80)
	input := evaluation.PsychometricInput{PolicyID: op.ID, EvaluatedAt: time.Now().UTC().Add(-time.Minute), Score: &score, Conclusion: "acredita", Motive: "Registrar informe QA", Evidence: pdf()}
	key := uuid.New()
	first, err := f.s.RecordPsychometric(ctx, f.analyst, f.id, key, input)
	if err != nil {
		t.Fatal(err)
	}
	var psych evaluation.Psychometric
	if json.Unmarshal(first.Data, &psych) != nil {
		t.Fatal("recibo ilegible")
	}
	again, err := f.s.RecordPsychometric(ctx, f.analyst, f.id, key, input)
	if err != nil || again.Applied || string(again.Data) != string(first.Data) {
		t.Fatal("duplicó informe", err)
	}
	changed := input
	changed.Motive = "Otro contenido"
	if _, err = f.s.RecordPsychometric(ctx, f.analyst, f.id, key, changed); !errors.Is(err, evaluation.ErrConflict) {
		t.Fatal("clave reutilizada", err)
	}
	own, err := f.s.Read(ctx, f.owner, f.id, uuid.Nil, false)
	if err != nil || own.State.Psychometric != nil {
		t.Fatal("informe provisional expuesto", err)
	}
	if _, err = f.s.Download(ctx, f.owner, f.id, psych.Evidence.ID, false); !errors.Is(err, evaluation.ErrForbidden) {
		t.Fatal("PDF provisional expuesto", err)
	}
	pub, err := f.s.PublishPsychometric(ctx, f.admin, f.id, uuid.New(), evaluation.PublishInput{SourceID: psych.ID, PolicyID: op.ID, Motive: "Publicación QA"})
	if err != nil {
		t.Fatal(err)
	}
	var psychoResult evaluation.Result
	_ = json.Unmarshal(pub.Data, &psychoResult)
	if _, err = f.s.Download(ctx, f.owner, f.id, psych.Evidence.ID, false); err != nil {
		t.Fatal("PDF publicado no disponible", err)
	}
	expected := map[string]uuid.UUID{"meritos": f.merit.ID, "oposicion_teorica": uuid.New(), "oposicion_psicotecnica": psychoResult.ID}
	consolidate := evaluation.PublishInput{PolicyID: op.ID, Motive: "Consolidación QA", ExpectedSources: expected}
	if _, err = f.s.Consolidate(ctx, f.admin, f.id, uuid.New(), consolidate); !errors.Is(err, evaluation.ErrPending) {
		t.Fatal("fabricó nota teórica", err)
	}
	theory := evaluation.Result{ID: uuid.New(), SubmissionID: f.id, Phase: "oposicion_teorica", Version: 1, SourceID: uuid.New(), PolicyID: uuid.New(), PolicyHash: strings.Repeat("d", 64), SnapshotHash: f.hash, Score: &score, Maximum: op.Definition.Opposition.PsychometricMaximum, CanAdvance: true, Motive: "Resultado sintético del productor T27", PublishedAt: time.Now().UTC().Add(-time.Second)}
	if err = f.repo.RecordFinalTheory(ctx, theory); !errors.Is(err, audit.ErrTransactionRequired) {
		t.Fatal("nota teórica fuera del commit", err)
	}
	if err = f.tx.WithinTx(ctx, func(tx pgx.Tx) error { return f.repo.RecordFinalTheory(db.WithTransaction(ctx, tx), theory) }); err != nil {
		t.Fatal(err)
	}
	if err = f.tx.WithinTx(ctx, func(tx pgx.Tx) error { return f.repo.RecordFinalTheory(db.WithTransaction(ctx, tx), theory) }); err != nil {
		t.Fatal("reintento definitivo T27 no idempotente", err)
	}
	altered := theory
	altered.Motive = "Intento de reutilizar publicación con otro contenido"
	if err = f.tx.WithinTx(ctx, func(tx pgx.Tx) error { return f.repo.RecordFinalTheory(db.WithTransaction(ctx, tx), altered) }); !errors.Is(err, evaluation.ErrConflict) {
		t.Fatal("publicación T27 sobreescrita", err)
	}
	expected["oposicion_teorica"] = theory.ID
	consolidated, err := f.s.Consolidate(ctx, f.admin, f.id, uuid.New(), consolidate)
	if err != nil {
		t.Fatal(err)
	}
	var result evaluation.Result
	_ = json.Unmarshal(consolidated.Data, &result)
	if result.Score == nil || *result.Score != 8000 || len(result.Breakdown) != 3 || !result.CanAdvance {
		t.Fatal("consolidación incorrecta", result)
	}
	if _, err = f.pool.Exec(ctx, `DELETE FROM evaluation_records WHERE id=$1`, result.ID); err == nil {
		t.Fatal("resultado eliminable")
	}
	filing := evaluation.FileAppealInput{Phase: "resultados", ResultID: result.ID, Motive: "Motivo de reconsideración QA", DocumentIDs: []uuid.UUID{f.documentID}, Evidence: func() *evaluation.EvidenceInput { e := pdf(); return &e }()}
	fileKey := uuid.New()
	var receipts [2]evaluation.Receipt
	var errs [2]error
	var wg sync.WaitGroup
	for n := range 2 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			receipts[i], errs[i] = f.s.FileAppeal(ctx, f.owner, f.id, fileKey, filing)
		}(n)
	}
	wg.Wait()
	if errs[0] != nil || errs[1] != nil || receipts[0].Applied == receipts[1].Applied {
		t.Fatal("reintentos concurrentes", errs)
	}
	var appeal evaluation.Appeal
	_ = json.Unmarshal(receipts[0].Data, &appeal)
	if _, err = f.s.FileAppeal(ctx, f.owner, f.id, uuid.New(), filing); !errors.Is(err, evaluation.ErrConflict) {
		t.Fatal("duplicó solicitud con otra clave", err)
	}
	if _, err = f.s.Attend(ctx, f.admin, f.id, appeal.ID, uuid.New(), evaluation.AttendInput{PreviousVersion: 1, Observation: "QA", Recommendation: "confirmar"}); !errors.Is(err, evaluation.ErrForbidden) {
		t.Fatal("Comisión suplantó TH", err)
	}
	attention := evaluation.AttendInput{PreviousVersion: 1, Observation: "Atención QA", Recommendation: "confirmar"}
	attended, err := f.s.Attend(ctx, f.analyst, f.id, appeal.ID, uuid.New(), attention)
	if err != nil {
		t.Fatal(err)
	}
	_ = json.Unmarshal(attended.Data, &appeal)
	if appeal.CommissionDeadline == nil || appeal.Attention.ActorID != f.analyst.ID {
		t.Fatal("faltan plazos o responsable")
	}
	failing := evaluation.NewService(NewRepository(f.q, f.tx, f.recorder, brokenNotifier{}), f.objects, f.recorder)
	resolve := evaluation.ResolveInput{PreviousVersion: 2, Decision: "confirmar", Motive: "Resolución QA"}
	if _, err = failing.Resolve(ctx, f.admin, f.id, appeal.ID, uuid.New(), resolve); !errors.Is(err, evaluation.ErrUnavailable) {
		t.Fatal("decisión sin correo durable", err)
	}
	loaded, err := f.repo.ReadCase(ctx, f.id, appeal.ID)
	if err != nil || loaded.Appeal.Status != "atendida" {
		t.Fatal("resolución sobrevivió rollback", err)
	}
	decision, err := f.s.Resolve(ctx, f.admin, f.id, appeal.ID, uuid.New(), resolve)
	if err != nil {
		t.Fatal(err)
	}
	_ = json.Unmarshal(decision.Data, &appeal)
	if appeal.Status != "resuelta" || appeal.Original.ID != result.ID || appeal.Resolution.EffectiveResult.ID != result.ID {
		t.Fatal("se perdió original/efecto")
	}
	var count int
	if err = f.pool.QueryRow(ctx, `SELECT count(*) FROM mail_outbox`).Scan(&count); err != nil || count != 4 {
		t.Fatal("correos duplicados o faltantes", count, err)
	}
	if err = f.pool.QueryRow(ctx, `SELECT count(*) FROM evaluation_upload_intents`).Scan(&count); err != nil || count != 0 {
		t.Fatal("intentos confirmados retenidos", count, err)
	}
	if len(f.objects.objects) != 2 {
		t.Fatal("objetos huérfanos por reintento", len(f.objects.objects))
	}
	history, err := f.repo.ListHistory(ctx, f.id, domain.Position{}, 100)
	if err != nil || len(history) != 7 {
		t.Fatal("historial incompleto", len(history), err)
	}
	if len(history[0].Data) == 0 {
		t.Fatal("falta detalle histórico para reconstruir resoluciones")
	}
	// Simula una caída entre el guardado del objeto y el commit de su metadata.
	orphan := &evaluation.Evidence{ID: uuid.New(), SubmissionID: f.id, StorageKey: "evaluation/qa-huerfano.pdf"}
	if err = f.repo.BeginUpload(ctx, orphan); err != nil {
		t.Fatal(err)
	}
	if err = f.objects.Save(ctx, orphan.StorageKey, strings.NewReader("objeto sintético")); err != nil {
		t.Fatal(err)
	}
	if _, err = f.pool.Exec(ctx, `UPDATE evaluation_upload_intents SET expires_at=now()-interval '1 second' WHERE id=$1`, orphan.ID); err != nil {
		t.Fatal(err)
	}
	if err = f.s.CleanupUploads(ctx); err != nil || len(f.objects.objects) != 2 {
		t.Fatal("limpieza dañó evidencia confirmada o conservó huérfano", err)
	}
	if err = f.pool.QueryRow(ctx, `SELECT count(*) FROM evaluation_upload_intents`).Scan(&count); err != nil || count != 0 {
		t.Fatal("intento vencido sin retirar", count, err)
	}
	// Un replay confirmado sobrevive al cierre, pero una actuación nueva no lo elude.
	for i := range f.config.Phases {
		if f.config.Phases[i].Code == "reconsideracion" {
			f.config.Phases[i].Enabled = false
		}
	}
	if _, err = f.calendar.Configure(ctx, f.admin, f.period, f.config); err != nil {
		t.Fatal(err)
	}
	replay, err := f.s.FileAppeal(ctx, f.owner, f.id, fileKey, filing)
	if err != nil || replay.Applied {
		t.Fatal("reintento tras cierre", err)
	}
	if _, err = f.s.FileAppeal(ctx, f.owner, f.id, uuid.New(), evaluation.FileAppealInput{Phase: "meritos", ResultID: f.merit.ID, Motive: "QA"}); !errors.Is(err, evaluation.ErrPhase) {
		t.Fatal("evadió cierre", err)
	}
}
func TestPostgresEvaluationQuotaAndAuditRollback(t *testing.T) {
	f := makeFixture(t, 50<<20)
	ctx := context.Background()
	f.approve(t, "reconsideracion")
	e := pdf()
	input := evaluation.FileAppealInput{Phase: "meritos", ResultID: f.merit.ID, Motive: "QA", Evidence: &e}
	if _, err := f.s.FileAppeal(ctx, f.owner, f.id, uuid.New(), input); !errors.Is(err, evaluation.ErrQuota) {
		t.Fatal("no comparte cuota T4", err)
	}
	if len(f.objects.objects) != 0 {
		t.Fatal("objeto sin metadata tras cuota")
	}
	input.Evidence = nil
	failing := evaluation.NewService(NewRepository(f.q, f.tx, brokenAudit{}, brokenNotifier{}), f.objects, f.recorder)
	if _, err := failing.FileAppeal(ctx, f.owner, f.id, uuid.New(), input); !errors.Is(err, audit.ErrUnavailable) {
		t.Fatal("auditoría omitida", err)
	}
	var count int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM reconsiderations`).Scan(&count); err != nil || count != 0 {
		t.Fatal("solicitud sin auditoría", count, err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE applicants SET role='postulante' WHERE id=$1`, f.analyst.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.CreatePolicy(ctx, f.analyst, f.period, "afiliados", "oposicion", evaluation.Definition{}); !errors.Is(err, evaluation.ErrForbidden) {
		t.Fatal("aceptó rol revocado", err)
	}
}
