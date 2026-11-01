import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
import { esErrorApi } from "@/lib/api-error"
import {
  listarPoliticasConvocatoria,
  listarRevisionesRetencion,
  obtenerPoliticaPorId,
  obtenerPoliticaVigenteConvocatoria,
  publicarPoliticaPrivacidad,
  restringirPoliticaPrivacidad,
  type PublicarPoliticaDto,
  type RestringirPoliticaDto,
} from "@/lib/clientes/privacidad"

export const clavesPrivacidad = {
  todas: ["privacidad"] as const,
  vigente: (convocatoriaId: string) =>
    [...clavesPrivacidad.todas, "vigente", convocatoriaId] as const,
  porId: (politicaId: string) =>
    [...clavesPrivacidad.todas, "detalle", politicaId] as const,
  historial: (convocatoriaId: string) =>
    [...clavesPrivacidad.todas, "historial", convocatoriaId] as const,
  retencion: (convocatoriaId: string) =>
    [...clavesPrivacidad.todas, "retencion", convocatoriaId] as const,
}

export function usePoliticaVigente(convocatoriaId?: string) {
  return useQuery({
    queryKey: clavesPrivacidad.vigente(convocatoriaId ?? ""),
    queryFn: () => obtenerPoliticaVigenteConvocatoria(convocatoriaId!),
    enabled: Boolean(convocatoriaId),
    staleTime: 60_000,
  })
}

export function usePoliticaPorId(politicaId?: string) {
  return useQuery({
    queryKey: clavesPrivacidad.porId(politicaId ?? ""),
    queryFn: () => obtenerPoliticaPorId(politicaId!),
    enabled: Boolean(politicaId),
    staleTime: Infinity,
  })
}

export function useHistorialPoliticas(convocatoriaId?: string) {
  return useQuery({
    queryKey: clavesPrivacidad.historial(convocatoriaId ?? ""),
    queryFn: () => listarPoliticasConvocatoria(convocatoriaId!),
    enabled: Boolean(convocatoriaId),
  })
}

export function useRevisionesRetencion(convocatoriaId?: string) {
  return useQuery({
    queryKey: clavesPrivacidad.retencion(convocatoriaId ?? ""),
    queryFn: () => listarRevisionesRetencion(convocatoriaId!),
    enabled: Boolean(convocatoriaId),
  })
}

export function usePublicarPolitica(convocatoriaId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (datos: PublicarPoliticaDto) =>
      publicarPoliticaPrivacidad(convocatoriaId, datos),
    onSuccess: (politica) => {
      toast.success(
        `Política de privacidad versión ${politica.version} publicada exitosamente.`,
      )
      void queryClient.invalidateQueries({
        queryKey: clavesPrivacidad.vigente(convocatoriaId),
      })
      void queryClient.invalidateQueries({
        queryKey: clavesPrivacidad.historial(convocatoriaId),
      })
    },
    onError: (error) => {
      if (esErrorApi(error)) {
        toast.error(`Error al publicar política: ${error.mensaje}`)
      } else {
        toast.error("Error inesperado al publicar política de privacidad.")
      }
    },
  })
}

export function useRestringirPolitica(convocatoriaId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (datos: RestringirPoliticaDto) =>
      restringirPoliticaPrivacidad(convocatoriaId, datos),
    onSuccess: (_, variables) => {
      toast.success(
        variables.restringido
          ? "Restricción cautelar de privacidad aplicada."
          : "Restricción de privacidad levantada.",
      )
      void queryClient.invalidateQueries({
        queryKey: clavesPrivacidad.vigente(convocatoriaId),
      })
      void queryClient.invalidateQueries({
        queryKey: clavesPrivacidad.historial(convocatoriaId),
      })
    },
    onError: (error) => {
      if (esErrorApi(error)) {
        toast.error(`Error al modificar restricción: ${error.mensaje}`)
      } else {
        toast.error("Error inesperado al modificar restricción de privacidad.")
      }
    },
  })
}
