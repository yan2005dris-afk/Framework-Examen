import { ClockIcon, HistoryIcon, PlusIcon, Trash2Icon } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"
import { calcularDuracionExperiencia } from "@/utils/validators"
import { AdjuntoUploader, type AdjuntoInfo } from "./adjunto-uploader"

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
  datos: Seccion6Data
  onChange: (nuevosDatos: Seccion6Data) => void
  disabled?: boolean
}

export function Seccion6Experiencia({
  datos,
  onChange,
  disabled = false,
}: Seccion6ExperienciaProps) {
  function agregarExperiencia() {
    const nuevo: ExperienciaItem = {
      id: `exp-${Date.now()}-${Math.random().toString(36).substring(2, 7)}`,
      tipo: "General",
      cargo: "",
      institucion: "",
      fechaInicio: "",
      fechaFin: "",
      actualmenteLaborando: false,
      actividades: "",
      descripcionFunciones: "",
      documentoPdf: null,
    }
    onChange({
      ...datos,
      experiencias: [...datos.experiencias, nuevo],
    })
  }

  function actualizarExperiencia(
    index: number,
    campo: keyof ExperienciaItem,
    valor: unknown,
  ) {
    const lista = [...datos.experiencias]
    lista[index] = {
      ...lista[index],
      [campo]: valor,
    }
    onChange({
      ...datos,
      experiencias: lista,
    })
  }

  function eliminarExperiencia(index: number) {
    onChange({
      ...datos,
      experiencias: datos.experiencias.filter((_, i) => i !== index),
    })
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center gap-2 border-b pb-3">
        <HistoryIcon className="size-5 text-primary" aria-hidden="true" />
        <div>
          <h2 className="text-lg font-semibold">6. Experiencia Profesional</h2>
          <p className="text-xs text-muted-foreground">
            Trayectoria laboral general y cargos directivos o gerenciales en el
            sector financiero o afín.
          </p>
        </div>
      </div>

      <div className="flex items-center justify-between">
        <span className="text-sm font-semibold text-foreground">
          Experiencias Laborales ({datos.experiencias.length})
        </span>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={agregarExperiencia}
          disabled={disabled}
          className="flex items-center gap-1.5"
        >
          <PlusIcon className="size-4" aria-hidden="true" />
          <span>Agregar experiencia</span>
        </Button>
      </div>

      {datos.experiencias.length === 0 ? (
        <div className="rounded-lg border border-dashed p-8 text-center text-sm text-muted-foreground">
          No ha registrado ninguna experiencia laboral. Haga clic en «Agregar
          experiencia» para ingresar su trayectoria.
        </div>
      ) : (
        <div className="flex flex-col gap-5">
          {datos.experiencias.map((exp, index) => {
            const duracion = calcularDuracionExperiencia(
              exp.fechaInicio,
              exp.fechaFin,
              exp.actualmenteLaborando,
            )
            const longitudFunciones = (exp.descripcionFunciones || "").length

            return (
              <div
                key={exp.id}
                className="flex flex-col gap-4 rounded-lg border bg-card p-4 shadow-2xs"
              >
                <div className="flex items-center justify-between border-b pb-2">
                  <span className="text-xs font-semibold text-muted-foreground uppercase">
                    Experiencia #{index + 1} ({exp.tipo})
                  </span>
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={() => eliminarExperiencia(index)}
                    disabled={disabled}
                    className="text-destructive hover:bg-destructive/10"
                  >
                    <Trash2Icon className="size-4" aria-hidden="true" />
                    <span className="ml-1 text-xs">Eliminar</span>
                  </Button>
                </div>

                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
                  <div className="flex flex-col gap-1.5">
                    <Label className="text-xs">Tipo de Experiencia *</Label>
                    <Select
                      value={exp.tipo}
                      onValueChange={(val) =>
                        actualizarExperiencia(
                          index,
                          "tipo",
                          val as "General" | "Dirección",
                        )
                      }
                      disabled={disabled}
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="General">General</SelectItem>
                        <SelectItem value="Dirección">
                          Dirección / Gerencial
                        </SelectItem>
                      </SelectContent>
                    </Select>
                  </div>

                  <div className="flex flex-col gap-1.5 sm:col-span-2 lg:col-span-1">
                    <Label className="text-xs">Cargo Desempeñado *</Label>
                    <Input
                      value={exp.cargo}
                      onChange={(e) =>
                        actualizarExperiencia(index, "cargo", e.target.value)
                      }
                      disabled={disabled}
                      placeholder="Ej. Gerente de Riesgos, Jefe de Inversiones..."
                    />
                  </div>

                  <div className="flex flex-col gap-1.5 sm:col-span-2 lg:col-span-1">
                    <Label className="text-xs">Empresa o Institución *</Label>
                    <Input
                      value={exp.institucion}
                      onChange={(e) =>
                        actualizarExperiencia(
                          index,
                          "institucion",
                          e.target.value,
                        )
                      }
                      disabled={disabled}
                      placeholder="Nombre de la entidad empleadora"
                    />
                  </div>

                  <div className="flex flex-col gap-1.5">
                    <Label className="text-xs">Fecha de Inicio *</Label>
                    <Input
                      type="date"
                      value={exp.fechaInicio}
                      onChange={(e) =>
                        actualizarExperiencia(
                          index,
                          "fechaInicio",
                          e.target.value,
                        )
                      }
                      disabled={disabled}
                    />
                  </div>

                  <div className="flex flex-col gap-1.5">
                    <Label className="text-xs">Fecha de Fin</Label>
                    <Input
                      type="date"
                      value={exp.fechaFin}
                      onChange={(e) =>
                        actualizarExperiencia(index, "fechaFin", e.target.value)
                      }
                      disabled={disabled || exp.actualmenteLaborando}
                    />
                  </div>

                  <div className="flex items-center gap-2 pt-6">
                    <Checkbox
                      id={`lab-${exp.id}`}
                      checked={Boolean(exp.actualmenteLaborando)}
                      onCheckedChange={(checked) =>
                        actualizarExperiencia(
                          index,
                          "actualmenteLaborando",
                          Boolean(checked),
                        )
                      }
                      disabled={disabled}
                    />
                    <Label
                      htmlFor={`lab-${exp.id}`}
                      className="text-xs font-normal cursor-pointer"
                    >
                      Laborando actualmente aquí
                    </Label>
                  </div>
                </div>

                {exp.fechaInicio && (
                  <div className="flex items-center gap-2 rounded-md bg-muted/50 px-3 py-2 text-xs text-muted-foreground">
                    <ClockIcon
                      className="size-3.5 text-primary"
                      aria-hidden="true"
                    />
                    <span>
                      Duración calculada:{" "}
                      <strong className="text-foreground">
                        {duracion.texto}
                      </strong>{" "}
                      ({duracion.anios} a, {duracion.meses} m, {duracion.dias}{" "}
                      d)
                    </span>
                  </div>
                )}

                <div className="flex flex-col gap-1.5">
                  <Label className="text-xs">
                    Área de Actividades / Sector *
                  </Label>
                  <Input
                    value={exp.actividades}
                    onChange={(e) =>
                      actualizarExperiencia(
                        index,
                        "actividades",
                        e.target.value,
                      )
                    }
                    disabled={disabled}
                    placeholder="Financiero, Fiduciario, Inversiones, Auditoría..."
                  />
                </div>

                <div className="flex flex-col gap-1.5">
                  <div className="flex items-center justify-between">
                    <Label className="text-xs">
                      Descripción de Funciones Principales * (máx. 1000
                      caracteres)
                    </Label>
                    <span
                      className={`text-[11px] ${longitudFunciones > 1000 ? "font-semibold text-destructive" : "text-muted-foreground"}`}
                    >
                      {longitudFunciones} / 1000
                    </span>
                  </div>
                  <Textarea
                    value={exp.descripcionFunciones}
                    onChange={(e) =>
                      actualizarExperiencia(
                        index,
                        "descripcionFunciones",
                        e.target.value,
                      )
                    }
                    disabled={disabled}
                    maxLength={1000}
                    rows={3}
                    placeholder="Describa las responsabilidades y logros clave en este puesto..."
                  />
                </div>

                <AdjuntoUploader
                  label="Certificado Laboral o Mecanizado IESS (PDF)"
                  maxMB={2}
                  valor={exp.documentoPdf}
                  onArchivoSeleccionado={(adjunto) =>
                    actualizarExperiencia(index, "documentoPdf", adjunto)
                  }
                  onEliminar={() =>
                    actualizarExperiencia(index, "documentoPdf", null)
                  }
                  disabled={disabled}
                  descripcion="Certificado laboral membretado con fechas o mecanizado del IESS (máx. 2 MB)."
                />
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
