import { afterEach, describe, expect, it, vi } from "vitest"

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

  it("solicitarRegistro envía POST a /auth/register/request", async () => {
    const mockFetch = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ mensaje: "Código enviado" }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    )
    globalThis.fetch = mockFetch

    const res = await solicitarRegistro({
      cedula: "0923456789",
      correo: "test@example.com",
      metodo_validacion: "fecha_expedicion",
    })

    expect(res.mensaje).toBe("Código enviado")
    const [url, opts] = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(url).toBe("/api/v1/auth/register/request")
    expect(opts.method).toBe("POST")
    expect(JSON.parse(String(opts.body))).toEqual({
      cedula: "0923456789",
      correo: "test@example.com",
      metodo_validacion: "fecha_expedicion",
    })
  })

  it("verificarYRegistrar envía POST a /auth/register/verify y recibe postulante", async () => {
    const mockPostulante = {
      id: "usr-1",
      cedula: "0923456789",
      nombres: "Juan",
      apellidos: "Pérez",
      nombre_completo: "Juan Pérez",
      correo: "test@example.com",
      metodo_validacion: "fecha_expedicion",
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
            metodo_validacion: "fecha_expedicion",
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
            metodo_validacion: "fecha_expedicion",
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

  it("solicitarRecuperacionPassword y restablecerPassword envían las rutas esperadas", async () => {
    const mockFetch = vi.fn().mockImplementation(() =>
      Promise.resolve(
        new Response(JSON.stringify({ mensaje: "OK" }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      ),
    )
    globalThis.fetch = mockFetch

    await solicitarRecuperacionPassword({ cedula: "0923456789" })
    const [url1] = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(url1).toBe("/api/v1/auth/password/forgot")

    await restablecerPassword({
      cedula: "0923456789",
      codigo_otp: "123456",
      new_password: "NewPassword123!",
    })
    const [url2] = mockFetch.mock.calls[1] as [string, RequestInit]
    expect(url2).toBe("/api/v1/auth/password/reset")
  })
})
