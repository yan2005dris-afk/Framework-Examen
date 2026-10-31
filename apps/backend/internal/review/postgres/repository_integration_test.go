package postgres

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/db"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/review"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/shared/domain"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/testutil"
	"testing"
	"time"
)

func TestPostgresReviewKeepsClosedPeriodRosterSnapshotAndCertificate(t *testing.T) {
	pool := testutil.Postgres(t, "REVIEW_TEST_DATABASE_URL")
	ctx := context.Background()
	period, otherPeriod, owner := uuid.New(), uuid.New(), uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO applicants(id,cedula,nombres,apellidos,correo) VALUES($1,'1710034008','Nombre actual','Distinto','review@example.test')`, owner); err != nil {
		t.Fatal(err)
	}
	for index, id := range []uuid.UUID{period, otherPeriod} {
		if _, err := pool.Exec(ctx, `INSERT INTO competitions(id,code,title,opens_at,closes_at) VALUES($1,$2,'Prueba',now()-interval '1 day',now()+interval '1 day')`, id, fmt.Sprintf("TEST-%d", index)); err != nil {
			t.Fatal(err)
		}
	}
	source := []byte(`{"seccion1":{"nombres":"Nombre sellado","apellidos":"Original"},"seccion4":{"sector":"afiliados"}}`)
	canonical, err := submission.CanonicalContent(source)
	if err != nil {
		t.Fatal(err)
	}
	ids := []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()}
	certificate := uuid.New()
	createdAt := time.Now().Add(-time.Hour)
	for i, id := range ids {
		var competition *uuid.UUID
		if i < 2 || i == 4 {
			competition = &period
		} else if i == 2 {
			competition = &otherPeriod
		}
		if _, err := pool.Exec(ctx, `INSERT INTO submissions(id,applicant_id,competition_id,code,content,created_at) VALUES($1,$2,$3,$4,$5,$6)`, id, owner, competition, fmt.Sprintf("REV-%d", i), canonical, createdAt); err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			if _, err := pool.Exec(ctx, `INSERT INTO documents(id,submission_id,section,record_id,name,mime,size_bytes,sha256,storage_key) VALUES($1,$2,5,'titulo-1','respaldo.pdf','application/pdf',100,repeat('a',64),'objeto-prueba')`, certificate, id); err != nil {
				t.Fatal(err)
			}
		}
		if i != 1 {
			if _, err := pool.Exec(ctx, `UPDATE submissions SET status='submitted',hash=$2,submitted_at=now() WHERE id=$1`, id, submission.HashContent(canonical)); err != nil {
				t.Fatal(err)
			}
		}
	}
	// El período se vuelve cerrado por transcurso del tiempo, sin reescribir su identidad.
	// Adelantar fechas sólo en la sesión de pruebas con el trigger deshabilitado permite reproducirlo sin dormir un día.
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `ALTER TABLE competitions DISABLE TRIGGER competitions_history`); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `UPDATE competitions SET opens_at=now()-interval '2 days',closes_at=now()-interval '1 day' WHERE id=$1`, period); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `ALTER TABLE competitions ENABLE TRIGGER competitions_history`); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	// La consulta del período cerrado no cambia la selección.
	repo := NewRepository(sqlcgen.New(pool), db.NewTxManager(pool))
	f := review.Filter{CompetitionID: &period}
	rows, err := repo.List(ctx, f, domain.Position{}, 20)
	if err != nil || len(rows) != 2 {
		t.Fatalf("se mezclaron convocatorias/borradores: %v %d", err, len(rows))
	}
	if string(rows[0].Content) == "" || rows[0].Hash != submission.HashContent(canonical) {
		t.Fatal("snapshot perdido")
	}
	counts, err := repo.Counts(ctx, f)
	if err != nil || len(counts) != 1 || counts[0].Applicants != 1 || counts[0].Submissions != 2 {
		t.Fatal(counts, err)
	}
	next, err := repo.List(ctx, f, domain.Position{At: rows[0].CreatedAt, ID: rows[0].ID}, 1)
	if err != nil || len(next) != 1 || next[0].ID != rows[1].ID {
		t.Fatal("cursor pierde empates de fecha", err)
	}
	exported, err := repo.Export(ctx, f)
	if err != nil || len(exported) != len(rows) {
		t.Fatal("exportación no conserva selección", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE submissions SET status='submitted',hash=$2,submitted_at=now() WHERE id=$1`, ids[1], submission.HashContent(canonical)); err == nil {
		t.Fatal("envío tardío admitido")
	}
	if _, err := pool.Exec(ctx, `INSERT INTO submissions(id,applicant_id,competition_id,code) VALUES($1,$2,$3,'FUERA-PLAZO')`, uuid.New(), owner, period); err == nil {
		t.Fatal("alta en período cerrado admitida")
	}
	legacy, err := repo.List(ctx, review.Filter{Legacy: true}, domain.Position{}, 20)
	if err != nil || len(legacy) != 1 || legacy[0].ID != ids[3] {
		t.Fatal("legado mezclado", err)
	}
	for _, statement := range []string{
		`UPDATE submissions SET content='{}' WHERE id=$1`,
		`UPDATE submissions SET competition_id=NULL WHERE id=$1`,
		`DELETE FROM submissions WHERE id=$1`,
	} {
		if _, err := pool.Exec(ctx, statement, ids[0]); err == nil {
			t.Fatal("snapshot modificable", statement)
		}
	}
	for _, statement := range []string{`UPDATE documents SET name='otro.pdf' WHERE id=$1`, `DELETE FROM documents WHERE id=$1`} {
		if _, err := pool.Exec(ctx, statement, certificate); err == nil {
			t.Fatal("certificado modificable", statement)
		}
	}
	if _, err := pool.Exec(ctx, `UPDATE competitions SET title='Otro período' WHERE id=$1`, period); err == nil {
		t.Fatal("identidad histórica modificable")
	}
	if _, err := pool.Exec(ctx, `UPDATE applicants SET nombres='Nuevo perfil' WHERE id=$1`, owner); err != nil {
		t.Fatal(err)
	}
	after, err := repo.List(ctx, f, domain.Position{}, 20)
	if err != nil || string(after[0].Content) != string(rows[0].Content) {
		t.Fatal("snapshot depende del perfil vigente", err)
	}
}
