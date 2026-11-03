import { useEffect, useState } from "react"
import { AlertCircleIcon, ClockIcon } from "lucide-react"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Progress } from "@/components/ui/progress"

type AlertaBloqueoProps = {
  pantalla?: boolean
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
  pantalla = false,
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
      className="border-destructive/30 bg-destructive-soft text-destructive"
    >
      <AlertCircleIcon className="size-4 shrink-0" />
      <div className="flex-1">
        <AlertTitle className="font-semibold capitalize">
          Cuenta temporalmente bloqueada
        </AlertTitle>
        <AlertDescription className="mt-1 space-y-1.5 text-xs">
          <p>{mensaje}</p>
          <div
            className={
              pantalla
                ? "mt-4 space-y-3 rounded-lg border bg-card p-5 text-foreground"
                : "flex items-center gap-1.5 font-mono font-medium"
            }
          >
            {pantalla && (
              <p className="text-xs font-semibold text-muted-foreground">
                Tiempo restante de bloqueo
              </p>
            )}
            <ClockIcon className="size-3.5" />
            <span
              className={
                pantalla
                  ? "block font-mono text-4xl font-bold tracking-tight"
                  : undefined
              }
            >
              <span className={pantalla ? "sr-only" : undefined}>
                Tiempo restante:{" "}
              </span>
              {tiempoFormateado}
            </span>
            {pantalla && (
              <>
                <Progress
                  value={Math.min(100, (segundos / segundosIniciales) * 100)}
                  aria-label="Tiempo de bloqueo restante"
                />
                <p className="text-xs text-muted-foreground">
                  La autorización final y la liberación del bloqueo corresponden
                  al servidor. No se restablece el plazo al recargar esta
                  pantalla.
                </p>
              </>
            )}
          </div>
        </AlertDescription>
      </div>
    </Alert>
  )
}
