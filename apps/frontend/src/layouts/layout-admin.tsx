import { Outlet, useLocation, useNavigate } from "react-router"

import { AppHeader } from "@/components/app-header"
import { AppSidebar, type VistaId } from "@/components/app-sidebar"
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar"
import { RUTAS } from "@/lib/rutas"

/**
 * Ruta del router para cada entrada del sidebar. Mientras "Reportes" no tenga
 * vista propia (FT8) apunta al panel, igual que el botón de inicio.
 */
const RUTA_POR_VISTA: Record<VistaId, string> = {
  dashboard: RUTAS.dashboard,
  formulario: RUTAS.postulacion,
  revision: RUTAS.postulacionRevision,
  imprimir: RUTAS.postulacionImprimir,
  postulaciones: RUTAS.dashboardPostulaciones,
  reportes: RUTAS.dashboard,
}

/**
 * Deriva la entrada activa del sidebar a partir de la URL, sin estado local.
 */
function vistaDesdeRuta(ruta: string): VistaId {
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
  const { pathname } = useLocation()
  const vistaActual = vistaDesdeRuta(pathname)

  return (
    <SidebarProvider>
      <AppSidebar
        vistaActual={vistaActual}
        onNavigate={(vista) => navegar(RUTA_POR_VISTA[vista])}
        rutas={RUTA_POR_VISTA}
      />
      <SidebarInset>
        <AppHeader vistaActual={vistaActual} />
        <Outlet />
      </SidebarInset>
    </SidebarProvider>
  )
}
