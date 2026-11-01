package evaluation

import "errors"

var ErrForbidden = errors.New("no tiene permiso para esta operación")
var ErrNotFound = errors.New("no se encontró el recurso solicitado")
var ErrInvalid = errors.New("los datos no cumplen el contrato de evaluación")
var ErrPolicy = errors.New("se requieren reglas aprobadas para esta operación")
var ErrConflict = errors.New("el resultado, la versión o la clave de reintento no coincide")
var ErrPending = errors.New("faltan resultados definitivos publicados, incluida la nota calculada por T27")
var ErrDeadline = errors.New("la actuación está fuera del plazo aprobado para el actor")
var ErrPhase = errors.New("la fase no está publicada y habilitada")
var ErrState = errors.New("la actuación no corresponde al estado actual")
var ErrQuota = errors.New("se excede la cuota de documentos del postulante")
var ErrUnavailable = errors.New("no fue posible confirmar la operación y su notificación")
