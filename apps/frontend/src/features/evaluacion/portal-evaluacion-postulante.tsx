import { useState } from "react"
import { Link } from "react-router"
import {
  AlertCircleIcon,
  AwardIcon,
  CheckCircle2Icon,
  ClockIcon,
  DownloadIcon,
  FileTextIcon,
  PlayIcon,
  ShieldAlertIcon,
  XCircleIcon,
} from "lucide-react"

import { EstadoCargando, EstadoError } from "@/components/estados"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Progress } from "@/components/ui/progress"
import { useListarPostulaciones } from "@/features/postulaciones/queries"
import { useResultadoMeritosPostulante } from "@/features/revision/queries"
import { useEstadoExamen } from "@/features/preguntas/queries"
import { esErrorApi } from "@/lib/api-error"
import { centipuntosAPuntos } from "@/lib/clientes/meritos"
import { descargarReporteExamenPdf } from "@/lib/clientes/preguntas"
import { RUTAS } from "@/lib/rutas"
import { formatearFechaEcuador } from "@/utils/fechas"

/**
 * Portal del Postulante para la Evaluación Técnica EF-14 (FT25).
 * Valida elegibilidad, requisitos de méritos, horario oficial y sala de examen.
 */
export function PortalEvaluacionPostulante() {
  const { data: postulacionesData, isPending: cargandoPostulacion } =
    useListarPostulaciones({ limit: 1 })
  const postulacion = postulacionesData?.items?.[0]
  const submissionId = postulacion?.id ?? ""

  // Consultar méritos para verificar elegibilidad
  const { data: meritosData, isPending: cargandoMeritos } =
    useResultadoMeritosPostulante(submissionId)
  const meritosPublicados = Boolean(
    meritosData?.publicado && meritosData.resultado,
  )
  const meritosAvanza = Boolean(
    meritosData?.habilitadoFaseTecnica ||
    meritosData?.resultado?.decision === "avanza",
  )

  // Consultar estado del examen
  const consultaExamen = useEstadoExamen(submissionId, Boolean(submissionId))
  const examen = consultaExamen.data

  const [descargandoPdf, setDescargandoPdf] = useState(false)

  if (cargandoPostulacion || cargandoMeritos) {
    return (
      <EstadoCargando mensaje="Verificando elegibilidad y estado de evaluación técnica…" />
    )
  }

  if (!postulacion) {
    return (
      <Alert>
        <AlertCircleIcon className="size-4" />
        <AlertTitle>Sin postulación registrada</AlertTitle>
        <AlertDescription>
          No se encontró un expediente activo asociado a su cuenta.
        </AlertDescription>
      </Alert>
    )
  }

  const handleDescargarPdf = async () => {
    try {
      setDescargandoPdf(true)
      const blob = await descargarReporteExamenPdf(submissionId)
      const url = URL.createObjectURL(blob)
      const a = document.createElement("a")
      a.href = url
      a.download = `reporte-evaluacion-${submissionId.slice(0, 8)}.pdf`
      a.click()
      URL.revokeObjectURL(url)
    } catch {
      // Error manejado en api
    } finally {
      setDescargandoPdf(false)
    }
  }

  // Determinar estado de la fase
  const es404 =
    consultaExamen.isError &&
    esErrorApi(consultaExamen.error) &&
    consultaExamen.error.estado === 404

  const esExamenActivo = examen?.estado === "active"
  const esExamenFinalizado = examen?.estado === "finished"

  return (
    <div className="space-y-6">
      {/* Encabezado */}
      <div className="flex flex-col gap-1">
        <div className="flex items-center gap-2 text-primary">
          <AwardIcon className="size-5" />
          <span className="text-xs font-semibold tracking-wider uppercase">
            Fase de Oposición
          </span>
        </div>
        <h1 className="text-2xl font-bold tracking-tight text-foreground sm:text-3xl">
          Evaluación Técnica de Conocimientos (EF-14)
        </h1>
        <p className="text-sm text-muted-foreground sm:text-base">
          Módulo oficial de rendición de pruebas teóricas para postulantes
          habilitados del concurso BIESS.
        </p>
      </div>

      {/* Tarjeta de Elegibilidad y Requisitos */}
      <Card>
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <CardTitle className="text-base font-semibold">
              Elegibilidad y Verificación de Fase Previa
            </CardTitle>
            {meritosAvanza ? (
              <Badge className="bg-emerald-500/10 text-emerald-700 hover:bg-emerald-500/20 dark:text-emerald-300">
                <CheckCircle2Icon className="mr-1 size-3.5" />
                Habilitado por Méritos
              </Badge>
            ) : meritosPublicados ? (
              <Badge variant="destructive">
                <XCircleIcon className="mr-1 size-3.5" />
                No Habilitado
              </Badge>
            ) : (
              <Badge variant="secondary">
                <ClockIcon className="mr-1 size-3.5" />
                Méritos en Revisión
              </Badge>
            )}
          </div>
          <CardDescription className="text-xs">
            Conforme a la normativa del concurso, sólo los postulantes con
            resolución aprobatoria de méritos son convocados a la evaluación
            técnica.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="rounded-md border p-3">
              <span className="text-xs font-medium text-muted-foreground">
                Fase 1: Méritos y Requisitos
              </span>
              <div className="mt-1 flex items-baseline gap-2">
                <span className="text-lg font-bold">
                  {meritosPublicados && meritosData?.resultado?.evaluacion
                    ? `${centipuntosAPuntos(meritosData.resultado.evaluacion.totalCentipuntos)} pts`
                    : "Pendiente"}
                </span>
                {meritosPublicados && (
                  <span className="text-xs text-muted-foreground">
                    ({meritosAvanza ? "Supera umbral" : "No alcanza umbral"})
                  </span>
                )}
              </div>
            </div>

            <div className="rounded-md border p-3">
              <span className="text-xs font-medium text-muted-foreground">
                Fase 2: Evaluación Técnica
              </span>
              <div className="mt-1 flex items-baseline gap-2">
                <span className="text-lg font-bold">
                  {esExamenFinalizado
                    ? `${examen.resultado?.puntaje ?? 0} pts`
                    : esExamenActivo
                      ? "En curso"
                      : meritosAvanza
                        ? "Habilitada"
                        : "Bloqueada"}
                </span>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Estado del Examen */}
      {consultaExamen.isPending ? (
        <EstadoCargando mensaje="Consultando sala de evaluación del servidor…" />
      ) : es404 ? (
        // Caso: Sin habilitación o evaluación aún no asignada
        <Card className="border-dashed">
          <CardHeader>
            <CardTitle className="text-base">
              Sala de Evaluación No Disponible
            </CardTitle>
            <CardDescription>
              {meritosAvanza
                ? "Su postulación ha sido aprobada en la fase de méritos. La programación del horario y apertura de la sala está siendo procesada por la Comisión de Selección."
                : "Usted no se encuentra habilitado para rendir esta evaluación debido a que su resultado de méritos no alcanzó la condición de avance."}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Alert className="border-muted bg-muted/30">
              <ShieldAlertIcon className="size-4 text-muted-foreground" />
              <AlertDescription className="text-xs text-muted-foreground">
                Las fechas, horas de apertura y términos son establecidos
                exclusivamente por la Comisión Especial de Selección en la zona
                horaria oficial de Ecuador (America/Guayaquil, UTC-5).
              </AlertDescription>
            </Alert>
          </CardContent>
          <CardFooter>
            <Button asChild variant="outline">
              <Link to={RUTAS.seguimiento}>Ver seguimiento de etapas</Link>
            </Button>
          </CardFooter>
        </Card>
      ) : consultaExamen.isError ? (
        <EstadoError
          error={consultaExamen.error}
          titulo="No se pudo comprobar la sala de evaluación"
          onReintentar={() => void consultaExamen.refetch()}
        />
      ) : esExamenActivo ? (
        // Caso: Intento en curso
        <Card className="border-primary/40 bg-primary/5">
          <CardHeader>
            <div className="flex items-center justify-between">
              <Badge className="bg-primary text-primary-foreground">
                <ClockIcon className="mr-1 size-3.5" />
                Intento Activo en Curso
              </Badge>
              <span className="font-mono text-xs text-muted-foreground">
                Versión {examen.version}
              </span>
            </div>
            <CardTitle className="mt-2 text-lg">
              Evaluación Técnica Iniciada
            </CardTitle>
            <CardDescription>
              Tiene una evaluación oficial en progreso. El cronómetro se ejecuta
              de manera estricta en el servidor.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <div className="flex justify-between text-xs">
                <span>Progreso de respuestas</span>
                <span className="font-semibold">
                  {examen.respondidas} de {examen.cantidadPreguntas} preguntas
                </span>
              </div>
              <Progress
                value={
                  (examen.respondidas / (examen.cantidadPreguntas || 1)) * 100
                }
              />
            </div>

            <div className="rounded-md border bg-background p-4 text-xs">
              <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
                <div>
                  <span className="text-muted-foreground">Iniciada:</span>
                  <p className="font-medium">
                    {formatearFechaEcuador(examen.iniciadaEn, {
                      incluirHora: true,
                    })}
                  </p>
                </div>
                <div>
                  <span className="text-muted-foreground">Vence:</span>
                  <p className="font-medium text-destructive">
                    {formatearFechaEcuador(examen.venceEn, {
                      incluirHora: true,
                    })}
                  </p>
                </div>
                <div>
                  <span className="text-muted-foreground">
                    Tiempo restante aproximado:
                  </span>
                  <p className="font-mono font-bold">
                    {Math.max(0, Math.floor(examen.segundosRestantes / 60))} min
                  </p>
                </div>
              </div>
            </div>
          </CardContent>
          <CardFooter className="gap-2">
            <Button asChild size="lg">
              <Link to={RUTAS.evaluacionExamen}>
                <PlayIcon className="mr-1.5 size-4" />
                Reanudar evaluación
              </Link>
            </Button>
          </CardFooter>
        </Card>
      ) : esExamenFinalizado ? (
        // Caso: Intento finalizado y calificado
        <Card className="border-emerald-500/30 bg-emerald-500/5">
          <CardHeader>
            <div className="flex items-center justify-between">
              <Badge className="bg-emerald-600 text-white">
                <CheckCircle2Icon className="mr-1 size-3.5" />
                Evaluación Calificada
              </Badge>
              <span className="font-mono text-xs text-muted-foreground">
                Versión {examen.version}
              </span>
            </div>
            <CardTitle className="mt-2 text-lg">
              Evaluación Técnica Finalizada
            </CardTitle>
            <CardDescription>
              Su prueba ha sido calificada de forma automática por el motor del
              servidor y consolidada en su expediente.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="rounded-lg border bg-background p-4">
              <div className="flex items-baseline justify-between">
                <div>
                  <span className="text-xs text-muted-foreground">
                    Calificación Obtenida:
                  </span>
                  <div className="text-3xl font-extrabold text-foreground">
                    {examen.resultado?.puntaje ?? 0}
                    <span className="text-sm font-normal text-muted-foreground">
                      {" "}
                      / {examen.resultado?.maximo ?? 100} puntos
                    </span>
                  </div>
                </div>
                <Badge
                  variant={
                    examen.resultado?.habilitaAvance ? "default" : "secondary"
                  }
                >
                  {examen.resultado?.habilitaAvance
                    ? "Aprueba Fase"
                    : "No Supera Umbral"}
                </Badge>
              </div>
            </div>
          </CardContent>
          <CardFooter className="flex flex-wrap gap-2">
            <Button asChild variant="outline">
              <Link to={RUTAS.evaluacionResultado}>
                <FileTextIcon className="mr-1.5 size-4" />
                Ver resultado detallado
              </Link>
            </Button>
            <Button
              variant="secondary"
              onClick={handleDescargarPdf}
              disabled={descargandoPdf}
            >
              <DownloadIcon className="mr-1.5 size-4" />
              {descargandoPdf ? "Descargando…" : "Descargar reporte PDF"}
            </Button>
          </CardFooter>
        </Card>
      ) : (
        // Caso: Habilitado pero aún no iniciado
        <Card>
          <CardHeader>
            <CardTitle className="text-base">
              Instrucciones y Condiciones de Rendición
            </CardTitle>
            <CardDescription>
              Revise atentamente las reglas oficiales antes de ingresar a la
              evaluación.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3 text-sm text-muted-foreground">
            <ul className="list-disc space-y-2 pl-5">
              <li>
                <strong>Un solo intento oficial:</strong> No se admiten
                reintentos ni reinicios. Una vez presionado el botón de inicio,
                el tiempo comienza a transcurrir.
              </li>
              <li>
                <strong>Cronómetro del servidor:</strong> El plazo se valida en
                el servidor; no depende del reloj de su dispositivo ni de si
                cierra la ventana.
              </li>
              <li>
                <strong>Guardado por pregunta:</strong> Cada respuesta se guarda
                de forma individual e inmediata. Si se desconecta, puede volver
                a ingresar y sus respuestas previas continuarán guardadas.
              </li>
              <li>
                <strong>Navegación secuencial:</strong> Se presenta una pregunta
                a la vez, con permutación aleatoria criptográfica única de
                opciones.
              </li>
            </ul>
          </CardContent>
          <CardFooter>
            <Button asChild size="lg">
              <Link to={RUTAS.evaluacionExamen}>
                <PlayIcon className="mr-1.5 size-4" />
                Ingresar a la sala de examen
              </Link>
            </Button>
          </CardFooter>
        </Card>
      )}
    </div>
  )
}
