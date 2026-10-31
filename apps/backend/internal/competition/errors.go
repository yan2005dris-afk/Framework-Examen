package competition

import (
	"errors"
)

var ErrInvalid = errors.New("la configuración de la convocatoria no es válida")
var ErrForbidden = errors.New("no tiene permiso para administrar convocatorias")
var ErrNotFound = errors.New("la convocatoria no existe")
var ErrVersion = errors.New("la convocatoria cambió; vuelva a consultarla antes de guardar")
var ErrDuplicate = errors.New("ya existe una convocatoria con ese código")
