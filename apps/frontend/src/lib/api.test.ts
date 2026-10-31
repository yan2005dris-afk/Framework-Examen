import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import {
  apiDelete,
  apiFetch,
  apiGet,
  apiPatch,
  apiPost,
  apiPut,
  configurarManejadorNoAutorizado,
  construirUrl,
} from "./api"
import { ErrorApi, esErrorApi } from "./api-error"

describe("Cliente HTTP base (api.ts)", () => {
  const fetchOriginal = globalThis.fetch

  beforeEach(() => {
    configurarManejadorNoAutorizado(null)
  })

  afterEach(() => {
    globalThis.fetch = fetchOriginal
    vi.restoreAllMocks()
  })

  describe("construirUrl", () => {
    it("combina baseUrl y ruta relativa sin duplicar slashes", () => {
      const url = construirUrl("/auth/login", undefined, "/api/v1")
      expect(url).toBe("/api/v1/auth/login")
    })

    it("respeta URLs absolutas completas", () => {
      const url = construirUrl("https://api.externa.com/datos")
      expect(url).toBe("https://api.externa.com/datos")
    })

    it("serializa query params omitiendo valores nulos o indefinidos", () => {
      const url = construirUrl("/submissions", {
        applicantId: "uuid-123",
        page: 2,
        pageSize: 10,
        nulo: null,
        indefinido: undefined,
        vacio: "",
      })

      expect(url).toBe(
        "/api/v1/submissions?applicantId=uuid-123&page=2&pageSize=10",
      )
    })

    it("concatena parámetros si la ruta ya contenía un query string", () => {
      const url = construirUrl("/submissions?orden=desc", { page: 1 })
      expect(url).toBe("/api/v1/submissions?orden=desc&page=1")
    })
  })

  describe("apiFetch - Peticiones exitosas", () => {
    it("realiza peticiones GET con credentials: 'include' y cabeceras JSON", async () => {
      const mockFetch = vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ ok: true }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      )
      globalThis.fetch = mockFetch

      const resultado = await apiGet<{ ok: boolean }>("/test")

      expect(resultado).toEqual({ ok: true })
      expect(mockFetch).toHaveBeenCalledTimes(1)

      const [urlLlamada, opcionesLlamada] = mockFetch.mock.calls[0] as [
        string,
        RequestInit,
      ]
      expect(urlLlamada).toBe("/api/v1/test")
      expect(opcionesLlamada.method).toBe("GET")
      expect(opcionesLlamada.credentials).toBe("include")

      const headers = opcionesLlamada.headers as Headers
      expect(headers.get("Accept")).toBe("application/json")
    })

    it("serializa el cuerpo como JSON en peticiones POST/PUT/PATCH", async () => {
      const mockFetch = vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ id: "1" }), {
          status: 201,
          headers: { "Content-Type": "application/json" },
        }),
      )
      globalThis.fetch = mockFetch

      const respuesta = await apiPost<{ id: string }>("/recursos", {
        nombre: "Test",
      })

      expect(respuesta).toEqual({ id: "1" })
      const [, opciones] = mockFetch.mock.calls[0] as [string, RequestInit]
      expect(opciones.method).toBe("POST")
      expect(opciones.body).toBe(JSON.stringify({ nombre: "Test" }))
      const headers = opciones.headers as Headers
      expect(headers.get("Content-Type")).toContain("application/json")
    })

    it("soporta peticiones directas con apiFetch", async () => {
      const mockFetch = vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ directo: true }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      )
      globalThis.fetch = mockFetch

      const res = await apiFetch<{ directo: boolean }>("/directo", {
        method: "GET",
      })
      expect(res.directo).toBe(true)
    })

    it("maneja respuestas 204 No Content devolviendo undefined", async () => {
      globalThis.fetch = vi.fn().mockResolvedValue(
        new Response(null, {
          status: 204,
        }),
      )

      const respuesta = await apiDelete("/recursos/1")
      expect(respuesta).toBeUndefined()
    })

    it("soporta peticiones PUT y PATCH", async () => {
      const mockFetch = vi.fn().mockImplementation(() =>
        Promise.resolve(
          new Response(JSON.stringify({ actualizado: true }), {
            status: 200,
            headers: { "Content-Type": "application/json" },
          }),
        ),
      )
      globalThis.fetch = mockFetch

      await apiPut("/recursos/1", { nombre: "A" })
      expect(mockFetch.mock.calls[0][1].method).toBe("PUT")

      await apiPatch("/recursos/1", { nombre: "B" })
      expect(mockFetch.mock.calls[1][1].method).toBe("PATCH")
    })
  })

  describe("apiFetch - Manejo de errores de la API", () => {
    it("desempaqueta el error estándar del backend { error: { codigo, mensaje, detalle } } (400)", async () => {
      globalThis.fetch = vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            error: {
              codigo: "cuerpo_invalido",
              mensaje: "El cuerpo de la petición no es válido.",
              detalle: ["campo cedula requerido"],
            },
          }),
          {
            status: 400,
            headers: { "Content-Type": "application/json" },
          },
        ),
      )

      await expect(apiPost("/test", {})).rejects.toSatisfy((error) => {
        expect(esErrorApi(error)).toBe(true)
        const errApi = error as ErrorApi
        expect(errApi.estado).toBe(400)
        expect(errApi.codigo).toBe("cuerpo_invalido")
        expect(errApi.message).toBe("El cuerpo de la petición no es válido.")
        expect(errApi.detalles).toEqual(["campo cedula requerido"])
        expect(errApi.esErrorValidacion()).toBe(true)
        return true
      })
    })

    it("identifica bloqueo de cuenta por exceso de intentos (429)", async () => {
      globalThis.fetch = vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            error: {
              codigo: "intentos_excedidos",
              mensaje: "registro fallido, inténtelo nuevamente en 10 minutos",
            },
          }),
          {
            status: 429,
            headers: { "Content-Type": "application/json" },
          },
        ),
      )

      await expect(apiPost("/auth/login", {})).rejects.toSatisfy((error) => {
        const errApi = error as ErrorApi
        expect(errApi.estado).toBe(429)
        expect(errApi.codigo).toBe("intentos_excedidos")
        expect(errApi.esBloqueoIntentos()).toBe(true)
        return true
      })
    })

    it("invoca el callback global de no autorizado cuando responde 401", async () => {
      const manejador401 = vi.fn()
      configurarManejadorNoAutorizado(manejador401)

      globalThis.fetch = vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            error: {
              codigo: "no_autenticado",
              mensaje: "Debe iniciar sesión.",
            },
          }),
          {
            status: 401,
            headers: { "Content-Type": "application/json" },
          },
        ),
      )

      await expect(apiGet("/auth/session")).rejects.toSatisfy((error) => {
        const errApi = error as ErrorApi
        expect(errApi.estado).toBe(401)
        expect(errApi.esNoAutorizado()).toBe(true)
        return true
      })

      expect(manejador401).toHaveBeenCalledTimes(1)
    })

    it("maneja errores 500 y respuestas HTML/texto plano del servidor", async () => {
      globalThis.fetch = vi.fn().mockResolvedValue(
        new Response("502 Bad Gateway - Proxy Error", {
          status: 502,
          statusText: "Bad Gateway",
          headers: { "Content-Type": "text/plain" },
        }),
      )

      await expect(apiGet("/cualquier-cosa")).rejects.toSatisfy((error) => {
        const errApi = error as ErrorApi
        expect(errApi.estado).toBe(502)
        expect(errApi.codigo).toBe("error_desconocido")
        expect(errApi.message).toContain("502 Bad Gateway")
        return true
      })
    })

    it("captura errores de red y los traduce a ErrorApi con estado 0 y codigo 'error_red'", async () => {
      globalThis.fetch = vi
        .fn()
        .mockRejectedValue(new TypeError("Failed to fetch"))

      await expect(apiGet("/offline")).rejects.toSatisfy((error) => {
        const errApi = error as ErrorApi
        expect(errApi.estado).toBe(0)
        expect(errApi.codigo).toBe("error_red")
        expect(errApi.esErrorRed()).toBe(true)
        return true
      })
    })

    it("respeta la cancelación por AbortSignal sin enmascarar AbortError", async () => {
      const abortError = new Error("The operation was aborted")
      abortError.name = "AbortError"
      globalThis.fetch = vi.fn().mockRejectedValue(abortError)

      const controlador = new AbortController()
      controlador.abort()

      await expect(
        apiGet("/cancelable", { signal: controlador.signal }),
      ).rejects.toThrow("The operation was aborted")
    })
  })
})
