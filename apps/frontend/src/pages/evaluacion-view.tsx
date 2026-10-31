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
 * Portal de acceso a la evaluación técnica EF-14.
 * Placeholder honesto: la tarea FT25 implementa la habilitación y control de horario del servidor.
 */
export function EvaluacionView() {
  return (
    <div className="mx-auto flex w-full max-w-5xl flex-1 flex-col gap-4 p-4 sm:p-6">
      <Card>
        <CardHeader>
          <CardTitle>Evaluación técnica (EF-14)</CardTitle>
          <CardDescription>
            La programación, verificación de elegibilidad y sala de espera se
            implementan en la tarea FT25.
          </CardDescription>
        </CardHeader>
        <CardContent className="flex flex-wrap gap-2">
          <Button asChild>
            <Link to={RUTAS.evaluacionExamen}>Rendir examen</Link>
          </Button>
          <Button asChild variant="outline">
            <Link to={RUTAS.evaluacionResultado}>Ver resultados</Link>
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}
