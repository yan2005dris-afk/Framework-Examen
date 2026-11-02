import { afterEach, describe, expect, it, vi } from "vitest"

import { obtenerCantones, obtenerPaises, obtenerProvincias } from "./catalogos"

function respuestaJson(cuerpo: unknown, status = 200): Response {
  return new Response(JSON.stringify(cuerpo), {
    status,
    headers: { "Content-Type": "application/json" },
  })
}

describe("Cliente de Catálogos (catalogos.ts)", () => {
  const fetchOriginal = globalThis.fetch

  afterEach(() => {
    globalThis.fetch = fetchOriginal
    vi.restoreAllMocks()
  })

  it("obtenerPaises devuelve los países remotos si la API responde", async () => {
    globalThis.fetch = vi
      .fn()
      .mockResolvedValue(respuestaJson({ paises: ["Ecuador", "Colombia"] }))

    const paises = await obtenerPaises()
    expect(paises).toEqual(["Ecuador", "Colombia"])
  })

  it("obtenerPaises propaga el error si la API falla", async () => {
    globalThis.fetch = vi.fn().mockRejectedValue(new Error("Network error"))

    await expect(obtenerPaises()).rejects.toThrow()
  })

  it("obtenerProvincias consulta la API de Ecuador", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(respuestaJson({ provincias: ["Azuay", "Guayas"] }))
    globalThis.fetch = fetchMock

    const provincias = await obtenerProvincias("Ecuador")
    expect(provincias).toEqual(["Azuay", "Guayas"])
    expect(String(fetchMock.mock.calls[0][0])).toContain(
      "/catalogs/provinces/Ecuador",
    )
  })

  it("obtenerProvincias propaga el error si la API falla", async () => {
    globalThis.fetch = vi
      .fn()
      .mockResolvedValue(
        respuestaJson(
          { error: "catalogo_no_disponible", mensaje: "No disponible" },
          503,
        ),
      )

    await expect(obtenerProvincias("Ecuador")).rejects.toThrow()
  })

  it("obtenerProvincias no consulta la API para otro país", async () => {
    const fetchMock = vi.fn()
    globalThis.fetch = fetchMock

    expect(await obtenerProvincias("Colombia")).toEqual([])
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it("obtenerCantones consulta la API con país y provincia", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(respuestaJson({ cantones: ["Durán", "Guayaquil"] }))
    globalThis.fetch = fetchMock

    const cantones = await obtenerCantones("Guayas", "Ecuador")
    expect(cantones).toEqual(["Durán", "Guayaquil"])
    expect(String(fetchMock.mock.calls[0][0])).toContain(
      "/catalogs/cities/Ecuador/Guayas",
    )
  })

  it("obtenerCantones propaga el error si la API falla", async () => {
    globalThis.fetch = vi.fn().mockRejectedValue(new Error("API no disponible"))

    await expect(obtenerCantones("Guayas", "Ecuador")).rejects.toThrow()
  })

  it("obtenerCantones no consulta la API sin provincia o para otro país", async () => {
    const fetchMock = vi.fn()
    globalThis.fetch = fetchMock

    expect(await obtenerCantones("", "Ecuador")).toEqual([])
    expect(await obtenerCantones("Antioquia", "Colombia")).toEqual([])
    expect(fetchMock).not.toHaveBeenCalled()
  })
})
