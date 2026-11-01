import { useEffect, useState } from "react"
import { useNavigate } from "react-router"
import {
  AlertCircleIcon,
  AlertTriangleIcon,
  ArrowLeftIcon,
  ArrowRightIcon,
  CheckCircle2Icon,
  ClockIcon,
  LockIcon,
  PlayIcon,
  SendIcon,
  ShieldAlertIcon,
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
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Progress } from "@/components/ui/progress"
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group"
import { Label } from "@/components/ui/label"
import { useListarPostulaciones } from "@/features/postulaciones/queries"
import {
  useEstadoExamen,
  useFinalizarExamen,
  useIniciarExamen,
  usePreguntaExamen,
  useResponderExamen,
} from "@/features/preguntas/queries"
import { RUTAS } from "@/lib/rutas"

/**
 * Sala de Examen en tiempo real (EF-14 / FT26).
 * Presenta navegación secuencial de reactivos, temporizador sincronizado con el servidor,
 * guardado optimista por pregunta y cierre automático o manual.
 */
export function SalaExamen() {
  const navigate = useNavigate()

  const { data: postulacionesData, isPending: cargandoPostulacion } =
    useListarPostulaciones({ limit: 1 })
  const postulacion = postulacionesData?.items?.[0]
  const submissionId = postulacion?.id ?? ""

  // Estado del examen en servidor
  const consultaExamen = useEstadoExamen(submissionId, Boolean(submissionId))
  const examen = consultaExamen.data

  const mutacionIniciar = useIniciarExamen(submissionId)
  const mutacionResponder = useResponderExamen(submissionId)
  const mutacionFinalizar = useFinalizarExamen(submissionId)

  // Índice de navegación de preguntas (1 a cantidadPreguntas)
  const [indiceActual, setIndiceActual] = useState(1)

  // Consulta de la pregunta actual
  const esActivo = examen?.estado === "active"
  const consultaPregunta = usePreguntaExamen(
    submissionId,
    indiceActual,
    esActivo,
  )
  const pregunta = consultaPregunta.data

  // Estado local de respuestas seleccionadas por preguntaId
  const [respuestasLocales, setRespuestasLocales] = useState<
    Record<string, string>
  >({})
  const [modalFinalizarAbierto, setModalFinalizarAbierto] = useState(false)

  // Temporizador regresivo en segundos sincronizado con el servidor
  const [ahora, setAhora] = useState(() => Date.now())

  useEffect(() => {
    if (!examen?.venceEn || examen.estado !== "active") return

    const intervalo = setInterval(() => {
      const actual = Date.now()
      setAhora(actual)

      const fechaFin = new Date(examen.venceEn).getTime()
      if (fechaFin <= actual) {
        clearInterval(intervalo)
        // Cierre automático al llegar a 00:00
        mutacionFinalizar.mutate(undefined, {
          onSuccess: () => {
            navigate(RUTAS.evaluacionResultado)
          },
        })
      }
    }, 1000)

    return () => clearInterval(intervalo)
  }, [examen?.venceEn, examen?.estado, mutacionFinalizar, navigate])

  const segundosRestantes =
    examen?.venceEn && examen.estado === "active"
      ? Math.max(
          0,
          Math.floor((new Date(examen.venceEn).getTime() - ahora) / 1000),
        )
      : 0

  if (cargandoPostulacion || consultaExamen.isPending) {
    return (
      <EstadoCargando mensaje="Cargando sala de examen de conocimientos…" />
    )
  }

  if (consultaExamen.isError) {
    return (
      <EstadoError
        error={consultaExamen.error}
        titulo="No se pudo acceder a la sala de examen"
        onReintentar={() => void consultaExamen.refetch()}
      />
    )
  }

  // Si el examen ya está finalizado, dirigir a la pantalla de resultados
  if (examen?.estado === "finished") {
    return (
      <Card className="mx-auto max-w-xl text-center">
        <CardHeader>
          <div className="mx-auto flex size-12 items-center justify-center rounded-full bg-emerald-500/10 text-emerald-600">
            <CheckCircle2Icon className="size-6" />
          </div>
          <CardTitle className="mt-2 text-xl">
            Evaluación Técnica Concluida
          </CardTitle>
          <CardDescription>
            Usted ya ha rendido y calificado su intento oficial de evaluación
            técnica.
          </CardDescription>
        </CardHeader>
        <CardFooter className="justify-center">
          <Button asChild>
            <a href={RUTAS.evaluacionResultado}>Ver resultados obtenidos</a>
          </Button>
        </CardFooter>
      </Card>
    )
  }

  // Si aún no se ha iniciado el examen
  if (!esActivo) {
    return (
      <Card className="mx-auto max-w-2xl">
        <CardHeader>
          <div className="flex items-center gap-2 text-primary">
            <LockIcon className="size-5" />
            <span className="text-xs font-semibold tracking-wider uppercase">
              Oposición Teórica
            </span>
          </div>
          <CardTitle className="mt-1 text-2xl">
            Examen Teórico de Conocimientos
          </CardTitle>
          <CardDescription>
            Sala oficial de evaluación técnica para postulantes habilitados.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <Alert className="border-amber-500/30 bg-amber-500/5 text-amber-900 dark:text-amber-200">
            <AlertTriangleIcon className="size-5 text-amber-600 dark:text-amber-400" />
            <AlertTitle className="font-semibold">
              Condiciones de Aplicación Oficial
            </AlertTitle>
            <AlertDescription className="space-y-2 text-xs leading-relaxed">
              <p>
                • <strong>Un solo intento:</strong> Una vez que presione
                &quot;Iniciar Evaluación Oficial&quot;, su intento quedará
                bloqueado y comenzará a correr el tiempo en el servidor.
              </p>
              <p>
                • <strong>Tiempo ininterrumpido:</strong> El reloj no se detiene
                si cierra la ventana o recarga la página.
              </p>
              <p>
                • <strong>Navegación y guardado:</strong> Podrá navegar entre
                preguntas. Cada opción elegida se guarda en el servidor
                automáticamente.
              </p>
              <p>
                • <strong>Calificación automática:</strong> Al finalizar o al
                agotarse el tiempo, sus respuestas serán evaluadas
                instantáneamente.
              </p>
            </AlertDescription>
          </Alert>

          <div className="rounded-lg border bg-muted/30 p-4 text-xs">
            <span className="font-medium text-muted-foreground">
              Recomendación técnica:
            </span>
            <p className="mt-1 text-muted-foreground">
              Asegúrese de contar con una conexión a internet estable. Si
              experimenta una desconexión accidental, reingrese a esta misma URL
              y su tiempo y respuestas continuarán activos.
            </p>
          </div>
        </CardContent>
        <CardFooter className="flex justify-between">
          <Button asChild variant="outline">
            <a href={RUTAS.evaluacion}>Regresar al portal</a>
          </Button>
          <Button
            size="lg"
            onClick={() => mutacionIniciar.mutate()}
            disabled={mutacionIniciar.isPending}
          >
            <PlayIcon className="mr-2 size-4" />
            {mutacionIniciar.isPending
              ? "Iniciando sala…"
              : "Iniciar Evaluación Oficial"}
          </Button>
        </CardFooter>
      </Card>
    )
  }

  // Formatear segundos en MM:SS
  const minutos = Math.floor(segundosRestantes / 60)
  const segundos = segundosRestantes % 60
  const tiempoFormateado = `${String(minutos).padStart(2, "0")}:${String(segundos).padStart(2, "0")}`
  const esTiempoCritico = segundosRestantes < 300 // Menos de 5 minutos

  const totalPreguntas = examen.cantidadPreguntas || 1
  const opcionSeleccionadaActual =
    (pregunta?.id ? respuestasLocales[pregunta.id] : undefined) ||
    pregunta?.seleccionada ||
    ""

  const handleSeleccionarOpcion = (opcionId: string) => {
    if (!pregunta?.id) return

    setRespuestasLocales((prev) => ({
      ...prev,
      [pregunta.id]: opcionId,
    }))

    mutacionResponder.mutate({
      versionAnterior: pregunta.versionIntento,
      preguntaId: pregunta.id,
      opcionId,
    })
  }

  const handleConfirmarFinalizar = () => {
    setModalFinalizarAbierto(false)
    mutacionFinalizar.mutate(undefined, {
      onSuccess: () => {
        navigate(RUTAS.evaluacionResultado)
      },
    })
  }

  return (
    <div className="space-y-6">
      {/* Barra de Cabecera Fija / Reloj y Progreso */}
      <Card className="border-primary/20 bg-card shadow-sm">
        <CardContent className="p-4 sm:p-6">
          <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
            <div className="space-y-1">
              <div className="flex items-center gap-2">
                <Badge variant="outline" className="text-xs font-semibold">
                  Intento Oficial Activo
                </Badge>
                <span className="text-xs text-muted-foreground">
                  Versión {examen.version}
                </span>
              </div>
              <h2 className="text-lg font-bold text-foreground">
                Examen de Conocimientos BIESS
              </h2>
            </div>

            {/* Cronómetro regresivo sincronizado */}
            <div className="flex items-center gap-3">
              <div
                className={`flex items-center gap-2 rounded-lg border px-3 py-1.5 font-mono text-lg font-bold sm:text-xl ${
                  esTiempoCritico
                    ? "border-destructive/50 bg-destructive/10 text-destructive animate-pulse"
                    : "border-primary/30 bg-primary/5 text-primary"
                }`}
              >
                <ClockIcon className="size-5 shrink-0" />
                <span>{tiempoFormateado}</span>
              </div>

              <Button
                variant="destructive"
                size="sm"
                onClick={() => setModalFinalizarAbierto(true)}
              >
                <SendIcon className="mr-1.5 size-3.5" />
                Finalizar
              </Button>
            </div>
          </div>

          <div className="mt-4 space-y-1.5">
            <div className="flex justify-between text-xs text-muted-foreground">
              <span>
                Respondidas: {examen.respondidas} de {totalPreguntas}
              </span>
              <span>
                {Math.round((examen.respondidas / totalPreguntas) * 100)}%
                completado
              </span>
            </div>
            <Progress
              value={(examen.respondidas / totalPreguntas) * 100}
              className="h-1.5"
            />
          </div>
        </CardContent>
      </Card>

      {/* Tira numerada de preguntas */}
      <Card>
        <CardContent className="p-3 sm:p-4">
          <div className="flex flex-wrap items-center gap-1.5">
            {Array.from({ length: totalPreguntas }, (_, i) => i + 1).map(
              (num) => {
                const esActual = num === indiceActual
                return (
                  <Button
                    key={num}
                    variant={esActual ? "default" : "outline"}
                    size="sm"
                    className={`size-8 p-0 text-xs font-mono ${
                      esActual
                        ? "font-bold ring-2 ring-primary ring-offset-2"
                        : ""
                    }`}
                    onClick={() => setIndiceActual(num)}
                  >
                    {num}
                  </Button>
                )
              },
            )}
          </div>
        </CardContent>
      </Card>

      {/* Contenedor de la Pregunta Actual */}
      <Card
        className="select-none"
        onCopy={(e) => {
          e.preventDefault()
          alert(
            "Por seguridad institucional, la copia de reactivos está inhabilitada durante la evaluación.",
          )
        }}
      >
        <CardHeader>
          <div className="flex items-center justify-between text-xs text-muted-foreground">
            <span className="font-semibold text-primary">
              Pregunta {indiceActual} de {totalPreguntas}
            </span>
            {mutacionResponder.isPending && (
              <span className="text-[11px] text-amber-600 animate-pulse">
                Guardando respuesta en servidor…
              </span>
            )}
          </div>
          {consultaPregunta.isPending ? (
            <EstadoCargando mensaje="Cargando pregunta…" />
          ) : consultaPregunta.isError ? (
            <EstadoError
              error={consultaPregunta.error}
              titulo="No se pudo cargar la pregunta"
              onReintentar={() => void consultaPregunta.refetch()}
            />
          ) : (
            <CardTitle className="mt-2 text-base font-medium leading-relaxed sm:text-lg">
              {pregunta?.enunciado}
            </CardTitle>
          )}
        </CardHeader>

        {pregunta && (
          <CardContent>
            <RadioGroup
              value={opcionSeleccionadaActual}
              onValueChange={handleSeleccionarOpcion}
              className="space-y-3"
            >
              {pregunta.opciones.map((op) => {
                const esSeleccionada = opcionSeleccionadaActual === op.id
                return (
                  <div
                    key={op.id}
                    onClick={() => handleSeleccionarOpcion(op.id)}
                    className={`flex cursor-pointer items-start gap-3 rounded-lg border p-3 text-sm transition-colors sm:p-4 ${
                      esSeleccionada
                        ? "border-primary bg-primary/5 font-medium text-foreground"
                        : "hover:bg-muted/50 text-muted-foreground"
                    }`}
                  >
                    <RadioGroupItem
                      value={op.id}
                      id={`opcion-${op.id}`}
                      className="mt-0.5"
                    />
                    <Label
                      htmlFor={`opcion-${op.id}`}
                      className="flex-1 cursor-pointer leading-normal text-inherit"
                    >
                      <span className="mr-2 font-mono font-bold uppercase text-foreground">
                        {op.id})
                      </span>
                      <span>{op.texto}</span>
                    </Label>
                  </div>
                )
              })}
            </RadioGroup>
          </CardContent>
        )}

        <CardFooter className="flex items-center justify-between border-t bg-muted/20 px-6 py-4">
          <Button
            variant="outline"
            size="sm"
            onClick={() => setIndiceActual((prev) => Math.max(1, prev - 1))}
            disabled={indiceActual === 1}
          >
            <ArrowLeftIcon className="mr-1.5 size-4" />
            Anterior
          </Button>

          {indiceActual < totalPreguntas ? (
            <Button
              size="sm"
              onClick={() =>
                setIndiceActual((prev) => Math.min(totalPreguntas, prev + 1))
              }
            >
              Siguiente
              <ArrowRightIcon className="ml-1.5 size-4" />
            </Button>
          ) : (
            <Button
              size="sm"
              variant="default"
              className="bg-emerald-600 hover:bg-emerald-700 text-white"
              onClick={() => setModalFinalizarAbierto(true)}
            >
              Revisar y finalizar
              <CheckCircle2Icon className="ml-1.5 size-4" />
            </Button>
          )}
        </CardFooter>
      </Card>

      {/* Modal de Confirmación para Finalizar Examen */}
      <Dialog
        open={modalFinalizarAbierto}
        onOpenChange={setModalFinalizarAbierto}
      >
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <ShieldAlertIcon className="size-5 text-amber-500" />
              ¿Confirmar finalización del examen?
            </DialogTitle>
            <DialogDescription>
              Revise el resumen de sus respuestas antes de calificar.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-2">
            <div className="rounded-lg border bg-muted/40 p-4 text-xs space-y-2">
              <div className="flex justify-between">
                <span className="text-muted-foreground">Total preguntas:</span>
                <span className="font-bold">{totalPreguntas}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">
                  Preguntas respondidas:
                </span>
                <span className="font-bold text-emerald-600">
                  {examen.respondidas}
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Sin responder:</span>
                <span className="font-bold text-destructive">
                  {totalPreguntas - examen.respondidas}
                </span>
              </div>
            </div>

            {totalPreguntas - examen.respondidas > 0 && (
              <Alert variant="destructive">
                <AlertCircleIcon className="size-4" />
                <AlertTitle>Preguntas pendientes</AlertTitle>
                <AlertDescription className="text-xs">
                  Aún tiene {totalPreguntas - examen.respondidas} pregunta(s)
                  sin responder. Toda pregunta no contestada se calificará con
                  cero puntos.
                </AlertDescription>
              </Alert>
            )}

            <p className="text-xs text-muted-foreground">
              Al confirmar, el intento se cerrará de forma permanente y se
              calculará su calificación oficial en el servidor.
            </p>
          </div>

          <DialogFooter className="gap-2 sm:gap-0">
            <Button
              variant="outline"
              onClick={() => setModalFinalizarAbierto(false)}
            >
              Continuar respondiendo
            </Button>
            <Button
              variant="destructive"
              onClick={handleConfirmarFinalizar}
              disabled={mutacionFinalizar.isPending}
            >
              {mutacionFinalizar.isPending
                ? "Calificando…"
                : "Finalizar y calificar"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
