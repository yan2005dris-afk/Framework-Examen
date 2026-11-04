package postgres

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/testutil"
)

// TestPostgresSectorColumnFollowsForm comprueba que el sector es una columna propia
// que escribe el adaptador, restringida a las opciones del formulario y coherente
// con el snapshot de los expedientes que ya no se editan.
func TestPostgresSectorColumnFollowsForm(t *testing.T) {
	repo, item, pool := integrationEnvironment(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	header := func(id uuid.UUID) (*string, bool) {
		t.Helper()
		var sector *string
		var hasContent bool
		if err := pool.QueryRow(ctx, `SELECT sector, content IS NOT NULL FROM submissions WHERE id=$1`, id).Scan(&sector, &hasContent); err != nil {
			t.Fatal(err)
		}
		return sector, hasContent
	}

	if sector, _ := header(item.ID); sector != nil {
		t.Fatalf("sin sección 4 el sector debe ser NULL: %q", *sector)
	}
	for _, want := range []string{"afiliados", "jubilados", ""} {
		if err := item.UpdateContent(json.RawMessage(`{"seccion4":{"sector":"`+want+`"}}`), time.Now().UTC()); err != nil {
			t.Fatal(err)
		}
		if err := repo.Update(ctx, item); err != nil {
			t.Fatalf("guardar sector %q: %v", want, err)
		}
		sector, hasContent := header(item.ID)
		if sector == nil || *sector != want {
			t.Fatalf("sector esperado %q, obtenido %v", want, sector)
		}
		// El borrador vive sólo en las tablas.
		if hasContent {
			t.Fatal("un borrador no debe guardar content")
		}
	}

	// Un sector fuera de definition.json lo rechaza la base aunque se salte el servicio:
	// primero la ventana de la convocatoria y, debajo, el CHECK de la columna.
	invalid := `UPDATE submissions SET sector='otro' WHERE id=$1`
	if _, err := pool.Exec(ctx, invalid, item.ID); err == nil || !strings.Contains(err.Error(), "El sector no está habilitado") {
		t.Fatalf("la ventana debe rechazar un sector no habilitado: %v", err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `ALTER TABLE submissions DISABLE TRIGGER submissions_window`); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, invalid, item.ID); err == nil || !strings.Contains(err.Error(), "submissions_sector_check") {
		t.Fatalf("sector inválido debe violar submissions_sector_check: %v", err)
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}

	// Un snapshot insertado sin sector lo recibe del propio snapshot; uno que discrepa se rechaza.
	snapshot := json.RawMessage(`{"seccion4":{"sector":"jubilados"}}`)
	canonical, _ := submission.CanonicalContent(snapshot)
	insert := func(sector any) (uuid.UUID, error) {
		id := uuid.New()
		_, err := pool.Exec(ctx, `INSERT INTO submissions(id,applicant_id,code,status,content,hash,submitted_at,competition_id,sector)
			VALUES($1,$2,$3,'submitted',$4,$5,now(),$6,$7)`, id, item.ApplicantID, "SNAP-"+id.String()[:8], snapshot,
			submission.HashContent(canonical), testutil.OpenCompetition(t, pool), sector)
		return id, err
	}
	derived, err := insert(nil)
	if err != nil {
		t.Fatalf("snapshot sin sector: %v", err)
	}
	if sector, hasContent := header(derived); sector == nil || *sector != "jubilados" || !hasContent {
		t.Fatalf("el sector debe derivarse del snapshot: %v", sector)
	}
	if _, err := insert("empleadores"); err == nil || !strings.Contains(err.Error(), "submissions_sector_snapshot") {
		t.Fatalf("un sector que discrepa del snapshot debe rechazarse: %v", err)
	}
	// Todo expediente enviado conserva su snapshot.
	if _, err := pool.Exec(ctx, `INSERT INTO submissions(id,applicant_id,code,status,hash,submitted_at,competition_id)
		VALUES($1,$2,'SIN-SNAPSHOT','submitted',repeat('a',64),now(),$3)`, uuid.New(), item.ApplicantID, testutil.OpenCompetition(t, pool)); err == nil ||
		!strings.Contains(err.Error(), "submissions_sealed_content") {
		t.Fatalf("un expediente enviado sin snapshot debe rechazarse: %v", err)
	}

	// El índice de revisión usa la columna, no una expresión sobre el JSON.
	// regclass lo resuelve en el esquema aislado sin recorrer el catálogo de los
	// demás paquetes, que pueden estar borrando sus esquemas en paralelo.
	var definition string
	if err := pool.QueryRow(ctx, `SELECT pg_get_indexdef('submissions_review_sector_idx'::regclass)`).Scan(&definition); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(definition, "(competition_id, sector, created_at DESC, id DESC)") || strings.Contains(definition, "seccion4") {
		t.Fatalf("índice de revisión inesperado: %s", definition)
	}
}
