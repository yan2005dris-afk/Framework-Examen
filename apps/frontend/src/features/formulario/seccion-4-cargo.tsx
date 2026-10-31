import { BriefcaseIcon, InfoIcon } from "lucide-react"

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
}: Seccion4CargoProps) {
  function actualizarSector(valor: "afiliados" | "jubilados" | "empleadores") {
    onChange({
      ...datos,
      sector: valor,
      verificacionIessCumplida: true,
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

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="sec4-sector">Sector de Representación *</Label>
          <Select
            value={datos.sector}
            onValueChange={(val) =>
              actualizarSector(val as "afiliados" | "jubilados" | "empleadores")
            }
            disabled={disabled}
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
            disabled={disabled}
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

      {datos.sector && (
        <div className="flex items-start gap-2.5 rounded-lg border bg-primary/5 p-3.5 text-xs text-foreground/90 sm:text-sm">
          <InfoIcon
            className="size-4 shrink-0 text-primary mt-0.5"
            aria-hidden="true"
          />
          <p>{MENSAJES_SECTOR[datos.sector]}</p>
        </div>
      )}

      {datos.cuentaConRespaldoOrganizacion === "Sí" && (
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
