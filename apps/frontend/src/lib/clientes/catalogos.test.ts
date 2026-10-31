import { afterEach, describe, expect, it, vi } from "vitest"

import { obtenerCantones, obtenerPaises, obtenerProvincias } from "./catalogos"

describe("Cliente de Catálogos (catalogos.ts)", () => {
  const fetchOriginal = globalThis.fetch

  afterEach(() => {
    globalThis.fetch = fetchOriginal
    vi.restoreAllMocks()
  })

  it("obtenerPaises devuelve los países remotos si la API responde", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ paises: ["Ecuador", "Colombia"] }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    )

    const paises = await obtenerPaises()
    expect(paises).toEqual(["Ecuador", "Colombia"])
  })

  it("obtenerPaises recurre al catálogo local si la API falla", async () => {
    globalThis.fetch = vi.fn().mockRejectedValue(new Error("Network error"))

    const paises = await obtenerPaises()
    expect(paises).toContain("Ecuador")
    expect(paises).toContain("Perú")
  })

  it("obtenerProvincias devuelve la lista de provincias de Ecuador", async () => {
    globalThis.fetch = vi.fn().mockRejectedValue(new Error("API no disponible"))

    const provincias = await obtenerProvincias("Ecuador")
    expect(provincias).toContain("Pichincha")
    expect(provincias).toContain("Guayas")
    expect(provincias).toContain("Azuay")
  })

  it("obtenerCantones devuelve los cantones correspondientes a la provincia", async () => {
    globalThis.fetch = vi.fn().mockRejectedValue(new Error("API no disponible"))

    const cantones = await obtenerCantones("Guayas", "Ecuador")
    expect(cantones).toContain("Guayaquil")
    expect(cantones).toContain("Samborondón")
    expect(cantones).toContain("Durán")
  })
})
