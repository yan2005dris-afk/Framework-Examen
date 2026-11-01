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
  consolidarExpediente,
  obtenerEvaluacionPostulante,
  obtenerExpedienteGradesAdmin,
  obtenerHistorialEvaluacionPostulante,
  obtenerHistorialGradesAdmin,
  publicarPsicotecnica,
  registrarPsicotecnica,
  type EntradaConsolidacionDto,
  type EntradaPsicotecnicaDto,
  type EntradaPublicarPsicotecnicaDto,
} from "@/lib/clientes/evaluacion"

const PRIVADO = {
  gcTime: 0,
  staleTime: 0,
  refetchOnWindowFocus: "always" as const,
}

export function useCuentaEvaluacion() {
  const { postulante } = useSesion()
  return `${postulante?.id ?? "sin-sesion"}:${postulante?.rol ?? "postulante"}`
}

export const clavesEvaluacion = {
  cuenta: (cuenta: string) => ["evaluacion-privada", cuenta] as const,
  postulante: (cuenta: string, id: string) =>
    [...clavesEvaluacion.cuenta(cuenta), "postulante", id] as const,
  postulanteHistorial: (cuenta: string, id: string) =>
    [...clavesEvaluacion.cuenta(cuenta), "postulante-historial", id] as const,
  admin: (cuenta: string, id: string) =>
    [...clavesEvaluacion.cuenta(cuenta), "admin", id] as const,
  adminHistorial: (cuenta: string, id: string) =>
    [...clavesEvaluacion.cuenta(cuenta), "admin-historial", id] as const,
}

/**
 * Consulta del postulante restringida únicamente a sus resultados publicados (FT20).
 */
export function useEvaluacionPostulante(submissionId: string) {
  const cuenta = useCuentaEvaluacion()
  return useQuery({
    ...PRIVADO,
    queryKey: clavesEvaluacion.postulante(cuenta, submissionId),
    queryFn: ({ signal }) => obtenerEvaluacionPostulante(submissionId, signal),
    enabled: Boolean(submissionId),
  })
}

/**
 * Historial paginado de resultados y versiones para el titular (FT20).
 */
export function useHistorialEvaluacionPostulante(submissionId: string) {
  const cuenta = useCuentaEvaluacion()
  return useInfiniteQuery({
    ...PRIVADO,
    queryKey: clavesEvaluacion.postulanteHistorial(cuenta, submissionId),
    queryFn: ({ pageParam, signal }) =>
      obtenerHistorialEvaluacionPostulante(
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
 * Expediente de evaluación y oposición para analista o administrador (FT20).
 */
export function useExpedienteGradesAdmin(submissionId: string) {
  const cuenta = useCuentaEvaluacion()
  return useQuery({
    ...PRIVADO,
    queryKey: clavesEvaluacion.admin(cuenta, submissionId),
    queryFn: ({ signal }) => obtenerExpedienteGradesAdmin(submissionId, signal),
    enabled: Boolean(submissionId),
  })
}

/**
 * Historial auditado de calificaciones y resoluciones para analista o administrador (FT20).
 */
export function useHistorialGradesAdmin(submissionId: string) {
  const cuenta = useCuentaEvaluacion()
  return useInfiniteQuery({
    ...PRIVADO,
    queryKey: clavesEvaluacion.adminHistorial(cuenta, submissionId),
    queryFn: ({ pageParam, signal }) =>
      obtenerHistorialGradesAdmin(submissionId, { cursor: pageParam }, signal),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (pagina) =>
      pagina.hasMore ? pagina.nextCursor : undefined,
    enabled: Boolean(submissionId),
    refetchOnWindowFocus: false,
  })
}

/**
 * Mutación para registrar un informe psicotécnico provisional con respaldo PDF (FT20).
 */
export function useRegistrarPsicotecnica(submissionId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (entrada: EntradaPsicotecnicaDto) =>
      registrarPsicotecnica(submissionId, entrada),
    onSuccess: () => {
      toast.success("Informe psicotécnico registrado con éxito.")
      void queryClient.invalidateQueries({
        queryKey: ["evaluacion-privada"],
      })
    },
    onError: (error) => {
      const msg = esErrorApi(error)
        ? error.message
        : "No se pudo registrar la evaluación psicotécnica."
      toast.error(msg)
    },
  })
}

/**
 * Mutación para publicar oficialmente la nota de psicotécnica (FT20).
 */
export function usePublicarPsicotecnica(submissionId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (entrada: EntradaPublicarPsicotecnicaDto) =>
      publicarPsicotecnica(submissionId, entrada),
    onSuccess: () => {
      toast.success("Resultado psicotécnico publicado oficialmente.")
      void queryClient.invalidateQueries({
        queryKey: ["evaluacion-privada"],
      })
    },
    onError: (error) => {
      const msg = esErrorApi(error)
        ? error.message
        : "No se pudo publicar el resultado psicotécnico."
      toast.error(msg)
    },
  })
}

/**
 * Mutación para consolidar fuentes definitivas de méritos, teórica y psicotécnica (FT20).
 */
export function useConsolidarExpediente(submissionId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (entrada: EntradaConsolidacionDto) =>
      consolidarExpediente(submissionId, entrada),
    onSuccess: () => {
      toast.success("Expediente consolidado oficialmente.")
      void queryClient.invalidateQueries({
        queryKey: ["evaluacion-privada"],
      })
    },
    onError: (error) => {
      const msg = esErrorApi(error)
        ? error.message
        : "No se pudo consolidar el expediente de calificaciones."
      toast.error(msg)
    },
  })
}
