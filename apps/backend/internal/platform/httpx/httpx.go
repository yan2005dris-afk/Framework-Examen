package httpx

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
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
		if requestID == "" {
			requestID = uuid.NewString()
		}

		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey, requestID)))
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
					"error", cause,
					"ruta", r.URL.Path,
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
		wrapped := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(wrapped, r)

		slog.InfoContext(r.Context(), "petición atendida",
			"metodo", r.Method,
			"ruta", r.URL.Path,
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
				w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-Request-ID")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID")
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
