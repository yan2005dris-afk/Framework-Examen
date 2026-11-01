import { afterEach, describe, expect, it, vi } from "vitest"
import {
  guardarConfiguracionConvocatoria,
  crearConvocatoria,
  listarConvocatoriasPublicas,
  obtenerDetallePublico,
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
