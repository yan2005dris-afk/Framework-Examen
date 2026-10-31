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
 * Seguimiento de fases, resultados y notificaciones del postulante.
 * Placeholder honesto: la tarea FT22 implementa la línea de tiempo y el estado por etapa.
 */
export function SeguimientoView() {
  return (
    <div className="mx-auto flex w-full max-w-5xl flex-1 flex-col gap-4 p-4 sm:p-6">
      <Card>
        <CardHeader>
          <CardTitle>Seguimiento de fases y resultados</CardTitle>
          <CardDescription>
            El estado de avance del concurso, notificaciones y resultados por
            etapa se implementan en la tarea FT22. Esta pantalla se encuentra en
            construcción.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Button asChild variant="outline">
            <Link to={RUTAS.postulacion}>Volver a mi postulación</Link>
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}
