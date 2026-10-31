import {
  ArrowRightIcon,
  ClipboardCheckIcon,
  FileSignatureIcon,
  ShieldCheckIcon,
} from "lucide-react"
import { Link } from "react-router"

import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { useSesion } from "@/features/auth/sesion-context"
import { useRutaInicial } from "@/features/auth/use-ruta-inicial"
import { RUTAS } from "@/lib/rutas"

const PASOS = [
  {
    titulo: "Regístrese con su cédula",
    detalle:
      "Validación de identidad con código OTP enviado al correo registrado.",
    icono: ShieldCheckIcon,
  },
  {
    titulo: "Complete el formulario EF-12",
    detalle:
      "Nueve secciones: datos personales, domicilio, formación, experiencia y declaraciones.",
    icono: ClipboardCheckIcon,
  },
  {
    titulo: "Revise, envíe e imprima",
    detalle:
      "El envío sella la postulación con un hash criptográfico y habilita la impresión.",
    icono: FileSignatureIcon,
  },
]

/**
 * Portada pública del concurso. Si hay sesión activa ofrece el acceso directo
 * a la ruta que corresponde al postulante.
 */
export function PortadaView() {
  const { estado, postulante } = useSesion()
  const { ruta, calculando } = useRutaInicial()
  const autenticado = estado === "autenticado"

  return (
    <div className="mx-auto w-full max-w-5xl px-4 py-12">
      <section className="space-y-6">
        <p className="text-sm font-medium text-muted-foreground">
          Portal de postulación · BIESS / IESS
        </p>
        <h1 className="max-w-3xl text-4xl font-semibold tracking-tight text-balance sm:text-5xl">
          Concurso de mérito para el Directorio del BIESS
        </h1>
        <p className="max-w-2xl text-lg text-muted-foreground">
          Registre su postulación, complete el formulario EF-12 en nueve
          secciones y remita su expediente con constancia criptográfica.
        </p>

        {autenticado && (
          <p className="text-sm text-muted-foreground">
            Sesión iniciada como{" "}
            <span className="font-medium text-foreground">
              {postulante?.nombre_completo}
            </span>
            .
          </p>
        )}

        <div className="flex flex-wrap items-center gap-3">
          {autenticado ? (
            ruta ? (
              <Button asChild>
                <Link to={ruta}>
                  {ruta === RUTAS.dashboard
                    ? "Ir al panel"
                    : "Continuar mi postulación"}
                  <ArrowRightIcon aria-hidden="true" />
                </Link>
              </Button>
            ) : (
              <Button disabled>
                {calculando ? "Preparando su panel…" : "Sesión no disponible"}
              </Button>
            )
          ) : (
            <>
              <Button asChild>
                <Link to={RUTAS.login}>
                  Ingresar
                  <ArrowRightIcon aria-hidden="true" />
                </Link>
              </Button>
              <Button asChild variant="outline">
                <Link to={RUTAS.registro}>Registrarme</Link>
              </Button>
            </>
          )}
        </div>
      </section>

      <section className="mt-12 grid gap-4 md:grid-cols-3">
        {PASOS.map((paso) => (
          <Card key={paso.titulo}>
            <CardHeader>
              <div className="mb-2 flex size-10 items-center justify-center rounded-lg bg-muted text-muted-foreground">
                <paso.icono className="size-5" aria-hidden="true" />
              </div>
              <CardTitle>{paso.titulo}</CardTitle>
            </CardHeader>
            <CardContent>
              <CardDescription>{paso.detalle}</CardDescription>
            </CardContent>
          </Card>
        ))}
      </section>
    </div>
  )
}
