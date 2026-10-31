package domain

import (
	"github.com/google/uuid"
	"time"
)

/** Position es la clave estable fecha/UUID, ligada a los filtros del contexto. */
type Position struct {
	At time.Time
	ID uuid.UUID
}

/** ParsePosition rechaza cursores incompletos, de otra consulta o con claves inválidas.
 * @param {Cursor} cursor - Token ya decodificado por httpx.
 * @param {string} scope - Identidad estable de filtros y recurso.
 * @returns {Position, error} Clave de lectura o ErrInvalidCursor.
 */
func ParsePosition(cursor *Cursor, scope string) (Position, error) {
	if cursor == nil {
		return Position{}, nil
	}
	owner, ok := cursor.Value(0)
	if !ok || owner != scope {
		return Position{}, ErrInvalidCursor
	}
	rawAt, ok := cursor.Value(1)
	if !ok {
		return Position{}, ErrInvalidCursor
	}
	at, err := time.Parse(time.RFC3339Nano, rawAt)
	if err != nil || at.IsZero() {
		return Position{}, ErrInvalidCursor
	}
	rawID, ok := cursor.Value(2)
	if !ok {
		return Position{}, ErrInvalidCursor
	}
	id, err := uuid.Parse(rawID)
	if err != nil || id == uuid.Nil {
		return Position{}, ErrInvalidCursor
	}
	if _, extra := cursor.Value(3); extra {
		return Position{}, ErrInvalidCursor
	}
	return Position{At: at, ID: id}, nil
}

/** EncodePosition entrega un cursor opaco sin exponer un contrato de claves al cliente. */
func EncodePosition(scope string, at time.Time, id uuid.UUID) string {
	return NewCursor(scope, at.UTC().Format(time.RFC3339Nano), id.String()).Encode()
}
