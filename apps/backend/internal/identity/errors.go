package identity

import (
	"errors"
	"time"
)

/**
 * Errores del contexto de identidad con mensajes literales exigidos por el EF-11.
 */
var (
	ErrDeliveryUnavailable    = errors.New("no se pudo aceptar la notificación; inténtelo más tarde")
	ErrApplicantNotFound      = errors.New("postulante no encontrado")
	ErrInvalidCredentials     = errors.New("sus credenciales de ingreso son incorrectas")
	ErrInvalidInformation     = errors.New("la información ingresada es incorrecta")
	ErrAccountLocked          = errors.New("registro fallido, inténtelo nuevamente en 10 minutos")
	ErrInvalidOTP             = errors.New("el código de verificación es incorrecto o ha expirado")
	ErrSessionExpired         = errors.New("la sesión ha expirado o no es válida")
	ErrApplicantAlreadyExists = errors.New("ya existe un postulante registrado con esa cédula o correo")
	ErrInvalidResetToken      = errors.New("el enlace de restablecimiento es inválido o ha expirado")

	// ErrUnavailable indica que el control de seguridad (caché de intentos y
	// desafíos OTP) no está disponible. Los flujos que dependen de él fallan
	// cerrados: nunca se degradan a una respuesta permisiva.
	ErrUnavailable = errors.New("el servicio de seguridad no está disponible")
	// ErrOTPLocked indica que un desafío agotó sus intentos verificables.
	ErrOTPLocked = errors.New("el desafío agotó sus intentos")
	// ErrResendLimited indica que aún no corresponde emitir otro código.
	ErrResendLimited = errors.New("espere antes de solicitar otro código")
)

/**
 * RetryError conserva el código de dominio del fallo y el tiempo restante
 * leído de la infraestructura. El adaptador lo traduce para que el dominio
 * decida el código HTTP y la cabecera Retry-After sin conocer la caché.
 */
type RetryError struct {
	Cause      error
	RetryAfter time.Duration
}

/**
 * Error devuelve el mensaje del error de dominio.
 *
 * @returns {string} Mensaje del error de dominio.
 */
func (e *RetryError) Error() string { return e.Cause.Error() }

/**
 * Unwrap expone el error de dominio para que errors.Is lo compare.
 *
 * @returns {error} Error de dominio envuelto.
 */
func (e *RetryError) Unwrap() error { return e.Cause }
