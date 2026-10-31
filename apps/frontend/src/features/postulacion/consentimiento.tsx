import { CheckSquareIcon, ShieldCheckIcon } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Label } from "@/components/ui/label"
import {
  PUNTOS_CONSENTIMIENTO_EF12,
  type ConsentimientoEF12Data,
} from "@/features/formulario/constantes"

export interface ConsentimientoEF12Props {
  datos: ConsentimientoEF12Data
  onChange: (nuevos: ConsentimientoEF12Data) => void
  disabled?: boolean
}

/**
 * Componente de Consentimiento y Declaración Juramentada EF-12 (punto 1.10.1).
 * Presenta los cinco literales obligatorios del concurso BIESS/IESS.
 */
export function ConsentimientoEF12({
  datos,
  onChange,
  disabled = false,
}: ConsentimientoEF12Props) {
  const puntos = PUNTOS_CONSENTIMIENTO_EF12

  const todosAceptados =
    datos.aceptacionTratamientoDatos &&
    datos.declaracionVeracidad &&
    datos.aceptacionNormasConcurso &&
    datos.autorizacionPublicacionNombre &&
    datos.constanciaEntregaFisica

  function togglePunto(id: keyof ConsentimientoEF12Data, valor: boolean) {
    onChange({
      ...datos,
      [id]: valor,
    })
  }

  function marcarTodos(marcar: boolean) {
    onChange({
      aceptacionTratamientoDatos: marcar,
      declaracionVeracidad: marcar,
      aceptacionNormasConcurso: marcar,
      autorizacionPublicacionNombre: marcar,
      constanciaEntregaFisica: marcar,
    })
  }

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-col gap-2 rounded-lg border border-primary/20 bg-primary/5 p-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-3">
          <ShieldCheckIcon
            className="size-6 text-primary shrink-0"
            aria-hidden="true"
          />
          <div>
            <h3 className="text-base font-semibold text-foreground">
              Declaración y Consentimiento Legal (EF-12 1.10.1)
            </h3>
            <p className="text-xs text-muted-foreground">
              Es requisito legal aceptar explícitamente los 5 puntos antes de
              enviar la postulación definitiva.
            </p>
          </div>
        </div>

        {!disabled && (
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => marcarTodos(!todosAceptados)}
            className="shrink-0 self-start text-xs sm:self-center"
          >
            <CheckSquareIcon className="size-3.5 mr-1.5" aria-hidden="true" />
            <span>
              {todosAceptados ? "Desmarcar todos" : "Aceptar los 5 puntos"}
            </span>
          </Button>
        )}
      </div>

      <div className="flex flex-col gap-3">
        {puntos.map((punto) => {
          const aceptado = Boolean(datos[punto.id])
          return (
            <div
              key={punto.id}
              className={`flex items-start gap-3 rounded-lg border p-4 transition-colors ${
                aceptado
                  ? "border-primary/40 bg-card shadow-2xs"
                  : "border-border bg-card/50"
              }`}
            >
              <Checkbox
                id={`consentimiento-${punto.id}`}
                checked={aceptado}
                onCheckedChange={(checked) =>
                  togglePunto(punto.id, checked === true)
                }
                disabled={disabled}
                className="mt-1"
              />
              <div className="flex flex-col gap-1">
                <Label
                  htmlFor={`consentimiento-${punto.id}`}
                  className="cursor-pointer text-sm font-medium text-foreground leading-snug"
                >
                  <span className="font-semibold text-primary">
                    Punto {punto.numero}:{" "}
                  </span>
                  {punto.titulo}
                </Label>
                <p className="text-xs text-muted-foreground leading-relaxed">
                  {punto.texto}
                </p>
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
