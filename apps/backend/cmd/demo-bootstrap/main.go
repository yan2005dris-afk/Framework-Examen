// Command demo-bootstrap crea únicamente un legado sintético antes de las migraciones nuevas.
package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/sql/migrations"
	"math/big"
	"net/url"
	"os"
	"time"
)

func check(e error) {
	if e != nil {
		panic("Bootstrap sintético incompleto: " + e.Error())
	}
}
func main() {
	u, e := url.Parse(os.Getenv("DATABASE_URL"))
	check(e)
	if os.Getenv("DEMO_CONFIRM") != "synthetic-demo" || u.Host != "localhost:15432" || u.Path != "/framework_synthetic_demo" {
		panic("Destino no autorizado")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	p, e := pgxpool.New(ctx, u.String())
	check(e)
	defer p.Close()
	var existing bool
	check(p.QueryRow(ctx, `SELECT to_regclass('public.goose_db_version') IS NOT NULL`).Scan(&existing))
	if existing {
		fmt.Println("Base existente: no se crea legado retrospectivo")
		return
	}
	check(migrations.RunTo(ctx, u.String(), 10))
	owner, _ := uuid.NewV7()
	id, _ := uuid.NewV7()
	n, e := rand.Int(rand.Reader, big.NewInt(700000))
	check(e)
	cedula := fmt.Sprintf("240%06d", 200000+n.Int64())
	sum := 0
	for i, c := range cedula {
		d := int(c - '0')
		if i%2 == 0 {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
	}
	cedula += fmt.Sprint((10 - sum%10) % 10)
	raw, _ := json.Marshal(map[string]interface{}{"origen": "synthetic-demo", "sinConvocatoria": true, "advertencia": "DATOS SINTÉTICOS - NO VÁLIDO"})
	canonical, e := submission.CanonicalContent(raw)
	check(e)
	tx, e := p.Begin(ctx)
	check(e)
	defer tx.Rollback(ctx)
	_, e = tx.Exec(ctx, `INSERT INTO applicants(id,cedula,nombres,apellidos,correo) VALUES($1,$2,'SINTETICO LEGADO','DATOS SINTÉTICOS - NO VÁLIDO',$3)`, owner, cedula, uuid.NewString()+"@synthetic.invalid")
	check(e)
	_, e = tx.Exec(ctx, `INSERT INTO submissions(id,applicant_id,code,status,content,hash,submitted_at) VALUES($1,$2,'SYNTHETIC-DEMO-SIN-CONVOCATORIA','submitted',$3,$4,clock_timestamp())`, id, owner, canonical, submission.HashContent(canonical))
	check(e)
	check(tx.Commit(ctx))
	check(migrations.Run(ctx, u.String()))
	fmt.Println("Legado synthetic-demo creado antes de restricciones; migrado sin alterar su sello ni inventar consentimiento")
}
