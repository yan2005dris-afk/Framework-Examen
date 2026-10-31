import { ShieldXIcon } from "lucide-react"
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

type SinPermisoViewProps = {
  rolRequerido?: "analista" | "administrador"
}

/**
 * Respuesta `403` del portal: hay sesión, pero el rol no alcanza para la ruta
 * solicitada. Se muestra como página propia en lugar de redirigir, para que el
 * usuario entienda por qué no puede entrar.
 */
export function SinPermisoView({
  rolRequerido = "analista",
}: SinPermisoViewProps) {
  const descripcion =
    rolRequerido === "administrador"
      ? "Esta sección está reservada a los miembros de la Comisión de Selección con rol de Administrador. Su cuenta no dispone de estos privilegios."
      : "El panel de revisión está reservado a las cuentas con rol de analista de Talento Humano o Administrador."

  return (
    <div className="mx-auto flex min-h-svh w-full max-w-lg items-center justify-center p-6">
      <Card className="w-full">
        <CardHeader>
          <div className="mb-2 flex size-10 items-center justify-center rounded-lg bg-muted text-muted-foreground">
            <ShieldXIcon className="size-5" aria-hidden="true" />
          </div>
          <CardTitle>No tiene permisos para esta sección</CardTitle>
          <CardDescription>{descripcion}</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-wrap gap-2">
          <Button asChild>
            <Link to={RUTAS.postulacion}>Ir a mi postulación</Link>
          </Button>
          <Button asChild variant="outline">
            <Link to={RUTAS.inicio}>Volver al inicio</Link>
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}
