package postgres

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit"
	auditpostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit/postgres"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/competition"
	competitionpostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/competition/postgres"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/evaluation"
	evaluationpostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/evaluation/postgres"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/exam"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/exam/report"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/merit"
	noticepostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/notice/postgres"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/db"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/mail"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/privacy"
	privacypostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/privacy/postgres"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/testutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type fixture struct {
	pool                  *pgxpool.Pool
	q                     *sqlcgen.Queries
	tx                    *db.TxManager
	repo                  *Repository
	privacy               *privacypostgres.Repository
	owner, admin, analyst *identity.Applicant
	sub, period           uuid.UUID
	merit                 merit.Publication
	calendar              *competition.Service
	config                competition.Configuration
	bank                  *exam.Bank
	examConfig            *exam.Config
	auditor               *auditpostgres.Repository
	queue                 *mail.Outbox
	snapshot              *submission.Submission
	policy                *privacy.Policy
}

func newFixture(t *testing.T, duration int) *fixture {
	t.Helper()
	ctx := context.Background()
	pool := testutil.Postgres(t, "EXAM_TEST_DATABASE_URL")
	q := sqlcgen.New(pool)
	tx := db.NewTxManager(pool)
	auditor := auditpostgres.NewRepository(q)
	f := &fixture{pool: pool, q: q, tx: tx, auditor: auditor, sub: uuid.New(), period: uuid.New(), owner: &identity.Applicant{ID: uuid.New(), Role: identity.RoleApplicant}, admin: &identity.Applicant{ID: uuid.New(), Role: identity.RoleAdministrator}, analyst: &identity.Applicant{ID: uuid.New(), Role: identity.RoleAnalyst}}
	for i, a := range []*identity.Applicant{f.owner, f.admin, f.analyst} {
		if _, e := pool.Exec(ctx, `INSERT INTO applicants(id,cedula,nombres,apellidos,correo,role) VALUES($1,$2,'Persona','QA',$3,$4)`, a.ID, fmt.Sprintf("000000000%d", i+1), a.ID.String()+"@example.test", a.Role); e != nil {
			t.Fatal(e)
		}
	}
	start, end := time.Now().UTC().Add(-3*time.Hour), time.Now().UTC().Add(3*time.Hour)
	f.calendar = competition.NewService(competitionpostgres.NewRepository(q, tx, auditor))
	period := &competition.Competition{ID: f.period, Code: "EXAM-QA-NO-OFICIAL", Title: "QA", OpensAt: start, ClosesAt: end, Configuration: competition.Configuration{Sectors: []string{"afiliados"}, Phases: []competition.Phase{}}}
	if e := f.calendar.Create(ctx, f.admin, period); e != nil {
		t.Fatal(e)
	}
	f.config = competition.Configuration{Version: 1, Approved: true, Published: true, FormEnabled: true, Sectors: []string{"afiliados"}, Phases: []competition.Phase{}}
	for _, code := range []string{"postulacion", "meritos", "oposicion_teorica", "oposicion_psicotecnica", "reconsideracion", "resultados"} {
		f.config.Phases = append(f.config.Phases, competition.Phase{Code: code, Title: "QA", OpensAt: start, ClosesAt: end, Published: true, Enabled: true})
	}
	current, e := f.calendar.Configure(ctx, f.admin, f.period, f.config)
	if e != nil {
		t.Fatal(e)
	}
	f.config = current.Configuration
	f.privacy = privacypostgres.NewRepository(q, tx, auditor)
	rules := privacy.Rules{Text: "QA NO ES TEXTO LEGAL", Treatment: "Tratamiento QA", Reference: "Acta QA", AccessDays: 30, DocumentDays: 10, ReportDays: 20, EmailDays: 10, PublicDays: 5, ReviewDays: 1, Historical: "revision_humana_sin_borrado_automatico", AllowExports: true, AllowEmail: true, AllowNames: true}
	f.policy, e = f.privacy.Publish(ctx, f.admin, f.period, privacy.PublishInput{Rules: rules})
	if e != nil {
		t.Fatal(e)
	}
	definition, _ := submission.FormDefinition()
	consent := definition.Consent
	consent.NamePublication = true
	consent.DataProcessing = true
	consent.Truthfulness = true
	consent.CompetitionRules = true
	consent.PhysicalDelivery = true
	consent.AcceptedAt = time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339)
	consent.PrivacyPolicyID = f.policy.ID.String()
	consent.PrivacyPolicyHash = f.policy.Hash
	content, _ := json.Marshal(submission.Form{Personal: &submission.PersonalSection{FirstName: "Persona", LastName: "QA"}, Position: &submission.PositionSection{Sector: "afiliados"}, Consent: consent})
	content, _ = submission.CanonicalContent(content)
	sent := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Microsecond)
	f.snapshot = &submission.Submission{ID: f.sub, ApplicantID: f.owner.ID, CompetitionID: &f.period, Code: "QA-EXAMEN", Status: submission.StatusSubmitted, Content: content, Hash: submission.HashContent(content), SubmittedAt: &sent}
	if _, e = pool.Exec(ctx, `INSERT INTO submissions(id,applicant_id,competition_id,code,status,content,created_at,updated_at) VALUES($1,$2,$3,'QA-EXAMEN','draft',$4,$5,$5)`, f.sub, f.owner.ID, f.period, content, sent); e != nil {
		t.Fatal(e)
	}
	if _, e = pool.Exec(ctx, `UPDATE submissions SET status='submitted',hash=$2,submitted_at=$3 WHERE id=$1`, f.sub, f.snapshot.Hash, sent); e != nil {
		t.Fatal(e)
	}
	if e = tx.WithinTx(ctx, func(tx pgx.Tx) error { return f.privacy.Bind(db.WithTransaction(ctx, tx), f.snapshot, consent) }); e != nil {
		t.Fatal(e)
	}
	rule, review, calc := uuid.New(), uuid.New(), uuid.New()
	f.merit = merit.Publication{ID: uuid.New(), SubmissionID: f.sub, Version: 1, CalculationID: calc, RuleSetID: rule, Decision: "avanza", SnapshotHash: f.snapshot.Hash, RulesHash: strings.Repeat("a", 64), CreatedAt: sent, Evaluation: merit.Evaluation{Total: 80, Maximum: 100, CanAdvance: true}}
	raw, _ := json.Marshal(f.merit)
	for _, st := range []struct {
		s string
		a []interface{}
	}{
		{`INSERT INTO merit_rule_sets(id,competition_id,sector,version,definition,definition_hash,actor_id,created_at) VALUES($1,$2,'afiliados',1,'{}',repeat('a',64),$3,now())`, []interface{}{rule, f.period, f.admin.ID}},
		{`INSERT INTO merit_reviews(id,submission_id,version,rule_set_id,payload) VALUES($1,$2,1,$3,'{}')`, []interface{}{review, f.sub, rule}},
		{`INSERT INTO merit_calculations(id,submission_id,version,review_id,rule_set_id,payload) VALUES($1,$2,1,$3,$4,'{}')`, []interface{}{calc, f.sub, review, rule}},
		{`INSERT INTO merit_publications(id,submission_id,version,calculation_id,decision,payload) VALUES($1,$2,1,$3,'avanza',$4)`, []interface{}{f.merit.ID, f.sub, calc, raw}},
		{`INSERT INTO merit_case_streams(submission_id,review_id,calculation_id,publication_id) VALUES($1,$2,$3,$4)`, []interface{}{f.sub, review, calc, f.merit.ID}},
	} {
		if _, e = pool.Exec(ctx, st.s, st.a...); e != nil {
			t.Fatal(e)
		}
	}
	f.queue, e = mail.NewOutbox(q, base64.StdEncoding.EncodeToString(make([]byte, 32)))
	if e != nil {
		t.Fatal(e)
	}
	notifier := &noticepostgres.Queue{Outbox: f.queue, Queries: q}
	f.repo = NewRepository(q, tx, auditor, evaluationpostgres.NewRepository(q, tx, auditor, notifier), report.NewRenderer(), f.privacy)
	input := exam.BankInput{Reference: "Banco QA NO OFICIAL", Questions: []exam.Question{{ID: "q1", Text: "¿Uno más uno?", Correct: "a", Options: []exam.Option{{ID: "a", Text: "Dos"}, {ID: "b", Text: "Tres"}}}, {ID: "q2", Text: "¿Dos más dos?", Correct: "b", Options: []exam.Option{{ID: "a", Text: "Tres"}, {ID: "b", Text: "Cuatro"}}}}}
	f.bank, e = f.repo.CreateBank(ctx, f.admin, f.period, "afiliados", input)
	if e != nil {
		t.Fatal(e)
	}
	f.examConfig, e = f.repo.Configure(ctx, f.admin, f.bank.ID, exam.ConfigRules{Count: 2, DurationSeconds: duration, Maximum: 100, Threshold: 50, Rounding: "inferior", AttemptPolicy: "un_intento", UniquePolicy: "orden_preguntas_unico_rechazar_agotamiento", Reference: "Reglas QA", OpensAt: time.Now().UTC().Add(-time.Hour), ClosesAt: end})
	if e != nil {
		t.Fatal(e)
	}
	return f
}

