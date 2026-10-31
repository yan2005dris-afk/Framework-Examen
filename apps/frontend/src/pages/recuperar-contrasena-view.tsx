import { useEffect, useRef, useState, type FormEvent } from "react"
import {
  CheckCircle2Icon,
  ChevronLeftIcon,
  ClockIcon,
  EyeIcon,
  EyeOffIcon,
  KeyRoundIcon,
  Loader2Icon,
  MailIcon,
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
  type DesafioOtp,
} from "@/lib/clientes/auth"
import { RUTAS } from "@/lib/rutas"
import {
  evaluarFortalezaPassword,
  validarCedulaEcuatoriana,
} from "@/utils/validators"

/** Pasos de recuperación alineados con los mockups 04a, 04b, 04c y 04d:
 * 1: Identidad (cédula)
 * 2: Correo / Código OTP
 * 3: Nueva contraseña
 * 4: Listo / Éxito
 */
type PasoRecuperacion = 1 | 2 | 3 | 4

export function RecuperarContrasenaView() {
  const navegar = useNavigate()

  const [paso, setPaso] = useState<PasoRecuperacion>(1)

  // ── Paso 1: Cédula ───────────────────────────────────────────────────────
  const [cedula, setCedula] = useState("")
  const [errorCedula, setErrorCedula] = useState<string | null>(null)

  // ── Paso 2: OTP ──────────────────────────────────────────────────────────
  const [desafio, setDesafio] = useState<DesafioOtp | null>(null)
  const [codigoOtp, setCodigoOtp] = useState("")
  const [errorOtp, setErrorOtp] = useState<string | null>(null)
  const [reenvioDisponibleEn, setReenvioDisponibleEn] = useState(0)
  const reenvioTimerRef = useRef<ReturnType<typeof setInterval> | null>(null)

  // ── Paso 3: Nueva Contraseña ─────────────────────────────────────────────
  const [password, setPassword] = useState("")
  const [confirmarPassword, setConfirmarPassword] = useState("")
  const [mostrarPassword, setMostrarPassword] = useState(false)
  const [errorPassword, setErrorPassword] = useState<string | null>(null)
  const [errorConfirmar, setErrorConfirmar] = useState<string | null>(null)

  // ── Estados transversales ────────────────────────────────────────────────
  const [errorGeneral, setErrorGeneral] = useState<string | null>(null)
  const [cargando, setCargando] = useState(false)
  const [reenviandoOtp, setReenviandoOtp] = useState(false)
  const [bloqueadoExpiraEn, setBloqueadoExpiraEn] = useState<Date | null>(null)

  const estaBloqueado = bloqueadoExpiraEn !== null

  useEffect(() => {
    return () => {
      if (reenvioTimerRef.current) clearInterval(reenvioTimerRef.current)
    }
  }, [])

  function iniciarCooldownReenvio(segundos: number) {
    setReenvioDisponibleEn(segundos)
    if (reenvioTimerRef.current) clearInterval(reenvioTimerRef.current)
    reenvioTimerRef.current = setInterval(() => {
      setReenvioDisponibleEn((prev) => {
        if (prev <= 1) {
          clearInterval(reenvioTimerRef.current!)
          return 0
        }
        return prev - 1
      })
    }, 1000)
  }

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

  // ── Envío Paso 1: Solicitar código OTP ───────────────────────────────────
  async function manejarSubmitPaso1(e: FormEvent) {
    e.preventDefault()
    if (estaBloqueado || cargando) return

    const valCedula = validarCedulaEcuatoriana(cedula)
    if (!valCedula.esValida) {
      setErrorCedula("la información ingresada es incorrecta")
      return
    }

    setErrorCedula(null)
    setErrorGeneral(null)
    setCargando(true)

    try {
      const challenge = await solicitarRecuperacionPassword({ cedula: cedula.trim() })
      setDesafio(challenge)
      iniciarCooldownReenvio(60)
      setPaso(2)
      toast.info(
        challenge.mensaje ||
          "Si la cédula está registrada, se ha enviado un código de recuperación a su correo.",
      )
    } catch (err: unknown) {
      if (esErrorApi(err)) {
        if (err.esBloqueoIntentos()) {
          setBloqueadoExpiraEn(err.retryAt ?? new Date(Date.now() + 600_000))
          setErrorGeneral(err.mensaje || "registro fallido, intente nuevamente en 10 minutos")
        } else if (err.esReenvioLimitado()) {
          const espera = err.retryAfterSeg ?? 60
          iniciarCooldownReenvio(espera)
          setPaso(2)
        } else {
          // Confirmación genérica sin revelar si la cédula existe (anti-oráculo)
          setPaso(2)
          iniciarCooldownReenvio(60)
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

  // ── Reenviar OTP (Paso 2) ────────────────────────────────────────────────
  async function manejarReenviarOtp() {
    if (estaBloqueado || reenviandoOtp || cargando || reenvioDisponibleEn > 0) return
    setReenviandoOtp(true)
    setErrorGeneral(null)

    try {
      const challenge = await solicitarRecuperacionPassword({ cedula: cedula.trim() })
      setDesafio(challenge)
      iniciarCooldownReenvio(60)
      toast.success("Código de recuperación reenviado a su correo.")
    } catch (err: unknown) {
      if (esErrorApi(err)) {
        if (err.esBloqueoIntentos()) {
          setBloqueadoExpiraEn(err.retryAt ?? new Date(Date.now() + 600_000))
          setErrorGeneral(err.mensaje || "registro fallido, intente nuevamente en 10 minutos")
        } else if (err.esReenvioLimitado()) {
          const espera = err.retryAfterSeg ?? 60
          iniciarCooldownReenvio(espera)
          setErrorGeneral(`Espere ${espera} segundos antes de reenviar el código.`)
        } else {
          setErrorGeneral(err.mensaje || "No fue posible reenviar el código de recuperación.")
        }
      } else {
        setErrorGeneral("No fue posible conectar con el servidor.")
      }
    } finally {
      setReenviandoOtp(false)
    }
  }

  // ── Paso 2: Verificar código OTP ─────────────────────────────────────────
  function manejarSubmitPaso2(e: FormEvent) {
    e.preventDefault()
    if (estaBloqueado || cargando) return

    if (!codigoOtp.trim() || codigoOtp.trim().length !== 6) {
      setErrorOtp("El código de verificación debe tener 6 dígitos.")
      return
    }

    setErrorOtp(null)
    setErrorGeneral(null)
    // Pasamos a configurar la nueva contraseña
    setPaso(3)
  }

  // ── Validaciones Paso 3 ──────────────────────────────────────────────────
  function validarCamposPaso3(): boolean {
    let valido = true

    const evaluacionPass = evaluarFortalezaPassword(password)
    if (
      !evaluacionPass.cumpleLongitud ||
      !evaluacionPass.cumpleMayuscula ||
      !evaluacionPass.cumpleMinuscula ||
      !evaluacionPass.cumpleNumero ||
      !evaluacionPass.cumpleEspecial
    ) {
      setErrorPassword("La contraseña no cumple con todos los requisitos de seguridad.")
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

  // ── Envío Paso 3: Restablecer Contraseña ──────────────────────────────────
  async function manejarSubmitPaso3(e: FormEvent) {
    e.preventDefault()
    if (estaBloqueado || cargando) return

    if (!validarCamposPaso3()) return

    setErrorGeneral(null)
    setCargando(true)

    try {
      await restablecerPassword({
        cedula: cedula.trim(),
        codigo_otp: codigoOtp.trim(),
        new_password: password,
      })

      setPaso(4)
      toast.success("Contraseña actualizada correctamente.")
    } catch (err: unknown) {
      if (esErrorApi(err)) {
        if (err.esBloqueoIntentos()) {
          setBloqueadoExpiraEn(err.retryAt ?? new Date(Date.now() + 600_000))
          setErrorGeneral(
            err.mensaje || "registro fallido, intente nuevamente en 10 minutos",
          )
        } else if (err.codigo === "otp_invalido" || err.esNoAutorizado()) {
          const mensajeOtp =
            err.mensaje || "El código de verificación es incorrecto o ha expirado."
          setErrorOtp(mensajeOtp)
          setErrorGeneral(mensajeOtp)
          setPaso(2)
        } else if (err.codigo === "password_debil") {
          setErrorPassword(err.mensaje || "La contraseña ingresada no es suficientemente segura.")
        } else {
          setErrorGeneral(
            err.mensaje || "No fue posible restablecer la contraseña.",
          )
        }
      } else {
        setErrorGeneral("No fue posible conectar con el servidor.")
      }
    } finally {
      setCargando(false)
    }
  }

  return (
    <div className="mx-auto flex min-h-[calc(100svh-4rem)] w-full max-w-lg items-center px-4 py-8">
      <Card className="w-full shadow-lg">
        <CardHeader className="space-y-1 text-center">
          <div className="mx-auto mb-2 flex size-12 items-center justify-center rounded-xl bg-primary/10 text-primary">
            {paso === 4 ? (
              <CheckCircle2Icon
                className="size-6 text-emerald-600"
                aria-hidden="true"
              />
            ) : (
              <KeyRoundIcon className="size-6" aria-hidden="true" />
            )}
          </div>
          <CardTitle className="text-2xl font-bold tracking-tight">
            {paso === 4 ? "Contraseña restablecida" : "Recuperar contraseña"}
          </CardTitle>
          <CardDescription>
            {paso === 1 && "Paso 1 de 3: Confirme su identidad ingresando su cédula"}
            {paso === 2 && "Paso 2 de 3: Ingrese el código enviado a su correo registrado"}
            {paso === 3 && "Paso 3 de 3: Defina su nueva contraseña de acceso"}
            {paso === 4 && "Su contraseña ha sido actualizada exitosamente"}
          </CardDescription>

          {paso < 4 && (
            <div className="mt-4 flex items-center justify-center gap-2 pt-2">
              {[1, 2, 3].map((n) => (
                <span
                  key={n}
                  className={`flex size-6 items-center justify-center rounded-full text-xs font-semibold ${
                    paso === n
                      ? "bg-primary text-primary-foreground"
                      : paso > n
                        ? "bg-primary/40 text-primary-foreground"
                        : "bg-muted text-muted-foreground"
                  }`}
                >
                  {n}
                </span>
              ))}
            </div>
          )}
        </CardHeader>

        {/* PASO 1: Cédula */}
        {paso === 1 && (
          <form onSubmit={manejarSubmitPaso1} noValidate>
            <CardContent className="space-y-4">
              {estaBloqueado ? (
                <AlertaBloqueo
                  expiraEn={bloqueadoExpiraEn ?? undefined}
                  mensaje="registro fallido, intente nuevamente en 10 minutos"
                  onExpirar={() => {
                    setBloqueadoExpiraEn(null)
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
                <Label htmlFor="cedula-recuperar">Cédula de ciudadanía *</Label>
                <Input
                  id="cedula-recuperar"
                  name="cedula"
                  type="text"
                  inputMode="numeric"
                  maxLength={10}
                  placeholder="Ej. 1712345675"
                  value={cedula}
                  disabled={cargando || estaBloqueado}
                  aria-invalid={Boolean(errorCedula)}
                  aria-describedby={
                    errorCedula ? "cedula-recuperar-error" : "cedula-recuperar-hint"
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
                <p className="text-xs text-muted-foreground" id="cedula-recuperar-hint">
                  La cédula con la que se registró en el portal.
                </p>
                {errorCedula && (
                  <p
                    id="cedula-recuperar-error"
                    className="text-xs text-destructive"
                  >
                    {errorCedula}
                  </p>
                )}
              </div>
            </CardContent>

            <CardFooter className="flex flex-col gap-3">
              <Button
                type="submit"
                className="w-full"
                disabled={cargando || estaBloqueado}
              >
                {cargando ? (
                  <>
                    <Loader2Icon className="mr-2 size-4 animate-spin" />
                    Enviando código…
                  </>
                ) : (
                  "Continuar"
                )}
              </Button>

              <p className="text-center text-xs text-muted-foreground">
                <Link
                  to={RUTAS.login}
                  className="font-medium text-primary hover:underline"
                >
                  ← Volver al inicio de sesión
                </Link>
              </p>
            </CardFooter>
          </form>
        )}

        {/* PASO 2: Código OTP */}
        {paso === 2 && (
          <form onSubmit={manejarSubmitPaso2} noValidate>
            <CardContent className="space-y-4">
              <div className="rounded-lg border bg-muted/40 p-3.5 text-xs">
                <div className="flex items-center gap-2 font-medium text-foreground">
                  <MailIcon className="size-4 text-primary shrink-0" />
                  <span>Código de recuperación enviado</span>
                </div>
                <p className="mt-1 text-muted-foreground">
                  Si la cédula <strong>{cedula}</strong> corresponde a una cuenta registrada,
                  enviamos un código de 6 dígitos a su correo electrónico.
                </p>
              </div>

              {desafio && (
                <p className="text-xs text-muted-foreground">
                  El código expira a las{" "}
                  <strong>
                    {desafio.expiraEn.toLocaleTimeString([], {
                      hour: "2-digit",
                      minute: "2-digit",
                    })}
                  </strong>
                  .
                </p>
              )}

              {estaBloqueado ? (
                <AlertaBloqueo
                  expiraEn={bloqueadoExpiraEn ?? undefined}
                  mensaje="registro fallido, intente nuevamente en 10 minutos"
                  onExpirar={() => {
                    setBloqueadoExpiraEn(null)
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
                  <Label htmlFor="otp-recuperar">
                    Código de verificación (6 dígitos) *
                  </Label>
                  <button
                    type="button"
                    onClick={() => { void manejarReenviarOtp() }}
                    disabled={cargando || reenviandoOtp || estaBloqueado || reenvioDisponibleEn > 0}
                    className="flex items-center gap-1 text-xs font-medium text-primary hover:underline disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    {reenviandoOtp ? (
                      <>
                        <Loader2Icon className="size-3 animate-spin" />
                        Reenviando…
                      </>
                    ) : reenvioDisponibleEn > 0 ? (
                      <>
                        <ClockIcon className="size-3" />
                        Reenviar ({reenvioDisponibleEn}s)
                      </>
                    ) : (
                      "Reenviar código"
                    )}
                  </button>
                </div>
                <Input
                  id="otp-recuperar"
                  name="otp"
                  type="text"
                  inputMode="numeric"
                  maxLength={6}
                  placeholder="000000"
                  value={codigoOtp}
                  disabled={cargando || estaBloqueado}
                  className="text-center font-mono text-lg tracking-widest"
                  aria-invalid={Boolean(errorOtp)}
                  aria-describedby={errorOtp ? "otp-recuperar-error" : undefined}
                  onChange={(e) => {
                    const limpio = e.target.value.replace(/\D/g, "")
                    setCodigoOtp(limpio)
                    if (errorOtp) setErrorOtp(null)
                  }}
                  required
                />
                {errorOtp && (
                  <p id="otp-recuperar-error" className="text-xs text-destructive">
                    {errorOtp}
                  </p>
                )}
              </div>
            </CardContent>

            <CardFooter className="flex flex-col gap-3">
              <Button
                type="submit"
                className="w-full"
                disabled={cargando || estaBloqueado}
              >
                Continuar
              </Button>

              <Button
                type="button"
                variant="ghost"
                onClick={() => setPaso(1)}
                disabled={cargando || estaBloqueado}
                className="w-full text-xs"
              >
                <ChevronLeftIcon className="mr-1 size-3.5" />
                Volver al paso anterior
              </Button>
            </CardFooter>
          </form>
        )}

        {/* PASO 3: Nueva contraseña */}
        {paso === 3 && (
          <form onSubmit={manejarSubmitPaso3} noValidate>
            <CardContent className="space-y-4">
              {estaBloqueado ? (
                <AlertaBloqueo
                  expiraEn={bloqueadoExpiraEn ?? undefined}
                  mensaje="registro fallido, intente nuevamente en 10 minutos"
                  onExpirar={() => {
                    setBloqueadoExpiraEn(null)
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
                <Label htmlFor="nueva-password">Nueva contraseña *</Label>
                <div className="relative">
                  <Input
                    id="nueva-password"
                    name="password"
                    type={mostrarPassword ? "text" : "password"}
                    placeholder="Mínimo 8 caracteres"
                    value={password}
                    disabled={cargando || estaBloqueado}
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
                    disabled={cargando || estaBloqueado}
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
                  Confirme la nueva contraseña *
                </Label>
                <Input
                  id="confirm-new-password"
                  name="confirm-new-password"
                  type={mostrarPassword ? "text" : "password"}
                  placeholder="Repita la nueva contraseña"
                  value={confirmarPassword}
                  disabled={cargando || estaBloqueado}
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
                disabled={cargando || estaBloqueado}
              >
                {cargando ? (
                  <>
                    <Loader2Icon className="mr-2 size-4 animate-spin" />
                    Guardando nueva contraseña…
                  </>
                ) : (
                  "Cambiar contraseña"
                )}
              </Button>

              <Button
                type="button"
                variant="ghost"
                onClick={() => setPaso(2)}
                disabled={cargando || estaBloqueado}
                className="w-full text-xs"
              >
                <ChevronLeftIcon className="mr-1 size-3.5" />
                Volver al paso anterior
              </Button>
            </CardFooter>
          </form>
        )}

        {/* PASO 4: Éxito */}
        {paso === 4 && (
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
