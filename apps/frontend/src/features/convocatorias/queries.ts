import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  actualizarEstadoConvocatoria,
  actualizarFase,
  alternarHabilitacionFormulario,
  crearConvocatoria,
  listarConvocatoriasAdmin,
  obtenerConvocatoriaPublica,
} from "@/lib/clientes/convocatorias"
import type {
  ActualizarConvocatoriaPayload,
  ActualizarFasePayload,
  CrearConvocatoriaPayload,
} from "@/types/convocatoria"

export const clavesConvocatorias = {
  publica: ["convocatoria-publica"] as const,
  adminLista: ["admin-convocatorias"] as const,
  adminDetalle: (id: string) => ["admin-convocatoria", id] as const,
}

/**
 * Consulta la convocatoria pública oficial con refresco periódico (FT14).
 * Si el concurso se cierra o se deshabilita mientras el usuario navega,
 * la pantalla se actualiza automáticamente.
 */
export function useConvocatoriaPublica() {
  return useQuery({
    queryKey: clavesConvocatorias.publica,
    queryFn: ({ signal }) => obtenerConvocatoriaPublica(signal),
    refetchInterval: 30_000,
    refetchIntervalInBackground: false,
    refetchOnWindowFocus: "always",
    refetchOnReconnect: "always",
  })
}

/**
 * Consulta el listado de convocatorias para el panel de administración (FT15).
 */
export function useConvocatoriasAdmin(cursor?: string) {
  return useQuery({
    queryKey: [...clavesConvocatorias.adminLista, cursor],
    queryFn: ({ signal }) => listarConvocatoriasAdmin(cursor, signal),
  })
}

/**
 * Mutación para registrar una nueva convocatoria.
 */
export function useCrearConvocatoria() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: CrearConvocatoriaPayload) =>
      crearConvocatoria(payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: clavesConvocatorias.adminLista,
      })
      void queryClient.invalidateQueries({
        queryKey: clavesConvocatorias.publica,
      })
    },
  })
}

/**
 * Mutación para modificar estado, fechas o cierre de convocatoria.
 */
export function useActualizarEstadoConvocatoria() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      id,
      payload,
    }: {
      id: string
      payload: ActualizarConvocatoriaPayload
    }) => actualizarEstadoConvocatoria(id, payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: clavesConvocatorias.adminLista,
      })
      void queryClient.invalidateQueries({
        queryKey: clavesConvocatorias.publica,
      })
    },
  })
}

/**
 * Mutación para habilitar o deshabilitar la recepción de formularios EF-12.
 */
export function useAlternarHabilitacionFormulario() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      convocatoriaId,
      habilitada,
      motivo,
    }: {
      convocatoriaId: string
      habilitada: boolean
      motivo?: string
    }) => alternarHabilitacionFormulario(convocatoriaId, habilitada, motivo),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: clavesConvocatorias.adminLista,
      })
      void queryClient.invalidateQueries({
        queryKey: clavesConvocatorias.publica,
      })
    },
  })
}

/**
 * Mutación para actualizar fechas o estado de una fase del concurso.
 */
export function useActualizarFase() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      convocatoriaId,
      faseId,
      payload,
    }: {
      convocatoriaId: string
      faseId: string
      payload: ActualizarFasePayload
    }) => actualizarFase(convocatoriaId, faseId, payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: clavesConvocatorias.adminLista,
      })
      void queryClient.invalidateQueries({
        queryKey: clavesConvocatorias.publica,
      })
    },
  })
}
