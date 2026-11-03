import { Link } from "react-router"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { RUTAS } from "@/lib/rutas"
import { AlertaBloqueo } from "./alerta-bloqueo"

/** Pantalla de bloqueo sin historiales inventados ni atribuir validación al Registro Civil. */
export function PantallaBloqueoRegistro({
  expiraEn,
  mensaje,
  onExpirar,
}: {
  expiraEn: Date
  mensaje?: string
  onExpirar: () => void
}) {
  return (
    <div className="mx-auto w-full max-w-[46rem] px-5 pt-8 pb-16">
      <Card>
        <CardHeader>
          <CardTitle>
            <h1 className="text-lg font-bold">
              Registro bloqueado temporalmente
            </h1>
          </CardTitle>
          <CardDescription>
            El servidor limitó temporalmente los intentos de registro o
            verificación.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-5">
          <AlertaBloqueo
            pantalla
            expiraEn={expiraEn}
            mensaje={mensaje}
            onExpirar={onExpirar}
          />
          <p className="text-sm text-muted-foreground">
            Espere a que termine el plazo antes de volver a intentarlo. No se
            muestran intentos, fechas o resultados que la API no haya
            proporcionado.
          </p>
          <Button asChild variant="outline">
            <Link to={RUTAS.login}>Ir al inicio de sesión</Link>
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}
