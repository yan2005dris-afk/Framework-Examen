/**
 * Configuración central del cliente HTTP y la API.
 * En desarrollo Vite reenvía /api a la API Go.
 * En producción nginx maneja la resolución hacia el backend.
 */
export const API_BASE_URL =
  (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? "/api/v1"
