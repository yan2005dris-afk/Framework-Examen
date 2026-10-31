import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  crearConvocatoria,
  guardarConfiguracionConvocatoria,
  listarConvocatoriasAdmin,
  obtenerConvocatoriaAdmin,
  obtenerConvocatoriaPublica,
} from "@/lib/clientes/convocatorias"
import type {
  ConfiguracionConvocatoria,
  CrearConvocatoriaPayload,
} from "@/types/convocatoria"

export const clavesConvocatorias = {
  publica: ["convocatoria-publica"] as const,
  adminLista: ["admin-convocatorias"] as const,
  adminDetalle: (id: string) => ["admin-convocatoria", id] as const,
}

/**
 * Consulta la convocatoria pública oficial con refresco periódico de 30 s (FT14).
 * Devuelve null si no hay convocatorias aprobadas y publicadas.
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
 * Consulta el listado administrativo de convocatorias (FT15).
 */
export function useConvocatoriasAdmin(cursor?: string) {
  return useQuery({
    queryKey: [...clavesConvocatorias.adminLista, cursor],
    queryFn: ({ signal }) => listarConvocatoriasAdmin(cursor, signal),
  })
}

/**
 * Consulta la configuración y versión de una convocatoria administrativa (T18).
 */
export function useConvocatoriaAdminDetalle(id: string | null | undefined) {
  return useQuery({
    queryKey: id
      ? clavesConvocatorias.adminDetalle(id)
      : ["admin-convocatoria-vacio"],
    queryFn: ({ signal }) =>
      id
        ? obtenerConvocatoriaAdmin(id, signal)
        : Promise.reject(new Error("ID requerido")),
    enabled: Boolean(id),
  })
}

/**
 * Mutación para registrar una nueva convocatoria inicial en estado deshabilitado.
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
 * Mutación para actualizar la configuración completa con control optimista de versión.
 */
export function useGuardarConfiguracionConvocatoria() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      id,
      config,
    }: {
      id: string
      config: ConfiguracionConvocatoria
    }) => guardarConfiguracionConvocatoria(id, config),
    onSuccess: (_, variables) => {
      void queryClient.invalidateQueries({
        queryKey: clavesConvocatorias.adminLista,
      })
      void queryClient.invalidateQueries({
        queryKey: clavesConvocatorias.adminDetalle(variables.id),
      })
      void queryClient.invalidateQueries({
        queryKey: clavesConvocatorias.publica,
      })
    },
  })
}
