import { Loader2Icon } from "lucide-react"

import { Skeleton } from "@/components/ui/skeleton"
import { cn } from "@/lib/utils"

export interface EstadoCargandoProps {
  mensaje?: string
  variante?: "spinner" | "esqueleto" | "pantalla-completa"
  className?: string
}

/**
 * Indicador de carga reutilizable y accesible para vistas asíncronas.
 */
export function EstadoCargando({
  mensaje = "Cargando información…",
  variante = "spinner",
  className,
}: EstadoCargandoProps) {
  if (variante === "pantalla-completa") {
    return (
      <div
        role="status"
        aria-live="polite"
        className={cn(
          "flex min-h-[50vh] flex-col items-center justify-center gap-3 p-6 text-muted-foreground",
          className,
        )}
      >
        <Loader2Icon
          className="size-8 animate-spin text-primary"
          aria-hidden="true"
        />
        <p className="text-sm font-medium">{mensaje}</p>
        <span className="sr-only">{mensaje}</span>
      </div>
    )
  }

  if (variante === "esqueleto") {
    return (
      <div
        role="status"
        aria-live="polite"
        className={cn("flex w-full flex-col gap-3 p-4", className)}
      >
        <Skeleton className="h-7 w-1/3" />
        <Skeleton className="h-4 w-2/3" />
        <div className="mt-2 flex flex-col gap-2">
          <Skeleton className="h-12 w-full" />
          <Skeleton className="h-12 w-full" />
          <Skeleton className="h-12 w-full" />
        </div>
        <span className="sr-only">{mensaje}</span>
      </div>
    )
  }

  return (
    <div
      role="status"
      aria-live="polite"
      className={cn(
        "flex items-center justify-center gap-2 py-8 text-sm text-muted-foreground",
        className,
      )}
    >
      <Loader2Icon
        className="size-5 animate-spin text-primary"
        aria-hidden="true"
      />
      <span>{mensaje}</span>
    </div>
  )
}
