import { AwardIcon } from "lucide-react"
import type { AdjuntoInfo } from "./adjunto-uploader"
import { EditorRegistros, type CampoRegistro } from "./editor-registros"
import { validarRegistroFormulario } from "./validacion"

export interface CapacitacionItem {
  id: string
  tipo: "Curso" | "Seminario" | "Taller"
  nombre: string
  institucion: string
  fechaInicio: string
  fechaFin: string
  horas: number
  documentoPdf?: AdjuntoInfo | null
}

export interface Seccion8Data {
  capacitaciones: CapacitacionItem[]
}

export interface Seccion8CapacitacionProps {
  datos: Seccion8Data
  onChange: (nuevosDatos: Seccion8Data) => void
  disabled?: boolean
  onSubirRespaldo?: (capId: string, archivo: File) => Promise<void>
  onEliminarRespaldo?: (capId: string) => Promise<void>
  onDescargarRespaldo?: (documentId: string) => void
  cargandoRespaldoId?: string | null
}

const CAMPOS: CampoRegistro<CapacitacionItem>[] = [
  {
    nombre: "tipo",
    etiqueta: "Tipo de evento *",
    tipo: "opcion",
    opciones: ["Curso", "Seminario", "Taller"],
  },
  { nombre: "nombre", etiqueta: "Nombre del evento *" },
  { nombre: "institucion", etiqueta: "Institución organizadora *" },
  { nombre: "fechaInicio", etiqueta: "Fecha de inicio *", tipo: "fecha" },
  { nombre: "fechaFin", etiqueta: "Fecha de fin *", tipo: "fecha" },
  { nombre: "horas", etiqueta: "Horas certificadas *", tipo: "numero" },
]

export function Seccion8Capacitacion({
  datos,
  onChange,
  disabled = false,
  ...respaldos
}: Seccion8CapacitacionProps) {
  return (
    <div className="min-w-0 space-y-6">
      <div className="flex items-start gap-2 border-b pb-3">
        <AwardIcon
          className="mt-0.5 size-5 shrink-0 text-primary"
          aria-hidden="true"
        />
        <div>
          <h2 className="text-lg font-semibold">
            8. Capacitación y actualización
          </h2>
          <p className="text-xs text-muted-foreground">
            Cursos, seminarios y talleres declarados con su duración y respaldo.
          </p>
        </div>
      </div>
      <EditorRegistros<CapacitacionItem>
        titulo="Eventos de capacitación"
        singular="capacitación"
        registros={datos.capacitaciones}
        crear={() => ({
          id: crypto.randomUUID(),
          tipo: "Curso",
          nombre: "",
          institucion: "",
          fechaInicio: "",
          fechaFin: "",
          horas: 0,
          documentoPdf: null,
        })}
        campos={CAMPOS}
        columnas={[
          {
            titulo: "Evento / tipo",
            contenido: (r) => (
              <>
                <strong>{r.nombre}</strong>
                <span className="block text-xs text-muted-foreground">
                  {r.tipo}
                </span>
              </>
            ),
          },
          { titulo: "Institución", contenido: (r) => r.institucion },
          {
            titulo: "Fechas",
            contenido: (r) => (
              <>
                {r.fechaInicio} – {r.fechaFin}
              </>
            ),
          },
          {
            titulo: "Horas",
            contenido: (r) => <span className="font-mono">{r.horas}</span>,
          },
        ]}
        onChange={(registros) =>
          onChange({ ...datos, capacitaciones: registros })
        }
        validar={(registro) => validarRegistroFormulario(8, registro)}
        disabled={disabled}
        {...respaldos}
      />
      <p className="text-sm font-semibold">
        Total declarado:{" "}
        {datos.capacitaciones.reduce(
          (total, registro) => total + (registro.horas || 0),
          0,
        )}{" "}
        horas
      </p>
    </div>
  )
}
