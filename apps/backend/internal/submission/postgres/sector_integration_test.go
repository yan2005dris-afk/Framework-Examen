package postgres

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// TestPostgresSectorColumnFollowsContent comprueba que el sector es una columna
// propia, calculada por PostgreSQL y restringida a las opciones del formulario.
func TestPostgresSectorColumnFollowsContent(t *testing.T) {
	repo, item, pool := integrationEnvironment(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	sector := func() *string {
		t.Helper()
		var value *string
		if err := pool.QueryRow(ctx, `SELECT sector FROM submissions WHERE id=$1`, item.ID).Scan(&value); err != nil {
			t.Fatal(err)
		}
		return value
	}

	if value := sector(); value != nil {
		t.Fatalf("sin sección 4 el sector debe ser NULL: %q", *value)
	}

	for _, want := range []string{"afiliados", "jubilados", ""} {
		if err := item.UpdateContent(json.RawMessage(`{"seccion4":{"sector":"`+want+`"}}`), time.Now().UTC()); err != nil {
			t.Fatal(err)
		}
		if err := repo.Update(ctx, item); err != nil {
			t.Fatalf("guardar sector %q: %v", want, err)
		}
		if value := sector(); value == nil || *value != want {
			t.Fatalf("sector esperado %q, obtenido %v", want, value)
		}
	}

	// Un valor fuera de definition.json lo rechaza la base aunque se salte el servicio:
	// primero la ventana de la convocatoria y, debajo, el CHECK de la columna.
	invalid := `UPDATE submissions SET content='{"seccion4":{"sector":"otro"}}' WHERE id=$1`
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

	// La columna es generada: no admite escrituras directas que la desincronicen del contenido.
	if _, err := pool.Exec(ctx, `UPDATE submissions SET sector='empleadores' WHERE id=$1`, item.ID); err == nil {
		t.Fatal("la columna generada no debe aceptar escrituras directas")
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
