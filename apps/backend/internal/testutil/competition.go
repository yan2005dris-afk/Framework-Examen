package testutil

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
	"testing"
)

/** OpenCompetition crea únicamente una convocatoria sintética dentro del esquema aislado de pruebas. */
func OpenCompetition(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	var schema string
	if err := pool.QueryRow(ctx, `SELECT current_schema()`).Scan(&schema); err != nil || !(strings.HasPrefix(schema, "audit_review_test_") || strings.HasPrefix(schema, "submission_test_") || strings.HasPrefix(schema, "document_test_")) {
		t.Fatal("la fixture exige un esquema aislado", err)
	}
	id := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO competitions(id,code,title,opens_at,closes_at) VALUES($1,$2,'Prueba técnica',now()-interval '1 day',now()+interval '1 day')`, id, "TEST-"+id.String()); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO competition_controls(competition_id,approved,published,form_enabled,sectors,phases)
 VALUES($1,true,true,true,ARRAY['afiliados','jubilados','empleadores'],jsonb_build_array(jsonb_build_object('codigo','postulacion','titulo','Prueba','apertura',now()-interval '1 day','cierre',now()+interval '1 day','publicada',true,'habilitada',true)))`, id); err != nil {
		t.Fatal(err)
	}
	return id
}

/** SeedLegacy representa datos anteriores a T18; sólo omite sus ventanas durante la siembra de un esquema de pruebas. */
func SeedLegacy(t *testing.T, pool *pgxpool.Pool, seed func()) {
	t.Helper()
	ctx := context.Background()
	var schema string
	if err := pool.QueryRow(ctx, `SELECT current_schema()`).Scan(&schema); err != nil || !strings.HasPrefix(schema, "audit_review_test_") {
		t.Fatal("la siembra legada exige un esquema aislado", err)
	}
	if _, err := pool.Exec(ctx, `ALTER TABLE submissions DISABLE TRIGGER submissions_window;ALTER TABLE documents DISABLE TRIGGER documents_window`); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := pool.Exec(ctx, `ALTER TABLE submissions ENABLE TRIGGER submissions_window;ALTER TABLE documents ENABLE TRIGGER documents_window`); err != nil {
			t.Error(err)
		}
	}()
	seed()
}
