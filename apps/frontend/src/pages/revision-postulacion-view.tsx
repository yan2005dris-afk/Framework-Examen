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
 * Revisión y envío de la postulación. Placeholder honesto: la tarea FT6
 * implementa el resumen de las nueve secciones y los cinco consentimientos.
 */
export function RevisionPostulacionView() {
  return (
    <div className="mx-auto w-full max-w-3xl px-4 py-8">
      <Card>
        <CardHeader>
          <CardTitle>Revisión y envío</CardTitle>
          <CardDescription>
            El resumen de las nueve secciones, los cinco puntos de
            consentimiento y el sellado con hash se implementan en la tarea FT6.
            Esta pantalla todavía está en construcción.
          </CardDescription>
        </CardHeader>
        <CardContent className="flex flex-wrap gap-2">
          <Button asChild variant="outline">
            <Link to={RUTAS.postulacion}>Volver al formulario</Link>
          </Button>
          <Button asChild variant="ghost">
            <Link to={RUTAS.postulacionImprimir}>Ver la impresión</Link>
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}
