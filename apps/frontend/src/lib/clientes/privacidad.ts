import { apiGet, apiPost } from "../api"
import type { PaginaCursor } from "@/types/api"

export interface ReglasPoliticaPrivacidadDto {
  texto: string
  informacionTratamiento: string
  referenciaAprobacion: string
  diasAcceso: number
  diasDocumentos: number
  diasReportes: number
  diasPublicacionNombres: number
  diasCorreo: number
  diasRevisionConservacion: number
  disposicionHistoricos: "revision_humana_sin_borrado_automatico"
  permiteExportacion: boolean
  permiteCorreo: boolean
  permiteNombresPublicos: boolean
}

export interface PoliticaPrivacidadDto {
  versionControl: number
  restringido: boolean
  id: string
  convocatoriaId: string
  version: number
  definicion: ReglasPoliticaPrivacidadDto
  hash: string
  aprobadaPor: string
  publicadaEn: string
}

export interface PublicarPoliticaDto {
  versionAnterior: number
  definicion: ReglasPoliticaPrivacidadDto
}

export interface RestringirPoliticaDto {
  versionAnterior: number
  restringido: boolean
  motivo: string
}

export interface RevisionRetencionDto {
  postulacionId: string
  politicaId: string
  revisarDesde: string
}

/**
 * Obtiene la política de privacidad vigente para una convocatoria dada.
 * Endpoint público sin autenticación requerida.
 */
export async function obtenerPoliticaVigenteConvocatoria(
  convocatoriaId: string,
): Promise<PoliticaPrivacidadDto> {
  return apiGet<PoliticaPrivacidadDto>(
    `/api/v1/public/privacy/competitions/${convocatoriaId}`,
  )
}

/**
 * Obtiene una versión documental histórica exacta de una política de privacidad por su ID.
 * Endpoint público sin autenticación requerida.
 */
export async function obtenerPoliticaPorId(
  politicaId: string,
): Promise<PoliticaPrivacidadDto> {
  return apiGet<PoliticaPrivacidadDto>(
    `/api/v1/public/privacy/policies/${politicaId}`,
  )
}

/**
 * Publica una nueva versión de política de privacidad para una convocatoria (rol Administrador/Fases).
 */
export async function publicarPoliticaPrivacidad(
  convocatoriaId: string,
  datos: PublicarPoliticaDto,
): Promise<PoliticaPrivacidadDto> {
  return apiPost<PoliticaPrivacidadDto>(
    `/api/v1/admin/privacy/competitions/${convocatoriaId}/policies`,
    datos,
  )
}

/**
 * Aplica o levanta una restricción cautelar/administrativa sobre la privacidad de una convocatoria.
 */
export async function restringirPoliticaPrivacidad(
  convocatoriaId: string,
  datos: RestringirPoliticaDto,
): Promise<{ restringido: boolean }> {
  return apiPost<{ restringido: boolean }>(
    `/api/v1/admin/privacy/competitions/${convocatoriaId}/restriction`,
    datos,
  )
}

/**
 * Lista el historial de versiones de políticas de privacidad de una convocatoria (paginado).
 */
export async function listarPoliticasConvocatoria(
  convocatoriaId: string,
  params?: { limit?: number; cursor?: string },
): Promise<PaginaCursor<PoliticaPrivacidadDto>> {
  const q = new URLSearchParams()
  if (params?.limit) q.set("limit", String(params.limit))
  if (params?.cursor) q.set("cursor", params.cursor)
  const qs = q.toString() ? `?${q.toString()}` : ""
  return apiGet<PaginaCursor<PoliticaPrivacidadDto>>(
    `/api/v1/admin/privacy/competitions/${convocatoriaId}/policies${qs}`,
  )
}

/**
 * Lista las revisiones de retención vencidas para evaluación de conservación de expedientes.
 */
export async function listarRevisionesRetencion(
  convocatoriaId: string,
  params?: { limit?: number; cursor?: string },
): Promise<PaginaCursor<RevisionRetencionDto>> {
  const q = new URLSearchParams()
  if (params?.limit) q.set("limit", String(params.limit))
  if (params?.cursor) q.set("cursor", params.cursor)
  const qs = q.toString() ? `?${q.toString()}` : ""
  return apiGet<PaginaCursor<RevisionRetencionDto>>(
    `/api/v1/admin/privacy/competitions/${convocatoriaId}/retention-review${qs}`,
  )
}
