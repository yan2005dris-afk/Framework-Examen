import { useState, type FormEvent } from "react"
import { EyeIcon, EyeOffIcon, Loader2Icon, ShieldCheckIcon } from "lucide-react"
import { Link, useNavigate, useSearchParams } from "react-router"

import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { AlertaBloqueo } from "@/features/auth/alerta-bloqueo"
import { useSesion } from "@/features/auth/sesion-context"
import { esErrorApi } from "@/lib/api-error"
import { iniciarSesion } from "@/lib/clientes/auth"
import { leerDestinoLogin, rutaInicial, RUTAS } from "@/lib/rutas"
import { validarCedulaEcuatoriana } from "@/utils/validators"

export function LoginView() {
  const [parametros] = useSearchParams()
  const destino = leerDestinoLogin(parametros.toString(), "")
  const navegar = useNavigate()
  const { iniciarSesionCon } = useSesion()

  const [cedula, setCedula] = useState("")
  const [password, setPassword] = useState("")
  const [mostrarPassword, setMostrarPassword] = useState(false)

  const [errorCedula, setErrorCedula] = useState<string | null>(null)
  const [errorPassword, setErrorPassword] = useState<string | null>(null)
  const [errorGeneral, setErrorGeneral] = useState<string | null>(null)

  const [cargando, setCargando] = useState(false)
  const [bloqueado, setBloqueado] = useState(false)
  const [intentosFallidos, setIntentosFallidos] = useState(0)

  function manejarBlurCedula() {
    if (!cedula.trim()) {
      setErrorCedula("Ingrese su número de cédula.")
      return
    }
    const val = validarCedulaEcuatoriana(cedula)
    if (!val.esValida) {
      setErrorCedula("la información ingresada es incorrecta")
    } else {
      setErrorCedula(null)
    }
  }

  function manejarBlurPassword() {
    if (!password) {
      setErrorPassword("Ingrese su contraseña.")
    } else {
      setErrorPassword(null)
    }
  }

  async function manejarSubmit(e: FormEvent) {
    e.preventDefault()
    if (bloqueado || cargando) return

    // Validación completa previa
    const valCedula = validarCedulaEcuatoriana(cedula)
    if (!valCedula.esValida) {
      setErrorCedula("la información ingresada es incorrecta")
      return
    }
    if (!password) {
      setErrorPassword("Ingrese su contraseña.")
      return
    }

    setErrorCedula(null)
    setErrorPassword(null)
    setErrorGeneral(null)
    setCargando(true)

    try {
      const respuesta = await iniciarSesion({
        cedula: cedula.trim(),
        password,
      })

      iniciarSesionCon(respuesta.postulante)
      const rutaDestino = destino || rutaInicial(respuesta.postulante)
      navegar(rutaDestino, { replace: true })
    } catch (err: unknown) {
      if (esErrorApi(err)) {
        if (err.esBloqueoIntentos()) {
          setBloqueado(true)
          setErrorGeneral(
            "registro fallido, inténtelo nuevamente en 10 minutos",
          )
        } else if (err.esNoAutorizado()) {
          const nuevosIntentos = intentosFallidos + 1
          setIntentosFallidos(nuevosIntentos)

          let mensaje = "sus credenciales de ingreso son incorrectas"
          if (nuevosIntentos >= 2) {
            const restantes = Math.max(0, 3 - nuevosIntentos)
            if (restantes > 0) {
              mensaje += `. Le queda ${restantes} intento${restantes > 1 ? "s" : ""}.`
            }
          }
          setErrorGeneral(mensaje)
        } else {
          setErrorGeneral(err.mensaje || "Ocurrió un error al iniciar sesión.")
        }
      } else {
        setErrorGeneral(
          "No fue posible conectar con el servidor. Verifique su conexión.",
        )
      }
    } finally {
      setCargando(false)
    }
  }

  return (
    <div className="mx-auto flex min-h-[calc(100svh-4rem)] w-full max-w-md items-center px-4 py-8">
      <Card className="w-full shadow-lg">
        <CardHeader className="space-y-1 text-center">
          <div className="mx-auto mb-2 flex size-12 items-center justify-center rounded-xl bg-primary/10 text-primary">
            <ShieldCheckIcon className="size-6" aria-hidden="true" />
          </div>
          <CardTitle className="text-2xl font-bold tracking-tight">
            Iniciar sesión
          </CardTitle>
          <CardDescription>
            Concurso de Méritos y Oposición — Directorio BIESS
          </CardDescription>
        </CardHeader>

        <form onSubmit={manejarSubmit} noValidate>
          <CardContent className="space-y-4">
            {bloqueado ? (
              <AlertaBloqueo
                segundosIniciales={600}
                mensaje="registro fallido, inténtelo nuevamente en 10 minutos"
                onExpirar={() => {
                  setBloqueado(false)
                  setErrorGeneral(null)
                  setIntentosFallidos(0)
                }}
              />
            ) : errorGeneral ? (
              <div
                role="alert"
                aria-live="assertive"
                className="rounded-lg border border-destructive/50 bg-destructive/10 p-3 text-xs font-medium text-destructive"
              >
                {errorGeneral}
              </div>
            ) : null}

            <div className="space-y-2">
              <Label htmlFor="cedula">Cédula de identidad</Label>
              <Input
                id="cedula"
                name="cedula"
                type="text"
                inputMode="numeric"
                maxLength={10}
                placeholder="Ej. 1710034008"
                value={cedula}
                disabled={cargando || bloqueado}
                aria-invalid={Boolean(errorCedula)}
                aria-describedby={errorCedula ? "cedula-error" : undefined}
                onChange={(e) => {
                  const limpio = e.target.value.replace(/\D/g, "")
                  setCedula(limpio)
                  if (errorCedula) setErrorCedula(null)
                }}
                onBlur={manejarBlurCedula}
                autoComplete="username"
                required
              />
              {errorCedula && (
                <p id="cedula-error" className="text-xs text-destructive">
                  {errorCedula}
                </p>
              )}
            </div>

            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <Label htmlFor="password">Contraseña</Label>
                <Link
                  to={RUTAS.recuperarContrasena}
                  className="text-xs font-medium text-primary hover:underline"
                  tabIndex={bloqueado ? -1 : 0}
                >
                  ¿Olvidaste tu contraseña?
                </Link>
              </div>
              <div className="relative">
                <Input
                  id="password"
                  name="password"
                  type={mostrarPassword ? "text" : "password"}
                  placeholder="Ingrese su contraseña"
                  value={password}
                  disabled={cargando || bloqueado}
                  aria-invalid={Boolean(errorPassword)}
                  aria-describedby={
                    errorPassword ? "password-error" : undefined
                  }
                  onChange={(e) => {
                    setPassword(e.target.value)
                    if (errorPassword) setErrorPassword(null)
                  }}
                  onBlur={manejarBlurPassword}
                  autoComplete="current-password"
                  required
                  className="pr-10"
                />
                <button
                  type="button"
                  onClick={() => setMostrarPassword(!mostrarPassword)}
                  disabled={cargando || bloqueado}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground focus:outline-none"
                  aria-label={
                    mostrarPassword ? "Ocultar contraseña" : "Ver contraseña"
                  }
                >
                  {mostrarPassword ? (
                    <EyeOffIcon className="size-4" />
                  ) : (
                    <EyeIcon className="size-4" />
                  )}
                </button>
              </div>
              {errorPassword && (
                <p id="password-error" className="text-xs text-destructive">
                  {errorPassword}
                </p>
              )}
            </div>
          </CardContent>

          <CardFooter className="flex flex-col gap-3">
            <Button
              type="submit"
              className="w-full"
              disabled={cargando || bloqueado}
            >
              {cargando ? (
                <>
                  <Loader2Icon className="mr-2 size-4 animate-spin" />
                  Iniciando sesión…
                </>
              ) : (
                "Iniciar sesión"
              )}
            </Button>

            <p className="text-center text-xs text-muted-foreground">
              ¿No tienes una cuenta aún?{" "}
              <Link
                to={RUTAS.registro}
                className="font-medium text-primary hover:underline"
              >
                Regístrate aquí
              </Link>
            </p>
          </CardFooter>
        </form>
      </Card>
    </div>
  )
}
