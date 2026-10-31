import { apiGet, apiPost, apiPut } from "@/lib/api"
import type { PaginaCursor } from "@/types/api"
import type {
  ConfiguracionConvocatoria,
  ConvocatoriaAdmin,
  ConvocatoriaPublica,
  CrearConvocatoriaPayload,
  ResumenConvocatoriaAdmin,
} from "@/types/convocatoria"

/**
 * Consulta la convocatoria pública oficial activa para el portal (FT14).
 * Consume GET /api/v1/competitions. Solo devuelve convocatorias aprobadas y publicadas.
 * Si no hay convocatorias activas (página vacía), devuelve null sin inventar datos ficticios.
 */
export async function obtenerConvocatoriaPublica(
  signal?: AbortSignal,
): Promise<ConvocatoriaPublica | null> {
  const res = await apiGet<PaginaCursor<ConvocatoriaPublica>>("/competitions", {
    params: { limit: 1 },
    signal,
  })

  if (res?.items && res.items.length > 0) {
    return res.items[0]
  }

  return null
}

/**
 * Consulta el detalle público de una convocatoria por su ID.
 * Consume GET /api/v1/competitions/{id}.
 */
export async function obtenerDetallePublico(
  id: string,
  signal?: AbortSignal,
): Promise<ConvocatoriaPublica> {
  return apiGet<ConvocatoriaPublica>(
    `/competitions/${encodeURIComponent(id)}`,
    {
      signal,
    },
  )
}

/**
 * Lista las convocatorias públicas publicadas con paginación por cursor.
 */
export async function listarConvocatoriasPublicas(
  cursor?: string,
  signal?: AbortSignal,
): Promise<PaginaCursor<ConvocatoriaPublica>> {
  return apiGet<PaginaCursor<ConvocatoriaPublica>>("/competitions", {
    params: { limit: 20, cursor },
    signal,
  })
}

/**
 * Lista convocatorias en el módulo de administración (revisión T9 / gestión T18).
 * Consume GET /api/v1/admin/competitions.
 */
export async function listarConvocatoriasAdmin(
  cursor?: string,
  signal?: AbortSignal,
): Promise<PaginaCursor<ResumenConvocatoriaAdmin>> {
  return apiGet<PaginaCursor<ResumenConvocatoriaAdmin>>("/admin/competitions", {
    params: { limit: 20, cursor },
    signal,
  })
}

/**
 * Consulta la configuración administrativa completa y versión de una convocatoria (T18).
 * Consume GET /api/v1/admin/competitions/{id}.
 */
export async function obtenerConvocatoriaAdmin(
  id: string,
  signal?: AbortSignal,
): Promise<ConvocatoriaAdmin> {
  return apiGet<ConvocatoriaAdmin>(
    `/admin/competitions/${encodeURIComponent(id)}`,
    { signal },
  )
}

/**
 * Crea una nueva convocatoria institucional en estado inicial deshabilitado (T18).
 * Consume POST /api/v1/admin/competitions. Solo rol administrador.
 */
export async function crearConvocatoria(
  payload: CrearConvocatoriaPayload,
  signal?: AbortSignal,
): Promise<ConvocatoriaAdmin> {
  return apiPost<ConvocatoriaAdmin>("/admin/competitions", payload, { signal })
}

/**
 * Actualiza la configuración, aprobación, publicación, fases y habilitación del EF-12 (T18).
 * Consume PUT /api/v1/admin/competitions/{id}/configuration con control optimista de versión.
 * Devuelve 409 si la versión cambió en el servidor.
 */
export async function guardarConfiguracionConvocatoria(
  id: string,
  config: ConfiguracionConvocatoria,
  signal?: AbortSignal,
): Promise<ConvocatoriaAdmin> {
  return apiPut<ConvocatoriaAdmin>(
    `/admin/competitions/${encodeURIComponent(id)}/configuration`,
    config,
    { signal },
  )
}
