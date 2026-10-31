package document

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"
)

type sessionRepo struct {
	identity.SessionRepository
	owner uuid.UUID
}

func (s sessionRepo) GetWithApplicant(_ context.Context, hash string) (*identity.Session, *identity.Applicant, error) {
	id := s.owner
	if hash == identity.HashSessionToken("other") {
		id = uuid.New()
	} else if hash != identity.HashSessionToken("owner") {
		return nil, nil, identity.ErrSessionExpired
	}
	return &identity.Session{ApplicantID: id, ExpiresAt: time.Now().Add(time.Hour)}, &identity.Applicant{ID: id}, nil
}

func documentHTTPFixture(t *testing.T) (http.Handler, *memoryRepo) {
	s, repo, _, _ := fixture(t)
	router := chi.NewRouter()
	auth := identity.NewService(identity.ServiceDependencies{Sessions: sessionRepo{owner: repo.item.ApplicantID}})
	router.Use(identity.RequireSession(auth))
	// Montar las rutas reales de ambos contextos detecta conflictos del árbol de chi.
	submission.NewHandler(submission.NewService(nil)).RegisterRoutes(router)
	NewHandler(s).RegisterRoutes(router)
	return router, repo
}

func multipartRequest(t *testing.T, path, token, mimeType, data string, section string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("section", section); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("recordId", "titulo-1"); err != nil {
		t.Fatal(err)
	}
	part, err := writer.CreatePart(textproto.MIMEHeader{"Content-Disposition": {`form-data; name="file"; filename="titulo.pdf"`}, "Content-Type": {mimeType}})
	if err != nil {
		t.Fatal(err)
	}
	part.Write([]byte(data))
	writer.Close()
	r := httptest.NewRequest("POST", path, &body)
	r.Header.Set("Content-Type", writer.FormDataContentType())
	if token != "" {
		r.AddCookie(&http.Cookie{Name: "session", Value: token})
	}
	return r
}

func TestDocumentHTTPUploadAuthorizationValidationAndDownload(t *testing.T) {
	for _, tc := range []struct {
		token, mime, data, section string
		status                     int
		code                       string
	}{
		{"", "application/pdf", "%PDF-1.7", "5", 401, "no_autenticado"},
		{"other", "application/pdf", "%PDF-1.7", "5", 403, "acceso_denegado"},
		{"owner", "image/png", "%PDF-1.7", "5", 415, "tipo_no_soportado"},
		{"owner", "application/pdf", "texto falso", "5", 415, "tipo_no_soportado"},
		{"owner", "application/pdf", "%PDF-1.7", "1", 400, "seccion_invalida"},
		{"owner", "application/pdf", "%PDF-1.7", "5", 201, ""},
	} {
		t.Run(tc.code+tc.token+tc.mime, func(t *testing.T) {
			router, repo := documentHTTPFixture(t)
			path := "/api/v1/submissions/" + repo.item.ID.String() + "/documents"
			r := multipartRequest(t, path, tc.token, tc.mime, tc.data, tc.section)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			if w.Code != tc.status || tc.code != "" && !strings.Contains(w.Body.String(), tc.code) {
				t.Fatalf("esperado %d %s, obtenido %d %s", tc.status, tc.code, w.Code, w.Body.String())
			}
			if tc.status != 201 {
				return
			}
			var d Document
			if err := json.Unmarshal(w.Body.Bytes(), &d); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(w.Body.String(), repo.docs[d.ID].StorageKey) {
				t.Fatal("expuso clave de almacenamiento")
			}
			for _, mp := range [][2]string{{"GET", path}, {"GET", "/api/v1/documents/" + d.ID.String() + "/download"}, {"DELETE", "/api/v1/documents/" + d.ID.String()}} {
				req := httptest.NewRequest(mp[0], mp[1], nil)
				req.AddCookie(&http.Cookie{Name: "session", Value: "owner"})
				resp := httptest.NewRecorder()
				router.ServeHTTP(resp, req)
				want := 200
				if mp[0] == "DELETE" {
					want = 204
				}
				if resp.Code != want {
					t.Fatalf("%s %s: %d %s", mp[0], mp[1], resp.Code, resp.Body)
				}
			}
		})
	}
}

func TestDocumentHTTPRejectsMalformedAndChunkedOversizeBodies(t *testing.T) {
	router, repo := documentHTTPFixture(t)
	path := "/api/v1/submissions/" + repo.item.ID.String() + "/documents"
	r := httptest.NewRequest("POST", path, strings.NewReader("malformed"))
	r.Header.Set("Content-Type", "multipart/form-data; boundary=none")
	r.AddCookie(&http.Cookie{Name: "session", Value: "owner"})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != 400 {
		t.Fatalf("multipart inválido: %d", w.Code)
	}
	r = multipartRequest(t, path, "owner", "application/pdf", "%PDF-1.7"+strings.Repeat("x", int(MaxOrganizationBytes)+65536), "5")
	r.ContentLength = -1
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != 413 {
		t.Fatalf("chunked excesivo: %d %s", w.Code, w.Body)
	}
	repo.item.Status = submission.StatusSubmitted
	r = multipartRequest(t, path, "owner", "application/pdf", "%PDF-1.7", "5")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != 409 {
		t.Fatalf("carga tras sello: %d", w.Code)
	}
}

func TestDocumentMultipartFieldsCannotBeOverriddenByQuery(t *testing.T) {
	router, repo := documentHTTPFixture(t)
	path := "/api/v1/submissions/" + repo.item.ID.String() + "/documents?section=9&recordId=otro"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, multipartRequest(t, path, "owner", "application/pdf", "%PDF-1.7", "5"))
	if w.Code != 201 {
		t.Fatalf("se usaron campos de la query: %d %s", w.Code, w.Body)
	}
}
