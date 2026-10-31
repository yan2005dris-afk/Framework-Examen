import { QueryClient, type QueryClientConfig } from "@tanstack/react-query"
import type { ReactNode } from "react"

import { esErrorApi } from "./api-error"

/**
 * Determina si una consulta fallida debe reintentarse.
 *
 * Regla del RF/TRD (FT4):
 * - Errores de red y errores de servidor (5xx) se reintentan con retroceso exponencial.
 * - Errores de cliente 4xx (400, 401, 403, 404, 409, 422) NO se reintentan.
 * - Límite máximo de 3 reintentos.
 */
export function deberiaReintentar(
  failureCount: number,
  error: unknown,
): boolean {
  if (failureCount >= 3) {
    return false
  }

  if (esErrorApi(error)) {
    // Los errores 4xx son fallos de cliente o reglas de negocio, no fallos transitorios.
    if (error.estado >= 400 && error.estado < 500) {
      return false
    }
  }

  return true
}

/**
 * Calcula el tiempo de espera con retroceso exponencial entre reintentos.
 * Base de 1000ms con tope de 30 segundos: 1s, 2s, 4s...
 */
export function calcularRetrocesoExponencial(attemptIndex: number): number {
  return Math.min(1000 * 2 ** attemptIndex, 30000)
}

/**
 * Crea una nueva instancia de QueryClient configurada con las políticas de
 * reintento, caché y staleTime del proyecto.
 */
export function crearQueryClient(config?: QueryClientConfig): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 1000 * 60 * 5, // 5 minutos de validez por defecto
        gcTime: 1000 * 60 * 10, // 10 minutos antes de liberar memoria
        retry: deberiaReintentar,
        retryDelay: calcularRetrocesoExponencial,
        refetchOnWindowFocus: false,
      },
      mutations: {
        retry: false,
      },
    },
    ...config,
  })
}

export interface ProveedorQueryProps {
  children: ReactNode
  client?: QueryClient
}
