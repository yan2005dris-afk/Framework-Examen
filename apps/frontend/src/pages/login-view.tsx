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
import { leerDestinoLogin, resolverDestinoLogin, RUTAS } from "@/lib/rutas"
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

  // Vencimiento de bloqueo persistido para soportar recarga de página
  const [tiempoBloqueo, setTiempoBloqueo] = useState<number | undefined>(() => {
    try {
      const guardado = sessionStorage.getItem("biess_auth_lockout")
      if (guardado) {
        const timestamp = Number(guardado)
        if (timestamp > Date.now()) {
          return timestamp
        }
        sessionStorage.removeItem("biess_auth_lockout")
      }
    } catch {
      // ignorar en entornos sin sessionStorage
    }
    return undefined
  })
  const [bloqueado, setBloqueado] = useState(Boolean(tiempoBloqueo))

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
      const rutaDestino = resolverDestinoLogin(destino, respuesta.postulante)
      navegar(rutaDestino, { replace: true })
    } catch (err: unknown) {
      if (esErrorApi(err)) {
        if (err.esBloqueoIntentos()) {
          const expira = Date.now() + 10 * 60 * 1000
          try {
            sessionStorage.setItem("biess_auth_lockout", expira.toString())
          } catch {
            // ignorar
          }
          setTiempoBloqueo(expira)
          setBloqueado(true)
          setErrorGeneral(
            err.mensaje ||
              "registro fallido, inténtelo nuevamente en 10 minutos",
          )
        } else if (err.esNoAutorizado()) {
          // El conteo de intentos restantes solo se muestra si lo informa el servidor
          const detalle = err.detalles?.length
            ? ` (${err.detalles.join(", ")})`
            : ""
          setErrorGeneral(
            `${err.mensaje || "sus credenciales de ingreso son incorrectas"}${detalle}`,
          )
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
    <div className="mx-auto w-full max-w-[46rem] space-y-6 px-5 pt-8 pb-16">
      <Card className="w-full">
        <CardHeader className="space-y-1">
          <CardTitle>
            <h1 className="text-base font-bold">Inicio de sesión</h1>
          </CardTitle>
          <CardDescription>
            Ingrese con la cédula y la contraseña de su registro.
          </CardDescription>
        </CardHeader>
        <form onSubmit={manejarSubmit} noValidate>
          <CardContent className="space-y-6 pb-5">
            {bloqueado ? (
              <AlertaBloqueo
                expiraEn={tiempoBloqueo}
                mensaje="registro fallido, inténtelo nuevamente en 10 minutos"
                onExpirar={() => {
                  setBloqueado(false)
                  setTiempoBloqueo(undefined)
                  setErrorGeneral(null)
                  try {
                    sessionStorage.removeItem("biess_auth_lockout")
                  } catch {
                    // ignorar
                  }
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
              <Label htmlFor="password">Contraseña</Label>
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
              <p className="text-xs text-muted-foreground">
                Su contraseña nunca se muestra ni se almacena en claro.
              </p>
            </div>

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
                "Ingresar al sistema"
              )}
            </Button>
            <p className="text-center text-[13px]">
              <Link
                to={RUTAS.recuperarContrasena}
                className="text-primary underline underline-offset-4"
              >
                ¿Olvidaste tu contraseña?
              </Link>
            </p>
          </CardContent>
        </form>
        <CardFooter className="flex flex-wrap justify-end gap-2">
          <span className="text-[13px] text-muted-foreground">
            ¿Aún no tienes cuenta?
          </span>
          <Button asChild variant="outline" size="sm">
            <Link to={RUTAS.registro}>Registrarse</Link>
          </Button>
        </CardFooter>
      </Card>
      <aside className="flex items-start gap-3 rounded-lg border border-primary/15 bg-brand-100 p-4 text-sm">
        <ShieldCheckIcon
          className="mt-0.5 size-5 shrink-0 text-primary"
          aria-hidden="true"
        />
        <div>
          <h2 className="font-semibold text-primary">Sesión segura</h2>
          <p className="mt-1 text-[13px] text-muted-foreground">
            Su sesión está protegida. No comparta su contraseña ni los códigos
            de verificación enviados a su correo.
          </p>
        </div>
      </aside>
    </div>
  )
}
