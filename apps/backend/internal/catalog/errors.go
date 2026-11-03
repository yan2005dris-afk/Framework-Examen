package catalog

import "errors"

var (
	ErrInvalidFilter  = errors.New("indique un país del catálogo y, para Ecuador, una provincia válida")
	ErrUnknownCatalog = errors.New("el catálogo solicitado no existe")
)
