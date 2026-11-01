import { apiGet, apiGetBlob, apiPost } from "../api"
import { generarUuid } from "../uuid"
import type { PaginaCursor } from "@/types/api"

export interface ReglasOrdenListaDto {
  orden: string[]
  referenciaOrden: string
}

export interface EntradaGenerarListaDto {
  reglas: ReglasOrdenListaDto
}

export interface ListaResultadosMetadataDto {
  id: string
  convocatoriaId: string
  sector: string
  version: number
  reglas: ReglasOrdenListaDto
  hashFuentes: string
  responsableId: string
  creadaEn: string
}

export interface ComponenteFaseResultadoDto {
  id: string
  postulacionId: string
  fase: string
  version: number
  puntaje?: number | null
  maximo?: number | null
  habilitaAvance: boolean
  motivo: string
}

export interface EntradaListaDetalleDto {
  postulacionId: string
  codigo: string
  hashEnvio: string
  nombres: string
  apellidos: string
  elegible: boolean
  puesto?: number
  empate: boolean
  consentimientoNombre: boolean
  resultados: Record<string, ComponenteFaseResultadoDto>
  reconsideraciones?: {
    id: string
    version: number
    resultadoEfectivoId: string
  }[]
}

export interface PublicacionListaMetadataDto {
  id: string
  listaId: string
  convocatoriaId: string
  sector: string
  version: number
  sustituyeId?: string
  referenciaAprobacion: string
  publica: boolean
  referenciaPrivacidad?: string
  responsableId: string
  publicadaEn: string
}

export interface EntradaPublicarListaDto {
  versionPublicadaAnterior: number
  sustituyeId?: string
  referenciaAprobacion: string
  publica: boolean
  referenciaPrivacidad?: string
}

export interface ProyeccionPublicaListaDto {
  id: string
  convocatoriaId: string
  sector: string
  version: number
  publicadaEn: string
  orden: string
  nombres: PaginaCursor<string>
}

/**
 * Genera una lista interna reproducible de resultados y elegibles para una convocatoria y sector (FT23).
 * Requiere Idempotency-Key.
 */
export async function generarListaResultados(
  period: string,
  sector: string,
  entrada: EntradaGenerarListaDto,
  idempotencyKey = generarUuid(),
  signal?: AbortSignal,
): Promise<ListaResultadosMetadataDto> {
  return apiPost<ListaResultadosMetadataDto>(
    `/admin/results/competitions/${encodeURIComponent(period)}/sectors/${encodeURIComponent(sector)}/lists`,
    entrada,
    {
      headers: {
        "Idempotency-Key": idempotencyKey,
      },
      signal,
    },
  )
}

/**
 * Consulta las versiones de listas internas generadas (FT23).
 */
export async function listarListasAdmin(
  period: string,
  sector: string,
  params?: { limit?: number; cursor?: string },
  signal?: AbortSignal,
): Promise<PaginaCursor<ListaResultadosMetadataDto>> {
  return apiGet<PaginaCursor<ListaResultadosMetadataDto>>(
    `/admin/results/competitions/${encodeURIComponent(period)}/sectors/${encodeURIComponent(sector)}/lists`,
    { params, signal },
  )
}

/**
 * Consulta las versiones oficiales publicadas de la lista (FT23).
 */
export async function listarPublicacionesListasAdmin(
  period: string,
  sector: string,
  params?: { limit?: number; cursor?: string },
  signal?: AbortSignal,
): Promise<PaginaCursor<PublicacionListaMetadataDto>> {
  return apiGet<PaginaCursor<PublicacionListaMetadataDto>>(
    `/admin/results/competitions/${encodeURIComponent(period)}/sectors/${encodeURIComponent(sector)}/publications`,
    { params, signal },
  )
}

/**
 * Consulta los metadatos y reglas de una lista interna (FT23).
 */
export async function obtenerListaMetadataAdmin(
  id: string,
  signal?: AbortSignal,
): Promise<ListaResultadosMetadataDto> {
  return apiGet<ListaResultadosMetadataDto>(
    `/admin/results/lists/${encodeURIComponent(id)}`,
    { signal },
  )
}

/**
 * Consulta las entradas paginadas del snapshot interno de la lista (FT23).
 */
export async function obtenerEntradasListaAdmin(
  id: string,
  params?: { limit?: number; cursor?: string },
  signal?: AbortSignal,
): Promise<PaginaCursor<EntradaListaDetalleDto>> {
  return apiGet<PaginaCursor<EntradaListaDetalleDto>>(
    `/admin/results/lists/${encodeURIComponent(id)}/entries`,
    { params, signal },
  )
}

/**
 * Descarga el archivo CSV oficial del snapshot interno de la lista (FT23).
 */
export async function descargarCsvListaAdmin(
  id: string,
  signal?: AbortSignal,
): Promise<Blob> {
  return apiGetBlob(
    `/admin/results/lists/${encodeURIComponent(id)}/export.csv`,
    { signal },
  )
}

/**
 * Publica oficialmente una lista de elegibles (interna o pública general) (FT23).
 * Exclusivo para el rol Administrador. Requiere Idempotency-Key.
 */
export async function publicarListaAdmin(
  id: string,
  entrada: EntradaPublicarListaDto,
  idempotencyKey = generarUuid(),
  signal?: AbortSignal,
): Promise<PublicacionListaMetadataDto> {
  return apiPost<PublicacionListaMetadataDto>(
    `/admin/results/lists/${encodeURIComponent(id)}/publish`,
    entrada,
    {
      headers: {
        "Idempotency-Key": idempotencyKey,
      },
      signal,
    },
  )
}

/**
 * Consulta pública sin sesión de la lista vigente de elegibles (FT23).
 * Proyección de nombres ordenada alfabéticamente sin notas ni datos sensibles.
 */
export async function obtenerResultadosPublicosVigentes(
  period: string,
  sector: string,
  params?: { limit?: number; cursor?: string },
  signal?: AbortSignal,
): Promise<ProyeccionPublicaListaDto> {
  return apiGet<ProyeccionPublicaListaDto>(
    `/public/results/competitions/${encodeURIComponent(period)}/sectors/${encodeURIComponent(sector)}`,
    { params, signal },
  )
}

/**
 * Consulta pública sin sesión de una versión histórica de la lista de elegibles (FT23).
 */
export async function obtenerResultadosPublicosHistoricos(
  publicationId: string,
  params?: { limit?: number; cursor?: string },
  signal?: AbortSignal,
): Promise<ProyeccionPublicaListaDto> {
  return apiGet<ProyeccionPublicaListaDto>(
    `/public/results/publications/${encodeURIComponent(publicationId)}`,
    { params, signal },
  )
}

