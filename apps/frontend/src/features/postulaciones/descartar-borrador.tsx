import { Trash2Icon } from "lucide-react"
import { useState } from "react"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"

export interface DescartarBorradorProps {
  /** Título de la convocatoria del borrador, para que el postulante confirme cuál descarta. */
  convocatoria: string
  pendiente: boolean
  onConfirmar: () => Promise<void>
}

/**
 * Botón con confirmación para descartar un borrador. Sólo se ofrece mientras
 * la postulación está en borrador: lo enviado es inmutable y el servidor
 * rechaza igualmente la baja de expedientes enviados o cerrados.
 */
export function DescartarBorrador({
  convocatoria,
  pendiente,
  onConfirmar,
}: DescartarBorradorProps) {
  const [abierto, setAbierto] = useState(false)

  async function confirmar() {
    try {
      await onConfirmar()
      setAbierto(false)
    } catch {
      // El rechazo del servidor ya se informa con un toast; el diálogo queda abierto.
    }
  }

  return (
    <Dialog open={abierto} onOpenChange={setAbierto}>
      <DialogTrigger asChild>
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="text-destructive"
        >
          <Trash2Icon data-icon="inline-start" />
          Descartar borrador
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>¿Descartar este borrador?</DialogTitle>
          <DialogDescription>
            Se eliminará el borrador de «{convocatoria}» con todos sus datos y
            documentos adjuntos. Esta acción no se puede deshacer.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <DialogClose asChild>
            <Button type="button" variant="outline" disabled={pendiente}>
              Cancelar
            </Button>
          </DialogClose>
          <Button
            type="button"
            variant="destructive"
            disabled={pendiente}
            onClick={() => void confirmar()}
          >
            {pendiente ? "Descartando…" : "Descartar borrador"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
