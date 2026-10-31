import { useState } from "react"

import { AppHeader } from "@/components/app-header"
import { AppSidebar, type VistaId } from "@/components/app-sidebar"
import { ThemeProvider } from "@/components/theme-provider"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar"
import { Toaster } from "@/components/ui/sonner"
import { TooltipProvider } from "@/components/ui/tooltip"
import { DashboardView } from "@/pages/dashboard-view"

function App() {
  const [vistaActual, setVistaActual] = useState<VistaId>("dashboard")

  return (
    <ThemeProvider
      attribute="class"
      defaultTheme="system"
      enableSystem
      disableTransitionOnChange
    >
      <TooltipProvider>
        <SidebarProvider>
          <AppSidebar vistaActual={vistaActual} onNavigate={setVistaActual} />
          <SidebarInset>
            <AppHeader vistaActual={vistaActual} />
            {vistaActual === "dashboard" ? (
              <DashboardView />
            ) : (
              <div className="flex flex-1 flex-col gap-4 p-4">
                <Card>
                  <CardHeader>
                    <CardTitle>Pantalla en construcción</CardTitle>
                    <CardDescription>
                      Esta vista se migrará desde el mockup en los siguientes
                      pasos, usando los componentes de shadcn/ui ya instalados.
                    </CardDescription>
                  </CardHeader>
                  <CardContent>
                    <Button
                      variant="outline"
                      onClick={() => setVistaActual("dashboard")}
                    >
                      Volver al inicio
                    </Button>
                  </CardContent>
                </Card>
              </div>
            )}
          </SidebarInset>
        </SidebarProvider>
        <Toaster />
      </TooltipProvider>
    </ThemeProvider>
  )
}

export default App
