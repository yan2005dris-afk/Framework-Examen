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
  generarListaResultados,
  listarListasAdmin,
  listarPublicacionesListasAdmin,
  obtenerEntradasListaAdmin,
  obtenerListaMetadataAdmin,
  obtenerResultadosPublicosVigentes,
  publicarListaAdmin,
  type EntradaGenerarListaDto,
  type EntradaPublicarListaDto,
} from "@/lib/clientes/resultados"

const PRIVADO = {
  gcTime: 0,
  staleTime: 0,
  refetchOnWindowFocus: "always" as const,
}

export function useCuentaResultados() {
  const { postulante } = useSesion()
  return `${postulante?.id ?? "sin-sesion"}:${postulante?.rol ?? "postulante"}`
}

export const clavesResultados = {
  cuenta: (cuenta: string) => ["resultados-privada", cuenta] as const,
  listas: (cuenta: string, period: string, sector: string) =>
    [...clavesResultados.cuenta(cuenta), "listas", period, sector] as const,
  publicaciones: (cuenta: string, period: string, sector: string) =>
    [
      ...clavesResultados.cuenta(cuenta),
      "publicaciones",
      period,
      sector,
    ] as const,
  listaMeta: (cuenta: string, id: string) =>
    [...clavesResultados.cuenta(cuenta), "lista-meta", id] as const,
  entradas: (cuenta: string, id: string) =>
    [...clavesResultados.cuenta(cuenta), "entradas", id] as const,
  publicoVigente: (period: string, sector: string) =>
    ["resultados-publicos", period, sector] as const,
}

/**
 * Consulta de versiones de listas internas generadas para un período y sector (FT23).
 */
export function useListasAdmin(period: string, sector: string) {
  const cuenta = useCuentaResultados()
  return useInfiniteQuery({
    ...PRIVADO,
    queryKey: clavesResultados.listas(cuenta, period, sector),
    queryFn: ({ pageParam, signal }) =>
      listarListasAdmin(period, sector, { cursor: pageParam }, signal),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (pagina) =>
      pagina.hasMore ? pagina.nextCursor : undefined,
    enabled: Boolean(period && sector),
  })
}

/**
 * Consulta de versiones oficiales publicadas para un período y sector (FT23).
 */
export function usePublicacionesAdmin(period: string, sector: string) {
  const cuenta = useCuentaResultados()
  return useInfiniteQuery({
    ...PRIVADO,
    queryKey: clavesResultados.publicaciones(cuenta, period, sector),
    queryFn: ({ pageParam, signal }) =>
      listarPublicacionesListasAdmin(
        period,
        sector,
        { cursor: pageParam },
        signal,
      ),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (pagina) =>
      pagina.hasMore ? pagina.nextCursor : undefined,
    enabled: Boolean(period && sector),
  })
}

/**
 * Metadatos y reglas de una lista interna (FT23).
 */
export function useListaMetadataAdmin(listId: string) {
  const cuenta = useCuentaResultados()
  return useQuery({
    ...PRIVADO,
    queryKey: clavesResultados.listaMeta(cuenta, listId),
    queryFn: ({ signal }) => obtenerListaMetadataAdmin(listId, signal),
    enabled: Boolean(listId),
  })
}

/**
 * Entradas paginadas con puestos y calificaciones de una lista interna (FT23).
 */
export function useEntradasListaAdmin(listId: string) {
  const cuenta = useCuentaResultados()
  return useInfiniteQuery({
    ...PRIVADO,
    queryKey: clavesResultados.entradas(cuenta, listId),
    queryFn: ({ pageParam, signal }) =>
      obtenerEntradasListaAdmin(listId, { cursor: pageParam }, signal),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (pagina) =>
      pagina.hasMore ? pagina.nextCursor : undefined,
    enabled: Boolean(listId),
  })
}

/**
 * Mutación para generar una lista interna reproducible (FT23).
 */
export function useGenerarListaAdmin(period: string, sector: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (entrada: EntradaGenerarListaDto) =>
      generarListaResultados(period, sector, entrada),
    onSuccess: () => {
      toast.success("Lista interna generada exitosamente.")
      void queryClient.invalidateQueries({
        queryKey: ["resultados-privada"],
      })
    },
    onError: (error) => {
      const msg = esErrorApi(error)
        ? error.message
        : "No se pudo generar la lista de elegibles."
      toast.error(msg)
    },
  })
}

/**
 * Mutación para publicar oficialmente la lista (FT23).
 */
export function usePublicarListaAdmin(listId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (entrada: EntradaPublicarListaDto) =>
      publicarListaAdmin(listId, entrada),
    onSuccess: () => {
      toast.success("Publicación oficial registrada con éxito.")
      void queryClient.invalidateQueries({
        queryKey: ["resultados-privada"],
      })
    },
    onError: (error) => {
      const msg = esErrorApi(error)
        ? error.message
        : "No se pudo registrar la publicación oficial."
      toast.error(msg)
    },
  })
}

/**
 * Consulta pública sin sesión de la lista vigente de elegibles (FT23).
 */
export function useResultadosPublicosVigentes(period: string, sector: string) {
  return useQuery({
    queryKey: clavesResultados.publicoVigente(period, sector),
    queryFn: ({ signal }) =>
      obtenerResultadosPublicosVigentes(period, sector, undefined, signal),
    enabled: Boolean(period && sector),
    staleTime: 60_000,
  })
}
