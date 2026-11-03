import { Link } from "react-router"
import type { ReactNode } from "react"
import { ThemeToggle } from "@/components/theme-toggle"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"
import { RUTAS } from "@/lib/rutas"

export interface EnlacePortal {
  titulo: string
  ruta: string
  activo?: boolean
}

interface CabeceraPortalProps {
  destino: string
  enlaces: EnlacePortal[]
  descripcion?: string
  cuenta?: { nombre: string; detalle: string }
  onSalir?: () => void
  saliendo?: boolean
}

/** Cabecera del set de mockups, sin datos ilustrativos ni marcas externas. */
export function CabeceraPortal({
  destino,
  enlaces,
  descripcion = "Concurso Público de Méritos y Oposición",
  cuenta,
  onSalir,
  saliendo,
}: CabeceraPortalProps) {
  const iniciales = cuenta?.nombre
    .trim()
    .split(/\s+/)
    .slice(0, 2)
    .map((n) => n[0])
    .join("")
  return (
    <>
      <a
        href="#contenido-portal"
        className="sr-only fixed top-2 left-4 z-50 rounded-md bg-primary px-4 py-2 text-primary-foreground focus:not-sr-only print:hidden"
      >
        Saltar al contenido
      </a>
      <div className="bg-brand-900 text-background dark:bg-card dark:text-foreground print:hidden">
        <div className="mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-2 px-5 py-2 text-[11px] tracking-wide">
          <span>
            REPÚBLICA DEL ECUADOR <span className="px-1 opacity-50">|</span>{" "}
            Comisión Especial de Selección y Designación
          </span>
          <span className="font-semibold">BIESS — IESS</span>
        </div>
      </div>
      <header className="sticky top-0 z-20 border-b bg-card print:hidden">
        <div className="mx-auto flex min-h-[4.25rem] max-w-6xl flex-wrap items-center justify-between gap-x-6 gap-y-3 px-5 py-3 lg:py-2">
          <Link to={destino} className="flex shrink-0 items-center gap-3">
            <span
              aria-hidden="true"
              className="grid size-10 place-items-center rounded-md bg-primary text-xs font-extrabold tracking-wide text-primary-foreground"
            >
              BIESS
            </span>
            <span>
              <span className="block text-[17px] leading-tight font-extrabold tracking-tight text-primary">
                Directorio del BIESS
              </span>
              <span className="mt-1 block text-xs leading-tight text-muted-foreground">
                {descripcion}
              </span>
            </span>
          </Link>
          <nav
            aria-label={
              cuenta ? "Navegación de la cuenta" : "Navegación pública"
            }
            className="order-3 flex w-full items-center gap-1 overflow-x-auto pb-1 lg:order-none lg:w-auto lg:flex-1 lg:justify-end lg:pb-0"
          >
            {enlaces.map((enlace) => (
              <Link
                key={enlace.ruta}
                to={enlace.ruta}
                aria-current={enlace.activo ? "page" : undefined}
                className={cn(
                  "shrink-0 rounded-md px-3 py-2 text-[13px] font-semibold transition-colors focus-visible:outline-2 focus-visible:outline-ring",
                  enlace.activo
                    ? "bg-brand-100 text-primary dark:bg-primary/15 dark:text-foreground"
                    : "text-secondary-foreground hover:bg-muted",
                )}
              >
                {enlace.titulo}
              </Link>
            ))}
          </nav>
          <div className="ml-auto flex min-w-0 items-center gap-2 lg:ml-0">
            {cuenta && (
              <>
                <span
                  aria-hidden="true"
                  className="hidden size-9 shrink-0 place-items-center rounded-full bg-brand-100 text-xs font-bold text-primary sm:grid"
                >
                  {iniciales}
                </span>
                <span className="hidden min-w-0 sm:block">
                  <span className="block max-w-40 truncate text-xs font-semibold">
                    {cuenta.nombre}
                  </span>
                  <span className="block text-[11px] text-muted-foreground">
                    {cuenta.detalle}
                  </span>
                </span>
              </>
            )}
            <ThemeToggle />
            {onSalir && (
              <Button
                variant="ghost"
                size="sm"
                onClick={onSalir}
                disabled={saliendo}
              >
                {saliendo ? "Saliendo…" : "Salir"}
              </Button>
            )}
          </div>
        </div>
      </header>
    </>
  )
}

/** Pie común, excluido de la hoja de impresión. */
export function PiePortal() {
  return (
    <footer className="mt-auto border-t print:hidden">
      <div className="mx-auto flex max-w-6xl flex-wrap justify-between gap-2 px-5 py-5 text-xs text-muted-foreground">
        <span>Banco del Instituto Ecuatoriano de Seguridad Social</span>
        <span>Canal seguro · Datos protegidos</span>
      </div>
    </footer>
  )
}

/** Marco para errores que se muestran fuera de los layouts protegidos. */
export function MarcoEstadoPortal({ children }: { children: ReactNode }) {
  return (
    <div className="portal-shell flex min-h-svh flex-col bg-background">
      <CabeceraPortal
        destino={RUTAS.inicio}
        enlaces={[{ titulo: "Convocatoria", ruta: RUTAS.inicio }]}
      />
      <main
        id="contenido-portal"
        className="mx-auto w-full max-w-[46rem] flex-1 px-5 pt-8 pb-16"
      >
        {children}
      </main>
      <PiePortal />
    </div>
  )
}
