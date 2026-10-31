package logger_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/logger"
)

func TestLogger_SanitizeSensitiveAttributes(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level:       slog.LevelInfo,
		ReplaceAttr: logger.SanitizeAttrForTest,
	})
	log := slog.New(handler)

	log.InfoContext(context.Background(), "evento de prueba",
		"password", "Secreto123*",
		"token", "raw-token-123",
		"otp", "654321",
		"cedula", "1710034008",
		"correo", "postulante@biess.ec",
		"campo_normal", "valor_publico",
	)

	out := buf.String()

	if strings.Contains(out, "Secreto123*") {
		t.Error("el log no debe contener la contraseña")
	}
	if strings.Contains(out, "raw-token-123") {
		t.Error("el log no debe contener el token")
	}
	if strings.Contains(out, "654321") {
		t.Error("el log no debe contener el código OTP")
	}
	if strings.Contains(out, "1710034008") {
		t.Error("la cédula debe estar truncada/enmascarada")
	}
	if !strings.Contains(out, "17******08") {
		t.Errorf("se esperaba la cédula enmascarada '17******08', log: %s", out)
	}
	if !strings.Contains(out, "p***@biess.ec") {
		t.Errorf("se esperaba el correo enmascarado 'p***@biess.ec', log: %s", out)
	}
	if !strings.Contains(out, "valor_publico") {
		t.Error("el campo normal debe conservarse")
	}
}

