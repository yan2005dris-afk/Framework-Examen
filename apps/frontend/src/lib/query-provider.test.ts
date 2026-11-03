import { describe, expect, it } from "vitest"

import { ErrorApi } from "./api-error"
import {
  calcularRetrocesoExponencial,
  crearQueryClient,
  deberiaReintentar,
} from "./query-client"

describe("query-client", () => {
  describe("deberiaReintentar", () => {
    it("no reintenta si ya se alcanzaron 3 intentos fallidos", () => {
      const errorRed = new ErrorApi("Fallo de conexión", {
        estado: 0,
        codigo: "error_red",
      })
      expect(deberiaReintentar(3, errorRed)).toBe(false)
      expect(deberiaReintentar(4, errorRed)).toBe(false)
    })

    it("reintenta errores de red (estado 0) antes del límite", () => {
      const errorRed = new ErrorApi("Fallo de conexión", {
        estado: 0,
        codigo: "error_red",
      })
      expect(deberiaReintentar(0, errorRed)).toBe(true)
      expect(deberiaReintentar(1, errorRed)).toBe(true)
      expect(deberiaReintentar(2, errorRed)).toBe(true)
    })

    it("reintenta errores de servidor 5xx", () => {
      const error500 = new ErrorApi("Error interno del servidor", {
        estado: 500,
        codigo: "error_interno",
      })
      const error503 = new ErrorApi("Servicio no disponible", {
        estado: 503,
        codigo: "servicio_no_disponible",
      })
      expect(deberiaReintentar(0, error500)).toBe(true)
      expect(deberiaReintentar(1, error503)).toBe(true)
    })

    it("reintenta errores genéricos de JavaScript antes del límite", () => {
      const errorGenerico = new Error("TypeError inesperado")
      expect(deberiaReintentar(0, errorGenerico)).toBe(true)
      expect(deberiaReintentar(2, errorGenerico)).toBe(true)
    })

    it("NUNCA reintenta errores 4xx del cliente o de negocio", () => {
      const codigos4xx = [400, 401, 403, 404, 409, 422, 429]
      for (const estado of codigos4xx) {
        const error = new ErrorApi(`Error ${estado}`, {
          estado,
          codigo: `error_${estado}`,
        })
        expect(deberiaReintentar(0, error)).toBe(false)
        expect(deberiaReintentar(1, error)).toBe(false)
      }
    })
  })

  describe("calcularRetrocesoExponencial", () => {
    it("calcula backoff exponencial con base 1s", () => {
      expect(calcularRetrocesoExponencial(0)).toBe(1000)
      expect(calcularRetrocesoExponencial(1)).toBe(2000)
      expect(calcularRetrocesoExponencial(2)).toBe(4000)
      expect(calcularRetrocesoExponencial(3)).toBe(8000)
    })

    it("limita el tiempo máximo de espera a 30 segundos", () => {
      expect(calcularRetrocesoExponencial(5)).toBe(30000)
      expect(calcularRetrocesoExponencial(10)).toBe(30000)
    })
  })

  describe("crearQueryClient", () => {
    it("crea un cliente con los valores por defecto configurados", () => {
      const cliente = crearQueryClient()
      const opciones = cliente.getDefaultOptions()

      expect(opciones.queries?.staleTime).toBe(1000 * 60 * 5)
      expect(opciones.queries?.gcTime).toBe(1000 * 60 * 10)
      expect(opciones.queries?.refetchOnWindowFocus).toBe(false)
      expect(opciones.mutations?.retry).toBe(false)
    })
  })
})
