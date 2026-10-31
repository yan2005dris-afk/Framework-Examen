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
 * Vista imprimible de la postulación (HU-11). Placeholder honesto: la tarea
 * FT6 implementa el formulario completo con espacio para la firma manuscrita y
 * los estilos `@media print`.
 */
export function ImprimirView() {
  return (
    <div className="mx-auto w-full max-w-3xl px-4 py-8">
      <Card>
        <CardHeader>
          <CardTitle>Impresión de la postulación</CardTitle>
          <CardDescription>
            La vista imprimible con el detalle de las nueve secciones y el
            espacio para la firma manuscrita se implementa en la tarea FT6
            (HU-11). Esta pantalla todavía está en construcción.
          </CardDescription>
        </CardHeader>
        <CardContent className="flex flex-wrap gap-2">
          <Button asChild variant="outline">
            <Link to={RUTAS.postulacionRevision}>Volver a la revisión</Link>
          </Button>
          <Button asChild variant="ghost">
            <Link to={RUTAS.postulacion}>Ir al formulario</Link>
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}
