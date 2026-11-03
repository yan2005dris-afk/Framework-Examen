import { Outlet, useLocation } from "react-router"
import { CabeceraPortal, PiePortal } from "@/components/cabecera-portal"
import { RUTAS } from "@/lib/rutas"

/** Marco público del mockup; los flujos OTP permanecen en sus rutas. */
export function LayoutPublico() {
  const { pathname } = useLocation()
  return (
    <div className="portal-shell flex min-h-svh flex-col bg-background">
      <CabeceraPortal
        destino={RUTAS.inicio}
        enlaces={[
          {
            titulo: "Convocatoria",
            ruta: RUTAS.inicio,
            activo: pathname === RUTAS.inicio,
          },
          {
            titulo: "Iniciar sesión",
            ruta: RUTAS.login,
            activo: pathname === RUTAS.login,
          },
          {
            titulo: "Registrarse",
            ruta: RUTAS.registro,
            activo: pathname === RUTAS.registro,
          },
        ]}
      />
      <main id="contenido-portal" className="flex min-w-0 flex-1 flex-col">
        <Outlet />
      </main>
      <PiePortal />
    </div>
  )
}
