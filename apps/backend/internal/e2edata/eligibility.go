package e2edata

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

/**
 * EscribeSimulacionElegibilidad publica el archivo de la simulación
 * controlada que el servidor carga al arrancar.
 *
 * El servidor no acepta una bandera del cliente: la situación laboral la
 * resuelve un puerto del servidor. Sin este archivo, toda cuenta recibe
 * `503 elegibilidad_no_disponible` y el formulario EF-12 no puede
 * completarse en el stack de desarrollo.
 *
 * @param {Dataset} dataset - Conjunto sembrado, con los identificadores reales.
 * @param {string} ruta - Ruta del archivo a escribir.
 * @returns {error} Si el archivo no se puede escribir.
 */
func EscribirSimulacionElegibilidad(dataset Dataset, ruta string) error {
	if ruta == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		return fmt.Errorf("e2edata: no se pudo preparar el directorio de la simulación: %w", err)
	}
	raw, err := json.MarshalIndent(simulacionElegibilidad(dataset), "", "  ")
	if err != nil {
		return fmt.Errorf("e2edata: no se pudo serializar la simulación de elegibilidad: %w", err)
	}
	if err := os.WriteFile(ruta, raw, 0o644); err != nil {
		return fmt.Errorf("e2edata: no se pudo escribir la simulación de elegibilidad: %w", err)
	}
	return nil
}

/**
 * simulacionElegibilidad construye el archivo que el servidor carga.
 *
 * Las cuentas sembradas son elegibles en los tres sectores salvo la que
 * representa el escenario cerrado, que no lo es en ninguno: ese expediente
 * debe quedar cerrado por no elegibilidad.
 *
 * @param {Dataset} dataset - Conjunto sembrado.
 * @returns {map[string]map[string]bool} Elegibilidad por cuenta y sector.
 */
func simulacionElegibilidad(dataset Dataset) map[string]map[string]bool {
	simulacion := map[string]map[string]bool{}
	for cedula, cuenta := range dataset.Accounts {
		if cuenta.Rol != RolPostulante {
			continue
		}
		simulacion[cuenta.ID.String()] = map[string]bool{
			SectorAfiliados:   cedula != "2403401355",
			SectorJubilados:   cedula != "2403401355",
			SectorEmpleadores: cedula != "2403401355",
		}
	}
	return simulacion
}
