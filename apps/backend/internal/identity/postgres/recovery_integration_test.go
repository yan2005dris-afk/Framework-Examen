package postgres

import (
	"context"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/db"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/testutil"
	"testing"
	"time"
)

func TestPostgresRecoveryRollsBackPasswordIfRevocationFailsAndRejectsStaleLogin(t *testing.T) {
	pool := testutil.Postgres(t, "AUTHORIZATION_TEST_DATABASE_URL")
	ctx := context.Background()
	id := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO applicants(id,cedula,nombres,apellidos,correo,password_hash) VALUES($1,'1710034008','Persona','Prueba','test@example.test','anterior')`, id); err != nil {
		t.Fatal(err)
	}
	repo := NewRepository(sqlcgen.New(pool), db.NewTxManager(pool))
	session := &identity.Session{ID: uuid.New(), ApplicantID: id, TokenHash: identity.HashSessionToken("sesion-prueba"), ExpiresAt: time.Now().Add(time.Hour)}
	if err := repo.CreateSession(ctx, session); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `CREATE FUNCTION reject_session_delete() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'fallo simulado'; END; $$;
 CREATE TRIGGER reject_delete BEFORE DELETE ON sessions FOR EACH ROW EXECUTE FUNCTION reject_session_delete()`); err != nil {
		t.Fatal(err)
	}
	if err := repo.ResetPasswordAndSessions(ctx, id, "nueva"); err == nil {
		t.Fatal("se ignoró el fallo al revocar sesiones")
	}
	var hash string
	if err := pool.QueryRow(ctx, `SELECT password_hash FROM applicants WHERE id=$1`, id).Scan(&hash); err != nil || hash != "anterior" {
		t.Fatal("clave cambió sin revocar sesiones", err)
	}
	if _, err := pool.Exec(ctx, `DROP TRIGGER reject_delete ON sessions`); err != nil {
		t.Fatal(err)
	}
	if err := repo.ResetPasswordAndSessions(ctx, id, "nueva"); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM sessions WHERE applicant_id=$1`, id).Scan(&count); err != nil || count != 0 {
		t.Fatal("sesiones antiguas aún activas", err)
	}
	if err := repo.CreateAuthenticatedSession(ctx, session, "anterior"); err != identity.ErrInvalidCredentials {
		t.Fatal("login anterior a la revocación aceptado", err)
	}
}
