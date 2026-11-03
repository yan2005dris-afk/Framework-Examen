import { useState } from "react"
import { Outlet, useLocation, useNavigate, useSearchParams } from "react-router"
import { CabeceraPortal, PiePortal } from "@/components/cabecera-portal"
import { useSesion } from "@/features/auth/sesion-context"
import { RUTAS, rutaPostulacion } from "@/lib/rutas"

/** Cabecera horizontal del postulante; conserva la convocatoria en cada enlace. */
export function LayoutPostulante() {
  const { postulante, cerrar } = useSesion()
  const navegar = useNavigate()
  const { pathname } = useLocation()
  const [parametros] = useSearchParams()
  const convocatoriaId = parametros.get("convocatoria")
  const [saliendo, setSaliendo] = useState(false)
  async function manejarSalida() {
    setSaliendo(true)
    await cerrar()
    navegar(RUTAS.login, { replace: true })
  }
  return (
    <div className="portal-shell flex min-h-svh flex-col bg-background">
      <CabeceraPortal
        destino={rutaPostulacion(RUTAS.postulacion, convocatoriaId)}
        cuenta={{
          nombre: postulante?.nombre_completo ?? "Mi cuenta",
          detalle: "Postulante",
        }}
        onSalir={() => void manejarSalida()}
        saliendo={saliendo}
        enlaces={[
          {
            titulo: "Formulario",
            ruta: rutaPostulacion(RUTAS.postulacion, convocatoriaId),
            activo:
              pathname === RUTAS.postulacion ||
              /^\/postulacion\/\d+$/.test(pathname),
          },
          {
            titulo: "Revisión",
            ruta: rutaPostulacion(RUTAS.postulacionRevision, convocatoriaId),
            activo: pathname === RUTAS.postulacionRevision,
          },
          {
            titulo: "Imprimir",
            ruta: rutaPostulacion(RUTAS.postulacionImprimir, convocatoriaId),
            activo: pathname === RUTAS.postulacionImprimir,
          },
        ]}
      />
      <main id="contenido-portal" className="min-w-0 flex-1">
        <Outlet />
      </main>
      <PiePortal />
    </div>
  )
}