type brokenFinal struct{}

func (brokenFinal) RecordFinalTheory(context.Context, evaluation.Result) error {
	return errors.New("cola caída QA")
}
func TestExamLifecycleFreezeAndAtomicCompletion(t *testing.T) {
	f := newFixture(t, 60)
	ctx := context.Background()
	if _, _, e := f.repo.Start(ctx, f.owner, f.sub); !errors.Is(e, exam.ErrNotFound) {
		t.Fatal("inició sin habilitación", e)
	}
	if e := f.repo.Enroll(ctx, f.analyst, f.sub, exam.EnrollmentInput{ConfigID: f.examConfig.ID, MeritID: f.merit.ID}); !errors.Is(e, exam.ErrForbidden) {
		t.Fatal("analista habilitó", e)
	}
	input := exam.EnrollmentInput{ConfigID: f.examConfig.ID, MeritID: f.merit.ID}
	if e := f.repo.Enroll(ctx, f.admin, f.sub, input); e != nil {
		t.Fatal(e)
	}
	if e := f.repo.Enroll(ctx, f.admin, f.sub, input); e != nil {
		t.Fatal("replay habilitación", e)
	}
	var attempts [2]*exam.Attempt
	var errs [2]error
	var wg sync.WaitGroup
	for i := range 2 {
		wg.Add(1)
		go func(i int) { defer wg.Done(); attempts[i], _, errs[i] = f.repo.Start(ctx, f.owner, f.sub) }(i)
	}
	wg.Wait()
	if errs[0] != nil || errs[1] != nil || attempts[0].ID != attempts[1].ID || !attempts[0].ExpiresAt.Equal(attempts[1].ExpiresAt) {
		t.Fatal("duplicó o extendió intento", errs)
	}
	a := attempts[0]
	if _, e := f.pool.Exec(ctx, `UPDATE exam_attempts SET version=version+1,payload=jsonb_set(jsonb_set(payload,'{matriz,0,correcta}','"fake"'),'{version}',to_jsonb(version+1)) WHERE submission_id=$1`, f.sub); e == nil {
		t.Fatal("matriz congelada mutable")
	}
	changed := f.bank.Input
	changed.PreviousVersion = 1
	changed.Questions[0].Correct = "b"
	if _, e := f.repo.CreateBank(ctx, f.admin, f.period, "afiliados", changed); e != nil {
		t.Fatal(e)
	}
	stored, _, e := f.repo.Read(ctx, f.owner, f.sub, false)
	if e != nil || stored.BankHash != a.BankHash {
		t.Fatal("cambio de banco alteró intento", e)
	}
	answer := exam.AnswerInput{PreviousVersion: 1, QuestionID: a.Questions[0].ID, OptionID: a.Questions[0].Correct}
	a, _, e = f.repo.Answer(ctx, f.owner, f.sub, answer)
	if e != nil {
		t.Fatal(e)
	}
	replay, _, e := f.repo.Answer(ctx, f.owner, f.sub, answer)
	if e != nil || replay.Version != a.Version {
		t.Fatal("reintento de guardado duplicó versión", e)
	}
	if _, _, e = f.repo.Answer(ctx, f.owner, f.sub, exam.AnswerInput{PreviousVersion: 1, QuestionID: a.Questions[1].ID, OptionID: a.Questions[1].Correct}); !errors.Is(e, exam.ErrConflict) {
		t.Fatal("perdió control de versión", e)
	}
	broken := NewRepository(f.q, f.tx, f.auditor, brokenFinal{}, report.NewRenderer(), f.privacy)
	if _, _, e = broken.Finish(ctx, f.owner, f.sub, false); e == nil {
		t.Fatal("cerró sin resultado transaccional")
	}
	var count int
	if e = f.pool.QueryRow(ctx, `SELECT count(*) FROM exam_reports`).Scan(&count); e != nil || count != 0 {
		t.Fatal("PDF sobrevivió al rollback", e)
	}
	for i := range 2 {
		wg.Add(1)
		go func(i int) { defer wg.Done(); attempts[i], _, errs[i] = f.repo.Finish(ctx, f.owner, f.sub, false) }(i)
	}
	wg.Wait()
	if errs[0] != nil || errs[1] != nil || attempts[0].Result.ID != attempts[1].Result.ID || *attempts[0].Result.Score != 50 {
		t.Fatal("duplicó calificación", errs)
	}
	if e = f.pool.QueryRow(ctx, `SELECT count(*) FROM evaluation_records WHERE action='teorica.finalizada'`).Scan(&count); e != nil || count != 1 {
		t.Fatal("notas duplicadas", count, e)
	}
	if e = f.pool.QueryRow(ctx, `SELECT count(*) FROM mail_outbox`).Scan(&count); e != nil || count != 1 {
		t.Fatal("correo duplicado", e)
	}
	reportData, e := f.repo.Report(ctx, f.owner, f.sub, false)
	if e != nil || !bytes.HasPrefix(reportData.PDF, []byte("%PDF-")) {
		t.Fatal("reporte ausente", e)
	}
	rerender, e := report.NewRenderer().Render(ctx, attempts[0])
	if e != nil || !bytes.Equal(rerender, reportData.PDF) {
		t.Fatal("reporte no reproducible", e)
	}
	capture := &captureSender{}
	if e = f.queue.Dispatch(ctx, capture, nil); e != nil {
		t.Fatal(e)
	}
	if len(capture.messages) != 1 || len(capture.messages[0].Attachments) != 1 || !bytes.Equal(capture.messages[0].Attachments[0].Data, reportData.PDF) {
		t.Fatal("PDF no adjuntado al correo definitivo")
	}
	if _, e = f.pool.Exec(ctx, `DELETE FROM exam_reports`); e == nil {
		t.Fatal("reporte eliminable")
	}
	if _, e = f.pool.Exec(ctx, `UPDATE exam_attempts SET version=version+1 WHERE submission_id=$1`, f.sub); e == nil {
		t.Fatal("intento final mutable")
	}
	if _, _, e = f.repo.Answer(ctx, f.owner, f.sub, answer); !errors.Is(e, exam.ErrExpired) {
		t.Fatal("aceptó respuesta tras cierre", e)
	}
	stranger := &identity.Applicant{ID: uuid.New(), Role: identity.RoleApplicant}
	if _, _, e = f.repo.Read(ctx, stranger, f.sub, false); !errors.Is(e, exam.ErrForbidden) {
		t.Fatal("lectura ajena", e)
	}
}

