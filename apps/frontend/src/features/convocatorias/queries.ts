import {
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query"
import { useSesion } from "@/features/auth/sesion-context"
import { esErrorApi } from "@/lib/api-error"
import {
  crearConvocatoria,
  guardarConfiguracionConvocatoria,
  listarConvocatoriasAdmin,
  listarConvocatoriasPublicas,
  obtenerConvocatoriaAdmin,
  obtenerConvocatoriaPublica,
  obtenerDetallePublico,
} from "@/lib/clientes/convocatorias"
import type {
  ConfiguracionConvocatoria,
  CrearConvocatoriaPayload,
} from "@/types/convocatoria"

export const clavesConvocatorias = {
  publica: ["convocatoria-publica"] as const,
  detallePublico: (id: string) => ["convocatoria-publica", id] as const,
  publicas: ["convocatorias-publicas"] as const,
  privada: (cuenta: string) => ["gestion-convocatorias", cuenta] as const,
  adminLista: (cuenta: string) =>
    ["gestion-convocatorias", cuenta, "lista"] as const,
  adminDetalle: (cuenta: string, id: string) =>
    ["gestion-convocatorias", cuenta, "detalle", id] as const,
}

const REFRESCO = {
  staleTime: 0,
  refetchInterval: 30_000,
  refetchIntervalInBackground: false,
  refetchOnWindowFocus: "always" as const,
  refetchOnReconnect: "always" as const,
}
const PRIVADO = { ...REFRESCO, gcTime: 0 }

/** Consulta la convocatoria exacta si se proporciona ID; no sustituye una retirada por otra. */
export function useConvocatoriaPublica(id?: string | null, habilitada = true) {
  return useQuery({
    ...REFRESCO,
    enabled: habilitada,
    queryKey: id
      ? clavesConvocatorias.detallePublico(id)
      : clavesConvocatorias.publica,
    queryFn: ({ signal }) =>
      id
        ? obtenerDetallePublico(id, signal)
        : obtenerConvocatoriaPublica(signal),
  })
}

/** Mantiene el histórico público paginado sin convertir el último UUID en una convocatoria activa. */
export function useConvocatoriasPublicas() {
  return useInfiniteQuery({
    ...REFRESCO,
    queryKey: clavesConvocatorias.publicas,
    initialPageParam: undefined as string | undefined,
    queryFn: ({ pageParam, signal }) =>
      listarConvocatoriasPublicas(pageParam, signal),
    getNextPageParam: (pagina) =>
      pagina.hasMore ? pagina.nextCursor : undefined,
  })
}

/** Caché administrativa aislada por cuenta y rol; se descarta al salir de la pantalla. */
export function useConvocatoriasAdmin() {
  const { postulante } = useSesion()
  const cuenta = `${postulante?.id ?? ""}:${postulante?.rol ?? ""}`
  return useInfiniteQuery({
    ...PRIVADO,
    queryKey: clavesConvocatorias.adminLista(cuenta),
    initialPageParam: undefined as string | undefined,
    enabled: postulante?.rol === "administrador",
    queryFn: ({ pageParam, signal }) =>
      listarConvocatoriasAdmin(pageParam, signal),
    getNextPageParam: (pagina) =>
      pagina.hasMore ? pagina.nextCursor : undefined,
  })
}

export function useConvocatoriaAdminDetalle(id: string | null | undefined) {
  const { postulante } = useSesion()
  const cuenta = `${postulante?.id ?? ""}:${postulante?.rol ?? ""}`
  return useQuery({
    ...PRIVADO,
    queryKey: clavesConvocatorias.adminDetalle(cuenta, id ?? ""),
    queryFn: ({ signal }) => obtenerConvocatoriaAdmin(id!, signal),
    enabled: Boolean(id) && postulante?.rol === "administrador",
  })
}

export function useCrearConvocatoria() {
  const cliente = useQueryClient()
  return useMutation({
    mutationFn: (payload: CrearConvocatoriaPayload) =>
      crearConvocatoria(payload),
    onSuccess: async () => {
      await cliente.invalidateQueries({ queryKey: ["gestion-convocatorias"] })
    },
  })
}

/** No aplica cambios optimistas. Conflicto exige volver a leer y confirmar, nunca reintento ciego. */
export function useGuardarConfiguracionConvocatoria() {
  const cliente = useQueryClient()
  const { postulante } = useSesion()
  const cuenta = `${postulante?.id ?? ""}:${postulante?.rol ?? ""}`
  return useMutation({
    mutationFn: ({
      id,
      config,
    }: {
      id: string
      config: ConfiguracionConvocatoria
    }) => guardarConfiguracionConvocatoria(id, config),
    onSuccess: async (detalle) => {
      cliente.setQueryData(
        clavesConvocatorias.adminDetalle(cuenta, detalle.id),
        detalle,
      )
      await Promise.all([
        cliente.invalidateQueries({ queryKey: ["gestion-convocatorias"] }),
        cliente.invalidateQueries({ queryKey: clavesConvocatorias.publica }),
        cliente.invalidateQueries({ queryKey: clavesConvocatorias.publicas }),
      ])
    },
    onError: async (error, { id }) => {
      if (esErrorApi(error) && error.esConflicto())
        await cliente.invalidateQueries({
          queryKey: clavesConvocatorias.adminDetalle(cuenta, id),
        })
    },
  })
}
