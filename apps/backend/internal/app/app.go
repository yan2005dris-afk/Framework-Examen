// Package app es la raíz de composición de la API: construye los
// colaboradores de la aplicación y decide qué se levanta según el modo.
//
// El mismo binario puede atender tráfico HTTP o cumplir sólo con las colas
// en segundo plano. Separar ambos permite escalar el trabajo en segundo plano
// por separado del tráfico y evita que cada réplica de la APIRepita las
// tareas periódicas.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/document"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/evaluation"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/exam"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	identityredis "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity/redis"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/cache"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/config"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/db"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/httpx"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/mail"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/telemetry"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/sql/migrations"
)

/**
 * Mode selecciona qué parte de la aplicación se levanta.
 */
type Mode string

const (
	// ModeAPI atiende tráfico HTTP.
	ModeAPI Mode = "api"
	// ModeWorker levanta sólo las colas en segundo plano.
	ModeWorker Mode = "worker"
)

/**
 * App agrupa los colaboradores ya construidos. La construcción es única: el
 * modo decide qué se usa, no cómo se arma.
 */
type App struct {
	Config config.Config
	Logger *slog.Logger
	Pool   *pgxpool.Pool
	Router chi.Router

	dependencies   map[string]httpx.Dependency
	queries        *sqlcgen.Queries
	cacheClient    cache.Cache
	attemptsStore  identity.AttemptsStore
	otpStore       identity.OTPStore
	otpErr         error
	outbox         *mail.Outbox
	sender         mail.Sender
	deliverySender mail.Sender
	documents      *document.Service
	evaluation     *evaluation.Service
	exams          exam.Repository
	closers        []func()
}

/**
 * New construye la aplicación en el modo indicado.
 *
 * @param {context.Context} ctx - Contexto de arranque.
 * @param {config.Config} cfg - Configuración leída del entorno.
 * @param {Mode} mode - Parte de la aplicación que se levanta.
 * @returns {App, error} Aplicación construida o error de arranque.
 */
func New(ctx context.Context, cfg config.Config, mode Mode) (*App, error) {
	log := slog.Default()
	a := &App{Config: cfg, Logger: log}

	if err := a.connectDatabase(ctx, log, mode); err != nil {
		return nil, err
	}
	a.connectRedis(ctx, log)
	if err := a.connectMail(log); err != nil {
		return nil, err
	}

	dependencies := map[string]httpx.Dependency{
		"database": {
			Required: true,
			Probe: func(ctx context.Context) error {
				if a.Pool == nil {
					return errors.New("no configurada")
				}
				return a.Pool.Ping(ctx)
			},
		},
		"redis": {
			Required: false,
			Probe: func(ctx context.Context) error {
				if rc, ok := a.cacheClient.(*cache.RedisClient); ok {
					return rc.Ping(ctx)
				}
				return errors.New("redis no conectado")
			},
		},
	}
	a.dependencies = dependencies

	a.Router = httpx.NewRouter(httpx.RouterDependencies{
		Dependencies: dependencies,
		Logger:       log,
		Pool:         a.Pool,
		Version:      cfg.Version,
	})

	if a.Pool == nil {
		log.Warn("sin base de datos: no se registran las rutas de autenticación ni postulaciones")
		return a, nil
	}

	if err := a.buildServices(ctx, log, mode); err != nil {
		return nil, err
	}
	return a, nil
}

/**
 * connectDatabase abre el pool y, en modo API, aplica las migraciones.
 *
 * El worker no migra: las migraciones son un paso de despliegue y aplicarlas
 * desde cada réplica competiría por el versionado de goose. El bloqueo
 * consultivo serializa a quienes sí migran, de modo que varias réplicas de
 * la API pueden arrancar a la vez sin aplicar dos veces la misma versión.
 *
 * @param {context.Context} ctx - Contexto de arranque.
 * @param {slog.Logger} log - Logger de la aplicación.
 * @param {Mode} mode - Modo de ejecución.
 * @returns {error} Error irrecuperable de arranque.
 */
