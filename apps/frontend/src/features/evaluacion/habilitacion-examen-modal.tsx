import { useState } from "react"
import { CheckCircle2Icon, ShieldAlertIcon, UserCheckIcon } from "lucide-react"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Label } from "@/components/ui/label"
import { useHabilitarPostulanteExamen } from "@/features/preguntas/queries"
import type { ComponenteFaseResultadoDto } from "@/lib/clientes/resultados"

export interface HabilitacionExamenModalProps {
  submissionId: string
  meritosPublicados?: ComponenteFaseResultadoDto | null
  abierto: boolean
  onOpenChange: (abierto: boolean) => void
  onExito?: () => void
}

export function HabilitacionExamenModal({
  submissionId,
  meritosPublicados,
  abierto,
  onOpenChange,
  onExito,
}: HabilitacionExamenModalProps) {
  const [configuracionId, setConfiguracionId] = useState("")
  const mutacionHabilitar = useHabilitarPostulanteExamen(submissionId)

  const meritosAvanza = meritosPublicados?.habilitaAvance === true

  const handleHabilitar = () => {
    if (!meritosPublicados?.id || !configuracionId.trim()) return

    mutacionHabilitar.mutate(
      {
        configuracionId: configuracionId.trim(),
        resultadoMeritosId: meritosPublicados.id,
      },
      {
        onSuccess: () => {
          onOpenChange(false)
          onExito?.()
        },
      },
    )
  }

  return (
    <Dialog open={abierto} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <UserCheckIcon className="size-5 text-primary" />
            Habilitar para Evaluación Técnica (FT25)
          </DialogTitle>
          <DialogDescription>
            Habilite individualmente al postulante para rendir el examen
            teórico, verificando el cumplimiento previo de la fase de méritos.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-2">
          {!meritosAvanza ? (
            <Alert variant="destructive">
              <ShieldAlertIcon className="size-4" />
              <AlertTitle>Requisito no cumplido</AlertTitle>
              <AlertDescription className="text-xs">
                El postulante no cuenta con una resolución de méritos publicada
                que habilite el avance a la siguiente fase. El servidor
                rechazará la habilitación.
              </AlertDescription>
            </Alert>
          ) : (
            <Alert className="border-emerald-500/20 bg-emerald-500/10 text-emerald-900 dark:text-emerald-200">
              <CheckCircle2Icon className="size-4 text-emerald-600 dark:text-emerald-400" />
              <AlertTitle>Méritos aprobados</AlertTitle>
              <AlertDescription className="text-xs">
                Resolución de méritos ID: {meritosPublicados.id} (Puntaje:{" "}
                {meritosPublicados.puntaje ?? 0} pts). Acredita avance a
                oposición técnica.
              </AlertDescription>
            </Alert>
          )}

          <div className="space-y-1.5">
            <Label htmlFor="config-id">
              ID de Configuración de Examen Aprobada *
            </Label>
            <input
              id="config-id"
              className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 font-mono text-xs shadow-sm transition-colors focus-visible:ring-1 focus-visible:ring-ring focus-visible:outline-none"
              placeholder="UUID de configuración aprobada del banco"
              value={configuracionId}
              onChange={(e) => setConfiguracionId(e.target.value)}
            />
            <p className="text-[11px] text-muted-foreground">
              Obtenga el ID desde la administración del banco de preguntas
              (EF-14).
            </p>
          </div>
        </div>

        <DialogFooter className="gap-2 sm:gap-0">
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancelar
          </Button>
          <Button
            onClick={handleHabilitar}
            disabled={
              !meritosAvanza ||
              !configuracionId.trim() ||
              mutacionHabilitar.isPending
            }
          >
            {mutacionHabilitar.isPending
              ? "Habilitando…"
              : "Habilitar postulante"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
