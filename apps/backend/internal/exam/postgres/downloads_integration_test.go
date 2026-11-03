package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/download"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/privacy"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/shared/domain"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type downloadSource struct{ url string }

func (s downloadSource) Save(context.Context, string, io.Reader) error { return nil }
func (s downloadSource) Delete(context.Context, string) error          { return nil }
func (s downloadSource) PresignGet(context.Context, string, string, time.Duration) (string, error) {
	return s.url, nil
}

func TestDownloadsRevokeIssuedCapabilitiesImmediately(t *testing.T) {
	f := newFixture(t, 60)
	ctx := context.Background()
	key := "synthetic/evidence.pdf"
	data := []byte("%PDF-1.7\nSINTETICO NO VALIDO\n%%EOF")
	hash := sha256.Sum256(data)
	var calls atomic.Int32
	var mode atomic.Int32
	var revokeFailed atomic.Bool
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		switch mode.Load() {
		case 1:
			_, _ = w.Write([]byte("%PDF-1.7\nALTERADO SIN AUTORIZACION\n%%EOF"))
			return
		case 2:
			// Simular que el cierre de sesión se confirma mientras S3 todavía responde.
			if e := f.q.DeleteSessionsByApplicantID(r.Context(), f.owner.ID); e != nil {
				revokeFailed.Store(true)
			}
		}
		_, _ = w.Write(data)
	}))
	defer upstream.Close()
	doc := uuid.New()
	_, e := f.pool.Exec(ctx, `INSERT INTO evaluation_evidence(id,submission_id,applicant_id,actor_id,applicant_upload,name,size_bytes,sha256,storage_key,created_at)VALUES($1,$2,$3,$4,true,'synthetic.pdf',$5,$6,$7,clock_timestamp())`, doc, f.sub, f.owner.ID, f.owner.ID, len(data), hex.EncodeToString(hash[:]), key)
	if e != nil {
		t.Fatal(e)
	}
	session := identity.HashSessionToken("synthetic-session")
	_, e = f.pool.Exec(ctx, `INSERT INTO sessions(id,applicant_id,token_hash,expires_at)VALUES($1,$2,$3,clock_timestamp()+interval '1 hour')`, uuid.New(), f.owner.ID, session)
	if e != nil {
		t.Fatal(e)
	}
	b, e := download.New(f.q, downloadSource{upstream.URL}, f.privacy, f.auditor, "http://download.test/downloads")
	if e != nil {
		t.Fatal(e)
	}
	principal := identity.WithDownloadSession(domain.WithAuditActor(ctx, f.owner.ID, f.owner.Role), session)
	issue := func() string {
		t.Helper()
		link, e := b.PresignGet(principal, key, "synthetic.pdf", 5*time.Minute)
		if e != nil {
			t.Fatal(e)
		}
		if strings.Contains(link, key) || strings.Contains(link, "X-Amz-") || strings.Contains(link, session) {
			t.Fatal("capacidad reveló almacenamiento o sesión")
		}
		u, _ := url.Parse(link)
		return u.Path
	}
	get := func(path string, want int) {
		t.Helper()
		w := httptest.NewRecorder()
		b.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != want {
			t.Fatalf("descarga: esperado %d, recibido %d", want, w.Code)
		}
		if want != 200 && strings.Contains(w.Body.String(), "%PDF") {
			t.Fatal("filtró documento denegado")
		}
		if w.Header().Get("Cache-Control") == "" || w.Header().Get("Location") != "" {
			t.Fatal("cache o redirección no permitida")
		}
	}
	first := issue()
	get(first, 200)
	before := calls.Load()
	if e = f.privacy.Block(ctx, f.admin, f.period, privacy.BlockInput{PreviousVersion: f.policy.ControlVersion, Blocked: true, Reason: "QA revocación inmediata"}); e != nil {
		t.Fatal(e)
	}
	get(first, 403)
	if calls.Load() != before {
		t.Fatal("contactó S3 tras revocación")
	}
	current, e := f.privacy.Current(ctx, f.period)
	if e != nil {
		t.Fatal(e)
	}
	if e = f.privacy.Block(ctx, f.admin, f.period, privacy.BlockInput{PreviousVersion: current.ControlVersion, Blocked: false, Reason: "QA reapertura"}); e != nil {
		t.Fatal(e)
	}
	get(first, 403)
	expired := issue()
	token := strings.TrimPrefix(expired, "/downloads/")
	tokenHash := sha256.Sum256([]byte(token))
	_, e = f.pool.Exec(ctx, `UPDATE download_grants SET expires_at=clock_timestamp()-interval '1 second' WHERE token_hash=$1`, hex.EncodeToString(tokenHash[:]))
	if e != nil {
		t.Fatal(e)
	}
	get(expired, 403)
	second := issue()
	get(second, 200)
	_, e = f.pool.Exec(ctx, `CREATE FUNCTION reject_download_audit()RETURNS trigger LANGUAGE plpgsql AS $$BEGIN RAISE EXCEPTION 'QA';END$$;CREATE TRIGGER reject_download_audit BEFORE INSERT ON download_access_events FOR EACH ROW EXECUTE FUNCTION reject_download_audit()`)
	if e != nil {
		t.Fatal(e)
	}
	get(second, 503)
	_, e = f.pool.Exec(ctx, `DROP TRIGGER reject_download_audit ON download_access_events;DROP FUNCTION reject_download_audit()`)
	if e != nil {
		t.Fatal(e)
	}
	mode.Store(1)
	get(second, 503)
	mode.Store(0)
	get(second, 200)
	mode.Store(2)
	get(second, 403)
	if revokeFailed.Load() {
		t.Fatal("falló la revocación concurrente de prueba")
	}
	mode.Store(0)
	get(second, 403)
	// La capacidad no es una sesión y tampoco admite manipulación ni tokens en query.
	get("/downloads/not-valid", 403)
	get(second+"?ignored=1", 403)
}
