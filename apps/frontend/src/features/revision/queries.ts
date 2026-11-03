import {
  infiniteQueryOptions,
  queryOptions,
  useInfiniteQuery,
  useQuery,
} from "@tanstack/react-query"
import { useSesion } from "@/features/auth/sesion-context"
import {
  listarConvocatoriasRevision,
  listarRevision,
  obtenerConteosRevision,
  obtenerExpedienteRevision,
  listarDocumentosRevision,
  listarEventosRevision,
  type FiltrosRevision,
} from "@/lib/clientes/revision"
import {
  obtenerExpedienteMeritosAdmin,
  obtenerResultadoMeritosPostulante,
  crearRevisionMeritos,
  calcularMeritos,
  publicarMeritos,
  type RevisionInput,
} from "@/lib/clientes/meritos"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
import { esErrorApi } from "@/lib/api-error"

const PRIVADO = {
  gcTime: 0,
  staleTime: 0,
  refetchOnWindowFocus: "always" as const,
}

export const clavesRevision = {
  cuenta: (cuenta: string) => ["revision-privada", cuenta] as const,
  nomina: (cuenta: string, filtros: FiltrosRevision) =>
    [...clavesRevision.cuenta(cuenta), "nomina", filtros] as const,
}

/** Cada selección inicia una cadena independiente de cursores opacos. */
export function opcionesNomina(cuenta: string, filtros: FiltrosRevision) {
  return infiniteQueryOptions({
    ...PRIVADO,
    queryKey: clavesRevision.nomina(cuenta, filtros),
    queryFn: ({ pageParam, signal }) =>
      listarRevision(filtros, pageParam, signal),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (pagina) =>
      pagina.hasMore ? pagina.nextCursor : undefined,
  })
}

export function opcionesConteos(cuenta: string, filtros: FiltrosRevision) {
  return queryOptions({
    ...PRIVADO,
    queryKey: [...clavesRevision.cuenta(cuenta), "conteos", filtros],
    queryFn: ({ signal }) => obtenerConteosRevision(filtros, signal),
    refetchInterval: 30_000,
    refetchIntervalInBackground: false,
    refetchOnReconnect: "always" as const,
  })
}

/** La identidad y el rol aíslan la caché, que se libera al abandonar la vista. */
export function useCuentaRevision() {
  const { postulante } = useSesion()
  return `${postulante?.id ?? "sin-sesion"}:${postulante?.rol ?? "postulante"}`
}

export function useConvocatoriasRevision() {
  const cuenta = useCuentaRevision()
  return useInfiniteQuery({
    ...PRIVADO,
    queryKey: [...clavesRevision.cuenta(cuenta), "convocatorias"],
    queryFn: ({ pageParam, signal }) =>
      listarConvocatoriasRevision(pageParam, signal),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (pagina) =>
      pagina.hasMore ? pagina.nextCursor : undefined,
  })
}

export function useNominaRevision(filtros: FiltrosRevision) {
  return useInfiniteQuery(opcionesNomina(useCuentaRevision(), filtros))
}

export function useConteosRevision(filtros: FiltrosRevision) {
  return useQuery(opcionesConteos(useCuentaRevision(), filtros))
}

export function useExpedienteRevision(id: string) {
  const cuenta = useCuentaRevision()
  return useQuery({
    ...PRIVADO,
    queryKey: [...clavesRevision.cuenta(cuenta), "expediente", id],
    queryFn: ({ signal }) => obtenerExpedienteRevision(id, signal),
  })
}

export function useDocumentosRevision(id: string) {
  const cuenta = useCuentaRevision()
  return useQuery({
    ...PRIVADO,
    queryKey: [...clavesRevision.cuenta(cuenta), "documentos", id],
    queryFn: ({ signal }) => listarDocumentosRevision(id, signal),
  })
}

export function useEventosRevision(id: string) {
  const cuenta = useCuentaRevision()
  return useInfiniteQuery({
    ...PRIVADO,
    queryKey: [...clavesRevision.cuenta(cuenta), "eventos", id],
    queryFn: ({ pageParam, signal }) =>
      listarEventosRevision(id, pageParam, signal),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (pagina) =>
      pagina.hasMore ? pagina.nextCursor : undefined,
    // Cada lectura queda auditada: el historial se actualiza solo por petición expresa.
    refetchOnWindowFocus: false,
  })
}

export function useExpedienteMeritosAdmin(id: string) {
  const cuenta = useCuentaRevision()
  return useQuery({
    ...PRIVADO,
    queryKey: [...clavesRevision.cuenta(cuenta), "meritos-admin", id],
    queryFn: ({ signal }) => obtenerExpedienteMeritosAdmin(id, signal),
  })
}

export function useResultadoMeritosPostulante(id: string) {
  const cuenta = useCuentaRevision()
  return useQuery({
    ...PRIVADO,
    queryKey: [...clavesRevision.cuenta(cuenta), "meritos-postulante", id],
    queryFn: ({ signal }) => obtenerResultadoMeritosPostulante(id, signal),
  })
}

export function useCrearRevisionMeritos(id: string) {
  const queryClient = useQueryClient()
  const cuenta = useCuentaRevision()
  return useMutation({
    mutationFn: (input: RevisionInput) => crearRevisionMeritos(id, input),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: [...clavesRevision.cuenta(cuenta), "meritos-admin", id],
      })
      toast.success("Revisión técnica de méritos guardada exitosamente.")
    },
    onError: (err) => {
      if (esErrorApi(err)) {
        toast.error(err.message || "Error al guardar la revisión de méritos.")
      } else {
        toast.error("No se pudo conectar con el servidor.")
      }
    },
  })
}

export function useCalcularMeritos(id: string) {
  const queryClient = useQueryClient()
  const cuenta = useCuentaRevision()
  return useMutation({
    mutationFn: (input: {
      revisionId: string
      versionAnterior: number
      motivo: string
    }) => calcularMeritos(id, input),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: [...clavesRevision.cuenta(cuenta), "meritos-admin", id],
      })
      toast.success("Cálculo oficial de méritos generado por el motor.")
    },
    onError: (err) => {
      if (esErrorApi(err)) {
        toast.error(err.message || "Error al calcular méritos.")
      } else {
        toast.error("No se pudo conectar con el servidor.")
      }
    },
  })
}

export function usePublicarMeritos(id: string) {
  const queryClient = useQueryClient()
  const cuenta = useCuentaRevision()
  return useMutation({
    mutationFn: (input: {
      calculoId: string
      versionAnterior: number
      decision: "avanza" | "excluido"
      motivo: string
      sustituyeId?: string
    }) => publicarMeritos(id, input),
    onSuccess: (data) => {
      queryClient.invalidateQueries({
        queryKey: [...clavesRevision.cuenta(cuenta), "meritos-admin", id],
      })
      toast.success(
        `Resolución oficial publicada: «${data.decision.toUpperCase()}».`,
      )
    },
    onError: (err) => {
      if (esErrorApi(err)) {
        toast.error(
          err.message || "Error al publicar la resolución de méritos.",
        )
      } else {
        toast.error("No se pudo conectar con el servidor.")
      }
    },
  })
}
