package postgres

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit"
	auditpostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit/postgres"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/db"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func authorizationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("AUTHORIZATION_TEST_DATABASE_URL")
	if url == "" {
		url = os.Getenv("TEST_DATABASE_URL")
	}
	if url == "" {
		url = os.Getenv("DATABASE_URL")
	}
	if url == "" {
		t.Skip("defina AUTHORIZATION_TEST_DATABASE_URL o TEST_DATABASE_URL para probar autorización en PostgreSQL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal("no se pudo abrir la base de pruebas")
	}
	t.Cleanup(admin.Close)
	schema := "authorization_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	quoted := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+quoted); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := admin.Exec(ctx, "DROP SCHEMA "+quoted+" CASCADE"); err != nil {
			t.Error(err)
		}
	})
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal("URL de pruebas inválida")
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal("no se pudo abrir el esquema aislado")
	}
	t.Cleanup(pool.Close)
	paths, err := filepath.Glob(filepath.Join("..", "..", "..", "sql", "migrations", "*.sql"))
	if err != nil || len(paths) == 0 {
		t.Fatal("no se encontraron migraciones")
	}
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		up := strings.Split(strings.Split(string(raw), "-- +goose Up")[1], "-- +goose Down")[0]
		if _, err := pool.Exec(ctx, up); err != nil {
			t.Fatalf("migración %s: %v", filepath.Base(path), err)
		}
	}
	return pool
}

func TestPostgresRoleChangesAreAtomicAuditedAndImmediatelyEffective(t *testing.T) {
	pool := authorizationPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	actor, target := uuid.New(), uuid.New()
	for index, item := range []struct {
		id   uuid.UUID
		role string
	}{{actor, identity.RoleAdministrator}, {target, identity.RoleApplicant}} {
		if _, err := pool.Exec(ctx, `INSERT INTO applicants (id, cedula, nombres, apellidos, correo, role) VALUES ($1,$2,'Cuenta','Prueba',$3,$4)`, item.id, []string{"1710034008", "0923456789"}[index], []string{"admin@example.test", "cuenta@example.test"}[index], item.role); err != nil {
			t.Fatal(err)
		}
	}
	queries := sqlcgen.New(pool)
	repo := NewRepository(queries, db.NewTxManager(pool))
	if err := repo.CreateSession(ctx, &identity.Session{ID: uuid.New(), ApplicantID: target, TokenHash: identity.HashSessionToken("cookie-prueba"), ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ChangeRole(ctx, actor, target, identity.RoleAnalyst, "Asignación del concurso"); err != nil {
		t.Fatal(err)
	}
	var previous, next, reason string
	var recordedActor uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT previous_role,new_role,reason,actor_id FROM applicant_role_events WHERE applicant_id=$1`, target).Scan(&previous, &next, &reason, &recordedActor); err != nil {
		t.Fatal(err)
	}
	if previous != identity.RoleApplicant || next != identity.RoleAnalyst || recordedActor != actor || reason != "Asignación del concurso" {
		t.Fatal("cambio sin historial correcto")
	}
	_, profile, err := repo.GetWithApplicant(ctx, identity.HashSessionToken("cookie-prueba"))
	if err != nil || profile.Role != identity.RoleAnalyst {
		t.Fatal("la cookie existente no refleja el rol vigente")
	}
	if _, err := repo.ChangeRole(ctx, target, actor, identity.RoleApplicant, "Escalamiento"); !errors.Is(err, identity.ErrRoleForbidden) {
		t.Fatalf("analista alteró permisos: %v", err)
	}
	// Forzar el fallo de auditoría debe revertir el UPDATE del rol.
	if _, err := pool.Exec(ctx, `CREATE FUNCTION reject_role_event() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'fallo forzado de auditoría'; END; $$; CREATE TRIGGER reject_role_event BEFORE INSERT ON applicant_role_events FOR EACH ROW EXECUTE FUNCTION reject_role_event()`); err != nil {
		t.Fatal(err)
	}
	if item, err := repo.ChangeRole(ctx, actor, target, identity.RoleAdministrator, "Debe revertirse"); err == nil || item != nil {
		t.Fatal("se confirmó un cambio sin auditoría")
	}
	var current string
	if err := pool.QueryRow(ctx, `SELECT role FROM applicants WHERE id=$1`, target).Scan(&current); err != nil || current != identity.RoleAnalyst {
		t.Fatal("el fallo dejó un rol parcialmente cambiado")
	}
	if _, err := pool.Exec(ctx, `DROP TRIGGER reject_role_event ON applicant_role_events; DROP FUNCTION reject_role_event()`); err != nil {
		t.Fatal(err)
	}
	// Una revocación concurrente invalida la autorización, incluso con actor resuelto antes.
	if _, err := pool.Exec(ctx, `UPDATE applicants SET role='postulante' WHERE id=$1`, actor); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ChangeRole(ctx, actor, target, identity.RoleAdministrator, "Sesión antigua"); !errors.Is(err, identity.ErrRoleForbidden) {
		t.Fatal("se usaron privilegios ya revocados")
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM applicant_role_events`).Scan(&count); err != nil || count != 2 {
		t.Fatalf("se perdió auditoría SQL o se registró el rollback: %d %v", count, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE applicants SET role='desconocido' WHERE id=$1`, target); err == nil {
		t.Fatal("la base admitió un cuarto rol")
	}
	recorder := auditpostgres.NewRepository(queries)
	if err := recorder.Record(ctx, audit.Event{ActorID: target, Role: identity.RoleAnalyst, Action: "formulario.consultado", ResourceID: uuid.New(), ResourceType: "postulacion", Result: "permitido", CreatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"applicant_role_events", "administrative_access_events"} {
		for _, statement := range []string{"DELETE FROM " + table, "UPDATE " + table + " SET id=id", "TRUNCATE " + table} {
			if _, err := pool.Exec(ctx, statement); err == nil {
				t.Fatalf("historial alterable: %s", statement)
			}
		}
	}
}
