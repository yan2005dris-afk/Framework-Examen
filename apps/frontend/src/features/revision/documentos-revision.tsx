import { useEffect, useRef, useState } from "react"
import { Button } from "@/components/ui/button"
import { EstadoCargando, EstadoError } from "@/components/estados"
import {
  obtenerEnlaceDocumentoRevision,
  type DocumentoRevision,
} from "@/lib/clientes/revision"
import { enlaceTemporalValido, fechaRevision } from "./modelo"
import { useDocumentosRevision } from "./queries"

function DocumentoDescargable({ documento }: { documento: DocumentoRevision }) {
  const [enlace, cambiarEnlace] = useState<{
    url: string
    expiresAt: string
  } | null>(null)
  const [cargando, cambiarCargando] = useState(false)
  const [error, cambiarError] = useState<unknown>()
  const peticion = useRef<AbortController | null>(null)
  useEffect(
    () => () => {
      peticion.current?.abort()
      peticion.current = null
    },
    [],
  )
  useEffect(() => {
    if (!enlace) return
    const temporizador = window.setTimeout(
      () => cambiarEnlace(null),
      Math.max(0, Date.parse(enlace.expiresAt) - Date.now()),
    )
    return () => window.clearTimeout(temporizador)
  }, [enlace])
  async function solicitar() {
    if (peticion.current) return
    const controlador = new AbortController()
    peticion.current = controlador
    cambiarCargando(true)
    cambiarError(undefined)
    cambiarEnlace(null)
    try {
      const respuesta = await obtenerEnlaceDocumentoRevision(
        documento.id,
        controlador.signal,
      )
      if (controlador.signal.aborted) return
      if (!enlaceTemporalValido(respuesta))
        throw new Error(
          "El enlace recibido no es válido o ya expiró. Solicite uno nuevo.",
        )
      cambiarEnlace(respuesta)
    } catch (fallo) {
      if (!controlador.signal.aborted) cambiarError(fallo)
    } finally {
      if (peticion.current === controlador) {
        peticion.current = null
        cambiarCargando(false)
      }
    }
  }
  return (
    <li className="space-y-3 rounded-lg border p-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0 flex-1">
          <p className="break-words font-medium">{documento.name}</p>
          <p className="text-xs text-muted-foreground">
            Sección {documento.section} · Registro{" "}
            {documento.recordId || "de sección"} ·{" "}
            {(documento.sizeBytes / 1024).toFixed(1)} KiB · {documento.mime}
          </p>
        </div>
        <Button
          variant="outline"
          size="sm"
          disabled={cargando}
          onClick={() => void solicitar()}
        >
          {cargando ? "Solicitando…" : "Solicitar descarga"}
        </Button>
      </div>
      <p className="break-all font-mono text-xs text-muted-foreground">
        SHA-256: {documento.sha256}
      </p>
      {enlace && (
        <p className="text-sm" role="status">
          <a
            className="font-medium text-primary underline underline-offset-4"
            href={enlace.url}
            target="_blank"
            rel="noopener noreferrer"
            referrerPolicy="no-referrer"
            onClick={(e) => {
              if (!enlaceTemporalValido(enlace)) {
                e.preventDefault()
                cambiarEnlace(null)
              }
            }}
          >
            Abrir / descargar PDF
          </a>
          <span className="ml-2 text-muted-foreground">
            Vence: {fechaRevision(enlace.expiresAt)}. Si falla o caduca,
            solicite otro enlace.
          </span>
        </p>
      )}
      {error !== undefined && (
        <EstadoError
          error={error}
          titulo="Descarga no disponible"
          onReintentar={() => void solicitar()}
        />
      )}
    </li>
  )
}

export function DocumentosRevision({ id }: { id: string }) {
  const consulta = useDocumentosRevision(id)
  return (
    <section className="space-y-4" aria-labelledby="titulo-respaldos">
      <h2 id="titulo-respaldos" className="text-lg font-semibold">
        Certificados y respaldos
      </h2>
      <p className="text-sm text-muted-foreground">
        Cada descarga exige autorización y registra el acceso. Los enlaces son
        temporales y se sirven directamente desde el almacenamiento.
      </p>
      {consulta.isPending ? (
        <EstadoCargando mensaje="Consultando certificados…" />
      ) : consulta.isError ? (
        <EstadoError
          error={consulta.error}
          onReintentar={() => void consulta.refetch()}
        />
      ) : consulta.data.length === 0 ? (
        <p className="text-sm text-muted-foreground">
          No hay documentos registrados para este expediente.
        </p>
      ) : (
        <ul className="space-y-3">
          {consulta.data.map((d) => (
            <DocumentoDescargable key={d.id} documento={d} />
          ))}
        </ul>
      )}
    </section>
  )
}
