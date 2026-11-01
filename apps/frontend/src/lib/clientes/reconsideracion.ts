import { apiGet, apiPost } from "../api"
import { generarUuid } from "../uuid"
import type { PaginaCursor } from "@/types/api"
import type {
  EvidenciaEvaluacionDto,
  PoliticaEvaluacionDto,
  ResultadoPublicadoDto,
} from "./evaluacion"

export type FaseReconsideracion =
  "meritos" | "oposicion_teorica" | "oposicion_psicotecnica" | "resultados"

export type EstadoReconsideracion = "recibida" | "atendida" | "resuelta"

export type RecomendacionAtencion = "confirmar" | "rectificar"

export type DecisionResolucion = "confirmar" | "rectificar"

export interface AtencionReconsideracionDto {
  responsableId: string
  fecha: string
  observacion: string
  recomendacion: RecomendacionAtencion
}

export interface ResolucionReconsideracionDto {
  responsableId: string
  fecha: string
  decision: DecisionResolucion
  motivo: string
  resultadoVigente: ResultadoPublicadoDto
  diferenciaPuntaje?: number
}

export interface ReconsideracionDto {
  id: string
  postulacionId: string
  version: number
  estado: EstadoReconsideracion
  resultadoOriginal: ResultadoPublicadoDto
  reglasPlazo: PoliticaEvaluacionDto
  motivo: string
  respaldo?: EvidenciaEvaluacionDto
  documentosOriginales: string[]
  responsableId: string
  presentadaEn: string
  cierreVentana: string
  limitePresentacion: string
  limiteTalentoHumano: string
  limiteComision?: string
  atencion?: AtencionReconsideracionDto
  resolucion?: ResolucionReconsideracionDto
}

export interface EntradaPresentarReconsideracionDto {
  fase: FaseReconsideracion
  resultadoId: string
  motivo: string
  respaldo?: {
    nombre: string
    pdfBase64: string
  }
  documentosOriginales?: string[]
}

export interface EntradaAtenderReconsideracionDto {
  versionAnterior: number
  observacion: string
  recomendacion: RecomendacionAtencion
}

export interface EntradaResolverReconsideracionDto {
  versionAnterior: number
  decision: DecisionResolucion
  motivo: string
  resultadoRectificadoId?: string
}

/**
 * Presenta una solicitud de reconsideración formal por parte del postulante (FT21).
 * Vinculada al resultado publicado exacto y dentro del plazo establecido por el servidor.
 */
export async function presentarReconsideracion(
  submissionId: string,
  entrada: EntradaPresentarReconsideracionDto,
  idempotencyKey = generarUuid(),
  signal?: AbortSignal,
): Promise<ReconsideracionDto> {
  return apiPost<ReconsideracionDto>(
    `/reconsiderations/submissions/${encodeURIComponent(submissionId)}`,
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
 * Lista las solicitudes de reconsideración presentadas por el postulante (FT21).
 * Paginadas por cursor mediante el protocolo estándar del servidor.
 */
export async function listarReconsideracionesPostulante(
  submissionId: string,
  params?: { limit?: number; cursor?: string },
  signal?: AbortSignal,
): Promise<PaginaCursor<ReconsideracionDto>> {
  return apiGet<PaginaCursor<ReconsideracionDto>>(
    `/reconsiderations/submissions/${encodeURIComponent(submissionId)}`,
    { params, signal },
  )
}

/**
 * Obtiene el detalle y las actuaciones de una solicitud de reconsideración del postulante (FT21).
 */
export async function obtenerReconsideracionPostulante(
  submissionId: string,
  appealId: string,
  signal?: AbortSignal,
): Promise<ReconsideracionDto> {
  return apiGet<ReconsideracionDto>(
    `/reconsiderations/submissions/${encodeURIComponent(submissionId)}/${encodeURIComponent(appealId)}`,
    { signal },
  )
}

/**
 * Lista auditada de reconsideraciones del expediente para Analista o Administrador (FT21).
 */
export async function listarReconsideracionesAdmin(
  submissionId: string,
  params?: { limit?: number; cursor?: string },
  signal?: AbortSignal,
): Promise<PaginaCursor<ReconsideracionDto>> {
  return apiGet<PaginaCursor<ReconsideracionDto>>(
    `/admin/reconsiderations/submissions/${encodeURIComponent(submissionId)}`,
    { params, signal },
  )
}

/**
 * Obtiene el detalle auditado de una reconsideración para Analista o Administrador (FT21).
 */
export async function obtenerReconsideracionAdmin(
  submissionId: string,
  appealId: string,
  signal?: AbortSignal,
): Promise<ReconsideracionDto> {
  return apiGet<ReconsideracionDto>(
    `/admin/reconsiderations/submissions/${encodeURIComponent(submissionId)}/${encodeURIComponent(appealId)}`,
    { signal },
  )
}

/**
 * Registra la atención técnica de Talento Humano para la reconsideración (FT21).
 * Exclusivo para el rol Analista. Transiciona el estado a "atendida".
 */
export async function atenderReconsideracion(
  submissionId: string,
  appealId: string,
  entrada: EntradaAtenderReconsideracionDto,
  idempotencyKey = generarUuid(),
  signal?: AbortSignal,
): Promise<ReconsideracionDto> {
  return apiPost<ReconsideracionDto>(
    `/admin/reconsiderations/submissions/${encodeURIComponent(submissionId)}/${encodeURIComponent(appealId)}/attend`,
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
 * Registra la resolución definitiva de la Comisión de Selección sobre la reconsideración (FT21).
 * Exclusivo para el rol Administrador. Transiciona el estado a "resuelta".
 */
export async function resolverReconsideracion(
  submissionId: string,
  appealId: string,
  entrada: EntradaResolverReconsideracionDto,
  idempotencyKey = generarUuid(),
  signal?: AbortSignal,
): Promise<ReconsideracionDto> {
  return apiPost<ReconsideracionDto>(
    `/admin/reconsiderations/submissions/${encodeURIComponent(submissionId)}/${encodeURIComponent(appealId)}/resolve`,
    entrada,
    {
      headers: {
        "Idempotency-Key": idempotencyKey,
      },
      signal,
    },
  )
}
