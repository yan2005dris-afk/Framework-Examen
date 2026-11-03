package submission

import (
	"bytes"
	"testing"
	"time"

	"github.com/google/uuid"
)

/**
 * TestNewIdentifierReturnsUUIDv7 verifica que los identificadores que genera
 * el servicio sean UUIDv7 y no UUIDv4: la versión del UUID se lee en el
 * nibble de versión.
 */
func TestNewIdentifierReturnsUUIDv7(t *testing.T) {
	id, err := newIdentifier()
	if err != nil {
		t.Fatalf("no se esperaba error y se obtuvo %v", err)
	}

	if id.Version() != 7 {
		t.Fatalf("se esperaba un UUIDv7 y se obtuvo la versión %d", id.Version())
	}
	if id.Variant() != uuid.RFC4122 {
		t.Fatalf("se esperaba la variante RFC 4122 y se obtuvo %v", id.Variant())
	}
}

/**
 * TestNewIdentifiersAreTimeOrdered verifica que dos identificadores
 * generados con separación en el tiempo ordenen de menor a mayor, que es
 * la propiedad que hace que el índice del PRIMARY KEY no se fragmente.
 *
 * Los 48 bits iniciales de un UUIDv7 son una marca de tiempo en
 * milisegundos, de modo que el orden lexicográfico de los bytes coincide con
 * el orden de creación.
 */
func TestNewIdentifiersAreTimeOrdered(t *testing.T) {
	first, err := newIdentifier()
	if err != nil {
		t.Fatalf("no se esperaba error y se obtuvo %v", err)
	}

	time.Sleep(2 * time.Millisecond)

	second, err := newIdentifier()
	if err != nil {
		t.Fatalf("no se esperaba error y se obtuvo %v", err)
	}

	if bytes.Compare(first[:], second[:]) >= 0 {
		t.Fatalf("se esperaba que %s fuera menor que %s", first, second)
	}
}

/**
 * TestUUIDv7EncodesCreationTime verifica que el UUIDv7 lleve dentro la marca
 * de tiempo de creación, de modo que se pueda recuperar sin una columna
 * adicional.
 */
func TestUUIDv7EncodesCreationTime(t *testing.T) {
	before := time.Now().Add(-time.Second)

	id, err := newIdentifier()
	if err != nil {
		t.Fatalf("no se esperaba error y se obtuvo %v", err)
	}

	decoded := timestampV7(t, id)

	if decoded.Before(before) || decoded.After(time.Now().Add(time.Second)) {
		t.Fatalf("la marca de tiempo %v está fuera del rango esperado", decoded)
	}
}

/**
 * timestampV7 decodifica la marca de tiempo en milisegundos que los 48 bits
 * iniciales de un UUIDv7 almacenan.
 *
 * @param {testing.T} t - Contexto de prueba.
 * @param {uuid.UUID} id - Identificador a decodificar.
 * @returns {time.Time} Momento de creación codificado en el identificador.
 */
func timestampV7(t *testing.T, id uuid.UUID) time.Time {
	t.Helper()

	var milliseconds int64
	for _, b := range id[:6] {
		milliseconds = milliseconds<<8 | int64(b)
	}

	return time.UnixMilli(milliseconds)
}
