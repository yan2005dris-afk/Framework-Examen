package postgres

import (
	"context"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"testing"
	"time"
)

func TestPurgeSessionsPreservesActiveAndRecentExpired(t *testing.T) {
	p := authorizationPool(t)
	ctx := context.Background()
	owner := uuid.New()
	_, err := p.Exec(ctx, `INSERT INTO applicants(id,cedula,nombres,apellidos,correo) VALUES($1,'1710034008','SINTETICO','NO VALIDO','sessions@synthetic.invalid')`, owner)
	if err != nil {
		t.Fatal(err)
	}
	for _, hours := range []int{-31 * 24, -29 * 24, 24} {
		_, err = p.Exec(ctx, `INSERT INTO sessions(id,applicant_id,token_hash,expires_at) VALUES($1,$2,$3,$4)`, uuid.New(), owner, uuid.NewString(), time.Now().Add(time.Duration(hours)*time.Hour))
		if err != nil {
			t.Fatal(err)
		}
	}
	q := sqlcgen.New(p)
	n, err := q.PurgeExpiredSessionBatch(ctx)
	if err != nil || n != 1 {
		t.Fatalf("purga=%d %v", n, err)
	}
	n, err = q.PurgeExpiredSessionBatch(ctx)
	if err != nil || n != 0 {
		t.Fatalf("no idempotente: %d %v", n, err)
	}
	var remaining int
	if err = p.QueryRow(ctx, `SELECT count(*) FROM sessions WHERE applicant_id=$1`, owner).Scan(&remaining); err != nil || remaining != 2 {
		t.Fatal("se borró una sesión activa o reciente", err)
	}
}
