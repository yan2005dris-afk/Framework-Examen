// Package download sirve capacidades revocables sin revelar firmas de almacenamiento al navegador.
package download

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/document"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/httpx"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/privacy"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/shared/domain"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var ErrDenied = submission.ErrForbidden
var ErrUnavailable = document.ErrStorageUnavailable

type Gate interface {
	Allow(context.Context, uuid.UUID, string) error
}
type Broker struct {
	next    document.Storage
	q       *sqlcgen.Queries
	gate    Gate
	auditor audit.Recorder
	public  string
	client  *http.Client
	slots   chan struct{}
}

/** New limita concurrencia y entrega URLs opacas; la firma S3 sólo se usa dentro del servidor. */
func New(q *sqlcgen.Queries, next document.Storage, gate Gate, recorder audit.Recorder, public string) (*Broker, error) {
	u, e := url.Parse(public)
	if e != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.RawQuery != "" || u.Fragment != "" || u.User != nil {
		return nil, ErrUnavailable
	}
	return &Broker{q: q, next: next, gate: gate, auditor: recorder, public: strings.TrimRight(public, "/"), client: &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, slots: make(chan struct{}, 8)}, nil
}
func (b *Broker) Save(ctx context.Context, key string, r io.Reader) error {
	return b.next.Save(ctx, key, r)
}
func (b *Broker) Delete(ctx context.Context, key string) error { return b.next.Delete(ctx, key) }
func (b *Broker) Ready(ctx context.Context) error {
	if r, ok := b.next.(interface{ Ready(context.Context) error }); ok {
		return r.Ready(ctx)
	}
	return nil
}
func digest(v string) string                                   { h := sha256.Sum256([]byte(v)); return hex.EncodeToString(h[:]) }
func lookupError(e error) error {
	if errors.Is(e, pgx.ErrNoRows) {
		return ErrDenied
	}
	return ErrUnavailable
}

func (b *Broker) authorize(ctx context.Context, key, session string, expected uuid.UUID) (sqlcgen.GetDownloadTargetRow, *identity.Applicant, error) {
	target, e := b.q.GetDownloadTarget(ctx, key)
	if e != nil {
		return target, nil, lookupError(e)
	}
	actor, e := b.q.GetDownloadSession(ctx, session)
	if e != nil {
		return target, nil, lookupError(e)
	}
	a := &identity.Applicant{ID: actor.ID, Role: actor.Role}
	if a.ID != expected {
		return target, nil, ErrDenied
	}
	if a.Role != identity.RoleApplicant && a.Role != identity.RoleAnalyst && a.Role != identity.RoleAdministrator {
		return target, nil, ErrDenied
	}
	if target.ApplicantID != a.ID {
		permission := identity.PermissionDocuments
		if target.Kind == "evaluacion" {
			permission = identity.PermissionGrades
		}
		if target.Status != "submitted" || !identity.HasPermission(a, permission) {
			return target, nil, ErrDenied
		}
	}
	if target.Status == "submitted" {
		if b.gate == nil {
			return target, nil, ErrDenied
		}
		if e = b.gate.Allow(ctx, target.SubmissionID, "documentos"); e != nil {
			if errors.Is(e, privacy.ErrDenied) {
				return target, nil, ErrDenied
			}
			return target, nil, ErrUnavailable
		}
	}
	if target.Kind == "evaluacion" && target.ApplicantID == a.ID {
		evidence, e := b.q.GetEvaluationEvidence(ctx, target.ID)
		if e != nil {
			return target, nil, lookupError(e)
		}
		if !evidence.ApplicantUpload && !evidence.Published {
			return target, nil, ErrDenied
		}
	}
	return target, a, nil
}

/** PresignGet crea una capacidad de cinco minutos ligada a la sesión validada; no devuelve una URL S3. */
func (b *Broker) PresignGet(ctx context.Context, key, name string, ttl time.Duration) (string, error) {
	if ttl != 5*time.Minute {
		return "", ErrDenied
	}
	actor := domain.AuditFromContext(ctx).ActorID
	session := identity.DownloadSession(ctx)
	if actor == uuid.Nil || session == "" {
		return "", ErrDenied
	}
	target, a, e := b.authorize(ctx, key, session, actor)
	if e != nil {
		return "", e
	}
	version, e := b.q.GetDownloadPrivacyVersion(ctx, target.SubmissionID)
	if e != nil {
		return "", ErrUnavailable
	}
	var token [32]byte
	if _, e := rand.Read(token[:]); e != nil {
		return "", ErrUnavailable
	}
	raw := base64.RawURLEncoding.EncodeToString(token[:])
	if e := b.q.CreateDownloadGrant(ctx, sqlcgen.CreateDownloadGrantParams{TokenHash: digest(raw), ActorID: actor, SessionHash: session, StorageKey: key, ActorRole: a.Role, PolicyVersion: version}); e != nil {
		return "", ErrUnavailable
	}
	return b.public + "/" + raw, nil
}

