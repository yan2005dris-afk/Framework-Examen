import { useState } from "react"
import {
  AlertCircleIcon,
  CheckCircle2Icon,
  ClockIcon,
  MailIcon,
  RefreshCwIcon,
  SendIcon,
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
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { useSesion } from "@/features/auth/sesion-context"
import {
  useNotificacionesAdmin,
  useReintentarNotificacionAdmin,
} from "./queries"
import type { NotificacionEtapaDto } from "@/lib/clientes/notificaciones"

export interface PanelNotificacionesAdminProps {
  submissionId: string
}

export function PanelNotificacionesAdmin({
  submissionId,
}: PanelNotificacionesAdminProps) {
  const { postulante } = useSesion()
  const esAdmin = postulante?.rol === "administrador"

  const {
    data: notificacionesData,
    isLoading,
    error,
    refetch,
  } = useNotificacionesAdmin(submissionId)

  const [notificacionAReintentar, setNotificacionAReintentar] =
    useState<NotificacionEtapaDto | null>(null)
  const [motivoReintento, setMotivoReintento] = useState("")

  const reintentarMutation = useReintentarNotificacionAdmin(
    submissionId,
    notificacionAReintentar?.id ?? "",
  )

  if (isLoading) {
    return <EstadoCargando mensaje="Cargando notificaciones del expediente…" />
  }

  if (error) {
    return (
      <EstadoError
        error={error}
        titulo="No se pudieron cargar las notificaciones"
        onReintentar={() => void refetch()}
      />
    )
  }

  const items = notificacionesData?.pages.flatMap((page) => page.items) ?? []

  const manejarReintento = async () => {
    if (!notificacionAReintentar || !motivoReintento.trim()) return

    await reintentarMutation.mutateAsync({
      reenviosAnteriores: notificacionAReintentar.reenvios,
      motivo: motivoReintento.trim(),
    })

    setNotificacionAReintentar(null)
    setMotivoReintento("")
  }

  const badgeEstado = (estado: NotificacionEtapaDto["estado"]) => {
    switch (estado) {
      case "smtp_accepted":
        return (
          <Badge
            variant="default"
            className="flex items-center gap-1 bg-emerald-600"
          >
            <CheckCircle2Icon className="size-3" />
            <span>Aceptada por SMTP</span>
          </Badge>
        )
      case "pending":
      case "processing":
        return (
          <Badge variant="secondary" className="flex items-center gap-1">
            <ClockIcon className="size-3" />
            <span>En cola de envío</span>
          </Badge>
        )
      case "failed":
        return (
          <Badge variant="destructive" className="flex items-center gap-1">
            <XCircleIcon className="size-3" />
            <span>Envío fallido</span>
          </Badge>
        )
    }
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <MailIcon className="size-5 text-primary" />
            <CardTitle>Comunicaciones y Notificaciones de Etapa</CardTitle>
          </div>
          <Badge variant="outline">
            {items.length === 1 ? "1 Aviso" : `${items.length} Avisos`}
          </Badge>
        </div>
        <CardDescription>
          Registro de entregas seguras de correo electrónico. La aceptación por
          el servidor SMTP no garantiza la apertura por el destinatario.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {items.length === 0 ? (
          <Alert>
            <AlertCircleIcon className="size-4" />
            <AlertTitle>Sin avisos registrados</AlertTitle>
            <AlertDescription className="text-xs">
              No existen eventos de notificación por correo vinculados a este
              expediente.
            </AlertDescription>
          </Alert>
        ) : (
          <div className="space-y-3">
            {items.map((notif) => (
              <div
                key={notif.id}
                className="rounded-lg border bg-card p-4 text-xs space-y-2"
              >
                <div className="flex flex-wrap items-center justify-between gap-2 border-b pb-2">
                  <div className="flex items-center gap-2">
                    <span className="font-semibold text-sm">
                      {notif.accion}
                    </span>
                    <Badge variant="outline" className="font-mono text-[10px]">
                      Plantilla: {notif.plantilla}
                    </Badge>
                  </div>
                  {badgeEstado(notif.estado)}
                </div>

                <div className="grid grid-cols-2 gap-2 text-muted-foreground sm:grid-cols-4 pt-1">
                  <div>
                    <span>Intentos de ciclo:</span>
                    <p className="font-semibold text-foreground">
                      {notif.intentosCiclo} / 5
                    </p>
                  </div>
                  <div>
                    <span>Total de intentos:</span>
                    <p className="font-semibold text-foreground">
                      {notif.intentosTotales}
                    </p>
                  </div>
                  <div>
                    <span>Reenvíos manuales:</span>
                    <p className="font-semibold text-foreground">
                      {notif.reenvios} / 3
                    </p>
                  </div>
                  <div>
                    <span>Última actualización:</span>
                    <p className="font-semibold text-foreground">
                      {new Date(notif.actualizadaEn).toLocaleDateString(
                        "es-EC",
                      )}{" "}
                      {new Date(notif.actualizadaEn).toLocaleTimeString(
                        "es-EC",
                        {
                          hour: "2-digit",
                          minute: "2-digit",
                        },
                      )}
                    </p>
                  </div>
                </div>

                {notif.codigoError && (
                  <div className="rounded bg-destructive/10 p-2 text-destructive font-mono text-[11px]">
                    Código de error: {notif.codigoError}
                  </div>
                )}

                {esAdmin && notif.estado === "failed" && notif.reenvios < 3 && (
                  <div className="pt-2">
                    <Button
                      size="sm"
                      variant="outline"
                      className="text-xs"
                      onClick={() => setNotificacionAReintentar(notif)}
                    >
                      <RefreshCwIcon className="size-3 mr-1" />
                      Reintentar notificación
                    </Button>
                  </div>
                )}
              </div>
            ))}
          </div>
        )}

        {/* Modal de confirmación de reintento */}
        <Dialog
          open={Boolean(notificacionAReintentar)}
          onOpenChange={(open) => !open && setNotificacionAReintentar(null)}
        >
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Reintentar Notificación Oficial</DialogTitle>
              <DialogDescription className="text-xs">
                Se encolará un nuevo intento de entrega para la notificación{" "}
                <strong>{notificacionAReintentar?.accion}</strong> utilizando el
                mismo Message-ID y destinatario institucional registrado.
              </DialogDescription>
            </DialogHeader>

            <div className="space-y-3 py-2 text-xs">
              <div className="space-y-1">
                <Label htmlFor="motivoReintento" className="text-xs">
                  Motivo justificado del reintento (obligatorio)
                </Label>
                <Textarea
                  id="motivoReintento"
                  rows={3}
                  value={motivoReintento}
                  onChange={(e) => setMotivoReintento(e.target.value)}
                  placeholder="Ej. Incidencia de red solventada en el servidor de correo institucional..."
                />
              </div>
            </div>

            <DialogFooter>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setNotificacionAReintentar(null)}
              >
                Cancelar
              </Button>
              <Button
                size="sm"
                disabled={
                  reintentarMutation.isPending || !motivoReintento.trim()
                }
                onClick={manejarReintento}
              >
                <SendIcon className="size-3 mr-1" />
                {reintentarMutation.isPending
                  ? "Programando..."
                  : "Confirmar Reintento"}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </CardContent>
    </Card>
  )
}
