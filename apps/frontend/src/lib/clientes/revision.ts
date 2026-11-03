import { apiGet, apiGetBlob, type ParametrosConsulta } from "@/lib/api"
import { ErrorApi } from "@/lib/api-error"
import { esUuid } from "@/lib/uuid"
import type { PostulacionDto } from "./postulaciones"
import type { PaginaCursor } from "@/types/api"
import type { SectorPostulacion } from "@/types/concurso"

export interface FiltrosRevision {
  periodo: string // UUID o "legado"; nunca se infiere la convocatoria activa.
  sector: SectorPostulacion | ""
  estado: "" | "submitted"
}

export interface ConvocatoriaRevision {
  id: string
  codigo: string
  titulo: string
  apertura: string
  cierre: string
  cerrada: boolean
}

export interface DuracionRevision {
  anios: number
  meses: number
  dias: number
  texto: string
}
export interface ResumenRevision {
  id: string
  codigo: string
  convocatoria: string
  nombres: string
  apellidos: string
  sector: string
  status: string
  submittedAt: string
  hash: string
  titulos: { nivel: string; titulo: string }[]
  experiencias: {
    id: string
    tipo: string
    cargo: string
    duracion: DuracionRevision | null
    documentoId?: string
  }[]
  publicaciones: number
  horasCapacitacion: number
}

export interface ConteosRevision {
  consultadoEn: string
  actualizarEnSegundos: number
  conteos: {
    sector: string
    status: string
    postulantes: number
    postulaciones: number
  }[]
}

export type ExpedienteRevision = PostulacionDto & {
  convocatoriaId?: string | null
}
export interface DocumentoRevision {
  id: string
  submissionId: string
  section: number
  recordId: string
  name: string
  mime: string
  sizeBytes: number
  sha256: string
  createdAt: string
}
export interface EventoRevision {
  id: string
  actorId?: string
  rol: string
  accion: string
  tipoRecurso: string
  recursoId: string
  resultado: string
  createdAt: string
}
export type FormatoMatriz = "xlsx" | "pdf"
export const TIPOS_MATRIZ: Record<FormatoMatriz, string> = {
  xlsx: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
  pdf: "application/pdf",
}

/** Serializa una selección explícita; conteos y exportaciones no reciben cursores. */
export function parametrosRevision(
  filtros: FiltrosRevision,
): ParametrosConsulta {
  if (
    !filtros.periodo ||
    (filtros.periodo !== "legado" && !esUuid(filtros.periodo))
  ) {
    throw new Error(
      "Seleccione una convocatoria válida o el legado sin convocatoria.",
    )
  }
  if (
    !["", "afiliados", "jubilados", "empleadores"].includes(filtros.sector) ||
    !["", "submitted"].includes(filtros.estado)
  ) {
    throw new Error("Los filtros no son válidos.")
  }
  return {
    ...(filtros.periodo === "legado"
      ? { sinConvocatoria: true }
      : { convocatoriaId: filtros.periodo }),
    sector: filtros.sector || undefined,
    status: filtros.estado || undefined,
  }
}

export const listarConvocatoriasRevision = (
  cursor?: string,
  signal?: AbortSignal,
) =>
  apiGet<PaginaCursor<ConvocatoriaRevision>>("/admin/competitions", {
    params: { limit: 50, cursor },
    signal,
  })

export const listarRevision = (
  filtros: FiltrosRevision,
  cursor?: string,
  signal?: AbortSignal,
) =>
  apiGet<PaginaCursor<ResumenRevision>>("/admin/submissions", {
    params: { ...parametrosRevision(filtros), limit: 20, cursor },
    signal,
  })

export async function obtenerConteosRevision(
  filtros: FiltrosRevision,
  signal?: AbortSignal,
): Promise<ConteosRevision> {
  const respuesta = await apiGet<ConteosRevision>("/admin/submissions/stats", {
    params: parametrosRevision(filtros),
    signal,
  })
  if (
    !respuesta ||
    !Number.isFinite(Date.parse(respuesta.consultadoEn)) ||
    !Array.isArray(respuesta.conteos) ||
    !respuesta.conteos.every(
      (c) =>
        c &&
        typeof c.sector === "string" &&
        c.status === "submitted" &&
        Number.isSafeInteger(c.postulantes) &&
        c.postulantes >= 0 &&
        Number.isSafeInteger(c.postulaciones) &&
        c.postulaciones >= 0,
    )
  ) {
    throw new ErrorApi(
      "La respuesta de conteos no es válida. No se mostrarán cifras incompletas.",
      { codigo: "conteos_invalidos", estado: 502 },
    )
  }
  return respuesta
}

export const obtenerExpedienteRevision = (id: string, signal?: AbortSignal) =>
  apiGet<ExpedienteRevision>(`/admin/submissions/${encodeURIComponent(id)}`, {
    signal,
  })

export const listarDocumentosRevision = (id: string, signal?: AbortSignal) =>
  apiGet<DocumentoRevision[]>(
    `/admin/submissions/${encodeURIComponent(id)}/documents`,
    { signal },
  )

export const obtenerEnlaceDocumentoRevision = (
  id: string,
  signal?: AbortSignal,
) =>
  apiGet<{ url: string; expiresAt: string }>(
    `/admin/documents/${encodeURIComponent(id)}/download`,
    { signal },
  )

export const listarEventosRevision = (
  id: string,
  cursor?: string,
  signal?: AbortSignal,
) =>
  apiGet<PaginaCursor<EventoRevision>>(
    `/admin/submissions/${encodeURIComponent(id)}/events`,
    { params: { limit: 20, cursor }, signal },
  )

/** Conserva los bytes originales y rechaza respuestas JSON/HTML disfrazadas de archivo. */
export async function exportarMatrizRevision(
  filtros: FiltrosRevision,
  formato: FormatoMatriz,
  signal?: AbortSignal,
): Promise<Blob> {
  const archivo = await apiGetBlob(`/admin/submissions/export.${formato}`, {
    params: parametrosRevision(filtros),
    signal,
    headers: { Accept: TIPOS_MATRIZ[formato] },
  })
  if (
    !archivo ||
    archivo.size === 0 ||
    archivo.type.split(";")[0] !== TIPOS_MATRIZ[formato]
  ) {
    throw new ErrorApi(
      "El servidor no devolvió un archivo válido. Intente de nuevo.",
      { codigo: "archivo_invalido", estado: 502 },
    )
  }
  return archivo
}
