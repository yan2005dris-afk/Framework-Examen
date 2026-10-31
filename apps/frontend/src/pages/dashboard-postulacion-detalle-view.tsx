import { Link, useParams } from "react-router"

import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { RUTAS } from "@/lib/rutas"

/**
 * Detalle de una postulación para el analista. Placeholder honesto: la tarea
 * FT7 implementa el expediente completo y el cambio de estado.
 */
export function DashboardPostulacionDetalleView() {
  const { id } = useParams<{ id: string }>()

  return (
    <div className="flex flex-1 flex-col gap-4 p-4">
      <Card>
        <CardHeader>
          <CardTitle>Detalle de la postulación</CardTitle>
          <CardDescription>
            El expediente completo del postulante y la revisión por secciones se
            implementan en la tarea FT7. Esta pantalla todavía está en
            construcción.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-muted-foreground">
            Identificador solicitado:{" "}
            <code className="rounded bg-muted px-1.5 py-0.5 font-mono text-xs text-foreground">
              {id}
            </code>
          </p>
          <Button asChild variant="outline">
            <Link to={RUTAS.dashboardPostulaciones}>Volver al listado</Link>
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}
