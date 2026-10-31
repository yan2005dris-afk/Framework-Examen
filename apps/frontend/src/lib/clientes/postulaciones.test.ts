import { afterEach, describe, expect, it, vi } from "vitest"

import {
  actualizarPostulacion,
  crearPostulacion,
  eliminarPostulacion,
  guardarSeccion,
  listarPostulaciones,
  listarPostulacionesCursor,
  obtenerPostulacion,
  sellarPostulacion,
} from "./postulaciones"

describe("Cliente de Postulaciones EF-12 (postulaciones.ts)", () => {
  const fetchOriginal = globalThis.fetch

  const mockPostulacion = {
    id: "sub-123",
    applicantId: "app-456",
    code: "POST-2026-001",
    status: "draft" as const,
    content: {},
    createdAt: "2026-10-31T10:00:00Z",
    updatedAt: "2026-10-31T10:00:00Z",
    progress: {
      total: 9,
      completed: 1,
      percentage: 11,
      sections: [{ number: 1, complete: true }],
    },
  }

  afterEach(() => {
    globalThis.fetch = fetchOriginal
    vi.restoreAllMocks()
  })

  it("crearPostulacion envía POST a /submissions", async () => {
    const mockFetch = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(mockPostulacion), {
        status: 201,
        headers: { "Content-Type": "application/json" },
      }),
    )
    globalThis.fetch = mockFetch

    const res = await crearPostulacion({ applicantId: "app-456" })
    expect(res.id).toBe("sub-123")
    const [url, opts] = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(url).toBe("/api/v1/submissions")
    expect(opts.method).toBe("POST")
  })

  it("obtenerPostulacion envía GET a /submissions/{id}", async () => {
    const mockFetch = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(mockPostulacion), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    )
    globalThis.fetch = mockFetch

    const res = await obtenerPostulacion("sub-123")
    expect(res.id).toBe("sub-123")
    const [url] = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(url).toBe("/api/v1/submissions/sub-123")
  })

  it("listarPostulaciones envía query params de paginación", async () => {
    const mockPagina = {
      items: [mockPostulacion],
      page: 1,
      pageSize: 10,
      total: 1,
    }

    const mockFetch = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(mockPagina), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    )
    globalThis.fetch = mockFetch

    const res = await listarPostulaciones({
      applicantId: "app-456",
      page: 1,
      pageSize: 10,
    })

    expect(res.items).toHaveLength(1)
    const [url] = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(url).toBe(
      "/api/v1/submissions?applicantId=app-456&page=1&pageSize=10",
    )
  })

  it("listarPostulacionesCursor envía cursor y limit", async () => {
    const mockPaginaCursor = {
      items: [mockPostulacion],
      nextCursor: "token-abc",
      hasMore: true,
      limit: 10,
    }

    const mockFetch = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(mockPaginaCursor), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    )
    globalThis.fetch = mockFetch

    const res = await listarPostulacionesCursor({
      applicantId: "app-456",
      cursor: "token-prev",
      limit: 10,
    })

    expect(res.nextCursor).toBe("token-abc")
    const [url] = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(url).toBe(
      "/api/v1/submissions?applicantId=app-456&limit=10&cursor=token-prev",
    )
  })

  it("actualizarPostulacion envía PUT a /submissions/{id}", async () => {
    const mockFetch = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(mockPostulacion), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    )
    globalThis.fetch = mockFetch

    const res = await actualizarPostulacion("sub-123", {
      seccion1: { nombres: "Ana" },
    })
    expect(res.id).toBe("sub-123")
    const [url, opts] = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(url).toBe("/api/v1/submissions/sub-123")
    expect(opts.method).toBe("PUT")
    expect(JSON.parse(String(opts.body))).toEqual({
      content: { seccion1: { nombres: "Ana" } },
    })
  })

  it("guardarSeccion envía POST a /submissions/{id}/sections/{n}", async () => {
    const mockFetch = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(mockPostulacion), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    )
    globalThis.fetch = mockFetch

    const datos = { nombres: "Damian", apellidos: "Torres" }
    const res = await guardarSeccion("sub-123", 1, datos)

    expect(res.progress.completed).toBe(1)
    const [url, opts] = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(url).toBe("/api/v1/submissions/sub-123/sections/1")
    expect(opts.method).toBe("POST")
    expect(JSON.parse(String(opts.body))).toEqual(datos)
  })

  it("sellarPostulacion envía POST a /submissions/{id}/submit y devuelve estado submitted", async () => {
    const sellado = {
      ...mockPostulacion,
      status: "submitted" as const,
      hash: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
      submittedAt: "2026-10-31T11:00:00Z",
    }

    const mockFetch = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(sellado), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    )
    globalThis.fetch = mockFetch

    const res = await sellarPostulacion("sub-123", {
      declaracionVeracidad: true,
    })

    expect(res.status).toBe("submitted")
    expect(res.hash).toBeDefined()
    const [url, opts] = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(url).toBe("/api/v1/submissions/sub-123/submit")
    expect(opts.method).toBe("POST")
  })

  it("eliminarPostulacion envía DELETE a /submissions/{id}", async () => {
    const mockFetch = vi.fn().mockResolvedValue(
      new Response(null, {
        status: 204,
      }),
    )
    globalThis.fetch = mockFetch

    await expect(eliminarPostulacion("sub-123")).resolves.toBeUndefined()
    const [url, opts] = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(url).toBe("/api/v1/submissions/sub-123")
    expect(opts.method).toBe("DELETE")
  })
})
