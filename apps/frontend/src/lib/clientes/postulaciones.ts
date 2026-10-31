import { apiDelete, apiGet, apiPost, apiPut } from "../api"
import type { Pagina, PaginaCursor } from "@/types/api"

/**
 * Estado en el ciclo de vida de la postulación.
 */
export type EstadoPostulacion =
  "draft" | "submitted" | "under_review" | "approved" | "rejected"

/**
 * Avance individual de una de las 9 secciones.
 */
export interface SeccionAvance {
  number: number
  complete: boolean
}

/**
 * Avance global del formulario EF-12.
 */
export interface ProgresoPostulacion {
  total: number
  completed: number
  percentage: number
  sections: SeccionAvance[]
}

/**
 * Representación pública de una postulación devuelta por la API.
 */
export interface PostulacionDto {
  id: string
  applicantId: string
  code: string
  status: EstadoPostulacion
  content: Record<string, unknown>
  hash?: string
  submittedAt?: string
  createdAt: string
  updatedAt: string
  progress: ProgresoPostulacion
}

/**
 * Petición para crear una nueva postulación en borrador.
 */
export interface CrearPostulacionDto {
  applicantId?: string
  code?: string
  content?: Record<string, unknown>
}

/**
 * Parámetros de búsqueda y paginación para el listado de postulaciones.
 */
export interface ListarPostulacionesParams {
  applicantId?: string
  page?: number
  pageSize?: number
  limit?: number
  cursor?: string
}

/**
 * Abre una nueva postulación en estado borrador.
 */
export async function crearPostulacion(
  dto: CrearPostulacionDto = {},
  signal?: AbortSignal,
): Promise<PostulacionDto> {
  return apiPost<PostulacionDto>("/submissions", dto, { signal })
}

/**
 * Obtiene el detalle completo y progreso de una postulación por su identificador.
 */
export async function obtenerPostulacion(
  id: string,
  signal?: AbortSignal,
): Promise<PostulacionDto> {
  return apiGet<PostulacionDto>(`/submissions/${id}`, { signal })
}

/**
 * Lista las postulaciones del postulante con soporte para paginación por offset o cursor.
 */
export async function listarPostulaciones(
  params?: ListarPostulacionesParams,
  signal?: AbortSignal,
): Promise<Pagina<PostulacionDto>> {
  return apiGet<Pagina<PostulacionDto>>("/submissions", {
    params: {
      applicantId: params?.applicantId,
      page: params?.page,
      pageSize: params?.pageSize,
      limit: params?.limit,
      cursor: params?.cursor,
    },
    signal,
  })
}

/**
 * Lista las postulaciones usando paginación por cursor.
 */
export async function listarPostulacionesCursor(
  params?: ListarPostulacionesParams,
  signal?: AbortSignal,
): Promise<PaginaCursor<PostulacionDto>> {
  return apiGet<PaginaCursor<PostulacionDto>>("/submissions", {
    params: {
      applicantId: params?.applicantId,
      limit: params?.limit,
      cursor: params?.cursor,
    },
    signal,
  })
}

/**
 * Actualiza el contenido global del formulario en borrador.
 */
export async function actualizarPostulacion(
  id: string,
  content: Record<string, unknown>,
  signal?: AbortSignal,
): Promise<PostulacionDto> {
  return apiPut<PostulacionDto>(`/submissions/${id}`, { content }, { signal })
}

/**
 * Guarda o actualiza los campos de una sección específica (1 a 9) y recalcula el avance.
 */
export async function guardarSeccion(
  id: string,
  seccion: number,
  datosSeccion: unknown,
  signal?: AbortSignal,
): Promise<PostulacionDto> {
  return apiPost<PostulacionDto>(
    `/submissions/${id}/sections/${seccion}`,
    datosSeccion,
    { signal },
  )
}

/**
 * Sella la postulación con el consentimiento legal, generando el hash criptográfico inmutable.
 */
export async function sellarPostulacion(
  id: string,
  consentimiento?: unknown,
  signal?: AbortSignal,
): Promise<PostulacionDto> {
  return apiPost<PostulacionDto>(
    `/submissions/${id}/submit`,
    consentimiento ? { consentimiento } : {},
    { signal },
  )
}

/**
 * Elimina una postulación que aún permanezca en estado borrador.
 */
export async function eliminarPostulacion(
  id: string,
  signal?: AbortSignal,
): Promise<void> {
  return apiDelete<void>(`/submissions/${id}`, { signal })
}
