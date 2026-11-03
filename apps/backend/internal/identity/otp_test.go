package identity_test

import (
	"testing"

	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
)

/**
 * TestGenerateOTPEsCodigoDeSeisDigitos comprueba que el código emitido por el
 * dominio es siempre numérico, de seis cifras y no empieza en cero.
 */
func TestGenerateOTPEsCodigoDeSeisDigitos(t *testing.T) {
	vistos := map[string]bool{}
	for i := 0; i < 200; i++ {
		code, err := identity.GenerateOTP()
		if err != nil {
			t.Fatalf("GenerateOTP: %v", err)
		}
		if len(code) != 6 {
			t.Fatalf("longitud esperada 6, obtuve %d (%q)", len(code), code)
		}
		if code[0] == '0' {
			t.Fatalf("el código no debe empezar en cero: %q", code)
		}
		for _, caracter := range code {
			if caracter < '0' || caracter > '9' {
				t.Fatalf("carácter no numérico en %q", code)
			}
		}
		vistos[code] = true
	}
	if len(vistos) < 150 {
		t.Fatalf("los códigos se repiten demasiado: %d distintos de 200", len(vistos))
	}
}
