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
 * Registro de postulante (EF-11). Placeholder honesto: la tarea FT3
 * implementa el wizard de tres pasos y la validación de identidad.
 */
export function RegistroView() {
  return (
    <div className="mx-auto w-full max-w-lg px-4 py-12">
      <Card>
        <CardHeader>
          <CardTitle>Registro de postulante</CardTitle>
          <CardDescription>
            El registro con cédula, validación OTP y creación de contraseña se
            implementa en la tarea FT3 (EF-11). Esta pantalla todavía está en
            construcción.
          </CardDescription>
        </CardHeader>
        <CardContent className="flex flex-wrap gap-2">
          <Button asChild variant="outline">
            <Link to={RUTAS.login}>Ya tengo cuenta</Link>
          </Button>
          <Button asChild variant="ghost">
            <Link to={RUTAS.inicio}>Volver al inicio</Link>
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}
