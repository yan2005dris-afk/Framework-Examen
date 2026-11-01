import { apiGet, apiPost } from "../api"
import type { PaginaCursor } from "@/types/api"

export interface ReglaCriterio {
  codigo: string
  nombre: string
  metrica:
    | "education_count"
    | "experience_days"
    | "publication_count"
    | "training_hours"
  filtro?: string
  tamanoUnidad: number
  centipuntosPorUnidad: number
  maximoCentipuntos: number
  pesoPuntosBase: number
  requiereEvidencia: boolean
}

export interface ReglaRequisito {
  codigo: string
  nombre: string
}

export interface ReglaInhabilidad {
  pregunta: string
  respuestaAdmitida: "Sí" | "No"
}

export interface DefinicionReglas {
  criterios: ReglaCriterio[]
  requisitos: ReglaRequisito[]
  inhabilidades: ReglaInhabilidad[]
  maximoTotalCentipuntos?: number
  umbralCentipuntos?: number
  redondeo: "inferior" | "mas_cercano"
  experiencias: "unir_intervalos" | "sumar_registros"
  computoDias: "dias_transcurridos" | "dias_inclusivos"
  desempates: string[]
}

export interface ReglasMeritosDto {
  id: string
  convocatoriaId: string
  sector: string
  version: number
  definicion: DefinicionReglas
  hashDefinicion: string
  aprobada: boolean
  referenciaAprobacion?: string
  aprobadaPor?: string
  aprobadaEn?: string
  responsableId: string
  creadaEn: string
}

export interface DesgloseCriterio {
  criterio: string
  unidadesVerificadas: number
  centipuntos: number
}

export interface EvaluacionMeritos {
  totalCentipuntos: number
  maximoCentipuntos: number
  umbralCentipuntos: number
  puedeAvanzar: boolean
  desglose: DesgloseCriterio[]
  impedimentos: string[]
  desempate: DesgloseCriterio[]
  desempateManual: boolean
}

export interface RevisionDocumento {
  seccion: number
  registroId: string
  documentoId: string
  resultado: "valido" | "invalido" | "no_presentado"
  observacion: string
}

export interface RevisionRequisito {
  codigo: string
  cumple: boolean
  observacion: string
}

export interface RevisionInhabilidad {
  pregunta: string
  verificada: boolean
  observacion: string
}

export interface RevisionInput {
  versionAnterior: number
  reglasId: string
  documentos: RevisionDocumento[]
  requisitos: RevisionRequisito[]
  inhabilidades: RevisionInhabilidad[]
  motivo: string
}

export interface RevisionMeritosDto {
  id: string
  postulacionId: string
  hashEnvio: string
  version: number
  revision: RevisionInput
  responsableId: string
  creadaEn: string
}

export interface CalculoMeritosDto {
  versionMotor: string
  id: string
  postulacionId: string
  version: number
  revisionId: string
  reglasId: string
  hashReglas: string
  hashEnvio: string
  evaluacion: EvaluacionMeritos
  motivo: string
  responsableId: string
  creadoEn: string
}

export interface PublicacionMeritosDto {
  id: string
  postulacionId: string
  version: number
  calculoId: string
  sustituyeId?: string
  decision: "avanza" | "excluido"
  motivo: string
  evaluacion: EvaluacionMeritos
  hashEnvio: string
  reglasId: string
  hashReglas: string
  responsableId: string
  publicadaEn: string
}

export interface ExpedienteMeritosAdminDto {
  reglasActivas?: ReglasMeritosDto | null
  revision?: RevisionMeritosDto | null
  calculo?: CalculoMeritosDto | null
  publicacion?: PublicacionMeritosDto | null
}

export interface ResultadoMeritosPostulanteDto {
  publicado: boolean
  habilitadoFaseTecnica: boolean
  resultado: PublicacionMeritosDto | null
}

/**
 * Convierte centipuntos enteros a valor numérico de puntos con 2 decimales (100 centipuntos = 1 punto).
 */
export function centipuntosAPuntos(centipuntos: number): string {
  return (centipuntos / 100).toFixed(2)
}

/**
 * Consulta el estado de méritos desde la vista del postulante (sólo entrega resoluciones publicadas).
 */
export async function obtenerResultadoMeritosPostulante(
  submissionId: string,
  signal?: AbortSignal,
): Promise<ResultadoMeritosPostulanteDto> {
  return apiGet<ResultadoMeritosPostulanteDto>(
    `/merits/submissions/${encodeURIComponent(submissionId)}`,
    { signal },
  )
}

/**
 * Consulta el expediente completo de méritos para el analista o administrador (reglas, última revisión, cálculo y publicación).
 */
export async function obtenerExpedienteMeritosAdmin(
  submissionId: string,
  signal?: AbortSignal,
): Promise<ExpedienteMeritosAdminDto> {
  return apiGet<ExpedienteMeritosAdminDto>(
    `/admin/merits/submissions/${encodeURIComponent(submissionId)}`,
    { signal },
  )
}

/**
 * Registra una nueva revisión de documentos, requisitos e inhabilidades.
 */
export async function crearRevisionMeritos(
  submissionId: string,
  input: RevisionInput,
  signal?: AbortSignal,
): Promise<RevisionMeritosDto> {
  return apiPost<RevisionMeritosDto>(
    `/admin/merits/submissions/${encodeURIComponent(submissionId)}/review`,
    input,
    { signal },
  )
}

/**
 * Solicita al motor del servidor el cálculo reproducible de méritos basado en la revisión vigente.
 */
export async function calcularMeritos(
  submissionId: string,
  input: { revisionId: string; versionAnterior: number; motivo: string },
  signal?: AbortSignal,
): Promise<CalculoMeritosDto> {
  return apiPost<CalculoMeritosDto>(
    `/admin/merits/submissions/${encodeURIComponent(submissionId)}/calculate`,
    input,
    { signal },
  )
}

/**
 * Publica una resolución oficial de méritos (avanza o excluido) por parte de un administrador.
 */
export async function publicarMeritos(
  submissionId: string,
  input: {
    calculoId: string
    versionAnterior: number
    decision: "avanza" | "excluido"
    motivo: string
    sustituyeId?: string
  },
  signal?: AbortSignal,
): Promise<PublicacionMeritosDto> {
  return apiPost<PublicacionMeritosDto>(
    `/admin/merits/submissions/${encodeURIComponent(submissionId)}/publish`,
    input,
    { signal },
  )
}

/**
 * Consulta las reglas activas de méritos por su identificador.
 */
export async function obtenerReglasMeritos(
  reglasId: string,
  signal?: AbortSignal,
): Promise<ReglasMeritosDto> {
  return apiGet<ReglasMeritosDto>(
    `/admin/merits/rules/${encodeURIComponent(reglasId)}`,
    { signal },
  )
}

/**
 * Lista las versiones de reglas aprobadas y borradores para una convocatoria y sector.
 */
export async function listarReglasMeritos(
  convocatoriaId: string,
  sector: string,
  params?: { limit?: number; cursor?: string },
  signal?: AbortSignal,
): Promise<PaginaCursor<ReglasMeritosDto>> {
  return apiGet<PaginaCursor<ReglasMeritosDto>>(
    `/admin/merits/competitions/${encodeURIComponent(convocatoriaId)}/sectors/${encodeURIComponent(sector)}/rules`,
    { params, signal },
  )
}
