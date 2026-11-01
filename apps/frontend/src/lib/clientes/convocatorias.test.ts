import { afterEach, describe, expect, it, vi } from "vitest"
import {
  guardarConfiguracionConvocatoria,
  crearConvocatoria,
  listarConvocatoriasPublicas,
  obtenerDetallePublico,
  listarConvocatoriasAdmin,
  obtenerConvocatoriaAdmin,
  obtenerConvocatoriaPublica,
} from "./convocatorias"
import { ErrorApi } from "../api-error"

afterEach(() => vi.unstubAllGlobals())
describe("Contrato HTTP de convocatorias T18", () => {
  it("mantiene el cursor opaco y solicita detalle público sin endpoint administrativo", async () => {
    const fetch = vi.fn().mockImplementation(() =>
      Promise.resolve(
        new Response(JSON.stringify({ items: [], hasMore: false }), {
          headers: { "Content-Type": "application/json" },
        }),
      ),
    )
    vi.stubGlobal("fetch", fetch)
    await listarConvocatoriasPublicas("cursor+/opaco=")
    expect(String(fetch.mock.calls[0][0])).toContain(
      "cursor=cursor%2B%2Fopaco%3D",
    )
    await obtenerDetallePublico("id/seguro")
    expect(String(fetch.mock.calls[1][0])).toContain(
      "/competitions/id%2Fseguro",
    )
    expect(String(fetch.mock.calls[1][0])).not.toContain("/admin/")
  })
  it("crea sin aprobación ni publicación implícitas", async () => {
    const fetch = vi.fn().mockImplementation(
      async () =>
        new Response(JSON.stringify({ data: { id: "qa" } }), {
          headers: { "Content-Type": "application/json" },
        }),
    )
    vi.stubGlobal("fetch", fetch)
    await crearConvocatoria({
      codigo: "QA",
      titulo: "Sintético",
      apertura: "2027-01-01T05:00:00Z",
      cierre: "2027-02-01T05:00:00Z",
      sectores: ["afiliados"],
      fases: [],
    })
    const opciones = fetch.mock.calls[0][1] as RequestInit
    expect(opciones.method).toBe("POST")
    expect(opciones.body).not.toContain("inicioAprobado")
    expect(opciones.body).not.toContain('"rol"')
  })
  it("un conflicto de versión conserva el error y no se reintenta como éxito", async () => {
    const fetch = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          error: { codigo: "version_conflicto", mensaje: "Versión obsoleta" },
        }),
        { status: 409, headers: { "Content-Type": "application/json" } },
      ),
    )
    vi.stubGlobal("fetch", fetch)
    await expect(
      guardarConfiguracionConvocatoria("qa", {
        version: 3,
        inicioAprobado: true,
        publicada: false,
        ef12Habilitado: false,
        sectores: ["afiliados"],
        fases: [],
      }),
    ).rejects.toMatchObject<Partial<ErrorApi>>({
      estado: 409,
      codigo: "version_conflicto",
    })
    expect(fetch).toHaveBeenCalledTimes(1)
    expect(
      JSON.parse(String((fetch.mock.calls[0][1] as RequestInit).body)).version,
    ).toBe(3)
  })
})

