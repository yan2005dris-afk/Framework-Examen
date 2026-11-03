package identity_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	. "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
)

/**
 * TestNewIDCard cubre el algoritmo módulo 10 del Registro Civil.
 */
func TestNewIDCard(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr error
	}{
		{name: "cédula válida", value: "1710034008"},
		{name: "cédula válida con espacios", value: " 1710034016 "},
		{name: "longitud incorrecta", value: "171003400", wantErr: ErrInvalidIDCardFormat},
		{name: "caracteres no numéricos", value: "abcdefghij", wantErr: ErrInvalidIDCardFormat},
		{name: "dígito verificador incorrecto", value: "1710034009", wantErr: ErrInvalidIDCard},
		{name: "provincia inexistente", value: "9910034008", wantErr: ErrInvalidIDCard},
		{name: "tercer dígito fuera de rango", value: "1760034008", wantErr: ErrInvalidIDCard},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := NewIDCard(test.value)

			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("se esperaba %v y se obtuvo %v", test.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("no se esperaba error y se obtuvo %v", err)
			}
			if len(got.String()) != 10 {
				t.Fatalf("se esperaba una cédula de 10 dígitos y se obtuvo %q", got)
			}
		})
	}
}

/**
 * TestNewApplicantAndFullName verifica el armado del postulante.
 */
func TestNewApplicantAndFullName(t *testing.T) {
	idCard, err := NewIDCard("1710034008")
	if err != nil {
		t.Fatalf("la cédula de prueba debe ser válida: %v", err)
	}

	applicant, err := NewApplicant(
		uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		idCard,
		"María",
		"Fernández",
		"Maria.Fernandez@Example.COM",
		"$argon2id$mockhash",
		"fecha_expedicion",
		time.Time{},
	)
	if err != nil {
		t.Fatalf("no se esperaba error y se obtuvo %v", err)
	}

	if applicant.FullName() != "María Fernández" {
		t.Fatalf("nombre completo inesperado: %q", applicant.FullName())
	}
	if applicant.Email != "maria.fernandez@example.com" {
		t.Fatalf("el correo debe quedar normalizado en minúsculas: %q", applicant.Email)
	}
	if applicant.Role != RoleApplicant {
		t.Fatalf("el registro público debe crear un postulante: %q", applicant.Role)
	}
}

/**
 * TestNewApplicantRequiresNames verifica la validación de nombres obligatorios.
 */
func TestNewApplicantRequiresNames(t *testing.T) {
	idCard, err := NewIDCard("1710034008")
	if err != nil {
		t.Fatalf("la cédula de prueba debe ser válida: %v", err)
	}

	_, err = NewApplicant(
		uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		idCard,
		"  ",
		"Fernández",
		"maria@example.com",
		"$argon2id$mockhash",
		"fecha_expedicion",
		time.Time{},
	)
	if !errors.Is(err, ErrInvalidApplicant) {
		t.Fatalf("se esperaba ErrInvalidApplicant y se obtuvo %v", err)
	}
}
