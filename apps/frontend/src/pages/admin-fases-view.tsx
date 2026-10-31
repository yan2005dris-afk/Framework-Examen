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
 * Cronograma y habilitación de fases del concurso.
 * Placeholder honesto: la tarea FT15 implementa la apertura/cierre de etapas y calendario oficial.
 */
export function AdminFasesView() {
  return (
    <div className="flex flex-1 flex-col gap-4 p-4">
      <Card>
        <CardHeader>
          <CardTitle>Fases del concurso</CardTitle>
          <CardDescription>
            Control de etapas, apertura de formularios y publicación de
            cronograma. Se implementa en la tarea FT15.
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
