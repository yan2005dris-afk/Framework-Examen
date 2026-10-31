package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/config"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/db"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/httpx"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/logger"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	submissionpostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission/postgres"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/sql/migrations"
)

/**
 * main arma la aplicación: configuración, base de datos, contextos y servidor
 * HTTP con apagado ordenado.
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

	var pool *pgxpool.Pool
	if cfg.DatabaseURL != "" {
		pool, err = db.Connect(ctx, cfg.DatabaseURL)
		if err != nil {
			if cfg.DatabaseRequired {
				log.Error("no se pudo conectar a la base de datos", "error", err)
				os.Exit(1)
			}
			log.Warn("sin base de datos: la API arranca solo con los healthchecks", "error", err)
		} else {
			defer pool.Close()

			if err := migrations.Run(ctx, cfg.DatabaseURL); err != nil {
				log.Error("no se pudieron aplicar las migraciones", "error", err)
				if cfg.DatabaseRequired {
					os.Exit(1)
				}
			}
		}
	} else {
		log.Warn("DATABASE_URL no está configurada: la API arranca solo con los healthchecks")
	}

	router := httpx.NewRouter(httpx.RouterDependencies{
		Logger:  log,
		Pool:    pool,
		Version: cfg.Version,
	})

	if pool != nil {
		queries := sqlcgen.New(pool)

		submissionRepository := submissionpostgres.NewRepository(queries)
		submissionService := submission.NewService(submissionRepository)
		submission.NewHandler(submissionService).RegisterRoutes(router)
	} else {
		log.Warn("sin base de datos: no se registran las rutas de postulaciones")
	}

	handler := httpx.RequestID(
		httpx.Recover(
			httpx.Logging(
				httpx.CORS(
					httpx.MaxBodySize(router, cfg.MaxBodyBytes),
					cfg.CORSOrigins,
				),
			),
		),
	)

	server := &http.Server{
		Addr:         cfg.Address(),
		Handler:      handler,
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
