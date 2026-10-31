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
 * Banco de preguntas y ponderaciones para el examen EF-14.
 * Placeholder honesto: la tarea FT24 implementa la carga, taxonomía y validación de reactivos.
 */
export function AdminPreguntasView() {
  return (
    <div className="flex flex-1 flex-col gap-4 p-4">
      <Card>
        <CardHeader>
          <CardTitle>Banco de preguntas (EF-14)</CardTitle>
          <CardDescription>
            Administración de reactivos teóricos y psicotécnicos para la
            Comisión de Selección. Se implementa en la tarea FT24.
          </CardDescription>
        </CardHeader>
        <CardContent className="flex flex-wrap gap-2">
          <Button asChild variant="outline">
            <Link to={RUTAS.adminConvocatorias}>Volver a convocatorias</Link>
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}
