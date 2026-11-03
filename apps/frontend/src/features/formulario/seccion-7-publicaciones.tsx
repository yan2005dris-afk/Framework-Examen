import { BookOpenIcon } from "lucide-react"
import type { AdjuntoInfo } from "./adjunto-uploader"
import { EditorRegistros, type CampoRegistro } from "./editor-registros"
import { validarRegistroFormulario } from "./validacion"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

export interface PublicacionItem {
  id: string
  titulo: string
  tipo: "Publicación" | "Trabajo de investigación"
  medio: string
  fechaPublicacion: string
  relacionCampo: "Sí" | "No"
  documentoPdf?: AdjuntoInfo | null
}

export interface Seccion7Data {
  cuentaConPublicaciones: "Sí" | "No" | ""
  publicaciones: PublicacionItem[]
}

export interface Seccion7PublicacionesProps {
  datos: Seccion7Data
  onChange: (nuevosDatos: Seccion7Data) => void
  disabled?: boolean
  onSubirRespaldo?: (pubId: string, archivo: File) => Promise<void>
  onEliminarRespaldo?: (pubId: string) => Promise<void>
  onDescargarRespaldo?: (documentId: string) => void
  cargandoRespaldoId?: string | null
}

const CAMPOS: CampoRegistro<PublicacionItem>[] = [
  { nombre: "titulo", etiqueta: "Título de la publicación *" },
  {
    nombre: "tipo",
    etiqueta: "Tipo *",
    tipo: "opcion",
    opciones: ["Publicación", "Trabajo de investigación"],
  },
  { nombre: "medio", etiqueta: "Medio de publicación o institución *" },
  {
    nombre: "fechaPublicacion",
    etiqueta: "Fecha de publicación *",
    tipo: "fecha",
  },
  {
    nombre: "relacionCampo",
    etiqueta: "¿Relacionada con el campo del concurso? *",
    tipo: "opcion",
    opciones: ["Sí", "No"],
  },
]

export function Seccion7Publicaciones({
  datos,
  onChange,
  disabled = false,
  ...respaldos
}: Seccion7PublicacionesProps) {
  return (
    <div className="min-w-0 space-y-6">
      <div className="flex items-start gap-2 border-b pb-3">
        <BookOpenIcon
          className="mt-0.5 size-5 shrink-0 text-primary"
          aria-hidden="true"
        />
        <div>
          <h2 className="text-lg font-semibold">
            7. Publicaciones e investigaciones
          </h2>
          <p className="text-xs text-muted-foreground">
            Libros, artículos y trabajos de investigación relacionados con las
            áreas del concurso.
          </p>
        </div>
      </div>
      <div className="space-y-2">
        <Label htmlFor="sec7-tiene-pub">
          ¿Cuenta con publicaciones o trabajos de investigación? *
        </Label>
        <Select
          value={datos.cuentaConPublicaciones}
          disabled={disabled}
          onValueChange={(valor) => {
            if (valor === "Sí" || valor === "No")
              onChange({ ...datos, cuentaConPublicaciones: valor })
          }}
        >
          <SelectTrigger id="sec7-tiene-pub">
            <SelectValue placeholder="Seleccione Sí o No" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="Sí">Sí</SelectItem>
            <SelectItem value="No" disabled={datos.publicaciones.length > 0}>
              No
            </SelectItem>
          </SelectContent>
        </Select>
        <p className="text-xs text-muted-foreground">
          Para declarar «No», elimine primero los registros con su confirmación.
          Cambiar la respuesta no borra publicaciones silenciosamente.
        </p>
      </div>
      {datos.cuentaConPublicaciones === "Sí" && (
        <EditorRegistros<PublicacionItem>
          titulo="Publicaciones e investigaciones"
          singular="publicación"
          registros={datos.publicaciones}
          crear={() => ({
            id: crypto.randomUUID(),
            titulo: "",
            tipo: "Publicación",
            medio: "",
            fechaPublicacion: "",
            relacionCampo: "Sí",
            documentoPdf: null,
          })}
          campos={CAMPOS}
          columnas={[
            {
              titulo: "Título / tipo",
              contenido: (r) => (
                <>
                  <strong>{r.titulo}</strong>
                  <span className="block text-xs text-muted-foreground">
                    {r.tipo}
                  </span>
                </>
              ),
            },
            { titulo: "Medio", contenido: (r) => r.medio },
            { titulo: "Fecha", contenido: (r) => r.fechaPublicacion },
            {
              titulo: "Relación con el campo",
              contenido: (r) => r.relacionCampo,
            },
          ]}
          onChange={(registros) =>
            onChange({ ...datos, publicaciones: registros })
          }
          validar={(registro) => validarRegistroFormulario(7, registro)}
          disabled={disabled}
          {...respaldos}
        />
      )}
    </div>
  )
}
