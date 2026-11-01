import { useState } from "react"
import {
  AlertTriangleIcon,
  CalendarIcon,
  InfoIcon,
  ScaleIcon,
  ShieldCheckIcon,
} from "lucide-react"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import type { PoliticaPrivacidadDto } from "@/lib/clientes/privacidad"

export interface ModalPoliticaPrivacidadProps {
  politica?: PoliticaPrivacidadDto | null
  abierto: boolean
  onCerrar: () => void
}

/**
 * Modal para lectura y consulta de la política institucional vigente de privacidad (FT28 / T28).
 * Presenta versión, fecha, hash, información de tratamiento (LOPDP Art. 7) y plazos de conservación.
 */
export function ModalPoliticaPrivacidad({
  politica,
  abierto,
  onCerrar,
}: ModalPoliticaPrivacidadProps) {
  const [seccionActiva, setSeccionActiva] = useState<
    "texto" | "tratamiento" | "conservacion"
  >("texto")

  if (!politica) return null

  const { definicion, version, hash, publicadaEn, restringido } = politica
  const fechaPublicacion = publicadaEn
    ? new Date(publicadaEn).toLocaleDateString("es-EC", {
        year: "numeric",
        month: "long",
        day: "numeric",
      })
    : "Sin fecha registrada"

  return (
    <Dialog open={abierto} onOpenChange={(open) => !open && onCerrar()}>
      <DialogContent className="max-w-2xl sm:max-h-[85vh] flex flex-col p-0">
        <DialogHeader className="p-6 pb-4 border-b">
          <div className="flex items-center gap-2 mb-1">
            <Badge variant="outline" className="font-mono text-xs">
              Versión {version}
            </Badge>
            {restringido ? (
              <Badge variant="destructive" className="text-xs">
                Restricción Cautelar Activa
              </Badge>
            ) : (
              <Badge
                variant="secondary"
                className="bg-emerald-500/10 text-emerald-700 dark:text-emerald-300 text-xs"
              >
                Vigente y Autorizada
              </Badge>
            )}
          </div>
          <DialogTitle className="text-lg font-bold flex items-center gap-2">
            <ShieldCheckIcon className="size-5 text-primary shrink-0" />
            <span>
              Política Institucional de Protección de Datos Personales
            </span>
          </DialogTitle>
          <DialogDescription className="text-xs text-muted-foreground flex flex-wrap items-center gap-x-4 gap-y-1 pt-1">
            <span className="flex items-center gap-1">
              <CalendarIcon className="size-3.5" />
              Publicada: {fechaPublicacion}
            </span>
            <span className="flex items-center gap-1">
              <ScaleIcon className="size-3.5" />
              Ref:{" "}
              {definicion.referenciaAprobacion || "Aprobación institucional"}
            </span>
          </DialogDescription>
        </DialogHeader>

        {restringido && (
          <div className="px-6 pt-4">
            <Alert variant="destructive">
              <AlertTriangleIcon className="size-4" />
              <AlertTitle>Medida Cautelar de Privacidad</AlertTitle>
              <AlertDescription className="text-xs">
                Esta política se encuentra temporalmente restringida. Toda
                salida o sello de expedientes queda bloqueado hasta su
                restitución formal.
              </AlertDescription>
            </Alert>
          </div>
        )}

        {/* Pestañas de navegación de contenido */}
        <div className="flex border-b px-6 bg-muted/20 text-xs font-medium">
          <button
            type="button"
            className={`py-2.5 px-3 border-b-2 transition-colors ${
              seccionActiva === "texto"
                ? "border-primary text-primary font-semibold"
                : "border-transparent text-muted-foreground hover:text-foreground"
            }`}
            onClick={() => setSeccionActiva("texto")}
          >
            Texto Institucional
          </button>
          <button
            type="button"
            className={`py-2.5 px-3 border-b-2 transition-colors ${
              seccionActiva === "tratamiento"
                ? "border-primary text-primary font-semibold"
                : "border-transparent text-muted-foreground hover:text-foreground"
            }`}
            onClick={() => setSeccionActiva("tratamiento")}
          >
            Tratamiento de Datos (LOPDP)
          </button>
          <button
            type="button"
            className={`py-2.5 px-3 border-b-2 transition-colors ${
              seccionActiva === "conservacion"
                ? "border-primary text-primary font-semibold"
                : "border-transparent text-muted-foreground hover:text-foreground"
            }`}
            onClick={() => setSeccionActiva("conservacion")}
          >
            Plazos y Conservación
          </button>
        </div>

        {/* Contenido scrolleable */}
        <div className="p-6 overflow-y-auto flex-1 space-y-4 text-xs leading-relaxed text-foreground">
          {seccionActiva === "texto" && (
            <div className="space-y-3">
              <div className="rounded-lg border bg-card p-4 space-y-2 whitespace-pre-wrap font-sans text-sm text-foreground">
                {definicion.texto}
              </div>
              <div className="p-3 rounded bg-muted/40 text-[11px] text-muted-foreground flex items-center justify-between">
                <span>Identificador criptográfico de versión:</span>
                <span className="font-mono">{hash}</span>
              </div>
            </div>
          )}

          {seccionActiva === "tratamiento" && (
            <div className="space-y-4">
              <div className="rounded-lg border bg-card p-4 space-y-2 whitespace-pre-wrap font-sans text-sm text-foreground">
                {definicion.informacionTratamiento}
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-2">
                <div className="border rounded p-3 bg-muted/20">
                  <span className="font-semibold block mb-1 text-foreground">
                    Publicación de nombres
                  </span>
                  <p className="text-muted-foreground">
                    {definicion.permiteNombresPublicos
                      ? `Autorizada en listas oficiales por ${definicion.diasPublicacionNombres} días tras el concurso.`
                      : "No autorizada para difusión abierta."}
                  </p>
                </div>
                <div className="border rounded p-3 bg-muted/20">
                  <span className="font-semibold block mb-1 text-foreground">
                    Comunicaciones por correo
                  </span>
                  <p className="text-muted-foreground">
                    {definicion.permiteCorreo
                      ? `Autorizada exclusivamente para actos del concurso durante ${definicion.diasCorreo} días.`
                      : "No autorizada."}
                  </p>
                </div>
              </div>
            </div>
          )}

          {seccionActiva === "conservacion" && (
            <div className="space-y-3">
              <p className="text-muted-foreground">
                Conforme al principio de conservación de evidencias de la LOPDP
                y la resolución institucional, los plazos técnicos aplicables
                son:
              </p>
              <div className="grid grid-cols-2 sm:grid-cols-3 gap-2 text-center">
                <div className="border rounded p-2.5 bg-muted/10">
                  <span className="text-xs text-muted-foreground block">
                    Acceso a Expediente
                  </span>
                  <span className="font-mono text-sm font-bold text-foreground">
                    {definicion.diasAcceso} días
                  </span>
                </div>
                <div className="border rounded p-2.5 bg-muted/10">
                  <span className="text-xs text-muted-foreground block">
                    Respaldos Documentales
                  </span>
                  <span className="font-mono text-sm font-bold text-foreground">
                    {definicion.diasDocumentos} días
                  </span>
                </div>
                <div className="border rounded p-2.5 bg-muted/10">
                  <span className="text-xs text-muted-foreground block">
                    Reportes Oficiales
                  </span>
                  <span className="font-mono text-sm font-bold text-foreground">
                    {definicion.diasReportes} días
                  </span>
                </div>
              </div>

              <div className="rounded-lg border p-3 bg-muted/20 space-y-1">
                <span className="font-semibold text-foreground flex items-center gap-1.5">
                  <InfoIcon className="size-3.5 text-primary" />
                  Disposición de Archivos Históricos
                </span>
                <p className="text-muted-foreground text-[11px]">
                  Régimen:{" "}
                  <strong className="text-foreground">
                    {definicion.disposicionHistoricos}
                  </strong>
                  . La revisión de retención se programará a los{" "}
                  <strong>{definicion.diasRevisionConservacion} días</strong>{" "}
                  del sello, sin purga automática sin auditoría.
                </p>
              </div>
            </div>
          )}
        </div>

        <DialogFooter className="p-4 border-t bg-muted/10 flex items-center justify-between sm:justify-between">
          <span className="text-[11px] text-muted-foreground hidden sm:inline">
            EF-1 Precondiciones · Ley Orgánica de Protección de Datos Personales
          </span>
          <Button type="button" size="sm" onClick={onCerrar}>
            Cerrar
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
