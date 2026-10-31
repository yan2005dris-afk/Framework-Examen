// Package domain es el kernel compartido entre contextos: errores base y
// primitivas transversales. Cada contexto define sus propios errores cuando
// necesita un mensaje más rico.
package domain

import "errors"

/**
 * Errores base del kernel compartido.
 */
var (
	ErrRequiredField = errors.New("el campo obligatorio está vacío")
	ErrInvalidID     = errors.New("el identificador no es válido")
	ErrNotFound      = errors.New("el recurso no existe")
	ErrInvalidState  = errors.New("el estado del recurso no permite la operación")
)
