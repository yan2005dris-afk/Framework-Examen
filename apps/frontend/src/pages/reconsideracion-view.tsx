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
 * Solicitud y consulta de reconsideraciones del postulante.
 * Placeholder honesto: la tarea FT21 implementa el formulario de reclamos y plazos.
 */
export function ReconsideracionView() {
  return (
    <div className="mx-auto flex w-full max-w-5xl flex-1 flex-col gap-4 p-4 sm:p-6">
      <Card>
        <CardHeader>
          <CardTitle>Solicitud de reconsideración</CardTitle>
          <CardDescription>
            La presentación de apelaciones sobre méritos y evaluación se
            implementa en la tarea FT21. Esta pantalla se encuentra en
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
