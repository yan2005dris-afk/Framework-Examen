import {
  AlertTriangleIcon,
  ArrowRightIcon,
  CalendarDaysIcon,
  CheckCircle2Icon,
  ClockIcon,
  ExternalLinkIcon,
  InfoIcon,
  LockIcon,
  ShieldAlertIcon,
  ShieldCheckIcon,
  UserCheckIcon,
  UsersIcon,
} from "lucide-react"
import { Link } from "react-router"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
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
import { useConvocatoriaPublica } from "@/features/convocatorias/queries"
import { RUTAS } from "@/lib/rutas"
import { NOMBRES_FASES_OFICIALES } from "@/types/convocatoria"
import {
  diasHastaCierre,
  formatearFechaEcuador,
  obtenerEstadoTemporal,
} from "@/utils/fechas"

/**
 * Portada oficial del Concurso Público de Méritos y Oposición (FT14).
 * Presenta el estado de apertura/cierre, el cronograma oficial de fases,
 * los requisitos de postulación y los enlaces oficiales a los portales IESS/BIESS.
 * No genera fechas ni convocatorias ficticias si la API no devuelve datos aprobados.
 */
export function PortadaView() {
  const { estado: estadoSesion, postulante } = useSesion()
  const { ruta: rutaDestino, calculando } = useRutaInicial()
  const {
    data: convocatoria,
    isLoading: cargandoConvocatoria,
    isError: errorConvocatoria,
  } = useConvocatoriaPublica()

  const autenticado = estadoSesion === "autenticado"

  // La disponibilidad real del formulario EF-12 la define el servidor mediante ef12Disponible
  const existeConvocatoria = Boolean(convocatoria)
  const abierta = Boolean(convocatoria?.ef12Disponible)
  const apertura = convocatoria?.apertura ?? ""
  const cierre = convocatoria?.cierre ?? ""
  const estadoTemporal =
    apertura && cierre
      ? obtenerEstadoTemporal(apertura, cierre, !abierta)
      : "cerrada"
  const diasRestantes = cierre ? diasHastaCierre(cierre) : 0

  return (
    <div className="flex w-full flex-col">
      {/* Barra institucional oficial (govbar) */}
      <aside
        className="w-full border-b bg-muted/40 py-1.5 text-xs text-muted-foreground"
        aria-label="Información institucional de la República del Ecuador"
      >
        <div className="mx-auto flex max-w-6xl items-center justify-between px-4">
          <div className="flex items-center gap-2">
            <span className="font-semibold text-foreground">
              REPÚBLICA DEL ECUADOR
            </span>
            <span aria-hidden="true">|</span>
            <span>Comisión Especial de Selección y Designación</span>
          </div>
          <span className="font-medium text-foreground">BIESS — IESS</span>
        </div>
      </aside>

      <main id="main" className="mx-auto w-full max-w-6xl px-4 py-8 md:py-12">
        {/* Caso 1: Error de red o indisponibilidad del servidor */}
        {errorConvocatoria && (
          <Alert variant="destructive" className="mb-8">
            <ShieldAlertIcon className="size-5" />
            <AlertTitle className="text-base font-semibold">
              Servicio de consultas temporalmente no disponible
            </AlertTitle>
            <AlertDescription className="mt-1 text-sm leading-relaxed text-foreground/90">
              No se pudo obtener la información oficial de la convocatoria desde
              el servidor. Por seguridad institucional, no se habilitan acciones
              de postulación hasta restablecer la conexión.
            </AlertDescription>
          </Alert>
        )}

        {/* Caso 2: Sin convocatoria publicada en el servidor */}
        {!cargandoConvocatoria && !errorConvocatoria && !existeConvocatoria && (
          <Alert className="mb-8 border-amber-500/50 bg-amber-500/10 text-foreground dark:bg-amber-500/20">
            <LockIcon className="size-5 text-amber-600 dark:text-amber-400" />
            <AlertTitle className="text-base font-semibold text-amber-900 dark:text-amber-200">
              Sin convocatoria activa para postulación
            </AlertTitle>
            <AlertDescription className="mt-1 text-sm leading-relaxed text-foreground/90">
              Actualmente no existe una convocatoria pública aprobada y en
              período de recepción de expedientes. La Comisión Especial de
              Selección y Designación publicará las fechas oficiales en este
              portal institucional en cuanto el período sea aprobado.
            </AlertDescription>
          </Alert>
        )}

        {/* Caso 3: Convocatoria existente pero no abierta para postulaciones */}
        {!cargandoConvocatoria && existeConvocatoria && !abierta && (
          <Alert
            variant="destructive"
            className="mb-8 border-destructive/50 bg-destructive/10 text-destructive-foreground dark:bg-destructive/20"
          >
            <LockIcon className="size-5" />
            <AlertTitle className="text-base font-semibold">
              {estadoTemporal === "programada"
                ? "El período de postulaciones iniciará próximamente"
                : "Convocatoria no admite postulaciones"}
            </AlertTitle>
            <AlertDescription className="mt-1 text-sm leading-relaxed text-foreground/90">
              {estadoTemporal === "programada" ? (
                <>
                  La recepción de expedientes digitales se abrirá el{" "}
                  <strong>
                    {formatearFechaEcuador(apertura, { incluirHora: true })}
                  </strong>
                  . En esa fecha se habilitará el registro de postulantes y el
                  acceso al formulario EF-12.
                </>
              ) : (
                <>
                  El plazo de recepción de postulaciones ha concluido o se
                  encuentra deshabilitado por la Comisión de Selección. Conforme
                  a la normativa vigente, no se admiten nuevos registros ni
                  envíos de expedientes.
                </>
              )}
            </AlertDescription>
          </Alert>
        )}

        {/* Sección Hero */}
        <section className="space-y-6">
          <div className="flex flex-wrap items-center gap-2">
            <p className="text-sm font-semibold tracking-wide text-primary uppercase">
              Convocatoria pública oficial
              {convocatoria?.codigo ? ` · ${convocatoria.codigo}` : ""}
            </p>
            {abierta ? (
              <Badge className="bg-emerald-600 hover:bg-emerald-700">
                <ClockIcon className="mr-1 size-3.5" />
                Postulaciones Abiertas
              </Badge>
            ) : estadoTemporal === "programada" ? (
              <Badge
                variant="outline"
                className="border-amber-500 text-amber-700 dark:text-amber-400"
              >
                <CalendarDaysIcon className="mr-1 size-3.5" />
                Programada
              </Badge>
            ) : (
              <Badge variant="destructive">
                <LockIcon className="mr-1 size-3.5" />
                Convocatoria Cerrada
              </Badge>
            )}
          </div>

          <h1 className="max-w-4xl text-3xl font-extrabold tracking-tight text-foreground sm:text-5xl">
            {convocatoria?.titulo ?? "Concurso Público de Méritos y Oposición"}
          </h1>

          <p className="max-w-3xl text-lg text-muted-foreground sm:text-xl">
            Designación de los{" "}
            <strong className="text-foreground">
              Miembros Principales y Alternos del Directorio del BIESS
            </strong>{" "}
            en representación de los Afiliados, Jubilados y Empleadores del
            Ecuador.
          </p>

          {/* Estado de sesión activa si existe */}
          {autenticado && (
            <div className="rounded-lg border bg-muted/30 p-3 text-sm text-muted-foreground">
              Sesión iniciada como{" "}
              <span className="font-semibold text-foreground">
                {postulante?.nombre_completo ||
                  `${postulante?.nombres} ${postulante?.apellidos}`}
              </span>{" "}
              ({postulante?.correo || postulante?.cedula}).
            </div>
          )}

          {/* Acciones principales de llamada a la acción (CTA) */}
          <div className="flex flex-wrap items-center gap-3 pt-2">
            {autenticado ? (
              rutaDestino ? (
                <Button asChild size="lg" className="font-semibold">
                  <Link to={rutaDestino}>
                    {rutaDestino.startsWith(RUTAS.dashboard)
                      ? "Ir al panel de revisión"
                      : rutaDestino.startsWith("/admin")
                        ? "Ir al panel administrativo"
                        : "Continuar mi postulación"}
                    <ArrowRightIcon
                      className="ml-2 size-4"
                      aria-hidden="true"
                    />
                  </Link>
                </Button>
              ) : (
                <Button size="lg" disabled>
                  {calculando ? "Preparando panel…" : "Sesión activa"}
                </Button>
              )
            ) : abierta ? (
              <>
                <Button asChild size="lg" className="font-semibold">
                  <Link to={RUTAS.registro}>
                    Iniciar postulación
                    <ArrowRightIcon
                      className="ml-2 size-4"
                      aria-hidden="true"
                    />
                  </Link>
                </Button>
                <Button asChild variant="outline" size="lg">
                  <Link to={RUTAS.login}>Ya tengo cuenta</Link>
                </Button>
              </>
            ) : (
              <>
                <Button asChild variant="outline" size="lg">
                  <Link to={RUTAS.login}>
                    Consultar postulación enviada
                    <ArrowRightIcon
                      className="ml-2 size-4"
                      aria-hidden="true"
                    />
                  </Link>
                </Button>
                <Button size="lg" disabled variant="secondary">
                  Postulaciones no disponibles
                </Button>
              </>
            )}
          </div>

          {/* Metadatos temporales de cierre */}
          {cierre && (
            <div className="flex flex-wrap items-center gap-6 pt-2 text-sm text-muted-foreground">
              <div className="flex items-center gap-2">
                <CalendarDaysIcon className="size-4 text-primary" />
                <span>
                  Cierre de postulaciones:{" "}
                  <strong className="text-foreground">
                    {formatearFechaEcuador(cierre, { incluirHora: true })}
                  </strong>
                  {abierta && diasRestantes > 0 && (
                    <span className="ml-1 text-emerald-600 dark:text-emerald-400">
                      ({diasRestantes}{" "}
                      {diasRestantes === 1 ? "día restante" : "días restantes"})
                    </span>
                  )}
                </span>
              </div>
              <div className="flex items-center gap-2">
                <ShieldCheckIcon className="size-4 text-primary" />
                <span>Autenticación por código OTP al correo electrónico</span>
              </div>
            </div>
          )}
        </section>

        {/* Sectores de representación tripartita */}
        <section className="mt-16 space-y-6">
          <div className="border-b pb-4">
            <h2 className="text-2xl font-bold tracking-tight text-foreground">
              Sectores de representación tripartita
            </h2>
            <p className="text-muted-foreground">
              El Directorio del BIESS se conforma de manera equitativa por tres
              sectores sociales
            </p>
          </div>

          <div className="grid gap-6 md:grid-cols-3">
            <Card className="flex flex-col justify-between border-t-4 border-t-blue-600 shadow-sm transition hover:shadow-md">
              <CardHeader>
                <div className="flex items-center justify-between">
                  <Badge
                    variant="outline"
                    className="border-blue-600 text-blue-700 dark:text-blue-400"
                  >
                    Afiliados
                  </Badge>
                  <UsersIcon className="size-5 text-blue-600" />
                </div>
                <CardTitle className="text-xl">
                  1 principal y 1 alterno
                </CardTitle>
                <CardDescription>
                  Representación de trabajadores en relación de dependencia
                </CardDescription>
              </CardHeader>
              <CardContent className="text-sm text-muted-foreground">
                Afiliados activos al IESS con aportaciones continuas y respaldo
                formal de una organización de trabajadores de representación
                nacional legalmente constituida.
              </CardContent>
            </Card>

            <Card className="flex flex-col justify-between border-t-4 border-t-emerald-600 shadow-sm transition hover:shadow-md">
              <CardHeader>
                <div className="flex items-center justify-between">
                  <Badge
                    variant="outline"
                    className="border-emerald-600 text-emerald-700 dark:text-emerald-400"
                  >
                    Jubilados
                  </Badge>
                  <UserCheckIcon className="size-5 text-emerald-600" />
                </div>
                <CardTitle className="text-xl">
                  1 principal y 1 alterno
                </CardTitle>
                <CardDescription>
                  Representación de jubilados y pensionistas
                </CardDescription>
              </CardHeader>
              <CardContent className="text-sm text-muted-foreground">
                Jubilados que acrediten su condición ante el IESS y cuenten con
                el respaldo de una organización nacional de jubilados
                debidamente reconocida ante la autoridad competente.
              </CardContent>
            </Card>

            <Card className="flex flex-col justify-between border-t-4 border-t-amber-600 shadow-sm transition hover:shadow-md">
              <CardHeader>
                <div className="flex items-center justify-between">
                  <Badge
                    variant="outline"
                    className="border-amber-600 text-amber-700 dark:text-amber-400"
                  >
                    Empleadores
                  </Badge>
                  <ShieldCheckIcon className="size-5 text-amber-600" />
                </div>
                <CardTitle className="text-xl">
                  1 principal y 1 alterno
                </CardTitle>
                <CardDescription>
                  Representación del sector productivo y empleador
                </CardDescription>
              </CardHeader>
              <CardContent className="text-sm text-muted-foreground">
                Empleadores que acrediten su calidad al día con sus obligaciones
                patronales y cuenten con el respaldo de una federación o cámara
                de representación nacional.
              </CardContent>
            </Card>
          </div>
        </section>

        {/* Fases del concurso publicadas en el servidor */}
        {convocatoria?.fases && convocatoria.fases.length > 0 && (
          <section className="mt-16 space-y-6">
            <div className="border-b pb-4">
              <h2 className="text-2xl font-bold tracking-tight text-foreground">
                Fases del concurso y cronograma oficial
              </h2>
              <p className="text-muted-foreground">
                Etapas aprobadas y publicadas por la Comisión Especial de
                Selección (zona horaria oficial {convocatoria.zonaHoraria})
              </p>
            </div>

            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              {convocatoria.fases.map((fase) => (
                <Card
                  key={fase.codigo}
                  className={`relative flex flex-col justify-between ${
                    fase.habilitada
                      ? "border-primary bg-primary/5 shadow-sm ring-1 ring-primary/20"
                      : "border-muted"
                  }`}
                >
                  <CardHeader className="pb-2">
                    <div className="flex items-center justify-between gap-2">
                      <span className="font-mono text-xs font-semibold uppercase text-muted-foreground">
                        {fase.codigo}
                      </span>
                      {fase.habilitada ? (
                        <Badge className="bg-primary text-primary-foreground text-xs">
                          En curso
                        </Badge>
                      ) : (
                        <Badge
                          variant="outline"
                          className="text-xs text-muted-foreground"
                        >
                          Publicada
                        </Badge>
                      )}
                    </div>
                    <CardTitle className="text-base font-semibold leading-snug">
                      {fase.titulo || NOMBRES_FASES_OFICIALES[fase.codigo]}
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-3 text-xs text-muted-foreground">
                    <div className="flex items-center gap-1.5 font-medium text-foreground">
                      <CalendarDaysIcon className="size-3.5 text-muted-foreground" />
                      <span>
                        {formatearFechaEcuador(fase.apertura)} al{" "}
                        {formatearFechaEcuador(fase.cierre)}
                      </span>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          </section>
        )}

        {/* Ruta de postulación en 4 pasos */}
        <section className="mt-16 space-y-6">
          <div className="border-b pb-4">
            <h2 className="text-2xl font-bold tracking-tight text-foreground">
              Ruta de postulación
            </h2>
            <p className="text-muted-foreground">
              Cuatro pasos secuenciales con guardado automático de borrador
            </p>
          </div>

          <ol className="grid gap-6 md:grid-cols-4">
            <li className="flex flex-col rounded-xl border bg-card p-5 shadow-sm">
              <span className="mb-2 inline-flex w-fit items-center rounded-md bg-primary/10 px-2 py-1 text-xs font-semibold text-primary">
                Paso 1
              </span>
              <h3 className="mb-1 text-base font-semibold text-foreground">
                Registro de identidad
              </h3>
              <p className="text-xs leading-relaxed text-muted-foreground">
                Cédula de ciudadanía validada mediante algoritmo Módulo 10 y
                código OTP seguro enviado a su correo electrónico personal.
              </p>
            </li>

            <li className="flex flex-col rounded-xl border bg-card p-5 shadow-sm">
              <span className="mb-2 inline-flex w-fit items-center rounded-md bg-primary/10 px-2 py-1 text-xs font-semibold text-primary">
                Paso 2
              </span>
              <h3 className="mb-1 text-base font-semibold text-foreground">
                Formulario EF-12
              </h3>
              <p className="text-xs leading-relaxed text-muted-foreground">
                Nueve secciones de captura con adjuntos de respaldo en formato
                PDF (títulos, experiencia, publicaciones y certificaciones).
              </p>
            </li>

            <li className="flex flex-col rounded-xl border bg-card p-5 shadow-sm">
              <span className="mb-2 inline-flex w-fit items-center rounded-md bg-primary/10 px-2 py-1 text-xs font-semibold text-primary">
                Paso 3
              </span>
              <h3 className="mb-1 text-base font-semibold text-foreground">
                Revisión y consentimiento
              </h3>
              <p className="text-xs leading-relaxed text-muted-foreground">
                Verificación íntegra de declaraciones juradas, declaración de no
                inhabilidades y autorización expresa de tratamiento de datos.
              </p>
            </li>

            <li className="flex flex-col rounded-xl border bg-card p-5 shadow-sm">
              <span className="mb-2 inline-flex w-fit items-center rounded-md bg-primary/10 px-2 py-1 text-xs font-semibold text-primary">
                Paso 4
              </span>
              <h3 className="mb-1 text-base font-semibold text-foreground">
                Envío y sellado criptográfico
              </h3>
              <p className="text-xs leading-relaxed text-muted-foreground">
                Generación de código único de postulación, hash SHA-256
                inmutable, comprobante de recepción e impresión oficial.
              </p>
            </li>
          </ol>
        </section>

        {/* Requisitos y Antes de empezar */}
        <section className="mt-16 grid gap-6 md:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle className="text-lg">
                Requisitos para postular
              </CardTitle>
              <CardDescription>
                Verifique el cumplimiento de condiciones antes de iniciar
              </CardDescription>
            </CardHeader>
            <CardContent>
              <ul className="space-y-2.5 text-sm text-muted-foreground">
                <li className="flex items-start gap-2">
                  <CheckCircle2Icon className="mt-0.5 size-4 text-emerald-600 shrink-0" />
                  <span>
                    Nacionalidad ecuatoriana y pleno goce de los derechos de
                    participación política.
                  </span>
                </li>
                <li className="flex items-start gap-2">
                  <CheckCircle2Icon className="mt-0.5 size-4 text-emerald-600 shrink-0" />
                  <span>
                    Calidad de afiliado activo, jubilado o empleador vigente
                    registrado en el IESS.
                  </span>
                </li>
                <li className="flex items-start gap-2">
                  <CheckCircle2Icon className="mt-0.5 size-4 text-emerald-600 shrink-0" />
                  <span>
                    Respaldo formal de una organización legalmente constituida
                    ante autoridad competente.
                  </span>
                </li>
                <li className="flex items-start gap-2">
                  <CheckCircle2Icon className="mt-0.5 size-4 text-emerald-600 shrink-0" />
                  <span>
                    Documentos de respaldo en formato PDF: máximo 2 MB por
                    archivo (4 MB para respaldo organizativo). Cuota total de 50
                    MB por postulante.
                  </span>
                </li>
              </ul>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-lg">Antes de empezar</CardTitle>
              <CardDescription>
                Tenga a mano la siguiente documentación
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <ul className="space-y-2.5 text-sm text-muted-foreground">
                <li className="flex items-start gap-2">
                  <CheckCircle2Icon className="mt-0.5 size-4 text-emerald-600 shrink-0" />
                  <span>Su número de cédula de ciudadanía ecuatoriana.</span>
                </li>
                <li className="flex items-start gap-2">
                  <CheckCircle2Icon className="mt-0.5 size-4 text-emerald-600 shrink-0" />
                  <span>
                    Un correo electrónico personal activo para recibir el código
                    OTP.
                  </span>
                </li>
                <li className="flex items-start gap-2">
                  <CheckCircle2Icon className="mt-0.5 size-4 text-emerald-600 shrink-0" />
                  <span>
                    Títulos académicos, certificados laborales y de capacitación
                    digitalizados en PDF.
                  </span>
                </li>
                <li className="flex items-start gap-2">
                  <CheckCircle2Icon className="mt-0.5 size-4 text-emerald-600 shrink-0" />
                  <span>
                    El documento de respaldo emitido por su organización
                    representativa.
                  </span>
                </li>
              </ul>

              <div className="rounded-lg border border-amber-500/30 bg-amber-500/10 p-3 text-xs text-amber-900 dark:text-amber-200">
                <div className="flex items-center gap-2 font-semibold">
                  <AlertTriangleIcon className="size-4 shrink-0 text-amber-600 dark:text-amber-400" />
                  <span>El formulario no se puede enviar dos veces</span>
                </div>
                <p className="mt-1">
                  Una postulación sellada y enviada queda permanentemente
                  bloqueada para edición con su respectivo hash criptográfico.
                </p>
              </div>
            </CardContent>
          </Card>
        </section>

        {/* Aviso de consentimiento y protección de datos */}
        <section className="mt-8">
          <Alert className="border-blue-500/30 bg-blue-500/5 text-foreground">
            <InfoIcon className="size-4 text-blue-600 dark:text-blue-400" />
            <AlertTitle className="text-sm font-semibold text-blue-900 dark:text-blue-300">
              Consentimiento y protección de datos personales
            </AlertTitle>
            <AlertDescription className="mt-1 text-xs text-muted-foreground">
              Al postular usted otorga consentimiento libre, previo y explícito
              para el tratamiento y verificación de sus datos personales y
              profesionales por parte de la Comisión de Selección, en estricto
              cumplimiento del artículo 7 de la Ley Orgánica de Protección de
              Datos Personales (LOPDP).
            </AlertDescription>
          </Alert>
        </section>

        {/* Enlaces de integración a portales IESS y BIESS */}
        <section className="mt-12 rounded-xl border bg-muted/20 p-6">
          <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
            <div className="space-y-1">
              <h3 className="text-sm font-semibold text-foreground">
                Enlaces oficiales e integración institucional
              </h3>
              <p className="text-xs text-muted-foreground">
                La postulación al concurso se gestiona exclusivamente en este
                portal. Para trámites institucionales o consulta de historia
                laboral, visite los portales de las entidades.
              </p>
            </div>
            <div className="flex flex-wrap items-center gap-3">
              <Button asChild variant="outline" size="sm">
                <a
                  href="https://www.biess.fin.ec"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center gap-1.5"
                >
                  <span>Portal Oficial BIESS</span>
                  <ExternalLinkIcon className="size-3.5" aria-hidden="true" />
                  <span className="sr-only"> (abre en una nueva pestaña)</span>
                </a>
              </Button>
              <Button asChild variant="outline" size="sm">
                <a
                  href="https://www.iess.gob.ec"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center gap-1.5"
                >
                  <span>Portal Oficial IESS</span>
                  <ExternalLinkIcon className="size-3.5" aria-hidden="true" />
                  <span className="sr-only"> (abre en una nueva pestaña)</span>
                </a>
              </Button>
            </div>
          </div>
        </section>
      </main>

      {/* Pie de página institucional */}
      <footer className="mt-16 border-t bg-muted/30 py-8 text-xs text-muted-foreground">
        <div className="mx-auto flex max-w-6xl flex-col items-center justify-between gap-4 px-4 sm:flex-row">
          <span>
            Banco del Instituto Ecuatoriano de Seguridad Social · Dirección
            Nacional de Tecnologías de la Información
          </span>
          <div className="flex items-center gap-4">
            <span>Canal seguro y cifrado</span>
            <span aria-hidden="true">·</span>
            <span>Datos protegidos por LOPDP</span>
          </div>
        </div>
      </footer>
    </div>
  )
}
