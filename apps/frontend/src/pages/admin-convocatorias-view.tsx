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
 * Panel de gestión de convocatorias para la Comisión de Selección.
 * Placeholder honesto: la tarea FT15 implementa el CRUD y publicación de convocatorias.
 */
export function AdminConvocatoriasView() {
  return (
    <div className="flex flex-1 flex-col gap-4 p-4">
      <Card>
        <CardHeader>
          <CardTitle>Gestión de Convocatorias</CardTitle>
          <CardDescription>
            Configuración y publicación de concursos BIESS/IESS. Se implementa
            en la tarea FT15.
          </CardDescription>
        </CardHeader>
        <CardContent className="flex flex-wrap gap-2">
          <Button asChild variant="outline">
            <Link to={RUTAS.adminFases}>Ir a gestión de fases</Link>
          </Button>
          <Button asChild variant="outline">
            <Link to={RUTAS.dashboard}>Ver panel general</Link>
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}
