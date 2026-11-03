import { CheckIcon } from "lucide-react"
import { Link } from "react-router"

import { Progress } from "@/components/ui/progress"
import { rutaPostulacion, rutaSeccion } from "@/lib/rutas"
import { cn } from "@/lib/utils"
import { SECCIONES_EF12_INFO } from "./constantes"

export interface BarraProgresoSeccionesProps {
  seccionActiva: number
  seccionesCompletadas: Record<number, boolean>
  onSeleccionarSeccion?: (numero: number) => void
  /** Convocatoria de la postulación; los enlaces la conservan en la URL. */
  convocatoriaId?: string
}

export function BarraProgresoSecciones({
  seccionActiva,
  seccionesCompletadas,
  onSeleccionarSeccion,
  convocatoriaId,
}: BarraProgresoSeccionesProps) {
  const totalCompletadas =
    Object.values(seccionesCompletadas).filter(Boolean).length
  const porcentaje = Math.round((totalCompletadas / 9) * 100)

  return (
    <aside className="min-w-0 rounded-lg border bg-card p-4 text-card-foreground shadow-sm min-[60rem]:sticky min-[60rem]:top-24">
      <p className="px-2 pb-3 text-[11px] font-bold tracking-widest text-muted-foreground uppercase">
        Formulario de postulación
      </p>
      <nav
        aria-label="Secciones del formulario"
        className="relative flex gap-1 overflow-x-auto pb-2 min-[60rem]:flex-col min-[60rem]:pb-0"
      >
        {SECCIONES_EF12_INFO.map((sec) => {
          const estaActiva = sec.numero === seccionActiva
          const estaCompleta = Boolean(seccionesCompletadas[sec.numero])

          return (
            <Link
              key={sec.numero}
              to={rutaPostulacion(rutaSeccion(sec.numero), convocatoriaId)}
              onClick={() => onSeleccionarSeccion?.(sec.numero)}
              aria-current={estaActiva ? "step" : undefined}
              className={cn(
                "flex min-w-44 shrink-0 items-center gap-3 rounded-md p-2 text-[13px] transition-colors min-[60rem]:min-w-0",
                estaActiva
                  ? "bg-brand-100 font-bold text-primary dark:bg-primary/15 dark:text-foreground"
                  : "text-secondary-foreground hover:bg-muted",
              )}
            >
              <span
                aria-hidden="true"
                className={cn(
                  "grid size-6 shrink-0 place-items-center rounded-full border font-mono text-[11px] font-bold",
                  estaActiva
                    ? "border-primary bg-primary text-primary-foreground"
                    : estaCompleta
                      ? "border-success/40 bg-success-soft text-success"
                      : "bg-card text-muted-foreground",
                )}
              >
                {estaCompleta && !estaActiva ? (
                  <CheckIcon className="size-3.5" />
                ) : (
                  sec.numero
                )}
              </span>
              <span>
                {sec.numero === 2
                  ? "Domicilio y contacto"
                  : sec.numero === 4
                    ? "Selección del cargo"
                    : sec.numero === 7
                      ? "Publicaciones"
                      : sec.titulo}
              </span>
              <span className="sr-only">
                {estaCompleta ? "Completada" : "Pendiente"}
              </span>
            </Link>
          )
        })}
      </nav>
      <div className="mt-4 space-y-2 border-t pt-4 text-xs text-muted-foreground">
        <p>Avance: {totalCompletadas} de 9 secciones</p>
        <Progress
          value={porcentaje}
          className="h-1.5"
          aria-label="Avance del formulario"
        />
      </div>
    </aside>
  )
}
