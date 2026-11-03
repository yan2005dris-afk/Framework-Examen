import { useState } from "react"
import {
  CheckCircle2Icon,
  ClockIcon,
  HelpCircleIcon,
  ScaleIcon,
  SendIcon,
  ShieldCheckIcon,
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
  useAtenderReconsideracion,
  useReconsideracionesAdmin,
  useResolverReconsideracion,
} from "./queries"
import type {
  DecisionResolucion,
  RecomendacionAtencion,
  ReconsideracionDto,
} from "@/lib/clientes/reconsideracion"

export interface PanelReconsideracionesAdminProps {
  submissionId: string
}

function ItemReconsideracionAdmin({
  reconsideracion,
  submissionId,
}: {
  reconsideracion: ReconsideracionDto
  submissionId: string
}) {
  const { postulante } = useSesion()
  const esAdmin = postulante?.rol === "administrador"
  const esAnalista = postulante?.rol === "analista"

  const atenderMutation = useAtenderReconsideracion(
    submissionId,
    reconsideracion.id,
  )
  const resolverMutation = useResolverReconsideracion(
    submissionId,
    reconsideracion.id,
  )

  // Estados locales para la atención de Analista
  const [recomendacion, setRecomendacion] =
    useState<RecomendacionAtencion>("confirmar")
  const [observacionAnalista, setObservacionAnalista] = useState("")

  // Estados locales para la resolución de Administrador
  const [decision, setDecision] = useState<DecisionResolucion>("confirmar")
  const [motivoResolucion, setMotivoResolucion] = useState("")
  const [resultadoRectificadoId, setResultadoRectificadoId] = useState("")

  const manejarAtender = async () => {
    if (!observacionAnalista.trim()) return
    await atenderMutation.mutateAsync({
      versionAnterior: reconsideracion.version,
      observacion: observacionAnalista.trim(),
      recomendacion,
    })
    setObservacionAnalista("")
  }

  const manejarResolver = async () => {
    if (!motivoResolucion.trim()) return
    if (decision === "rectificar" && !resultadoRectificadoId.trim()) return

    await resolverMutation.mutateAsync({
      versionAnterior: reconsideracion.version,
      decision,
      motivo: motivoResolucion.trim(),
      resultadoRectificadoId:
        decision === "rectificar" ? resultadoRectificadoId.trim() : undefined,
    })
    setMotivoResolucion("")
  }

  const badgeEstado = (estado: ReconsideracionDto["estado"]) => {
    switch (estado) {
      case "recibida":
        return (
          <Badge variant="secondary" className="flex items-center gap-1">
            <ClockIcon className="size-3" />
            <span>Recibida (En espera de Analista)</span>
          </Badge>
        )
      case "atendida":
        return (
          <Badge
            variant="outline"
            className="flex items-center gap-1 border-primary text-primary"
          >
            <CheckCircle2Icon className="size-3" />
            <span>Atendida (En espera de Comisión)</span>
          </Badge>
        )
      case "resuelta":
        return (
          <Badge
            variant="default"
            className="flex items-center gap-1 bg-emerald-600"
          >
            <ShieldCheckIcon className="size-3" />
            <span>Resuelta</span>
          </Badge>
        )
    }
  }

  return (
    <div className="rounded-xl border bg-card p-5 space-y-4">
      {/* Cabecera del reclamo */}
      <div className="flex flex-wrap items-center justify-between gap-2 border-b pb-3">
        <div className="flex items-center gap-2">
          <ScaleIcon className="size-5 text-primary" />
          <span className="font-semibold text-sm">
            Impugnación a fase:{" "}
            <strong className="capitalize text-primary">
              {reconsideracion.resultadoOriginal.fase.replace("_", " ")}
            </strong>
          </span>
          <span className="text-xs text-muted-foreground">
            (versión {reconsideracion.version})
          </span>
        </div>
        {badgeEstado(reconsideracion.estado)}
      </div>

      {/* Datos del resultado impugnado y motivos del postulante */}
      <div className="space-y-2 text-xs">
        <div className="rounded bg-muted/40 p-3 space-y-1">
          <p className="font-semibold text-foreground">
            Resultado original impugnado:
          </p>
          <p className="text-muted-foreground">
            Puntaje: {reconsideracion.resultadoOriginal.puntaje ?? "—"} /{" "}
            {reconsideracion.resultadoOriginal.maximo ?? "—"} pts | Habilita:{" "}
            {reconsideracion.resultadoOriginal.habilitaAvance ? "Sí" : "No"} |
            Motivo: {reconsideracion.resultadoOriginal.motivo}
          </p>
        </div>

        <div className="p-1 space-y-1">
          <span className="font-semibold text-foreground">
            Motivación fundamentada del postulante:
          </span>
          <p className="rounded border bg-background p-2 text-foreground">
            {reconsideracion.motivo}
          </p>
        </div>

        {reconsideracion.documentosOriginales &&
          reconsideracion.documentosOriginales.length > 0 && (
            <div className="p-1">
              <span className="font-semibold text-foreground">
                Documentos originales vinculados:{" "}
              </span>
              <span className="font-mono text-muted-foreground">
                {reconsideracion.documentosOriginales.join(", ")}
              </span>
            </div>
          )}

        {/* Plazos oficiales del servidor */}
        <div className="grid grid-cols-2 gap-2 rounded border bg-muted/20 p-2 text-[11px] sm:grid-cols-4">
          <div>
            <span className="text-muted-foreground">Presentada:</span>
            <p className="font-medium">
              {new Date(reconsideracion.presentadaEn).toLocaleDateString(
                "es-EC",
              )}
            </p>
          </div>
          <div>
            <span className="text-muted-foreground">
              Límite Talento Humano:
            </span>
            <p className="font-medium">
              {new Date(reconsideracion.limiteTalentoHumano).toLocaleDateString(
                "es-EC",
              )}
            </p>
          </div>
          <div>
            <span className="text-muted-foreground">Límite Comisión:</span>
            <p className="font-medium">
              {reconsideracion.limiteComision
                ? new Date(reconsideracion.limiteComision).toLocaleDateString(
                    "es-EC",
                  )
                : "Pendiente de atención"}
            </p>
          </div>
          <div>
            <span className="text-muted-foreground">Cierre de Ventana:</span>
            <p className="font-medium">
              {new Date(reconsideracion.cierreVentana).toLocaleDateString(
                "es-EC",
              )}
            </p>
          </div>
        </div>
      </div>

      {/* Actuación 1: Atención de Talento Humano (Analista) */}
      <div className="rounded-lg border p-3 space-y-3">
        <div className="flex items-center justify-between">
          <span className="text-xs font-bold text-foreground">
            1. Atención Técnica de Talento Humano
          </span>
          {reconsideracion.atencion ? (
            <Badge
              variant={
                reconsideracion.atencion.recomendacion === "rectificar"
                  ? "default"
                  : "secondary"
              }
              className="text-[10px] capitalize"
            >
              Recomendación: {reconsideracion.atencion.recomendacion}
            </Badge>
          ) : (
            <Badge variant="outline" className="text-[10px]">
              Pendiente de Analista
            </Badge>
          )}
        </div>

        {reconsideracion.atencion ? (
          <div className="space-y-1 text-xs">
            <p className="text-muted-foreground">
              Atendido el{" "}
              {new Date(reconsideracion.atencion.fecha).toLocaleDateString(
                "es-EC",
              )}{" "}
              por responsable {reconsideracion.atencion.responsableId}.
            </p>
            <div className="rounded bg-muted/40 p-2">
              <span className="font-medium text-foreground">
                Observaciones:
              </span>
              <p className="mt-1">{reconsideracion.atencion.observacion}</p>
            </div>
          </div>
        ) : esAnalista && reconsideracion.estado === "recibida" ? (
          <div className="space-y-3 pt-2">
            <div className="grid gap-3 sm:grid-cols-2">
              <div className="space-y-1">
                <Label htmlFor="recomendacion" className="text-xs">
                  Recomendación técnica
                </Label>
                <Select
                  value={recomendacion}
                  onValueChange={(val: RecomendacionAtencion) =>
                    setRecomendacion(val)
                  }
                >
                  <SelectTrigger id="recomendacion">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="confirmar">
                      Confirmar resultado original
                    </SelectItem>
                    <SelectItem value="rectificar">
                      Rectificar calificación
                    </SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
            <div className="space-y-1">
              <Label htmlFor="observacionAnalista" className="text-xs">
                Informe técnico y sustento de la recomendación
              </Label>
              <Textarea
                id="observacionAnalista"
                rows={2}
                value={observacionAnalista}
                onChange={(e) => setObservacionAnalista(e.target.value)}
                placeholder="Detalle el análisis documental y la recomendación técnica a la Comisión..."
              />
            </div>
            <Button
              size="sm"
              disabled={
                atenderMutation.isPending || !observacionAnalista.trim()
              }
              onClick={manejarAtender}
            >
              <SendIcon className="size-3.5 mr-1.5" />
              {atenderMutation.isPending
                ? "Registrando..."
                : "Registrar Atención"}
            </Button>
          </div>
        ) : (
          <p className="text-xs text-muted-foreground">
            {reconsideracion.estado === "recibida"
              ? "Esta solicitud debe ser atendida por un Analista de Talento Humano."
              : "La atención técnica fue completada."}
          </p>
        )}
      </div>

      {/* Actuación 2: Resolución de la Comisión (Administrador) */}
      <div className="rounded-lg border p-3 space-y-3">
        <div className="flex items-center justify-between">
          <span className="text-xs font-bold text-foreground">
            2. Resolución Oficial de la Comisión de Selección
          </span>
          {reconsideracion.resolucion ? (
            <Badge
              variant={
                reconsideracion.resolucion.decision === "rectificar"
                  ? "default"
                  : "secondary"
              }
              className="text-[10px] capitalize"
            >
              Decisión: {reconsideracion.resolucion.decision}
            </Badge>
          ) : (
            <Badge variant="outline" className="text-[10px]">
              Pendiente de Comisión
            </Badge>
          )}
        </div>

        {reconsideracion.resolucion ? (
          <div className="space-y-2 text-xs">
            <p className="text-muted-foreground">
              Resuelto el{" "}
              {new Date(reconsideracion.resolucion.fecha).toLocaleDateString(
                "es-EC",
              )}{" "}
              por responsable {reconsideracion.resolucion.responsableId}.
            </p>
            <div className="rounded bg-muted/40 p-2 space-y-1">
              <span className="font-medium text-foreground">
                Motivación de la resolución:
              </span>
              <p>{reconsideracion.resolucion.motivo}</p>
            </div>
            {reconsideracion.resolucion.decision === "rectificar" && (
              <div className="rounded border bg-emerald-50/40 p-2 text-emerald-900 dark:bg-emerald-950/20 dark:text-emerald-300">
                <p className="font-semibold">
                  Calificación rectificada oficialmente:
                </p>
                <p>
                  Nuevo puntaje:{" "}
                  {reconsideracion.resolucion.resultadoVigente.puntaje ?? "—"}{" "}
                  pts
                  {reconsideracion.resolucion.diferenciaPuntaje !==
                    undefined && (
                    <span>
                      {" "}
                      (Diferencia: +
                      {reconsideracion.resolucion.diferenciaPuntaje} pts)
                    </span>
                  )}
                </p>
              </div>
            )}
          </div>
        ) : esAdmin && reconsideracion.estado === "atendida" ? (
          <div className="space-y-3 pt-2">
            <div className="grid gap-3 sm:grid-cols-2">
              <div className="space-y-1">
                <Label htmlFor="decision" className="text-xs">
                  Decisión de la Comisión
                </Label>
                <Select
                  value={decision}
                  onValueChange={(val: DecisionResolucion) => setDecision(val)}
                >
                  <SelectTrigger id="decision">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="confirmar">
                      Confirmar calificación original
                    </SelectItem>
                    <SelectItem value="rectificar">
                      Rectificar con nueva publicación
                    </SelectItem>
                  </SelectContent>
                </Select>
              </div>

              {decision === "rectificar" && (
                <div className="space-y-1">
                  <Label htmlFor="resultadoRectificadoId" className="text-xs">
                    UUID del resultado rectificado publicado
                  </Label>
                  <Input
                    id="resultadoRectificadoId"
                    value={resultadoRectificadoId}
                    onChange={(e) => setResultadoRectificadoId(e.target.value)}
                    placeholder="UUID de la nueva versión publicada"
                  />
                  <p className="text-[10px] text-muted-foreground">
                    Para rectificar, debe existir previamente una nueva
                    publicación oficial de esta fase con mayor versión.
                  </p>
                </div>
              )}
            </div>

            <div className="space-y-1">
              <Label htmlFor="motivoResolucion" className="text-xs">
                Motivación del acta o resolución
              </Label>
              <Textarea
                id="motivoResolucion"
                rows={2}
                value={motivoResolucion}
                onChange={(e) => setMotivoResolucion(e.target.value)}
                placeholder="Fundamentación formal del dictamen de la Comisión..."
              />
            </div>

            <Button
              size="sm"
              disabled={
                resolverMutation.isPending ||
                !motivoResolucion.trim() ||
                (decision === "rectificar" && !resultadoRectificadoId.trim())
              }
              onClick={manejarResolver}
            >
              <ShieldCheckIcon className="size-3.5 mr-1.5" />
              {resolverMutation.isPending
                ? "Resolviendo..."
                : "Emitir Resolución"}
            </Button>
          </div>
        ) : (
          <p className="text-xs text-muted-foreground">
            {reconsideracion.estado === "recibida"
              ? "En espera de que Talento Humano emita su informe técnico antes de resolver."
              : "Esta solicitud debe ser resuelta por un Administrador de la Comisión."}
          </p>
        )}
      </div>
    </div>
  )
}

