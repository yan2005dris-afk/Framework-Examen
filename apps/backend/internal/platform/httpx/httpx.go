package httpx

import (
	"context"
	"encoding/json"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/shared/domain"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/telemetry"
)

/**
 * requestIDKey es la clave de contexto donde se guarda el identificador de la
 * petición.
 */
type contextKey string

const requestIDKey contextKey = "request_id"

/**
 * ErrorResponse es el envelope estándar de error de la API.
 */
type ErrorResponse struct {
	Code    string   `json:"codigo"`
	Message string   `json:"mensaje"`
	Details []string `json:"detalle,omitempty"`
}

/**
 * Respond serializa una respuesta con el código HTTP indicado.
 *
 * @param {http.ResponseWriter} writer - Respuesta HTTP en construcción.
 * @param {http.Request} request - Petición que originó la respuesta.
 * @param {number} status - Código de estado HTTP.
 * @param {any} payload - Cuerpo que se serializa como JSON.
 */
func Respond(writer http.ResponseWriter, request *http.Request, status int, payload any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)

	if err := json.NewEncoder(writer).Encode(payload); err != nil {
		slog.ErrorContext(request.Context(), "no se pudo escribir la respuesta", "error", err)
	}
}

/**
 * RespondError responde con el envelope estándar de error.
 *
 * @param {http.ResponseWriter} writer - Respuesta HTTP en construcción.
 * @param {http.Request} request - Petición que originó la respuesta.
 * @param {number} status - Código de estado HTTP.
 * @param {string} code - Código de error estable para el cliente.
 * @param {string} message - Mensaje legible para el usuario.
 */
func RespondError(writer http.ResponseWriter, request *http.Request, status int, code string, message string) {
	Respond(writer, request, status, map[string]any{
		"error": ErrorResponse{
			Code:    code,
			Message: message,
		},
	})
}

/**
 * RequestID agrega un identificador único a cada petición y lo devuelve en
 * la cabecera X-Request-ID.
 *
 * @param {http.Handler} next - Handler siguiente de la cadena.
 * @returns {http.Handler} Handler que inyecta el identificador.
 */
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if parsed, e := uuid.Parse(requestID); e != nil || parsed == uuid.Nil {
			requestID = uuid.NewString()
		}

		w.Header().Set("X-Request-ID", requestID)
		id, _ := uuid.Parse(requestID)
		ctx := domain.WithAuditRequest(context.WithValue(r.Context(), requestIDKey, requestID), id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

/**
 * GetRequestID devuelve el identificador de la petición almacenado en el
 * contexto.
 *
 * @param {Context} ctx - Contexto de la petición.
 * @returns {string} Identificador de la petición o una cadena vacía.
 */
func GetRequestID(ctx context.Context) string {
	requestID, ok := ctx.Value(requestIDKey).(string)
	if !ok {
		return ""
	}
	return requestID
}

/**
 * Recover captura los panics y responde con un error JSON en vez de cortar la
 * conexión.
 *
 * @param {http.Handler} next - Handler siguiente de la cadena.
 * @returns {http.Handler} Handler que recupera los panics.
 */
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if cause := recover(); cause != nil {
				slog.ErrorContext(r.Context(), "panic recuperado en la petición",
					"error", "panic",
					"ruta", "panic",
					"request_id", GetRequestID(r.Context()),
				)
				RespondError(w, r, http.StatusInternalServerError, "error_interno", "Ocurrió un error al procesar la solicitud.")
			}
		}()

		next.ServeHTTP(w, r)
	})
}

/**
 * Logging registra cada petición atendida con el logger estructurado.
 *
 * @param {http.Handler} next - Handler siguiente de la cadena.
 * @returns {http.Handler} Handler que mide y registra cada petición.
 */
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rc := chi.NewRouteContext()
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rc))
		parent, trace, span := telemetry.TraceParent()
		w.Header().Set("Traceparent", parent)
		wrapped := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(wrapped, r)

		telemetry.Default.Observe("http", "request", telemetry.HTTPClass(wrapped.Status()), time.Since(start))
		pattern := rc.RoutePattern()
		if strings.Contains(pattern, "/register/request") || strings.Contains(pattern, "/password/forgot") {
			telemetry.Default.Observe("otp", "issue", telemetry.HTTPClass(wrapped.Status()), time.Since(start))
		}
		if strings.Contains(pattern, "/register/verify") || strings.Contains(pattern, "/password/reset") {
			telemetry.Default.Observe("otp", "verify", telemetry.HTTPClass(wrapped.Status()), time.Since(start))
		}
		if strings.Contains(pattern, "export") {
			telemetry.Default.Observe("exports", "request", telemetry.HTTPClass(wrapped.Status()), time.Since(start))
		}
		if wrapped.Status() == 429 {
			telemetry.Default.Observe("rate_limit", "request", "denied", time.Since(start))
		}

		slog.InfoContext(r.Context(), "petición atendida",
			"metodo", r.Method,
			"ruta", telemetry.SafeRoute(rc.RoutePattern()),
			"trace_id", trace, "span_id", span,
			"estado", wrapped.Status(),
			"bytes", wrapped.BytesWritten(),
			"duracion_ms", time.Since(start).Milliseconds(),
			"request_id", GetRequestID(r.Context()),
		)
	})
}

