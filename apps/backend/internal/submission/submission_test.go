package submission

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

/**
 * TestNewSubmissionCanonicalizesContent verifica que el contenido se guarde
 * siempre con las claves ordenadas.
 */
func TestNewSubmissionCanonicalizesContent(t *testing.T) {
	content := json.RawMessage(`{"seccion9":{"q1":"Sí"},"seccion1":{"nombres":"Ana"}}`)

	item, err := NewSubmission(uuid.New(), uuid.New(), "EF12-2026-ABC123", content, time.Now())
	if err != nil {
		t.Fatalf("no se esperaba error y se obtuvo %v", err)
	}

	expected := `{"seccion1":{"nombres":"Ana"},"seccion9":{"q1":"Sí"}}`
	if string(item.Content) != expected {
		t.Fatalf("se esperaba %s y se obtuvo %s", expected, string(item.Content))
	}
}

/**
 * TestNewSubmissionRejectsInvalidContent verifica el rechazo de contenidos que
 * no son un objeto JSON.
 */
func TestNewSubmissionRejectsInvalidContent(t *testing.T) {
	_, err := NewSubmission(uuid.New(), uuid.New(), "EF12-2026-ABC123", json.RawMessage(`[]`), time.Now())
	if !errors.Is(err, ErrInvalidContent) {
		t.Fatalf("se esperaba ErrInvalidContent y se obtuvo %v", err)
	}
}

/**
 * TestMarkAsSentLocksSubmission verifica que al enviar la postulación se
 * calcule el hash y se bloqueen las ediciones.
 */
func TestMarkAsSentLocksSubmission(t *testing.T) {
	now := time.Now()
	item, err := NewSubmission(uuid.New(), uuid.New(), "EF12-2026-ABC123", json.RawMessage(`{"seccion1":{}}`), now)
	if err != nil {
		t.Fatalf("no se esperaba error y se obtuvo %v", err)
	}

	if err := item.MarkAsSent(now); err != nil {
		t.Fatalf("no se esperaba error y se obtuvo %v", err)
	}

	if item.Status != StatusSubmitted {
		t.Fatalf("se esperaba el estado submitted y se obtuvo %s", item.Status)
	}
	if len(item.Hash) != 64 {
		t.Fatalf("se esperaba un hash SHA-256 de 64 caracteres y se obtuvo %q", item.Hash)
	}
	if item.SubmittedAt == nil {
		t.Fatal("se esperaba la fecha de envío registrada")
	}
	if item.IsEditable() {
		t.Fatal("una postulación enviada no debe ser editable")
	}

	err = item.UpdateContent(json.RawMessage(`{"seccion1":{"nombres":"Otro"}}`), now)
	if !errors.Is(err, ErrNotEditable) {
		t.Fatalf("se esperaba ErrNotEditable y se obtuvo %v", err)
	}
}

/**
 * TestHashContentIsStableAcrossKeyOrder verifica que el hash no dependa del
 * orden de las claves del formulario.
 */
func TestHashContentIsStableAcrossKeyOrder(t *testing.T) {
	first := json.RawMessage(`{"seccion2":{"canton":"Quito"},"seccion1":{"nombres":"Ana"}}`)
	second := json.RawMessage(`{"seccion1":{"nombres":"Ana"},"seccion2":{"canton":"Quito"}}`)

	canonicalFirst, err := CanonicalContent(first)
	if err != nil {
		t.Fatalf("no se esperaba error y se obtuvo %v", err)
	}

	canonicalSecond, err := CanonicalContent(second)
	if err != nil {
		t.Fatalf("no se esperaba error y se obtuvo %v", err)
	}

	if HashContent(canonicalFirst) != HashContent(canonicalSecond) {
		t.Fatal("el hash debe ser idéntico para el mismo formulario")
	}
}
