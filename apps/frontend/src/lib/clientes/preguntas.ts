import { apiGet, apiGetBlob, apiPost, apiPut } from "../api"
import type { PaginaCursor } from "@/types/api"
import type { ResultadoPublicadoDto } from "./evaluacion"

export interface OpcionPreguntaDto {
  id: string
  texto: string
}

export interface PreguntaBancoDto {
  id: string
  enunciado: string
  opciones: OpcionPreguntaDto[]
  correcta: string
}

export interface EntradaBancoPreguntasDto {
  versionAnterior: number
  referencia: string
  preguntas: PreguntaBancoDto[]
}

export interface BancoPreguntasMetadataDto {
  id: string
  convocatoriaId: string
  sector: string
  version: number
  definicion: EntradaBancoPreguntasDto
  hash: string
  responsableId: string
  creadaEn: string
}

export interface ReglasConfiguracionExamenDto {
  cantidadPreguntas: number
  duracionSegundos: number
  maximo: number
  umbral: number
  redondeo: "inferior" | "mas_cercano"
  politicaIntentos: "un_intento"
  politicaUnicidad: "orden_preguntas_unico_rechazar_agotamiento"
  referenciaAprobacion: string
  abreEn: string
  cierraEn: string
}

export interface ConfiguracionExamenDto {
  id: string
  bancoId: string
  convocatoriaId: string
  sector: string
  reglas: ReglasConfiguracionExamenDto
  hash: string
  creadaEn: string
  responsableId: string
}

export interface EntradaHabilitacionExamenDto {
  configuracionId: string
  resultadoMeritosId: string
}

export interface VistaExamenDto {
  id: string
  postulacionId: string
  version: number
  estado: "active" | "finished"
  iniciadaEn: string
  venceEn: string
  servidorEn: string
  segundosRestantes: number
  cantidadPreguntas: number
  respondidas: number
  resultado?: ResultadoPublicadoDto
}

export interface PreguntaExamenPublicaDto {
  id: string
  enunciado: string
  opciones: OpcionPreguntaDto[]
  seleccionada?: string
  indice: number
  versionIntento: number
}

export interface EntradaResponderExamenDto {
  versionAnterior: number
  preguntaId: string
  opcionId: string
}

/**
 * Carga o reemplaza un banco de preguntas mediante una nueva versión completa inmutable (FT24).
 * Sólo administradores con permiso preguntas.gestionar.
 */
export async function crearVersionBancoPreguntas(
  period: string,
  sector: string,
  entrada: EntradaBancoPreguntasDto,
  signal?: AbortSignal,
): Promise<BancoPreguntasMetadataDto> {
  return apiPost<BancoPreguntasMetadataDto>(
    `/admin/question-bank/competitions/${encodeURIComponent(period)}/sectors/${encodeURIComponent(sector)}/versions`,
    entrada,
    { signal },
  )
}

/**
 * Consulta el listado histórico de versiones de banco para una convocatoria y sector (FT24).
 */
export async function listarVersionesBancoPreguntas(
  period: string,
  sector: string,
  params?: { limit?: number; cursor?: string },
  signal?: AbortSignal,
): Promise<PaginaCursor<BancoPreguntasMetadataDto>> {
  return apiGet<PaginaCursor<BancoPreguntasMetadataDto>>(
    `/admin/question-bank/competitions/${encodeURIComponent(period)}/sectors/${encodeURIComponent(sector)}/versions`,
    { params, signal },
  )
}

/**
 * Consulta el banco completo con matriz de respuestas correctas (lectura auditada para Administrador) (FT24).
 */
export async function obtenerVersionBancoPreguntas(
  id: string,
  signal?: AbortSignal,
): Promise<BancoPreguntasMetadataDto> {
  return apiGet<BancoPreguntasMetadataDto>(
    `/admin/question-bank/versions/${encodeURIComponent(id)}`,
    { signal },
  )
}

/**
 * Aprueba y congela la configuración de evaluación técnica para una versión de banco (FT25).
 */
