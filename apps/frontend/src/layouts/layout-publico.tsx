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
      <header className="border-b">
        <div className="mx-auto flex h-16 w-full max-w-5xl items-center gap-3 px-4">
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
              Concurso de Mérito
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

      <main className="flex flex-1 flex-col">
        <Outlet />
      </main>

      <footer className="border-t">
        <p className="mx-auto w-full max-w-5xl px-4 py-4 text-xs text-muted-foreground">
          Portal de postulación al concurso de mérito del BIESS · 2026
        </p>
      </footer>
    </div>
  )
}
