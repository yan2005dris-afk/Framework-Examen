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
    <div className="flex flex-1 items-center justify-center p-6">
      <Card className="w-full max-w-lg">
        <CardHeader>
          <div className="mb-2 flex size-10 items-center justify-center rounded-lg bg-muted text-muted-foreground">
            <FileQuestionIcon className="size-5" aria-hidden="true" />
          </div>
          <CardTitle>Página no encontrada</CardTitle>
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
