package identity

import "errors"

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
)
