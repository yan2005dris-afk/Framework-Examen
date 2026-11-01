import { AlertTriangleIcon, UserIcon } from "lucide-react"
import { useEffect, useRef } from "react"

import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { useSesion } from "@/features/auth/sesion-context"
import { validarCedulaPareja } from "./validacion"

export interface Seccion1Data {
  nombres: string
  apellidos: string
  cedula: string
  genero: string
  fechaNacimiento: string
  nacionalidad: string
  estadoCivil: string
  nombreConyuge?: string
  cedulaConyuge?: string
  verificadoRegistroCivil?: boolean
}

export interface Seccion1PersonalProps {
  datos: Seccion1Data
  onChange: (nuevosDatos: Seccion1Data) => void
  disabled?: boolean
  onBloqueoNacionalidad?: (bloqueado: boolean) => void
}

export function Seccion1Personal({
  datos,
  onChange,
  disabled = false,
  onBloqueoNacionalidad,
}: Seccion1PersonalProps) {
  const { postulante } = useSesion()

  const haPrecargadoRef = useRef(false)

  // Precarga inicial desde la sesión si los campos vienen vacíos
  useEffect(() => {
    if (haPrecargadoRef.current) return
    if (postulante && (!datos.nombres || !datos.cedula)) {
      haPrecargadoRef.current = true
      onChange({
        ...datos,
        nombres: datos.nombres || postulante.nombres || "",
        apellidos: datos.apellidos || postulante.apellidos || "",
        cedula: datos.cedula || postulante.cedula || "",
        nacionalidad: datos.nacionalidad || "Ecuatoriana",
      })
    }
  }, [postulante, datos, onChange])

  const esExtranjera = datos.nacionalidad === "Extranjera"
  const requierePareja =
    datos.estadoCivil === "Casado/a" || datos.estadoCivil === "Unión de hecho"

  useEffect(() => {
    onBloqueoNacionalidad?.(esExtranjera)
  }, [esExtranjera, onBloqueoNacionalidad])

  function actualizarCampo<K extends keyof Seccion1Data>(
    campo: K,
    valor: Seccion1Data[K],
  ) {
    onChange({
      ...datos,
      [campo]: valor,
    })
  }

  const validacionCedulaConyuge = validarCedulaPareja(datos.cedulaConyuge)

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center gap-2 border-b pb-3">
        <UserIcon className="size-5 text-primary" aria-hidden="true" />
        <div>
          <h2 className="text-lg font-semibold">1. Información Personal</h2>
          <p className="text-xs text-muted-foreground">
            Datos del postulante vinculados a su cuenta OTP y declaraciones
            personales del concurso.
          </p>
        </div>
      </div>

      {esExtranjera && (
        <div
          role="alert"
          className="flex items-start gap-3 rounded-lg border border-destructive bg-destructive/10 p-4 text-sm text-destructive"
        >
          <AlertTriangleIcon className="size-5 shrink-0" aria-hidden="true" />
          <div>
            <h4 className="font-semibold">Nacionalidad no admitida</h4>
            <p className="mt-0.5">
              De acuerdo con las bases del concurso, la postulación está
              restringida a ciudadanos de nacionalidad ecuatoriana. El
              formulario no permitirá el envío.
            </p>
          </div>
        </div>
      )}

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="sec1-cedula">Cédula de Identidad</Label>
          <Input
            id="sec1-cedula"
            value={datos.cedula}
            disabled
            readOnly
            className="bg-muted font-mono"
          />
          <span className="text-[11px] text-muted-foreground">
            Vinculada a su cuenta autenticada.
          </span>
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="sec1-nacionalidad">Nacionalidad *</Label>
          <Select
            value={datos.nacionalidad || "Ecuatoriana"}
            onValueChange={(val) => actualizarCampo("nacionalidad", val)}
            disabled={disabled}
          >
            <SelectTrigger id="sec1-nacionalidad">
              <SelectValue placeholder="Seleccione su nacionalidad" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="Ecuatoriana">Ecuatoriana</SelectItem>
              <SelectItem value="Extranjera">Extranjera</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="sec1-nombres">Nombres *</Label>
          <Input
            id="sec1-nombres"
            value={datos.nombres}
            onChange={(e) => actualizarCampo("nombres", e.target.value)}
            disabled={disabled}
            placeholder="Nombres completos"
          />
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="sec1-apellidos">Apellidos *</Label>
          <Input
            id="sec1-apellidos"
            value={datos.apellidos}
            onChange={(e) => actualizarCampo("apellidos", e.target.value)}
            disabled={disabled}
            placeholder="Apellidos completos"
          />
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="sec1-genero">Género *</Label>
          <Select
            value={datos.genero || ""}
            onValueChange={(val) => actualizarCampo("genero", val)}
            disabled={disabled}
          >
            <SelectTrigger id="sec1-genero">
              <SelectValue placeholder="Seleccione género" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="Masculino">Masculino</SelectItem>
              <SelectItem value="Femenino">Femenino</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="sec1-nacimiento">Fecha de Nacimiento *</Label>
          <Input
            id="sec1-nacimiento"
            type="date"
            value={datos.fechaNacimiento || ""}
            onChange={(e) => actualizarCampo("fechaNacimiento", e.target.value)}
            disabled={disabled}
          />
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="sec1-estadoCivil">Estado Civil *</Label>
          <Select
            value={datos.estadoCivil || ""}
            onValueChange={(val) => actualizarCampo("estadoCivil", val)}
            disabled={disabled}
          >
            <SelectTrigger id="sec1-estadoCivil">
              <SelectValue placeholder="Seleccione estado civil" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="Soltero/a">Soltero/a</SelectItem>
              <SelectItem value="Casado/a">Casado/a</SelectItem>
              <SelectItem value="Unión de hecho">Unión de hecho</SelectItem>
              <SelectItem value="Divorciado/a">Divorciado/a</SelectItem>
              <SelectItem value="Viudo/a">Viudo/a</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      {requierePareja && (
        <div className="rounded-lg border bg-muted/20 p-4">
          <h3 className="mb-3 text-sm font-semibold text-foreground">
            Datos del Cónyuge o Conviviente
          </h3>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="sec1-conyuge-nombre">
                Nombres y Apellidos de la Pareja *
              </Label>
              <Input
                id="sec1-conyuge-nombre"
                value={datos.nombreConyuge || ""}
                onChange={(e) =>
                  actualizarCampo("nombreConyuge", e.target.value)
                }
                disabled={disabled}
                placeholder="Nombre completo del cónyuge"
              />
            </div>

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="sec1-conyuge-cedula">Cédula de la Pareja *</Label>
              <Input
                id="sec1-conyuge-cedula"
                value={datos.cedulaConyuge || ""}
                onChange={(e) =>
                  actualizarCampo("cedulaConyuge", e.target.value)
                }
                disabled={disabled}
                maxLength={10}
                placeholder="0999999999"
              />
              {validacionCedulaConyuge && !validacionCedulaConyuge.esValida && (
                <span className="text-[11px] text-destructive">
                  {validacionCedulaConyuge.error}
                </span>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
