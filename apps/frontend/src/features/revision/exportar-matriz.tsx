import { useEffect, useRef, useState } from "react"
import { DownloadIcon, Loader2Icon } from "lucide-react"
import { Button } from "@/components/ui/button"
import { EstadoError } from "@/components/estados"
import {
  exportarMatrizRevision,
  type FiltrosRevision,
  type FormatoMatriz,
} from "@/lib/clientes/revision"

/** No guarda archivos ni capacidades de descarga en la caché de consultas. */
export function ExportarMatriz({ filtros }: { filtros: FiltrosRevision }) {
  const [estado, cambiarEstado] = useState<
    "inactivo" | "generando" | "iniciado" | "cancelado" | "error"
  >("inactivo")
  const [error, cambiarError] = useState<unknown>()
  const [formato, cambiarFormato] = useState<FormatoMatriz>("xlsx")
  const peticion = useRef<AbortController | null>(null)
  useEffect(
    () => () => {
      peticion.current?.abort()
      peticion.current = null
    },
    [],
  )

  async function descargar(tipo: FormatoMatriz) {
    if (peticion.current) return
    const controlador = new AbortController()
    peticion.current = controlador
    cambiarFormato(tipo)
    cambiarEstado("generando")
    cambiarError(undefined)
    try {
      const archivo = await exportarMatrizRevision(
        filtros,
        tipo,
        controlador.signal,
      )
      if (controlador.signal.aborted) return
      const url = URL.createObjectURL(archivo)
      try {
        const enlace = document.createElement("a")
        enlace.href = url
        enlace.download = `matriz-ef13-${filtros.periodo}-${filtros.sector || "todos"}.${tipo}`
        document.body.append(enlace)
        try {
          enlace.click()
        } finally {
          enlace.remove()
        }
      } finally {
        // Se da tiempo al navegador para tomar el archivo; la URL nunca se persiste.
        window.setTimeout(() => URL.revokeObjectURL(url), 1000)
      }
      cambiarEstado("iniciado")
    } catch (fallo) {
      if (controlador.signal.aborted) return
      cambiarError(fallo)
      cambiarEstado("error")
    } finally {
      if (peticion.current === controlador) peticion.current = null
    }
  }

  return (
    <section
      className="space-y-4 rounded-xl border bg-card p-5"
      aria-labelledby="titulo-matriz"
    >
      <div>
        <h2 id="titulo-matriz" className="font-semibold">
          Matriz comparativa interna
        </h2>
        <p className="mt-1 text-sm text-muted-foreground">
          EF13-TECNICA-v1 · Propuesta pendiente de aprobación institucional. No
          es una lista pública de elegibles ni contiene calificaciones de
          méritos.
        </p>
      </div>
      <p className="text-sm text-muted-foreground">
        Exporta todos los expedientes de esta selección, no solo las filas
        cargadas. Excel y PDF usan las mismas columnas y filtros. Máximo 10.000
        expedientes; si se supera el límite, acote el sector. El archivo refleja
        el momento de generación.
      </p>
      <div className="flex flex-wrap gap-3">
        {(["xlsx", "pdf"] as const).map((tipo) => (
          <Button
            key={tipo}
            variant={tipo === "xlsx" ? "default" : "outline"}
            disabled={estado === "generando"}
            onClick={() => void descargar(tipo)}
          >
            {estado === "generando" && formato === tipo ? (
              <Loader2Icon className="animate-spin" />
            ) : (
              <DownloadIcon />
            )}
            {tipo === "xlsx" ? "Descargar Excel" : "Descargar PDF"}
          </Button>
        ))}
        {estado === "generando" && (
          <Button
            variant="ghost"
            onClick={() => {
              peticion.current?.abort()
              peticion.current = null
              cambiarEstado("cancelado")
            }}
          >
            Cancelar
          </Button>
        )}
      </div>
      <div aria-live="polite" className="text-sm text-muted-foreground">
        {estado === "generando" && `Generando ${formato.toUpperCase()}…`}
        {estado === "iniciado" &&
          "Descarga iniciada en el navegador. Revise su carpeta de descargas; conserve el archivo solo en un entorno autorizado."}
        {estado === "cancelado" && "Descarga cancelada."}
      </div>
      {estado === "error" && (
        <EstadoError
          error={error}
          titulo="No se pudo exportar la matriz"
          onReintentar={() => void descargar(formato)}
        />
      )}
    </section>
  )
}
