import { useState } from "react"
import {
  AlertTriangleIcon,
  CheckSquareIcon,
  ExternalLinkIcon,
  FileTextIcon,
  ShieldCheckIcon,
} from "lucide-react"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Label } from "@/components/ui/label"
import {
  PUNTOS_CONSENTIMIENTO_EF12,
  type ClavePuntoConsentimiento,
  type ConsentimientoEF12Data,
} from "./constantes"
import { ModalPoliticaPrivacidad } from "@/features/privacidad/modal-politica-privacidad"
import type { PoliticaPrivacidadDto } from "@/lib/clientes/privacidad"

export interface ConsentimientoEF12Props {
  datos: ConsentimientoEF12Data
  onChange: (nuevos: ConsentimientoEF12Data) => void
  politica?: PoliticaPrivacidadDto | null
  disabled?: boolean
}

/**
 * Componente de Consentimiento y Declaración Juramentada EF-12 (punto 1.10.1 / FT28).
 * Presenta los cinco literales obligatorios del concurso BIESS/IESS y vincula
 * la política institucional de privacidad aprobada (T28).
 */
export function ConsentimientoEF12({
  datos,
  onChange,
  politica,
  disabled = false,
}: ConsentimientoEF12Props) {
  const puntos = PUNTOS_CONSENTIMIENTO_EF12
  const [modalAbierto, setModalAbierto] = useState(false)

  const todosAceptados =
    datos.aceptacionTratamientoDatos &&
    datos.declaracionVeracidad &&
    datos.aceptacionNormasConcurso &&
    datos.autorizacionPublicacionNombre &&
    datos.constanciaEntregaFisica

  function togglePunto(id: ClavePuntoConsentimiento, valor: boolean) {
    onChange({
      ...datos,
      [id]: valor,
    })
  }

  function marcarTodos(marcar: boolean) {
    onChange({
      ...datos,
      aceptacionTratamientoDatos: marcar,
      declaracionVeracidad: marcar,
      aceptacionNormasConcurso: marcar,
      autorizacionPublicacionNombre: marcar,
      constanciaEntregaFisica: marcar,
    })
  }

  return (
    <div className="flex flex-col gap-5">
      {/* Información de la Política Institucional Vigente (T28 / FT28) */}
      {politica && (
        <div className="rounded-lg border bg-muted/30 p-4 space-y-3">
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex items-center gap-2">
              <FileTextIcon className="size-4 text-primary shrink-0" />
              <span className="text-xs font-semibold text-foreground">
                Política Institucional de Protección de Datos
              </span>
              <Badge variant="outline" className="text-[11px] font-mono">
                v{politica.version}
              </Badge>
              {politica.restringido && (
                <Badge variant="destructive" className="text-[11px]">
                  Restringida
                </Badge>
              )}
            </div>

            <Button
              type="button"
              variant="outline"
              size="sm"
              className="text-xs h-7 gap-1.5 self-start sm:self-auto"
              onClick={() => setModalAbierto(true)}
            >
              <span>Ver texto completo y plazos</span>
              <ExternalLinkIcon className="size-3" />
            </Button>
          </div>

          <div className="text-xs text-muted-foreground flex flex-wrap items-center gap-x-4 gap-y-1">
            <span>
              <strong>Aprobación:</strong>{" "}
              {politica.definicion.referenciaAprobacion || "Resolución Oficial"}
            </span>
            <span className="font-mono text-[11px] truncate max-w-xs">
              Hash: {politica.hash.substring(0, 16)}…
            </span>
          </div>

          {politica.restringido && (
            <Alert variant="destructive" className="py-2">
              <AlertTriangleIcon className="size-4" />
              <AlertTitle className="text-xs font-semibold">
                Sello temporalmente deshabilitado
              </AlertTitle>
              <AlertDescription className="text-xs">
                La política institucional de esta convocatoria se encuentra bajo
                restricción cautelar. No es posible enviar postulaciones hasta
                su restitución.
              </AlertDescription>
            </Alert>
          )}
        </div>
      )}

      {/* Cabecera del bloque de los 5 puntos de consentimiento */}
      <div className="flex flex-col gap-2 rounded-lg border border-primary/20 bg-primary/5 p-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-3">
          <ShieldCheckIcon
            className="size-6 text-primary shrink-0"
            aria-hidden="true"
          />
          <div>
            <h3 className="text-base font-semibold text-foreground">
              Declaración y Consentimiento Legal
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

      {/* Listado estricto de los 5 puntos del concurso */}
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

      <ModalPoliticaPrivacidad
        politica={politica}
        abierto={modalAbierto}
        onCerrar={() => setModalAbierto(false)}
      />
    </div>
  )
}