export async function configurarVersionBanco(
  bankId: string,
  reglas: ReglasConfiguracionExamenDto,
  signal?: AbortSignal,
): Promise<ConfiguracionExamenDto> {
  return apiPost<ConfiguracionExamenDto>(
    `/admin/question-bank/versions/${encodeURIComponent(bankId)}/configurations`,
    reglas,
    { signal },
  )
}

/**
 * Habilita individualmente a un postulante que superó la fase de méritos (FT25).
 */
export async function habilitarPostulanteExamen(
  submissionId: string,
  entrada: EntradaHabilitacionExamenDto,
  signal?: AbortSignal,
): Promise<{ fase: string }> {
  return apiPost<{ fase: string }>(
    `/admin/question-bank/submissions/${encodeURIComponent(submissionId)}/enable`,
    entrada,
    { signal },
  )
}

/**
 * Consulta el estado y reloj del examen activo o finalizado del postulante (FT25-FT26).
 */
export async function obtenerEstadoExamen(
  submissionId: string,
  signal?: AbortSignal,
): Promise<VistaExamenDto> {
  return apiGet<VistaExamenDto>(
    `/exams/submissions/${encodeURIComponent(submissionId)}`,
    { signal },
  )
}

/**
 * Inicia el intento oficial o recupera el intento existente si ya fue iniciado (FT26).
 */
export async function iniciarExamen(
  submissionId: string,
  signal?: AbortSignal,
): Promise<VistaExamenDto> {
  return apiPost<VistaExamenDto>(
    `/exams/submissions/${encodeURIComponent(submissionId)}/start`,
    {},
    { signal },
  )
}

/**
 * Obtiene una pregunta individual por índice (1 a cantidadPreguntas) sin exponer la matriz completa (FT26).
 */
export async function obtenerPreguntaExamen(
  submissionId: string,
  index: number,
  signal?: AbortSignal,
): Promise<PreguntaExamenPublicaDto> {
  return apiGet<PreguntaExamenPublicaDto>(
    `/exams/submissions/${encodeURIComponent(submissionId)}/questions/${index}`,
    { signal },
  )
}

/**
 * Guarda una respuesta individual con control optimista de versión del intento (FT26).
 */
export async function responderPreguntaExamen(
  submissionId: string,
  entrada: EntradaResponderExamenDto,
  signal?: AbortSignal,
): Promise<VistaExamenDto> {
  return apiPut<VistaExamenDto>(
    `/exams/submissions/${encodeURIComponent(submissionId)}/answers`,
    entrada,
    { signal },
  )
}

/**
 * Cierre definitivo del examen con calificación en el servidor (FT26-FT27).
 */
export async function finalizarExamen(
  submissionId: string,
  signal?: AbortSignal,
): Promise<VistaExamenDto> {
  return apiPost<VistaExamenDto>(
    `/exams/submissions/${encodeURIComponent(submissionId)}/finish`,
    {},
    { signal },
  )
}

/**
 * Descarga del reporte PDF oficial del examen rendido (FT27).
 */
export async function descargarReporteExamenPdf(
  submissionId: string,
  signal?: AbortSignal,
): Promise<Blob> {
  return apiGetBlob(
    `/exams/submissions/${encodeURIComponent(submissionId)}/report.pdf`,
    { signal },
  )
}

/**
 * Consulta administrativa auditada del examen rendido (para analista o administrador).
 */
export async function obtenerExamenAdmin(
  submissionId: string,
  signal?: AbortSignal,
): Promise<VistaExamenDto> {
  return apiGet<VistaExamenDto>(
    `/admin/grades/submissions/${encodeURIComponent(submissionId)}/exam`,
    { signal },
  )
}

/**
 * Descarga administrativa auditada del reporte PDF del examen (para analista o administrador).
 */
export async function descargarReporteExamenPdfAdmin(
  submissionId: string,
  signal?: AbortSignal,
): Promise<Blob> {
  return apiGetBlob(
    `/admin/grades/submissions/${encodeURIComponent(submissionId)}/exam/report.pdf`,
    { signal },
  )
}
