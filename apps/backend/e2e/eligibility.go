package e2e

import (
	"context"

	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/eligibility"
)

/**
 * ProveedorElegibilidad es el puerto controlado que responde por la situación
 * laboral del postulante durante las pruebas.
 *
 * Sustituye a la simulación leída de disco porque sus claves son los
 * identificadores de las cuentas: el seed los genera en cada corrida y no
 * sería práctico reconstruir el archivo antes de cada prueba.
 */
type ProveedorElegibilidad struct {
	// cuentas son los identificadores de las cuentas sembradas.
	cuentas map[uuid.UUID]bool
	// sectorInvalido es el sector que responde no elegible, si se indica.
	sectorInvalido string
}

/**
 * NewProveedorElegibilidad construye el proveedor con las cuentas que pueden
 * ser evaluadas.
 *
 * @param {map[uuid.UUID]bool} cuentas - Cuentas evaluables.
 * @returns {*ProveedorElegibilidad} Proveedor listo para usar.
 */
func NewProveedorElegibilidad(cuentas map[uuid.UUID]bool) *ProveedorElegibilidad {
	return &ProveedorElegibilidad{cuentas: cuentas}
}

/**
 * Resolve responde por la elegibilidad de una cuenta en un sector.
 *
 * @param {context.Context} ctx - Contexto de la evaluación.
 * @param {uuid.UUID} cuenta - Identificador de la cuenta consultada.
 * @param {string} sector - Sector evaluado.
 * @returns {eligibility.Condition} Condición de elegibilidad.
 * @returns {error} Si la cuenta no está sembrada o el sector no es válido.
 */
func (p *ProveedorElegibilidad) Resolve(ctx context.Context, cuenta uuid.UUID, sector string) (eligibility.Condition, error) {
	if err := ctx.Err(); err != nil {
		return eligibility.Condition{}, err
	}
	conocida, ok := p.cuentas[cuenta]
	if !ok {
		return eligibility.Condition{}, errElegibilidadNoDisponible
	}
	elegible := conocida && sector != p.sectorInvalido
	return eligibility.Condition{Eligible: elegible, Source: "simulacion_controlada_servidor"}, nil
}

/**
 * errElegibilidadNoDisponible se usa cuando la cuenta no está sembrada: el
 * servidor falla cerrado y no devuelve "no elegible" por error.
 */
var errElegibilidadNoDisponible = errorElegibilidad("la situación laboral no está disponible")

/**
 * errorElegibilidad es un error simple con mensaje fijo.
 */
type errorElegibilidad string

/**
 * Error devuelve el mensaje del error.
 *
 * @returns {string} Mensaje.
 */
func (e errorElegibilidad) Error() string { return string(e) }
