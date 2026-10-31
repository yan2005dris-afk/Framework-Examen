import type { ReactNode } from "react"
import { Navigate, useLocation } from "react-router"

import { Skeleton } from "@/components/ui/skeleton"
import { useSesion } from "@/features/auth/sesion-context"
import { construirRutaLogin, puedeAcceder, type TipoAcceso } from "@/lib/rutas"
import { SinPermisoView } from "@/pages/sin-permiso-view"

type GuardSesionProps = {
  /** Acceso que exige la ruta. */
  acceso: TipoAcceso
  children: ReactNode
}

/**
 * Guard de rutas: consulta la sesión y decide entre renderizar la vista,
 * mostrar la carga inicial, redirigir al login o responder `403`.
 *
 * Mientras el estado es `cargando` nunca renderiza el login, así que no hay
 * parpadeo al entrar a una ruta protegida. El destino se conserva en
 * `/login?destino=…` para volver allí después de autenticarse.
 */
export function GuardSesion({ acceso, children }: GuardSesionProps) {
  const { postulante, estado } = useSesion()
  const ubicacion = useLocation()

  if (acceso === "publico") {
    return <>{children}</>
  }

  if (estado === "cargando") {
    return <PantallaCargando />
  }

  if (!puedeAcceder(acceso, postulante)) {
    if (!postulante) {
      const destino = `${ubicacion.pathname}${ubicacion.search}`
      return <Navigate to={construirRutaLogin(destino)} replace />
    }
    return (
      <SinPermisoView
        rolRequerido={acceso === "con-rol-admin" ? "administrador" : "analista"}
      />
    )
  }

  return <>{children}</>
}

/**
 * Estado de carga a pantalla completa, anunciado a lectores de pantalla.
 */
function PantallaCargando() {
  return (
    <div
      className="flex min-h-svh flex-col items-center justify-center gap-4 p-6"
      role="status"
      aria-live="polite"
    >
      <Skeleton className="h-8 w-56" />
      <Skeleton className="h-4 w-72" />
      <Skeleton className="h-4 w-64" />
      <span className="sr-only">Verificando su sesión…</span>
    </div>
  )
}
