import { GraduationCapIcon, PlusIcon, Trash2Icon } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { AdjuntoUploader, type AdjuntoInfo } from "./adjunto-uploader"

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
}

const NIVELES_CATALOGO = [
  "Tercer Nivel de Grado",
  "Cuarto Nivel - Especialidad Académica",
  "Cuarto Nivel - Maestría Académica",
  "Cuarto Nivel PhD - Doctorado",
]

const AREAS_CATALOGO = [
  "Finanzas",
  "Mercado de Capitales",
  "Administración de Empresas",
  "Economía",
  "Derecho o materias afines",
]

export function Seccion5Academica({
  datos,
  onChange,
  disabled = false,
}: Seccion5AcademicaProps) {
  function agregarTitulo() {
    const nuevo: TituloItem = {
      id: `tit-${Date.now()}-${Math.random().toString(36).substring(2, 7)}`,
      nivel: "Tercer Nivel de Grado",
      institucion: "",
      tipo: "Nacional",
      titulo: "",
      area: "Finanzas",
      registroSenescyt: "",
      fechaRegistroSenescyt: "",
      documentoPdf: null,
    }
    onChange({
      ...datos,
      titulos: [...datos.titulos, nuevo],
    })
  }

  function actualizarTitulo(
    index: number,
    campo: keyof TituloItem,
    valor: unknown,
  ) {
    const lista = [...datos.titulos]
    lista[index] = {
      ...lista[index],
      [campo]: valor,
    }
    onChange({
      ...datos,
      titulos: lista,
    })
  }

  function eliminarTitulo(index: number) {
    onChange({
      ...datos,
      titulos: datos.titulos.filter((_, i) => i !== index),
    })
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center gap-2 border-b pb-3">
        <GraduationCapIcon className="size-5 text-primary" aria-hidden="true" />
        <div>
          <h2 className="text-lg font-semibold">5. Formación Académica</h2>
          <p className="text-xs text-muted-foreground">
            Títulos de tercer y cuarto nivel registrados en SENESCYT en áreas
            afines al concurso.
          </p>
        </div>
      </div>

      <div className="flex items-center justify-between">
        <span className="text-sm font-semibold text-foreground">
          Títulos Registrados ({datos.titulos.length})
        </span>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={agregarTitulo}
          disabled={disabled}
          className="flex items-center gap-1.5"
        >
          <PlusIcon className="size-4" aria-hidden="true" />
          <span>Agregar título</span>
        </Button>
      </div>

      {datos.titulos.length === 0 ? (
        <div className="rounded-lg border border-dashed p-8 text-center text-sm text-muted-foreground">
          No ha registrado ningún título académico. Haga clic en «Agregar
          título» para registrar sus estudios superiores.
        </div>
      ) : (
        <div className="flex flex-col gap-5">
          {datos.titulos.map((titulo, index) => (
            <div
              key={titulo.id}
              className="flex flex-col gap-4 rounded-lg border bg-card p-4 shadow-2xs"
            >
              <div className="flex items-center justify-between border-b pb-2">
                <span className="text-xs font-semibold text-muted-foreground uppercase">
                  Título #{index + 1}
                </span>
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  onClick={() => eliminarTitulo(index)}
                  disabled={disabled}
                  className="text-destructive hover:bg-destructive/10"
                >
                  <Trash2Icon className="size-4" aria-hidden="true" />
                  <span className="ml-1 text-xs">Eliminar</span>
                </Button>
              </div>

              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
                <div className="flex flex-col gap-1.5">
                  <Label className="text-xs">Nivel de Instrucción *</Label>
                  <Select
                    value={titulo.nivel}
                    onValueChange={(val) =>
                      actualizarTitulo(index, "nivel", val)
                    }
                    disabled={disabled}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {NIVELES_CATALOGO.map((n) => (
                        <SelectItem key={n} value={n}>
                          {n}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>

                <div className="flex flex-col gap-1.5 sm:col-span-2 lg:col-span-1">
                  <Label className="text-xs">
                    Nombre del Título Obtenido *
                  </Label>
                  <Input
                    value={titulo.titulo}
                    onChange={(e) =>
                      actualizarTitulo(index, "titulo", e.target.value)
                    }
                    disabled={disabled}
                    placeholder="Ej. Economista, Ingeniero en Finanzas..."
                  />
                </div>

                <div className="flex flex-col gap-1.5 sm:col-span-2 lg:col-span-1">
                  <Label className="text-xs">
                    Institución de Educación Superior *
                  </Label>
                  <Input
                    value={titulo.institucion}
                    onChange={(e) =>
                      actualizarTitulo(index, "institucion", e.target.value)
                    }
                    disabled={disabled}
                    placeholder="Universidad o Escuela Politécnica"
                  />
                </div>

                <div className="flex flex-col gap-1.5">
                  <Label className="text-xs">Tipo de Institución *</Label>
                  <Select
                    value={titulo.tipo}
                    onValueChange={(val) =>
                      actualizarTitulo(
                        index,
                        "tipo",
                        val as "Nacional" | "Extranjera",
                      )
                    }
                    disabled={disabled}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="Nacional">Nacional</SelectItem>
                      <SelectItem value="Extranjera">Extranjera</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <div className="flex flex-col gap-1.5">
                  <Label className="text-xs">Área de Conocimiento *</Label>
                  <Select
                    value={titulo.area}
                    onValueChange={(val) =>
                      actualizarTitulo(index, "area", val)
                    }
                    disabled={disabled}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {AREAS_CATALOGO.map((a) => (
                        <SelectItem key={a} value={a}>
                          {a}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>

                <div className="flex flex-col gap-1.5">
                  <Label className="text-xs">N° Registro SENESCYT *</Label>
                  <Input
                    value={titulo.registroSenescyt}
                    onChange={(e) =>
                      actualizarTitulo(
                        index,
                        "registroSenescyt",
                        e.target.value,
                      )
                    }
                    disabled={disabled}
                    placeholder="1005-2015-123456"
                  />
                </div>

                <div className="flex flex-col gap-1.5">
                  <Label className="text-xs">Fecha Registro SENESCYT *</Label>
                  <Input
                    type="date"
                    value={titulo.fechaRegistroSenescyt}
                    onChange={(e) =>
                      actualizarTitulo(
                        index,
                        "fechaRegistroSenescyt",
                        e.target.value,
                      )
                    }
                    disabled={disabled}
                  />
                </div>
              </div>

              <AdjuntoUploader
                label="Copia de Título o Certificado SENESCYT (PDF)"
                maxMB={2}
                valor={titulo.documentoPdf}
                onArchivoSeleccionado={(adjunto) =>
                  actualizarTitulo(index, "documentoPdf", adjunto)
                }
                onEliminar={() => actualizarTitulo(index, "documentoPdf", null)}
                disabled={disabled}
                descripcion="Certificado de registro emitido por SENESCYT o título universitario escaneado (máx. 2 MB)."
              />
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
