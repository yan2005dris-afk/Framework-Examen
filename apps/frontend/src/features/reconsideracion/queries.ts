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
  atenderReconsideracion,
  listarReconsideracionesAdmin,
  listarReconsideracionesPostulante,
  obtenerReconsideracionAdmin,
  obtenerReconsideracionPostulante,
  presentarReconsideracion,
  resolverReconsideracion,
  type EntradaAtenderReconsideracionDto,
  type EntradaPresentarReconsideracionDto,
  type EntradaResolverReconsideracionDto,
} from "@/lib/clientes/reconsideracion"

const PRIVADO = {
  gcTime: 0,
  staleTime: 0,
  refetchOnWindowFocus: "always" as const,
}

export function useCuentaReconsideracion() {
  const { postulante } = useSesion()
  return `${postulante?.id ?? "sin-sesion"}:${postulante?.rol ?? "postulante"}`
}

export const clavesReconsideracion = {
  cuenta: (cuenta: string) => ["reconsideracion-privada", cuenta] as const,
  postulanteLista: (cuenta: string, id: string) =>
    [...clavesReconsideracion.cuenta(cuenta), "postulante-lista", id] as const,
  postulanteDetalle: (cuenta: string, id: string, appealId: string) =>
    [
      ...clavesReconsideracion.cuenta(cuenta),
      "postulante-detalle",
      id,
      appealId,
    ] as const,
  adminLista: (cuenta: string, id: string) =>
    [...clavesReconsideracion.cuenta(cuenta), "admin-lista", id] as const,
  adminDetalle: (cuenta: string, id: string, appealId: string) =>
    [
      ...clavesReconsideracion.cuenta(cuenta),
      "admin-detalle",
      id,
      appealId,
    ] as const,
}

/**
 * Consulta de reconsideraciones presentadas por el postulante (FT21).
 */
export function useReconsideracionesPostulante(submissionId: string) {
  const cuenta = useCuentaReconsideracion()
  return useInfiniteQuery({
    ...PRIVADO,
    queryKey: clavesReconsideracion.postulanteLista(cuenta, submissionId),
    queryFn: ({ pageParam, signal }) =>
      listarReconsideracionesPostulante(
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
 * Consulta del detalle y actuaciones de una reconsideración del postulante (FT21).
 */
export function useReconsideracionPostulanteDetalle(
  submissionId: string,
  appealId: string,
) {
  const cuenta = useCuentaReconsideracion()
  return useQuery({
    ...PRIVADO,
    queryKey: clavesReconsideracion.postulanteDetalle(
      cuenta,
      submissionId,
      appealId,
    ),
    queryFn: ({ signal }) =>
      obtenerReconsideracionPostulante(submissionId, appealId, signal),
    enabled: Boolean(submissionId && appealId),
  })
}

/**
 * Mutación para que el postulante presente una solicitud de reconsideración (FT21).
 */
export function usePresentarReconsideracion(submissionId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (entrada: EntradaPresentarReconsideracionDto) =>
      presentarReconsideracion(submissionId, entrada),
    onSuccess: () => {
      toast.success("Solicitud de reconsideración presentada correctamente.")
      void queryClient.invalidateQueries({
        queryKey: ["reconsideracion-privada"],
      })
      void queryClient.invalidateQueries({
        queryKey: ["evaluacion-privada"],
      })
    },
    onError: (error) => {
      const msg = esErrorApi(error)
        ? error.message
        : "No se pudo presentar la solicitud de reconsideración."
      toast.error(msg)
    },
  })
}

/**
 * Consulta administrativa de reconsideraciones de un expediente (FT21).
 */
export function useReconsideracionesAdmin(submissionId: string) {
  const cuenta = useCuentaReconsideracion()
  return useInfiniteQuery({
    ...PRIVADO,
    queryKey: clavesReconsideracion.adminLista(cuenta, submissionId),
    queryFn: ({ pageParam, signal }) =>
      listarReconsideracionesAdmin(submissionId, { cursor: pageParam }, signal),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (pagina) =>
      pagina.hasMore ? pagina.nextCursor : undefined,
    enabled: Boolean(submissionId),
  })
}

/**
 * Detalle auditado de una reconsideración para analista o administrador (FT21).
 */
export function useReconsideracionAdminDetalle(
  submissionId: string,
  appealId: string,
) {
  const cuenta = useCuentaReconsideracion()
  return useQuery({
    ...PRIVADO,
    queryKey: clavesReconsideracion.adminDetalle(
      cuenta,
      submissionId,
      appealId,
    ),
    queryFn: ({ signal }) =>
      obtenerReconsideracionAdmin(submissionId, appealId, signal),
    enabled: Boolean(submissionId && appealId),
  })
}

/**
 * Mutación para que el Analista atienda una reconsideración (FT21).
 */
export function useAtenderReconsideracion(
  submissionId: string,
  appealId: string,
) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (entrada: EntradaAtenderReconsideracionDto) =>
      atenderReconsideracion(submissionId, appealId, entrada),
    onSuccess: () => {
      toast.success("Atención de reconsideración registrada con éxito.")
      void queryClient.invalidateQueries({
        queryKey: ["reconsideracion-privada"],
      })
    },
    onError: (error) => {
      const msg = esErrorApi(error)
        ? error.message
        : "No se pudo registrar la atención de la reconsideración."
      toast.error(msg)
    },
  })
}

/**
 * Mutación para que el Administrador resuelva la reconsideración (FT21).
 */
export function useResolverReconsideracion(
  submissionId: string,
  appealId: string,
) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (entrada: EntradaResolverReconsideracionDto) =>
      resolverReconsideracion(submissionId, appealId, entrada),
    onSuccess: () => {
      toast.success("Resolución de reconsideración registrada oficialmente.")
      void queryClient.invalidateQueries({
        queryKey: ["reconsideracion-privada"],
      })
      void queryClient.invalidateQueries({
        queryKey: ["evaluacion-privada"],
      })
    },
    onError: (error) => {
      const msg = esErrorApi(error)
        ? error.message
        : "No se pudo registrar la resolución de la reconsideración."
      toast.error(msg)
    },
  })
}
