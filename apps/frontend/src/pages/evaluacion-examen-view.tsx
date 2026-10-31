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
 * Aplicación de examen con reloj de cuenta regresiva y navegación de preguntas.
 * Placeholder honesto: la tarea FT26 implementa el banco dinámico y el timer del servidor.
 */
export function EvaluacionExamenView() {
  return (
    <div className="mx-auto flex w-full max-w-5xl flex-1 flex-col gap-4 p-4 sm:p-6">
      <Card>
        <CardHeader>
          <CardTitle>Examen de conocimientos</CardTitle>
          <CardDescription>
            La interfaz de examen en tiempo real con preguntas individuales y
            reloj sincronizado se implementa en la tarea FT26.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Button asChild variant="outline">
            <Link to={RUTAS.evaluacion}>Volver al portal de evaluación</Link>
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}
