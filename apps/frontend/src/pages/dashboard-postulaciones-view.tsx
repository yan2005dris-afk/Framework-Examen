import { Link } from "react-router"

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
 * Listado de postulaciones para el analista. Placeholder honesto: la tarea FT7
 * implementa la tabla con datos reales y la paginación por cursor.
 */
export function DashboardPostulacionesView() {
  return (
    <div className="flex flex-1 flex-col gap-4 p-4">
      <Card>
        <CardHeader>
          <CardTitle>Postulaciones recibidas</CardTitle>
          <CardDescription>
            El listado de revisión con paginación por cursor se implementa en la
            tarea FT7. Esta pantalla todavía está en construcción.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Button asChild variant="outline">
            <Link to={RUTAS.dashboard}>Volver al panel</Link>
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}
