package submission

import "errors"

/**
 * Errores del contexto de postulaciones. La capa HTTP los traduce a
 * respuestas con código y mensaje; nunca deben filtrar detalles técnicos.
 */
var (
	ErrNotFound          = errors.New("la postulación no existe")
	ErrNotEditable       = errors.New("la postulación ya fue enviada y no admite cambios")
	ErrInvalidContent    = errors.New("el contenido de la postulación no es válido")
	ErrInvalidApplicant  = errors.New("el postulante indicado no es válido")
	ErrDuplicateCode     = errors.New("ya existe una postulación con ese código")
	ErrInvalidIdentifier = errors.New("el identificador de la postulación no es válido")
)
