import { BookOpenIcon, PlusIcon, Trash2Icon } from "lucide-react"

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
}

export function Seccion7Publicaciones({
  datos,
  onChange,
  disabled = false,
}: Seccion7PublicacionesProps) {
  function manejarCambioTiene(valor: "Sí" | "No") {
    onChange({
      ...datos,
      cuentaConPublicaciones: valor,
      publicaciones: valor === "No" ? [] : datos.publicaciones,
    })
  }

  function agregarPublicacion() {
    const nueva: PublicacionItem = {
      id: `pub-${Date.now()}-${Math.random().toString(36).substring(2, 7)}`,
      titulo: "",
      tipo: "Publicación",
      medio: "",
      fechaPublicacion: "",
      relacionCampo: "Sí",
      documentoPdf: null,
    }
    onChange({
      ...datos,
      publicaciones: [...datos.publicaciones, nueva],
    })
  }

  function actualizarPublicacion(
    index: number,
    campo: keyof PublicacionItem,
    valor: unknown,
  ) {
    const lista = [...datos.publicaciones]
    lista[index] = {
      ...lista[index],
      [campo]: valor,
    }
    onChange({
      ...datos,
      publicaciones: lista,
    })
  }

  function eliminarPublicacion(index: number) {
    onChange({
      ...datos,
      publicaciones: datos.publicaciones.filter((_, i) => i !== index),
    })
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center gap-2 border-b pb-3">
        <BookOpenIcon className="size-5 text-primary" aria-hidden="true" />
        <div>
          <h2 className="text-lg font-semibold">
            7. Publicaciones e Investigaciones
          </h2>
          <p className="text-xs text-muted-foreground">
            Libros, artículos indexados y trabajos de investigación científica
            relacionados con las áreas del concurso.
          </p>
        </div>
      </div>

      <div className="flex flex-col gap-2 rounded-lg border bg-muted/20 p-4">
        <Label htmlFor="sec7-tiene-pub" className="text-sm font-medium">
          ¿Cuenta con publicaciones o trabajos de investigación en el área
          financiera o afín? *
        </Label>
        <Select
          value={datos.cuentaConPublicaciones}
          onValueChange={(val) => manejarCambioTiene(val as "Sí" | "No")}
          disabled={disabled}
        >
          <SelectTrigger id="sec7-tiene-pub" className="w-full sm:w-60">
            <SelectValue placeholder="Seleccione Sí o No" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="Sí">Sí</SelectItem>
            <SelectItem value="No">No</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {datos.cuentaConPublicaciones === "Sí" && (
        <div className="flex flex-col gap-4">
          <div className="flex items-center justify-between">
            <span className="text-sm font-semibold text-foreground">
              Publicaciones Registradas ({datos.publicaciones.length})
            </span>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={agregarPublicacion}
              disabled={disabled}
              className="flex items-center gap-1.5"
            >
              <PlusIcon className="size-4" aria-hidden="true" />
              <span>Agregar publicación</span>
            </Button>
          </div>

          {datos.publicaciones.length === 0 ? (
            <p className="rounded-lg border border-dashed p-6 text-center text-xs text-muted-foreground">
              Ha indicado que «Sí» cuenta con publicaciones. Agregue al menos
              una haciendo clic en «Agregar publicación».
            </p>
          ) : (
            <div className="flex flex-col gap-5">
              {datos.publicaciones.map((pub, index) => (
                <div
                  key={pub.id}
                  className="flex flex-col gap-4 rounded-lg border bg-card p-4 shadow-2xs"
                >
                  <div className="flex items-center justify-between border-b pb-2">
                    <span className="text-xs font-semibold text-muted-foreground uppercase">
                      Publicación #{index + 1}
                    </span>
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      onClick={() => eliminarPublicacion(index)}
                      disabled={disabled}
                      className="text-destructive hover:bg-destructive/10"
                    >
                      <Trash2Icon className="size-4" aria-hidden="true" />
                      <span className="ml-1 text-xs">Eliminar</span>
                    </Button>
                  </div>

                  <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
                    <div className="flex flex-col gap-1.5 sm:col-span-2">
                      <Label className="text-xs">
                        Título de la Obra o Paper *
                      </Label>
                      <Input
                        value={pub.titulo}
                        onChange={(e) =>
                          actualizarPublicacion(index, "titulo", e.target.value)
                        }
                        disabled={disabled}
                        placeholder="Título de la publicación o artículo científico"
                      />
                    </div>

                    <div className="flex flex-col gap-1.5">
                      <Label className="text-xs">Tipo de Obra *</Label>
                      <Select
                        value={pub.tipo}
                        onValueChange={(val) =>
                          actualizarPublicacion(
                            index,
                            "tipo",
                            val as "Publicación" | "Trabajo de investigación",
                          )
                        }
                        disabled={disabled}
                      >
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="Publicación">
                            Publicación (Libro/Artículo)
                          </SelectItem>
                          <SelectItem value="Trabajo de investigación">
                            Trabajo de investigación
                          </SelectItem>
                        </SelectContent>
                      </Select>
                    </div>

                    <div className="flex flex-col gap-1.5 sm:col-span-2">
                      <Label className="text-xs">
                        Medio de Publicación / Editorial / Revista *
                      </Label>
                      <Input
                        value={pub.medio}
                        onChange={(e) =>
                          actualizarPublicacion(index, "medio", e.target.value)
                        }
                        disabled={disabled}
                        placeholder="Editorial, Revista Indexada (Scopus, Latindex, etc.)..."
                      />
                    </div>

                    <div className="flex flex-col gap-1.5">
                      <Label className="text-xs">Fecha de Publicación *</Label>
                      <Input
                        type="date"
                        value={pub.fechaPublicacion}
                        onChange={(e) =>
                          actualizarPublicacion(
                            index,
                            "fechaPublicacion",
                            e.target.value,
                          )
                        }
                        disabled={disabled}
                      />
                    </div>

                    <div className="flex flex-col gap-1.5">
                      <Label className="text-xs">
                        ¿Tiene relación directa con el campo de postulación? *
                      </Label>
                      <Select
                        value={pub.relacionCampo}
                        onValueChange={(val) =>
                          actualizarPublicacion(
                            index,
                            "relacionCampo",
                            val as "Sí" | "No",
                          )
                        }
                        disabled={disabled}
                      >
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="Sí">Sí</SelectItem>
                          <SelectItem value="No">No</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                  </div>

                  <AdjuntoUploader
                    label="Archivo de la Publicación o Certificado Editorial (PDF)"
                    maxMB={2}
                    valor={pub.documentoPdf}
                    onArchivoSeleccionado={(adjunto) =>
                      actualizarPublicacion(index, "documentoPdf", adjunto)
                    }
                    onEliminar={() =>
                      actualizarPublicacion(index, "documentoPdf", null)
                    }
                    disabled={disabled}
                    descripcion="Copia de portada, índice y créditos o constancia de publicación indexada (máx. 2 MB)."
                  />
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
