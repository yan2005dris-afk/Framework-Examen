import { useEffect, useState } from "react"
import { AlertCircleIcon, ClockIcon } from "lucide-react"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"

type AlertaBloqueoProps = {
  /** Segundos iniciales de bloqueo (por defecto 600 s = 10 minutos). */
  segundosIniciales?: number
  /** Momento Unix (ms) o Date de vencimiento del bloqueo informado o persistido. */
  expiraEn?: number | Date
  /** Mensaje que describe el bloqueo. */
  mensaje?: string
  /** Callback cuando el temporizador llega a cero y se desbloquea. */
  onExpirar?: () => void
}

function calcularSegundosRestantes(
  expiraEn?: number | Date,
  segundosIniciales = 600,
): number {
  if (expiraEn) {
    const ms = typeof expiraEn === "number" ? expiraEn : expiraEn.getTime()
    return Math.max(0, Math.ceil((ms - Date.now()) / 1000))
  }
  return segundosIniciales
}

/**
 * Alerta de bloqueo por intentos fallidos (EF-11 / HU-03):
 * Muestra el mensaje literal del RF y una cuenta regresiva que rehabilita la UI al terminar.
 */
export function AlertaBloqueo({
  segundosIniciales = 600,
  expiraEn,
  mensaje = "registro fallido, inténtelo nuevamente en 10 minutos",
  onExpirar,
}: AlertaBloqueoProps) {
  const [prevExpiraEn, setPrevExpiraEn] = useState(expiraEn)
  const [segundos, setSegundos] = useState(() =>
    calcularSegundosRestantes(expiraEn, segundosIniciales),
  )

  if (prevExpiraEn !== expiraEn) {
    setPrevExpiraEn(expiraEn)
    setSegundos(calcularSegundosRestantes(expiraEn, segundosIniciales))
  }

  useEffect(() => {
    if (segundos <= 0) {
      onExpirar?.()
      return
    }

    const intervalo = setInterval(() => {
      setSegundos((prev) => {
        const actual = expiraEn
          ? calcularSegundosRestantes(expiraEn, segundosIniciales)
          : prev - 1
        if (actual <= 0) {
          clearInterval(intervalo)
          onExpirar?.()
          return 0
        }
        return actual
      })
    }, 1000)

    return () => clearInterval(intervalo)
  }, [segundos, expiraEn, segundosIniciales, onExpirar])

  if (segundos <= 0) {
    return null
  }

  const minutos = Math.floor(segundos / 60)
  const restoSegundos = segundos % 60
  const tiempoFormateado = `${minutos.toString().padStart(2, "0")}:${restoSegundos.toString().padStart(2, "0")}`

  return (
    <Alert
      variant="destructive"
      className="border-red-500/50 bg-red-500/10 text-red-700 dark:text-red-400"
    >
      <AlertCircleIcon className="size-4 shrink-0" />
      <div className="flex-1">
        <AlertTitle className="font-semibold capitalize">
          Cuenta temporalmente bloqueada
        </AlertTitle>
        <AlertDescription className="mt-1 space-y-1.5 text-xs">
          <p>{mensaje}</p>
          <div className="flex items-center gap-1.5 font-mono font-medium text-red-600 dark:text-red-300">
            <ClockIcon className="size-3.5" />
            <span>Tiempo restante: {tiempoFormateado}</span>
          </div>
        </AlertDescription>
      </div>
    </Alert>
  )
}
