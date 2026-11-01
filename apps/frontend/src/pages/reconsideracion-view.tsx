import { useState } from "react"
import { Link, useSearchParams } from "react-router"
import {
  AlertCircleIcon,
  FileTextIcon,
  HelpCircleIcon,
  ScaleIcon,
  SendIcon,
  ShieldAlertIcon,
} from "lucide-react"

import { EstadoCargando } from "@/components/estados"
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
import { Checkbox } from "@/components/ui/checkbox"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"
import { useEvaluacionPostulante } from "@/features/evaluacion/queries"
import { useDocumentosPostulacion } from "@/features/formulario/use-documentos"
import { useListarPostulaciones } from "@/features/postulaciones/queries"
import {
  usePresentarReconsideracion,
  useReconsideracionesPostulante,
} from "@/features/reconsideracion/queries"
import { validarYConvertirPdfBase64 } from "@/lib/clientes/evaluacion"
import type {
  EntradaPresentarReconsideracionDto,
  FaseReconsideracion,
  ReconsideracionDto,
} from "@/lib/clientes/reconsideracion"
import { RUTAS } from "@/lib/rutas"

export function ReconsideracionView() {
  const [searchParams] = useSearchParams()
  const faseInicial = (searchParams.get("fase") as FaseReconsideracion) || ""

  const { data: postulacionesData, isLoading: cargandoPostulacion } =
    useListarPostulaciones({ limit: 1 })
  const postulacion = postulacionesData?.items?.[0]
  const submissionId = postulacion?.id ?? ""

  const { data: evaluacionData } = useEvaluacionPostulante(submissionId)
  const { data: reconsideracionesData } =
    useReconsideracionesPostulante(submissionId)
  const { documentos } = useDocumentosPostulacion(submissionId)

  const presentarMutation = usePresentarReconsideracion(submissionId)

  // Estados del formulario
  const [faseSeleccionada, setFaseSeleccionada] = useState<
    FaseReconsideracion | ""
  >(faseInicial)
  const [motivo, setMotivo] = useState("")
  const [documentosSeleccionados, setDocumentosSeleccionados] = useState<
    string[]
  >([])
  const [archivoRespaldo, setArchivoRespaldo] = useState<File | null>(null)
  const [errorValidacion, setErrorValidacion] = useState<string | null>(null)
  const [mostrarFormulario, setMostrarFormulario] = useState(
    Boolean(faseInicial),
  )

  if (cargandoPostulacion) {
    return <EstadoCargando mensaje="Cargando postulación del usuario…" />
  }

  if (!postulacion) {
    return (
      <div className="mx-auto flex w-full max-w-5xl flex-1 flex-col gap-4 p-4 sm:p-6">
        <Alert>
          <AlertCircleIcon className="size-4" />
          <AlertTitle>Sin postulación registrada</AlertTitle>
          <AlertDescription className="text-xs">
            No se encontró una postulación activa vinculada a su cuenta.
          </AlertDescription>
        </Alert>
      </div>
    )
  }

  const resultadosPublicados = evaluacionData?.expediente.resultados ?? {}
  const listaReconsideraciones: ReconsideracionDto[] =
    reconsideracionesData?.pages.flatMap((p) => p.items) ?? []

  // Fases que tienen un resultado publicado oficial
  const fasesConResultado = Object.keys(
    resultadosPublicados,
  ) as FaseReconsideracion[]

  // Fases que ya tienen una reconsideración registrada
  const fasesImpugnadas = new Set(
    listaReconsideraciones.map((r) => r.resultadoOriginal.fase),
  )

  // Fases elegibles para impugnar (tienen resultado publicado y aún no han sido impugnadas)
  const fasesElegibles = fasesConResultado.filter(
    (fase) => !fasesImpugnadas.has(fase),
  )

  const manejarCambioDocumento = (docId: string, checked: boolean) => {
    if (checked) {
      if (documentosSeleccionados.length >= 30) {
        setErrorValidacion("Puede seleccionar un máximo de 30 documentos.")
        return
      }
      setDocumentosSeleccionados([...documentosSeleccionados, docId])
    } else {
      setDocumentosSeleccionados(
        documentosSeleccionados.filter((id) => id !== docId),
      )
    }
  }

  const manejarEnvio = async (e: React.FormEvent) => {
    e.preventDefault()
    setErrorValidacion(null)

    if (!faseSeleccionada) {
      setErrorValidacion("Debe seleccionar la fase que desea impugnar.")
      return
    }

    const resultadoVigente = resultadosPublicados[faseSeleccionada]
    if (!resultadoVigente) {
      setErrorValidacion(
        "No existe una publicación oficial vigente para la fase seleccionada.",
      )
      return
    }

    if (!motivo.trim()) {
      setErrorValidacion(
        "Debe fundamentar detalladamente los motivos de su reconsideración.",
      )
      return
    }

    try {
      let respaldoPayload: { nombre: string; pdfBase64: string } | undefined =
        undefined

      if (archivoRespaldo) {
        respaldoPayload = await validarYConvertirPdfBase64(archivoRespaldo)
      }

      const payload: EntradaPresentarReconsideracionDto = {
        fase: faseSeleccionada,
        resultadoId: resultadoVigente.id,
        motivo: motivo.trim(),
        documentosOriginales:
          documentosSeleccionados.length > 0
            ? documentosSeleccionados
            : undefined,
        respaldo: respaldoPayload,
      }

      await presentarMutation.mutateAsync(payload)
      setMotivo("")
      setFaseSeleccionada("")
      setDocumentosSeleccionados([])
      setArchivoRespaldo(null)
      setMostrarFormulario(false)
    } catch (err) {
      setErrorValidacion(
        err instanceof Error ? err.message : "Error al presentar la solicitud.",
      )
    }
  }

  return (
    <div className="mx-auto flex w-full max-w-5xl flex-1 flex-col gap-6 p-4 sm:p-6">
      {/* Encabezado Principal */}
      <Card>
        <CardHeader>
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div className="flex items-center gap-2">
              <ScaleIcon className="size-5 text-primary" />
              <CardTitle>Solicitudes de Reconsideración</CardTitle>
            </div>
            <Badge variant="outline" className="font-mono text-xs">
              Expediente: {postulacion.code}
            </Badge>
          </div>
          <CardDescription>
            Presentación de impugnaciones motivadas contra resoluciones
            oficiales publicadas dentro de los términos improrrogables del
            concurso.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <Alert className="border-primary/20 bg-primary/5">
            <HelpCircleIcon className="size-4 text-primary" />
            <AlertTitle className="text-xs font-semibold">
              Reglas y términos del derecho a reconsideración
            </AlertTitle>
            <AlertDescription className="text-xs space-y-1 mt-1 text-muted-foreground">
              <p>
                • La impugnación solo puede interponerse contra publicaciones
                oficiales vigentes de cada etapa.
              </p>
              <p>
                • Se admite una única solicitud por cada fase y resultado
                publicado.
              </p>
              <p>
                • Los plazos y horarios límite son calculados por el servidor
                con la hora oficial del Ecuador (America/Guayaquil). No se
                admiten presentaciones extemporáneas.
              </p>
            </AlertDescription>
          </Alert>

          <div className="flex gap-2">
            <Button
              variant={mostrarFormulario ? "secondary" : "default"}
              size="sm"
              disabled={fasesElegibles.length === 0 && !mostrarFormulario}
              onClick={() => setMostrarFormulario(!mostrarFormulario)}
            >
              {mostrarFormulario
                ? "Ocultar Formulario"
                : "Nueva Solicitud de Reconsideración"}
            </Button>
            <Button asChild variant="outline" size="sm">
              <Link to={RUTAS.seguimiento}>Ver seguimiento de fases</Link>
            </Button>
          </div>

          {fasesElegibles.length === 0 && !mostrarFormulario && (
            <p className="text-xs text-muted-foreground">
              {fasesConResultado.length === 0
                ? "Aún no existen resultados oficiales publicados disponibles para impugnar."
                : "Ya ha presentado reconsideraciones para todas las fases actualmente publicadas."}
            </p>
          )}
        </CardContent>
      </Card>

      {/* Formulario de Presentación */}
      {mostrarFormulario && (
        <Card className="border-primary/40">
          <CardHeader>
            <div className="flex items-center gap-2">
              <FileTextIcon className="size-5 text-primary" />
              <CardTitle className="text-base">
                Formulario de Reconsideración Formal
              </CardTitle>
            </div>
            <CardDescription>
              Seleccione el resultado publicado que desea impugnar y exponga sus
              fundamentos técnicos o documentales.
            </CardDescription>
          </CardHeader>
          <form onSubmit={manejarEnvio}>
            <CardContent className="space-y-4">
              {errorValidacion && (
                <Alert variant="destructive">
                  <ShieldAlertIcon className="size-4" />
                  <AlertTitle>Error en la solicitud</AlertTitle>
                  <AlertDescription className="text-xs">
                    {errorValidacion}
                  </AlertDescription>
                </Alert>
              )}

              {/* Selección de Fase / Resultado */}
              <div className="space-y-2">
                <Label htmlFor="fase" className="text-xs font-semibold">
                  Fase del concurso a impugnar
                </Label>
                <Select
                  value={faseSeleccionada}
                  onValueChange={(val: FaseReconsideracion) =>
                    setFaseSeleccionada(val)
                  }
                >
                  <SelectTrigger id="fase">
                    <SelectValue placeholder="Seleccione la fase publicada" />
                  </SelectTrigger>
                  <SelectContent>
                    {fasesElegibles.map((fase) => {
                      const res = resultadosPublicados[fase]
                      return (
                        <SelectItem key={fase} value={fase}>
                          {fase.replace("_", " ").toUpperCase()} — Puntaje:{" "}
                          {res?.puntaje ?? "—"} pts (v{res?.version})
                        </SelectItem>
                      )
                    })}
                  </SelectContent>
                </Select>
              </div>

              {/* Fundamentación del Reclamo */}
              <div className="space-y-2">
                <Label htmlFor="motivo" className="text-xs font-semibold">
                  Fundamentación detallada del reclamo (máx. 8000 caracteres)
                </Label>
                <Textarea
                  id="motivo"
                  rows={4}
                  maxLength={8000}
                  value={motivo}
                  onChange={(e) => setMotivo(e.target.value)}
                  placeholder="Detalle de forma clara y fundamentada los errores u omisiones en la calificación de su expediente..."
                />
                <span className="text-[10px] text-muted-foreground">
                  {motivo.length} / 8000 caracteres
                </span>
              </div>

              {/* Documentos Originales Vinculados del EF-12 */}
              {documentos.length > 0 && (
                <div className="space-y-2 border-t pt-3">
                  <Label className="text-xs font-semibold">
                    Vincular documentos ya aportados en su postulación
                    (opcional)
                  </Label>
                  <p className="text-[11px] text-muted-foreground">
                    Seleccione qué archivos de su expediente respaldan esta
                    impugnación para que el analista los revise.
                  </p>
                  <div className="max-h-40 overflow-y-auto space-y-1.5 rounded-lg border p-2">
                    {documentos.map((doc) => (
                      <div
                        key={doc.id}
                        className="flex items-center space-x-2 text-xs"
                      >
                        <Checkbox
                          id={`doc-${doc.id}`}
                          checked={documentosSeleccionados.includes(doc.id)}
                          onCheckedChange={(checked) =>
                            manejarCambioDocumento(doc.id, Boolean(checked))
                          }
                        />
                        <label
                          htmlFor={`doc-${doc.id}`}
                          className="cursor-pointer truncate font-medium"
                        >
                          Sección {doc.section}: {doc.name} (
                          {(doc.sizeBytes / 1024).toFixed(1)} KiB)
                        </label>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* Documento de Respaldo Adicional en PDF */}
              <div className="space-y-2 border-t pt-3">
                <Label htmlFor="respaldoPdf" className="text-xs font-semibold">
                  Respaldo documental adicional (opcional, PDF máx. 2 MiB)
                </Label>
                <Input
                  id="respaldoPdf"
                  type="file"
                  accept="application/pdf"
                  onChange={(e) => {
                    const f = e.target.files?.[0]
                    setArchivoRespaldo(f ?? null)
                  }}
                />
                {archivoRespaldo && (
                  <p className="text-[11px] text-muted-foreground">
                    Archivo seleccionado: {archivoRespaldo.name} (
                    {(archivoRespaldo.size / 1024).toFixed(1)} KiB)
                  </p>
                )}
              </div>
            </CardContent>
            <CardFooter className="flex justify-between border-t p-4">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => setMostrarFormulario(false)}
              >
                Cancelar
              </Button>
              <Button
                type="submit"
                size="sm"
                disabled={presentarMutation.isPending}
              >
                <SendIcon className="size-4 mr-2" />
                {presentarMutation.isPending
                  ? "Enviando Solicitud..."
                  : "Presentar Reconsideración"}
              </Button>
            </CardFooter>
          </form>
        </Card>
      )}

      {/* Historial de Reconsideraciones Presentadas */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <FileTextIcon className="size-5 text-primary" />
              <CardTitle>Mis Reconsideraciones Presentadas</CardTitle>
            </div>
            <Badge variant="outline">
              {listaReconsideraciones.length === 1
                ? "1 Solicitud"
                : `${listaReconsideraciones.length} Solicitudes`}
            </Badge>
          </div>
          <CardDescription>
            Estado del trámite, informes de Talento Humano y resoluciones
            definitivas de la Comisión de Selección.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {listaReconsideraciones.length === 0 ? (
            <div className="rounded-lg border border-dashed p-6 text-center text-xs text-muted-foreground">
              No ha presentado solicitudes de reconsideración en este concurso.
            </div>
          ) : (
            <div className="space-y-4">
              {listaReconsideraciones.map((rec) => {
                return (
                  <div
                    key={rec.id}
                    className="rounded-xl border bg-card p-4 space-y-4 text-xs"
                  >
                    <div className="flex flex-wrap items-center justify-between gap-2 border-b pb-2">
                      <div className="flex items-center gap-2">
                        <span className="font-semibold text-sm capitalize">
                          Impugnación a:{" "}
                          {rec.resultadoOriginal.fase.replace("_", " ")}
                        </span>
                        <span className="text-muted-foreground">
                          (versión {rec.version})
                        </span>
                      </div>
                      <Badge
                        variant={
                          rec.estado === "resuelta"
                            ? "default"
                            : rec.estado === "atendida"
                              ? "outline"
                              : "secondary"
                        }
                        className="capitalize"
                      >
                        Estado: {rec.estado}
                      </Badge>
                    </div>

                    {/* Resumen del reclamo */}
                    <div className="space-y-2">
                      <div className="rounded bg-muted/40 p-2.5 space-y-1">
                        <p className="font-medium text-foreground">
                          Resultado impugnado:
                        </p>
                        <p className="text-muted-foreground">
                          Puntaje original:{" "}
                          {rec.resultadoOriginal.puntaje ?? "—"} pts | Motivo:{" "}
                          {rec.resultadoOriginal.motivo}
                        </p>
                      </div>

                      <div className="p-1">
                        <span className="font-semibold text-foreground">
                          Motivos expuestos:
                        </span>
                        <p className="mt-1 rounded border bg-background p-2 text-foreground">
                          {rec.motivo}
                        </p>
                      </div>

                      {/* Fechas y Plazos del Servidor */}
                      <div className="grid grid-cols-2 gap-2 rounded border bg-muted/20 p-2 text-[11px] sm:grid-cols-4">
                        <div>
                          <span className="text-muted-foreground">
                            Presentada:
                          </span>
                          <p className="font-medium">
                            {new Date(rec.presentadaEn).toLocaleDateString(
                              "es-EC",
                            )}
                          </p>
                        </div>
                        <div>
                          <span className="text-muted-foreground">
                            Límite Talento Humano:
                          </span>
                          <p className="font-medium">
                            {new Date(
                              rec.limiteTalentoHumano,
                            ).toLocaleDateString("es-EC")}
                          </p>
                        </div>
                        <div>
                          <span className="text-muted-foreground">
                            Límite Comisión:
                          </span>
                          <p className="font-medium">
                            {rec.limiteComision
                              ? new Date(rec.limiteComision).toLocaleDateString(
                                  "es-EC",
                                )
                              : "En atención"}
                          </p>
                        </div>
                        <div>
                          <span className="text-muted-foreground">
                            Cierre de Términos:
                          </span>
                          <p className="font-medium">
                            {new Date(rec.cierreVentana).toLocaleDateString(
                              "es-EC",
                            )}
                          </p>
                        </div>
                      </div>
                    </div>

                    {/* Dictamen de Talento Humano */}
                    {rec.atencion && (
                      <div className="rounded-lg border p-3 space-y-1 bg-muted/20">
                        <div className="flex items-center justify-between">
                          <span className="font-bold">
                            Informe Técnico de Talento Humano
                          </span>
                          <Badge
                            variant={
                              rec.atencion.recomendacion === "rectificar"
                                ? "default"
                                : "secondary"
                            }
                            className="text-[10px] capitalize"
                          >
                            Recomendación: {rec.atencion.recomendacion}
                          </Badge>
                        </div>
                        <p className="text-muted-foreground pt-1">
                          {rec.atencion.observacion}
                        </p>
                      </div>
                    )}

                    {/* Resolución de la Comisión */}
                    {rec.resolucion ? (
                      <div className="rounded-lg border p-3 space-y-2 border-primary/30 bg-primary/5">
                        <div className="flex items-center justify-between">
                          <span className="font-bold text-primary">
                            Resolución Definitiva de la Comisión
                          </span>
                          <Badge
                            variant={
                              rec.resolucion.decision === "rectificar"
                                ? "default"
                                : "secondary"
                            }
                            className="text-[10px] capitalize"
                          >
                            Decisión: {rec.resolucion.decision}
                          </Badge>
                        </div>
                        <p className="text-foreground">
                          {rec.resolucion.motivo}
                        </p>
                        {rec.resolucion.decision === "rectificar" && (
                          <div className="rounded border bg-emerald-50/50 p-2 text-emerald-900 dark:bg-emerald-950/20 dark:text-emerald-300">
                            <p className="font-semibold">
                              Calificación rectificada oficialmente:
                            </p>
                            <p>
                              Nuevo puntaje:{" "}
                              {rec.resolucion.resultadoVigente.puntaje ?? "—"}{" "}
                              pts
                              {rec.resolucion.diferenciaPuntaje !==
                                undefined && (
                                <span>
                                  {" "}
                                  (Diferencia: +
                                  {rec.resolucion.diferenciaPuntaje} pts)
                                </span>
                              )}
                            </p>
                          </div>
                        )}
                      </div>
                    ) : (
                      <p className="text-[11px] text-muted-foreground italic">
                        {rec.estado === "recibida"
                          ? "Su trámite se encuentra en análisis técnico por parte de Talento Humano."
                          : "Talento Humano ha emitido su recomendación; pendiente de resolución por la Comisión de Selección."}
                      </p>
                    )}
                  </div>
                )
              })}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
