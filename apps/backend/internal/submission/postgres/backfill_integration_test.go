package postgres

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/testutil"
)

// legacySubmission inserta una postulación como existía antes de 00024: sólo con `content`.
func legacySubmission(t *testing.T, pool *pgxpool.Pool, applicant uuid.UUID, status string, content json.RawMessage) uuid.UUID {
	t.Helper()
	id := uuid.New()
	competition := testutil.OpenCompetition(t, pool)
	var hash any
	var submitted any
	if status == "submitted" {
		canonical, err := submission.CanonicalContent(content)
		if err != nil {
			t.Fatal(err)
		}
		hash, submitted = submission.HashContent(canonical), time.Now().UTC()
	}
	if _, err := pool.Exec(context.Background(), `INSERT INTO submissions(id,applicant_id,code,status,content,hash,submitted_at,competition_id)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, id, applicant, "LEGADO-"+id.String()[:8], status, content, hash, submitted, competition); err != nil {
		t.Fatalf("postulación heredada %s: %v", status, err)
	}
	return id
}

// runBackfill ejecuta el relleno en su propia transacción, como la migración 00025.
func runBackfill(t *testing.T, pool *pgxpool.Pool) (BackfillReport, error) {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	report, err := BackfillForms(ctx, tx, time.Now())
	if err != nil {
		return report, err
	}
	return report, tx.Commit(ctx)
}

// TestPostgresBackfillNormalizesExistingSubmissions cubre la migración de datos de la fase 4.
func TestPostgresBackfillNormalizesExistingSubmissions(t *testing.T) {
	repo, item, pool := integrationEnvironment(t)
	ctx := context.Background()
	_, cases := sharedCases(t)
	base := cases["Formulario completo sin firma digital"]

	// Borrador completo y borrador con una referencia a un PDF que no existe.
	draft := legacySubmission(t, pool, item.ApplicantID, "draft", base)
	foreign := patchJSON(t, base, []string{"seccion4", "documentoRespaldoPdf"}, json.RawMessage(`{"id":"`+uuid.NewString()+`","nombre":"x.pdf","tamano":"1"}`), false)
	withForeign := legacySubmission(t, pool, item.ApplicantID, "draft", foreign)
	// Expediente sellado y uno sellado que no corresponde al formulario actual.
	sealed := legacySubmission(t, pool, item.ApplicantID, "submitted", base)
	odd := legacySubmission(t, pool, item.ApplicantID, "submitted", json.RawMessage(`{"desconocido":1}`))
	var sealedBefore string
	if err := pool.QueryRow(ctx, `SELECT md5(content::text)||hash||updated_at FROM submissions WHERE id=$1`, sealed).Scan(&sealedBefore); err != nil {
		t.Fatal(err)
	}

	report, err := runBackfill(t, pool)
	if err != nil {
		t.Fatalf("relleno: %v", err)
	}
	// El borrador vacío del entorno más los dos heredados; el sellado válido sí se descompone.
	if report.Drafts != 3 || report.Sealed != 1 || !slices.Equal(report.Skipped, []uuid.UUID{odd}) {
		t.Fatalf("resumen inesperado: %+v", report)
	}

	for _, id := range []uuid.UUID{draft, withForeign} {
		got, err := repo.FindByID(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		want, err := repo.expectedForm(ctx, id, base, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		if loaded, _ := submission.DecodeForm(got.Content); canonicalText(loaded) != canonicalText(want) {
			t.Fatalf("borrador %s leído desde las tablas:\n%s\n%s", id, got.Content, canonicalText(want))
		}
	}

	var sealedAfter string
	var sections int
	if err := pool.QueryRow(ctx, `SELECT md5(content::text)||hash||updated_at FROM submissions WHERE id=$1`, sealed).Scan(&sealedAfter); err != nil {
		t.Fatal(err)
	}
	if sealedAfter != sealedBefore {
		t.Fatal("el relleno modificó un expediente sellado")
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM submission_sections WHERE submission_id=$1`, sealed).Scan(&sections); err != nil || sections != 9 {
		t.Fatalf("el expediente sellado debe quedar descompuesto en sus 9 secciones: %d %v", sections, err)
	}

	// Los triggers de las tablas vuelven a proteger los expedientes enviados.
	if _, err := pool.Exec(ctx, `UPDATE submission_personal SET first_name='Otra' WHERE submission_id=$1`, sealed); err == nil ||
		!strings.Contains(err.Error(), "El expediente no admite cambios") {
		t.Fatalf("los triggers deben quedar reactivados: %v", err)
	}

	// Repetir el relleno da el mismo resultado.
	again, err := runBackfill(t, pool)
	if err != nil || again.Drafts != report.Drafts || again.Sealed != report.Sealed {
		t.Fatalf("el relleno debe ser idempotente: %+v %v", again, err)
	}
}

// TestPostgresBackfillFailsOnInvalidDraft exige que un borrador que no cabe en las
// tablas detenga la migración, sin dejar nada a medias.
func TestPostgresBackfillFailsOnInvalidDraft(t *testing.T) {
	_, item, pool := integrationEnvironment(t)
	ctx := context.Background()
	invalid := legacySubmission(t, pool, item.ApplicantID, "draft", json.RawMessage(`{"seccion1":{"genero":"Otro"}}`))

	_, err := runBackfill(t, pool)
	if err == nil || !strings.Contains(err.Error(), invalid.String()) {
		t.Fatalf("el relleno debe fallar y nombrar el borrador: %v", err)
	}
	var rows int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM submission_sections`).Scan(&rows); err != nil || rows != 0 {
		t.Fatalf("un relleno fallido no debe dejar filas: %d %v", rows, err)
	}
}
