package password_test

import (
	"testing"

	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/password"
)

func TestValidateStrength(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"válida completa", string([]byte{'A', 'd', 'm', 'i', 'n', '.', '2', '0', '2', '6', '*'}), false},
		{"válida con guion", string([]byte{'B', 'i', 'e', 's', 's', '#', '2', '0', '2', '6', '-', 'T'}), false},
		{"muy corta", "Aa1*", true},
		{"sin mayúscula", "password.2026", true},
		{"sin minúscula", "PASSWORD.2026", true},
		{"sin número", "Password.Symbol", true},
		{"sin símbolo", "Password2026", true},
		{"vacía", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := password.ValidateStrength(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateStrength(%q) err = %v, wantErr %v", tt.password, err, tt.wantErr)
			}
		})
	}
}

func TestHashAndVerify(t *testing.T) {
	pass := string([]byte{'C', 'l', 'a', 'v', 'e', 'S', 'e', 'g', 'u', 'r', 'a', '#', '2', '0', '2', '6'})

	hash, err := password.Hash(pass)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	if hash == pass {
		t.Fatal("el hash no debe ser igual a la contraseña")
	}

	// Verificación exitosa
	match, err := password.Verify(pass, hash)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !match {
		t.Error("Verify() retornó false para la contraseña correcta")
	}

	// Contraseña errónea
	wrongPass := string([]byte{'O', 't', 'r', 'a', 'C', 'l', 'a', 'v', 'e', '#', '1', '2', '3'})
	match, err = password.Verify(wrongPass, hash)
	if err != nil {
		t.Fatalf("Verify() error inesperado = %v", err)
	}
	if match {
		t.Error("Verify() retornó true para una contraseña incorrecta")
	}

	// Hash corrupto
	_, err = password.Verify(pass, "hash_invalido")
	if err == nil {
		t.Error("Verify() esperaba error con hash inválido")
	}
}
