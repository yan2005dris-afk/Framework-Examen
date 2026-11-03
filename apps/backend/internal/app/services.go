package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

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
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	identitypostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity/postgres"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/merit"
	meritpostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/merit/postgres"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/notice"
	noticepostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/notice/postgres"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/cache"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/db"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/download"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/httpx"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/mail"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/storage"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/privacy"
	privacypostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/privacy/postgres"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/results"
	resultspostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/results/postgres"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/review"
	reviewpostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/review/postgres"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	submissionnotification "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission/notification"
	submissionpostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission/postgres"
	submissionprint "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission/print"
)

/**
 * buildServices construye los colaboradores de cada contexto y, en modo API,
 * registra las rutas sobre el router.
 *
 * @param {context.Context} ctx - Contexto de arranque.
 * @param {slog.Logger} log - Logger de la aplicación.
 * @param {Mode} mode - Modo de ejecución.
 * @returns {error} Error irrecuperable de arranque.
 */
func (a *App) buildServices(ctx context.Context, log *slog.Logger, mode Mode) error {
	cfg := a.Config
	queries := sqlcgen.New(a.Pool)
	a.queries = queries
	txManager := db.NewTxManager(a.Pool)

	outbox, outboxErr := mail.NewOutbox(queries, cfg.MailOutboxKey)
	if outboxErr != nil {
		log.Warn("MAIL_OUTBOX_KEY ausente o inválida: nuevos OTP y envíos finales bloqueados")
	}
	a.outbox = outbox

	// El sello de datos sintéticos marca las impresiones en un entorno de
	// demostración; nunca aparece en producción.
	demoMark := ""
	if cfg.Environment != "production" && os.Getenv("DATA_ORIGIN") == "synthetic-demo" {
		demoMark = "DATOS SINTÉTICOS - NO VÁLIDO"
	}

	printer := submissionprint.NewRenderer().WithMark(demoMark)
	var deliverySender mail.Sender
	if a.sender != nil {
		deliverySender = &submissionnotification.PrintingSender{Next: &mail.MarkedSender{Next: a.sender, Mark: demoMark}, Printer: printer}
	}

	auditor := auditpostgres.NewRepository(queries, txManager)
	privacyRepository := privacypostgres.NewRepository(queries, txManager, auditor)
	if deliverySender != nil {
		deliverySender = &privacypostgres.Sender{Gate: privacyRepository, Next: deliverySender}
	}
	a.deliverySender = deliverySender

	conditions, err := simulated.Load(cfg.EligibilitySimulationFile)
	if err != nil {
		return fmt.Errorf("no se pudo cargar la simulación controlada de elegibilidad: %w", err)
	}
	eligibilityService := eligibility.NewService(conditions, eligibilitypostgres.NewRepository(queries))

	var objects document.Storage
	switch cfg.StorageDriver {
	case "local":
		objects, err = storage.NewSignedLocal(cfg.StorageDir, cfg.StoragePublicURL, cfg.StorageSigningKey)
	case "s3":
		objects, err = storage.NewS3(storage.S3Config{
			Endpoint:       cfg.S3Endpoint,
			PublicEndpoint: cfg.S3PublicEndpoint,
			Region:         cfg.S3Region,
			AccessKey:      cfg.S3AccessKey,
			SecretKey:      cfg.S3SecretKey,
			SessionToken:   cfg.S3SessionToken,
		})
	default:
		err = fmt.Errorf("STORAGE_DRIVER debe ser local o s3")
	}
	if err != nil {
		return fmt.Errorf("no se pudo configurar el almacenamiento: %w", err)
	}
	objects = &storage.Observed{Next: objects}

	publicDownload := os.Getenv("DOWNLOAD_PUBLIC_URL")
	if publicDownload == "" {
		publicDownload = strings.TrimRight(cfg.PublicURL, "/") + "/downloads"
	}
	broker, brokerErr := download.New(queries, objects, privacyRepository, auditor, publicDownload)
	if brokerErr == nil {
		if mode == ModeAPI {
			downloadRouter := chi.NewRouter()
			downloadRouter.Handle("/downloads/{token}", broker)
			addr := os.Getenv("DOWNLOAD_LISTEN_ADDR")
			if addr == "" {
				addr = "0.0.0.0:8082"
			}
			if listener, lErr := net.Listen("tcp", addr); lErr == nil {
				downloadServer := &http.Server{
					Handler:           httpx.RequestID(httpx.Logging(httpx.Recover(downloadRouter))),
					ReadHeaderTimeout: 3 * time.Second,
					ReadTimeout:       5 * time.Second,
					WriteTimeout:      25 * time.Second,
					IdleTimeout:       30 * time.Second,
				}
				a.closers = append(a.closers, func() { downloadServer.Close() })
				go func() {
					if e := downloadServer.Serve(listener); e != nil && e != http.ErrServerClosed {
						log.Error("se detuvo el servidor de descargas", "error", e)
					}
				}()
			}
		}
		objects = broker
	}

	documentService := document.NewService(documentpostgres.NewRepository(a.Pool, txManager), objects).WithAudit(auditor).WithEligibility(eligibilityService)
	evaluationService := evaluation.NewService(
		evaluationpostgres.NewRepository(queries, txManager, auditor, &evaluationnotification.Outbox{Queue: outbox, Queries: queries}),
		objects, auditor,
	)
	examRepository := exampostgres.NewRepository(
		queries, txManager, auditor,
		evaluationpostgres.NewRepository(queries, txManager, auditor, &evaluationnotification.Outbox{Queue: outbox, Queries: queries}),
		examreport.NewRenderer().WithMark(demoMark), privacyRepository,
	)
	a.documents = documentService
	a.evaluation = evaluationService
	a.exams = examRepository

	if a.dependencies != nil {
		a.dependencies["storage"] = httpx.Dependency{
			Required: false,
			Probe: func(ctx context.Context) error {
				if r, ok := objects.(interface{ Ready(context.Context) error }); ok {
					return r.Ready(ctx)
				}
				return nil
			},
		}
		a.dependencies["outbox"] = httpx.Dependency{
			Required: false,
			Probe: func(ctx context.Context) error {
				if outbox != nil {
					return outbox.Ready(ctx)
				}
				return errors.New("outbox no disponible")
			},
		}
		a.dependencies["smtp"] = httpx.Dependency{
			Required: false,
			Probe: func(ctx context.Context) error {
				if a.deliverySender != nil {
					if r, ok := a.deliverySender.(interface{ Ready(context.Context) error }); ok {
						return r.Ready(ctx)
					}
				}
				if a.sender != nil {
					if r, ok := a.sender.(interface{ Ready(context.Context) error }); ok {
						return r.Ready(ctx)
					}
				}
				return errors.New("smtp no disponible")
			},
		}
		a.dependencies["downloads"] = httpx.Dependency{
			Required: false,
			Probe: func(ctx context.Context) error {
				if brokerErr != nil {
					return brokerErr
				}
				return nil
			},
		}
	}

	if mode != ModeAPI {
		return nil
	}

	// ── Rutas ────────────────────────────────────────────────────────────────
	identityRepo := identitypostgres.NewRepository(queries, txManager)
	privacyHandler := privacy.NewHandler(privacyRepository, auditor)
	privacyHandler.RegisterPublic(a.Router)

	stageQueue := &noticepostgres.Queue{Outbox: outbox, Queries: queries}
	resultsHandler := results.NewHandler(
		results.NewService(resultspostgres.NewRepository(queries, txManager, auditor, stageQueue), auditor).
			WithPrivacy(privacyRepository).WithMark(demoMark),
	)
	resultsHandler.RegisterPublic(a.Router)

	var identityMailer identity.Mailer
	if a.sender != nil && outboxErr == nil {
		identityMailer = outbox
	}
	identityService := identity.NewService(identity.ServiceDependencies{
		Applicants: identityRepo,
		Sessions:   identityRepo,
		Recovery:   identityRepo,
		Resets:     identityRepo,
		Attempts:   a.attemptsStore,
		OTPStore:   a.otpStore,
		Mailer:     identityMailer,
		SessionTTL: cfg.SessionTTL,
	})
	identity.NewHandler(identityService, cfg.IsProduction()).RegisterRoutes(a.Router)

	competitionHandler := competition.NewHandler(competition.NewService(competitionpostgres.NewRepository(queries, txManager, auditor)))
	competitionHandler.RegisterPublic(a.Router)

	submissionRepository := submissionpostgres.NewRepository(queries, txManager)
	confirmations := &submissionnotification.QueuedConfirmation{Outbox: outbox, Applicants: identityRepo}
	submissionService := submission.NewService(submissionRepository).WithConfirmationQueue(confirmations).
		WithDocuments(documentService).WithApplicants(identityRepo).WithAudit(auditor).
		WithEligibility(eligibilityService).WithPrinter(printer).WithPrivacy(privacyRepository)

	a.Router.Group(func(r chi.Router) {
		r.Use(identity.RequireSession(identityService))
		r.Use(identity.ProtectAdministration)
		r.Use(privacyRepository.Guard)
		privacyHandler.RegisterAdministration(r)
		exam.NewHandler(examRepository, auditor).RegisterRoutes(r)
		competitionHandler.RegisterAdministration(r)
		resultsHandler.RegisterAdministration(r)
		notice.NewHandler(notice.NewService(noticepostgres.NewRepository(queries, txManager, auditor), auditor)).RegisterRoutes(r)
		evaluation.NewHandler(evaluationService).RegisterRoutes(r)
		merit.NewHandler(merit.NewService(meritpostgres.NewRepository(queries, txManager, auditor).WithNotifier(stageQueue), auditor)).RegisterRoutes(r)
		catalog.NewHandler(catalog.NewService(catalogpostgres.NewRepository(queries), cache.NewCatalogStore(a.cacheClient))).RegisterRoutes(r)
		identity.RegisterRoleRoutes(r, identity.NewRoleService(identityRepo))
		submission.NewHandler(submissionService).RegisterRoutes(r)
		document.NewHandler(documentService).RegisterRoutes(r)
		audit.RegisterHistoryRoutes(r, audit.NewHistoryService(auditor, auditor))
		review.NewHandler(review.NewService(reviewpostgres.NewRepository(queries, txManager), auditor)).RegisterRoutes(r)
	})

	return nil
}
