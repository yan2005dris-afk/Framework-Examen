import {
  AlertCircleIcon,
  BriefcaseIcon,
  CheckCircle2Icon,
  InfoIcon,
} from "lucide-react"

import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import type { ElegibilidadEvidencia } from "@/lib/clientes/postulaciones"
import { AdjuntoUploader, type AdjuntoInfo } from "./adjunto-uploader"

export interface Seccion4Data {
  sector: "afiliados" | "jubilados" | "empleadores" | ""
  verificacionIessCumplida?: boolean
  cuentaConRespaldoOrganizacion: "Sí" | "No" | ""
  nombreOrganizacion?: string
  documentoRespaldoPdf?: AdjuntoInfo | null
}

export interface Seccion4CargoProps {
  datos: Seccion4Data
  onChange: (nuevosDatos: Seccion4Data) => void
  disabled?: boolean
  /** Evidencia de elegibilidad devuelta por el servidor (FT17). */
  elegibilidad?: ElegibilidadEvidencia | null
}

const MENSAJES_SECTOR = {
  afiliados:
    "Postulación para Representante de los Afiliados al Directorio del BIESS. Se requiere acreditar la condición de afiliado activo al IESS.",
  jubilados:
    "Postulación para Representante de los Jubilados al Directorio del BIESS. Se requiere acreditar la condición de jubilado del IESS.",
  empleadores:
    "Postulación para Representante de los Empleadores al Directorio del BIESS. Se requiere acreditar la condición de empleador al día en sus obligaciones.",
}

