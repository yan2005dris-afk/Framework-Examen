// Package appkit arma el router de la API a partir de dependencias explícitas.
//
// El binario de producción y las pruebas E2E construyen exactamente el mismo
// router: las pruebas no reimplementan el cableado, lo reutilizan. Así una
// ruta o un middleware nuevo queda cubierto por la suite sin duplicar
// contextos en dos lugares.
package appkit

import (
	"context"
	"fmt"
	"log/slog"
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
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	identitypostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity/postgres"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/merit"
	meritpostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/merit/postgres"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/notice"
	noticepostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/notice/postgres"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/cache"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/db"
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
 * Dependencies agrupa lo que el router necesita para registrar los contextos.
 *
 * Los campos son explícitos a propósito: la suite E2E inyecta su propio
 * almacenamiento, su propio remitente de correo y su propio clave de buzón,
 * sin alterar el comportamiento del binario de producción.
 */
type Dependencies struct {
	// Logger es el registrador transversal de la aplicación.
	Logger *slog.Logger
	// Pool es el pool de PostgreSQL ya conectado.
	Pool *pgxpool.Pool
	// Queries agrupa las consultas tipadas de sqlc.
	Queries *sqlcgen.Queries
	// Version es la versión publicada en el healthcheck.
	Version string
	// IsProduction endurece la política de cookies de sesión.
	IsProduction bool
	// SessionTTL define la vigencia de la cookie de sesión.
	SessionTTL time.Duration
	// AttemptsStore cuenta los intentos de acceso por cuenta.
	AttemptsStore *cache.AttemptsStore
	// OTPStore emite y verifica los códigos de un solo uso.
	OTPStore *cache.OTPStore
	// CatalogStore cachea los catálogos geográficos y las opciones EF-12.
	CatalogStore cache.Cache
	// Mailer entrega los correos de identidad; nil bloquea el OTP.
	Mailer identity.Mailer
	// DeliverySender entrega las notificaciones del ciclo de vida del
	// expediente; nil desactiva ese envío sin afectar al resto de la API.
	DeliverySender mail.Sender
	// Outbox persiste los correos que el servidor debe entregar.
	Outbox *mail.Outbox
	// Storage conserva los documentos adjuntos.
	Storage document.Storage
	// Printer genera la hoja imprimible del expediente sellado.
	Printer *submissionprint.Renderer
	// EligibilityProvider resuelve la situación laboral del postulante.
	EligibilityProvider eligibility.Provider
	// DemoMark se imprime en las salidas cuando el origen es sintético.
	DemoMark string
}

/**
 * NewRouter construye el router raíz y registra las rutas de cada contexto.
 *
 * @param {context.Context} ctx - Contexto de vida de las tareas de fondo.
 * @param {Dependencies} deps - Dependencias del cableado de la API.
 * @returns {chi.Router} Router con las rutas públicas y administrativas.
 * @returns {mail.Sender} Remitente de entrega con el guard de privacidad.
 * @returns {error} Si la configuración de los controles de identidad no es válida.
 */
func NewRouter(ctx context.Context, deps Dependencies) (chi.Router, mail.Sender, error) {
	if deps.Logger == nil {
		deps.Logger = slog.Default()
	}
	if deps.SessionTTL <= 0 {
		deps.SessionTTL = 24 * time.Hour
	}

	router := httpx.NewRouter(httpx.RouterDependencies{
		Logger:  deps.Logger,
		Pool:    deps.Pool,
		Version: deps.Version,
	})

	queries := deps.Queries
	if queries == nil && deps.Pool != nil {
		queries = sqlcgen.New(deps.Pool)
	}
	if queries == nil {
		return nil, nil, fmt.Errorf("appkit: se requiere un pool o unas consultas tipadas")
	}

	identityRepo := identitypostgres.NewRepository(queries, db.NewTxManager(deps.Pool))
	auditor := auditpostgres.NewRepository(queries, db.NewTxManager(deps.Pool))
	privacyRepository := privacypostgres.NewRepository(queries, db.NewTxManager(deps.Pool), auditor)
	privacyHandler := privacy.NewHandler(privacyRepository, auditor)
	privacyHandler.RegisterPublic(router)

	competitionHandler := competition.NewHandler(competition.NewService(competitionpostgres.NewRepository(queries, db.NewTxManager(deps.Pool), auditor)))
	competitionHandler.RegisterPublic(router)

	identityService := identity.NewService(identity.ServiceDependencies{
		Applicants: identityRepo,
		Sessions:   identityRepo,
		Recovery:   identityRepo,
		Resets:     identityRepo,
		Attempts:   deps.AttemptsStore,
		OTPStore:   deps.OTPStore,
		Mailer:     deps.Mailer,
		SessionTTL: deps.SessionTTL,
	})
	identity.NewHandler(identityService, deps.IsProduction).RegisterRoutes(router)

	eligibilityProvider := deps.EligibilityProvider
	if eligibilityProvider == nil {
		eligibilityProvider = &simulated.Provider{}
	}
	eligibilityService := eligibility.NewService(eligibilityProvider, eligibilitypostgres.NewRepository(queries))

	documentService := document.NewService(documentpostgres.NewRepository(deps.Pool, db.NewTxManager(deps.Pool)), deps.Storage).WithAudit(auditor).WithEligibility(eligibilityService)
	evaluationService := evaluation.NewService(evaluationpostgres.NewRepository(queries, db.NewTxManager(deps.Pool), auditor, &evaluationnotification.Outbox{Queue: deps.Outbox, Queries: queries}), deps.Storage, auditor)
	examRepository := exampostgres.NewRepository(queries, db.NewTxManager(deps.Pool), auditor, evaluationpostgres.NewRepository(queries, db.NewTxManager(deps.Pool), auditor, &evaluationnotification.Outbox{Queue: deps.Outbox, Queries: queries}), examreport.NewRenderer().WithMark(deps.DemoMark), privacyRepository)

	confirmations := &submissionnotification.QueuedConfirmation{Outbox: deps.Outbox, Applicants: identityRepo}
	submissionService := submission.NewService(submissionpostgres.NewRepository(queries, db.NewTxManager(deps.Pool))).
		WithConfirmationQueue(confirmations).
		WithDocuments(documentService).
		WithApplicants(identityRepo).
		WithAudit(auditor).
		WithEligibility(eligibilityService).
		WithPrinter(deps.Printer).
		WithPrivacy(privacyRepository)

	// El envío del expediente pasa por el guard de privacidad: sin política
	// publicada el correo no sale. El guard vive aquí porque la política es
	// parte del dominio y no del transporte.
	deliverySender := deps.DeliverySender
	if deliverySender != nil {
		deliverySender = &privacypostgres.Sender{Gate: privacyRepository, Next: deliverySender}
	}

	router.Group(func(r chi.Router) {
		r.Use(identity.RequireSession(identityService))
		r.Use(identity.ProtectAdministration)
		r.Use(privacyRepository.Guard)
		privacyHandler.RegisterAdministration(r)
		exam.NewHandler(examRepository, auditor).RegisterRoutes(r)
		competitionHandler.RegisterAdministration(r)
		results.NewHandler(results.NewService(resultspostgres.NewRepository(queries, db.NewTxManager(deps.Pool), auditor, &noticepostgres.Queue{Outbox: deps.Outbox, Queries: queries}), auditor).WithPrivacy(privacyRepository).WithMark(deps.DemoMark)).RegisterAdministration(r)
		notice.NewHandler(notice.NewService(noticepostgres.NewRepository(queries, db.NewTxManager(deps.Pool), auditor), auditor)).RegisterRoutes(r)
		evaluation.NewHandler(evaluationService).RegisterRoutes(r)
		merit.NewHandler(merit.NewService(meritpostgres.NewRepository(queries, db.NewTxManager(deps.Pool), auditor).WithNotifier(&noticepostgres.Queue{Outbox: deps.Outbox, Queries: queries}), auditor)).RegisterRoutes(r)
		catalog.NewHandler(catalog.NewService(catalogpostgres.NewRepository(queries), cache.NewCatalogStore(deps.CatalogStore))).RegisterRoutes(r)
		identity.RegisterRoleRoutes(r, identity.NewRoleService(identityRepo))
		submission.NewHandler(submissionService).RegisterRoutes(r)
		document.NewHandler(documentService).RegisterRoutes(r)
		audit.RegisterHistoryRoutes(r, audit.NewHistoryService(auditor, auditor))
		review.NewHandler(review.NewService(reviewpostgres.NewRepository(queries, db.NewTxManager(deps.Pool)), auditor)).RegisterRoutes(r)
	})

	// Tareas de fondo que dependen de servicios ya construidos. Se lanzan
	// aquí para que el binario y las pruebas E2E compartan el mismo ciclo
	// de vida: la suite hereda la expiración de exámenes y la limpieza de
	// cargas sin duplicar la lógica.
	go cleanupEvaluationUploads(ctx, evaluationService)
	go expireExams(ctx, examRepository)
	go cleanupDocuments(ctx, documentService)

	return router, deliverySender, nil
}

/**
 * NewStorage construye el almacenamiento de documentos según el controlador
 * configurado en el entorno.
 *
 * @param {string} driver - "local" o "s3".
 * @param {string} localDir - Directorio base del almacenamiento local.
 * @param {string} localPublicURL - URL pública del almacenamiento local.
 * @param {string} localSigningKey - Clave de firma de las URLs temporales.
 * @param {storage.S3Config} s3 - Parámetros del almacenamiento en S3.
 * @returns {document.Storage} Almacenamiento listo para adjuntos.
 * @throws {error} Si el controlador no es válido o faltan credenciales.
 */
func NewStorage(driver string, localDir string, localPublicURL string, localSigningKey string, s3 storage.S3Config) (document.Storage, error) {
	switch driver {
	case "local":
		return storage.NewSignedLocal(localDir, localPublicURL, localSigningKey)
	case "s3":
		return storage.NewS3(s3)
	default:
		return nil, fmt.Errorf("STORAGE_DRIVER debe ser local o s3")
	}
}
