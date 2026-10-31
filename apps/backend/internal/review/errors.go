package review

import "errors"

/** Errores estables del contexto de revisión; HTTP nunca expone errores de persistencia. */
var (
	ErrForbidden           = errors.New("su rol no permite consultar el concurso")
	ErrInvalidFilter       = errors.New("indique convocatoriaId o sinConvocatoria=true y filtros válidos")
	ErrCompetitionNotFound = errors.New("la convocatoria no existe")
	ErrExportTooLarge      = errors.New("la selección supera los límites de exportación; filtre por sector")
	ErrSnapshot            = errors.New("el expediente sellado no es íntegro o no se puede interpretar")
)
