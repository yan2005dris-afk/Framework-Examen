package httpx

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

/**
 * RouterDependencies son los componentes transversales que necesita el router
 * raíz de la API.
 */
type Dependency struct {
	Required bool
	Probe    func(context.Context) error
}

type RouterDependencies struct {
	Dependencies map[string]Dependency
	Logger       *slog.Logger
	Pool         *pgxpool.Pool
	Version      string
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
		dependencies := deps.Dependencies
		if dependencies == nil {
			dependencies = map[string]Dependency{"database": {Required: true, Probe: func(ctx context.Context) error {
				if deps.Pool == nil {
					return errors.New("no configurada")
				}
				return deps.Pool.Ping(ctx)
			}}}
		}
		checks := map[string]string{}
		failed := false
		degraded := false
		var mu sync.Mutex
		var wg sync.WaitGroup
		for name, d := range dependencies {
			wg.Add(1)
			go func(name string, d Dependency) {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
				defer cancel()
				state := "ok"
				if d.Probe == nil || d.Probe(ctx) != nil {
					state = "no_disponible"
				}
				if name == "database" && deps.Dependencies == nil && deps.Pool == nil {
					state = "no_configurada"
				}
				mu.Lock()
				checks[name] = state
				if state != "ok" {
					degraded = true
					failed = failed || d.Required
				}
				mu.Unlock()
			}(name, d)
		}
		wg.Wait()
		status, code := http.StatusOK, "ready"
		if degraded {
			code = "degraded"
		}
		if failed {
			status, code = http.StatusServiceUnavailable, "not_ready"
		}
		body := map[string]interface{}{"status": code, "checks": checks}
		if failed {
			body["error"] = ErrorResponse{Code: "dependencias_no_disponibles", Message: "Una dependencia obligatoria no está disponible."}
		}
		w.Header().Set("Cache-Control", "no-store")
		Respond(w, r, status, body)
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
