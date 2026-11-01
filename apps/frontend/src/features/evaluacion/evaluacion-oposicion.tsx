import { useState } from "react"
import {
  AlertCircleIcon,
  CheckCircle2Icon,
  ClockIcon,
  DownloadIcon,
  FileCheckIcon,
  LayersIcon,
  SendIcon,
  ShieldAlertIcon,
  UploadIcon,
} from "lucide-react"

import { EstadoCargando, EstadoError } from "@/components/estados"
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
import { useSesion } from "@/features/auth/sesion-context"
import {
  descargarEvidenciaAdmin,
  validarYConvertirPdfBase64,
  type EntradaConsolidacionDto,
  type EntradaPsicotecnicaDto,
  type EntradaPublicarPsicotecnicaDto,
} from "@/lib/clientes/evaluacion"
import {
  useConsolidarExpediente,
  useExpedienteGradesAdmin,
  usePublicarPsicotecnica,
  useRegistrarPsicotecnica,
} from "./queries"

export interface EvaluacionOposicionProps {
  submissionId: string
  convocatoriaId?: string
}

export function EvaluacionOposicion({
  submissionId,
}: EvaluacionOposicionProps) {
  const { postulante } = useSesion()
  const esAdmin = postulante?.rol === "administrador"
  const esAnalistaOAdmin =
    postulante?.rol === "analista" || postulante?.rol === "administrador"

  const {
    data: expedienteData,
    isLoading,
    error,
    refetch,
  } = useExpedienteGradesAdmin(submissionId)

  const registrarMutation = useRegistrarPsicotecnica(submissionId)
  const publicarMutation = usePublicarPsicotecnica(submissionId)
  const consolidarMutation = useConsolidarExpediente(submissionId)

  // Estados locales para el registro psicotécnico
  const [evaluadaEn, setEvaluadaEn] = useState(() =>
    new Date().toISOString().slice(0, 16),
  )
  const [puntajePsicotecnico, setPuntajePsicotecnico] = useState("")
  const [conclusion, setConclusion] = useState("")
  const [motivoRegistro, setMotivoRegistro] = useState("")
  const [archivoRespaldo, setArchivoRespaldo] = useState<File | null>(null)
  const [errorArchivo, setErrorArchivo] = useState<string | null>(null)

  // Estados para publicación y consolidación
  const [motivoPublicacion, setMotivoPublicacion] = useState("")
  const [motivoConsolidacion, setMotivoConsolidacion] = useState("")
  const [descargandoEvidencia, setDescargandoEvidencia] = useState(false)

  if (isLoading) {
    return <EstadoCargando mensaje="Cargando expediente de calificaciones…" />
  }

  if (error || !expedienteData) {
    return (
      <EstadoError
        error={error}
        titulo="No se pudo cargar el expediente de evaluación"
        onReintentar={() => void refetch()}
      />
    )
  }

  const { expediente, reglasOposicion, fuentesPublicadas } = expedienteData
  const psicotecnicaRegistrada = expediente.psicotecnicaRegistrada
  const psicotecnicaPublicada = expediente.resultados["oposicion_psicotecnica"]
  const meritosPublicados = fuentesPublicadas["meritos"]
  const teoricaPublicada = fuentesPublicadas["oposicion_teorica"]
  const consolidadoPublicado = expediente.resultados["resultados"]

  const reglasDef = reglasOposicion?.definicion.oposicion
  const tieneReglasAprobadas = Boolean(reglasOposicion?.aprobada && reglasDef)

  const manejarArchivo = (e: React.ChangeEvent<HTMLInputElement>) => {
    setErrorArchivo(null)
    const file = e.target.files?.[0]
    if (!file) {
      setArchivoRespaldo(null)
      return
    }
    setArchivoRespaldo(file)
  }

  const manejarRegistroPsicotecnica = async () => {
    if (!reglasOposicion || !reglasDef) return
    if (!archivoRespaldo) {
      setErrorArchivo("Debe adjuntar el informe psicotécnico en PDF.")
      return
    }
    if (!conclusion) {
      setErrorArchivo("Debe seleccionar una conclusión diagnóstica.")
      return
    }
    if (!motivoRegistro.trim()) {
      setErrorArchivo("Debe ingresar la motivación técnica del registro.")
      return
    }

    try {
      const { nombre, pdfBase64 } =
        await validarYConvertirPdfBase64(archivoRespaldo)

      const payload: EntradaPsicotecnicaDto = {
        versionAnterior: psicotecnicaRegistrada?.version ?? 0,
        reglasId: reglasOposicion.id,
        evaluadaEn: new Date(evaluadaEn).toISOString(),
        puntaje:
          reglasDef.modoPsicotecnico === "puntaje" && puntajePsicotecnico !== ""
            ? Number(puntajePsicotecnico)
            : null,
        conclusion,
        motivo: motivoRegistro.trim(),
        respaldo: {
          nombre,
          pdfBase64,
        },
      }

      await registrarMutation.mutateAsync(payload)
      setArchivoRespaldo(null)
      setMotivoRegistro("")
      setErrorArchivo(null)
    } catch (err) {
      setErrorArchivo(
        err instanceof Error ? err.message : "Error al procesar el archivo.",
      )
    }
  }

  const manejarPublicarPsicotecnica = async () => {
    if (!psicotecnicaRegistrada || !reglasOposicion) return
    if (!motivoPublicacion.trim()) return

    const payload: EntradaPublicarPsicotecnicaDto = {
      versionAnterior: psicotecnicaPublicada?.version ?? 0,
      fuenteId: psicotecnicaRegistrada.id,
      reglasId: reglasOposicion.id,
      sustituyeId: psicotecnicaPublicada?.id,
      motivo: motivoPublicacion.trim(),
    }

    await publicarMutation.mutateAsync(payload)
    setMotivoPublicacion("")
  }

  const manejarConsolidar = async () => {
    if (!reglasOposicion) return
    if (!motivoConsolidacion.trim()) return

    if (!meritosPublicados || !teoricaPublicada || !psicotecnicaPublicada)
      return

    const payload: EntradaConsolidacionDto = {
      versionAnterior: consolidadoPublicado?.version ?? 0,
      reglasId: reglasOposicion.id,
      sustituyeId: consolidadoPublicado?.id,
      motivo: motivoConsolidacion.trim(),
      fuentesEsperadas: {
        meritos: meritosPublicados.id,
        oposicion_teorica: teoricaPublicada.id,
        oposicion_psicotecnica: psicotecnicaPublicada.id,
      },
    }

    await consolidarMutation.mutateAsync(payload)
    setMotivoConsolidacion("")
  }

  const descargarRespaldoPsicotecnico = async (evidenciaId: string) => {
    try {
      setDescargandoEvidencia(true)
      const res = await descargarEvidenciaAdmin(submissionId, evidenciaId)
      window.open(res.url, "_blank", "noopener,noreferrer")
    } catch {
      // Error handled by api
    } finally {
      setDescargandoEvidencia(false)
    }
  }

  const fuentesCompletasParaConsolidar = Boolean(
    meritosPublicados && teoricaPublicada && psicotecnicaPublicada,
  )

  return (
    <div className="space-y-6">
      {/* Encabezado y Resumen de Reglas */}
      <Card>
        <CardHeader>
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div className="flex items-center gap-2">
              <LayersIcon className="size-5 text-primary" />
              <CardTitle>
                Expediente de Oposición y Calificaciones (FT20)
              </CardTitle>
            </div>
            {reglasOposicion ? (
              <Badge variant={tieneReglasAprobadas ? "default" : "secondary"}>
                {tieneReglasAprobadas
                  ? `Reglas v${reglasOposicion.version} Aprobadas`
                  : "Reglas Pendientes"}
              </Badge>
            ) : (
              <Badge variant="outline">Sin Reglas Configuradas</Badge>
            )}
          </div>
          <CardDescription>
            Registro de pruebas psicotécnicas, publicación oficial y
            consolidación definitiva de calificaciones.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {reglasDef && (
            <div className="grid grid-cols-2 gap-4 rounded-lg border bg-muted/40 p-4 text-xs sm:grid-cols-4">
              <div>
                <span className="text-muted-foreground">Instrumento:</span>
                <p className="font-semibold">
                  {reglasDef.instrumento} (v{reglasDef.versionInstrumento})
                </p>
              </div>
              <div>
                <span className="text-muted-foreground">
                  Modo Psicotécnico:
                </span>
                <p className="font-semibold capitalize">
                  {reglasDef.modoPsicotecnico === "puntaje"
                    ? `Puntaje (máx. ${reglasDef.maximoPsicotecnico})`
                    : "Concepto cualitativo"}
                </p>
              </div>
              <div>
                <span className="text-muted-foreground">Ponderación Base:</span>
                <p className="font-semibold">
                  M: {reglasDef.pesoMeritos / 100}% | T:{" "}
                  {reglasDef.pesoTeorica / 100}% | P:{" "}
                  {reglasDef.pesoPsicotecnico / 100}%
                </p>
              </div>
              <div>
                <span className="text-muted-foreground">
                  Escala Total / Umbral:
                </span>
                <p className="font-semibold">
                  {reglasDef.maximoTotal} pts (Mín. {reglasDef.umbral})
                </p>
              </div>
            </div>
          )}

          {/* Resumen de Fuentes Publicadas */}
          <div className="grid gap-3 sm:grid-cols-3">
            {/* Fuente 1: Méritos */}
            <div className="rounded-lg border p-3">
              <div className="flex items-center justify-between">
                <span className="text-xs font-semibold">1. Méritos (T20)</span>
                {meritosPublicados ? (
                  <Badge variant="default" className="text-[10px]">
                    Publicado
                  </Badge>
                ) : (
                  <Badge variant="outline" className="text-[10px]">
                    Pendiente
                  </Badge>
                )}
              </div>
              <p className="mt-2 text-lg font-bold">
                {meritosPublicados?.puntaje ?? "—"}
                <span className="text-xs font-normal text-muted-foreground">
                  {" "}
                  / {meritosPublicados?.maximo ?? "10000"} pts
                </span>
              </p>
            </div>

            {/* Fuente 2: Oposición Teórica */}
            <div className="rounded-lg border p-3">
              <div className="flex items-center justify-between">
                <span className="text-xs font-semibold">
                  2. Oposición Teórica (T27)
                </span>
                {teoricaPublicada ? (
                  <Badge variant="default" className="text-[10px]">
                    Publicado
                  </Badge>
                ) : (
                  <Badge variant="outline" className="text-[10px]">
                    Pendiente T27
                  </Badge>
                )}
              </div>
              <p className="mt-2 text-lg font-bold">
                {teoricaPublicada?.puntaje ?? "—"}
                <span className="text-xs font-normal text-muted-foreground">
                  {" "}
                  / {teoricaPublicada?.maximo ?? "—"} pts
                </span>
              </p>
              {!teoricaPublicada && (
                <p className="text-[10px] text-muted-foreground">
                  Nota calculada en examen de conocimientos.
                </p>
              )}
            </div>

            {/* Fuente 3: Oposición Psicotécnica */}
            <div className="rounded-lg border p-3">
              <div className="flex items-center justify-between">
                <span className="text-xs font-semibold">
                  3. Oposición Psicotécnica
                </span>
                {psicotecnicaPublicada ? (
                  <Badge variant="default" className="text-[10px]">
                    Oficial v{psicotecnicaPublicada.version}
                  </Badge>
                ) : psicotecnicaRegistrada ? (
                  <Badge variant="secondary" className="text-[10px]">
                    Provisional v{psicotecnicaRegistrada.version}
                  </Badge>
                ) : (
                  <Badge variant="outline" className="text-[10px]">
                    Sin Registro
                  </Badge>
                )}
              </div>
              <p className="mt-2 text-lg font-bold">
                {psicotecnicaPublicada?.puntaje ??
                  psicotecnicaRegistrada?.puntaje ??
                  (psicotecnicaRegistrada?.conclusion || "—")}
                <span className="text-xs font-normal text-muted-foreground">
                  {psicotecnicaPublicada?.maximo
                    ? ` / ${psicotecnicaPublicada.maximo} pts`
                    : ""}
                </span>
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Bloque Psicotécnico: Estado y Registro */}
      <Card>
        <CardHeader>
          <div className="flex items-center gap-2">
            <FileCheckIcon className="size-5 text-primary" />
            <CardTitle>Evaluación Psicotécnica del Postulante</CardTitle>
          </div>
          <CardDescription>
            Registro de la aplicación del instrumento psicométrico con evidencia
            digital en PDF.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          {/* Registro Actual */}
          {psicotecnicaRegistrada ? (
            <div className="rounded-lg border p-4 space-y-3">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <div className="flex items-center gap-2">
                  <Badge variant="secondary">
                    Versión registrada: v{psicotecnicaRegistrada.version}
                  </Badge>
                  <span className="text-xs text-muted-foreground">
                    Evaluada:{" "}
                    {new Date(
                      psicotecnicaRegistrada.evaluadaEn,
                    ).toLocaleDateString("es-EC")}
                  </span>
                </div>
                {psicotecnicaPublicada ? (
                  <Badge variant="default" className="flex items-center gap-1">
                    <CheckCircle2Icon className="size-3" />
                    <span>Publicada Oficialmente</span>
                  </Badge>
                ) : (
                  <Badge variant="outline" className="flex items-center gap-1">
                    <ClockIcon className="size-3" />
                    <span>Pendiente de Publicación Oficial</span>
                  </Badge>
                )}
              </div>

              <div className="grid grid-cols-1 gap-2 text-xs sm:grid-cols-3">
                <div>
                  <span className="text-muted-foreground">Conclusión:</span>
                  <p className="font-semibold">
                    {psicotecnicaRegistrada.conclusion}
                  </p>
                </div>
                {psicotecnicaRegistrada.puntaje !== null && (
                  <div>
                    <span className="text-muted-foreground">Puntaje:</span>
                    <p className="font-semibold">
                      {psicotecnicaRegistrada.puntaje} /{" "}
                      {psicotecnicaRegistrada.maximo ?? "—"} pts
                    </p>
                  </div>
                )}
                <div>
                  <span className="text-muted-foreground">Respaldo PDF:</span>
                  <div className="flex items-center gap-2">
                    <span className="truncate font-mono">
                      {psicotecnicaRegistrada.respaldo.nombre}
                    </span>
                    <Button
                      size="sm"
                      variant="ghost"
                      className="h-6 px-2 text-xs"
                      disabled={descargandoEvidencia}
                      onClick={() =>
                        descargarRespaldoPsicotecnico(
                          psicotecnicaRegistrada.respaldo.id,
                        )
                      }
                    >
                      <DownloadIcon className="size-3 mr-1" />
                      Descargar
                    </Button>
                  </div>
                </div>
              </div>

              <div className="rounded bg-muted/50 p-2 text-xs">
                <span className="font-medium text-muted-foreground">
                  Motivación técnica:
                </span>
                <p className="mt-1">{psicotecnicaRegistrada.motivo}</p>
              </div>
            </div>
          ) : (
            <Alert>
              <AlertCircleIcon className="size-4" />
              <AlertTitle>Sin evaluación psicotécnica registrada</AlertTitle>
              <AlertDescription className="text-xs">
                El postulante aún no cuenta con un informe psicométrico
                registrado para esta convocatoria.
              </AlertDescription>
            </Alert>
          )}

          {/* Formulario de Registro / Corrección (Analista o Admin) */}
          {esAnalistaOAdmin && tieneReglasAprobadas && (
            <div className="space-y-4 border-t pt-4">
              <h3 className="text-sm font-semibold">
                {psicotecnicaRegistrada
                  ? "Registrar nueva versión del informe psicotécnico"
                  : "Registrar informe psicotécnico"}
              </h3>

              {errorArchivo && (
                <Alert variant="destructive">
                  <ShieldAlertIcon className="size-4" />
                  <AlertTitle>Error en el registro</AlertTitle>
                  <AlertDescription className="text-xs">
                    {errorArchivo}
                  </AlertDescription>
                </Alert>
              )}

              <div className="grid gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="evaluadaEn" className="text-xs">
                    Fecha y hora de evaluación
                  </Label>
                  <Input
                    id="evaluadaEn"
                    type="datetime-local"
                    value={evaluadaEn}
                    onChange={(e) => setEvaluadaEn(e.target.value)}
                  />
                </div>

                {reglasDef?.modoPsicotecnico === "puntaje" && (
                  <div className="space-y-2">
                    <Label htmlFor="puntaje" className="text-xs">
                      Puntaje obtenido (Máx. {reglasDef.maximoPsicotecnico})
                    </Label>
                    <Input
                      id="puntaje"
                      type="number"
                      min={0}
                      max={reglasDef.maximoPsicotecnico ?? undefined}
                      value={puntajePsicotecnico}
                      onChange={(e) => setPuntajePsicotecnico(e.target.value)}
                      placeholder="Ej. 85"
                    />
                  </div>
                )}

                <div className="space-y-2">
                  <Label htmlFor="conclusion" className="text-xs">
                    Conclusión diagnóstica
                  </Label>
                  <Select
                    value={conclusion}
                    onValueChange={(val) => setConclusion(val)}
                  >
                    <SelectTrigger id="conclusion">
                      <SelectValue placeholder="Seleccione conclusión" />
                    </SelectTrigger>
                    <SelectContent>
                      {reglasDef?.conclusiones.map((c) => (
                        <SelectItem key={c} value={c}>
                          {c}{" "}
                          {reglasDef.conclusionesAdmitidas.includes(c)
                            ? "(Habilita avance)"
                            : "(No habilitante)"}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="archivoRespaldo" className="text-xs">
                    Informe de resultados (PDF, máx. 2 MiB)
                  </Label>
                  <div className="flex items-center gap-2">
                    <Input
                      id="archivoRespaldo"
                      type="file"
                      accept="application/pdf"
                      onChange={manejarArchivo}
                    />
                  </div>
                  {archivoRespaldo && (
                    <p className="text-[11px] text-muted-foreground">
                      Archivo: {archivoRespaldo.name} (
                      {(archivoRespaldo.size / 1024).toFixed(1)} KiB)
                    </p>
                  )}
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="motivoRegistro" className="text-xs">
                  Motivación técnica y observaciones
                </Label>
                <Textarea
                  id="motivoRegistro"
                  rows={2}
                  value={motivoRegistro}
                  onChange={(e) => setMotivoRegistro(e.target.value)}
                  placeholder="Detalle los instrumentos aplicados y criterios diagnósticos..."
                />
              </div>

              <Button
                size="sm"
                disabled={registrarMutation.isPending}
                onClick={manejarRegistroPsicotecnica}
              >
                <UploadIcon className="size-4 mr-2" />
                {registrarMutation.isPending
                  ? "Registrando..."
                  : "Guardar Evaluación Psicotécnica"}
              </Button>
            </div>
          )}

          {/* Formulario de Publicación Oficial (Exclusivo Administrador) */}
          {esAdmin && psicotecnicaRegistrada && !psicotecnicaPublicada && (
            <div className="space-y-4 border-t pt-4">
              <h3 className="text-sm font-semibold text-primary">
                Publicación oficial del resultado psicotécnico
              </h3>
              <p className="text-xs text-muted-foreground">
                Como Administrador de la Comisión, autorice la publicación del
                informe para que sea visible al postulante en su portal de
                seguimiento.
              </p>
              <div className="space-y-2">
                <Label htmlFor="motivoPublicacion" className="text-xs">
                  Motivo de publicación oficial
                </Label>
                <Textarea
                  id="motivoPublicacion"
                  rows={2}
                  value={motivoPublicacion}
                  onChange={(e) => setMotivoPublicacion(e.target.value)}
                  placeholder="Resolución de publicación oficial de la fase psicotécnica..."
                />
              </div>
              <Button
                size="sm"
                variant="default"
                disabled={
                  publicarMutation.isPending || !motivoPublicacion.trim()
                }
                onClick={manejarPublicarPsicotecnica}
              >
                <SendIcon className="size-4 mr-2" />
                {publicarMutation.isPending
                  ? "Publicando..."
                  : "Publicar Nota Oficial"}
              </Button>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Bloque de Consolidación Definitiva */}
      <Card>
        <CardHeader>
          <div className="flex items-center gap-2">
            <LayersIcon className="size-5 text-primary" />
            <CardTitle>Consolidación Definitiva del Concurso</CardTitle>
          </div>
          <CardDescription>
            Ponderación exacta de las tres fuentes (Méritos, Teórica y
            Psicotécnica) según el motor consolidacion-v1.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {consolidadoPublicado ? (
            <div className="rounded-lg border p-4 space-y-4">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Badge variant="default">
                    Consolidado Oficial v{consolidadoPublicado.version}
                  </Badge>
                  <span className="text-xs text-muted-foreground">
                    Publicado:{" "}
                    {new Date(
                      consolidadoPublicado.publicadaEn,
                    ).toLocaleDateString("es-EC")}
                  </span>
                </div>
                <Badge
                  variant={
                    consolidadoPublicado.habilitaAvance
                      ? "default"
                      : "destructive"
                  }
                >
                  {consolidadoPublicado.habilitaAvance
                    ? "Supera Umbral"
                    : "No Supera Umbral"}
                </Badge>
              </div>

              <div className="text-2xl font-black">
                {consolidadoPublicado.puntaje ?? 0}
                <span className="text-sm font-normal text-muted-foreground">
                  {" "}
                  / {consolidadoPublicado.maximo ?? 10000} pts
                </span>
              </div>

              {consolidadoPublicado.desglose && (
                <div className="space-y-2 border-t pt-2">
                  <span className="text-xs font-semibold">
                    Desglose ponderado por componente:
                  </span>
                  <div className="grid gap-2 sm:grid-cols-3">
                    {consolidadoPublicado.desglose.map((d) => (
                      <div
                        key={d.fase}
                        className="rounded border bg-muted/40 p-2 text-xs"
                      >
                        <p className="font-semibold capitalize">
                          {d.fase.replace("_", " ")}
                        </p>
                        <p className="text-muted-foreground">
                          Origen: {d.puntajeOrigen} / {d.maximoOrigen} (
                          {d.pesoPuntosBase / 100}%)
                        </p>
                        <p className="font-bold text-primary">
                          Aporte: {d.aporte} pts
                        </p>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          ) : (
            <div>
              {!fuentesCompletasParaConsolidar ? (
                <Alert>
                  <AlertCircleIcon className="size-4" />
                  <AlertTitle>Fuentes requeridas incompletas</AlertTitle>
                  <AlertDescription className="text-xs space-y-1">
                    <p>
                      Para consolidar el concurso se requiere la publicación
                      oficial de las tres etapas:
                    </p>
                    <ul className="list-disc pl-5">
                      <li>
                        Méritos:{" "}
                        {meritosPublicados ? (
                          <span className="text-emerald-600 font-semibold">
                            Disponible
                          </span>
                        ) : (
                          <span className="text-destructive font-semibold">
                            Pendiente
                          </span>
                        )}
                      </li>
                      <li>
                        Oposición Teórica:{" "}
                        {teoricaPublicada ? (
                          <span className="text-emerald-600 font-semibold">
                            Disponible
                          </span>
                        ) : (
                          <span className="text-destructive font-semibold">
                            Pendiente de examen técnico (T27)
                          </span>
                        )}
                      </li>
                      <li>
                        Oposición Psicotécnica:{" "}
                        {psicotecnicaPublicada ? (
                          <span className="text-emerald-600 font-semibold">
                            Disponible
                          </span>
                        ) : (
                          <span className="text-destructive font-semibold">
                            Pendiente de publicación oficial
                          </span>
                        )}
                      </li>
                    </ul>
                  </AlertDescription>
                </Alert>
              ) : esAdmin ? (
                <div className="space-y-4">
                  <Alert className="border-emerald-500/40 bg-emerald-50/40 dark:bg-emerald-950/20">
                    <CheckCircle2Icon className="size-4 text-emerald-600" />
                    <AlertTitle>
                      Todas las fuentes oficiales disponibles
                    </AlertTitle>
                    <AlertDescription className="text-xs">
                      Se han verificado las publicaciones de Méritos, Examen
                      Teórico y Psicotécnica. Puede proceder con la
                      consolidación.
                    </AlertDescription>
                  </Alert>
                  <div className="space-y-2">
                    <Label htmlFor="motivoConsolidacion" className="text-xs">
                      Motivación del acta de consolidación
                    </Label>
                    <Textarea
                      id="motivoConsolidacion"
                      rows={2}
                      value={motivoConsolidacion}
                      onChange={(e) => setMotivoConsolidacion(e.target.value)}
                      placeholder="Resolución de consolidación definitiva de calificaciones..."
                    />
                  </div>
                  <Button
                    size="sm"
                    disabled={
                      consolidarMutation.isPending ||
                      !motivoConsolidacion.trim()
                    }
                    onClick={manejarConsolidar}
                  >
                    <SendIcon className="size-4 mr-2" />
                    {consolidarMutation.isPending
                      ? "Consolidando..."
                      : "Consolidar Expediente"}
                  </Button>
                </div>
              ) : (
                <p className="text-xs text-muted-foreground">
                  La consolidación definitiva debe ser ejecutada por un
                  Administrador de la Comisión de Selección una vez completadas
                  las tres fuentes.
                </p>
              )}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
