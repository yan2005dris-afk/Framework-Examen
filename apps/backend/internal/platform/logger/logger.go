package logger

import (
	"log/slog"
	"os"
	"strings"
)

/**
 * New construye el logger estructurado de la aplicación. En desarrollo usa
 * formato texto legible; en producción, JSON para el agregador de logs.
 *
 * @param {string} environment - Entorno de ejecución ("desarrollo", "produccion").
 * @returns {slog.Logger} Logger configurado con el nivel indicado por LOG_LEVEL.
 */
func New(environment string) *slog.Logger {
	level := levelFromEnv()

	opts := &slog.HandlerOptions{
		Level:       level,
		ReplaceAttr: sanitizeAttr,
	}

	var handler slog.Handler
	if environment == "produccion" || environment == "production" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}

/**
 * SanitizeAttrForTest expone la función de sanitización para pruebas unitarias.
 */
func SanitizeAttrForTest(groups []string, a slog.Attr) slog.Attr {
	return sanitizeAttr(groups, a)
}

/**
 * sanitizeAttr sanitiza atributos sensibles en los logs técnicos (XT-07).
 */
func sanitizeAttr(_ []string, a slog.Attr) slog.Attr {
	key := strings.ToLower(a.Key)
	switch key {
	case "password", "contrasena", "contraseña", "token", "otp", "secret", "authorization", "cookie":
		return slog.String(a.Key, "[REDACTED]")
	case "cedula", "id_card":
		s := a.Value.String()
		if len(s) == 10 {
			return slog.String(a.Key, s[:2]+"******"+s[8:])
		}
		return slog.String(a.Key, "[MASKED]")
	case "correo", "email":
		s := a.Value.String()
		parts := strings.Split(s, "@")
		if len(parts) == 2 && len(parts[0]) > 1 {
			return slog.String(a.Key, string(parts[0][0])+"***@"+parts[1])
		}
		return slog.String(a.Key, "[MASKED]")
	}
	return a
}

/**
 * levelFromEnv resuelve el nivel de log a partir de la variable LOG_LEVEL.
 *
 * @returns {slog.Level} Nivel de log configurado.
 */
func levelFromEnv() slog.Level {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("LOG_LEVEL"))) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
