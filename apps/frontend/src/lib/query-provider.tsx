import { QueryClientProvider } from "@tanstack/react-query"
import { useState } from "react"

import { crearQueryClient, type ProveedorQueryProps } from "./query-client"

/**
 * Proveedor de TanStack Query para la aplicación.
 * Permite inyectar un QueryClient personalizado (útil en tests) o crea uno por sesión.
 */
export function ProveedorQuery({ children, client }: ProveedorQueryProps) {
  const [queryClient] = useState(() => client ?? crearQueryClient())

  return (
    <QueryClientProvider client={client ?? queryClient}>
      {children}
    </QueryClientProvider>
  )
}
