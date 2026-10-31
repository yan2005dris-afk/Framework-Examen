import { AlertCircleIcon, ShieldAlertIcon } from "lucide-react"

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
import { PREGUNTAS_INHABILIDADES } from "./constantes"

export interface DetalleRemocionQ7 {
  institucion: string
  cargo: string
  fechaFin: string
  motivo: string
}

export interface Seccion9Data {
  [key: string]: string | DetalleRemocionQ7 | undefined
  q1: string
  q2: string
  q3: string
  q4: string
  q5: string
  q6: string
  q7: string
  q7Detalles?: DetalleRemocionQ7
  q8: string
  q9: string
  q10: string
  q11: string
  q12: string
  q13: string
}

export interface Seccion9InhabilidadesProps {
  datos: Seccion9Data
  onChange: (nuevosDatos: Seccion9Data) => void
  disabled?: boolean
}

export function Seccion9Inhabilidades({
  datos,
  onChange,
  disabled = false,
}: Seccion9InhabilidadesProps) {
  function actualizarRespuesta(id: string, valor: "Sí" | "No") {
    const nuevos: Seccion9Data = {
      ...datos,
      [id]: valor,
    }
    if (id === "q7") {
      if (valor === "No") {
        delete nuevos.q7Detalles
      } else if (!nuevos.q7Detalles) {
        nuevos.q7Detalles = {
          institucion: "",
          cargo: "",
          fechaFin: "",
          motivo: "",
        }
      }
    }
    onChange(nuevos)
  }

  function actualizarDetalleQ7(campo: keyof DetalleRemocionQ7, valor: string) {
    const detallesActuales: DetalleRemocionQ7 = datos.q7Detalles || {
      institucion: "",
      cargo: "",
      fechaFin: "",
      motivo: "",
    }
    onChange({
      ...datos,
      q7Detalles: {
        ...detallesActuales,
        [campo]: valor,
      },
    })
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center gap-2 border-b pb-3">
        <ShieldAlertIcon className="size-5 text-primary" aria-hidden="true" />
        <div>
          <h2 className="text-lg font-semibold">
            9. Inhabilidades y Prohibiciones Legales
          </h2>
          <p className="text-xs text-muted-foreground">
            Declaración juramentada obligatoria sobre las 13 causales legales de
            inhabilidad del concurso.
          </p>
        </div>
      </div>

      <div className="flex flex-col gap-4">
        {PREGUNTAS_INHABILIDADES.map((pregunta) => {
          const valorActual =
            ((datos as Record<string, unknown>)[pregunta.id] as string) || ""
          const esQ7 = pregunta.id === "q7"
          const mostrarDetalleQ7 = esQ7 && valorActual === "Sí"

          return (
            <div
              key={pregunta.id}
              className="flex flex-col gap-3 rounded-lg border bg-card p-4 shadow-2xs"
            >
              <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <div className="flex-1">
                  <span className="font-mono text-xs font-semibold text-primary">
                    {pregunta.numero}
                  </span>
                  <p className="mt-1 text-sm text-foreground">
                    {pregunta.texto}
                  </p>
                </div>

                <div className="w-full shrink-0 sm:w-36">
                  <Select
                    value={valorActual}
                    onValueChange={(val) =>
                      actualizarRespuesta(pregunta.id, val as "Sí" | "No")
                    }
                    disabled={disabled}
                  >
                    <SelectTrigger className="w-full">
                      <SelectValue placeholder="Seleccione" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="Sí">Sí</SelectItem>
                      <SelectItem value="No">No</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>

              {mostrarDetalleQ7 && (
                <div className="mt-3 flex flex-col gap-3 rounded-md border border-amber-500/30 bg-amber-500/5 p-4 text-xs">
                  <div className="flex items-center gap-1.5 font-semibold text-amber-800 dark:text-amber-400">
                    <AlertCircleIcon className="size-4" aria-hidden="true" />
                    <span>Detalle obligatorio de la remoción (1.9.7)</span>
                  </div>

                  <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                    <div className="flex flex-col gap-1">
                      <Label className="text-xs">
                        Institución u Organismo de Control *
                      </Label>
                      <Input
                        value={datos.q7Detalles?.institucion || ""}
                        onChange={(e) =>
                          actualizarDetalleQ7("institucion", e.target.value)
                        }
                        disabled={disabled}
                        placeholder="Nombre de la institución"
                      />
                    </div>

                    <div className="flex flex-col gap-1">
                      <Label className="text-xs">Cargo que Desempeñaba *</Label>
                      <Input
                        value={datos.q7Detalles?.cargo || ""}
                        onChange={(e) =>
                          actualizarDetalleQ7("cargo", e.target.value)
                        }
                        disabled={disabled}
                        placeholder="Cargo ejercido"
                      />
                    </div>

                    <div className="flex flex-col gap-1">
                      <Label className="text-xs">
                        Fecha de Salida / Remoción *
                      </Label>
                      <Input
                        type="date"
                        value={datos.q7Detalles?.fechaFin || ""}
                        onChange={(e) =>
                          actualizarDetalleQ7("fechaFin", e.target.value)
                        }
                        disabled={disabled}
                      />
                    </div>

                    <div className="flex flex-col gap-1 sm:col-span-2">
                      <Label className="text-xs">Motivo de la Remoción *</Label>
                      <Textarea
                        value={datos.q7Detalles?.motivo || ""}
                        onChange={(e) =>
                          actualizarDetalleQ7("motivo", e.target.value)
                        }
                        disabled={disabled}
                        rows={2}
                        placeholder="Explique las circunstancias o resolución del organismo de control"
                      />
                    </div>
                  </div>
                </div>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}
