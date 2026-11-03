import { RefreshCwIcon, TriangleAlertIcon } from "lucide-react"
import { isRouteErrorResponse, Link, useRouteError } from "react-router"

import { Button } from "@/components/ui/button"
import { MarcoEstadoPortal } from "@/components/cabecera-portal"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { RUTAS } from "@/lib/rutas"

export interface PantallaErrorRutaProps {
  /** Mensaje técnico del error; sólo se muestra en desarrollo. */
  detalle?: string
  mostrarDetalle: boolean
}

/**
 * Pantalla que reemplaza a la de React Router cuando una vista falla al
 * mostrarse: por ejemplo, si una extensión del navegador modificó la página.
 * Ofrece recargar, que suele bastar, sin mostrar el error técnico al postulante.
 */
export function PantallaErrorRuta({
  detalle,
  mostrarDetalle,
}: PantallaErrorRutaProps) {
  return (
    <MarcoEstadoPortal>
      <Card className="w-full">
        <CardHeader>
          <div className="mb-2 flex size-10 items-center justify-center rounded-lg bg-muted text-muted-foreground">
            <TriangleAlertIcon className="size-5" aria-hidden="true" />
          </div>
          <CardTitle>
            <h1>Ocurrió un problema al mostrar esta página</h1>
          </CardTitle>
          <CardDescription>
            Sus datos guardados no se perdieron. Recargue la página para
            continuar. Si el problema se repite, desactive la traducción
            automática o las extensiones del navegador para este sitio.
          </CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <div className="flex flex-wrap gap-2">
            <Button type="button" onClick={() => window.location.reload()}>
              <RefreshCwIcon data-icon="inline-start" />
              Recargar la página
            </Button>
            <Button asChild variant="outline">
              <Link to={RUTAS.inicio}>Volver al inicio</Link>
            </Button>
          </div>
          {mostrarDetalle && detalle && (
            <pre className="overflow-x-auto rounded-md bg-muted p-3 text-xs text-muted-foreground">
              {detalle}
            </pre>
          )}
        </CardContent>
      </Card>
    </MarcoEstadoPortal>
  )
}

/** `errorElement` de las rutas: el detalle técnico sólo se ve en desarrollo. */
export function ErrorRutaView() {
  const error = useRouteError()
  const detalle = isRouteErrorResponse(error)
    ? `${error.status} ${error.statusText}`
    : error instanceof Error
      ? error.message
      : undefined
  return (
    <PantallaErrorRuta detalle={detalle} mostrarDetalle={import.meta.env.DEV} />
  )
}
