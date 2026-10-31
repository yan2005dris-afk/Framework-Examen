import { afterEach, describe, expect, it, vi } from "vitest"
import {
  actualizarEstadoConvocatoria,
  actualizarFase,
  alternarHabilitacionFormulario,
  crearConvocatoria,
  listarConvocatoriasAdmin,
  obtenerConvocatoriaPublica,
} from "./convocatorias"

describe("Cliente de Convocatorias y Fases (convocatorias.ts - FT14/FT15)", () => {
  const fetchOriginal = globalThis.fetch

  afterEach(() => {
    globalThis.fetch = fetchOriginal
    vi.restoreAllMocks()
  })

  it("obtenerConvocatoriaPublica devuelve la convocatoria oficial con las 6 fases reglamentarias", async () => {
    const res = await obtenerConvocatoriaPublica()
    expect(res).toBeDefined()
    expect(res.codigo).toBe("BIESS-DIR-2026")
    expect(res.fases).toHaveLength(6)
    expect(res.fases[0].tipo).toBe("convocatoria_postulacion")
    expect(res.fases[5].tipo).toBe("designacion_final")
  })

  it("obtenerConvocatoriaPublica consume la lista de la API cuando responde exitosamente", async () => {
    const mockCompetitions = {
      items: [
        {
          id: "01931580-0000-7000-8000-000000000001",
          codigo: "BIESS-2026-TEST",
          titulo: "Concurso Prueba",
          apertura: "2026-10-01T00:00:00Z",
          cierre: "2026-12-31T00:00:00Z",
          cerrada: false,
        },
      ],
      hasMore: false,
    }

    globalThis.fetch = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(mockCompetitions), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    )

    const res = await obtenerConvocatoriaPublica()
    expect(res.codigo).toBe("BIESS-2026-TEST")
    expect(res.titulo).toBe("Concurso Prueba")
  })

  it("listarConvocatoriasAdmin devuelve la lista paginada de convocatorias", async () => {
    const mockCompetitions = {
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

    globalThis.fetch = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(mockCompetitions), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    )

    const res = await listarConvocatoriasAdmin("cursor-1")
    expect(res.items).toHaveLength(1)
    expect(res.nextCursor).toBe("cursor-2")
    expect(res.hasMore).toBe(true)
  })

  it("crearConvocatoria envía POST a /admin/competitions", async () => {
    const nueva = {
      id: "nueva-1",
      codigo: "BIESS-2027",
      titulo: "Concurso 2027",
      apertura: "2027-01-01T00:00:00Z",
      cierre: "2027-02-01T00:00:00Z",
      estado: "publicada" as const,
      cerrada: false,
      postulacionHabilitada: true,
      fases: [],
      createdAt: "2026-10-31T00:00:00Z",
    }

    const mockFetch = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(nueva), {
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
    })

    expect(res.codigo).toBe("BIESS-2027")
    const [url, opts] = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(url).toContain("/admin/competitions")
    expect(opts.method).toBe("POST")
  })

  it("actualizarEstadoConvocatoria envía PATCH a /admin/competitions/:id", async () => {
    const mockFetch = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          id: "comp-1",
          codigo: "BIESS-2026",
          cerrada: true,
          estado: "cerrada",
        }),
        {
          status: 200,
          headers: { "Content-Type": "application/json" },
        },
      ),
    )
    globalThis.fetch = mockFetch

    const res = await actualizarEstadoConvocatoria("comp-1", {
      cerrada: true,
      estado: "cerrada",
      motivo: "Resolución CD 701 finalizada",
    })

    expect(res.cerrada).toBe(true)
    const [url, opts] = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(url).toContain("/admin/competitions/comp-1")
    expect(opts.method).toBe("PATCH")
  })

  it("alternarHabilitacionFormulario envía PATCH a /admin/competitions/:id/postulacion", async () => {
    const mockFetch = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ ok: true }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    )
    globalThis.fetch = mockFetch

    const habilitada = await alternarHabilitacionFormulario(
      "comp-1",
      false,
      "Suspensión temporal",
    )

    expect(habilitada).toBe(false)
    const [url, opts] = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(url).toContain("/admin/competitions/comp-1/postulacion")
    expect(opts.method).toBe("PATCH")
  })

  it("actualizarFase envía PATCH a /admin/competitions/:id/fases/:faseId", async () => {
    const mockFetch = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          id: "fase-1",
          convocatoriaId: "comp-1",
          estado: "finalizada",
        }),
        {
          status: 200,
          headers: { "Content-Type": "application/json" },
        },
      ),
    )
    globalThis.fetch = mockFetch

    const fase = await actualizarFase("comp-1", "fase-1", {
      estado: "finalizada",
      motivo: "Etapa de méritos concluida",
    })

    expect(fase.estado).toBe("finalizada")
    const [url, opts] = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(url).toContain("/admin/competitions/comp-1/fases/fase-1")
    expect(opts.method).toBe("PATCH")
  })
})
