import { apiGet } from "../api"
import { LISTA_PROVINCIAS, PAISES, PROVINCIAS_ECUADOR } from "@/data/ecuadorGeo"

/**
 * Obtiene la lista de países disponibles para residencia o formación.
 * Intenta consultar la caché de la API y si no responde o está desconectada, recurre al catálogo local.
 */
export async function obtenerPaises(signal?: AbortSignal): Promise<string[]> {
  try {
    const respuesta = await apiGet<{ paises: string[] }>(
      "/catalogs/countries",
      {
        signal,
      },
    )
    return respuesta.paises
  } catch {
    return PAISES
  }
}

/**
 * Obtiene las provincias de Ecuador o de un país determinado.
 */
export async function obtenerProvincias(
  pais = "Ecuador",
  signal?: AbortSignal,
): Promise<string[]> {
  if (pais.toLowerCase() === "ecuador") {
    try {
      const respuesta = await apiGet<{ provincias: string[] }>(
        `/catalogs/provinces/${encodeURIComponent(pais)}`,
        { signal },
      )
      return respuesta.provincias
    } catch {
      return LISTA_PROVINCIAS
    }
  }
  return []
}

/**
 * Obtiene los cantones correspondientes a una provincia de Ecuador.
 */
export async function obtenerCantones(
  provincia: string,
  pais = "Ecuador",
  signal?: AbortSignal,
): Promise<string[]> {
  if (pais.toLowerCase() === "ecuador" && provincia) {
    try {
      const respuesta = await apiGet<{ cantones: string[] }>(
        `/catalogs/cities/${encodeURIComponent(pais)}/${encodeURIComponent(provincia)}`,
        { signal },
      )
      return respuesta.cantones
    } catch {
      return PROVINCIAS_ECUADOR[provincia] ?? []
    }
  }
  return []
}
