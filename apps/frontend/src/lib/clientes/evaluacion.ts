import { apiGet, apiPost } from "../api"
import { generarUuid } from "../uuid"
import type { PaginaCursor } from "@/types/api"

export const LIMITE_EVIDENCIA_BYTES = 2 * 1024 * 1024 // 2 MiB

export interface PoliticaOposicionDto {
  instrumento: string
  versionInstrumento: string
  modoPsicotecnico: "puntaje" | "concepto"
  maximoPsicotecnico: number | null
  conclusiones: string[]
  conclusionesAdmitidas: string[]
  maximoTeorico: number
  pesoMeritos: number
  pesoTeorica: number
  pesoPsicotecnico: number
  maximoTotal: number
  umbral: number
  redondeo: "inferior" | "mas_cercano"
}

export interface TerminoPlazoDto {
  cantidad: number
  unidad: "minutos_transcurridos" | "dias_calendario" | "dias_habiles"
  incluyeDiaInicial: boolean
  horaLimite: string
}

export interface CalendarioPlazoDto {
  finVentana: "exigir_termino_completo" | "limitar_a_fase"
  zonaHoraria: string
  diasLaborables: number[]
  feriados: string[]
}

export interface PoliticaReconsideracionDto {
  postulante: TerminoPlazoDto
  talentoHumano: TerminoPlazoDto
  comision: TerminoPlazoDto
  calendario: CalendarioPlazoDto
}

export interface PoliticaEvaluacionDto {
  id: string
  convocatoriaId: string
  sector: string
  tipo: "oposicion" | "reconsideracion"
  version: number
  definicion: {
    oposicion?: PoliticaOposicionDto
    reconsideracion?: PoliticaReconsideracionDto
  }
  hash: string
  aprobada: boolean
  referenciaAprobacion?: string
  responsableId: string
  creadaEn: string
  aprobadaPor?: string
  aprobadaEn?: string
}

export interface EvidenciaEvaluacionDto {
  id: string
  postulacionId: string
  responsableId: string
  nombre: string
  tamano: number
  sha256: string
  creadaEn: string
}

export interface RegistroPsicotecnicoDto {
  id: string
  postulacionId: string
  version: number
  reglasId: string
  hashReglas: string
  hashEnvio: string
  instrumento: string
  versionInstrumento: string
  puntaje: number | null
  maximo: number | null
  conclusion: string
  evaluadaEn: string
  respaldo: EvidenciaEvaluacionDto
  motivo: string
  responsableId: string
  registradaEn: string
}

export interface ComponenteDesgloseDto {
  fase: string
  resultadoId: string
  puntajeOrigen: number
  maximoOrigen: number
  pesoPuntosBase: number
  aporte: number
}

export interface ResultadoPublicadoDto {
  id: string
  postulacionId: string
  fase: string
  version: number
  sustituyeId?: string
  fuenteId: string
  reglasId: string
  hashReglas: string
  hashEnvio: string
  puntaje: number | null
  maximo: number | null
  habilitaAvance: boolean
  motivo: string
  responsableId: string
  publicadaEn: string
  versionMotor?: string
  desglose?: ComponenteDesgloseDto[]
}

export interface EstadoExpedienteEvaluacionDto {
  psicotecnicaRegistrada?: RegistroPsicotecnicoDto
  resultados: Record<string, ResultadoPublicadoDto>
}

export interface CasoEvaluacionDto {
  sector: string
  expediente: EstadoExpedienteEvaluacionDto
  reglasOposicion?: PoliticaEvaluacionDto
  reglasReconsideracion?: PoliticaEvaluacionDto
  fuentesPublicadas: Record<string, ResultadoPublicadoDto>
  reconsideracion?: unknown
  consultadoEn: string
}

export interface HistorialEvaluacionItemDto {
  id: string
  accion: string
  fecha: string
  detalle?: Record<string, unknown> | null
}

export interface EnlaceDescargaEvaluacionDto {
  url: string
  expiraEn: string
}

export interface EntradaPsicotecnicaDto {
  versionAnterior: number
  reglasId: string
  evaluadaEn: string
  puntaje: number | null
  conclusion: string
  motivo: string
  respaldo: {
    nombre: string
    pdfBase64: string
  }
}

export interface EntradaPublicarPsicotecnicaDto {
  versionAnterior: number
  fuenteId: string
  reglasId: string
  sustituyeId?: string
  motivo: string
}

export interface EntradaConsolidacionDto {
  versionAnterior: number
  reglasId: string
  sustituyeId?: string
  motivo: string
  fuentesEsperadas: {
    meritos: string
    oposicion_teorica: string
    oposicion_psicotecnica: string
  }
}

/**
 * Valida que un archivo sea un PDF de máximo 2 MiB y extrae su contenido en Base64 sin cabecera de datos.
 */
export async function validarYConvertirPdfBase64(
  archivo: File,
): Promise<{ nombre: string; pdfBase64: string }> {
  if (archivo.size > LIMITE_EVIDENCIA_BYTES) {
    throw new Error(
      `El archivo supera el límite de 2 MiB permitido (${(archivo.size / (1024 * 1024)).toFixed(2)} MiB).`,
    )
  }

  const esPdf =
    archivo.type === "application/pdf" ||
    archivo.name.toLowerCase().endsWith(".pdf")
  if (!esPdf) {
    throw new Error("Solo se admiten documentos en formato PDF.")
  }

  const buffer = await archivo.arrayBuffer()
  const bytes = new Uint8Array(buffer)
  let binary = ""
  for (let i = 0; i < bytes.byteLength; i++) {
    binary += String.fromCharCode(bytes[i])
  }
  const base64 = btoa(binary)

  return {
    nombre: archivo.name,
    pdfBase64: base64,
  }
}

