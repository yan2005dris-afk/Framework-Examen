package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/app"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/config"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/logger"
)

/**
 * main levanta sólo las colas en segundo plano: correo, purga de sesiones,
 * limpieza de respaldos y cierre de exámenes.
 *
 * El worker comparte el cableado con la API (internal/app), pero no sirve
 * tráfico HTTP ni aplica migraciones: las migraciones son un paso de
 * despliegue y las aplica cmd/migrate o la propia API al arrancar.
 */
func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("no se pudo cargar la configuración", "error", err)
		os.Exit(1)
	}

	log := logger.New(cfg.Environment)
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	application, err := app.New(ctx, cfg, app.ModeWorker)
	if err != nil {
		log.Error("no se pudo arrancar el worker", "error", err)
		os.Exit(1)
	}
	defer application.Close()

	log.Info("worker iniciado", "entorno", cfg.Environment)
	application.RunWorkers(ctx)
	log.Info("worker detenido")
}
