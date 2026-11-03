import {
  ExternalLinkIcon,
  FileTextIcon,
  Loader2Icon,
  Trash2Icon,
  UploadCloudIcon,
} from "lucide-react"
import { useRef, type ChangeEvent } from "react"
import { toast } from "sonner"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"

export interface AdjuntoInfo {
  id?: string
  nombre: string
  tamano: string
  sizeBytes?: number
  fechaSubida?: string
  base64?: string
  referenciaId?: string
  sha256?: string
  archivo?: File
}

export interface AdjuntoUploaderProps {
  label?: string
  maxMB: number
  valor?: AdjuntoInfo | null
  onArchivoSeleccionado: (adjunto: AdjuntoInfo, archivo?: File) => void
  onEliminar: () => void
  onDescargar?: () => void
  disabled?: boolean
  cargando?: boolean
  error?: string | null
  descripcion?: string
}

function formatearTamano(bytes: number): string {
  if (bytes < 1024 * 1024) {
    return `${(bytes / 1024).toFixed(1)} KB`
  }
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

/**
 * Componente accesible para adjuntar y validar documentos PDF con límite estricto de tamaño.
 */
export function AdjuntoUploader({
  label = "Documento de respaldo (PDF)",
  maxMB,
  valor,
  onArchivoSeleccionado,
  onEliminar,
  onDescargar,
  disabled = false,
  cargando = false,
  error = null,
  descripcion,
}: AdjuntoUploaderProps) {
  const inputRef = useRef<HTMLInputElement>(null)
  const maxBytes = maxMB * 1024 * 1024

  function manejarCambioArchivo(evento: ChangeEvent<HTMLInputElement>) {
    const archivo = evento.target.files?.[0]
    if (!archivo) return

    if (
      archivo.type !== "application/pdf" &&
      !archivo.name.toLowerCase().endsWith(".pdf")
    ) {
      toast.error("Solo se permiten archivos en formato PDF.")
      if (inputRef.current) inputRef.current.value = ""
      return
    }

    if (archivo.size > maxBytes) {
      toast.error(
        `El archivo supera el tamaño máximo permitido de ${maxMB} MB (${formatearTamano(archivo.size)}).`,
      )
      if (inputRef.current) inputRef.current.value = ""
      return
    }

    const adjunto: AdjuntoInfo = {
      nombre: archivo.name,
      tamano: formatearTamano(archivo.size),
      sizeBytes: archivo.size,
      fechaSubida: new Date().toISOString(),
      archivo,
    }

    onArchivoSeleccionado(adjunto, archivo)
    if (inputRef.current) inputRef.current.value = ""
  }

  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex items-center justify-between">
        <label className="text-sm font-medium leading-none text-foreground">
          {label}
        </label>
        <Badge variant="outline" className="text-[11px] font-normal">
          Máx. {maxMB} MB (PDF)
        </Badge>
      </div>
      {descripcion && (
        <p className="text-xs text-muted-foreground">{descripcion}</p>
      )}

      <input
        ref={inputRef}
        type="file"
        accept="application/pdf,.pdf"
        className="hidden"
        onChange={manejarCambioArchivo}
        disabled={disabled || cargando}
      />

      {cargando ? (
        <div className="flex items-center justify-center gap-2 rounded-lg border border-dashed bg-muted/20 py-4 text-sm text-muted-foreground">
          <Loader2Icon className="size-4 animate-spin text-primary" />
          <span>Subiendo respaldo PDF al servidor…</span>
        </div>
      ) : valor?.nombre ? (
        <div className="flex items-center justify-between rounded-lg border bg-muted/30 px-3 py-2 text-sm">
          <div className="flex items-center gap-2 overflow-hidden">
            <FileTextIcon
              className="size-4 shrink-0 text-primary"
              aria-hidden="true"
            />
            <span className="truncate font-medium">{valor.nombre}</span>
            <span className="shrink-0 text-xs text-muted-foreground">
              ({valor.tamano})
            </span>
            {valor.id && (
              <Badge variant="secondary" className="text-[10px]">
                Confirmado
              </Badge>
            )}
          </div>
          <div className="flex items-center gap-1">
            {onDescargar && (valor.id || valor.referenciaId) && (
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={onDescargar}
                className="size-8 p-0 text-muted-foreground hover:text-foreground"
                title="Ver o descargar respaldo PDF"
              >
                <ExternalLinkIcon className="size-4" aria-hidden="true" />
                <span className="sr-only">Ver documento</span>
              </Button>
            )}
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={onEliminar}
              disabled={disabled || cargando}
              className="size-8 p-0 text-muted-foreground hover:text-destructive"
              title="Eliminar archivo adjunto"
            >
              <Trash2Icon className="size-4" aria-hidden="true" />
              <span className="sr-only">Eliminar archivo</span>
            </Button>
          </div>
        </div>
      ) : (
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={disabled || cargando}
          onClick={() => inputRef.current?.click()}
          className="flex w-full items-center justify-center gap-2 border-dashed py-5 hover:bg-muted/50"
        >
          <UploadCloudIcon
            className="size-4 text-muted-foreground"
            aria-hidden="true"
          />
          <span>Seleccionar archivo PDF (máx. {maxMB} MB)</span>
        </Button>
      )}

      {error && <p className="text-xs font-medium text-destructive">{error}</p>}
    </div>
  )
}
