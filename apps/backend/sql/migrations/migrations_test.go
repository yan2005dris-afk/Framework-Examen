package migrations

import (
	"context"
	"os"
	"testing"
)

func TestEnvOr(t *testing.T) {
	key := "TEST_ENV_OR_KEY_FRAMEWORK"
	fallback := "default_val"

	os.Unsetenv(key)
	if val := envOr(key, fallback); val != fallback {
		t.Fatalf("se esperaba %q, obtenido %q", fallback, val)
	}

	expected := "custom_val"
	os.Setenv(key, expected)
	defer os.Unsetenv(key)

	if val := envOr(key, fallback); val != expected {
		t.Fatalf("se esperaba %q, obtenido %q", expected, val)
	}
}

func TestUpSeedAdminMissingPassword(t *testing.T) {
	os.Unsetenv("INIT_ADMIN_PASSWORD")
	err := upSeedAdmin(context.Background(), nil)
	if err == nil {
		t.Fatal("se esperaba error cuando falta INIT_ADMIN_PASSWORD")
	}
}

func TestUpSeedAdminWeakPassword(t *testing.T) {
	os.Setenv("INIT_ADMIN_PASSWORD", "weak")
	defer os.Unsetenv("INIT_ADMIN_PASSWORD")

	err := upSeedAdmin(context.Background(), nil)
	if err == nil {
		t.Fatal("se esperaba error con contraseña débil")
	}
}