export function Seccion4Cargo({
  datos,
  onChange,
  disabled = false,
  elegibilidad,
}: Seccion4CargoProps) {
  function actualizarSector(valor: "afiliados" | "jubilados" | "empleadores") {
    onChange({
      ...datos,
      sector: valor,
      // La verificación real es determinada por el servidor; el cliente no la fuerza.
      verificacionIessCumplida: elegibilidad?.elegible ?? true,
    })
  }

  function actualizarRespaldo(valor: "Sí" | "No") {
    onChange({
      ...datos,
      cuentaConRespaldoOrganizacion: valor,
      nombreOrganizacion: valor === "No" ? "" : datos.nombreOrganizacion,
      documentoRespaldoPdf: valor === "No" ? null : datos.documentoRespaldoPdf,
    })
  }

  // Estado de no elegibilidad conforme a mockup 06-04b
  const esNoElegible = elegibilidad && !elegibilidad.elegible

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center gap-2 border-b pb-3">
        <BriefcaseIcon className="size-5 text-primary" aria-hidden="true" />
        <div>
          <h2 className="text-lg font-semibold">
            4. Selección del Cargo al que Postula
          </h2>
          <p className="text-xs text-muted-foreground">
            Definición del sector de representación (Afiliados, Jubilados o
            Empleadores) y respaldo gremial.
          </p>
        </div>
      </div>

      {/* Banner de NO ELEGIBLE (Mockup 06-04b) */}
      {esNoElegible && (
        <div
          role="alert"
          aria-live="assertive"
          className="rounded-lg border border-destructive/50 bg-destructive/10 p-4 space-y-3"
        >
          <div className="flex items-start gap-2.5">
            <AlertCircleIcon className="size-5 text-destructive shrink-0 mt-0.5" />
            <div>
              <h3 className="font-semibold text-destructive">
                {elegibilidad.mensaje ||
                  "Postulante No Elegible para el Sector"}
              </h3>
              <p className="text-xs text-destructive/90 mt-1">
                Su postulación para el sector seleccionado no puede continuar
                por no cumplir los requisitos legales de elegibilidad.
              </p>
            </div>
          </div>

          <div className="rounded border bg-background/80 p-3 text-xs space-y-1.5 font-sans">
            <div className="flex justify-between">
              <span className="text-muted-foreground">
                Sector seleccionado:
              </span>
              <span className="font-medium capitalize">
                {elegibilidad.sector || datos.sector}
              </span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">
                Resultado de interoperabilidad:
              </span>
              <span className="font-medium">{elegibilidad.mensaje}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">
                Origen de verificación:
              </span>
              <span className="font-mono text-muted-foreground">
                {elegibilidad.origen === "simulacion_controlada_servidor"
                  ? "simulación controlada del servidor"
                  : elegibilidad.origen}
              </span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">
                Estado de la postulación:
              </span>
              <span className="font-semibold text-destructive">
                Cerrada automáticamente
              </span>
            </div>
          </div>

          <div className="rounded bg-muted/60 p-3 text-xs text-muted-foreground">
            <p className="font-medium text-foreground">¿Qué puede hacer?</p>
            <p className="mt-0.5">
              Si considera que el resultado es incorrecto, puede presentar un
              reclamo ante la Comisión Especial dentro del plazo de la
              convocatoria.
            </p>
          </div>
        </div>
      )}

      {/* Banner de ELEGIBILIDAD CONFIRMADA (Mockup 06-04) */}
      {elegibilidad && elegibilidad.elegible && (
        <div className="flex items-start gap-2.5 rounded-lg border border-emerald-500/30 bg-emerald-500/10 p-3.5 text-xs text-emerald-800 dark:text-emerald-300">
          <CheckCircle2Icon className="size-4 shrink-0 text-emerald-600 dark:text-emerald-400 mt-0.5" />
          <div className="space-y-0.5">
            <p className="font-semibold">
              {elegibilidad.mensaje || "Elegibilidad confirmada para el sector"}
            </p>
            <p className="text-xs opacity-90">
              Origen:{" "}
              {elegibilidad.origen === "simulacion_controlada_servidor"
                ? "Simulación controlada del servidor"
                : elegibilidad.origen}
              {elegibilidad.verificadoEn &&
                ` · Verificado el ${new Date(elegibilidad.verificadoEn).toLocaleDateString()}`}
            </p>
          </div>
        </div>
      )}

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="sec4-sector">Sector de Representación *</Label>
          <Select
            value={datos.sector}
            onValueChange={(val) =>
              actualizarSector(val as "afiliados" | "jubilados" | "empleadores")
            }
            disabled={disabled || Boolean(esNoElegible)}
          >
            <SelectTrigger id="sec4-sector">
              <SelectValue placeholder="Seleccione el sector" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="afiliados">Afiliados al IESS</SelectItem>
              <SelectItem value="jubilados">Jubilados del IESS</SelectItem>
              <SelectItem value="empleadores">Empleadores</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="sec4-tiene-respaldo">
            ¿Cuenta con respaldo de alguna organización o gremio? *
          </Label>
          <Select
            value={datos.cuentaConRespaldoOrganizacion}
            onValueChange={(val) => actualizarRespaldo(val as "Sí" | "No")}
            disabled={disabled || Boolean(esNoElegible)}
          >
            <SelectTrigger id="sec4-tiene-respaldo">
              <SelectValue placeholder="Seleccione Sí o No" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="Sí">Sí</SelectItem>
              <SelectItem value="No">No</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      {datos.sector && !esNoElegible && (
        <div className="flex items-start gap-2.5 rounded-lg border bg-primary/5 p-3.5 text-xs text-foreground/90 sm:text-sm">
          <InfoIcon
            className="size-4 shrink-0 text-primary mt-0.5"
            aria-hidden="true"
          />
          <p>{MENSAJES_SECTOR[datos.sector]}</p>
        </div>
      )}

      {datos.cuentaConRespaldoOrganizacion === "Sí" && !esNoElegible && (
        <div className="flex flex-col gap-4 rounded-lg border bg-muted/20 p-4">
          <h3 className="text-sm font-semibold text-foreground">
            Datos de la Organización de Respaldo
          </h3>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="sec4-nombre-organizacion">
              Nombre de la Organización, Gremio o Asociación *
            </Label>
            <Input
              id="sec4-nombre-organizacion"
              value={datos.nombreOrganizacion || ""}
              onChange={(e) =>
                onChange({
                  ...datos,
                  nombreOrganizacion: e.target.value,
                })
              }
              disabled={disabled}
              placeholder="Ej. Federación Nacional de Cámaras, Colegio Profesional..."
            />
          </div>

          <AdjuntoUploader
            label="Documento de Respaldo de la Organización (PDF) *"
            maxMB={4}
            valor={datos.documentoRespaldoPdf}
            onArchivoSeleccionado={(adjunto) =>
              onChange({
                ...datos,
                documentoRespaldoPdf: adjunto,
              })
            }
            onEliminar={() =>
              onChange({
                ...datos,
                documentoRespaldoPdf: null,
              })
            }
            disabled={disabled}
            descripcion="Certificado, oficio o aval expedido por la directiva de la organización (límite 4 MB)."
          />
        </div>
      )}
    </div>
  )
}
