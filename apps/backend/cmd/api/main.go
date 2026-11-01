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
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit"
	auditpostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit/postgres"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/catalog"
	catalogpostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/catalog/postgres"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/competition"
	competitionpostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/competition/postgres"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/document"
	documentpostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/document/postgres"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/eligibility"
	eligibilitypostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/eligibility/postgres"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/eligibility/simulated"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/evaluation"
	evaluationnotification "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/evaluation/notification"
	evaluationpostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/evaluation/postgres"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/exam"
	exampostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/exam/postgres"
	examreport "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/exam/report"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/merit"
	meritpostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/merit/postgres"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/notice"
	noticepostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/notice/postgres"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/privacy"
	privacypostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/privacy/postgres"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/results"
	resultspostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/results/postgres"

	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	identitypostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity/postgres"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/cache"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/config"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/db"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/httpx"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/logger"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/mail"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/storage"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/telemetry"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/review"
	reviewpostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/review/postgres"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	submissionnotification "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission/notification"
	submissionpostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission/postgres"
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
	router := httpx.NewRouter(httpx.RouterDependencies{
		Logger:  log,
		Pool:    pool,
		Version: cfg.Version,
	})

	if pool != nil {
		queries := sqlcgen.New(pool)
		go purgeSessions(ctx, queries)
		outbox, outboxErr := mail.NewOutbox(queries, cfg.MailOutboxKey)
		if outboxErr != nil {
			log.Warn("MAIL_OUTBOX_KEY ausente o inválida: nuevos OTP y envíos finales bloqueados")
		}
		// Incluso sin SMTP/clave se retiran cargas vencidas; nunca se marcan como entregadas.
		printer := submissionprint.NewRenderer().WithMark(demoMark)
		var deliverySender mail.Sender
		if smtpSender != nil {
			deliverySender = &submissionnotification.PrintingSender{Next: &mail.MarkedSender{Next: smtpSender, Mark: demoMark}, Printer: printer}
		}

		var identityMailer identity.Mailer
		if smtpSender != nil && outboxErr == nil {
			identityMailer = outbox
		}

		// Contexto de identidad (EF-11: registro, login, sesión, OTP)
		identityRepo := identitypostgres.NewRepository(queries, db.NewTxManager(pool))
		auditor := auditpostgres.NewRepository(queries, db.NewTxManager(pool))
		privacyRepository := privacypostgres.NewRepository(queries, db.NewTxManager(pool), auditor)
		privacyHandler := privacy.NewHandler(privacyRepository, auditor)
		privacyHandler.RegisterPublic(router)
		if deliverySender != nil {
			deliverySender = &privacypostgres.Sender{Gate: privacyRepository, Next: deliverySender}
		}
		go outbox.Run(ctx, deliverySender, otpStore.Active)
		stageQueue := &noticepostgres.Queue{Outbox: outbox, Queries: queries}
		resultsHandler := results.NewHandler(results.NewService(resultspostgres.NewRepository(queries, db.NewTxManager(pool), auditor, stageQueue), auditor).WithPrivacy(privacyRepository).WithMark(demoMark))
		resultsHandler.RegisterPublic(router)
		identityService := identity.NewService(identity.ServiceDependencies{
			Applicants: identityRepo,
			Sessions:   identityRepo,
			Recovery:   identityRepo,
			Resets:     identityRepo,
			Attempts:   attemptsStore,
			OTPStore:   otpStore,
			Mailer:     identityMailer,
			SessionTTL: cfg.SessionTTL,
		})
		identity.NewHandler(identityService, cfg.IsProduction()).RegisterRoutes(router)
		competitionHandler := competition.NewHandler(competition.NewService(competitionpostgres.NewRepository(queries, db.NewTxManager(pool), auditor)))
		competitionHandler.RegisterPublic(router)
		conditions, err := simulated.Load(cfg.EligibilitySimulationFile)
		if err != nil {
			log.Error("no se pudo cargar la simulación controlada de elegibilidad")
			os.Exit(1)
		}
		eligibilityService := eligibility.NewService(conditions, eligibilitypostgres.NewRepository(queries))

		// Contexto de postulaciones
		submissionRepository := submissionpostgres.NewRepository(queries, db.NewTxManager(pool))
		confirmations := &submissionnotification.QueuedConfirmation{Outbox: outbox, Applicants: identityRepo}
		var objects document.Storage
		switch cfg.StorageDriver {
		case "local":
			objects, err = storage.NewSignedLocal(cfg.StorageDir, cfg.StoragePublicURL, cfg.StorageSigningKey)
		case "s3":
			objects, err = storage.NewS3(storage.S3Config{Endpoint: cfg.S3Endpoint, PublicEndpoint: cfg.S3PublicEndpoint, Region: cfg.S3Region, AccessKey: cfg.S3AccessKey, SecretKey: cfg.S3SecretKey, SessionToken: cfg.S3SessionToken})
		default:
			err = fmt.Errorf("STORAGE_DRIVER debe ser local o s3")
		}
		if err != nil {
			log.Error("no se pudo configurar el almacenamiento", "error", err)
			os.Exit(1)
		}
		objects = &storage.Observed{Next: objects}
		documentService := document.NewService(documentpostgres.NewRepository(pool, db.NewTxManager(pool)), objects).WithAudit(auditor).WithEligibility(eligibilityService)
		evaluationService := evaluation.NewService(evaluationpostgres.NewRepository(queries, db.NewTxManager(pool), auditor, &evaluationnotification.Outbox{Queue: outbox, Queries: queries}), objects, auditor)
		go cleanupEvaluationUploads(ctx, evaluationService)
		examRepository := exampostgres.NewRepository(queries, db.NewTxManager(pool), auditor, evaluationpostgres.NewRepository(queries, db.NewTxManager(pool), auditor, &evaluationnotification.Outbox{Queue: outbox, Queries: queries}), examreport.NewRenderer().WithMark(demoMark), privacyRepository)
		go expireExams(ctx, examRepository)
		submissionService := submission.NewService(submissionRepository).WithConfirmationQueue(confirmations).
			WithDocuments(documentService).WithApplicants(identityRepo).WithAudit(auditor).WithEligibility(eligibilityService).WithPrinter(printer).WithPrivacy(privacyRepository)
		router.Group(func(r chi.Router) {
			r.Use(identity.RequireSession(identityService))
			r.Use(identity.ProtectAdministration)
			r.Use(privacyRepository.Guard)
			privacyHandler.RegisterAdministration(r)
			exam.NewHandler(examRepository, auditor).RegisterRoutes(r)
			competitionHandler.RegisterAdministration(r)
			resultsHandler.RegisterAdministration(r)
			notice.NewHandler(notice.NewService(noticepostgres.NewRepository(queries, db.NewTxManager(pool), auditor), auditor)).RegisterRoutes(r)
			evaluation.NewHandler(evaluationService).RegisterRoutes(r)
			merit.NewHandler(merit.NewService(meritpostgres.NewRepository(queries, db.NewTxManager(pool), auditor).WithNotifier(stageQueue), auditor)).RegisterRoutes(r)
			catalog.NewHandler(catalog.NewService(catalogpostgres.NewRepository(queries), cache.NewCatalogStore(cacheClient))).RegisterRoutes(r)
			identity.RegisterRoleRoutes(r, identity.NewRoleService(identityRepo))
			submission.NewHandler(submissionService).RegisterRoutes(r)
			document.NewHandler(documentService).RegisterRoutes(r)
			audit.RegisterHistoryRoutes(r, audit.NewHistoryService(auditor, auditor))
			review.NewHandler(review.NewService(reviewpostgres.NewRepository(queries, db.NewTxManager(pool)), auditor)).RegisterRoutes(r)
		})
		go cleanupDocuments(ctx, documentService)
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

/** cleanupEvaluationUploads conserva los intentos fallidos hasta poder borrar sus objetos, sin tocar evidencias confirmadas. */
func cleanupEvaluationUploads(ctx context.Context, service *evaluation.Service) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		start := time.Now()
		run, cancel := context.WithTimeout(ctx, 20*time.Second)
		err := service.CleanupUploads(run)
		cancel()
		telemetry.Default.Observe("cleanup", "uploads", telemetry.Result(err), time.Since(start))
		if err != nil && ctx.Err() == nil {
			slog.WarnContext(ctx, "no se pudieron retirar intentos de respaldo vencidos")
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

/** expireExams recupera cierres vencidos tras reinicio; el plazo lo aplica PostgreSQL aunque el navegador esté cerrado. */
func expireExams(ctx context.Context, r exam.Repository) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		run, cancel := context.WithTimeout(ctx, 20*time.Second)
		started := time.Now()
		err := r.Expire(run)
		telemetry.Default.Observe("cleanup", "expire_exams", telemetry.Result(err), time.Since(started))
		cancel()
		if err != nil && ctx.Err() == nil {
			slog.WarnContext(ctx, "no se pudieron confirmar cierres de evaluación pendientes")
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
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
