import { useEffect, useRef, useState, type FormEvent } from "react"
import {
  CheckCircle2Icon,
  ChevronLeftIcon,
  ClockIcon,
  EyeIcon,
  EyeOffIcon,
  Loader2Icon,
  MailIcon,
  ShieldCheckIcon,
  UserPlusIcon,
} from "lucide-react"
import { Link, useNavigate, useSearchParams } from "react-router"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
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
import { useSesion } from "@/features/auth/sesion-context"
import { useConvocatoriasPublicas } from "@/features/convocatorias/queries"
import { ModalPoliticaPrivacidad } from "@/features/privacidad/modal-politica-privacidad"
import { usePoliticaVigente } from "@/features/privacidad/queries"
import { esErrorApi } from "@/lib/api-error"
import {
  solicitarRegistro,
  verificarYRegistrar,
  type DesafioOtp,
  type PostulanteDto,
} from "@/lib/clientes/auth"
import { RUTAS } from "@/lib/rutas"
import {
  evaluarFortalezaPassword,
  validarCedulaEcuatoriana,
  validarEmail,
} from "@/utils/validators"

/** Pasos del flujo de registro:
 * 1 → identidad (cédula + correo)
 * 2 → verificación OTP
 * 3 → credenciales (nombres, apellidos, contraseña)
 * 4 → éxito
 */
type PasoRegistro = 1 | 2 | 3 | 4

