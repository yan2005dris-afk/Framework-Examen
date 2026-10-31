package httpx

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

/**
 * RouterDependencies son los componentes transversales que necesita el router
 * raíz de la API.
 */
type RouterDependencies struct {
	Logger  *slog.Logger
	Pool    *pgxpool.Pool
	Version string
}

/**
 * NewRouter construye el router raíz con los middlewares transversales.
 * Cada bounded context registra sus rutas sobre este router.
 *
 * @param {RouterDependencies} deps - Logger, pool de base de datos y versión.
 * @returns {chi.Router} Router listo para montar las rutas de los contextos.
 */
func NewRouter(deps RouterDependencies) chi.Router {
	r := chi.NewRouter()

	r.NotFound(NotFoundHandler)
	r.MethodNotAllowed(MethodNotAllowedHandler)

	r.Get("/healthz", handleHealth(deps))
	r.Get("/readyz", handleReady(deps))

	return r
}

/**
 * handleHealth responde el estado de vida del proceso.
 *
 * @param {RouterDependencies} deps - Dependencias del router.
 * @returns {http.HandlerFunc} Handler de healthcheck.
 */
func handleHealth(deps RouterDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		Respond(w, r, http.StatusOK, map[string]any{
			"status":  "ok",
			"version": deps.Version,
			"time":    time.Now().UTC(),
		})
	}
}

/**
 * handleReady verifica que la base de datos y dependencias principales estén listas (XT-08).
 *
 * @param {RouterDependencies} deps - Dependencias del router.
 * @returns {http.HandlerFunc} Handler de readiness.
 */
func handleReady(deps RouterDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		checks := map[string]string{
			"database": "ok",
		}

		if deps.Pool == nil {
			checks["database"] = "no_configurada"
			Respond(w, r, http.StatusServiceUnavailable, map[string]any{
				"status": "not_ready",
				"checks": checks,
				"error": ErrorResponse{
					Code:    "base_datos_no_configurada",
					Message: "La base de datos no está configurada en este entorno.",
				},
			})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := deps.Pool.Ping(ctx); err != nil {
			checks["database"] = "no_disponible"
			Respond(w, r, http.StatusServiceUnavailable, map[string]any{
				"status": "not_ready",
				"checks": checks,
				"error": ErrorResponse{
					Code:    "base_datos_no_disponible",
					Message: "La base de datos no está disponible.",
				},
			})
			return
		}

		Respond(w, r, http.StatusOK, map[string]any{
			"status": "ready",
			"checks": checks,
		})
	}
}

/**
 * NotFoundHandler responde con el envelope estándar de error.
 *
 * @param {http.ResponseWriter} writer - Respuesta HTTP en construcción.
 * @param {http.Request} request - Petición que originó la respuesta.
 */
func NotFoundHandler(writer http.ResponseWriter, request *http.Request) {
	RespondError(writer, request, http.StatusNotFound, "ruta_no_encontrada", "El recurso solicitado no existe.")
}

/**
 * MethodNotAllowedHandler responde con el envelope estándar de error.
 *
 * @param {http.ResponseWriter} writer - Respuesta HTTP en construcción.
 * @param {http.Request} request - Petición que originó la respuesta.
 */
func MethodNotAllowedHandler(writer http.ResponseWriter, request *http.Request) {
	RespondError(writer, request, http.StatusMethodNotAllowed, "metodo_no_permitido", "El método HTTP no está permitido para este recurso.")
}