export function PanelReconsideracionesAdmin({
  submissionId,
}: PanelReconsideracionesAdminProps) {
  const {
    data: reconsideracionesData,
    isLoading,
    error,
    refetch,
  } = useReconsideracionesAdmin(submissionId)

  if (isLoading) {
    return (
      <EstadoCargando mensaje="Cargando reconsideraciones del expediente…" />
    )
  }

  if (error) {
    return (
      <EstadoError
        error={error}
        titulo="No se pudieron cargar las reconsideraciones"
        onReintentar={() => void refetch()}
      />
    )
  }

  const items = reconsideracionesData?.pages.flatMap((page) => page.items) ?? []

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <ScaleIcon className="size-5 text-primary" />
            <CardTitle>Bandeja de Reconsideraciones</CardTitle>
          </div>
          <Badge variant="outline">
            {items.length === 1 ? "1 Solicitud" : `${items.length} Solicitudes`}
          </Badge>
        </div>
        <CardDescription>
          Atención técnica de impugnaciones por Talento Humano y resolución
          definitiva de la Comisión de Selección.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {items.length === 0 ? (
          <Alert>
            <HelpCircleIcon className="size-4" />
            <AlertTitle>Sin reconsideraciones presentadas</AlertTitle>
            <AlertDescription className="text-xs">
              El postulante no ha presentado solicitudes de reconsideración
              contra sus resultados publicados en este expediente.
            </AlertDescription>
          </Alert>
        ) : (
          <div className="space-y-4">
            {items.map((rec) => (
              <ItemReconsideracionAdmin
                key={rec.id}
                reconsideracion={rec}
                submissionId={submissionId}
              />
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
