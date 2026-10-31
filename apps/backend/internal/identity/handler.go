package identity

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/httpx"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/password"
)

const sessionCookieName = "session"

type contextKey string

const applicantContextKey contextKey = "identity:applicant"

/**
 * Handler expone los casos de uso de identidad (EF-11) por HTTP.
 */
type Handler struct {
	service      *Service
	isProduction bool
}

/**
 * NewHandler crea una nueva instancia del Handler de identidad.
 *
 * @param {*Service} service - Servicio de identidad.
 * @param {bool} isProduction - Indica si se aplican directivas de producción (cookie Secure).
 * @returns {*Handler} Handler configurado.
 */
func NewHandler(service *Service, isProduction bool) *Handler {
	return &Handler{
		service:      service,
		isProduction: isProduction,
	}
}

/**
 * RegisterRoutes registra las rutas de autenticación en el router de Chi.
 *
 * @param {chi.Router} router - Router HTTP principal.
 */
func (h *Handler) RegisterRoutes(router chi.Router) {
	router.Route("/api/v1/auth", func(r chi.Router) {
		r.Post("/register/request", h.requestRegistration)
		r.Post("/register/verify", h.verifyAndRegister)
		r.Post("/login", h.login)
		r.Post("/logout", h.logout)
		r.Get("/session", h.getSession)
		r.Post("/password/forgot", h.forgotPassword)
		r.Post("/password/reset", h.resetPassword)
	})
}

func (h *Handler) requestRegistration(w http.ResponseWriter, r *http.Request) {
	var body RegisterRequestDTO
	if err := decodeJSON(r, &body); err != nil {
		httpx.RespondError(w, r, http.StatusBadRequest, "cuerpo_invalido", "El cuerpo de la petición no es válido.")
		return
	}

	err := h.service.RequestRegistration(r.Context(), body)
	if err != nil {
		switch {
		case errors.Is(err, ErrAccountLocked):
			httpx.RespondError(w, r, http.StatusTooManyRequests, "intentos_excedidos", ErrAccountLocked.Error())
		case errors.Is(err, ErrApplicantAlreadyExists):
			httpx.RespondError(w, r, http.StatusConflict, "postulante_existente", err.Error())
		case errors.Is(err, ErrInvalidInformation):
			httpx.RespondError(w, r, http.StatusUnprocessableEntity, "informacion_invalida", err.Error())
		default:
			httpx.RespondError(w, r, http.StatusInternalServerError, "error_interno", "Error al procesar la solicitud de registro.")
		}
		return
	}

	httpx.Respond(w, r, http.StatusOK, map[string]any{
		"mensaje": "Código de verificación enviado al correo electrónico.",
	})
}

func (h *Handler) verifyAndRegister(w http.ResponseWriter, r *http.Request) {
	var body RegisterVerifyDTO
	if err := decodeJSON(r, &body); err != nil {
		httpx.RespondError(w, r, http.StatusBadRequest, "cuerpo_invalido", "El cuerpo de la petición no es válido.")
		return
	}

	ip := clientIP(r)
	ua := r.UserAgent()

	applicant, token, err := h.service.VerifyAndRegister(r.Context(), body, ip, ua)
	if err != nil {
		switch {
		case errors.Is(err, ErrAccountLocked):
			httpx.RespondError(w, r, http.StatusTooManyRequests, "intentos_excedidos", ErrAccountLocked.Error())
		case errors.Is(err, ErrInvalidOTP):
			httpx.RespondError(w, r, http.StatusUnauthorized, "otp_invalido", ErrInvalidOTP.Error())
		case errors.Is(err, password.ErrWeakPassword):
			httpx.RespondError(w, r, http.StatusUnprocessableEntity, "password_debil", err.Error())
		case errors.Is(err, ErrApplicantAlreadyExists):
			httpx.RespondError(w, r, http.StatusConflict, "postulante_existente", err.Error())
		default:
			httpx.RespondError(w, r, http.StatusInternalServerError, "error_interno", "Error al completar el registro.")
		}
		return
	}

	if token != "" {
		h.setSessionCookie(w, token, int(h.service.sessionTTL.Seconds()))
	}

	httpx.Respond(w, r, http.StatusCreated, map[string]any{
		"postulante": ToResponse(applicant),
	})
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var body LoginDTO
	if err := decodeJSON(r, &body); err != nil {
		httpx.RespondError(w, r, http.StatusBadRequest, "cuerpo_invalido", "El cuerpo de la petición no es válido.")
		return
	}

	ip := clientIP(r)
	ua := r.UserAgent()

	applicant, token, err := h.service.Login(r.Context(), body.Cedula, body.Password, ip, ua)
	if err != nil {
		switch {
		case errors.Is(err, ErrAccountLocked):
			httpx.RespondError(w, r, http.StatusTooManyRequests, "intentos_excedidos", ErrAccountLocked.Error())
		case errors.Is(err, ErrInvalidCredentials):
			httpx.RespondError(w, r, http.StatusUnauthorized, "credenciales_invalidas", ErrInvalidCredentials.Error())
		default:
			httpx.RespondError(w, r, http.StatusInternalServerError, "error_interno", "Error al iniciar sesión.")
		}
		return
	}

	h.setSessionCookie(w, token, int(h.service.sessionTTL.Seconds()))

	httpx.Respond(w, r, http.StatusOK, map[string]any{
		"postulante": ToResponse(applicant),
	})
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil && cookie.Value != "" {
		_ = h.service.Logout(r.Context(), cookie.Value)
	}

	h.clearSessionCookie(w)

	httpx.Respond(w, r, http.StatusOK, map[string]any{
		"mensaje": "Sesión cerrada correctamente.",
	})
}