describe("Cliente de Convocatorias y Fases (convocatorias.ts - Contrato T18)", () => {
  const fetchOriginal = globalThis.fetch

  afterEach(() => {
    globalThis.fetch = fetchOriginal
    vi.restoreAllMocks()
  })

  it("obtenerConvocatoriaPublica devuelve null si la API responde con lista vacía (sin inventar datos)", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          items: [],
          hasMore: false,
          limit: 1,
        }),
        {
          status: 200,
          headers: { "Content-Type": "application/json" },
        },
      ),
    )

    const res = await obtenerConvocatoriaPublica()
    expect(res).toBeNull()
  })

  it("obtenerConvocatoriaPublica devuelve la convocatoria pública publicada de la API", async () => {
    const mockPublica = {
      id: "01931580-0000-7000-8000-000000000001",
      codigo: "BIESS-2026-OFICIAL",
      titulo: "Concurso Oficial BIESS",
      apertura: "2026-10-15T00:00:00-05:00",
      cierre: "2026-11-15T23:59:59-05:00",
      sectores: ["afiliados", "jubilados", "empleadores"],
      fases: [
        {
          codigo: "postulacion",
          titulo: "Recepción de postulaciones",
          apertura: "2026-10-15T00:00:00-05:00",
          cierre: "2026-11-15T23:59:59-05:00",
          publicada: true,
          habilitada: true,
        },
      ],
      ef12Disponible: true,
      zonaHoraria: "America/Guayaquil",
      limites: "[apertura,cierre)",
    }

    globalThis.fetch = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          items: [mockPublica],
          hasMore: false,
          limit: 1,
        }),
        {
          status: 200,
          headers: { "Content-Type": "application/json" },
        },
      ),
    )

    const res = await obtenerConvocatoriaPublica()
    expect(res).not.toBeNull()
    expect(res?.codigo).toBe("BIESS-2026-OFICIAL")
    expect(res?.ef12Disponible).toBe(true)
    expect(res?.fases).toHaveLength(1)
  })

  it("obtenerDetallePublico consulta GET /competitions/{id}", async () => {
    const mockDetalle = {
      id: "comp-123",
      codigo: "BIESS-2026",
      titulo: "Concurso 2026",
      apertura: "2026-10-15T00:00:00Z",
      cierre: "2026-11-15T23:59:59Z",
      sectores: ["afiliados"],
      fases: [],
      ef12Disponible: false,
      zonaHoraria: "America/Guayaquil",
      limites: "[apertura,cierre)",
    }

    const mockFetch = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(mockDetalle), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    )
    globalThis.fetch = mockFetch

    const res = await obtenerDetallePublico("comp-123")
    expect(res.id).toBe("comp-123")
    const [url] = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(url).toContain("/competitions/comp-123")
  })

  it("listarConvocatoriasAdmin envía GET a /admin/competitions", async () => {
    const mockList = {
      items: [
        {
          id: "comp-1",
          codigo: "BIESS-DIR-2026",
          titulo: "Directorio BIESS",
          apertura: "2026-10-15T00:00:00Z",
          cierre: "2026-11-15T23:59:00Z",
          cerrada: false,
        },
      ],
      nextCursor: "cursor-2",
      hasMore: true,
      limit: 20,
    }

    const mockFetch = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(mockList), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    )
    globalThis.fetch = mockFetch

    const res = await listarConvocatoriasAdmin("cursor-1")
    expect(res.items).toHaveLength(1)
    const [url] = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(url).toContain("/admin/competitions")
  })

  it("obtenerConvocatoriaAdmin envía GET a /admin/competitions/{id}", async () => {
    const mockAdmin = {
      id: "comp-1",
      codigo: "BIESS-DIR-2026",
      titulo: "Directorio BIESS",
      apertura: "2026-10-15T00:00:00Z",
      cierre: "2026-11-15T23:59:00Z",
      version: 3,
      inicioAprobado: true,
      publicada: true,
      ef12Habilitado: true,
      sectores: ["afiliados", "jubilados"],
      fases: [],
    }

    const mockFetch = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(mockAdmin), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    )
    globalThis.fetch = mockFetch

    const res = await obtenerConvocatoriaAdmin("comp-1")
    expect(res.version).toBe(3)
    expect(res.inicioAprobado).toBe(true)
    const [url] = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(url).toContain("/admin/competitions/comp-1")
  })

  it("crearConvocatoria envía POST a /admin/competitions con sectores y período deshabilitado", async () => {
    const mockCreada = {
      id: "comp-nueva",
      codigo: "BIESS-2027",
      titulo: "Concurso 2027",
      apertura: "2027-01-01T00:00:00Z",
      cierre: "2027-02-01T00:00:00Z",
      version: 1,
      inicioAprobado: false,
      publicada: false,
      ef12Habilitado: false,
      sectores: ["afiliados", "jubilados", "empleadores"],
      fases: [],
    }

    const mockFetch = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(mockCreada), {
        status: 201,
        headers: { "Content-Type": "application/json" },
      }),
    )
    globalThis.fetch = mockFetch

    const res = await crearConvocatoria({
      codigo: "BIESS-2027",
      titulo: "Concurso 2027",
      apertura: "2027-01-01T00:00:00Z",
      cierre: "2027-02-01T00:00:00Z",
      sectores: ["afiliados", "jubilados", "empleadores"],
    })

    expect(res.id).toBe("comp-nueva")
    expect(res.inicioAprobado).toBe(false)
    const [url, opts] = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(url).toContain("/admin/competitions")
    expect(opts.method).toBe("POST")
  })

  it("guardarConfiguracionConvocatoria envía PUT con version a /admin/competitions/{id}/configuration", async () => {
    const mockConfig = {
      version: 2,
      inicioAprobado: true,
      publicada: true,
      ef12Habilitado: true,
      sectores: ["afiliados" as const],
      fases: [],
    }

    const mockFetch = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          id: "comp-1",
          ...mockConfig,
          version: 3,
        }),
        {
          status: 200,
          headers: { "Content-Type": "application/json" },
        },
      ),
    )
    globalThis.fetch = mockFetch

    const res = await guardarConfiguracionConvocatoria("comp-1", mockConfig)
    expect(res.version).toBe(3)
    const [url, opts] = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(url).toContain("/admin/competitions/comp-1/configuration")
    expect(opts.method).toBe("PUT")
    const cuerpo = JSON.parse(opts.body as string)
    expect(cuerpo.version).toBe(2)
  })
})
