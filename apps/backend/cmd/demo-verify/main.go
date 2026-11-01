// Command demo-verify comprueba un respaldo sintético sin emitir documentos ni credenciales.
package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/storage"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

func must(e error) {
	if e != nil {
		panic("No se verificó el respaldo sintético: " + e.Error())
	}
}
func main() {
	u, e := url.Parse(os.Getenv("DATABASE_URL"))
	must(e)
	if os.Getenv("DEMO_CONFIRM") != "synthetic-demo" || u.Host != "localhost:15432" || !(u.Path == "/framework_synthetic_demo" || strings.HasPrefix(u.Path, "/synthetic_restore_")) {
		panic("Destino de demostración no autorizado")
	}
	endpoint := os.Getenv("VERIFY_S3_ENDPOINT")
	if endpoint != "http://localhost:19000/synthetic-demo" && endpoint != "http://localhost:19010/synthetic-demo" {
		panic("Endpoint ajeno a QA")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	p, e := pgxpool.New(ctx, u.String())
	must(e)
	defer p.Close()
	s, e := storage.NewS3(storage.S3Config{Endpoint: endpoint, Region: "us-east-1", AccessKey: os.Getenv("DEMO_S3_ACCESS_KEY"), SecretKey: os.Getenv("DEMO_S3_SECRET_KEY")})
	must(e)
	result := map[string]interface{}{"origin": "synthetic-demo", "warning": "DATOS SINTÉTICOS - NO VÁLIDO"}
	tables := []string{}
	rows, e := p.Query(ctx, `SELECT tablename FROM pg_tables WHERE schemaname='public' ORDER BY tablename`)
	must(e)
	for rows.Next() {
		var t string
		must(rows.Scan(&t))
		tables = append(tables, t)
	}
	must(rows.Err())
	rows.Close()
	hashes := map[string]string{}
	counts := map[string]int{}
	for _, table := range tables {
		q := `SELECT to_jsonb(t)::text FROM ` + pgx.Identifier{table}.Sanitize() + ` t ORDER BY to_jsonb(t)::text`
		rs, e := p.Query(ctx, q)
		must(e)
		h := sha256.New()
		for rs.Next() {
			var v string
			must(rs.Scan(&v))
			h.Write([]byte(v))
			h.Write([]byte{10})
			counts[table]++
		}
		must(rs.Err())
		rs.Close()
		hashes[table] = hex.EncodeToString(h.Sum(nil))
	}
	result["tableHashes"] = hashes
	result["tableCounts"] = counts
	var version int64
	must(p.QueryRow(ctx, `SELECT max(version_id) FROM goose_db_version WHERE is_applied`).Scan(&version))
	if version < 19 {
		panic("Migraciones incompletas")
	}
	result["migrationVersion"] = version
	rs, e := p.Query(ctx, `SELECT id,applicant_id,code,content,hash,submitted_at,competition_id FROM submissions WHERE status='submitted'`)
	must(e)
	sealed := 0
	for rs.Next() {
		var item submission.Submission
		must(rs.Scan(&item.ID, &item.ApplicantID, &item.Code, &item.Content, &item.Hash, &item.SubmittedAt, &item.CompetitionID))
		item.Status = submission.StatusSubmitted
		must(submission.ValidateSeal(&item))
		sealed++
	}
	must(rs.Err())
	rs.Close()
	result["verifiedSeals"] = sealed
	reports, e := p.Query(ctx, `SELECT pdf,sha256 FROM exam_reports ORDER BY attempt_id`)
	must(e)
	reportCount := 0
	for reports.Next() {
		var data []byte
		var want string
		must(reports.Scan(&data, &want))
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != want {
			panic("Hash de PDF de examen incorrecto")
		}
		reportCount++
	}
	must(reports.Err())
	reports.Close()
	result["verifiedExamPDFs"] = reportCount
	key, e := base64.StdEncoding.DecodeString(os.Getenv("MAIL_OUTBOX_KEY"))
	must(e)
	block, e := aes.NewCipher(key)
	must(e)
	box, e := cipher.NewGCM(block)
	must(e)
	pending, e := p.Query(ctx, `SELECT id::text,ciphertext FROM mail_outbox WHERE octet_length(ciphertext)>0 ORDER BY id`)
	must(e)
	payloadCount := 0
	for pending.Next() {
		var id string
		var data []byte
		must(pending.Scan(&id, &data))
		if len(data) < box.NonceSize() {
			panic("Carga cifrada incompleta")
		}
		raw, e := box.Open(nil, data[:box.NonceSize()], data[box.NonceSize():], []byte(id))
		must(e)
		if !json.Valid(raw) {
			panic("Carga de correo no recuperable")
		}
		payloadCount++
	}
	must(pending.Err())
	pending.Close()
	result["decryptedOutboxPayloads"] = payloadCount

	objects, e := p.Query(ctx, `SELECT storage_key,sha256 FROM documents UNION ALL SELECT storage_key,sha256 FROM evaluation_evidence ORDER BY 1`)
	must(e)
	verified := 0
	objectHash := sha256.New()
	client := &http.Client{Timeout: 15 * time.Second}
	for objects.Next() {
		var key, want string
		must(objects.Scan(&key, &want))
		signed, e := s.PresignGet(ctx, key, "synthetic-demo.pdf", 5*time.Minute)
		must(e)
		r, e := http.NewRequestWithContext(ctx, "GET", signed, nil)
		must(e)
		response, e := client.Do(r)
		if e != nil {
			panic("Objeto no disponible")
		}
		if response.StatusCode != 200 {
			response.Body.Close()
			panic("Objeto no recuperado")
		}
		h := sha256.New()
		_, e = io.Copy(h, io.LimitReader(response.Body, 10<<20))
		response.Body.Close()
		must(e)
		got := hex.EncodeToString(h.Sum(nil))
		if got != want {
			panic("SHA256 de objeto no coincide")
		}
		objectHash.Write([]byte(key + ":" + got + "\n"))
		verified++
	}
	must(objects.Err())
	objects.Close()
	result["verifiedObjects"] = verified
	result["objectsManifestHash"] = hex.EncodeToString(objectHash.Sum(nil))
	raw, e := json.MarshalIndent(result, "", "  ")
	must(e)
	path := os.Getenv("VERIFY_OUTPUT")
	if path == "" {
		panic("VERIFY_OUTPUT requerido")
	}
	must(os.WriteFile(path, raw, 0600))
	fmt.Printf("Respaldo verificado: migración %d, %d sellos, %d objetos SHA-256; auditoría y outbox incluidos.\n", version, sealed, verified)
}