func (h *Handler) getSession(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		httpx.RespondError(w, r, http.StatusUnauthorized, "sin_sesion", "No hay una sesión activa.")
		return
	}

	applicant, err := h.service.ValidateSession(r.Context(), cookie.Value)
	if err != nil {
		h.clearSessionCookie(w)
		httpx.RespondError(w, r, http.StatusUnauthorized, "sesion_expirada", "La sesión ha expirado.")
		return
	}

	httpx.Respond(w, r, http.StatusOK, map[string]any{
		"postulante": ToResponse(applicant),
	})
}

func (h *Handler) forgotPassword(w http.ResponseWriter, r *http.Request) {
	var body ForgotPasswordDTO
	if err := decodeJSON(r, &body); err != nil {
		httpx.RespondError(w, r, http.StatusBadRequest, "cuerpo_invalido", "El cuerpo de la petición no es válido.")
		return
	}

	_ = h.service.ForgotPassword(r.Context(), body.Cedula)

	httpx.Respond(w, r, http.StatusOK, map[string]any{
		"mensaje": "Si la cédula está registrada, se ha enviado un código de recuperación a su correo.",
	})
}

func (h *Handler) resetPassword(w http.ResponseWriter, r *http.Request) {
	var body ResetPasswordDTO
	if err := decodeJSON(r, &body); err != nil {
		httpx.RespondError(w, r, http.StatusBadRequest, "cuerpo_invalido", "El cuerpo de la petición no es válido.")
		return
	}

	err := h.service.ResetPassword(r.Context(), body)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidOTP):
			httpx.RespondError(w, r, http.StatusUnauthorized, "otp_invalido", ErrInvalidOTP.Error())
		case errors.Is(err, password.ErrWeakPassword):
			httpx.RespondError(w, r, http.StatusUnprocessableEntity, "password_debil", err.Error())
		case errors.Is(err, ErrInvalidInformation):
			httpx.RespondError(w, r, http.StatusUnprocessableEntity, "informacion_invalida", err.Error())
		default:
			httpx.RespondError(w, r, http.StatusInternalServerError, "error_interno", "Error al restablecer la contraseña.")
		}
		return
	}

	h.clearSessionCookie(w)

	httpx.Respond(w, r, http.StatusOK, map[string]any{
		"mensaje": "Contraseña actualizada correctamente. Inicie sesión con sus nuevas credenciales.",
	})
}

// ── Cookie Helpers ───────────────────────────────────────────────────────────

func (h *Handler) setSessionCookie(w http.ResponseWriter, token string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   h.isProduction,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.isProduction,
		SameSite: http.SameSiteLaxMode,
	})
}

// ── Middleware ───────────────────────────────────────────────────────────────

/**
 * RequireSession es el middleware HTTP que valida la presencia de una cookie de sesión
 * válida en PostgreSQL antes de permitir el acceso a rutas protegidas.
 *
 * @param {*Service} service - Servicio de identidad.
 * @returns {func(http.Handler) http.Handler} Middleware de Chi.
 */
func RequireSession(service *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(sessionCookieName)
			if err != nil || cookie.Value == "" {
				httpx.RespondError(w, r, http.StatusUnauthorized, "no_autenticado", "Debe iniciar sesión para acceder a este recurso.")
				return
			}

			applicant, err := service.ValidateSession(r.Context(), cookie.Value)
			if err != nil {
				httpx.RespondError(w, r, http.StatusUnauthorized, "sesion_invalida", "Su sesión ha expirado o no es válida.")
				return
			}

			ctx := context.WithValue(r.Context(), applicantContextKey, applicant)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

/**
 * ApplicantFromContext obtiene el postulante autenticado almacenado en el contexto de la petición.
 *
 * @param {context.Context} ctx - Contexto de la petición HTTP.
 * @returns {*Applicant, bool} Postulante y true si existe en el contexto.
 */
func ApplicantFromContext(ctx context.Context) (*Applicant, bool) {
	applicant, ok := ctx.Value(applicantContextKey).(*Applicant)
	return applicant, ok
}

// ── Utility Helpers ──────────────────────────────────────────────────────────

func decodeJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(target)
}

func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		parts := strings.Split(fwd, ",")
		return strings.TrimSpace(parts[0])
	}
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}
	return r.RemoteAddr
}
