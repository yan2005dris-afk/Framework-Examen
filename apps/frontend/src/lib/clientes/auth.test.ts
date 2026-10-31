import { afterEach, describe, expect, it, vi } from "vitest"

import { esErrorApi } from "../api-error"
import {
  cerrarSesion,
  iniciarSesion,
  obtenerSesion,
  restablecerPassword,
  solicitarRecuperacionPassword,
  solicitarRegistro,
  verificarYRegistrar,
} from "./auth"

describe("Cliente de Autenticación e Identidad (auth.ts)", () => {
  const fetchOriginal = globalThis.fetch

  afterEach(() => {
    globalThis.fetch = fetchOriginal
    vi.restoreAllMocks()
  })

  it("solicitarRegistro envía POST a /auth/register/request y devuelve DesafioOtp", async () => {
    const expiraIso = "2026-10-31T20:00:00Z"
    const mockFetch = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          mensaje: "Código enviado",
          desafio_id: "desafio-uuid-123",
          expira_en: expiraIso,
        }),
        {
          status: 200,
          headers: { "Content-Type": "application/json" },
        },
      ),
    )
    globalThis.fetch = mockFetch

    const res = await solicitarRegistro({
      cedula: "0923456789",
      correo: "test@example.com",
      metodo_validacion: "correo",
    })

    expect(res.mensaje).toBe("Código enviado")
    expect(res.desafioId).toBe("desafio-uuid-123")
    expect(res.expiraEn.toISOString()).toBe(new Date(expiraIso).toISOString())

    const [url, opts] = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(url).toBe("/api/v1/auth/register/request")
    expect(opts.method).toBe("POST")
    expect(JSON.parse(String(opts.body))).toEqual({
      cedula: "0923456789",
      correo: "test@example.com",
      metodo_validacion: "correo",
    })
  })

  it("solicitarRegistro captura Retry-After y X-Retry-At en error 429", async () => {
    const retryAtIso = "2026-10-31T20:10:00Z"
    const mockFetch = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          error: {
            codigo: "reenvio_limitado",
            mensaje: "Espere antes de solicitar otro código",
          },
        }),
        {
          status: 429,
          headers: {
            "Content-Type": "application/json",
            "Retry-After": "45",
            "X-Retry-At": retryAtIso,
          },
        },
      ),
    )
    globalThis.fetch = mockFetch

    try {
      await solicitarRegistro({
        cedula: "0923456789",
        correo: "test@example.com",
      })
      expect.fail("Debería haber lanzado un ErrorApi")
    } catch (err) {
      expect(esErrorApi(err)).toBe(true)
      if (esErrorApi(err)) {
        expect(err.estado).toBe(429)
        expect(err.codigo).toBe("reenvio_limitado")
        expect(err.esReenvioLimitado()).toBe(true)
        expect(err.retryAfterSeg).toBe(45)
        expect(err.retryAt?.toISOString()).toBe(new Date(retryAtIso).toISOString())
      }
    }
  })

  it("verificarYRegistrar envía POST a /auth/register/verify y recibe postulante", async () => {
    const mockPostulante = {
      id: "usr-1",
      cedula: "0923456789",
      nombres: "Juan",
      apellidos: "Pérez",
      nombre_completo: "Juan Pérez",
      correo: "test@example.com",
      metodo_validacion: "correo",
      fecha_registro: "2026-10-31T10:00:00Z",
    }

    const mockFetch = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ postulante: mockPostulante }), {
        status: 201,
        headers: { "Content-Type": "application/json" },
      }),
    )
    globalThis.fetch = mockFetch

    const res = await verificarYRegistrar({
      cedula: "0923456789",
      correo: "test@example.com",
      codigo_otp: "123456",
      nombres: "Juan",
      apellidos: "Pérez",
      password: "Password123!",
    })

    expect(res.postulante.id).toBe("usr-1")
    expect(res.postulante.nombre_completo).toBe("Juan Pérez")
  })

  it("iniciarSesion envía POST a /auth/login", async () => {
    const mockFetch = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          postulante: {
            id: "usr-1",
            cedula: "0923456789",
            nombres: "Juan",
            apellidos: "Pérez",
            nombre_completo: "Juan Pérez",
            correo: "test@example.com",
            metodo_validacion: "correo",
            fecha_registro: "2026-10-31T10:00:00Z",
          },
        }),
        {
          status: 200,
          headers: { "Content-Type": "application/json" },
        },
      ),
    )
    globalThis.fetch = mockFetch

    const res = await iniciarSesion({
      cedula: "0923456789",
      password: "Password123!",
    })

    expect(res.postulante.cedula).toBe("0923456789")
  })

  it("obtenerSesion envía GET a /auth/session", async () => {
    const mockFetch = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          postulante: {
            id: "usr-1",
            cedula: "0923456789",
            nombres: "Juan",
            apellidos: "Pérez",
            nombre_completo: "Juan Pérez",
            correo: "test@example.com",
            metodo_validacion: "correo",
            fecha_registro: "2026-10-31T10:00:00Z",
          },
        }),
        {
          status: 200,
          headers: { "Content-Type": "application/json" },
        },
      ),
    )
    globalThis.fetch = mockFetch

    const res = await obtenerSesion()
    expect(res.postulante.id).toBe("usr-1")
    const [url, opts] = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(url).toBe("/api/v1/auth/session")
    expect(opts.method).toBe("GET")
  })

  it("cerrarSesion envía POST a /auth/logout", async () => {
    const mockFetch = vi.fn().mockResolvedValue(
      new Response(null, {
        status: 204,
      }),
    )
    globalThis.fetch = mockFetch

    await expect(cerrarSesion()).resolves.toBeUndefined()
    const [url, opts] = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(url).toBe("/api/v1/auth/logout")
    expect(opts.method).toBe("POST")
  })

  it("solicitarRecuperacionPassword y restablecerPassword envían las rutas esperadas y devuelven DesafioOtp", async () => {
    const expiraIso = "2026-10-31T20:00:00Z"
    const mockFetch = vi.fn().mockImplementation((url: string) => {
      if (url.includes("/auth/password/forgot")) {
        return Promise.resolve(
          new Response(
            JSON.stringify({
              mensaje: "Código enviado",
              desafio_id: "desafio-recup-123",
              expira_en: expiraIso,
            }),
            {
              status: 200,
              headers: { "Content-Type": "application/json" },
            },
          ),
        )
      }
      return Promise.resolve(
        new Response(JSON.stringify({ mensaje: "Contraseña actualizada" }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      )
    })
    globalThis.fetch = mockFetch

    const desafio = await solicitarRecuperacionPassword({ cedula: "0923456789" })
    expect(desafio.desafioId).toBe("desafio-recup-123")
    expect(desafio.expiraEn.toISOString()).toBe(new Date(expiraIso).toISOString())

    const [url1] = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(url1).toBe("/api/v1/auth/password/forgot")

    const resReset = await restablecerPassword({
      cedula: "0923456789",
      codigo_otp: "123456",
      new_password: "NewPassword123!",
    })
    expect(resReset.mensaje).toBe("Contraseña actualizada")
    const [url2] = mockFetch.mock.calls[1] as [string, RequestInit]
    expect(url2).toBe("/api/v1/auth/password/reset")
  })
})
