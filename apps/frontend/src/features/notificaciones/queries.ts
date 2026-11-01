import {
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query"
import { toast } from "sonner"
import { useSesion } from "@/features/auth/sesion-context"
import { esErrorApi } from "@/lib/api-error"
import {
  listarNotificacionesAdmin,
  listarNotificacionesPostulante,
  obtenerHistorialNotificacionAdmin,
  obtenerHistorialNotificacionPostulante,
  reintentarNotificacionAdmin,
  type EntradaReintentoNotificacionDto,
} from "@/lib/clientes/notificaciones"

const PRIVADO = {
  gcTime: 0,
  staleTime: 0,
  refetchOnWindowFocus: "always" as const,
}

export function useCuentaNotificaciones() {
  const { postulante } = useSesion()
  return `${postulante?.id ?? "sin-sesion"}:${postulante?.rol ?? "postulante"}`
}

export const clavesNotificaciones = {
  cuenta: (cuenta: string) => ["notificaciones-privada", cuenta] as const,
  postulante: (cuenta: string, id: string) =>
    [...clavesNotificaciones.cuenta(cuenta), "postulante", id] as const,
  postulanteHistorial: (cuenta: string, id: string, mailId: string) =>
    [
      ...clavesNotificaciones.cuenta(cuenta),
      "postulante-historial",
      id,
      mailId,
    ] as const,
  admin: (cuenta: string, id: string) =>
    [...clavesNotificaciones.cuenta(cuenta), "admin", id] as const,
  adminHistorial: (cuenta: string, id: string, mailId: string) =>
    [
      ...clavesNotificaciones.cuenta(cuenta),
      "admin-historial",
      id,
      mailId,
    ] as const,
}

/**
 * Consulta de avisos oficiales notificados al titular (FT22).
 */
export function useNotificacionesPostulante(submissionId: string) {
  const cuenta = useCuentaNotificaciones()
  return useInfiniteQuery({
    ...PRIVADO,
    queryKey: clavesNotificaciones.postulante(cuenta, submissionId),
    queryFn: ({ pageParam, signal }) =>
      listarNotificacionesPostulante(
        submissionId,
        { cursor: pageParam },
        signal,
      ),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (pagina) =>
      pagina.hasMore ? pagina.nextCursor : undefined,
    enabled: Boolean(submissionId),
  })
}

/**
 * Consulta del historial de intentos para una notificación del postulante (FT22).
 */
export function useHistorialNotificacionPostulante(
  submissionId: string,
  mailId: string,
) {
  const cuenta = useCuentaNotificaciones()
  return useInfiniteQuery({
    ...PRIVADO,
    queryKey: clavesNotificaciones.postulanteHistorial(
      cuenta,
      submissionId,
      mailId,
    ),
    queryFn: ({ pageParam, signal }) =>
      obtenerHistorialNotificacionPostulante(
        submissionId,
        mailId,
        { cursor: pageParam },
        signal,
      ),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (pagina) =>
      pagina.hasMore ? pagina.nextCursor : undefined,
    enabled: Boolean(submissionId && mailId),
  })
}

/**
 * Consulta de notificaciones del expediente para Analista o Administrador (FT22).
 */
export function useNotificacionesAdmin(submissionId: string) {
  const cuenta = useCuentaNotificaciones()
  return useInfiniteQuery({
    ...PRIVADO,
    queryKey: clavesNotificaciones.admin(cuenta, submissionId),
    queryFn: ({ pageParam, signal }) =>
      listarNotificacionesAdmin(submissionId, { cursor: pageParam }, signal),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (pagina) =>
      pagina.hasMore ? pagina.nextCursor : undefined,
    enabled: Boolean(submissionId),
  })
}

/**
 * Historial auditado de intentos de envío para analista o administrador (FT22).
 */
export function useHistorialNotificacionAdmin(
  submissionId: string,
  mailId: string,
) {
  const cuenta = useCuentaNotificaciones()
  return useQuery({
    ...PRIVADO,
    queryKey: clavesNotificaciones.adminHistorial(cuenta, submissionId, mailId),
    queryFn: ({ signal }) =>
      obtenerHistorialNotificacionAdmin(
        submissionId,
        mailId,
        undefined,
        signal,
      ),
    enabled: Boolean(submissionId && mailId),
  })
}

/**
 * Mutación para reintentar el envío de una notificación fallida vigente (FT22).
 */
export function useReintentarNotificacionAdmin(
  submissionId: string,
  mailId: string,
) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (entrada: EntradaReintentoNotificacionDto) =>
      reintentarNotificacionAdmin(submissionId, mailId, entrada),
    onSuccess: () => {
      toast.success("Reintento de notificación encolado con éxito.")
      void queryClient.invalidateQueries({
        queryKey: ["notificaciones-privada"],
      })
    },
    onError: (error) => {
      const msg = esErrorApi(error)
        ? error.message
        : "No se pudo programar el reintento de la notificación."
      toast.error(msg)
    },
  })
}
