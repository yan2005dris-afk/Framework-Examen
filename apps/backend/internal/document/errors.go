package document

import "errors"

var (
	ErrNotFound        = errors.New("el documento no existe")
	ErrUnsupportedType = errors.New("solo se admiten archivos PDF")
	ErrFileTooLarge    = errors.New("el archivo supera el tamaño permitido para esta sección")
	ErrQuotaExceeded   = errors.New("los documentos del postulante superan el límite de 50 MiB")
	ErrInvalidSection  = errors.New("solo las secciones 4 a 8 admiten respaldos PDF")
	ErrInvalidRecord   = errors.New("el registro de la sección no existe o no es válido")
	ErrInvalidName     = errors.New("el nombre del archivo no es válido")
	ErrDuplicate       = errors.New("el registro ya tiene un respaldo; elimínelo antes de reemplazarlo")
)
