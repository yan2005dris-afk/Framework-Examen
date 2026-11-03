package config

import (
	"os"
	"testing"
)

/**
 * TestLoadOrigins comprueba el contrato de CORS_ALLOWED_ORIGINS: una variable
 * definida —incluso vacía— se respeta al pie de la letra y vacío significa
 * denegar todo origen cruzado.
 */
func TestLoadOrigins(t *testing.T) {
	t.Run("variable_vacia_deniega_todo", func(t *testing.T) {
		t.Setenv("CORS_ALLOWED_ORIGINS", "")
		origins := loadOrigins("production")
		if len(origins) != 0 {
			t.Fatalf("una lista vacía debía denegar todo, obtuve %v", origins)
		}
	})

	t.Run("variable_ausente_en_produccion_deniega_todo", func(t *testing.T) {
		t.Setenv("CORS_ALLOWED_ORIGINS", "http://temporal.example")
		os.Unsetenv("CORS_ALLOWED_ORIGINS")
		origins := loadOrigins("production")
		if len(origins) != 0 {
			t.Fatalf("en producción sin variable se esperaba cero orígenes, obtuve %v", origins)
		}
	})

	t.Run("variable_ausente_en_desarrollo_usa_localhost", func(t *testing.T) {
		t.Setenv("CORS_ALLOWED_ORIGINS", "http://temporal.example")
		os.Unsetenv("CORS_ALLOWED_ORIGINS")
		origins := loadOrigins("development")
		if len(origins) != 1 || origins[0] != "http://localhost:5173" {
			t.Fatalf("en desarrollo se esperaba localhost:5173, obtuve %v", origins)
		}
	})

	t.Run("variable_explicita_manda", func(t *testing.T) {
		t.Setenv("CORS_ALLOWED_ORIGINS", "https://concurso.biess.gob.ec, https://otro.example")
		origins := loadOrigins("production")
		if len(origins) != 2 || origins[0] != "https://concurso.biess.gob.ec" || origins[1] != "https://otro.example" {
			t.Fatalf("la lista explícita debía respetarse, obtuve %v", origins)
		}
	})
}
