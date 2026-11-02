import { apiGet } from "../api"

/**
 * Obtiene la lista de países disponibles para residencia o formación.
 * Los errores de red o de la API se propagan: la vista decide si usa un respaldo.
 */
export async function obtenerPaises(signal?: AbortSignal): Promise<string[]> {
  const respuesta = await apiGet<{ paises: string[] }>("/catalogs/countries", {
    signal,
  })
  return respuesta.paises
}

/**
 * Obtiene las provincias de Ecuador. Para otros países devuelve una lista
 * vacía sin consultar la API, porque la provincia se escribe a mano.
 */
export async function obtenerProvincias(
  pais = "Ecuador",
  signal?: AbortSignal,
): Promise<string[]> {
  if (pais.toLowerCase() !== "ecuador") return []
  const respuesta = await apiGet<{ provincias: string[] }>(
    `/catalogs/provinces/${encodeURIComponent(pais)}`,
    { signal },
  )
  return respuesta.provincias
}

/**
 * Obtiene los cantones de una provincia de Ecuador. Para otros países o sin
 * provincia devuelve una lista vacía sin consultar la API.
 */
export async function obtenerCantones(
  provincia: string,
  pais = "Ecuador",
  signal?: AbortSignal,
): Promise<string[]> {
  if (pais.toLowerCase() !== "ecuador" || !provincia) return []
  const respuesta = await apiGet<{ cantones: string[] }>(
    `/catalogs/cities/${encodeURIComponent(pais)}/${encodeURIComponent(provincia)}`,
    { signal },
  )
  return respuesta.cantones
}
