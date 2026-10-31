package domain

import (
	"errors"
	"github.com/google/uuid"
	"testing"
	"time"
)

func TestPositionRejectsWrongScopeMissingAndExtraValues(t *testing.T) {
	now := time.Now().UTC()
	id := uuid.New()
	raw := EncodePosition("scope", now, id)
	cursor, err := DecodeCursor(raw)
	if err != nil {
		t.Fatal(err)
	}
	p, err := ParsePosition(&cursor, "scope")
	if err != nil || p.ID != id || !p.At.Equal(now) {
		t.Fatal(p, err)
	}
	for _, c := range []Cursor{NewCursor("otro", now.Format(time.RFC3339Nano), id.String()), NewCursor("scope"), NewCursor("scope", "ayer", id.String()), NewCursor("scope", now.Format(time.RFC3339Nano), uuid.Nil.String()), NewCursor("scope", now.Format(time.RFC3339Nano), id.String(), "extra")} {
		if _, err := ParsePosition(&c, "scope"); !errors.Is(err, ErrInvalidCursor) {
			t.Fatal("cursor inválido admitido")
		}
	}
}
