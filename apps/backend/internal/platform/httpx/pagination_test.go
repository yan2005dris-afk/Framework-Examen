package httpx

import (
	"net/http/httptest"
	"testing"

	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/shared/domain"
)

/**
 * TestParsePageRequest verifica la lectura de los parámetros de offset.
 */
func TestParsePageRequest(t *testing.T) {
	request := httptest.NewRequest("GET", "/api/v1/submissions?page=3&pageSize=15", nil)

	page, err := ParsePageRequest(request)
	if err != nil {
		t.Fatalf("no se esperaba error y se obtuvo %v", err)
	}

	if page.Page != 3 || page.PageSize != 15 {
		t.Fatalf("se esperaba la página 3 con 15 elementos y se obtuvo %d/%d", page.Page, page.PageSize)
	}
}

/**
 * TestParsePageRequestRejectsInvalidValues verifica el rechazo de parámetros
 * no numéricos.
 */
func TestParsePageRequestRejectsInvalidValues(t *testing.T) {
	request := httptest.NewRequest("GET", "/api/v1/submissions?page=abc", nil)

	if _, err := ParsePageRequest(request); err == nil {
		t.Fatal("se esperaba un error con un parámetro de página inválido")
	}
}

/**
 * TestParseCursorRequest verifica la lectura y decodificación del cursor.
 */
func TestParseCursorRequest(t *testing.T) {
	cursor := domain.NewCursor("2026-10-30T22:00:00Z", "11111111-1111-1111-1111-111111111111")
	request := httptest.NewRequest("GET", "/api/v1/submissions?limit=10&cursor="+cursor.Encode(), nil)

	parsed, err := ParseCursorRequest(request)
	if err != nil {
		t.Fatalf("no se esperaba error y se obtuvo %v", err)
	}

	if parsed.Limit != 10 {
		t.Fatalf("se esperaba un límite de 10 y se obtuvo %d", parsed.Limit)
	}
	if parsed.Cursor == nil {
		t.Fatal("se esperaba un cursor decodificado")
	}

	value, _ := parsed.Cursor.Value(1)
	if value != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("el identificador del cursor no se conservó: %q", value)
	}
}

/**
 * TestParseCursorRequestWithoutCursor verifica la primera página sin cursor.
 */
func TestParseCursorRequestWithoutCursor(t *testing.T) {
	request := httptest.NewRequest("GET", "/api/v1/submissions", nil)

	parsed, err := ParseCursorRequest(request)
	if err != nil {
		t.Fatalf("no se esperaba error y se obtuvo %v", err)
	}

	if parsed.Cursor != nil {
		t.Fatal("la primera página no debe llevar cursor")
	}
	if parsed.Limit != domain.DefaultPageSize {
		t.Fatalf("se esperaba el límite por defecto y se obtuvo %d", parsed.Limit)
	}
}

/**
 * TestParseCursorRequestRejectsInvalidCursor verifica el rechazo de cursores
 * corruptos.
 */
func TestParseCursorRequestRejectsInvalidCursor(t *testing.T) {
	request := httptest.NewRequest("GET", "/api/v1/submissions?cursor=abc!!", nil)

	if _, err := ParseCursorRequest(request); err == nil {
		t.Fatal("se esperaba un error con un cursor inválido")
	}
}
