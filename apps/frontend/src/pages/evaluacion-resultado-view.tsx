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
 * Nota final, desglose de respuestas y acta de examen.
 * Placeholder honesto: la tarea FT27 implementa el cálculo de calificación y generación de reporte.
 */
export function EvaluacionResultadoView() {
  return (
    <div className="mx-auto flex w-full max-w-5xl flex-1 flex-col gap-4 p-4 sm:p-6">
      <Card>
        <CardHeader>
          <CardTitle>Resultados de la evaluación</CardTitle>
          <CardDescription>
            La consulta de puntajes obtenidos y descarga de acta se implementan
            en la tarea FT27.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Button asChild variant="outline">
            <Link to={RUTAS.evaluacion}>Volver a evaluación</Link>
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}
