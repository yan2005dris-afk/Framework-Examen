package storage

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"mime"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

/** SignedLocal firma capacidades de lectura para un servidor separado de la API. */
type SignedLocal struct {
	*Local
	publicURL *url.URL
	secret    []byte
	now       func() time.Time
}

/** NewSignedLocal exige una clave estable compartida por API y servidor de objetos.
 * @returns {SignedLocal, error} Adaptador local con firma HMAC-SHA256.
 */
func NewSignedLocal(base, publicURL, secret string) (*SignedLocal, error) {
	u, err := url.Parse(publicURL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, errors.New("STORAGE_PUBLIC_URL debe ser una URL HTTP válida")
	}
	if len(secret) < 32 {
		return nil, errors.New("STORAGE_SIGNING_KEY requiere al menos 32 bytes")
	}
	local, err := NewLocal(base)
	if err != nil {
		return nil, err
	}
	return &SignedLocal{Local: local, publicURL: u, secret: []byte(secret), now: time.Now}, nil
}

/** PresignGet crea un enlace de lectura vinculado a clave, nombre y vencimiento.
 * @returns {string, error} URL temporal; nunca una ruta del disco.
 */
func (s *SignedLocal) PresignGet(ctx context.Context, key, name string, ttl time.Duration) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if ttl < time.Second || ttl > 15*time.Minute {
		return "", errors.New("duración de descarga no válida")
	}
	if _, err := s.resolvePath(key); err != nil {
		return "", err
	}
	u := *s.publicURL
	u.Path = strings.TrimRight(u.Path, "/") + "/objects/" + key
	q := url.Values{"expires": {strconv.FormatInt(s.now().Add(ttl).Unix(), 10)}, "name": {name}}
	q.Set("signature", s.signature(key, q.Get("name"), q.Get("expires")))
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (s *SignedLocal) signature(key, name, expires string) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte("GET\n" + key + "\n" + url.QueryEscape(name) + "\n" + expires))
	return hex.EncodeToString(mac.Sum(nil))
}

/** ServeHTTP sirve exclusivamente GET/HEAD firmados desde el proceso de objetos.
 * @param {http.Request} r - Capacidad temporal; no registrar su query string.
 */
func (s *SignedLocal) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.Method != "GET" && r.Method != "HEAD" {
		w.WriteHeader(405)
		return
	}
	prefix := strings.TrimRight(s.publicURL.Path, "/") + "/objects/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		http.NotFound(w, r)
		return
	}
	key := strings.TrimPrefix(r.URL.Path, prefix)
	q := r.URL.Query()
	expires, err := strconv.ParseInt(q.Get("expires"), 10, 64)
	now := s.now().Unix()
	provided, decodeErr := hex.DecodeString(q.Get("signature"))
	expected, _ := hex.DecodeString(s.signature(key, q.Get("name"), q.Get("expires")))
	if err != nil || decodeErr != nil || expires <= now || expires > now+900 || !hmac.Equal(provided, expected) {
		w.WriteHeader(403)
		return
	}
	file, err := s.Get(r.Context(), key)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()
	f, ok := file.(*os.File)
	if !ok {
		w.WriteHeader(500)
		return
	}
	stat, err := f.Stat()
	if err != nil || !stat.Mode().IsRegular() {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": q.Get("name")}))
	http.ServeContent(w, r, "document.pdf", stat.ModTime(), f)
}
