import { useState, type FormEvent } from "react"
import {
  CheckCircle2Icon,
  ChevronLeftIcon,
  EyeIcon,
  EyeOffIcon,
  Loader2Icon,
  MailIcon,
  ShieldCheckIcon,
  UserPlusIcon,
} from "lucide-react"
import { Link, useNavigate } from "react-router"

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
import { useSesion } from "@/features/auth/sesion-context"
import { esErrorApi } from "@/lib/api-error"
import {
  solicitarRegistro,
  verificarYRegistrar,
  type PostulanteDto,
} from "@/lib/clientes/auth"
import { RUTAS } from "@/lib/rutas"
import {
  evaluarFortalezaPassword,
  validarCedulaEcuatoriana,
  validarEmail,
} from "@/utils/validators"

type PasoRegistro = 1 | 2 | 3

export function RegistroView() {
  const navegar = useNavigate()
  const { iniciarSesionCon } = useSesion()

  const [paso, setPaso] = useState<PasoRegistro>(1)

  // Datos del paso 1
  const [cedula, setCedula] = useState("")
  const [correo, setCorreo] = useState("")
  const [errorCedula, setErrorCedula] = useState<string | null>(null)
  const [errorCorreo, setErrorCorreo] = useState<string | null>(null)

  // Datos del paso 2
  const [codigoOtp, setCodigoOtp] = useState("")
  const [nombres, setNombres] = useState("")
  const [apellidos, setApellidos] = useState("")
  const [password, setPassword] = useState("")
  const [confirmarPassword, setConfirmarPassword] = useState("")
  const [mostrarPassword, setMostrarPassword] = useState(false)

  const [errorOtp, setErrorOtp] = useState<string | null>(null)
  const [errorNombres, setErrorNombres] = useState<string | null>(null)
  const [errorApellidos, setErrorApellidos] = useState<string | null>(null)
  const [errorPassword, setErrorPassword] = useState<string | null>(null)
  const [errorConfirmar, setErrorConfirmar] = useState<string | null>(null)

  // Estados transversales
  const [errorGeneral, setErrorGeneral] = useState<string | null>(null)
  const [cargando, setCargando] = useState(false)
  const [bloqueado, setBloqueado] = useState(false)
  const [postulanteRegistrado, setPostulanteRegistrado] =
    useState<PostulanteDto | null>(null)

  // Validaciones inmediatas Paso 1
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

  // Envío Paso 1: Solicitar código OTP
  async function manejarSubmitPaso1(e: FormEvent) {
    e.preventDefault()
    if (bloqueado || cargando) return

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
      await solicitarRegistro({
        cedula: cedula.trim(),
        correo: correo.trim().toLowerCase(),
        metodo_validacion: "correo",
      })
      setPaso(2)
    } catch (err: unknown) {
      if (esErrorApi(err)) {
        if (err.esBloqueoIntentos()) {
          setBloqueado(true)
          setErrorGeneral(
            "registro fallido, inténtelo nuevamente en 10 minutos",
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

  // Validaciones Paso 2
  function validarCamposPaso2(): boolean {
    let valido = true

    if (!codigoOtp.trim() || codigoOtp.trim().length !== 6) {
      setErrorOtp("El código de verificación debe tener 6 dígitos.")
      valido = false
    } else {
      setErrorOtp(null)
    }

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

  // Envío Paso 2: Verificar OTP y Registrarse
  async function manejarSubmitPaso2(e: FormEvent) {
    e.preventDefault()
    if (bloqueado || cargando) return

    if (!validarCamposPaso2()) {
      return
    }

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
      setPaso(3)
    } catch (err: unknown) {
      if (esErrorApi(err)) {
        if (err.esBloqueoIntentos()) {
          setBloqueado(true)
          setErrorGeneral(
            "registro fallido, inténtelo nuevamente en 10 minutos",
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

  return (
    <div className="mx-auto flex min-h-[calc(100svh-4rem)] w-full max-w-lg items-center px-4 py-8">
      <Card className="w-full shadow-lg">
        <CardHeader className="space-y-1 text-center">
          <div className="mx-auto mb-2 flex size-12 items-center justify-center rounded-xl bg-primary/10 text-primary">
            {paso === 3 ? (
              <CheckCircle2Icon
                className="size-6 text-emerald-600"
                aria-hidden="true"
              />
            ) : (
              <UserPlusIcon className="size-6" aria-hidden="true" />
            )}
          </div>
          <CardTitle className="text-2xl font-bold tracking-tight">
            {paso === 3 ? "¡Registro completado!" : "Registro de postulante"}
          </CardTitle>
          <CardDescription>
            {paso === 1 &&
              "Paso 1 de 2: Ingrese su cédula y correo de contacto"}
            {paso === 2 &&
              "Paso 2 de 2: Verifique su código y cree su contraseña"}
            {paso === 3 && "Su cuenta fue creada y su sesión está activa"}
          </CardDescription>

          {/* Indicador de progreso de pasos */}
          {paso < 3 && (
            <div className="mt-4 flex items-center justify-center gap-2 pt-2">
              <span
                className={`flex size-6 items-center justify-center rounded-full text-xs font-semibold ${
                  paso === 1
                    ? "bg-primary text-primary-foreground"
                    : "bg-muted text-muted-foreground"
                }`}
              >
                1
              </span>
              <div className="h-0.5 w-12 bg-muted" />
              <span
                className={`flex size-6 items-center justify-center rounded-full text-xs font-semibold ${
                  paso === 2
                    ? "bg-primary text-primary-foreground"
                    : "bg-muted text-muted-foreground"
                }`}
              >
                2
              </span>
            </div>
          )}
        </CardHeader>

        {/* PASO 1: Cédula y Correo */}
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
                <Label htmlFor="cedula">Cédula de identidad (Ecuador)</Label>
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
                <Label htmlFor="correo">Correo electrónico</Label>
                <Input
                  id="correo"
                  name="correo"
                  type="email"
                  placeholder="nombre@ejemplo.com"
                  value={correo}
                  disabled={cargando || bloqueado}
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

        {/* PASO 2: OTP, Nombres, Apellidos y Contraseña */}
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
                <Label htmlFor="otp">Código de verificación (6 dígitos)</Label>
                <Input
                  id="otp"
                  name="otp"
                  type="text"
                  inputMode="numeric"
                  maxLength={6}
                  placeholder="000000"
                  value={codigoOtp}
                  disabled={cargando || bloqueado}
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

              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="nombres">Nombres</Label>
                  <Input
                    id="nombres"
                    name="nombres"
                    type="text"
                    placeholder="Ej. Juan Carlos"
                    value={nombres}
                    disabled={cargando || bloqueado}
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
                    disabled={cargando || bloqueado}
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
                    disabled={cargando || bloqueado}
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
                  disabled={cargando || bloqueado}
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
                disabled={cargando || bloqueado}
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

        {/* PASO 3: Éxito y Bienvenida */}
        {paso === 3 && (
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
                Oposición del Directorio BIESS (Formulario EF-12).
              </p>
            </div>

            <div className="flex flex-col gap-2 pt-2">
              <Button
                onClick={() => navegar(RUTAS.postulacion, { replace: true })}
                className="w-full"
              >
                Comenzar postulación (EF-12)
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
    </div>
  )
}