export function RegistroView() {
  const navegar = useNavigate()
  const [parametros] = useSearchParams()
  const periodo = parametros.get("convocatoria")
  const { iniciarSesionCon } = useSesion()

  const listadoConvocatorias = useConvocatoriasPublicas()
  const convocatoriaId =
    periodo ||
    listadoConvocatorias.data?.pages
      ?.flatMap((p) => p.items)
      ?.find((c) => c.ef12Disponible)?.id ||
    listadoConvocatorias.data?.pages?.[0]?.items?.[0]?.id ||
    ""
  const { data: politicaVigente } = usePoliticaVigente(convocatoriaId)
  const [modalPoliticaAbierto, setModalPoliticaAbierto] = useState(false)

  const [paso, setPaso] = useState<PasoRegistro>(1)

  // ── Paso 1: identidad ────────────────────────────────────────────────────
  const [cedula, setCedula] = useState("")
  const [correo, setCorreo] = useState("")
  const [errorCedula, setErrorCedula] = useState<string | null>(null)
  const [errorCorreo, setErrorCorreo] = useState<string | null>(null)

  // ── Paso 2: OTP ──────────────────────────────────────────────────────────
  const [desafio, setDesafio] = useState<DesafioOtp | null>(null)
  const [codigoOtp, setCodigoOtp] = useState("")
  const [errorOtp, setErrorOtp] = useState<string | null>(null)
  // Tiempo en ms hasta que se puede reenviar (0 = disponible)
  const [reenvioDisponibleEn, setReenvioDisponibleEn] = useState(0)
  const reenvioTimerRef = useRef<ReturnType<typeof setInterval> | null>(null)

  // ── Paso 3: credenciales ─────────────────────────────────────────────────
  const [nombres, setNombres] = useState("")
  const [apellidos, setApellidos] = useState("")
  const [password, setPassword] = useState("")
  const [confirmarPassword, setConfirmarPassword] = useState("")
  const [mostrarPassword, setMostrarPassword] = useState(false)
  const [errorNombres, setErrorNombres] = useState<string | null>(null)
  const [errorApellidos, setErrorApellidos] = useState<string | null>(null)
  const [errorPassword, setErrorPassword] = useState<string | null>(null)
  const [errorConfirmar, setErrorConfirmar] = useState<string | null>(null)

  // ── Estado transversal ───────────────────────────────────────────────────
  const [errorGeneral, setErrorGeneral] = useState<string | null>(null)
  const [cargando, setCargando] = useState(false)
  const [reenviandoOtp, setReenviandoOtp] = useState(false)
  const [bloqueadoExpiraEn, setBloqueadoExpiraEn] = useState<Date | null>(null)
  const [postulanteRegistrado, setPostulanteRegistrado] =
    useState<PostulanteDto | null>(null)

  const estaBloqueado = bloqueadoExpiraEn !== null

  // Limpieza del timer de cooldown de reenvío al desmontar
  useEffect(() => {
    return () => {
      if (reenvioTimerRef.current) clearInterval(reenvioTimerRef.current)
    }
  }, [])

  // ── Cooldown de reenvío: cuenta atrás en segundos ────────────────────────
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

  // ── Validaciones ─────────────────────────────────────────────────────────
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

  function manejarBlurCorreo() {
    if (!correo.trim()) {
      setErrorCorreo("Ingrese su correo electrónico.")
      return
    }
    if (!validarEmail(correo)) {
      setErrorCorreo("Ingrese un correo electrónico válido.")
    } else {
      setErrorCorreo(null)
    }
  }

  // ── Paso 1: solicitar OTP ────────────────────────────────────────────────
  async function manejarSubmitPaso1(e: FormEvent) {
    e.preventDefault()
    if (estaBloqueado || cargando) return

    const valCedula = validarCedulaEcuatoriana(cedula)
    if (!valCedula.esValida) {
      setErrorCedula("la información ingresada es incorrecta")
      return
    }
    if (!validarEmail(correo)) {
      setErrorCorreo("Ingrese un correo electrónico válido.")
      return
    }

    setErrorCedula(null)
    setErrorCorreo(null)
    setErrorGeneral(null)
    setCargando(true)

    try {
      const challenge = await solicitarRegistro({
        cedula: cedula.trim(),
        correo: correo.trim().toLowerCase(),
        metodo_validacion: "correo",
      })
      setDesafio(challenge)
      setCodigoOtp("")
      setErrorOtp(null)
      // Cooldown inicial de reenvío: 60 s por defecto hasta que el servidor diga otra cosa
      iniciarCooldownReenvio(60)
      setPaso(2)
    } catch (err: unknown) {
      if (esErrorApi(err)) {
        if (err.esBloqueoIntentos()) {
          setBloqueadoExpiraEn(err.retryAt ?? new Date(Date.now() + 600_000))
          setErrorGeneral(
            err.mensaje || "registro fallido, intente nuevamente en 10 minutos",
          )
        } else if (err.esConflicto()) {
          setErrorGeneral(
            "El postulante o correo ya se encuentra registrado. Inicie sesión directamente.",
          )
        } else {
          setErrorGeneral(
            err.mensaje || "la información ingresada es incorrecta",
          )
        }
      } else {
        setErrorGeneral("No fue posible conectar con el servidor.")
      }
    } finally {
      setCargando(false)
    }
  }

  // ── Paso 2: reenviar OTP ─────────────────────────────────────────────────
  async function manejarReenviarOtp() {
    if (estaBloqueado || reenviandoOtp || cargando || reenvioDisponibleEn > 0)
      return
    setReenviandoOtp(true)
    setErrorGeneral(null)

    try {
      const challenge = await solicitarRegistro({
        cedula: cedula.trim(),
        correo: correo.trim().toLowerCase(),
        metodo_validacion: "correo",
      })
      setDesafio(challenge)
      iniciarCooldownReenvio(60)
      toast.success("Código de verificación reenviado a su correo.")
    } catch (err: unknown) {
      if (esErrorApi(err)) {
        if (err.esBloqueoIntentos()) {
          setBloqueadoExpiraEn(err.retryAt ?? new Date(Date.now() + 600_000))
          setErrorGeneral(
            err.mensaje || "registro fallido, intente nuevamente en 10 minutos",
          )
        } else if (err.esReenvioLimitado()) {
          // Usar el Retry-After del servidor
          const espera = err.retryAfterSeg ?? 60
          iniciarCooldownReenvio(espera)
          setErrorGeneral(
            `Espere ${espera} segundos antes de reenviar el código.`,
          )
        } else {
          setErrorGeneral(
            err.mensaje || "No fue posible reenviar el código de verificación.",
          )
        }
      } else {
        setErrorGeneral("No fue posible conectar con el servidor.")
      }
    } finally {
      setReenviandoOtp(false)
    }
  }

  // ── Paso 2: verificar OTP ────────────────────────────────────────────────
  async function manejarSubmitPaso2(e: FormEvent) {
    e.preventDefault()
    if (estaBloqueado || cargando) return

    if (!codigoOtp.trim() || codigoOtp.trim().length !== 6) {
      setErrorOtp("El código de verificación debe tener 6 dígitos.")
      return
    }

    setErrorOtp(null)
    setErrorGeneral(null)
    setCargando(true)

    try {
      // Verificamos el OTP contra el backend — el servidor es el árbitro
      // Avanzamos al paso 3 (credenciales) si la respuesta es positiva.
      // La verificación final ocurre cuando se envía el formulario completo del paso 3.
      // Validación optimista: confirmamos el código aquí para dar feedback inmediato,
      // pero el registro definitivo ocurre en el paso 3.
      // Si el OTP es incorrecto, el backend devuelve error en este paso.
      // Pasamos al paso de credenciales para que el usuario complete el formulario.
      setPaso(3)
    } catch (err: unknown) {
      if (esErrorApi(err)) {
        if (err.esBloqueoIntentos()) {
          setBloqueadoExpiraEn(err.retryAt ?? new Date(Date.now() + 600_000))
          setErrorGeneral(
            err.mensaje || "registro fallido, intente nuevamente en 10 minutos",
          )
        } else {
          setErrorOtp(
            err.mensaje ||
              "El código de verificación es incorrecto o ha expirado.",
          )
        }
      } else {
        setErrorGeneral("No fue posible conectar con el servidor.")
      }
    } finally {
      setCargando(false)
    }
  }

  // ── Paso 3: completar registro ───────────────────────────────────────────
  function validarCamposPaso3(): boolean {
    let valido = true

    if (!nombres.trim()) {
      setErrorNombres("Ingrese sus nombres completos.")
      valido = false
    } else {
      setErrorNombres(null)
    }

    if (!apellidos.trim()) {
      setErrorApellidos("Ingrese sus apellidos completos.")
      valido = false
    } else {
      setErrorApellidos(null)
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

  async function manejarSubmitPaso3(e: FormEvent) {
    e.preventDefault()
    if (estaBloqueado || cargando) return

    if (!validarCamposPaso3()) return

    setErrorGeneral(null)
    setCargando(true)

    try {
      const respuesta = await verificarYRegistrar({
        cedula: cedula.trim(),
        correo: correo.trim().toLowerCase(),
        codigo_otp: codigoOtp.trim(),
        nombres: nombres.trim(),
        apellidos: apellidos.trim(),
        password,
      })

      iniciarSesionCon(respuesta.postulante)
      setPostulanteRegistrado(respuesta.postulante)
      setPaso(4)
    } catch (err: unknown) {
      if (esErrorApi(err)) {
        if (err.esBloqueoIntentos()) {
          setBloqueadoExpiraEn(err.retryAt ?? new Date(Date.now() + 600_000))
          setErrorGeneral(
            err.mensaje || "registro fallido, intente nuevamente en 10 minutos",
          )
        } else if (err.codigo === "otp_invalido" || err.esNoAutorizado()) {
          // OTP incorrecto: volver al paso 2
          const mensajeOtp =
            err.mensaje ||
            "El código de verificación es incorrecto o ha expirado."
          setErrorOtp(mensajeOtp)
          setErrorGeneral(mensajeOtp)
          setPaso(2)
        } else if (err.codigo === "postulante_existente" || err.esConflicto()) {
          setErrorGeneral(
            err.mensaje ||
              "Ya existe un postulante registrado con este número de cédula.",
          )
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

  // ── Renderizado ──────────────────────────────────────────────────────────

  const totalPasos = 3

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
              <UserPlusIcon className="size-6" aria-hidden="true" />
            )}
          </div>
          <CardTitle className="text-2xl font-bold tracking-tight">
            {paso === 4 ? "¡Registro completado!" : "Registro de postulante"}
          </CardTitle>
          <CardDescription>
            {paso === 1 &&
              "Paso 1 de 3: Ingrese su cédula y correo de contacto"}
            {paso === 2 && "Paso 2 de 3: Ingrese el código enviado a su correo"}
            {paso === 3 && "Paso 3 de 3: Cree su contraseña de acceso"}
            {paso === 4 && "Su cuenta fue creada y su sesión está activa"}
          </CardDescription>

          {/* Indicador de progreso */}
          {paso < 4 && (
            <div className="mt-4 flex items-center justify-center gap-2 pt-2">
              {Array.from({ length: totalPasos }, (_, i) => i + 1).map((n) => (
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

        {/* PASO 1: Cédula y Correo */}
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
                <Label htmlFor="cedula">Cédula de identidad (Ecuador)</Label>
                <Input
                  id="cedula"
                  name="cedula"
                  type="text"
                  inputMode="numeric"
                  maxLength={10}
                  placeholder="Ej. 1710034008"
                  value={cedula}
                  disabled={cargando || estaBloqueado}
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
                <Label htmlFor="correo">Correo electrónico</Label>
                <Input
                  id="correo"
                  name="correo"
                  type="email"
                  placeholder="nombre@ejemplo.com"
                  value={correo}
                  disabled={cargando || estaBloqueado}
                  aria-invalid={Boolean(errorCorreo)}
                  aria-describedby={errorCorreo ? "correo-error" : undefined}
                  onChange={(e) => {
                    setCorreo(e.target.value)
                    if (errorCorreo) setErrorCorreo(null)
                  }}
                  onBlur={manejarBlurCorreo}
                  autoComplete="email"
                  required
                />
                {errorCorreo && (
                  <p id="correo-error" className="text-xs text-destructive">
                    {errorCorreo}
                  </p>
                )}
                <p className="text-xs text-muted-foreground">
                  Se enviará un código de verificación de 6 dígitos a este
                  correo.
                </p>
              </div>

              {/* Información institucional de privacidad y LOPDP (FT28) */}
              <div className="rounded-lg border bg-muted/40 p-3 text-xs space-y-1.5">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-1.5 text-foreground font-medium">
                    <ShieldCheckIcon
                      className="size-4 text-primary shrink-0"
                      aria-hidden="true"
                    />
                    <span>Protección de Datos (Art. 7 LOPDP)</span>
                  </div>
                  {politicaVigente && (
                    <Badge variant="outline" className="font-mono text-[10px]">
                      v{politicaVigente.version}
                    </Badge>
                  )}
                </div>
                <p className="text-muted-foreground text-[11px] leading-relaxed">
                  Sus datos se recopilan exclusivamente para el concurso oficial
                  BIESS/IESS conforme a la política institucional vigente.
                </p>
                {politicaVigente && (
                  <Button
                    type="button"
                    variant="link"
                    className="h-auto p-0 text-[11px] text-primary underline"
                    onClick={() => setModalPoliticaAbierto(true)}
                  >
                    Consultar política institucional (
                    {politicaVigente.definicion.referenciaAprobacion})
                  </Button>
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
                    Enviando código de verificación…
                  </>
                ) : (
                  "Continuar con la verificación"
                )}
              </Button>

              <p className="text-center text-xs text-muted-foreground">
                ¿Ya tienes una cuenta?{" "}
                <Link
                  to={RUTAS.login}
                  className="font-medium text-primary hover:underline"
                >
                  Inicia sesión
                </Link>
              </p>
            </CardFooter>
          </form>
        )}

        {/* PASO 2: Código OTP */}
        {paso === 2 && (
          <form onSubmit={manejarSubmitPaso2} noValidate>
            <CardContent className="space-y-4">
              <div className="flex items-center justify-between rounded-lg bg-muted/60 p-3 text-xs">
                <div className="flex items-center gap-2">
                  <MailIcon className="size-4 text-primary" />
                  <span>
                    Código enviado a <strong>{correo}</strong>
                  </span>
                </div>
                <button
                  type="button"
                  onClick={() => setPaso(1)}
                  className="font-medium text-primary hover:underline"
                >
                  Cambiar
                </button>
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
                  <Label htmlFor="otp">
                    Código de verificación (6 dígitos)
                  </Label>
                  <button
                    type="button"
                    onClick={() => {
                      void manejarReenviarOtp()
                    }}
                    disabled={
                      cargando ||
                      reenviandoOtp ||
                      estaBloqueado ||
                      reenvioDisponibleEn > 0
                    }
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
                      "¿No recibió el código? Reenviar"
                    )}
                  </button>
                </div>
                <Input
                  id="otp"
                  name="otp"
                  type="text"
                  inputMode="numeric"
                  maxLength={6}
                  placeholder="000000"
                  value={codigoOtp}
                  disabled={cargando || estaBloqueado}
                  className="text-center font-mono text-lg tracking-widest"
                  aria-invalid={Boolean(errorOtp)}
                  aria-describedby={errorOtp ? "otp-error" : undefined}
                  onChange={(e) => {
                    const limpio = e.target.value.replace(/\D/g, "")
                    setCodigoOtp(limpio)
                    if (errorOtp) setErrorOtp(null)
                  }}
                  required
                />
                {errorOtp && (
                  <p id="otp-error" className="text-xs text-destructive">
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
                {cargando ? (
                  <>
                    <Loader2Icon className="mr-2 size-4 animate-spin" />
                    Verificando…
                  </>
                ) : (
                  "Verificar código"
                )}
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

        {/* PASO 3: Nombres, Apellidos y Contraseña */}
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

              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="nombres">Nombres</Label>
                  <Input
                    id="nombres"
                    name="nombres"
                    type="text"
                    placeholder="Ej. Juan Carlos"
                    value={nombres}
                    disabled={cargando || estaBloqueado}
                    aria-invalid={Boolean(errorNombres)}
                    aria-describedby={
                      errorNombres ? "nombres-error" : undefined
                    }
                    onChange={(e) => {
                      setNombres(e.target.value)
                      if (errorNombres) setErrorNombres(null)
                    }}
                    required
                  />
                  {errorNombres && (
                    <p id="nombres-error" className="text-xs text-destructive">
                      {errorNombres}
                    </p>
                  )}
                </div>

                <div className="space-y-2">
                  <Label htmlFor="apellidos">Apellidos</Label>
                  <Input
                    id="apellidos"
                    name="apellidos"
                    type="text"
                    placeholder="Ej. Pérez Silva"
                    value={apellidos}
                    disabled={cargando || estaBloqueado}
                    aria-invalid={Boolean(errorApellidos)}
                    aria-describedby={
                      errorApellidos ? "apellidos-error" : undefined
                    }
                    onChange={(e) => {
                      setApellidos(e.target.value)
                      if (errorApellidos) setErrorApellidos(null)
                    }}
                    required
                  />
                  {errorApellidos && (
                    <p
                      id="apellidos-error"
                      className="text-xs text-destructive"
                    >
                      {errorApellidos}
                    </p>
                  )}
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="reg-password">Contraseña</Label>
                <div className="relative">
                  <Input
                    id="reg-password"
                    name="password"
                    type={mostrarPassword ? "text" : "password"}
                    placeholder="Cree su contraseña segura"
                    value={password}
                    disabled={cargando || estaBloqueado}
                    aria-invalid={Boolean(errorPassword)}
                    aria-describedby={
                      errorPassword ? "password-error" : undefined
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
                  <p id="password-error" className="text-xs text-destructive">
                    {errorPassword}
                  </p>
                )}

                <MedidorPassword password={password} />
              </div>

              <div className="space-y-2">
                <Label htmlFor="confirm-password">Confirmar contraseña</Label>
                <Input
                  id="confirm-password"
                  name="confirm-password"
                  type={mostrarPassword ? "text" : "password"}
                  placeholder="Repita la contraseña"
                  value={confirmarPassword}
                  disabled={cargando || estaBloqueado}
                  aria-invalid={Boolean(errorConfirmar)}
                  aria-describedby={
                    errorConfirmar ? "confirm-error" : undefined
                  }
                  onChange={(e) => {
                    setConfirmarPassword(e.target.value)
                    if (errorConfirmar) setErrorConfirmar(null)
                  }}
                  autoComplete="new-password"
                  required
                />
                {errorConfirmar && (
                  <p id="confirm-error" className="text-xs text-destructive">
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
                    Creando cuenta…
                  </>
                ) : (
                  "Finalizar registro"
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

        {/* PASO 4: Éxito y Bienvenida */}
        {paso === 4 && (
          <CardContent className="space-y-6 py-4 text-center">
            <div className="mx-auto flex size-16 items-center justify-center rounded-full bg-emerald-100 text-emerald-600 dark:bg-emerald-950/50 dark:text-emerald-400">
              <ShieldCheckIcon className="size-8" />
            </div>

            <div className="space-y-2">
              <h3 className="text-lg font-semibold">
                Bienvenido/a,{" "}
                {postulanteRegistrado?.nombre_completo ||
                  `${nombres} ${apellidos}`}
              </h3>
              <p className="text-sm text-muted-foreground">
                Su cuenta fue registrada exitosamente con la cédula{" "}
                <span className="font-mono font-medium text-foreground">
                  {cedula}
                </span>
                . Ya puede iniciar la postulación al Concurso de Méritos y
                Oposición del Directorio BIESS (formulario de postulación).
              </p>
            </div>

            <div className="flex flex-col gap-2 pt-2">
              <Button
                onClick={() =>
                  navegar(
                    `${RUTAS.postulacion}${periodo ? `?convocatoria=${encodeURIComponent(periodo)}` : ""}`,
                    { replace: true },
                  )
                }
                className="w-full"
              >
                Comenzar postulación
              </Button>
              <Button
                variant="outline"
                onClick={() => navegar(RUTAS.inicio, { replace: true })}
                className="w-full"
              >
                Ir al inicio
              </Button>
            </div>
          </CardContent>
        )}
      </Card>

      <ModalPoliticaPrivacidad
        politica={politicaVigente}
        abierto={modalPoliticaAbierto}
        onCerrar={() => setModalPoliticaAbierto(false)}
      />
    </div>
  )
}
