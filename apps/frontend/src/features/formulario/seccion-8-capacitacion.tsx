import { AwardIcon, PlusIcon, Trash2Icon } from "lucide-react"

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
}

export function Seccion8Capacitacion({
  datos,
  onChange,
  disabled = false,
}: Seccion8CapacitacionProps) {
  function agregarCapacitacion() {
    const nueva: CapacitacionItem = {
      id: `cap-${Date.now()}-${Math.random().toString(36).substring(2, 7)}`,
      tipo: "Curso",
      nombre: "",
      institucion: "",
      fechaInicio: "",
      fechaFin: "",
      horas: 40,
      documentoPdf: null,
    }
    onChange({
      ...datos,
      capacitaciones: [...datos.capacitaciones, nueva],
    })
  }

  function actualizarCapacitacion(
    index: number,
    campo: keyof CapacitacionItem,
    valor: unknown,
  ) {
    const lista = [...datos.capacitaciones]
    lista[index] = {
      ...lista[index],
      [campo]: valor,
    }
    onChange({
      ...datos,
      capacitaciones: lista,
    })
  }

  function eliminarCapacitacion(index: number) {
    onChange({
      ...datos,
      capacitaciones: datos.capacitaciones.filter((_, i) => i !== index),
    })
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center gap-2 border-b pb-3">
        <AwardIcon className="size-5 text-primary" aria-hidden="true" />
        <div>
          <h2 className="text-lg font-semibold">
            8. Capacitación y Actualización
          </h2>
          <p className="text-xs text-muted-foreground">
            Cursos, seminarios o talleres realizados en áreas afines (mínimo de
            horas certificadas).
          </p>
        </div>
      </div>

      <div className="flex items-center justify-between">
        <span className="text-sm font-semibold text-foreground">
          Eventos de Capacitación ({datos.capacitaciones.length})
        </span>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={agregarCapacitacion}
          disabled={disabled}
          className="flex items-center gap-1.5"
        >
          <PlusIcon className="size-4" aria-hidden="true" />
          <span>Agregar capacitación</span>
        </Button>
      </div>

      {datos.capacitaciones.length === 0 ? (
        <div className="rounded-lg border border-dashed p-8 text-center text-sm text-muted-foreground">
          <p>No ha registrado capacitaciones adicionales.</p>
          <p className="mt-1 text-xs">
            (Esta sección admite una lista declarada vacía si no dispone de
            certificados adicionales).
          </p>
        </div>
      ) : (
        <div className="flex flex-col gap-5">
          {datos.capacitaciones.map((cap, index) => (
            <div
              key={cap.id}
              className="flex flex-col gap-4 rounded-lg border bg-card p-4 shadow-2xs"
            >
              <div className="flex items-center justify-between border-b pb-2">
                <span className="text-xs font-semibold text-muted-foreground uppercase">
                  Capacitación #{index + 1} ({cap.tipo})
                </span>
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  onClick={() => eliminarCapacitacion(index)}
                  disabled={disabled}
                  className="text-destructive hover:bg-destructive/10"
                >
                  <Trash2Icon className="size-4" aria-hidden="true" />
                  <span className="ml-1 text-xs">Eliminar</span>
                </Button>
              </div>

              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
                <div className="flex flex-col gap-1.5">
                  <Label className="text-xs">Tipo de Evento *</Label>
                  <Select
                    value={cap.tipo}
                    onValueChange={(val) =>
                      actualizarCapacitacion(
                        index,
                        "tipo",
                        val as "Curso" | "Seminario" | "Taller",
                      )
                    }
                    disabled={disabled}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="Curso">Curso</SelectItem>
                      <SelectItem value="Seminario">Seminario</SelectItem>
                      <SelectItem value="Taller">Taller</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <div className="flex flex-col gap-1.5 sm:col-span-2">
                  <Label className="text-xs">
                    Nombre del Evento de Capacitación *
                  </Label>
                  <Input
                    value={cap.nombre}
                    onChange={(e) =>
                      actualizarCapacitacion(index, "nombre", e.target.value)
                    }
                    disabled={disabled}
                    placeholder="Ej. Diplomado en Normas Internacionales de Información Financiera..."
                  />
                </div>

                <div className="flex flex-col gap-1.5 sm:col-span-2 lg:col-span-1">
                  <Label className="text-xs">Institución que Certifica *</Label>
                  <Input
                    value={cap.institucion}
                    onChange={(e) =>
                      actualizarCapacitacion(
                        index,
                        "institucion",
                        e.target.value,
                      )
                    }
                    disabled={disabled}
                    placeholder="Entidad capacitadora o universidad"
                  />
                </div>

                <div className="flex flex-col gap-1.5">
                  <Label className="text-xs">Fecha de Inicio *</Label>
                  <Input
                    type="date"
                    value={cap.fechaInicio}
                    onChange={(e) =>
                      actualizarCapacitacion(
                        index,
                        "fechaInicio",
                        e.target.value,
                      )
                    }
                    disabled={disabled}
                  />
                </div>

                <div className="flex flex-col gap-1.5">
                  <Label className="text-xs">Fecha de Fin *</Label>
                  <Input
                    type="date"
                    value={cap.fechaFin}
                    onChange={(e) =>
                      actualizarCapacitacion(index, "fechaFin", e.target.value)
                    }
                    disabled={disabled}
                  />
                </div>

                <div className="flex flex-col gap-1.5">
                  <Label className="text-xs">Número de Horas *</Label>
                  <Input
                    type="number"
                    min={1}
                    value={cap.horas || ""}
                    onChange={(e) =>
                      actualizarCapacitacion(
                        index,
                        "horas",
                        Number.parseInt(e.target.value, 10) || 0,
                      )
                    }
                    disabled={disabled}
                    placeholder="40"
                  />
                </div>
              </div>

              <AdjuntoUploader
                label="Certificado o Diploma de Aprobación (PDF)"
                maxMB={2}
                valor={cap.documentoPdf}
                onArchivoSeleccionado={(adjunto) =>
                  actualizarCapacitacion(index, "documentoPdf", adjunto)
                }
                onEliminar={() =>
                  actualizarCapacitacion(index, "documentoPdf", null)
                }
                disabled={disabled}
                descripcion="Certificado escaneado que indique número de horas y aprobación (máx. 2 MB)."
              />
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