/** ServeHTTP revalida capacidad, sesión, rol, propiedad y privacidad antes de liberar bytes; no redirige al bucket. */
func (b *Broker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, private, max-age=0")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	deny := func(e error) {
		status, code := 403, "descarga_revocada"
		if errors.Is(e, ErrUnavailable) {
			status, code = 503, "descarga_no_disponible"
		}
		httpx.RespondError(w, r, status, code, "No se puede acceder al documento.")
	}
	if r.Method != "GET" && r.Method != "HEAD" {
		w.WriteHeader(405)
		return
	}
	if r.Header.Get("Range") != "" {
		w.WriteHeader(416)
		return
	}
	raw := strings.TrimPrefix(r.URL.Path, "/downloads/")
	decoded, e := base64.RawURLEncoding.DecodeString(raw)
	if e != nil || len(decoded) != 32 || len(r.URL.Query()) != 0 {
		deny(ErrDenied)
		return
	}
	select {
	case b.slots <- struct{}{}:
		defer func() { <-b.slots }()
	default:
		deny(ErrUnavailable)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	grant, e := b.q.GetLiveDownloadGrant(ctx, digest(raw))
	if e != nil {
		deny(lookupError(e))
		return
	}
	target, a, e := b.authorize(ctx, grant.StorageKey, grant.SessionHash, grant.ActorID)
	if e != nil {
		deny(e)
		return
	}
	version, e := b.q.GetDownloadPrivacyVersion(ctx, target.SubmissionID)
	if e != nil {
		deny(ErrUnavailable)
		return
	}
	if a.Role != grant.ActorRole || version != grant.PolicyVersion {
		deny(ErrDenied)
		return
	}
	var data []byte
	if r.Method == "GET" {
		signed, e := b.next.PresignGet(ctx, grant.StorageKey, target.Name, time.Minute)
		if e != nil {
			deny(ErrUnavailable)
			return
		}
		req, e := http.NewRequestWithContext(ctx, "GET", signed, nil)
		if e != nil {
			deny(ErrUnavailable)
			return
		}
		resp, e := b.client.Do(req)
		if e != nil {
			deny(ErrUnavailable)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			deny(ErrUnavailable)
			return
		}
		data, e = io.ReadAll(io.LimitReader(resp.Body, 4<<20+1))
		if e != nil || len(data) > 4<<20 || int64(len(data)) != target.SizeBytes || !bytes.HasPrefix(data, []byte("%PDF-")) {
			deny(ErrUnavailable)
			return
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != target.Sha256 {
			deny(ErrUnavailable)
			return
		}
	}
	// Puede haberse revocado durante la lectura del proveedor: volver a consultar antes del primer byte.
	target, a, e = b.authorize(ctx, grant.StorageKey, grant.SessionHash, grant.ActorID)
	if e != nil {
		deny(e)
		return
	}
	if _, e = b.q.GetLiveDownloadGrant(ctx, digest(raw)); e != nil {
		deny(lookupError(e))
		return
	}
	version, e = b.q.GetDownloadPrivacyVersion(ctx, target.SubmissionID)
	if e != nil {
		deny(ErrUnavailable)
		return
	}
	if a.Role != grant.ActorRole || version != grant.PolicyVersion {
		deny(ErrDenied)
		return
	}
	ctx = domain.WithAuditActor(ctx, a.ID, a.Role)
	requestID, _ := uuid.Parse(httpx.GetRequestID(ctx))
	if e = b.q.CreateDownloadAccess(ctx, sqlcgen.CreateDownloadAccessParams{ActorID: a.ID, ActorRole: a.Role, SubmissionID: target.SubmissionID, ResourceID: target.ID, RequestID: pgtype.UUID{Bytes: requestID, Valid: requestID != uuid.Nil}}); e != nil {
		deny(ErrUnavailable)
		return
	}
	if a.Role != identity.RoleApplicant {
		if e = audit.Record(ctx, b.auditor, audit.Event{ActorID: a.ID, Role: a.Role, SubmissionID: target.SubmissionID, ResourceID: target.ID, ResourceType: "documento", Action: "documento.descarga_verificada", CreatedAt: time.Now().UTC()}, nil); e != nil {
			deny(ErrUnavailable)
			return
		}
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": target.Name}))
	w.WriteHeader(200)
	if r.Method == "GET" {
		_, _ = w.Write(data)
	}
}
