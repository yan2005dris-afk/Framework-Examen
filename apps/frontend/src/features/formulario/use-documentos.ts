import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
import {
  abrirDocumentoSeguro,
  calcularCuotaUtilizada,
  eliminarDocumento,
  listarDocumentos,
  subirDocumento,
  type DocumentoDto,
} from "@/lib/clientes/documentos"
import { esErrorApi } from "@/lib/api-error"

export const clavesDocumentos = {
  todos: ["documentos"] as const,
  postulacion: (submissionId: string) =>
    [...clavesDocumentos.todos, submissionId] as const,
}

/**
 * Hook para consultar y vigilar los documentos adjuntos de una postulación.
 */
export function useDocumentosPostulacion(submissionId?: string) {
  const query = useQuery({
    queryKey: clavesDocumentos.postulacion(submissionId ?? ""),
    queryFn: ({ signal }) => listarDocumentos(submissionId!, signal),
    enabled: Boolean(submissionId),
    staleTime: 30_000,
  })

  const documentos = query.data ?? []
  const cuota = calcularCuotaUtilizada(documentos)

  return {
    ...query,
    documentos,
    cuota,
  }
}

/**
 * Hook para subir un respaldo PDF enlazándolo al registro guardado.
 */
export function useSubirDocumento(submissionId?: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      seccion,
      archivo,
      recordId,
    }: {
      seccion: number
      archivo: File
      recordId?: string
    }) => {
      if (!submissionId) {
        throw new Error(
          "Debe guardar el borrador antes de adjuntar documentos.",
        )
      }
      return subirDocumento(submissionId, seccion, archivo, recordId)
    },
    onSuccess: (nuevoDoc: DocumentoDto) => {
      if (submissionId) {
        queryClient.setQueryData<DocumentoDto[]>(
          clavesDocumentos.postulacion(submissionId),
          (prev) => {
            const filtrados = (prev ?? []).filter(
              (d) =>
                !(
                  d.section === nuevoDoc.section &&
                  d.recordId === nuevoDoc.recordId
                ),
            )
            return [...filtrados, nuevoDoc]
          },
        )
      }
      toast.success(`Respaldo «${nuevoDoc.name}» cargado correctamente.`)
    },
    onError: (err: unknown) => {
      if (esErrorApi(err)) {
        if (err.codigo === "archivo_muy_grande") {
          toast.error("El archivo excede el tamaño máximo permitido.")
        } else if (err.codigo === "cuota_documentos_excedida") {
          toast.error(
            "Ha excedido la cuota total de 50 MiB en documentos de postulación.",
          )
        } else if (err.codigo === "tipo_no_soportado") {
          toast.error(
            "El archivo no es un documento PDF válido o está corrupto.",
          )
        } else if (err.codigo === "registro_invalido") {
          toast.error(
            "El registro debe guardarse en el borrador antes de asociar su documento.",
          )
        } else {
          toast.error(err.message || "Error al subir el documento de respaldo.")
        }
      } else {
        toast.error("No se pudo conectar con el servidor para subir el PDF.")
      }
    },
  })
}

/**
 * Hook para eliminar un respaldo PDF de la postulación.
 */
export function useEliminarDocumento(submissionId?: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (documentId: string) => eliminarDocumento(documentId),
    onSuccess: (_, documentId) => {
      if (submissionId) {
        queryClient.setQueryData<DocumentoDto[]>(
          clavesDocumentos.postulacion(submissionId),
          (prev) => (prev ?? []).filter((d) => d.id !== documentId),
        )
      }
      toast.success("Documento eliminado correctamente.")
    },
    onError: (err: unknown) => {
      if (esErrorApi(err)) {
        toast.error(
          err.message || "No se pudo eliminar el documento del servidor.",
        )
      } else {
        toast.error("Error al intentar eliminar el documento.")
      }
    },
  })
}

/**
 * Abre de forma segura el documento en una nueva pestaña mediante su URL temporal.
 */
export function useDescargarDocumento() {
  return useMutation({
    mutationFn: (documentId: string) => abrirDocumentoSeguro(documentId),
    onError: (err: unknown) => {
      if (esErrorApi(err)) {
        toast.error(
          err.message || "El enlace temporal ha expirado. Vuelva a intentarlo.",
        )
      } else {
        toast.error("No se pudo abrir el documento.")
      }
    },
  })
}
