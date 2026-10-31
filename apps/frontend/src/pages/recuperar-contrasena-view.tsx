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
 * Recuperación de contraseña (EF-11). Placeholder honesto: la tarea FT3
 * implementa la solicitud del código OTP y el restablecimiento.
 */
export function RecuperarContrasenaView() {
  return (
    <div className="mx-auto w-full max-w-md px-4 py-12">
      <Card>
        <CardHeader>
          <CardTitle>Recuperar contraseña</CardTitle>
          <CardDescription>
            La solicitud del código de recuperación y el cambio de contraseña se
            implementan en la tarea FT3 (EF-11). Esta pantalla todavía está en
            construcción.
          </CardDescription>
        </CardHeader>
        <CardContent className="flex flex-wrap gap-2">
          <Button asChild variant="outline">
            <Link to={RUTAS.login}>Volver al inicio de sesión</Link>
          </Button>
          <Button asChild variant="ghost">
            <Link to={RUTAS.inicio}>Ir a la portada</Link>
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}
