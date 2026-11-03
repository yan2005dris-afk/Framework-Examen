package download

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDownloadConfigurationRejectsAmbiguousPublicURLs(t *testing.T) {
	for _, public := range []string{"", "/downloads", "file:///downloads", "https://user:secret@example.test/downloads", "https://example.test/downloads?token=x", "https://example.test/downloads#fragment"} {
		if _, err := New(nil, nil, nil, nil, public); err != ErrUnavailable {
			t.Fatal("aceptó configuración insegura")
		}
	}
}

func TestDownloadRejectsInvalidRequestsBeforeStorageOrDatabase(t *testing.T) {
	b, err := New(nil, nil, nil, nil, "https://example.test/downloads")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, method, path, rangeHeader string
		status                          int
	}{
		{"mutacion", "POST", "/downloads/invalid", "", 405},
		{"rango", "GET", "/downloads/invalid", "bytes=0-10", 416},
		{"token", "GET", "/downloads/invalid", "", 403},
		{"query", "GET", "/downloads/" + strings.Repeat("A", 43) + "?token=ignored", "", 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, tc.path, nil)
			r.Header.Set("Range", tc.rangeHeader)
			w := httptest.NewRecorder()
			b.ServeHTTP(w, r)
			if w.Code != tc.status || !strings.Contains(w.Header().Get("Cache-Control"), "no-store") || w.Header().Get("Location") != "" {
				t.Fatal("respuesta de descarga insegura")
			}
		})
	}
	if _, err := b.PresignGet(context.Background(), "private", "private.pdf", time.Minute); err != ErrDenied {
		t.Fatal("aceptó otra vigencia")
	}
	if _, err := b.PresignGet(context.Background(), "private", "private.pdf", 5*time.Minute); err != ErrDenied {
		t.Fatal("emitió una capacidad sin sesión ni actor")
	}
}