type captureSender struct{ messages []mail.Message }

func (c *captureSender) Send(_ context.Context, m mail.Message) error {
	c.messages = append(c.messages, m)
	return nil
}
func TestAutomaticExpiryAndPrivacyRestrictions(t *testing.T) {
	f := newFixture(t, 1)
	ctx := context.Background()
	if e := f.repo.Enroll(ctx, f.admin, f.sub, exam.EnrollmentInput{ConfigID: f.examConfig.ID, MeritID: f.merit.ID}); e != nil {
		t.Fatal(e)
	}
	a, _, e := f.repo.Start(ctx, f.owner, f.sub)
	if e != nil {
		t.Fatal(e)
	}
	time.Sleep(time.Until(a.ExpiresAt) + 20*time.Millisecond)
	if _, _, e = f.repo.Answer(ctx, f.owner, f.sub, exam.AnswerInput{PreviousVersion: 1, QuestionID: a.Questions[0].ID, OptionID: a.Questions[0].Correct}); !errors.Is(e, exam.ErrExpired) {
		t.Fatal("respuesta tardía aceptada", e)
	}
	if e = f.repo.Expire(ctx); e != nil {
		t.Fatal(e)
	}
	finished, _, e := f.repo.Read(ctx, f.owner, f.sub, false)
	if e != nil || finished.Status != "finished" || *finished.Result.Score != 0 {
		t.Fatal("no cerró por reloj servidor", e)
	}
	before, e := f.privacy.Current(ctx, f.period)
	if e != nil {
		t.Fatal(e)
	}
	if e = f.privacy.Block(ctx, f.admin, f.period, privacy.BlockInput{PreviousVersion: before.ControlVersion, Blocked: true, Reason: "Revisión QA"}); e != nil {
		t.Fatal(e)
	}
	if e = f.privacy.Allow(ctx, f.sub, "datos"); !errors.Is(e, privacy.ErrDenied) {
		t.Fatal("restricción omitida", e)
	}
	guard := f.privacy.Guard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("PRIVADO")) }))
	w := httptest.NewRecorder()
	guard.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/submissions/"+f.sub.String()+"/print.pdf", nil))
	if w.Code != 409 || strings.Contains(w.Body.String(), "PRIVADO") {
		t.Fatal("salida de datos sin política", w.Code, w.Body.String())
	}
	sender := &captureSender{}
	gated := &privacypostgres.Sender{Gate: f.privacy, Next: sender}
	if e = f.queue.Dispatch(ctx, gated, nil); e != nil {
		t.Fatal(e)
	}
	if len(sender.messages) != 0 {
		t.Fatal("correo salió con política restringida")
	}
	if _, e = f.repo.Report(ctx, f.owner, f.sub, false); !errors.Is(e, privacy.ErrDenied) {
		t.Fatal("reporte sin política", e)
	}
	var count int
	if e = f.pool.QueryRow(ctx, `SELECT count(*) FROM exam_reports`).Scan(&count); e != nil || count != 1 {
		t.Fatal("restricción borró evidencia", e)
	}
}

