import { useEffect, useState } from "react"
import { AlertCircleIcon, ClockIcon } from "lucide-react"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"

type AlertaBloqueoProps = {
  /** Segundos iniciales de bloqueo (por defecto 600 s = 10 minutos). */
  segundosIniciales?: number
  /** Mensaje que describe el bloqueo. */
  mensaje?: string
  /** Callback cuando el temporizador llega a cero y se desbloquea. */
  onExpirar?: () => void
}

/**
 * Alerta de bloqueo por intentos fallidos (EF-11 / HU-03):
 * Muestra el mensaje literal del RF y una cuenta regresiva que rehabilita la UI al terminar.
 */
export function AlertaBloqueo({
  segundosIniciales = 600,
  mensaje = "registro fallido, inténtelo nuevamente en 10 minutos",
  onExpirar,
}: AlertaBloqueoProps) {
  const [segundos, setSegundos] = useState(segundosIniciales)
  const [prevSegundosIniciales, setPrevSegundosIniciales] =
    useState(segundosIniciales)

  if (prevSegundosIniciales !== segundosIniciales) {
    setPrevSegundosIniciales(segundosIniciales)
    setSegundos(segundosIniciales)
  }

  useEffect(() => {
    if (segundos <= 0) {
      onExpirar?.()
      return
    }

    const intervalo = setInterval(() => {
      setSegundos((prev) => {
        if (prev <= 1) {
          clearInterval(intervalo)
          onExpirar?.()
          return 0
        }
        return prev - 1
      })
    }, 1000)

    return () => clearInterval(intervalo)
  }, [segundos, onExpirar])

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
