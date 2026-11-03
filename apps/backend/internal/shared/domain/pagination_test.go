package domain

import (
	"errors"
	"testing"
)

/**
 * TestNewPageRequestClampsValues verifica los límites de la paginación por
 * offset.
 */
func TestNewPageRequestClampsValues(t *testing.T) {
	tests := []struct {
		name         string
		page         int
		pageSize     int
		wantPage     int
		wantPageSize int
	}{
		{name: "valores por defecto", page: 0, pageSize: 0, wantPage: 1, wantPageSize: DefaultPageSize},
		{name: "página negativa", page: -3, pageSize: 10, wantPage: 1, wantPageSize: 10},
		{name: "tamaño mayor al máximo", page: 2, pageSize: 500, wantPage: 2, wantPageSize: MaxPageSize},
		{name: "valores válidos", page: 3, pageSize: 25, wantPage: 3, wantPageSize: 25},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := NewPageRequest(test.page, test.pageSize)

			if request.Page != test.wantPage {
				t.Fatalf("se esperaba la página %d y se obtuvo %d", test.wantPage, request.Page)
			}
			if request.PageSize != test.wantPageSize {
				t.Fatalf("se esperaba pageSize %d y se obtuvo %d", test.wantPageSize, request.PageSize)
			}
		})
	}
}

/**
 * TestPageRequestOffsetAndLimit verifica el cálculo de LIMIT y OFFSET.
 */
func TestPageRequestOffsetAndLimit(t *testing.T) {
	request := NewPageRequest(3, 20)

	if request.Limit() != 20 {
		t.Fatalf("se esperaba un límite de 20 y se obtuvo %d", request.Limit())
	}
	if request.Offset() != 40 {
		t.Fatalf("se esperaba un offset de 40 y se obtuvo %d", request.Offset())
	}
}

/**
 * TestCursorRoundTrip verifica que el cursor sobreviva a la codificación.
 */
func TestCursorRoundTrip(t *testing.T) {
	cursor := NewCursor("2026-10-30T22:00:00Z", "11111111-1111-1111-1111-111111111111")
	encoded := cursor.Encode()

	decoded, err := DecodeCursor(encoded)
	if err != nil {
		t.Fatalf("no se esperaba error y se obtuvo %v", err)
	}

	value, ok := decoded.Value(0)
	if !ok || value != "2026-10-30T22:00:00Z" {
		t.Fatalf("la fecha del cursor no se conservó: %q", value)
	}

	value, ok = decoded.Value(1)
	if !ok || value != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("el identificador del cursor no se conservó: %q", value)
	}

	if _, ok := decoded.Value(5); ok {
		t.Fatal("una posición fuera del cursor no debe existir")
	}
}

/**
 * TestDecodeCursorRejectsInvalidToken verifica el rechazo de cursores inválidos.
 */
func TestDecodeCursorRejectsInvalidToken(t *testing.T) {
	if _, err := DecodeCursor("no-es-base64-valido!!"); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("se esperaba ErrInvalidCursor y se obtuvo %v", err)
	}
}

/**
 * TestNewCursorRequestClampsLimit verifica el límite duro de la paginación.
 */
func TestNewCursorRequestClampsLimit(t *testing.T) {
	if request := NewCursorRequest(0, nil); request.Limit != DefaultPageSize {
		t.Fatalf("se esperaba el límite por defecto y se obtuvo %d", request.Limit)
	}
	if request := NewCursorRequest(10_000, nil); request.Limit != MaxPageSize {
		t.Fatalf("se esperaba el límite máximo y se obtuvo %d", request.Limit)
	}
}
