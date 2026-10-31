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

	var handler slog.Handler
	if environment == "produccion" || environment == "production" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	}

	return slog.New(handler)
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
