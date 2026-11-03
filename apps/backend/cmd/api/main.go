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

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/eligibility/simulated"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/appkit"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/cache"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/config"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/db"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/httpx"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/logger"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/mail"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/storage"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/telemetry"
	submissionnotification "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission/notification"
	submissionprint "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission/print"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/sql/migrations"
)

/**
 * main arma la aplicación: configuración, base de datos, Redis, contextos y
 * servidor HTTP con apagado ordenado.
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

	// ── Base de datos ────────────────────────────────────────────────────────
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

			migrationStarted := time.Now()
			migrationErr := migrations.Run(ctx, cfg.DatabaseURL)
			telemetry.Default.Observe("migration", "run", telemetry.Result(migrationErr), time.Since(migrationStarted))
			if err := migrationErr; err != nil {
				log.Error("no se pudieron aplicar las migraciones", "error", err)
				if cfg.DatabaseRequired {
					os.Exit(1)
				}
			}
		}
	} else {
		log.Warn("DATABASE_URL no está configurada: la API arranca solo con los healthchecks")
	}

	// ── Redis: caché opcional, controles de identidad obligatorios ────────────
	// Sin Redis se conservan lectura de sesiones y catálogos desde PostgreSQL,
	// pero login y flujos OTP fallan cerrados; nunca se usa Noop para seguridad.
	var cacheClient cache.Cache
	var securityCache cache.Cache = &cache.UnavailableCache{}
	if cfg.RedisURL != "" {
		redisClient, redisErr := cache.NewRedis(ctx, cfg.RedisURL)
		if redisErr != nil {
			log.Warn("Redis no disponible: autenticación bloqueada; catálogos consultan PostgreSQL")
			cacheClient = &cache.NoopCache{}
		} else {
			defer redisClient.Close()
			cacheClient = redisClient
			securityCache = redisClient
			log.Info("Redis conectado")
		}
	} else {
		log.Warn("REDIS_URL no configurada: autenticación bloqueada; catálogos consultan PostgreSQL")
		cacheClient = &cache.NoopCache{}
	}

	attemptsStore := cache.NewAttemptsStore(securityCache)
	otpStore, err := cache.NewConfiguredOTPStore(securityCache, cfg.OTPResendDelay, cfg.OTPMaxSends)
	if err != nil {
		log.Error("configuración OTP no válida")
		os.Exit(1)
	}

	// ── Correo (SMTP / Mailpit) ──────────────────────────────────────────────
	var smtpSender mail.Sender
	if cfg.SMTPHost != "" {
		smtpSender = mail.NewSMTPSender(mail.Config{
			Host:     cfg.SMTPHost,
			Port:     cfg.SMTPPort,
			User:     cfg.SMTPUser,
			Password: cfg.SMTPPassword,
			From:     cfg.SMTPFrom,
			UseTLS:   cfg.SMTPUseTLS,
		})
		log.Info("Cliente de correo configurado", "host", cfg.SMTPHost, "puerto", cfg.SMTPPort)
	}

	demoMark := ""
	if cfg.Environment != "production" && os.Getenv("DATA_ORIGIN") == "synthetic-demo" {
		demoMark = "DATOS SINTÉTICOS - NO VÁLIDO"
	}

	// ── Router y contextos ───────────────────────────────────────────────────
	// El router se construye en appkit: el binario y las pruebas E2E comparten
	// el mismo cableado de contextos y middlewares.
	var router chi.Router
	var guardedDelivery mail.Sender
	if pool != nil {
		queries := sqlcgen.New(pool)
		go purgeSessions(ctx, queries)
		outbox, outboxErr := mail.NewOutbox(queries, cfg.MailOutboxKey)
		if outboxErr != nil {
			log.Warn("MAIL_OUTBOX_KEY ausente o inválida: nuevos OTP y envíos finales bloqueados")
		}
		// Incluso sin SMTP/clave se retiran cargas vencidas; nunca se marcan como entregadas.
		printer := submissionprint.NewRenderer().WithMark(demoMark)
		var smtpDelivery mail.Sender
		if smtpSender != nil {
			smtpDelivery = &submissionnotification.PrintingSender{Next: &mail.MarkedSender{Next: smtpSender, Mark: demoMark}, Printer: printer}
		}

		var identityMailer identity.Mailer
		if smtpSender != nil && outboxErr == nil {
			identityMailer = outbox
		}

		conditions, err := simulated.Load(cfg.EligibilitySimulationFile)
		if err != nil {
			log.Error("no se pudo cargar la simulación controlada de elegibilidad")
			os.Exit(1)
		}

		objects, err := appkit.NewStorage(cfg.StorageDriver, cfg.StorageDir, cfg.StoragePublicURL, cfg.StorageSigningKey, storage.S3Config{
			Endpoint:       cfg.S3Endpoint,
			PublicEndpoint: cfg.S3PublicEndpoint,
			Region:         cfg.S3Region,
			AccessKey:      cfg.S3AccessKey,
			SecretKey:      cfg.S3SecretKey,
			SessionToken:   cfg.S3SessionToken,
		})
		if err != nil {
			log.Error("no se pudo configurar el almacenamiento", "error", err)
			os.Exit(1)
		}
		objects = &storage.Observed{Next: objects}

		router, guardedDelivery, err = appkit.NewRouter(ctx, appkit.Dependencies{
			Logger:              log,
			Pool:                pool,
			Queries:             queries,
			Version:             cfg.Version,
			IsProduction:        cfg.IsProduction(),
			SessionTTL:          cfg.SessionTTL,
			AttemptsStore:       attemptsStore,
			OTPStore:            otpStore,
			CatalogStore:        cacheClient,
			Mailer:              identityMailer,
			DeliverySender:      smtpDelivery,
			Outbox:              outbox,
			Storage:             objects,
			Printer:             printer,
			EligibilityProvider: conditions,
			DemoMark:            demoMark,
		})
		if err != nil {
			log.Error("no se pudo construir el router", "error", err)
			os.Exit(1)
		}
		go outbox.Run(ctx, guardedDelivery, otpStore.Active)
	} else {
		log.Warn("sin base de datos: no se registran las rutas de autenticación ni postulaciones")
	}

	// ── Middleware ───────────────────────────────────────────────────────────
	handler := httpx.RequestID(
		httpx.Logging(
			httpx.Recover(
				httpx.CORS(
					httpx.CSRF(
						httpx.MaxBodySize(httpx.Deadline(router, cfg.RequestTimeout), cfg.MaxBodyBytes),
						cfg.CORSOrigins,
					),
					cfg.CORSOrigins,
				),
			),
		),
	)

	// Métricas fuera del proxy público; loopback por defecto y sin etiquetas personales.
	metricsAddr := os.Getenv("METRICS_ADDR")
	if metricsAddr == "" {
		metricsAddr = "127.0.0.1:9090"
	}
	metricsServer := &http.Server{Addr: metricsAddr, Handler: telemetry.Default.Handler(), ReadHeaderTimeout: 3 * time.Second, WriteTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second}
	go func() {
		if e := metricsServer.ListenAndServe(); e != nil && e != http.ErrServerClosed {
			log.Warn("métricas no disponibles")
		}
	}()
	defer metricsServer.Close()
	// ── Servidor HTTP ────────────────────────────────────────────────────────
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

/** purgeSessions elimina sólo sesiones que llevan treinta días expiradas, en lotes acotados. */
func purgeSessions(ctx context.Context, q *sqlcgen.Queries) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		run, cancel := context.WithTimeout(ctx, 10*time.Second)
		started := time.Now()
		_, err := q.PurgeExpiredSessionBatch(run)
		cancel()
		telemetry.Default.Observe("cleanup", "sessions", telemetry.Result(err), time.Since(started))
		if err != nil && ctx.Err() == nil {
			slog.WarnContext(ctx, "purga de sesiones pendiente")
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
