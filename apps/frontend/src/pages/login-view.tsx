import { Link, useSearchParams } from "react-router"

import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { RUTAS, leerDestinoLogin } from "@/lib/rutas"

/**
 * Login del postulante (EF-11). Placeholder honesto: la tarea FT3 implementa
 * el formulario, el bloqueo por intentos y la redirección por
 * `mustCompleteApplication`.
 *
 * Mientras tanto muestra el destino que el guard dejó en la query string, para
 * verificar que la ruta protegida se conserva.
 */
export function LoginView() {
  const [parametros] = useSearchParams()
  const destino = leerDestinoLogin(parametros.toString())

  return (
    <div className="mx-auto w-full max-w-md px-4 py-12">
      <Card>
        <CardHeader>
          <CardTitle>Iniciar sesión</CardTitle>
          <CardDescription>
            Ingreso con cédula y contraseña (EF-11). Esta pantalla se implementa
            en la tarea FT3 y todavía está en construcción.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4 text-sm text-muted-foreground">
          <p>
            Al autenticarse, el portal lo devolverá a{" "}
            <code className="rounded bg-muted px-1.5 py-0.5 font-mono text-xs text-foreground">
              {destino}
            </code>
            .
          </p>
          <div className="flex flex-wrap gap-2">
            <Button asChild variant="outline">
              <Link to={RUTAS.registro}>Crear una cuenta</Link>
            </Button>
            <Button asChild variant="ghost">
              <Link to={RUTAS.inicio}>Volver al inicio</Link>
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
