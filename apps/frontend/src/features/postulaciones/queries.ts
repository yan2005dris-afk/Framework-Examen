import {
  useMutation,
  useQuery,
  useQueryClient,
  type UseQueryOptions,
} from "@tanstack/react-query"
import { toast } from "sonner"

import { esErrorApi } from "@/lib/api-error"
import {
  obtenerCantones,
  obtenerPaises,
  obtenerProvincias,
} from "@/lib/clientes/catalogos"
import {
  obtenerDefinicionFormulario,
  type DefinicionFormularioEF12,
} from "@/lib/clientes/formulario"
import {
  actualizarPostulacion,
  crearPostulacion,
  eliminarPostulacion,
  guardarSeccion,
  listarPostulaciones,
  obtenerPostulacion,
  sellarPostulacion,
  type CrearPostulacionDto,
  type ListarPostulacionesParams,
  type PostulacionDto,
} from "@/lib/clientes/postulaciones"
import type { Pagina } from "@/types/api"

/**
 * Factory de claves de consulta para centralizar la invalidación de caché.
 */
export const postulacionKeys = {
  todas: ["postulaciones"] as const,
  listas: () => [...postulacionKeys.todas, "lista"] as const,
  lista: (params?: ListarPostulacionesParams) =>
    [...postulacionKeys.listas(), params] as const,
  detalles: () => [...postulacionKeys.todas, "detalle"] as const,
  detalle: (id: string) => [...postulacionKeys.detalles(), id] as const,
  miPostulacion: () => [...postulacionKeys.todas, "mi-postulacion"] as const,
  definicion: () => ["formulario-definicion"] as const,
  paises: () => ["catalogos", "paises"] as const,
  provincias: (pais?: string) => ["catalogos", "provincias", pais] as const,
  cantones: (provincia?: string, pais?: string) =>
    ["catalogos", "cantones", pais, provincia] as const,
}

/**
 * Hook para obtener una postulación específica por su ID.
 */
export function usePostulacion(
  id?: string,
  opciones?: Partial<UseQueryOptions<PostulacionDto, unknown>>,
) {
  return useQuery({
    queryKey: postulacionKeys.detalle(id ?? ""),
    queryFn: ({ signal }) => obtenerPostulacion(id!, signal),
    enabled: Boolean(id),
    ...opciones,
  })
}

/**
 * Hook para listar postulaciones con filtros y paginación (uso analista).
 */
export function useListarPostulaciones(
  params?: ListarPostulacionesParams,
  opciones?: Partial<UseQueryOptions<Pagina<PostulacionDto>, unknown>>,
) {
  return useQuery({
    queryKey: postulacionKeys.lista(params),
    queryFn: ({ signal }) => listarPostulaciones(params, signal),
    ...opciones,
  })
}

/**
 * Hook para obtener la definición del contrato del EF-12 (catálogos y reglas).
 */
export function useDefinicionFormulario(
  opciones?: Partial<UseQueryOptions<DefinicionFormularioEF12, unknown>>,
) {
  return useQuery({
    queryKey: postulacionKeys.definicion(),
    queryFn: ({ signal }) => obtenerDefinicionFormulario(signal),
    staleTime: Number.POSITIVE_INFINITY, // La definición del formulario no cambia durante la sesión
    ...opciones,
  })
}

/**
 * Hook para obtener la lista de países disponibles.
 */
export function useCatalogosPaises() {
  return useQuery({
    queryKey: postulacionKeys.paises(),
    queryFn: ({ signal }) => obtenerPaises(signal),
    staleTime: Number.POSITIVE_INFINITY,
  })
}

/**
 * Hook para obtener las provincias de un país.
 */
export function useCatalogosProvincias(pais = "Ecuador") {
  return useQuery({
    queryKey: postulacionKeys.provincias(pais),
    queryFn: ({ signal }) => obtenerProvincias(pais, signal),
    enabled: Boolean(pais),
    staleTime: Number.POSITIVE_INFINITY,
  })
}

/**
 * Hook para obtener los cantones de una provincia.
 */
export function useCatalogosCantones(provincia?: string, pais = "Ecuador") {
  return useQuery({
    queryKey: postulacionKeys.cantones(provincia, pais),
    queryFn: ({ signal }) => obtenerCantones(provincia!, pais, signal),
    enabled: Boolean(provincia),
    staleTime: Number.POSITIVE_INFINITY,
  })
}

/**
 * Mutación para crear un nuevo borrador de postulación.
 */
export function useCrearPostulacion() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (dto?: CrearPostulacionDto) => crearPostulacion(dto),
    onSuccess: (nuevaPostulacion) => {
      queryClient.invalidateQueries({ queryKey: postulacionKeys.todas })
      toast.success("Borrador de postulación creado.")
      return nuevaPostulacion
    },
    onError: (error) => {
      const mensaje = esErrorApi(error)
        ? error.mensaje
        : "No se pudo crear el borrador de la postulación."
      toast.error(mensaje)
    },
  })
}

/**
 * Mutación para guardar una sección individual del formulario EF-12 (1 a 9).
 */
export function useGuardarSeccion() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      id,
      seccion,
      datos,
    }: {
      id: string
      seccion: number
      datos: unknown
    }) => guardarSeccion(id, seccion, datos),
    onSuccess: (postulacionActualizada, variables) => {
      queryClient.invalidateQueries({
        queryKey: postulacionKeys.detalle(variables.id),
      })
      queryClient.invalidateQueries({
        queryKey: postulacionKeys.todas,
      })
      toast.success(`Sección ${variables.seccion} guardada correctamente.`)
      return postulacionActualizada
    },
    onError: (error, variables) => {
      const mensaje = esErrorApi(error)
        ? error.mensaje
        : `Error al guardar la sección ${variables.seccion}.`
      toast.error(mensaje)
    },
  })
}

/**
 * Mutación para actualizar el contenido global del borrador.
 */
export function useActualizarPostulacion() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      id,
      content,
    }: {
      id: string
      content: Record<string, unknown>
    }) => actualizarPostulacion(id, content),
    onSuccess: (postulacionActualizada, variables) => {
      queryClient.invalidateQueries({
        queryKey: postulacionKeys.detalle(variables.id),
      })
      toast.success("Borrador guardado exitosamente.")
      return postulacionActualizada
    },
    onError: (error) => {
      const mensaje = esErrorApi(error)
        ? error.mensaje
        : "No se pudo actualizar el borrador."
      toast.error(mensaje)
    },
  })
}

/**
 * Mutación para sellar y enviar la postulación definitivamente.
 */
export function useSellarPostulacion() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      id,
      consentimiento,
    }: {
      id: string
      consentimiento?: unknown
    }) => sellarPostulacion(id, consentimiento),
    onSuccess: (postulacionSellada, variables) => {
      queryClient.invalidateQueries({
        queryKey: postulacionKeys.detalle(variables.id),
      })
      queryClient.invalidateQueries({
        queryKey: postulacionKeys.todas,
      })
      toast.success("¡Postulación enviada exitosamente!")
      return postulacionSellada
    },
    onError: (error) => {
      const mensaje = esErrorApi(error)
        ? error.mensaje
        : "No se pudo enviar la postulación."
      toast.error(mensaje)
    },
  })
}

/**
 * Mutación para eliminar un borrador de postulación.
 */
export function useEliminarPostulacion() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => eliminarPostulacion(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: postulacionKeys.todas })
      toast.success("Postulación eliminada.")
    },
    onError: (error) => {
      const mensaje = esErrorApi(error)
        ? error.mensaje
        : "No se pudo eliminar la postulación."
      toast.error(mensaje)
    },
  })
}
