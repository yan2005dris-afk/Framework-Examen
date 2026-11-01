package db

import (
	"context"
	"strings"
	"testing"
)

func TestInvalidDatabaseConfigurationDoesNotLeakCredentials(t *testing.T) {
	_, err := Connect(context.Background(), "postgres://private-user:NEVER-DISPLAY-THIS@localhost/%zz")
	if err == nil || strings.Contains(err.Error(), "NEVER-DISPLAY") || strings.Contains(err.Error(), "private-user") {
		t.Fatal("la configuración filtró credenciales")
	}
}
