import { FileQuestionIcon } from "lucide-react"
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
 * Página `404` para cualquier URL que no exista en el router.
 */
export function NoEncontradoView() {
  return (
    <div className="mx-auto w-full max-w-[46rem] px-5 pt-8 pb-16">
      <Card className="w-full">
        <CardHeader>
          <div className="mb-2 flex size-10 items-center justify-center rounded-lg bg-muted text-muted-foreground">
            <FileQuestionIcon className="size-5" aria-hidden="true" />
          </div>
          <CardTitle>
            <h1>Página no encontrada</h1>
          </CardTitle>
          <CardDescription>
            La dirección que ingresó no corresponde a ninguna pantalla del
            portal de postulación.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Button asChild>
            <Link to={RUTAS.inicio}>Volver al inicio</Link>
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}