/**
 * CORS aplica la política de orígenes permitidos a la respuesta.
 *
 * @param {http.Handler} next - Handler siguiente de la cadena.
 * @param {string[]} origins - Orígenes permitidos; vacío deniega todo.
 * @returns {http.Handler} Handler con CORS aplicado.
 */
func CORS(next http.Handler, origins []string) http.Handler {
	allowed := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		allowed[strings.TrimSpace(origin)] = struct{}{}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			if _, ok := allowed[origin]; ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-Request-ID, Idempotency-Key")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID, Retry-After, X-Retry-At, Traceparent")
				w.Header().Add("Vary", "Origin")
			}
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

/**
 * CSRF valida que las peticiones mutantes (POST, PUT, PATCH, DELETE) que
 * llevan credenciales de sesión provengan de un origen verificable: el mismo
 * origen de la petición o un origen de la lista blanca. Una petición con
 * cookie y sin ninguna señal de origen se rechaza en vez de asumirse legítima
 * (XT-01).
 *
 * La lista blanca cubre orígenes adicionales —por ejemplo, una SPA servida en
 * otro dominio—. El despliegue normal sirve la SPA y la API desde el mismo
 * origen detrás de nginx, y ese caso lo resuelve la comparación de origen.
 *
 * @param {http.Handler} next - Handler siguiente de la cadena.
 * @param {string[]} origins - Orígenes autorizados (e.g. "http://localhost:5173").
 * @returns {http.Handler} Handler con validación CSRF por origen.
 */
func CSRF(next http.Handler, origins []string) http.Handler {
	allowed := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		trimmed := strings.TrimRight(strings.TrimSpace(origin), "/")
		if trimmed != "" {
			allowed[trimmed] = struct{}{}
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}

		// La protección importa sólo cuando la petición lleva credenciales
		// ambientales: sin cookies no hay estado de sesión que otro sitio
		// pueda falsificar, y el control de sesión responderá 401 por su cuenta.
		if len(r.Cookies()) == 0 {
			next.ServeHTTP(w, r)
			return
		}

		// El navegador envía Sec-Fetch-Site en toda petición y no permite que
		// el sitio lo fije: "cross-site" delata una falsificación.
		site := strings.ToLower(strings.TrimSpace(r.Header.Get("Sec-Fetch-Site")))
		if site == "cross-site" {
			RespondError(w, r, http.StatusForbidden, "origen_no_permitido", "Petición cruzada no autorizada.")
			return
		}

		origin := strings.TrimRight(strings.TrimSpace(r.Header.Get("Origin")), "/")
		if origin == "" {
			// Sin Origin ni Sec-Fetch-Site no hay forma de verificar de dónde
			// viene la llamada: se rechaza en lugar de asumir que es propia.
			if site != "same-origin" {
				RespondError(w, r, http.StatusForbidden, "origen_no_verificable", "No se pudo verificar el origen de la petición.")
				return
			}
			next.ServeHTTP(w, r)
			return
		}

		// Mismo origen: el navegador no puede emitir un Origin distinto del
		// suyo, así que la comparación es concluyente.
		if sameOrigin(r, origin) {
			next.ServeHTTP(w, r)
			return
		}

		if _, ok := allowed[origin]; !ok {
			RespondError(w, r, http.StatusForbidden, "origen_no_permitido", "Origen de la petición no autorizado.")
			return
		}

		next.ServeHTTP(w, r)
	})
}

/**
 * sameOrigin compara el origen declarado por el cliente con el origen
 * efectivo de la petición, usando la cabecera Host y el esquema reenviado
 * por el proxy de despliegue.
 *
 * @param {http.Request} request - Petición entrante.
 * @param {string} origin - Origen declarado, sin barra final.
 * @returns {bool} True si ambos coinciden.
 */
func sameOrigin(request *http.Request, origin string) bool {
	host := strings.ToLower(strings.TrimSpace(request.Host))
	if host == "" {
		return false
	}

	scheme := "http"
	if request.TLS != nil {
		scheme = "https"
	}
	// nginx reenvía el esquema original; detrás de un proxy el TLS termina
	// antes de llegar a la API.
	if forwarded := strings.TrimSpace(strings.Split(request.Header.Get("X-Forwarded-Proto"), ",")[0]); forwarded != "" {
		scheme = strings.ToLower(forwarded)
	}

	return strings.EqualFold(origin, scheme+"://"+host)
}

/**
 * MaxBodySize limita el tamaño del cuerpo de la petición y responde 413
 * cuando se supera el máximo permitido.
 *
 * @param {http.Handler} next - Handler siguiente de la cadena.
 * @param {number} maxBytes - Tamaño máximo admitido en bytes.
 * @returns {http.Handler} Handler que acota el cuerpo de la petición.
 */
func MaxBodySize(next http.Handler, maxBytes int64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength > maxBytes {
			RespondError(w, r, http.StatusRequestEntityTooLarge, "cuerpo_muy_grande", "El contenido de la petición excede el tamaño permitido.")
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		next.ServeHTTP(w, r)
	})
}

/** Deadline propaga un plazo a DB, Redis y almacenamiento; el servidor conserva límites de lectura/escritura. */
func Deadline(next http.Handler, timeout time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
