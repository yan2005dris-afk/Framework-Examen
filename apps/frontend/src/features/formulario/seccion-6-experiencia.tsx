import { duracionExperienciaImprimible } from "./modelo"
import { HistoryIcon } from "lucide-react"
import type { AdjuntoInfo } from "./adjunto-uploader"
import { EditorRegistros, type CampoRegistro } from "./editor-registros"
import { validarRegistroFormulario } from "./validacion"
import { AREAS_EF12 } from "./constantes"
import { calcularDuracionExperiencia } from "@/helpers/experiencia"

export interface ExperienciaItem {
  id: string
  tipo: "General" | "Dirección"
  cargo: string
  institucion: string
  fechaInicio: string
  fechaFin: string
  actualmenteLaborando?: boolean
  actividades: string
  descripcionFunciones: string
  documentoPdf?: AdjuntoInfo | null
}

export interface Seccion6Data {
  experiencias: ExperienciaItem[]
}

export interface Seccion6ExperienciaProps {
  fechaEnvio?: string
  datos: Seccion6Data
  onChange: (nuevosDatos: Seccion6Data) => void
  disabled?: boolean
  onSubirRespaldo?: (expId: string, archivo: File) => Promise<void>
  onEliminarRespaldo?: (expId: string) => Promise<void>
  onDescargarRespaldo?: (documentId: string) => void
  cargandoRespaldoId?: string | null
}

const CAMPOS: CampoRegistro<ExperienciaItem>[] = [
  {
    nombre: "tipo",
    etiqueta: "Tipo de experiencia *",
    tipo: "opcion",
    opciones: ["General", "Dirección"],
  },
  { nombre: "cargo", etiqueta: "Cargo desempeñado *" },
  { nombre: "institucion", etiqueta: "Empresa o institución *" },
  { nombre: "fechaInicio", etiqueta: "Fecha de inicio *", tipo: "fecha" },
  {
    nombre: "fechaFin",
    etiqueta: "Fecha de fin *",
    tipo: "fecha",
    deshabilitado: (r) => Boolean(r.actualmenteLaborando),
  },
  {
    nombre: "actualmenteLaborando",
    etiqueta: "Laborando actualmente aquí",
    tipo: "casilla",
    alCambiar: (r, valor) => ({
      ...r,
      actualmenteLaborando: valor === true,
      fechaFin: valor === true ? "" : r.fechaFin,
    }),
  },
  {
    nombre: "actividades",
    etiqueta: "Área de actividades *",
    tipo: "opcion",
    opciones: AREAS_EF12,
  },
  {
    nombre: "descripcionFunciones",
    etiqueta: "Descripción de funciones * (máximo 1000 caracteres)",
    tipo: "area",
  },
]

export function Seccion6Experiencia({
  datos,
  onChange,
  disabled = false,
  fechaEnvio,
  ...respaldos
}: Seccion6ExperienciaProps) {
  return (
    <div className="min-w-0 space-y-6">
      <div className="flex items-start gap-2 border-b pb-3">
        <HistoryIcon
          className="mt-0.5 size-5 shrink-0 text-primary"
          aria-hidden="true"
        />
        <div>
          <h2 className="text-lg font-semibold">6. Experiencia profesional</h2>
          <p className="text-xs text-muted-foreground">
            Trayectoria laboral general y cargos directivos. La duración del
            envío se conserva en el expediente sellado.
          </p>
        </div>
      </div>
      <EditorRegistros<ExperienciaItem>
        titulo="Experiencias laborales"
        singular="experiencia"
        registros={datos.experiencias}
        crear={() => ({
          id: crypto.randomUUID(),
          tipo: "General",
          cargo: "",
          institucion: "",
          fechaInicio: "",
          fechaFin: "",
          actualmenteLaborando: false,
          actividades: "",
          descripcionFunciones: "",
          documentoPdf: null,
        })}
        campos={CAMPOS}
        columnas={[
          {
            titulo: "Tipo / cargo",
            contenido: (r) => (
              <>
                <strong>{r.cargo}</strong>
                <span className="block text-xs text-muted-foreground">
                  {r.tipo}
                </span>
              </>
            ),
          },
          { titulo: "Institución", contenido: (r) => r.institucion },
          {
            titulo: "Período / duración",
            contenido: (r) => (
              <>
                {r.fechaInicio} –{" "}
                {r.actualmenteLaborando
                  ? fechaEnvio
                    ? "Vigente al envío"
                    : "Actualidad"
                  : r.fechaFin}
                <span className="block text-xs text-muted-foreground">
                  {fechaEnvio
                    ? duracionExperienciaImprimible(r, fechaEnvio)
                    : calcularDuracionExperiencia(
                        r.fechaInicio,
                        r.fechaFin,
                        r.actualmenteLaborando,
                      ).texto}
                </span>
              </>
            ),
          },
        ]}
        onChange={(registros) =>
          onChange({ ...datos, experiencias: registros })
        }
        validar={(registro) => validarRegistroFormulario(6, registro)}
        disabled={disabled}
        {...respaldos}
      />
    </div>
  )
}