type sessionStub struct {
	identity.SessionRepository
	actor *identity.Applicant
}

func (s sessionStub) GetWithApplicant(context.Context, string) (*identity.Session, *identity.Applicant, error) {
	return &identity.Session{ExpiresAt: time.Now().Add(time.Hour)}, s.actor, nil
}
func TestExamHTTPNeverExposesMatrixToApplicant(t *testing.T) {
	f := newFixture(t, 60)
	ctx := context.Background()
	if e := f.repo.Enroll(ctx, f.admin, f.sub, exam.EnrollmentInput{ConfigID: f.examConfig.ID, MeritID: f.merit.ID}); e != nil {
		t.Fatal(e)
	}
	if _, _, e := f.repo.Start(ctx, f.owner, f.sub); e != nil {
		t.Fatal(e)
	}
	router := chi.NewRouter()
	router.Use(identity.RequireSession(identity.NewService(identity.ServiceDependencies{Sessions: sessionStub{actor: f.owner}})))
	router.Use(identity.ProtectAdministration)
	exam.NewHandler(f.repo, f.auditor).RegisterRoutes(router)
	for _, path := range []string{"/api/v1/exams/submissions/" + f.sub.String(), "/api/v1/exams/submissions/" + f.sub.String() + "/questions/1"} {
		r := httptest.NewRequest("GET", path, nil)
		r.AddCookie(&http.Cookie{Name: "session", Value: "qa"})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
		for _, key := range []string{"correcta", "matriz", "hashBanco", "respuestas"} {
			if strings.Contains(w.Body.String(), key) {
				t.Fatal("matriz filtrada", w.Body.String())
			}
		}
	}
	r := httptest.NewRequest("GET", "/api/v1/admin/question-bank/versions/"+f.bank.ID.String(), nil)
	r.AddCookie(&http.Cookie{Name: "session", Value: "qa"})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("postulante consultó respuestas", w.Code)
	}
	r = httptest.NewRequest("POST", "/api/v1/exams/submissions/"+f.sub.String()+"/finish", strings.NewReader(`{"puntaje":100}`))
	r.AddCookie(&http.Cookie{Name: "session", Value: "qa"})
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != 422 {
		t.Fatal("aceptó puntaje cliente", w.Code)
	}
}

