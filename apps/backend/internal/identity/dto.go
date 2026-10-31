package identity

import (
	"time"

	"github.com/google/uuid"
)

/**
 * RegisterRequestDTO es la entrada para iniciar el registro con envío de código OTP.
 */
type RegisterRequestDTO struct {
	Cedula           string `json:"cedula"`
	Correo           string `json:"correo"`
	MetodoValidacion string `json:"metodo_validacion,omitempty"` // "fecha_expedicion" | "codigo_dactilar" | "correo"
}

/**
 * RegisterVerifyDTO es la entrada para validar el código OTP y completar el registro.
 */
type RegisterVerifyDTO struct {
	Cedula    string `json:"cedula"`
	Correo    string `json:"correo"`
	CodigoOTP string `json:"codigo_otp"`
	Nombres   string `json:"nombres"`
	Apellidos string `json:"apellidos"`
	Password  string `json:"password"`
}

/**
 * LoginDTO son las credenciales de ingreso para iniciar sesión.
 */
type LoginDTO struct {
	Cedula   string `json:"cedula"`
	Password string `json:"password"`
}

/**
 * ForgotPasswordDTO inicia la recuperación de contraseña solicitando un código OTP.
 */
type ForgotPasswordDTO struct {
	Cedula string `json:"cedula"`
}

/**
 * ResetPasswordDTO confirma el código OTP y actualiza la contraseña del postulante.
 */
type ResetPasswordDTO struct {
	Cedula      string `json:"cedula"`
	CodigoOTP   string `json:"codigo_otp"`
	NewPassword string `json:"new_password"`
}

/**
 * ApplicantResponseDTO es el perfil del postulante devuelto al cliente (sin hashes sensibles).
 */
type ApplicantResponseDTO struct {
	ID               uuid.UUID `json:"id"`
	Cedula           string    `json:"cedula"`
	Nombres          string    `json:"nombres"`
	Apellidos        string    `json:"apellidos"`
	NombreCompleto   string    `json:"nombre_completo"`
	Correo           string    `json:"correo"`
	Rol              string    `json:"rol"`
	MetodoValidacion string    `json:"metodo_validacion"`
	FechaRegistro    time.Time `json:"fecha_registro"`
}

/**
 * ToResponse convierte un Applicant de dominio a su DTO de salida seguro.
 *
 * @param {*Applicant} a - Postulante de dominio.
 * @returns {ApplicantResponseDTO} DTO sin secretos.
 */
func ToResponse(a *Applicant) ApplicantResponseDTO {
	return ApplicantResponseDTO{
		ID:               a.ID,
		Cedula:           a.IDCard.String(),
		Nombres:          a.FirstName,
		Apellidos:        a.LastName,
		NombreCompleto:   a.FullName(),
		Correo:           a.Email,
		Rol:              a.Role,
		MetodoValidacion: a.ValidationMethod,
		FechaRegistro:    a.RegisteredAt,
	}
}
