import { useState } from "react"
import { Link, Outlet, useLocation } from "react-router"
import { CabeceraPortal, PiePortal } from "@/components/cabecera-portal"
import { RUTAS } from "@/lib/rutas"
import { useSesion } from "@/features/auth/sesion-context"
import {
  busquedaRevision,
  leerFiltrosRevision,
} from "@/features/revision/modelo"
import { cn } from "@/lib/utils"

/** Administración con navegación superior, conservando filtros y permisos. */
export function LayoutAdmin() {
  const { pathname, search } = useLocation()
  const { postulante, cerrar } = useSesion()
  const [saliendo, setSaliendo] = useState(false)
  const { filtros } = leerFiltrosRevision(new URLSearchParams(search))
  const busqueda = busquedaRevision(filtros)
  const administrador = postulante?.rol === "administrador"
  const gestion = [
    { titulo: "Convocatorias", ruta: RUTAS.adminConvocatorias },
    { titulo: "Fases y cronograma", ruta: RUTAS.adminFases },
    { titulo: "Banco de preguntas", ruta: RUTAS.adminPreguntas },
  ]
  return (
    <div className="portal-shell flex min-h-svh flex-col bg-background">
      <CabeceraPortal
        destino={`${RUTAS.dashboard}${busqueda}`}
        descripcion="Módulo de administración · Talento Humano"
        cuenta={{
          nombre: postulante?.nombre_completo ?? "Mi cuenta",
          detalle: administrador ? "Administrador" : "Analista",
        }}
        saliendo={saliendo}
        onSalir={() => {
          setSaliendo(true)
          void cerrar().finally(() => setSaliendo(false))
        }}
        enlaces={[
          {
            titulo: "Dashboard",
            ruta: `${RUTAS.dashboard}${busqueda}`,
            activo: pathname === RUTAS.dashboard,
          },
          {
            titulo: "Postulaciones",
            ruta: `${RUTAS.dashboardPostulaciones}${busqueda}`,
            activo: pathname.startsWith(RUTAS.dashboardPostulaciones),
          },
          {
            titulo: "Matriz",
            ruta: `${RUTAS.dashboardReportes}${busqueda}`,
            activo: pathname.startsWith(RUTAS.dashboardReportes),
          },
        ]}
      />
      {administrador && (
        <nav
          aria-label="Gestión administrativa"
          className="border-b bg-muted/30 print:hidden"
        >
          <div className="mx-auto flex max-w-6xl gap-2 overflow-x-auto px-5 py-2">
            {gestion.map((enlace) => (
              <Link
                key={enlace.ruta}
                to={enlace.ruta}
                aria-current={
                  pathname.startsWith(enlace.ruta) ? "page" : undefined
                }
                className={cn(
                  "shrink-0 rounded-md px-3 py-2 text-xs font-medium hover:bg-muted",
                  pathname.startsWith(enlace.ruta)
                    ? "bg-brand-100 text-primary"
                    : "text-muted-foreground",
                )}
              >
                {enlace.titulo}
              </Link>
            ))}
          </div>
        </nav>
      )}
      <div id="contenido-portal" className="min-w-0 flex-1">
        <Outlet />
      </div>
      <PiePortal />
    </div>
  )
}
