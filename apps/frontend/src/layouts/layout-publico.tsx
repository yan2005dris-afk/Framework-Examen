import { ShieldCheckIcon } from "lucide-react"
import { Link, Outlet } from "react-router"

import { ThemeToggle } from "@/components/theme-toggle"
import { Button } from "@/components/ui/button"
import { RUTAS } from "@/lib/rutas"

/**
 * Shell de las rutas públicas (portada, registro, login y recuperación).
 * No monta el guard: estas vistas se ven sin sesión.
 */
export function LayoutPublico() {
  return (
    <div className="flex min-h-svh flex-col bg-background">
      <a
        href="#contenido-publico"
        className="sr-only focus:not-sr-only focus:p-3"
      >
        Saltar al contenido
      </a>
      <div className="border-b bg-muted/50 px-4 py-2 text-xs text-muted-foreground">
        <div className="mx-auto flex max-w-6xl flex-wrap justify-between gap-2">
          <span>
            REPÚBLICA DEL ECUADOR · Comisión Especial de Selección y Designación
          </span>
          <span>BIESS — IESS</span>
        </div>
      </div>
      <header className="border-b">
        <div className="mx-auto flex min-h-16 w-full max-w-6xl flex-wrap py-3 items-center gap-3 px-4">
          <Link
            to={RUTAS.inicio}
            className="flex shrink-0 items-center gap-2 font-semibold"
          >
            <ShieldCheckIcon
              className="size-5 text-primary"
              aria-hidden="true"
            />
            <span>BIESS</span>
            <span className="hidden text-xs font-normal text-muted-foreground sm:inline">
              Concurso Público de Méritos y Oposición
            </span>
          </Link>

          <div className="ml-auto flex items-center gap-2">
            <ThemeToggle />
            <Button asChild variant="ghost" size="sm">
              <Link to={RUTAS.login}>Iniciar sesión</Link>
            </Button>
            <Button asChild size="sm">
              <Link to={RUTAS.registro}>Registrarse</Link>
            </Button>
          </div>
        </div>
      </header>

      <main id="contenido-publico" className="flex flex-1 flex-col">
        <Outlet />
      </main>

      <footer className="border-t">
        <p className="mx-auto w-full max-w-6xl px-4 py-4 text-xs text-muted-foreground">
          Portal de postulación al concurso de méritos y oposición del BIESS
        </p>
      </footer>
    </div>
  )
}
