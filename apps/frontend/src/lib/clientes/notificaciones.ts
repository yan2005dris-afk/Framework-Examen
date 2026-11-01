import { apiGet, apiPost } from "../api"
import type { PaginaCursor } from "@/types/api"

export type EstadoNotificacion =
  "pending" | "processing" | "smtp_accepted" | "failed"

export interface NotificacionEtapaDto {
  id: string
  accion: string
  plantilla: string
  estado: EstadoNotificacion
  intentosCiclo: number
  intentosTotales: number
  reenvios: number
  codigoError?: string
  creadaEn: string
  actualizadaEn: string
}

export interface HistorialIntentoNotificacionDto {
  id: string
  estado: string
  codigoError?: string
  intento: number
  ciclo: number
  fecha: string
}

export interface EntradaReintentoNotificacionDto {
  reenviosAnteriores: number
  motivo: string
}

/**
 * Consulta de avisos de etapa y confirmaciones emitidas para el postulante titular (FT22).
 */
export async function listarNotificacionesPostulante(
  submissionId: string,
  params?: { limit?: number; cursor?: string },
  signal?: AbortSignal,
): Promise<PaginaCursor<NotificacionEtapaDto>> {
  return apiGet<PaginaCursor<NotificacionEtapaDto>>(
    `/notifications/submissions/${encodeURIComponent(submissionId)}`,
    { params, signal },
  )
}

/**
 * Consulta del historial de intentos y eventos de entrega para una notificación del titular (FT22).
 */
export async function obtenerHistorialNotificacionPostulante(
  submissionId: string,
  mailId: string,
  params?: { limit?: number; cursor?: string },
  signal?: AbortSignal,
): Promise<PaginaCursor<HistorialIntentoNotificacionDto>> {
  return apiGet<PaginaCursor<HistorialIntentoNotificacionDto>>(
    `/notifications/submissions/${encodeURIComponent(submissionId)}/${encodeURIComponent(mailId)}/history`,
    { params, signal },
  )
}

/**
 * Consulta administrativa auditada de notificaciones para analista o administrador (FT22).
 */
export async function listarNotificacionesAdmin(
  submissionId: string,
  params?: { limit?: number; cursor?: string },
  signal?: AbortSignal,
): Promise<PaginaCursor<NotificacionEtapaDto>> {
  return apiGet<PaginaCursor<NotificacionEtapaDto>>(
    `/admin/notifications/submissions/${encodeURIComponent(submissionId)}`,
    { params, signal },
  )
}

/**
 * Historial auditado de intentos de envío para analista o administrador (FT22).
 */
export async function obtenerHistorialNotificacionAdmin(
  submissionId: string,
  mailId: string,
  params?: { limit?: number; cursor?: string },
  signal?: AbortSignal,
): Promise<PaginaCursor<HistorialIntentoNotificacionDto>> {
  return apiGet<PaginaCursor<HistorialIntentoNotificacionDto>>(
    `/admin/notifications/submissions/${encodeURIComponent(submissionId)}/${encodeURIComponent(mailId)}/history`,
    { params, signal },
  )
}

/**
 * Solicita el reintento manual de una notificación fallida vigente (FT22).
 * Exclusivo para el rol Administrador de la Comisión.
 */
export async function reintentarNotificacionAdmin(
  submissionId: string,
  mailId: string,
  entrada: EntradaReintentoNotificacionDto,
  signal?: AbortSignal,
): Promise<NotificacionEtapaDto> {
  return apiPost<NotificacionEtapaDto>(
    `/admin/notifications/submissions/${encodeURIComponent(submissionId)}/${encodeURIComponent(mailId)}/retry`,
    entrada,
    { signal },
  )
}
