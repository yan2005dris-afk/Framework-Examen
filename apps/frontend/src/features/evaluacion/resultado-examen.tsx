import { useState } from "react"
import { Link } from "react-router"
import {
  AwardIcon,
  CheckCircle2Icon,
  ClockIcon,
  DownloadIcon,
  FileCheckIcon,
  FileTextIcon,
  LayersIcon,
  ShieldCheckIcon,
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
import { useListarPostulaciones } from "@/features/postulaciones/queries"
import { useEstadoExamen } from "@/features/preguntas/queries"
import { descargarReporteExamenPdf } from "@/lib/clientes/preguntas"
import { RUTAS } from "@/lib/rutas"
import { formatearFechaEcuador } from "@/utils/fechas"

/**
 * Pantalla de Resultado de la Evaluación Técnica EF-14 (FT27).
 * Presenta la nota calculada por el motor del servidor, condición de avance,
 * trazabilidad e integración al expediente general y descarga del reporte PDF.
 */
export function ResultadoExamen() {
  const { data: postulacionesData, isPending: cargandoPostulacion } =
    useListarPostulaciones({ limit: 1 })
  const postulacion = postulacionesData?.items?.[0]
  const submissionId = postulacion?.id ?? ""

  const consultaExamen = useEstadoExamen(submissionId, Boolean(submissionId))
  const examen = consultaExamen.data

  const [descargandoPdf, setDescargandoPdf] = useState(false)

  if (cargandoPostulacion || consultaExamen.isPending) {
    return <EstadoCargando mensaje="Consultando acta oficial de examen…" />
  }

  if (consultaExamen.isError) {
    return (
      <EstadoError
        error={consultaExamen.error}
        titulo="No se pudo obtener el resultado del examen"
        onReintentar={() => void consultaExamen.refetch()}
      />
    )
  }

  if (!examen || examen.estado !== "finished") {
    return (
      <Card className="mx-auto max-w-xl text-center">
        <CardHeader>
          <div className="mx-auto flex size-12 items-center justify-center rounded-full bg-amber-500/10 text-amber-600">
            <ClockIcon className="size-6" />
          </div>
          <CardTitle className="mt-2 text-xl">
            Evaluación Aún No Finalizada
          </CardTitle>
          <CardDescription>
            {examen?.estado === "active"
              ? "Usted tiene un examen actualmente en curso. Para consultar sus resultados debe concluir su intento."
              : "No se registra un examen finalizado para esta postulación."}
          </CardDescription>
        </CardHeader>
        <CardFooter className="justify-center gap-3">
          {examen?.estado === "active" ? (
            <Button asChild>
              <Link to={RUTAS.evaluacionExamen}>Reanudar examen</Link>
            </Button>
          ) : (
            <Button asChild variant="outline">
              <Link to={RUTAS.evaluacion}>Ir al portal de evaluación</Link>
            </Button>
          )}
        </CardFooter>
      </Card>
    )
  }

  const handleDescargarPdf = async () => {
    try {
      setDescargandoPdf(true)
      const blob = await descargarReporteExamenPdf(submissionId)
      const url = URL.createObjectURL(blob)
      const a = document.createElement("a")
      a.href = url
      a.download = `acta-evaluacion-${submissionId.slice(0, 8)}.pdf`
      a.click()
      URL.revokeObjectURL(url)
    } catch {
      // Error manejado en api
    } finally {
      setDescargandoPdf(false)
    }
  }

  const resultado = examen.resultado
  const aprobo = resultado?.habilitaAvance === true
  const puntaje = resultado?.puntaje ?? 0
  const maximo = resultado?.maximo ?? 100

  return (
    <div className="space-y-6">
      {/* Encabezado */}
      <div className="flex flex-col gap-1">
        <div className="flex items-center gap-2 text-primary">
          <AwardIcon className="size-5" />
          <span className="text-xs font-semibold tracking-wider uppercase">
            Acta Oficial de Calificación
          </span>
        </div>
        <h1 className="text-2xl font-bold tracking-tight text-foreground sm:text-3xl">
          Resultado de Evaluación Técnica (EF-14)
        </h1>
        <p className="text-sm text-muted-foreground sm:text-base">
          Calificación definitiva calculada por el servidor e incorporada al
          expediente de oposición del concurso BIESS.
        </p>
      </div>

      {/* Tarjeta Principal de Puntaje y Estado */}
      <Card
        className={`border-2 ${
          aprobo
            ? "border-emerald-500/40 bg-emerald-500/5"
            : "border-destructive/40 bg-destructive/5"
        }`}
      >
        <CardContent className="p-6 sm:p-8">
          <div className="flex flex-col gap-6 sm:flex-row sm:items-center sm:justify-between">
            <div className="space-y-2">
              <div className="flex items-center gap-2">
                <Badge
                  className={
                    aprobo
                      ? "bg-emerald-600 text-white"
                      : "bg-destructive text-destructive-foreground"
                  }
                >
                  {aprobo ? (
                    <>
                      <CheckCircle2Icon className="mr-1 size-3.5" />
                      Supera Umbral Mínimo
                    </>
                  ) : (
                    <>
                      <XCircleIcon className="mr-1 size-3.5" />
                      No Alcanza Umbral
                    </>
                  )}
                </Badge>
                <span className="font-mono text-xs text-muted-foreground">
                  Versión {examen.version}
                </span>
              </div>
              <h2 className="text-xl font-bold text-foreground sm:text-2xl">
                {aprobo
                  ? "Aprobación de Fase Técnica"
                  : "Calificación Insuficiente"}
              </h2>
              <p className="max-w-xl text-xs text-muted-foreground sm:text-sm">
                {resultado?.motivo ||
                  "Calificación obtenida según el número de respuestas correctas sobre la cantidad total de reactivos aplicados."}
              </p>
            </div>

            {/* Visualización de la Nota */}
            <div className="flex flex-col items-center justify-center rounded-xl border bg-background p-6 text-center shadow-sm sm:min-w-44">
              <span className="text-xs font-medium text-muted-foreground">
                Nota Final Obtenida
              </span>
              <div className="mt-1 font-mono text-4xl font-black text-foreground">
                {puntaje}
              </div>
              <span className="text-xs text-muted-foreground">
                de {maximo} pts
              </span>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Desglose y Detalles del Intento */}
      <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="flex items-center gap-2 text-base font-semibold">
              <FileCheckIcon className="size-4 text-primary" />
              Datos de Rendición de la Prueba
            </CardTitle>
            <CardDescription className="text-xs">
              Tiempos certificados y registro del intento oficial.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3 text-xs">
            <div className="flex justify-between border-b pb-2">
              <span className="text-muted-foreground">Inicio del examen:</span>
              <span className="font-medium text-foreground">
                {formatearFechaEcuador(examen.iniciadaEn, {
                  incluirHora: true,
                })}
              </span>
            </div>

            <div className="flex justify-between border-b pb-2">
              <span className="text-muted-foreground">Cierre del examen:</span>
              <span className="font-medium text-foreground">
                {formatearFechaEcuador(examen.venceEn, {
                  incluirHora: true,
                })}
              </span>
            </div>

            <div className="flex justify-between border-b pb-2">
              <span className="text-muted-foreground">
                Total reactivos aplicados:
              </span>
              <span className="font-mono font-bold text-foreground">
                {examen.cantidadPreguntas} preguntas
              </span>
            </div>

            <div className="flex justify-between">
              <span className="text-muted-foreground">
                Preguntas contestadas:
              </span>
              <span className="font-mono font-medium text-foreground">
                {examen.respondidas} de {examen.cantidadPreguntas}
              </span>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="flex items-center gap-2 text-base font-semibold">
              <ShieldCheckIcon className="size-4 text-primary" />
              Garantía de Calificación e Integración
            </CardTitle>
            <CardDescription className="text-xs">
              Conexión directa con el expediente general de méritos y oposición.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3 text-xs leading-relaxed text-muted-foreground">
            <p>
              • <strong>Motor de Calificación examen-v1:</strong> La nota se
              computa mediante comparación contra la matriz congelada en el
              servidor, sin injerencia del navegador cliente.
            </p>
            <p>
              • <strong>Inmutabilidad:</strong> Este resultado se incorpora
              directamente a la fase de oposición para consolidarse con el
              resultado de méritos y psicotécnica (FT20).
            </p>
            <p>
              • <strong>Reporte PDF firmado:</strong> Un acta inmutable con
              sello criptográfico SHA-256 ha sido registrada en el sistema.
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Aviso institucional y acciones */}
      <Alert className="border-primary/20 bg-primary/5">
        <FileTextIcon className="size-5 text-primary" />
        <AlertTitle className="text-foreground">
          Acta y Notificación por Correo
        </AlertTitle>
        <AlertDescription className="text-xs text-muted-foreground">
          Una copia de este resultado ha sido encolada para su envío formal a su
          correo electrónico registrado. Puede descargar el acta oficial en PDF
          en cualquier momento.
        </AlertDescription>
      </Alert>

      <div className="flex flex-wrap items-center justify-between gap-3 border-t pt-4">
        <div className="flex flex-wrap gap-2">
          <Button asChild variant="outline">
            <Link to={RUTAS.evaluacion}>
              <LayersIcon className="mr-1.5 size-4" />
              Expediente de oposición
            </Link>
          </Button>
          <Button asChild variant="outline">
            <Link to={RUTAS.seguimiento}>Seguimiento de fases</Link>
          </Button>
        </div>

        <Button onClick={handleDescargarPdf} disabled={descargandoPdf}>
          <DownloadIcon className="mr-1.5 size-4" />
          {descargandoPdf
            ? "Descargando acta…"
            : "Descargar Acta Oficial (PDF)"}
        </Button>
      </div>
    </div>
  )
}