type brokenAudit struct{}

func (brokenAudit) Record(context.Context, audit.Event) error         { return audit.ErrUnavailable }
func (brokenAudit) RecordWorkflow(context.Context, audit.Event) error { return audit.ErrUnavailable }
func TestPrivacyVersionBindingAndAuditRollback(t *testing.T) {
	f := newFixture(t, 60)
	ctx := context.Background()
	current, e := f.privacy.Current(ctx, f.period)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = f.privacy.Publish(ctx, f.analyst, f.period, privacy.PublishInput{PreviousVersion: current.ControlVersion, Rules: current.Rules}); !errors.Is(e, privacy.ErrForbidden) {
		t.Fatal("analista aprobó privacidad", e)
	}
	bad := privacypostgres.NewRepository(f.q, f.tx, brokenAudit{})
	if _, e = bad.Publish(ctx, f.admin, f.period, privacy.PublishInput{PreviousVersion: current.ControlVersion, Rules: current.Rules}); !errors.Is(e, audit.ErrUnavailable) {
		t.Fatal("publicó sin auditoría", e)
	}
	var count int
	if e = f.pool.QueryRow(ctx, `SELECT count(*) FROM privacy_policies`).Scan(&count); e != nil || count != 1 {
		t.Fatal("política sobrevivió rollback", e)
	}
	nextRules := current.Rules
	nextRules.Text = "Nueva política QA NO OFICIAL"
	next, e := f.privacy.Publish(ctx, f.admin, f.period, privacy.PublishInput{PreviousVersion: current.ControlVersion, Rules: nextRules})
	if e != nil || next.ID == current.ID {
		t.Fatal("no versionó política", e)
	}
	if e = f.privacy.Allow(ctx, f.sub, "datos"); e != nil {
		t.Fatal("nueva política reescribió consentimiento anterior", e)
	}
	if _, e = f.pool.Exec(ctx, `UPDATE privacy_bindings SET policy_id=$2 WHERE submission_id=$1`, f.sub, next.ID); e == nil {
		t.Fatal("aceptación retroactiva permitida")
	}
	if _, e = f.pool.Exec(ctx, `DELETE FROM privacy_policies`); e == nil {
		t.Fatal("historial de política eliminable")
	}
	form, _ := submission.DecodeForm(f.snapshot.Content)
	if e = f.privacy.Bind(ctx, f.snapshot, form.Consent); !errors.Is(e, audit.ErrTransactionRequired) {
		t.Fatal("aceptación sin transacción", e)
	}
	if e = f.tx.WithinTx(ctx, func(tx pgx.Tx) error { return f.privacy.Bind(db.WithTransaction(ctx, tx), f.snapshot, form.Consent) }); !errors.Is(e, privacy.ErrDenied) {
		t.Fatal("aceptó versión obsoleta al enviar", e)
	}
	// La revisión devuelve sólo metadatos; ninguna fecha de revisión dispara borrado.
	due, e := f.privacy.Reviews(ctx, f.period, time.Time{}, uuid.Nil, 100)
	if e != nil || len(due) != 0 {
		t.Fatal("revisión prematura", e)
	}
	if e = bad.Block(ctx, f.admin, f.period, privacy.BlockInput{PreviousVersion: next.ControlVersion, Blocked: true, Reason: "QA"}); !errors.Is(e, audit.ErrUnavailable) {
		t.Fatal("restricción sin auditoría", e)
	}
	if e = f.privacy.Allow(ctx, f.sub, "datos"); e != nil {
		t.Fatal("restricción sobrevivió rollback", e)
	}
	if _, e = f.pool.Exec(ctx, `UPDATE applicants SET role='postulante' WHERE id=$1`, f.admin.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = f.privacy.Publish(ctx, f.admin, f.period, privacy.PublishInput{PreviousVersion: next.ControlVersion, Rules: nextRules}); !errors.Is(e, privacy.ErrForbidden) {
		t.Fatal("aprobó con rol revocado", e)
	}
}
func TestWorkerAndManualFinishRaceAfterDeadline(t *testing.T) {
	f := newFixture(t, 1)
	ctx := context.Background()
	if e := f.repo.Enroll(ctx, f.admin, f.sub, exam.EnrollmentInput{ConfigID: f.examConfig.ID, MeritID: f.merit.ID}); e != nil {
		t.Fatal(e)
	}
	a, _, e := f.repo.Start(ctx, f.owner, f.sub)
	if e != nil {
		t.Fatal(e)
	}
	time.Sleep(time.Until(a.ExpiresAt) + 20*time.Millisecond)
	var errs [2]error
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); errs[0] = f.repo.Expire(ctx) }()
	go func() { defer wg.Done(); _, _, errs[1] = f.repo.Finish(ctx, f.owner, f.sub, false) }()
	wg.Wait()
	if errs[0] != nil || errs[1] != nil {
		t.Fatal(errs)
	}
	var count int
	if e = f.pool.QueryRow(ctx, `SELECT count(*) FROM evaluation_records WHERE action='teorica.finalizada'`).Scan(&count); e != nil || count != 1 {
		t.Fatal("carrera duplicó nota", count, e)
	}
}

func TestConfigurationHTTPRequiresExplicitThreshold(t *testing.T) {
	f := newFixture(t, 60)
	router := chi.NewRouter()
	router.Use(identity.RequireSession(identity.NewService(identity.ServiceDependencies{Sessions: sessionStub{actor: f.admin}})))
	router.Use(identity.ProtectAdministration)
	exam.NewHandler(f.repo, f.auditor).RegisterRoutes(router)
	raw, _ := json.Marshal(f.examConfig.Rules)
	var body map[string]interface{}
	_ = json.Unmarshal(raw, &body)
	delete(body, "umbral")
	raw, _ = json.Marshal(body)
	call := func(raw []byte) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/api/v1/admin/question-bank/versions/"+f.bank.ID.String()+"/configurations", bytes.NewReader(raw))
		r.AddCookie(&http.Cookie{Name: "session", Value: "qa"})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}
	w := call(raw)
	if w.Code != 422 {
		t.Fatal("umbral implícito aceptado", w.Code, w.Body.String())
	}
	body["umbral"] = 0
	raw, _ = json.Marshal(body)
	w = call(raw)
	if w.Code != 409 {
		t.Fatal("cero explícito no llegó a control de configuración única", w.Code, w.Body.String())
	}
}
