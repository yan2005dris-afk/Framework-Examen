import { MapPinIcon } from "lucide-react"
import { useEffect } from "react"

import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { PROVINCIAS_ECUADOR } from "@/data/ecuadorGeo"
import { useSesion } from "@/features/auth/sesion-context"
import { useCatalogosPaises } from "@/features/postulaciones/queries"
import { validarEmail } from "@/utils/validators"

export interface Seccion2Data {
  pais: string
  provincia: string
  canton: string
  telefonoCelular: string
  correo1: string
  correo2?: string
  callePrincipal: string
  numero: string
  calleSecundaria: string
  sector: string
  referencia: string
  telefonoDomiciliario: string
}

export interface Seccion2ContactoProps {
  datos: Seccion2Data
  onChange: (nuevosDatos: Seccion2Data) => void
  disabled?: boolean
}

export function Seccion2Contacto({
  datos,
  onChange,
  disabled = false,
}: Seccion2ContactoProps) {
  const { postulante } = useSesion()
  const { data: listaPaises = [] } = useCatalogosPaises()

  useEffect(() => {
    if (postulante?.correo && !datos.correo1) {
      onChange({
        ...datos,
        correo1: postulante.correo,
        pais: datos.pais || "Ecuador",
      })
    }
  }, [postulante, datos, onChange])

  const paisActual = datos.pais || "Ecuador"
  const esEcuador = paisActual.toLowerCase() === "ecuador"

  const provinciasDisponibles = esEcuador ? Object.keys(PROVINCIAS_ECUADOR) : []
  const cantonesDisponibles =
    esEcuador && datos.provincia
      ? PROVINCIAS_ECUADOR[datos.provincia] || []
      : []

  function actualizarCampo<K extends keyof Seccion2Data>(
    campo: K,
    valor: Seccion2Data[K],
  ) {
    if (campo === "pais") {
      // Al cambiar de país, resetear provincia y cantón
      onChange({
        ...datos,
        pais: valor as string,
        provincia: "",
        canton: "",
      })
      return
    }

    if (campo === "provincia" && esEcuador) {
      // Al cambiar provincia en Ecuador, resetear cantón
      onChange({
        ...datos,
        provincia: valor as string,
        canton: "",
      })
      return
    }

    onChange({
      ...datos,
      [campo]: valor,
    })
  }

  const errorCorreo2 =
    datos.correo2 && !validarEmail(datos.correo2) ? "Correo inválido" : null

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center gap-2 border-b pb-3">
        <MapPinIcon className="size-5 text-primary" aria-hidden="true" />
        <div>
          <h2 className="text-lg font-semibold">
            2. Información Domiciliaria y de Contacto
          </h2>
          <p className="text-xs text-muted-foreground">
            Lugar de residencia habitual y canales oficiales de comunicación.
          </p>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="sec2-pais">País *</Label>
          <Select
            value={paisActual}
            onValueChange={(val) => actualizarCampo("pais", val)}
            disabled={disabled}
          >
            <SelectTrigger id="sec2-pais">
              <SelectValue placeholder="Seleccione país" />
            </SelectTrigger>
            <SelectContent className="max-h-60">
              {listaPaises.map((pais) => (
                <SelectItem key={pais} value={pais}>
                  {pais}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="sec2-provincia">Provincia / Estado *</Label>
          {esEcuador ? (
            <Select
              value={datos.provincia || ""}
              onValueChange={(val) => actualizarCampo("provincia", val)}
              disabled={disabled}
            >
              <SelectTrigger id="sec2-provincia">
                <SelectValue placeholder="Seleccione provincia" />
              </SelectTrigger>
              <SelectContent className="max-h-60">
                {provinciasDisponibles.map((prov) => (
                  <SelectItem key={prov} value={prov}>
                    {prov}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          ) : (
            <Input
              id="sec2-provincia"
              value={datos.provincia || ""}
              onChange={(e) => actualizarCampo("provincia", e.target.value)}
              disabled={disabled}
              placeholder="Escriba la provincia o estado"
            />
          )}
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="sec2-canton">Cantón / Ciudad *</Label>
          {esEcuador ? (
            <Select
              value={datos.canton || ""}
              onValueChange={(val) => actualizarCampo("canton", val)}
              disabled={disabled || !datos.provincia}
            >
              <SelectTrigger id="sec2-canton">
                <SelectValue
                  placeholder={
                    datos.provincia
                      ? "Seleccione cantón"
                      : "Elija primero provincia"
                  }
                />
              </SelectTrigger>
              <SelectContent className="max-h-60">
                {cantonesDisponibles.map((cant) => (
                  <SelectItem key={cant} value={cant}>
                    {cant}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          ) : (
            <Input
              id="sec2-canton"
              value={datos.canton || ""}
              onChange={(e) => actualizarCampo("canton", e.target.value)}
              disabled={disabled}
              placeholder="Escriba la ciudad o cantón"
            />
          )}
        </div>
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="sec2-correo1">Correo Principal (Cuenta OTP) *</Label>
          <Input
            id="sec2-correo1"
            value={datos.correo1}
            disabled
            readOnly
            className="bg-muted font-mono"
          />
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="sec2-correo2">Correo Alternativo (Opcional)</Label>
          <Input
            id="sec2-correo2"
            type="email"
            value={datos.correo2 || ""}
            onChange={(e) => actualizarCampo("correo2", e.target.value)}
            disabled={disabled}
            placeholder="correo.alternativo@example.com"
          />
          {errorCorreo2 && (
            <span className="text-[11px] text-destructive">{errorCorreo2}</span>
          )}
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="sec2-celular">Teléfono Celular *</Label>
          <Input
            id="sec2-celular"
            value={datos.telefonoCelular}
            onChange={(e) => actualizarCampo("telefonoCelular", e.target.value)}
            disabled={disabled}
            placeholder="0999999999"
          />
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="sec2-convencional">Teléfono Domiciliario *</Label>
          <Input
            id="sec2-convencional"
            value={datos.telefonoDomiciliario}
            onChange={(e) =>
              actualizarCampo("telefonoDomiciliario", e.target.value)
            }
            disabled={disabled}
            placeholder="022222222"
          />
        </div>
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <div className="flex flex-col gap-1.5 sm:col-span-2">
          <Label htmlFor="sec2-calle-principal">Calle Principal *</Label>
          <Input
            id="sec2-calle-principal"
            value={datos.callePrincipal}
            onChange={(e) => actualizarCampo("callePrincipal", e.target.value)}
            disabled={disabled}
            placeholder="Av. 10 de Agosto"
          />
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="sec2-numero">Número *</Label>
          <Input
            id="sec2-numero"
            value={datos.numero}
            onChange={(e) => actualizarCampo("numero", e.target.value)}
            disabled={disabled}
            placeholder="N24-12 o S/N"
          />
        </div>

        <div className="flex flex-col gap-1.5 sm:col-span-2">
          <Label htmlFor="sec2-calle-secundaria">Calle Secundaria *</Label>
          <Input
            id="sec2-calle-secundaria"
            value={datos.calleSecundaria}
            onChange={(e) => actualizarCampo("calleSecundaria", e.target.value)}
            disabled={disabled}
            placeholder="Intersección o transversal"
          />
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="sec2-sector">Sector / Barrio *</Label>
          <Input
            id="sec2-sector"
            value={datos.sector}
            onChange={(e) => actualizarCampo("sector", e.target.value)}
            disabled={disabled}
            placeholder="Centro Norte"
          />
        </div>

        <div className="flex flex-col gap-1.5 sm:col-span-3">
          <Label htmlFor="sec2-referencia">Referencia de Ubicación *</Label>
          <Input
            id="sec2-referencia"
            value={datos.referencia}
            onChange={(e) => actualizarCampo("referencia", e.target.value)}
            disabled={disabled}
            placeholder="Junto al parque, frente a la farmacia..."
          />
        </div>
      </div>
    </div>
  )
}
