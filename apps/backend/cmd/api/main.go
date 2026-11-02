package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/app"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/config"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/logger"
)

/**
 * main arma la aplicación: configuración, base de datos, Redis, contextos y
 * servidor HTTP con apagado ordenado.
 *
 * El proceso worker es otro binario (cmd/worker) que comparte el mismo
 * cableado: esta variante atiende tráfico HTTP y nothing más.
 */
func main() {
	// La imagen de producción es distroless: no hay shell ni curl, así que el
	// propio binario actúa como sonda de salud del contenedor.
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(runHealthcheck())
	}

	cfg, err := config.Load()
	if err != nil {
		slog.Error("no se pudo cargar la configuración", "error", err)
		os.Exit(1)
	}

	log := logger.New(cfg.Environment)
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	application, err := app.New(ctx, cfg, app.ModeAPI)
	if err != nil {
		log.Error("no se pudo arrancar la API", "error", err)
		os.Exit(1)
	}
	defer application.Close()

	// ── Servidor HTTP ────────────────────────────────────────────────────────
	server := &http.Server{
		Addr:         cfg.Address(),
		Handler:      application.Handler(),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	go func() {
		log.Info("API iniciada", "direccion", cfg.Address(), "entorno", cfg.Environment)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("la API se detuvo con error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	log.Info("apagando la API de forma ordenada")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("error durante el apagado de la API", "error", err)
	}

	log.Info("API detenida")
}

/**
 * runHealthcheck consulta el endpoint de salud de la API en ejecución.
 *
 * @returns {number} 0 si la API responde, 1 en caso contrario.
 */
func runHealthcheck() int {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	client := &http.Client{Timeout: 3 * time.Second}

	response, err := client.Get("http://127.0.0.1:" + port + "/healthz")
	if err != nil {
		fmt.Fprintln(os.Stderr, "healthcheck: no se pudo consultar /healthz:", err)
		return 1
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "healthcheck: /healthz respondió %d\n", response.StatusCode)
		return 1
	}

	return 0
}
