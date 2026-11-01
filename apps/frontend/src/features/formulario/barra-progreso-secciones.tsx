import { CheckCircle2Icon, CircleIcon } from "lucide-react"
import { Link } from "react-router"

import { Progress } from "@/components/ui/progress"
import { rutaSeccion } from "@/lib/rutas"
import { cn } from "@/lib/utils"
import { SECCIONES_EF12_INFO } from "./constantes"

export interface BarraProgresoSeccionesProps {
  seccionActiva: number
  seccionesCompletadas: Record<number, boolean>
  onSeleccionarSeccion?: (numero: number) => void
}

export function BarraProgresoSecciones({
  seccionActiva,
  seccionesCompletadas,
  onSeleccionarSeccion,
}: BarraProgresoSeccionesProps) {
  const totalCompletadas =
    Object.values(seccionesCompletadas).filter(Boolean).length
  const porcentaje = Math.round((totalCompletadas / 9) * 100)

  return (
    <div className="flex flex-col gap-3 rounded-lg border bg-card p-4 text-card-foreground shadow-xs">
      <div className="flex items-center justify-between text-xs font-medium sm:text-sm">
        <span className="text-muted-foreground">
          Progreso del Formulario de Postulación
        </span>
        <span className="font-semibold text-primary">
          {totalCompletadas} de 9 completadas ({porcentaje}%)
        </span>
      </div>
      <Progress value={porcentaje} className="h-2" />

      <nav
        aria-label="Progreso por sección"
        className="mt-1 flex items-center justify-between gap-1 overflow-x-auto pb-1"
      >
        {SECCIONES_EF12_INFO.map((sec) => {
          const estaActiva = sec.numero === seccionActiva
          const estaCompleta = Boolean(seccionesCompletadas[sec.numero])

          return (
            <Link
              key={sec.numero}
              to={rutaSeccion(sec.numero)}
              onClick={() => onSeleccionarSeccion?.(sec.numero)}
              className={cn(
                "group flex min-w-16 shrink-0 flex-col items-center gap-1 rounded-md px-2 py-1.5 text-center text-xs transition-colors sm:min-w-20",
                estaActiva
                  ? "bg-primary text-primary-foreground font-semibold shadow-xs"
                  : "text-muted-foreground hover:bg-muted hover:text-foreground",
              )}
            >
              <div className="flex items-center gap-1">
                {estaCompleta ? (
                  <CheckCircle2Icon
                    className={cn(
                      "size-3.5",
                      estaActiva
                        ? "text-primary-foreground"
                        : "text-emerald-600 dark:text-emerald-400",
                    )}
                    aria-hidden="true"
                  />
                ) : (
                  <CircleIcon
                    className={cn(
                      "size-3.5 opacity-60",
                      estaActiva && "opacity-90",
                    )}
                    aria-hidden="true"
                  />
                )}
                <span>Sec. {sec.numero}</span>
              </div>
              <span className="truncate max-w-20 text-[10px] sm:max-w-24">
                {sec.tituloCorto}
              </span>
            </Link>
          )
        })}
      </nav>
    </div>
  )
}
