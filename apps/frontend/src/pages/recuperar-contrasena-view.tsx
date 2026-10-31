import { useState, type FormEvent } from "react"
import {
  CheckCircle2Icon,
  ChevronLeftIcon,
  EyeIcon,
  EyeOffIcon,
  KeyRoundIcon,
  Loader2Icon,
  ShieldCheckIcon,
} from "lucide-react"
import { Link, useNavigate } from "react-router"
import { toast } from "sonner"

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
import { MedidorPassword } from "@/features/auth/medidor-password"
import { esErrorApi } from "@/lib/api-error"
import {
  restablecerPassword,
  solicitarRecuperacionPassword,
} from "@/lib/clientes/auth"
import { RUTAS } from "@/lib/rutas"
import {
  evaluarFortalezaPassword,
  validarCedulaEcuatoriana,
} from "@/utils/validators"

type PasoRecuperacion = 1 | 2 | 3

export function RecuperarContrasenaView() {
  const navegar = useNavigate()

  const [paso, setPaso] = useState<PasoRecuperacion>(1)

  // Datos Paso 1
  const [cedula, setCedula] = useState("")
  const [errorCedula, setErrorCedula] = useState<string | null>(null)

  // Datos Paso 2
  const [codigoOtp, setCodigoOtp] = useState("")
  const [password, setPassword] = useState("")
  const [confirmarPassword, setConfirmarPassword] = useState("")
  const [mostrarPassword, setMostrarPassword] = useState(false)

  const [errorOtp, setErrorOtp] = useState<string | null>(null)
  const [errorPassword, setErrorPassword] = useState<string | null>(null)
  const [errorConfirmar, setErrorConfirmar] = useState<string | null>(null)

  // Estados transversales
  const [errorGeneral, setErrorGeneral] = useState<string | null>(null)
  const [cargando, setCargando] = useState(false)
  const [reenviandoOtp, setReenviandoOtp] = useState(false)
  const [bloqueado, setBloqueado] = useState(false)

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

  // Envío Paso 1: Solicitar código OTP al correo registrado
  async function manejarSubmitPaso1(e: FormEvent) {
    e.preventDefault()
    if (bloqueado || cargando) return

    const valCedula = validarCedulaEcuatoriana(cedula)
    if (!valCedula.esValida) {
      setErrorCedula("la información ingresada es incorrecta")
      return
    }

    setErrorCedula(null)
    setErrorGeneral(null)
    setCargando(true)

    try {
      const res = await solicitarRecuperacionPassword({ cedula: cedula.trim() })
      setPaso(2)
      toast.info(
        res.mensaje ||
          "Si la cédula está registrada, se ha enviado un código de recuperación a su correo.",
      )
    } catch (err: unknown) {
      if (esErrorApi(err)) {
        if (err.esBloqueoIntentos()) {
          setBloqueado(true)
          setErrorGeneral(
            err.mensaje ||
              "registro fallido, inténtelo nuevamente en 10 minutos",
          )
        } else {
          // Confirmación genérica sin revelar si la cédula existe
          setPaso(2)
          toast.info(
            "Si la cédula está registrada, se ha enviado un código de recuperación a su correo.",
          )
        }
      } else {
        setErrorGeneral("No fue posible conectar con el servidor.")
      }
    } finally {
      setCargando(false)
    }
  }

  // Validaciones Paso 2
  function validarCamposPaso2(): boolean {
    let valido = true

    if (!codigoOtp.trim() || codigoOtp.trim().length !== 6) {
      setErrorOtp("El código de verificación debe tener 6 dígitos.")
      valido = false
    } else {
      setErrorOtp(null)
    }

    const evaluacionPass = evaluarFortalezaPassword(password)
    if (
      !evaluacionPass.cumpleLongitud ||
      !evaluacionPass.cumpleMayuscula ||
      !evaluacionPass.cumpleMinuscula ||
      !evaluacionPass.cumpleNumero ||
      !evaluacionPass.cumpleEspecial
    ) {
      setErrorPassword(
        "La contraseña no cumple con todos los requisitos de seguridad.",
      )
      valido = false
    } else {
      setErrorPassword(null)
    }

    if (password !== confirmarPassword) {
      setErrorConfirmar("Las contraseñas no coinciden.")
      valido = false
    } else {
      setErrorConfirmar(null)
    }

    return valido
  }

  // Envío Paso 2: Validar OTP y cambiar contraseña
  async function manejarSubmitPaso2(e: FormEvent) {
    e.preventDefault()
    if (bloqueado || cargando) return

    if (!validarCamposPaso2()) {
      return
    }

    setErrorGeneral(null)
    setCargando(true)

    try {
      await restablecerPassword({
        cedula: cedula.trim(),
        codigo_otp: codigoOtp.trim(),
        new_password: password,
      })
      setPaso(3)
    } catch (err: unknown) {
      if (esErrorApi(err)) {
        if (err.esBloqueoIntentos()) {
          setBloqueado(true)
          setErrorGeneral(
            err.mensaje ||
              "registro fallido, inténtelo nuevamente en 10 minutos",
          )
        } else if (err.codigo === "otp_invalido" || err.esNoAutorizado()) {
          const mensajeOtp =
            err.mensaje ||
            "El código de verificación es incorrecto o ha expirado."
          setErrorOtp(mensajeOtp)
          setErrorGeneral(mensajeOtp)
        } else {
          setErrorGeneral(
            err.mensaje || "Código de verificación incorrecto o expirado.",
          )
        }
      } else {
        setErrorGeneral("No fue posible conectar con el servidor.")
      }
    } finally {
      setCargando(false)
    }
  }

  async function manejarReenviarOtp() {
    if (bloqueado || reenviandoOtp || cargando) return
    setReenviandoOtp(true)
    setErrorGeneral(null)

    try {
      const res = await solicitarRecuperacionPassword({ cedula: cedula.trim() })
      toast.success(
        res.mensaje || "Código de recuperación reenviado a su correo.",
      )
    } catch (err: unknown) {
      if (esErrorApi(err)) {
        if (err.esBloqueoIntentos()) {
          setBloqueado(true)
          setErrorGeneral(
            err.mensaje ||
              "registro fallido, inténtelo nuevamente en 10 minutos",
          )
        } else {
          toast.info(
            "Si la cédula está registrada, se ha enviado el código de recuperación.",
          )
        }
      } else {
        setErrorGeneral("No fue posible conectar con el servidor.")
      }
    } finally {
      setReenviandoOtp(false)
    }
  }

  return (
    <div className="mx-auto flex min-h-[calc(100svh-4rem)] w-full max-w-md items-center px-4 py-8">
      <Card className="w-full shadow-lg">
        <CardHeader className="space-y-1 text-center">
          <div className="mx-auto mb-2 flex size-12 items-center justify-center rounded-xl bg-primary/10 text-primary">
            {paso === 3 ? (
              <CheckCircle2Icon
                className="size-6 text-emerald-600"
                aria-hidden="true"
              />
            ) : (
              <KeyRoundIcon className="size-6" aria-hidden="true" />
            )}
          </div>
          <CardTitle className="text-2xl font-bold tracking-tight">
            {paso === 3 ? "¡Contraseña actualizada!" : "Recuperar contraseña"}
          </CardTitle>
          <CardDescription>
            {paso === 1 &&
              "Ingrese su número de cédula para recibir el código de seguridad"}
            {paso === 2 &&
              "Ingrese el código recibido y defina su nueva contraseña"}
            {paso === 3 && "Su contraseña se actualizó correctamente"}
          </CardDescription>
        </CardHeader>

        {/* PASO 1: Ingreso de Cédula */}
        {paso === 1 && (
          <form onSubmit={manejarSubmitPaso1} noValidate>
            <CardContent className="space-y-4">
              {bloqueado ? (
                <AlertaBloqueo
                  segundosIniciales={600}
                  mensaje="registro fallido, inténtelo nuevamente en 10 minutos"
                  onExpirar={() => {
                    setBloqueado(false)
                    setErrorGeneral(null)
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
                <Label htmlFor="cedula-rec">Cédula de identidad</Label>
                <Input
                  id="cedula-rec"
                  name="cedula"
                  type="text"
                  inputMode="numeric"
                  maxLength={10}
                  placeholder="Ej. 1710034008"
                  value={cedula}
                  disabled={cargando || bloqueado}
                  aria-invalid={Boolean(errorCedula)}
                  aria-describedby={
                    errorCedula ? "cedula-rec-error" : undefined
                  }
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
                  <p id="cedula-rec-error" className="text-xs text-destructive">
                    {errorCedula}
                  </p>
                )}
                <p className="text-xs text-muted-foreground">
                  El código de seguridad se enviará a la dirección de correo
                  registrada con esta cédula.
                </p>
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
                    Enviando código…
                  </>
                ) : (
                  "Enviar código de recuperación"
                )}
              </Button>

              <Button
                type="button"
                variant="ghost"
                asChild
                className="w-full text-xs"
              >
                <Link to={RUTAS.login}>
                  <ChevronLeftIcon className="mr-1 size-3.5" />
                  Volver al inicio de sesión
                </Link>
              </Button>
            </CardFooter>
          </form>
        )}

        {/* PASO 2: OTP y Nueva Contraseña */}
        {paso === 2 && (
          <form onSubmit={manejarSubmitPaso2} noValidate>
            <CardContent className="space-y-4">
              <div className="rounded-lg bg-muted/60 p-3 text-xs text-muted-foreground">
                Código de seguridad enviado al correo vinculado a la cédula{" "}
                <span className="font-mono font-medium text-foreground">
                  {cedula}
                </span>
                .
              </div>

              {bloqueado ? (
                <AlertaBloqueo
                  segundosIniciales={600}
                  mensaje="registro fallido, inténtelo nuevamente en 10 minutos"
                  onExpirar={() => {
                    setBloqueado(false)
                    setErrorGeneral(null)
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
                <div className="flex items-center justify-between">
                  <Label htmlFor="otp-rec">
                    Código de seguridad (6 dígitos)
                  </Label>
                  <button
                    type="button"
                    onClick={() => {
                      void manejarReenviarOtp()
                    }}
                    disabled={cargando || reenviandoOtp || bloqueado}
                    className="text-xs font-medium text-primary hover:underline disabled:opacity-50"
                  >
                    {reenviandoOtp
                      ? "Reenviando…"
                      : "¿No recibió el código? Reenviar"}
                  </button>
                </div>
                <Input
                  id="otp-rec"
                  name="otp"
                  type="text"
                  inputMode="numeric"
                  maxLength={6}
                  placeholder="000000"
                  value={codigoOtp}
                  disabled={cargando || bloqueado}
                  className="text-center font-mono text-lg tracking-widest"
                  aria-invalid={Boolean(errorOtp)}
                  aria-describedby={errorOtp ? "otp-rec-error" : undefined}
                  onChange={(e) => {
                    const limpio = e.target.value.replace(/\D/g, "")
                    setCodigoOtp(limpio)
                    if (errorOtp) setErrorOtp(null)
                  }}
                  required
                />
                {errorOtp && (
                  <p id="otp-rec-error" className="text-xs text-destructive">
                    {errorOtp}
                  </p>
                )}
              </div>

              <div className="space-y-2">
                <Label htmlFor="new-password">Nueva contraseña</Label>
                <div className="relative">
                  <Input
                    id="new-password"
                    name="new-password"
                    type={mostrarPassword ? "text" : "password"}
                    placeholder="Ingrese su nueva contraseña"
                    value={password}
                    disabled={cargando || bloqueado}
                    aria-invalid={Boolean(errorPassword)}
                    aria-describedby={
                      errorPassword ? "new-password-error" : undefined
                    }
                    onChange={(e) => {
                      setPassword(e.target.value)
                      if (errorPassword) setErrorPassword(null)
                    }}
                    autoComplete="new-password"
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
                  <p
                    id="new-password-error"
                    className="text-xs text-destructive"
                  >
                    {errorPassword}
                  </p>
                )}

                <MedidorPassword password={password} />
              </div>

              <div className="space-y-2">
                <Label htmlFor="confirm-new-password">
                  Confirmar nueva contraseña
                </Label>
                <Input
                  id="confirm-new-password"
                  name="confirm-new-password"
                  type={mostrarPassword ? "text" : "password"}
                  placeholder="Repita la nueva contraseña"
                  value={confirmarPassword}
                  disabled={cargando || bloqueado}
                  aria-invalid={Boolean(errorConfirmar)}
                  aria-describedby={
                    errorConfirmar ? "confirm-new-error" : undefined
                  }
                  onChange={(e) => {
                    setConfirmarPassword(e.target.value)
                    if (errorConfirmar) setErrorConfirmar(null)
                  }}
                  autoComplete="new-password"
                  required
                />
                {errorConfirmar && (
                  <p
                    id="confirm-new-error"
                    className="text-xs text-destructive"
                  >
                    {errorConfirmar}
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
                    Restableciendo contraseña…
                  </>
                ) : (
                  "Guardar nueva contraseña"
                )}
              </Button>

              <Button
                type="button"
                variant="ghost"
                onClick={() => setPaso(1)}
                disabled={cargando || bloqueado}
                className="w-full text-xs"
              >
                <ChevronLeftIcon className="mr-1 size-3.5" />
                Volver al paso anterior
              </Button>
            </CardFooter>
          </form>
        )}

        {/* PASO 3: Éxito */}
        {paso === 3 && (
          <CardContent className="space-y-6 py-4 text-center">
            <div className="mx-auto flex size-16 items-center justify-center rounded-full bg-emerald-100 text-emerald-600 dark:bg-emerald-950/50 dark:text-emerald-400">
              <ShieldCheckIcon className="size-8" />
            </div>

            <div className="space-y-2">
              <h3 className="text-lg font-semibold">
                Contraseña restablecida con éxito
              </h3>
              <p className="text-sm text-muted-foreground">
                Su contraseña ha sido actualizada en el sistema. Ya puede
                ingresar con sus nuevas credenciales.
              </p>
            </div>

            <div className="pt-2">
              <Button
                onClick={() => navegar(RUTAS.login, { replace: true })}
                className="w-full"
              >
                Ir a iniciar sesión
              </Button>
            </div>
          </CardContent>
        )}
      </Card>
    </div>
  )
}