/**
 * Obtiene los resultados de evaluación publicados visibles para el titular (FT20).
 * Omite registros provisionales que aún no han sido publicados oficialmente.
 */
export async function obtenerEvaluacionPostulante(
  submissionId: string,
  signal?: AbortSignal,
): Promise<CasoEvaluacionDto> {
  return apiGet<CasoEvaluacionDto>(
    `/evaluations/submissions/${encodeURIComponent(submissionId)}`,
    { signal },
  )
}

/**
 * Obtiene el historial de evaluaciones del postulante paginado por cursor (FT20).
 */
export async function obtenerHistorialEvaluacionPostulante(
  submissionId: string,
  params?: { limit?: number; cursor?: string },
  signal?: AbortSignal,
): Promise<PaginaCursor<HistorialEvaluacionItemDto>> {
  return apiGet<PaginaCursor<HistorialEvaluacionItemDto>>(
    `/evaluations/submissions/${encodeURIComponent(submissionId)}/history`,
    { params, signal },
  )
}

/**
 * Obtiene el enlace firmado temporal de descarga para evidencia de evaluación del titular (FT20).
 */
export async function descargarEvidenciaPostulante(
  submissionId: string,
  evidenceId: string,
  signal?: AbortSignal,
): Promise<EnlaceDescargaEvaluacionDto> {
  return apiGet<EnlaceDescargaEvaluacionDto>(
    `/evaluations/submissions/${encodeURIComponent(submissionId)}/evidence/${encodeURIComponent(evidenceId)}/download`,
    { signal },
  )
}

/**
 * Obtiene el expediente de evaluación completo para roles analista o administrador (FT20).
 * Incluye registros psicotécnicos provisionales y políticas vigentes.
 */
export async function obtenerExpedienteGradesAdmin(
  submissionId: string,
  signal?: AbortSignal,
): Promise<CasoEvaluacionDto> {
  return apiGet<CasoEvaluacionDto>(
    `/admin/grades/submissions/${encodeURIComponent(submissionId)}`,
    { signal },
  )
}

/**
 * Obtiene el historial administrativo de calificaciones y revisiones paginado por cursor (FT20).
 */
export async function obtenerHistorialGradesAdmin(
  submissionId: string,
  params?: { limit?: number; cursor?: string },
  signal?: AbortSignal,
): Promise<PaginaCursor<HistorialEvaluacionItemDto>> {
  return apiGet<PaginaCursor<HistorialEvaluacionItemDto>>(
    `/admin/grades/submissions/${encodeURIComponent(submissionId)}/history`,
    { params, signal },
  )
}

/**
 * Obtiene el enlace firmado temporal de descarga de evidencia para analista o administrador (FT20).
 */
export async function descargarEvidenciaAdmin(
  submissionId: string,
  evidenceId: string,
  signal?: AbortSignal,
): Promise<EnlaceDescargaEvaluacionDto> {
  return apiGet<EnlaceDescargaEvaluacionDto>(
    `/admin/grades/submissions/${encodeURIComponent(submissionId)}/evidence/${encodeURIComponent(evidenceId)}/download`,
    { signal },
  )
}

/**
 * Registra un informe psicotécnico provisional para la postulación (FT20).
 * Requiere Idempotency-Key para garantizar atomicidad y reintentos seguros.
 */
export async function registrarPsicotecnica(
  submissionId: string,
  entrada: EntradaPsicotecnicaDto,
  idempotencyKey = generarUuid(),
  signal?: AbortSignal,
): Promise<RegistroPsicotecnicoDto> {
  return apiPost<RegistroPsicotecnicoDto>(
    `/admin/grades/submissions/${encodeURIComponent(submissionId)}/psychometric`,
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
 * Publica oficialmente el resultado psicotécnico de la postulación (FT20).
 * Exclusivo para el rol Administrador (Comisión de Selección).
 */
export async function publicarPsicotecnica(
  submissionId: string,
  entrada: EntradaPublicarPsicotecnicaDto,
  idempotencyKey = generarUuid(),
  signal?: AbortSignal,
): Promise<ResultadoPublicadoDto> {
  return apiPost<ResultadoPublicadoDto>(
    `/admin/grades/submissions/${encodeURIComponent(submissionId)}/psychometric/publish`,
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
 * Consolida definitivamente las notas de méritos, teórica y psicotécnica (FT20).
 * Exclusivo para el rol Administrador (Comisión de Selección).
 */
export async function consolidarExpediente(
  submissionId: string,
  entrada: EntradaConsolidacionDto,
  idempotencyKey = generarUuid(),
  signal?: AbortSignal,
): Promise<ResultadoPublicadoDto> {
  return apiPost<ResultadoPublicadoDto>(
    `/admin/grades/submissions/${encodeURIComponent(submissionId)}/consolidate`,
    entrada,
    {
      headers: {
        "Idempotency-Key": idempotencyKey,
      },
      signal,
    },
  )
}
