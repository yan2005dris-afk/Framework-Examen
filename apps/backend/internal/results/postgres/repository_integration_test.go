package postgres

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit"
	auditpostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit/postgres"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/competition"
	competitionpostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/competition/postgres"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/evaluation"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	noticepostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/notice/postgres"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/db"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/mail"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/results"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/results/testfixture"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/shared/domain"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/testutil"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type failure struct{}

func (failure) RecordWorkflow(context.Context, audit.Event) error { return audit.ErrUnavailable }
func (failure) Record(context.Context, audit.Event) error         { return audit.ErrUnavailable }
func (failure) Enqueue(context.Context, uuid.UUID, uuid.UUID, string) error {
	return errors.New("cola QA no disponible")
}
func seedSource(t *testing.T, pool *pgxpool.Pool, period uuid.UUID, policy *evaluation.Policy, n int) results.Source {
	t.Helper()
	ctx := context.Background()
	s, err := testfixture.Source(period, policy, int64(90-n*10))
	if err != nil {
		t.Fatal(err)
	}
	owner := s.Snapshot.ApplicantID
	if _, err = pool.Exec(ctx, `INSERT INTO applicants(id,cedula,nombres,apellidos,correo) VALUES($1,$2,'Persona','QA',$3)`, owner, fmt.Sprintf("00000000%02d", n+10), owner.String()+"@example.test"); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO submissions(id,applicant_id,competition_id,code,status,content,created_at,updated_at) VALUES($1,$2,$3,$4,'draft',$5,$6,$6)`, s.Snapshot.ID, owner, period, s.Snapshot.Code, s.Snapshot.Content, s.Snapshot.CreatedAt); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE submissions SET status='submitted',hash=$2,submitted_at=$3 WHERE id=$1`, s.Snapshot.ID, s.Snapshot.Hash, s.Snapshot.SubmittedAt); err != nil {
		t.Fatal(err)
	}
	m := s.Merit
	review := uuid.New()
	raw, _ := json.Marshal(m)
	// Fuentes T20/T21/T27 sintéticas ya finalizadas: sus productores se prueban en sus contextos.
	statements := []struct {
		sql  string
		args []interface{}
	}{
		{`INSERT INTO merit_rule_sets(id,competition_id,sector,version,definition,definition_hash,actor_id,created_at) VALUES($1,$2,'afiliados',$3,'{}',$4,$5,now())`, []interface{}{m.RuleSetID, period, n + 1, m.RulesHash, owner}},
		{`INSERT INTO merit_reviews(id,submission_id,version,rule_set_id,payload) VALUES($1,$2,1,$3,'{}')`, []interface{}{review, s.Snapshot.ID, m.RuleSetID}},
		{`INSERT INTO merit_calculations(id,submission_id,version,review_id,rule_set_id,payload) VALUES($1,$2,1,$3,$4,'{}')`, []interface{}{m.CalculationID, s.Snapshot.ID, review, m.RuleSetID}},
		{`INSERT INTO merit_publications(id,submission_id,version,calculation_id,decision,payload) VALUES($1,$2,1,$3,'avanza',$4)`, []interface{}{m.ID, s.Snapshot.ID, m.CalculationID, raw}},
		{`INSERT INTO merit_case_streams(submission_id,review_id,calculation_id,publication_id) VALUES($1,$2,$3,$4)`, []interface{}{s.Snapshot.ID, review, m.CalculationID, m.ID}},
	}
	for _, st := range statements {
		if _, err = pool.Exec(ctx, st.sql, st.args...); err != nil {
			t.Fatal(err)
		}
	}
	raw, _ = json.Marshal(s.State)
	if _, err = pool.Exec(ctx, `INSERT INTO evaluation_heads(submission_id,state) VALUES($1,$2)`, s.Snapshot.ID, raw); err != nil {
		t.Fatal(err)
	}
	raw, _ = json.Marshal(s.Theory)
	if _, err = pool.Exec(ctx, `INSERT INTO evaluation_records(id,submission_id,action,phase,payload,created_at) VALUES($1,$2,'teorica.finalizada','oposicion_teorica',$3,$4)`, s.Theory.ID, s.Snapshot.ID, raw, s.Theory.PublishedAt); err != nil {
		t.Fatal(err)
	}
	return s
}
func TestResultListsVersionedPublicationPrivacyAndRollback(t *testing.T) {
	pool := testutil.Postgres(t, "RESULTS_TEST_DATABASE_URL")
	ctx := context.Background()
	q := sqlcgen.New(pool)
	tx := db.NewTxManager(pool)
	auditor := auditpostgres.NewRepository(q)
	admin := &identity.Applicant{ID: uuid.New(), Role: identity.RoleAdministrator}
	analyst := &identity.Applicant{ID: uuid.New(), Role: identity.RoleAnalyst}
	for i, a := range []*identity.Applicant{admin, analyst} {
		if _, err := pool.Exec(ctx, `INSERT INTO applicants(id,cedula,nombres,apellidos,correo,role) VALUES($1,$2,'Operador','QA',$3,$4)`, a.ID, fmt.Sprintf("000000000%d", i+1), a.ID.String()+"@example.test", a.Role); err != nil {
			t.Fatal(err)
		}
	}
	start, end := time.Now().UTC().Add(-3*time.Hour), time.Now().UTC().Add(3*time.Hour)
	period := &competition.Competition{ID: uuid.New(), Code: "LISTAS-QA-NO-OFICIAL", Title: "Prueba de publicación", OpensAt: start, ClosesAt: end, Configuration: competition.Configuration{Sectors: []string{"afiliados"}, Phases: []competition.Phase{}}}
	calendar := competition.NewService(competitionpostgres.NewRepository(q, tx, auditor))
	if err := calendar.Create(ctx, admin, period); err != nil {
		t.Fatal(err)
	}
	config := competition.Configuration{Version: 1, Approved: true, Published: true, FormEnabled: true, Sectors: []string{"afiliados"}, Phases: []competition.Phase{{Code: "postulacion", Title: "QA", OpensAt: start, ClosesAt: end, Published: true, Enabled: true}}}
	current, err := calendar.Configure(ctx, admin, period.ID, config)
	if err != nil {
		t.Fatal(err)
	}
	config = current.Configuration
	policy := testfixture.Policy(period.ID)
	raw, _ := json.Marshal(policy.Definition)
	if _, err = pool.Exec(ctx, `INSERT INTO evaluation_policies(id,competition_id,sector,kind,version,definition,definition_hash,approved,approval_reference,approved_by,approved_at,actor_id,created_at) VALUES($1,$2,'afiliados','oposicion',1,$3,$4,true,'QA',$5,now(),$5,now())`, policy.ID, period.ID, raw, policy.Hash, admin.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO evaluation_policy_streams(competition_id,sector,kind,version,active_id) VALUES($1,'afiliados','oposicion',1,$2)`, period.ID, policy.ID); err != nil {
		t.Fatal(err)
	}
	source := seedSource(t, pool, period.ID, policy, 0)
	seedSource(t, pool, period.ID, policy, 1)
	outbox, err := mail.NewOutbox(q, base64.StdEncoding.EncodeToString(make([]byte, 32)))
	if err != nil {
		t.Fatal(err)
	}
	queue := &noticepostgres.Queue{Outbox: outbox, Queries: q}
	repo := NewRepository(q, tx, auditor, queue)
	service := results.NewService(repo, auditor)
	input := results.GenerateInput{Rules: results.Rules{Order: []string{"resultados", "meritos"}, Reference: "Acta QA no oficial"}}
	if _, err = service.Generate(ctx, analyst, period.ID, "afiliados", uuid.New(), input); !errors.Is(err, results.ErrPhase) {
		t.Fatal("publicó antes del cierre de reclamaciones", err)
	}
	now := time.Now().UTC()
	config.FormEnabled = false
	config.Phases = []competition.Phase{{Code: "postulacion", Title: "QA", OpensAt: start, ClosesAt: now.Add(-2 * time.Hour), Published: true}, {Code: "reconsideracion", Title: "QA", OpensAt: now.Add(-90 * time.Minute), ClosesAt: now.Add(-30 * time.Minute), Published: true}, {Code: "resultados", Title: "QA", OpensAt: now.Add(-time.Minute), ClosesAt: end, Enabled: true, Published: true}}
	current, err = calendar.Configure(ctx, admin, period.ID, config)
	if err != nil {
		t.Fatal(err)
	}
	config = current.Configuration
	k := uuid.New()
	draft, err := service.Generate(ctx, analyst, period.ID, "afiliados", k, input)
	if err != nil {
		t.Fatal(err)
	}
	again, err := service.Generate(ctx, analyst, period.ID, "afiliados", k, input)
	if err != nil || again.ID != draft.ID || len(again.Entries) != 2 {
		t.Fatal("duplicó borrador o perdió entradas", err)
	}
	publication := results.PublishInput{Reference: "Acta QA de publicación", Public: true, PrivacyReference: "Política QA de nombres, no oficial"}
	if _, err = service.Publish(ctx, analyst, draft.ID, uuid.New(), publication); !errors.Is(err, results.ErrForbidden) {
		t.Fatal("analista publicó", err)
	}
	router := chi.NewRouter()
	results.NewHandler(service).RegisterPublic(router)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/public/results/competitions/"+period.ID.String()+"/sectors/afiliados", nil))
	if w.Code != 404 {
		t.Fatal("borrador público", w.Code)
	}
	// Una fuente distinta invalida el borrador; nunca se publica una consolidación obsoleta.
	state := source.State
	v := state.Results["oposicion_psicotecnica"]
	old := v.ID
	v.ID = uuid.New()
	state.Results[v.Phase] = v
	raw, _ = json.Marshal(state)
	if _, err = pool.Exec(ctx, `UPDATE evaluation_heads SET state=$2 WHERE submission_id=$1`, source.Snapshot.ID, raw); err != nil {
		t.Fatal(err)
	}
	if _, err = service.Publish(ctx, admin, draft.ID, uuid.New(), publication); err == nil {
		t.Fatal("aceptó fuentes distintas")
	}
	v.ID = old
	state.Results[v.Phase] = v
	raw, _ = json.Marshal(state)
	if _, err = pool.Exec(ctx, `UPDATE evaluation_heads SET state=$2 WHERE submission_id=$1`, source.Snapshot.ID, raw); err != nil {
		t.Fatal(err)
	}
	pubKey := uuid.New()
	published, err := service.Publish(ctx, admin, draft.ID, pubKey, publication)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := service.Publish(ctx, admin, draft.ID, pubKey, publication)
	if err != nil || replay.ID != published.ID {
		t.Fatal("publicación no idempotente", err)
	}
	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/public/results/competitions/"+period.ID.String()+"/sectors/afiliados", nil))
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, private := range []string{source.Snapshot.ID.String(), source.Snapshot.Hash, source.Snapshot.Code, "privado@example.test", "0000000000", `"puntaje"`, "responsableId", "referenciaPrivacidad"} {
		if strings.Contains(w.Body.String(), private) {
			t.Fatal("proyección pública expone datos", private)
		}
	}
	var count int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM mail_outbox WHERE kind='results.list'`).Scan(&count); err != nil || count != 2 {
		t.Fatal("avisos duplicados o ausentes", count, err)
	}
	csv, err := service.Export(ctx, analyst, draft.ID)
	if err != nil || !strings.Contains(string(csv), source.Snapshot.Hash) || !strings.Contains(string(csv), source.Merit.ID.String()) {
		t.Fatal("reporte no vinculado a T9/resultados", err)
	}
	second, err := service.Generate(ctx, analyst, period.ID, "afiliados", uuid.New(), input)
	if err != nil {
		t.Fatal(err)
	}
	next := results.PublishInput{PreviousVersion: 1, SupersedesID: &published.ID, Reference: "Segunda acta QA"}
	broken := results.NewService(NewRepository(q, tx, auditor, failure{}), auditor)
	if _, err = broken.Publish(ctx, admin, second.ID, uuid.New(), next); err == nil {
		t.Fatal("confirmó sin cola")
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM result_list_publications`).Scan(&count); err != nil || count != 1 {
		t.Fatal("publicación sobrevivió rollback", err)
	}
	if _, err = service.Publish(ctx, admin, second.ID, uuid.New(), next); err != nil {
		t.Fatal(err)
	}
	if _, err = service.Public(ctx, period.ID, "afiliados", true); !errors.Is(err, results.ErrNotFound) {
		t.Fatal("lista interna nueva dejó vigente versión pública anterior", err)
	}
	if _, err = service.Public(ctx, published.ID, "", false); err != nil {
		t.Fatal("perdió histórico público", err)
	}
	history, err := service.Publications(ctx, analyst, period.ID, "afiliados", domain.NewCursorRequest(100, nil))
	if err != nil || len(history.Items) != 2 {
		t.Fatal("faltan versiones", err)
	}
	if _, err = pool.Exec(ctx, `DELETE FROM result_lists WHERE id=$1`, draft.ID); err == nil {
		t.Fatal("lista histórica eliminable")
	}
	failAudit := results.NewService(NewRepository(q, tx, failure{}, queue), failure{})
	if v, err := failAudit.Get(ctx, admin, draft.ID); !errors.Is(err, audit.ErrUnavailable) || v != nil {
		t.Fatal("lectura sin auditoría", err)
	}
	if _, err = failAudit.Generate(ctx, analyst, period.ID, "afiliados", uuid.New(), input); !errors.Is(err, audit.ErrUnavailable) {
		t.Fatal("generó sin auditoría", err)
	}
	if _, err = pool.Exec(ctx, `UPDATE applicants SET role='postulante' WHERE id=$1`, analyst.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = service.Generate(ctx, analyst, period.ID, "afiliados", uuid.New(), input); !errors.Is(err, results.ErrForbidden) {
		t.Fatal("aceptó rol revocado", err)
	}
}
