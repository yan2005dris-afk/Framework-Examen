import { Link, Outlet, useLocation, useNavigate } from "react-router"

import { AppHeader } from "@/components/app-header"
import { AppSidebar, type VistaId } from "@/components/app-sidebar"
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar"
import { RUTAS } from "@/lib/rutas"
import { useSesion } from "@/features/auth/sesion-context"
import {
  busquedaRevision,
  leerFiltrosRevision,
} from "@/features/revision/modelo"

/**
 * Ruta del router para cada entrada del sidebar administrativo.
 */
const RUTA_POR_VISTA: Record<VistaId, string> = {
  dashboard: RUTAS.dashboard,
  formulario: RUTAS.postulacion,
  revision: RUTAS.postulacionRevision,
  imprimir: RUTAS.postulacionImprimir,
  postulaciones: RUTAS.dashboardPostulaciones,
  reportes: RUTAS.dashboardReportes,
  convocatorias: RUTAS.adminConvocatorias,
  fases: RUTAS.adminFases,
}

/**
 * Deriva la entrada activa del sidebar a partir de la URL, sin estado local.
 */
function vistaDesdeRuta(ruta: string): VistaId {
  if (ruta.startsWith(RUTAS.adminConvocatorias)) return "convocatorias"
  if (ruta.startsWith(RUTAS.adminFases)) return "fases"
  if (ruta.startsWith(RUTAS.dashboardReportes)) return "reportes"
  if (ruta.startsWith(RUTAS.dashboardPostulaciones)) {
    return "postulaciones"
  }
  if (ruta.startsWith(RUTAS.postulacionRevision)) {
    return "revision"
  }
  if (ruta.startsWith(RUTAS.postulacionImprimir)) {
    return "imprimir"
  }
  if (ruta.startsWith(RUTAS.postulacion)) {
    return "formulario"
  }
  return "dashboard"
}

/**
 * Shell del panel de revisión: sidebar + header + contenido. La navegación del
 * sidebar va por URL (`Link`/`navigate`), así que no recarga la página.
 */
export function LayoutAdmin() {
  const navegar = useNavigate()
  const { pathname, search } = useLocation()
  const { postulante, cerrar } = useSesion()
  const { filtros } = leerFiltrosRevision(new URLSearchParams(search))
  const rutas = Object.fromEntries(
    Object.entries(RUTA_POR_VISTA).map(([vista, ruta]) => [
      vista,
      vista === "convocatorias" || vista === "fases"
        ? ruta
        : `${ruta}${busquedaRevision(filtros)}`,
    ]),
  )
  const vistaActual = vistaDesdeRuta(pathname)

  return (
    <SidebarProvider>
      <AppSidebar
        administrativo
        rol={postulante?.rol}
        vistaActual={vistaActual}
        onNavigate={(vista) => navegar(rutas[vista])}
        rutas={rutas}
      />
      <SidebarInset>
        <AppHeader
          vistaActual={vistaActual}
          rutaInicio={`${RUTAS.dashboard}${busquedaRevision(filtros)}`}
          identidadAdministrativa={{
            iniciales: `${postulante?.nombres?.[0] ?? ""}${postulante?.apellidos?.[0] ?? ""}`,
            rol:
              postulante?.rol === "administrador"
                ? "Administrador"
                : "Analista",
          }}
        />
        <div className="flex flex-wrap justify-end gap-4 border-b px-4 py-2 text-xs text-muted-foreground">
          {postulante?.rol === "administrador" && (
            <Link className="underline" to={RUTAS.adminConvocatorias}>
              Gestión administrativa
            </Link>
          )}
          <button
            type="button"
            className="underline"
            onClick={() => void cerrar()}
          >
            Cerrar sesión
          </button>
        </div>
        <Outlet />
      </SidebarInset>
    </SidebarProvider>
  )
}
