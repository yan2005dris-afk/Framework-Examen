package merit

import "errors"

var ErrForbidden = errors.New("no tiene permiso para esta operación de méritos")
var ErrNotFound = errors.New("el expediente o la versión solicitada no existe")
var ErrInvalid = errors.New("la revisión o las reglas de méritos no son válidas")
var ErrRulesPending = errors.New("no existen reglas de méritos aprobadas para la convocatoria y sector")
var ErrConflict = errors.New("el expediente o sus reglas cambiaron; vuelva a consultar antes de continuar")
var ErrEvidence = errors.New("las evidencias verificadas no corresponden al formulario sellado")
var ErrPhase = errors.New("la fase de méritos no está habilitada en este momento")
