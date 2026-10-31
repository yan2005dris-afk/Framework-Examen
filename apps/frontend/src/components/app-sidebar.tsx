import {
  BarChart3Icon,
  ClipboardCheckIcon,
  FileTextIcon,
  LayoutDashboardIcon,
  LifeBuoyIcon,
  PrinterIcon,
  ShieldCheckIcon,
  UsersIcon,
} from "lucide-react"
import { Link } from "react-router"

import { Badge } from "@/components/ui/badge"
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuBadge,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarRail,
} from "@/components/ui/sidebar"

export type VistaId =
  | "dashboard"
  | "formulario"
  | "revision"
  | "imprimir"
  | "postulaciones"
  | "reportes"

type EntradaMenu = {
  id: VistaId
  titulo: string
  icono: typeof LayoutDashboardIcon
  badge?: string
}

const grupos: { titulo: string; entradas: EntradaMenu[] }[] = [
  {
    titulo: "Postulación",
    entradas: [
      {
        id: "dashboard",
        titulo: "Inicio",
        icono: LayoutDashboardIcon,
      },
      {
        id: "formulario",
        titulo: "Formulario EF-12",
        icono: FileTextIcon,
        badge: "9",
      },
      {
        id: "revision",
        titulo: "Revisión y envío",
        icono: ClipboardCheckIcon,
      },
      {
        id: "imprimir",
        titulo: "Imprimir",
        icono: PrinterIcon,
      },
    ],
  },
  {
    titulo: "Administración",
    entradas: [
      {
        id: "postulaciones",
        titulo: "Postulaciones",
        icono: UsersIcon,
        badge: "128",
      },
      {
        id: "reportes",
        titulo: "Reportes",
        icono: BarChart3Icon,
      },
    ],
  },
]

type AppSidebarProps = {
  vistaActual: VistaId
  onNavigate: (vista: VistaId) => void
  /**
   * Ruta del router para cada vista. Cuando una entrada la tiene, se renderiza
   * como enlace (`Link`) para no recargar la página; si falta, se conserva el
   * `onNavigate` original.
   */
  rutas?: Partial<Record<VistaId, string>>
}

export function AppSidebar({
  vistaActual,
  onNavigate,
  rutas,
}: AppSidebarProps) {
  return (
    <Sidebar collapsible="icon">
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton size="lg">
              <div className="flex aspect-square size-9 items-center justify-center rounded-lg bg-sidebar-primary text-sidebar-primary-foreground">
                <ShieldCheckIcon className="size-5" />
              </div>
              <div className="grid flex-1 text-left leading-tight">
                <span className="truncate font-semibold">BIESS</span>
                <span className="truncate text-xs text-muted-foreground">
                  Concurso de Mérito
                </span>
              </div>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>

      <SidebarContent>
        {grupos.map((grupo) => (
          <SidebarGroup key={grupo.titulo}>
            <SidebarGroupLabel>{grupo.titulo}</SidebarGroupLabel>
            <SidebarGroupContent>
              <SidebarMenu>
                {grupo.entradas.map((entrada) => {
                  const ruta = rutas?.[entrada.id]
                  const contenido = (
                    <>
                      <entrada.icono />
                      <span>{entrada.titulo}</span>
                    </>
                  )

                  return (
                    <SidebarMenuItem key={entrada.id}>
                      <SidebarMenuButton
                        asChild
                        isActive={vistaActual === entrada.id}
                        tooltip={entrada.titulo}
                      >
                        {ruta ? (
                          <Link to={ruta}>{contenido}</Link>
                        ) : (
                          <button
                            type="button"
                            onClick={() => onNavigate(entrada.id)}
                          >
                            {contenido}
                          </button>
                        )}
                      </SidebarMenuButton>
                      {entrada.badge && (
                        <SidebarMenuBadge>{entrada.badge}</SidebarMenuBadge>
                      )}
                    </SidebarMenuItem>
                  )
                })}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        ))}
      </SidebarContent>

      <SidebarFooter>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton tooltip="Centro de ayuda">
              <LifeBuoyIcon />
              <span>Centro de ayuda</span>
              <Badge variant="outline" className="ml-auto">
                EF-12
              </Badge>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarFooter>

      <SidebarRail />
    </Sidebar>
  )
}
