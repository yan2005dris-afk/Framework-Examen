import { ChevronRightIcon, SearchIcon } from "lucide-react"
import { Link } from "react-router"

import { ThemeToggle } from "@/components/theme-toggle"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Badge } from "@/components/ui/badge"
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb"
import { Button } from "@/components/ui/button"
import { SidebarTrigger } from "@/components/ui/sidebar"
import type { VistaId } from "@/components/app-sidebar"
import { RUTAS } from "@/lib/rutas"

const titulos: Record<VistaId, string> = {
  dashboard: "Inicio",
  formulario: "Formulario EF-12",
  revision: "Revisión y envío",
  imprimir: "Imprimir",
  postulaciones: "Postulaciones",
  reportes: "Reportes",
}

type AppHeaderProps = {
  identidadAdministrativa?: { iniciales: string; rol: string }
  vistaActual: VistaId
  /** Ruta a la que vuelve la miga "Concurso CPMO 2026". */
  rutaInicio?: string
}

export function AppHeader({
  vistaActual,
  rutaInicio = RUTAS.dashboard,
  identidadAdministrativa,
}: AppHeaderProps) {
  return (
    <header className="flex h-16 shrink-0 items-center gap-2 border-b px-4">
      <SidebarTrigger className="-ml-1" />
      <SeparatorVertical />
      <Breadcrumb>
        <BreadcrumbList>
          <BreadcrumbItem>
            <BreadcrumbLink asChild>
              <Link to={rutaInicio}>
                {identidadAdministrativa
                  ? "Revisión interna"
                  : "Concurso CPMO 2026"}
              </Link>
            </BreadcrumbLink>
          </BreadcrumbItem>
          <BreadcrumbSeparator>
            <ChevronRightIcon />
          </BreadcrumbSeparator>
          <BreadcrumbItem>
            <BreadcrumbPage>{titulos[vistaActual]}</BreadcrumbPage>
          </BreadcrumbItem>
        </BreadcrumbList>
      </Breadcrumb>

      <div className="ml-auto flex items-center gap-2">
        {!identidadAdministrativa && (
          <Button
            variant="outline"
            size="sm"
            className="hidden w-64 justify-start gap-2 text-muted-foreground md:flex"
          >
            <SearchIcon className="size-4" />
            Buscar...
            <kbd className="ml-auto rounded border bg-muted px-1.5 font-mono text-xs">
              ⌘K
            </kbd>
          </Button>
        )}
        <Badge variant="secondary" className="hidden sm:inline-flex">
          {identidadAdministrativa?.rol ?? "Borrador"}
        </Badge>
        <ThemeToggle />
        <Avatar className="size-8">
          <AvatarFallback className="text-xs">
            {identidadAdministrativa?.iniciales ?? "CM"}
          </AvatarFallback>
        </Avatar>
      </div>
    </header>
  )
}

function SeparatorVertical() {
  return <div className="h-4 w-px bg-border" />
}
