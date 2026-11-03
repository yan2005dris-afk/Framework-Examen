import { InboxIcon } from "lucide-react"
import type { ReactNode } from "react"

import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

export interface EstadoVacioProps {
  icono?: ReactNode
  titulo: string
  descripcion?: string
  accion?: {
    texto: string
    onClick: () => void
  }
  children?: ReactNode
  className?: string
}

/**
 * Estado vacío reutilizable para listados, tablas o paneles sin registros disponibles.
 */
export function EstadoVacio({
  icono,
  titulo,
  descripcion,
  accion,
  children,
  className,
}: EstadoVacioProps) {
  return (
    <div
      className={cn(
        "flex min-h-48 flex-col items-center justify-center rounded-lg border border-dashed p-8 text-center",
        className,
      )}
    >
      <div className="mb-3 flex size-12 items-center justify-center rounded-full bg-muted text-muted-foreground">
        {icono ?? <InboxIcon className="size-6" aria-hidden="true" />}
      </div>
      <h3 className="text-base font-semibold text-foreground">{titulo}</h3>
      {descripcion && (
        <p className="mt-1 max-w-sm text-sm text-muted-foreground">
          {descripcion}
        </p>
      )}
      {accion && (
        <Button
          variant="outline"
          size="sm"
          onClick={accion.onClick}
          className="mt-4"
        >
          {accion.texto}
        </Button>
      )}
      {children && <div className="mt-4">{children}</div>}
    </div>
  )
}
