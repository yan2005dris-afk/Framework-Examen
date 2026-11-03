import { PlusIcon, Trash2Icon, UsersIcon } from "lucide-react"

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
import type { Familiar } from "@/types/concurso"

export interface Seccion3Data {
  tieneFamiliaresIessBiess: "Sí" | "No" | ""
  familiares: Familiar[]
}

export interface Seccion3AdicionalProps {
  datos: Seccion3Data
  onChange: (nuevosDatos: Seccion3Data) => void
  disabled?: boolean
}

const PARENTESCOS_CATALOGO = [
  "conviviente",
  "cónyuge",
  "cuñado/a",
  "hermano/a",
  "hijo político",
  "hijo/a",
  "madre",
  "nieto/a",
  "nuera",
  "padre",
  "primo hermano/a",
  "sobrino/a",
  "suegro/a",
  "tío/a",
  "yerno",
]

export function Seccion3Adicional({
  datos,
  onChange,
  disabled = false,
}: Seccion3AdicionalProps) {
  function manejarCambioTieneFamiliares(valor: "Sí" | "No") {
    onChange({
      ...datos,
      tieneFamiliaresIessBiess: valor,
      familiares: valor === "No" ? [] : datos.familiares,
    })
  }

  function agregarFamiliar() {
    const nuevo: Familiar = {
      id: `fam-${Date.now()}-${Math.random().toString(36).substring(2, 7)}`,
      nombres: "",
      parentesco: "hermano/a",
      institucion: "IESS",
      area: "",
      cargo: "",
    }
    onChange({
      ...datos,
      familiares: [...datos.familiares, nuevo],
    })
  }

  function actualizarFamiliar(
    index: number,
    campo: keyof Familiar,
    valor: string,
  ) {
    const lista = [...datos.familiares]
    lista[index] = {
      ...lista[index],
      [campo]: valor,
    }
    onChange({
      ...datos,
      familiares: lista,
    })
  }

  function eliminarFamiliar(index: number) {
    const lista = datos.familiares.filter((_, i) => i !== index)
    onChange({
      ...datos,
      familiares: lista,
    })
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center gap-2 border-b pb-3">
        <UsersIcon className="size-5 text-primary" aria-hidden="true" />
        <div>
          <h2 className="text-lg font-semibold">3. Información Adicional</h2>
          <p className="text-xs text-muted-foreground">
            Declaración de parientes hasta el cuarto grado de consanguinidad o
            segundo de afinidad que laboren en el IESS o BIESS.
          </p>
        </div>
      </div>

      <div className="flex flex-col gap-2 rounded-lg border bg-muted/20 p-4">
        <Label htmlFor="sec3-tiene-familiares" className="text-sm font-medium">
          ¿Tiene familiares dentro del 4to grado de consanguinidad o 2do de
          afinidad laborando en el IESS o BIESS? *
        </Label>
        <Select
          value={datos.tieneFamiliaresIessBiess}
          onValueChange={(val) =>
            manejarCambioTieneFamiliares(val as "Sí" | "No")
          }
          disabled={disabled}
        >
          <SelectTrigger id="sec3-tiene-familiares" className="w-full sm:w-60">
            <SelectValue placeholder="Seleccione Sí o No" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="Sí">Sí</SelectItem>
            <SelectItem value="No">No</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {datos.tieneFamiliaresIessBiess === "Sí" && (
        <div className="flex flex-col gap-4">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-semibold text-foreground">
              Detalle de Familiares ({datos.familiares.length})
            </h3>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={agregarFamiliar}
              disabled={disabled}
              className="flex items-center gap-1.5"
            >
              <PlusIcon className="size-4" aria-hidden="true" />
              <span>Agregar familiar</span>
            </Button>
          </div>

          {datos.familiares.length === 0 ? (
            <p className="rounded-lg border border-dashed p-6 text-center text-xs text-muted-foreground">
              Ha indicado que «Sí» tiene familiares. Debe registrar al menos un
              familiar haciendo clic en «Agregar familiar».
            </p>
          ) : (
            <div className="flex flex-col gap-4">
              {datos.familiares.map((familiar, index) => (
                <div
                  key={familiar.id}
                  className="relative grid grid-cols-1 gap-3 rounded-lg border bg-card p-4 shadow-2xs sm:grid-cols-2 lg:grid-cols-3"
                >
                  <div className="flex flex-col gap-1.5 sm:col-span-2 lg:col-span-1">
                    <Label className="text-xs">Nombres y Apellidos *</Label>
                    <Input
                      value={familiar.nombres}
                      onChange={(e) =>
                        actualizarFamiliar(index, "nombres", e.target.value)
                      }
                      disabled={disabled}
                      placeholder="Nombre del familiar"
                    />
                  </div>

                  <div className="flex flex-col gap-1.5">
                    <Label className="text-xs">Parentesco *</Label>
                    <Select
                      value={familiar.parentesco}
                      onValueChange={(val) =>
                        actualizarFamiliar(index, "parentesco", val)
                      }
                      disabled={disabled}
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent className="max-h-56">
                        {PARENTESCOS_CATALOGO.map((p) => (
                          <SelectItem key={p} value={p}>
                            {p}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>

                  <div className="flex flex-col gap-1.5">
                    <Label className="text-xs">Institución *</Label>
                    <Select
                      value={familiar.institucion}
                      onValueChange={(val) =>
                        actualizarFamiliar(
                          index,
                          "institucion",
                          val as "IESS" | "BIESS",
                        )
                      }
                      disabled={disabled}
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="IESS">IESS</SelectItem>
                        <SelectItem value="BIESS">BIESS</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>

                  <div className="flex flex-col gap-1.5">
                    <Label className="text-xs">Área / Departamento *</Label>
                    <Input
                      value={familiar.area}
                      onChange={(e) =>
                        actualizarFamiliar(index, "area", e.target.value)
                      }
                      disabled={disabled}
                      placeholder="Dirección Financiera, Riesgos..."
                    />
                  </div>

                  <div className="flex flex-col gap-1.5">
                    <Label className="text-xs">Cargo *</Label>
                    <Input
                      value={familiar.cargo}
                      onChange={(e) =>
                        actualizarFamiliar(index, "cargo", e.target.value)
                      }
                      disabled={disabled}
                      placeholder="Analista, Especialista..."
                    />
                  </div>

                  <div className="flex items-end justify-end">
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      onClick={() => eliminarFamiliar(index)}
                      disabled={disabled}
                      className="text-destructive hover:bg-destructive/10"
                    >
                      <Trash2Icon className="size-4" aria-hidden="true" />
                      <span className="ml-1 text-xs">Eliminar</span>
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
