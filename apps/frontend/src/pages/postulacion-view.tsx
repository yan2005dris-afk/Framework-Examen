import { Link, useParams } from "react-router"

import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { RUTAS, rutaSeccion } from "@/lib/rutas"

/**
 * Formulario EF-12. Atiende tanto `/postulacion` como `/postulacion/:seccion`.
 * Placeholder honesto: la tarea FT5 implementa las nueve secciones, el
 * guardado parcial y la barra de progreso.
 */
export function PostulacionView() {
  const { seccion } = useParams<{ seccion: string }>()
  const numero = Number(seccion)
  const esSeccionValida = Number.isInteger(numero) && numero >= 1 && numero <= 9

  if (seccion && !esSeccionValida) {
    return (
      <div className="mx-auto w-full max-w-3xl px-4 py-12">
        <Card>
          <CardHeader>
            <CardTitle>Sección no encontrada</CardTitle>
            <CardDescription>
              El formulario EF-12 tiene nueve secciones numeradas del 1 al 9. La
              sección «{seccion}» no existe.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button asChild variant="outline">
              <Link to={RUTAS.postulacion}>Abrir el formulario</Link>
            </Button>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="mx-auto w-full max-w-3xl px-4 py-8">
      <Card>
        <CardHeader>
          <CardTitle>
            {esSeccionValida
              ? `Sección ${numero} del formulario EF-12`
              : "Formulario EF-12"}
          </CardTitle>
          <CardDescription>
            El formulario en nueve secciones, con guardado parcial y barra de
            progreso, se implementa en la tarea FT5. Esta pantalla todavía está
            en construcción.
          </CardDescription>
        </CardHeader>
        <CardContent className="flex flex-wrap gap-2">
          {esSeccionValida && numero < 9 && (
            <Button asChild variant="outline">
              <Link to={rutaSeccion(numero + 1)}>
                Ir a la sección {numero + 1}
              </Link>
            </Button>
          )}
          <Button asChild variant="ghost">
            <Link to={RUTAS.postulacionRevision}>Ver la revisión</Link>
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}