func (a *App) connectDatabase(ctx context.Context, log *slog.Logger, mode Mode) error {
	if a.Config.DatabaseURL == "" {
		log.Warn("DATABASE_URL no está configurada: la aplicación arranca solo con los healthchecks")
		return nil
	}

	pool, err := db.Connect(ctx, a.Config.DatabaseURL)
	if err != nil {
		if a.Config.DatabaseRequired {
			return fmt.Errorf("no se pudo conectar a la base de datos: %w", err)
		}
		log.Warn("sin base de datos: la aplicación arranca solo con los healthchecks", "error", err)
		return nil
	}
	a.Pool = pool
	a.closers = append(a.closers, pool.Close)

	if mode != ModeAPI {
		return nil
	}

	started := time.Now()
	migrationErr := migrations.Run(ctx, a.Config.DatabaseURL)
	telemetry.Default.Observe("migration", "run", telemetry.Result(migrationErr), time.Since(started))
	if migrationErr != nil {
		log.Error("no se pudieron aplicar las migraciones", "error", migrationErr)
		if a.Config.DatabaseRequired {
			return fmt.Errorf("no se pudieron aplicar las migraciones: %w", migrationErr)
		}
	}
	return nil
}

/**
 * connectRedis prepara la caché y los controles de identidad. Sin Redis se
 * conservan la lectura de sesiones y catálogos desde PostgreSQL, pero el
 * login y los flujos OTP fallan cerrados; nunca se usa Noop para seguridad.
 *
 * @param {context.Context} ctx - Contexto de arranque.
 * @param {slog.Logger} log - Logger de la aplicación.
 */
func (a *App) connectRedis(ctx context.Context, log *slog.Logger) {
	var cacheClient cache.Cache
	var securityCache cache.Cache = &cache.UnavailableCache{}
	if a.Config.RedisURL != "" {
		redisClient, redisErr := cache.NewRedis(ctx, a.Config.RedisURL)
		if redisErr != nil {
			log.Warn("Redis no disponible: autenticación bloqueada; catálogos consultan PostgreSQL")
			cacheClient = &cache.NoopCache{}
		} else {
			a.closers = append(a.closers, func() { _ = redisClient.Close() })
			cacheClient = redisClient
			securityCache = redisClient
			log.Info("Redis conectado")
		}
	} else {
		log.Warn("REDIS_URL no configurada: autenticación bloqueada; catálogos consultan PostgreSQL")
		cacheClient = &cache.NoopCache{}
	}
	a.cacheClient = cacheClient

	attemptsStore := cache.NewAttemptsStore(securityCache, identity.MaxLoginAttempts, identity.LockoutDuration)
	otpStore, err := cache.NewConfiguredOTPStore(securityCache, identity.OTPTTL, a.Config.OTPResendDelay, a.Config.OTPMaxSends)
	if err != nil {
		// La configuración de OTP inválida no deja continuar: es un fallo de
		// despliegue, no una degradación aceptable.
		log.Error("configuración OTP no válida", "error", err)
		a.otpStore = nil
		a.otpErr = err
		return
	}
	a.attemptsStore = identityredis.NewAttemptsStore(attemptsStore)
	a.otpStore = identityredis.NewOTPStore(otpStore)
}

/**
 * connectMail prepara el remitente SMTP. Sin SMTP la aplicación sigue
 * funcionando, pero no confirma ningún envío.
 *
 * @param {slog.Logger} log - Logger de la aplicación.
 * @returns {error} Error de configuración irrecuperable.
 */
func (a *App) connectMail(log *slog.Logger) error {
	if a.Config.SMTPHost == "" {
		return nil
	}
	a.sender = mail.NewSMTPSender(mail.Config{
		Host:     a.Config.SMTPHost,
		Port:     a.Config.SMTPPort,
		User:     a.Config.SMTPUser,
		Password: a.Config.SMTPPassword,
		From:     a.Config.SMTPFrom,
		UseTLS:   a.Config.SMTPUseTLS,
	})
	log.Info("Cliente de correo configurado", "host", a.Config.SMTPHost, "puerto", a.Config.SMTPPort)
	return nil
}

/**
 * Close libera los recursos abiertos durante el arranque.
 */
func (a *App) Close() {
	for _, closer := range a.closers {
		if closer != nil {
			closer()
		}
	}
}

/**
 * Handler devuelve el manejador HTTP con la cadena de middlewares.
 *
 * @returns {http.Handler} Manejador listo para el servidor.
 */
func (a *App) Handler() http.Handler {
	return httpx.RequestID(
		httpx.Logging(
			httpx.Recover(
				httpx.CORS(
					httpx.CSRF(
						httpx.MaxBodySize(a.Router, a.Config.MaxBodyBytes),
						a.Config.CORSOrigins,
					),
					a.Config.CORSOrigins,
				),
			),
		),
	)
}
