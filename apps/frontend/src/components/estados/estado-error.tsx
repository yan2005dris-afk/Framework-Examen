import { AlertCircleIcon, RotateCcwIcon } from "lucide-react"

import { Button } from "@/components/ui/button"
import {
  Card,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { esErrorApi } from "@/lib/api-error"
import { cn } from "@/lib/utils"

export interface EstadoErrorProps {
  error?: unknown
  titulo?: string
  mensaje?: string
  onReintentar?: () => void
  className?: string
}

/**
 * Visualización accesible y uniforme de errores para vistas asíncronas.
 */
export function EstadoError({
  error,
  titulo = "No se pudo cargar la información",
  mensaje,
  onReintentar,
  className,
}: EstadoErrorProps) {
  let textoDetalle = mensaje
  let codigo: string | undefined

  if (!textoDetalle) {
    if (esErrorApi(error)) {
      textoDetalle = error.mensaje
      codigo = error.codigo
    } else if (error instanceof Error) {
      textoDetalle = error.message
    } else if (typeof error === "string") {
      textoDetalle = error
    } else {
      textoDetalle = "Ocurrió un error inesperado al procesar la solicitud."
    }
  }

  return (
    <Card
      role="alert"
      className={cn("border-destructive/40 bg-destructive/5", className)}
    >
      <CardHeader>
        <div className="flex items-center gap-2 text-destructive">
          <AlertCircleIcon className="size-5 shrink-0" aria-hidden="true" />
          <CardTitle className="text-base font-semibold">{titulo}</CardTitle>
        </div>
        <CardDescription className="text-sm text-foreground/80">
          {textoDetalle}
        </CardDescription>
        {codigo && (
          <p className="font-mono text-xs text-muted-foreground">
            Código: {codigo}
          </p>
        )}
      </CardHeader>
      {onReintentar && (
        <CardFooter className="pt-0">
          <Button
            variant="outline"
            size="sm"
            onClick={onReintentar}
            className="flex items-center gap-1.5"
          >
            <RotateCcwIcon className="size-3.5" aria-hidden="true" />
            <span>Reintentar</span>
          </Button>
        </CardFooter>
      )}
    </Card>
  )
}
