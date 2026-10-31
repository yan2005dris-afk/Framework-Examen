package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	auditpostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit/postgres"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/competition"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/eligibility"
	eligibilitypostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/eligibility/postgres"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	identitypostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity/postgres"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/db"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	submissionpostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission/postgres"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/testutil"
	"testing"
	"time"
)

type conditionStub struct {
	value bool
	fail  bool
}

func (p *conditionStub) Resolve(context.Context, uuid.UUID, string) (eligibility.Condition, error) {
	if p.fail {
		return eligibility.Condition{}, errors.New("proveedor caído")
	}
	return eligibility.Condition{Eligible: p.value, Source: "simulacion_controlada_servidor"}, nil
}

func TestPostgresLifecycleApprovalWindowEligibilityAndHistory(t *testing.T) {
	pool := testutil.Postgres(t, "LIFECYCLE_TEST_DATABASE_URL")
	ctx := context.Background()
	q := sqlcgen.New(pool)
	tx := db.NewTxManager(pool)
	auditor := auditpostgres.NewRepository(q)
	actor, owner := uuid.New(), uuid.New()
	for i, id := range []uuid.UUID{actor, owner} {
		role := "postulante"
		if i == 0 {
			role = "administrador"
		}
		if _, err := pool.Exec(ctx, `INSERT INTO applicants(id,cedula,nombres,apellidos,correo,role) VALUES($1,$2,'Persona','Prueba',$3,$4)`, id, []string{"0000000001", "0000000002"}[i], id.String()+"@example.test", role); err != nil {
			t.Fatal(err)
		}
	}
	repo := NewRepository(q, tx, auditor)
	service := competition.NewService(repo)
	admin := &identity.Applicant{ID: actor, Role: identity.RoleAdministrator}
	start, end := time.Now().Add(-time.Hour).UTC(), time.Now().Add(time.Hour).UTC()
	c := &competition.Competition{ID: uuid.New(), Code: "CICLO-PRUEBA", Title: "Período sintético", OpensAt: start, ClosesAt: end, Configuration: competition.Configuration{Sectors: []string{"afiliados"}, Phases: []competition.Phase{}}}
	if err := service.Create(ctx, admin, c); err != nil {
		t.Fatal(err)
	}
	duplicate := *c
	duplicate.ID = uuid.New()
	duplicate.Version = 0
	if err := service.Create(ctx, admin, &duplicate); !errors.Is(err, competition.ErrDuplicate) {
		t.Fatal("duplicado no distinguido", err)
	}
	subrepo := submissionpostgres.NewRepository(q, tx)
	provider := &conditionStub{value: true}
	subservice := submission.NewService(subrepo).WithEligibility(eligibility.NewService(provider, eligibilitypostgres.NewRepository(q)))
	if _, err := subservice.Create(ctx, submission.CreateInput{ApplicantID: owner, CompetitionID: &c.ID}); !errors.Is(err, submission.ErrInvalidCompetition) {
		t.Fatal("alta sin habilitación", err)
	}
	config := competition.Configuration{Version: 1, Approved: true, Published: true, FormEnabled: true, Sectors: []string{"afiliados"}, Phases: []competition.Phase{{Code: "postulacion", Title: "Recepción sintética", OpensAt: start, ClosesAt: end, Published: true, Enabled: true}}}
	current, err := service.Configure(ctx, admin, c.ID, config)
	if err != nil || current.Version != 2 || current.ApprovedBy == nil || *current.ApprovedBy != actor {
		t.Fatal(current, err)
	}
	if _, err = service.Configure(ctx, admin, c.ID, config); !errors.Is(err, competition.ErrVersion) {
		t.Fatal("se perdió actualización concurrente", err)
	}
	// Este borrador representa una pestaña abierta antes del cierre de la fase.
	openDraft, err := subservice.Create(ctx, submission.CreateInput{ApplicantID: actor, CompetitionID: &c.ID})
	if err != nil {
		t.Fatal(err)
	}
	docID := uuid.New()
	if _, err = pool.Exec(ctx, `INSERT INTO documents(id,submission_id,section,record_id,name,mime,size_bytes,sha256,storage_key) VALUES($1,$2,5,'titulo','qa.pdf','application/pdf',10,repeat('a',64),'qa-objeto')`, docID, openDraft.ID); err != nil {
		t.Fatal(err)
	}
	item, err := subservice.Create(ctx, submission.CreateInput{ApplicantID: owner, CompetitionID: &c.ID})
	if err != nil {
		t.Fatal(err)
	}
	item, err = subservice.SaveSection(ctx, item.ID, 4, json.RawMessage(`{"sector":"afiliados","verificacionIessCumplida":false}`))
	if err != nil || item.Eligibility == nil || !item.Eligibility.Eligible {
		t.Fatal(item, err)
	}
	provider.fail = true
	if _, err = subservice.SaveSection(ctx, item.ID, 4, json.RawMessage(`{"sector":"afiliados"}`)); !errors.Is(err, submission.ErrEligibilityUnavailable) {
		t.Fatal("fallo técnico no distinguido", err)
	}
	loaded, _ := subrepo.FindByID(ctx, item.ID)
	if loaded.Status != submission.StatusDraft {
		t.Fatal("fallo técnico cerró el expediente")
	}
	provider.fail = false
	provider.value = false
	if _, err = subservice.SaveSection(ctx, item.ID, 4, json.RawMessage(`{"sector":"afiliados","verificacionIessCumplida":true}`)); !errors.Is(err, submission.ErrEligibilityClosed) {
		t.Fatal("se aceptó la bandera del cliente", err)
	}
	loaded, err = subrepo.FindByID(ctx, item.ID)
	if err != nil || loaded.Status != submission.StatusClosed || loaded.Eligibility == nil || loaded.Eligibility.Eligible || loaded.Hash != "" {
		t.Fatal("cierre no durable", loaded, err)
	}
	if _, err = subservice.Create(ctx, submission.CreateInput{ApplicantID: owner, CompetitionID: &c.ID}); !errors.Is(err, submission.ErrEligibilityClosed) {
		t.Fatal("se evadió el cierre creando otro borrador", err)
	}
	var count int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM eligibility_checks WHERE submission_id=$1`, item.ID).Scan(&count); err != nil || count != 2 {
		t.Fatal("evidencia no transaccional", count, err)
	}
	if _, err = pool.Exec(ctx, `UPDATE eligibility_checks SET result=true WHERE submission_id=$1`, item.ID); err == nil {
		t.Fatal("evidencia modificable")
	}
	if _, err = pool.Exec(ctx, `UPDATE submissions SET status='draft' WHERE id=$1`, item.ID); err == nil {
		t.Fatal("cierre reabierto mediante SQL")
	}
	config.Version = 2
	config.Phases[0].ClosesAt = time.Now().Add(-time.Second)
	if _, err = service.Configure(ctx, admin, c.ID, config); err != nil {
		t.Fatal(err)
	}
	if _, err = subservice.SaveSection(ctx, openDraft.ID, 2, json.RawMessage(`{}`)); !errors.Is(err, submission.ErrInvalidCompetition) {
		t.Fatal("pestaña abierta guardó fuera de plazo", err)
	}
	if _, err = pool.Exec(ctx, `UPDATE documents SET name='otro.pdf' WHERE id=$1`, docID); err == nil {
		t.Fatal("adjunto modificado fuera de plazo")
	}
	if _, err = pool.Exec(ctx, `DELETE FROM documents WHERE id=$1`, docID); err == nil {
		t.Fatal("adjunto retirado fuera de plazo")
	}
	// No existe bandera del navegador que permita volver a abrir el plazo.
	if _, err = subservice.Create(ctx, submission.CreateInput{ApplicantID: actor, CompetitionID: &c.ID}); !errors.Is(err, submission.ErrInvalidCompetition) {
		t.Fatal("alta fuera de plazo", err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM competition_versions WHERE competition_id=$1`, c.ID).Scan(&count); err != nil || count != 3 {
		t.Fatal("configuración histórica incompleta", count, err)
	}
	if _, err = identitypostgres.NewRepository(q, tx).ChangeRole(ctx, actor, actor, "analista", "Revocación sintética de prueba"); err != nil {
		t.Fatal(err)
	}
	config.Version = 3
	if _, err = service.Configure(ctx, admin, c.ID, config); !errors.Is(err, competition.ErrForbidden) {
		t.Fatal("el rol de una sesión antigua conservó permisos", err)
	}
}
