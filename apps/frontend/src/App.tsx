import { RouterProvider } from "react-router"

import { ThemeProvider } from "@/components/theme-provider"
import { Toaster } from "@/components/ui/sonner"
import { TooltipProvider } from "@/components/ui/tooltip"
import { ProveedorSesion } from "@/features/auth/sesion-context"
import { router } from "@/router"

function App() {
  return (
    <ThemeProvider
      attribute="class"
      defaultTheme="system"
      enableSystem
      disableTransitionOnChange
    >
      <TooltipProvider>
        <ProveedorSesion>
          <RouterProvider router={router} />
          <Toaster />
        </ProveedorSesion>
      </TooltipProvider>
    </ThemeProvider>
  )
}

export default App
