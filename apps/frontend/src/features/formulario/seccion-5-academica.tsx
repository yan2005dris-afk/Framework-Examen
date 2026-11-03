import { GraduationCapIcon } from "lucide-react"
import type { AdjuntoInfo } from "./adjunto-uploader"
import { EditorRegistros, type CampoRegistro } from "./editor-registros"
import { validarRegistroFormulario } from "./validacion"
import { AREAS_EF12 } from "./constantes"

export interface TituloItem {
  id: string
  nivel: string
  institucion: string
  tipo: "Nacional" | "Extranjera"
  titulo: string
  area: string
  registroSenescyt: string
  fechaRegistroSenescyt: string
  documentoPdf?: AdjuntoInfo | null
}

export interface Seccion5Data {
  titulos: TituloItem[]
}

export interface Seccion5AcademicaProps {
  datos: Seccion5Data
  onChange: (nuevosDatos: Seccion5Data) => void
  disabled?: boolean
  onSubirRespaldo?: (tituloId: string, archivo: File) => Promise<void>
  onEliminarRespaldo?: (tituloId: string) => Promise<void>
  onDescargarRespaldo?: (documentId: string) => void
  cargandoRespaldoId?: string | null
}

const CAMPOS: CampoRegistro<TituloItem>[] = [
  {
    nombre: "nivel",
    etiqueta: "Nivel de instrucción *",
    tipo: "opcion",
    opciones: [
      "Tercer Nivel de Grado",
      "Cuarto Nivel - Especialidad Académica",
      "Cuarto Nivel - Maestría Académica",
      "Cuarto Nivel PhD - Doctorado",
    ],
  },
  { nombre: "titulo", etiqueta: "Título obtenido *" },
  { nombre: "institucion", etiqueta: "Institución de educación superior *" },
  {
    nombre: "tipo",
    etiqueta: "Tipo de institución *",
    tipo: "opcion",
    opciones: ["Nacional", "Extranjera"],
  },
  {
    nombre: "area",
    etiqueta: "Área de conocimiento *",
    tipo: "opcion",
    opciones: AREAS_EF12,
  },
  { nombre: "registroSenescyt", etiqueta: "Registro SENESCYT *" },
  {
    nombre: "fechaRegistroSenescyt",
    etiqueta: "Fecha de registro SENESCYT *",
    tipo: "fecha",
  },
]

export function Seccion5Academica({
  datos,
  onChange,
  disabled = false,
  ...respaldos
}: Seccion5AcademicaProps) {
  return (
    <div className="min-w-0 space-y-6">
      <div className="flex items-start gap-2 border-b pb-3">
        <GraduationCapIcon
          className="mt-0.5 size-5 shrink-0 text-primary"
          aria-hidden="true"
        />
        <div>
          <h2 className="text-lg font-semibold">5. Formación académica</h2>
          <p className="text-xs text-muted-foreground">
            Títulos de tercer y cuarto nivel declarados en áreas afines al
            concurso.
          </p>
        </div>
      </div>
      <EditorRegistros<TituloItem>
        titulo="Títulos registrados"
        singular="título"
        registros={datos.titulos}
        crear={() => ({
          id: crypto.randomUUID(),
          nivel: "Tercer Nivel de Grado",
          institucion: "",
          tipo: "Nacional",
          titulo: "",
          area: "Finanzas",
          registroSenescyt: "",
          fechaRegistroSenescyt: "",
          documentoPdf: null,
        })}
        campos={CAMPOS}
        columnas={[
          {
            titulo: "Nivel / título",
            contenido: (r) => (
              <>
                <strong>{r.titulo}</strong>
                <span className="block text-xs text-muted-foreground">
                  {r.nivel}
                </span>
              </>
            ),
          },
          { titulo: "Institución", contenido: (r) => r.institucion },
          { titulo: "Registro SENESCYT", contenido: (r) => r.registroSenescyt },
        ]}
        onChange={(registros) => onChange({ ...datos, titulos: registros })}
        validar={(registro) => validarRegistroFormulario(5, registro)}
        disabled={disabled}
        {...respaldos}
      />
    </div>
  )
}
