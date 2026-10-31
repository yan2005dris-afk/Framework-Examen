import { createBrowserRouter } from "react-router"

import { GuardSesion } from "@/components/guard-sesion"
import { LayoutAdmin } from "@/layouts/layout-admin"
import { LayoutPostulante } from "@/layouts/layout-postulante"
import { LayoutPublico } from "@/layouts/layout-publico"
import { DashboardPostulacionDetalleView } from "@/pages/dashboard-postulacion-detalle-view"
import { DashboardPostulacionesView } from "@/pages/dashboard-postulaciones-view"
import { DashboardView } from "@/pages/dashboard-view"
import { ImprimirView } from "@/pages/imprimir-view"
import { LoginView } from "@/pages/login-view"
import { NoEncontradoView } from "@/pages/no-encontrado-view"
import { PortadaView } from "@/pages/portada-view"
import { PostulacionView } from "@/pages/postulacion-view"
import { RecuperarContrasenaView } from "@/pages/recuperar-contrasena-view"
import { RegistroView } from "@/pages/registro-view"
import { RevisionPostulacionView } from "@/pages/revision-postulacion-view"
import { ACCESO_POR_RUTA, RUTAS } from "@/lib/rutas"

/**
 * Mapa de rutas del portal.
 *
 * El acceso se declara una sola vez en {@link ACCESO_POR_RUTA} y el guard se
 * monta sobre el layout del grupo: así las rutas hijas (`/postulacion/:seccion`
 * y `/dashboard/postulaciones/:id`) heredan la misma regla y no hay que
 * repetirla en cada hoja.
 */
export const router = createBrowserRouter([
  {
    element: <LayoutPublico />,
    children: [
      { path: RUTAS.inicio, element: <PortadaView /> },
      { path: RUTAS.registro, element: <RegistroView /> },
      { path: RUTAS.login, element: <LoginView /> },
      { path: RUTAS.recuperarContrasena, element: <RecuperarContrasenaView /> },
      { path: "*", element: <NoEncontradoView /> },
    ],
  },
  {
    path: RUTAS.postulacion,
    element: (
      <GuardSesion acceso={ACCESO_POR_RUTA[RUTAS.postulacion]}>
        <LayoutPostulante />
      </GuardSesion>
    ),
    children: [
      { index: true, element: <PostulacionView /> },
      { path: "revision", element: <RevisionPostulacionView /> },
      { path: "imprimir", element: <ImprimirView /> },
      { path: ":seccion", element: <PostulacionView /> },
    ],
  },
  {
    path: RUTAS.dashboard,
    element: (
      <GuardSesion acceso={ACCESO_POR_RUTA[RUTAS.dashboard]}>
        <LayoutAdmin />
      </GuardSesion>
    ),
    children: [
      { index: true, element: <DashboardView /> },
      { path: "postulaciones", element: <DashboardPostulacionesView /> },
      {
        path: "postulaciones/:id",
        element: <DashboardPostulacionDetalleView />,
      },
    ],
  },
])
