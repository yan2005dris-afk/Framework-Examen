import { apiDelete, apiGet, apiPost } from "../api"

export const LIMITE_ORGANIZACION_BYTES = 4 * 1024 * 1024 // 4 MiB (sección 4)
export const LIMITE_REGISTRO_BYTES = 2 * 1024 * 1024 // 2 MiB (secciones 5 a 8)
export const LIMITE_TOTAL_POSTULANTE_BYTES = 50 * 1024 * 1024 // 50 MiB

/**
 * Representación del documento devuelta por la API tras confirmar carga y metadatos.
 */
export interface DocumentoDto {
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

/**
 * Enlace temporal firmado para descarga segura sin pasar el binario por la API.
 */
export interface EnlaceDescargaDto {
  url: string
  expiresAt: string
}

/**
 * Formatea bytes en representación legible (KiB o MiB).
 */
export function formatearTamanoBytes(bytes: number): string {
  if (bytes < 1024 * 1024) {
    return `${(bytes / 1024).toFixed(1)} KiB`
  }
  return `${(bytes / (1024 * 1024)).toFixed(1)} MiB`
}

/**
 * Determina el límite de tamaño en bytes según la sección del EF-12.
 */
export function limitePorSeccion(seccion: number): number {
  if (seccion === 4) return LIMITE_ORGANIZACION_BYTES
  if (seccion >= 5 && seccion <= 8) return LIMITE_REGISTRO_BYTES
  throw new Error(`Sección ${seccion} no admite respaldos documentales.`)
}

/**
 * Calcula el resumen de cuota consumida de los documentos vigentes.
 */
export function calcularCuotaUtilizada(documentos: DocumentoDto[]): {
  bytes: number
  porcentaje: number
  formateado: string
  excedida: boolean
} {
  const bytes = documentos.reduce((acc, doc) => acc + (doc.sizeBytes || 0), 0)
  const porcentaje = Math.min(
    100,
    Math.round((bytes / LIMITE_TOTAL_POSTULANTE_BYTES) * 100),
  )
  return {
    bytes,
    porcentaje,
    formateado: `${formatearTamanoBytes(bytes)} de ${formatearTamanoBytes(LIMITE_TOTAL_POSTULANTE_BYTES)}`,
    excedida: bytes > LIMITE_TOTAL_POSTULANTE_BYTES,
  }
}

/**
 * Sube un archivo PDF mediante multipart/form-data vinculado a una sección y registro.
 * El registro debe existir previamente en el borrador guardado en el servidor.
 */
export async function subirDocumento(
  submissionId: string,
  seccion: number,
  archivo: File,
  recordId = "",
  signal?: AbortSignal,
): Promise<DocumentoDto> {
  const formData = new FormData()
  formData.append("section", String(seccion))
  formData.append("recordId", recordId)
  formData.append("file", archivo, archivo.name)

  return apiPost<DocumentoDto>(
    `/submissions/${encodeURIComponent(submissionId)}/documents`,
    formData,
    { signal },
  )
}

/**
 * Lista los documentos respaldados de una postulación.
 */
export async function listarDocumentos(
  submissionId: string,
  signal?: AbortSignal,
): Promise<DocumentoDto[]> {
  return apiGet<DocumentoDto[]>(
    `/submissions/${encodeURIComponent(submissionId)}/documents`,
    { signal },
  )
}

/**
 * Elimina un documento respaldado por su identificador único.
 */
export async function eliminarDocumento(
  documentId: string,
  signal?: AbortSignal,
): Promise<void> {
  return apiDelete<void>(`/documents/${encodeURIComponent(documentId)}`, {
    signal,
  })
}

/**
 * Obtiene una URL temporal firmada para visualizar o descargar el documento.
 */
export async function obtenerEnlaceDescarga(
  documentId: string,
  signal?: AbortSignal,
): Promise<EnlaceDescargaDto> {
  return apiGet<EnlaceDescargaDto>(
    `/documents/${encodeURIComponent(documentId)}/download`,
    { signal },
  )
}

/**
 * Abre de forma segura el documento en una nueva pestaña sin guardar la URL temporal
 * en almacenamiento persistente del navegador (localStorage / sessionStorage).
 */
export async function abrirDocumentoSeguro(documentId: string): Promise<void> {
  const { url } = await obtenerEnlaceDescarga(documentId)
  if (typeof window !== "undefined") {
    window.open(url, "_blank", "noopener,noreferrer")
  }
}
