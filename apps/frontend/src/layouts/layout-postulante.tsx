import { ShieldCheckIcon } from "lucide-react"
import { useState, type ReactNode } from "react"
import { Link, Outlet, useLocation, useNavigate } from "react-router"

import { ThemeToggle } from "@/components/theme-toggle"
import { Button } from "@/components/ui/button"
import { useSesion } from "@/features/auth/sesion-context"
import { RUTAS } from "@/lib/rutas"
import { cn } from "@/lib/utils"

/**
 * Shell del postulante: cabecera con la navegación del formulario, contenido y
 * cierre de sesión. No usa el sidebar del panel del analista.
 */
export function LayoutPostulante() {
  const { postulante, cerrar } = useSesion()
  const navegar = useNavigate()
  const { pathname } = useLocation()
  const [saliendo, setSaliendo] = useState(false)

  const enRevision = pathname.startsWith(RUTAS.postulacionRevision)
  const enImpresion = pathname.startsWith(RUTAS.postulacionImprimir)
  const enFormulario =
    !enRevision &&
    !enImpresion &&
    (pathname === RUTAS.postulacion ||
      pathname.startsWith(`${RUTAS.postulacion}/`))

  async function manejarSalida() {
    setSaliendo(true)
    await cerrar()
    navegar(RUTAS.login, { replace: true })
  }

  return (
    <div className="flex min-h-svh flex-col bg-background">
      <header className="sticky top-0 z-20 border-b bg-background/95 backdrop-blur print:hidden">
        <div className="mx-auto flex h-16 w-full max-w-5xl items-center gap-3 px-4">
          <Link
            to={RUTAS.postulacion}
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

          <nav
            aria-label="Secciones de la postulación"
            className="ml-2 flex items-center gap-1 overflow-x-auto"
          >
            <EnlaceNav ruta={RUTAS.postulacion} activo={enFormulario}>
              Formulario
            </EnlaceNav>
            <EnlaceNav ruta={RUTAS.postulacionRevision} activo={enRevision}>
              Revisión
            </EnlaceNav>
            <EnlaceNav ruta={RUTAS.postulacionImprimir} activo={enImpresion}>
              Imprimir
            </EnlaceNav>
          </nav>

          <div className="ml-auto flex shrink-0 items-center gap-2">
            <span className="hidden max-w-48 truncate text-sm text-muted-foreground md:inline">
              {postulante?.nombre_completo}
            </span>
            <ThemeToggle />
            <Button
              variant="outline"
              size="sm"
              onClick={manejarSalida}
              disabled={saliendo}
            >
              {saliendo ? "Saliendo…" : "Salir"}
            </Button>
          </div>
        </div>
      </header>

      <main className="flex-1">
        <Outlet />
      </main>

      <footer className="border-t print:hidden">
        <p className="mx-auto w-full max-w-5xl px-4 py-4 text-xs text-muted-foreground">
          Formulario de postulación · Los datos personales se tratan conforme al
          requerimiento del concurso.
        </p>
      </footer>
    </div>
  )
}

type EnlaceNavProps = {
  ruta: string
  activo: boolean
  children: ReactNode
}

/**
 * Enlace de la navegación del postulante. Usa `Link` para no recargar la
 * página y marca el estado activo con `aria-current`.
 */
function EnlaceNav({ ruta, activo, children }: EnlaceNavProps) {
  return (
    <Link
      to={ruta}
      aria-current={activo ? "page" : undefined}
      className={cn(
        "rounded-md px-3 py-2 text-sm whitespace-nowrap transition-colors",
        activo
          ? "bg-muted font-medium text-foreground"
          : "text-muted-foreground hover:bg-muted/60 hover:text-foreground",
      )}
    >
      {children}
    </Link>
  )
}
