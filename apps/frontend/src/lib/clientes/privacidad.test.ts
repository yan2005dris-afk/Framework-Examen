import { afterEach, describe, expect, it, vi } from "vitest"
import {
  listarPoliticasConvocatoria,
  listarRevisionesRetencion,
  obtenerPoliticaPorId,
  obtenerPoliticaVigenteConvocatoria,
  publicarPoliticaPrivacidad,
  restringirPoliticaPrivacidad,
  type PoliticaPrivacidadDto,
  type PublicarPoliticaDto,
  type RestringirPoliticaDto,
} from "./privacidad"

describe("Cliente de Privacidad y Tratamiento de Datos (privacidad.ts - FT28)", () => {
  const fetchOriginal = globalThis.fetch

  afterEach(() => {
    globalThis.fetch = fetchOriginal
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  const mockPolitica: PoliticaPrivacidadDto = {
    id: "pol-123",
    convocatoriaId: "conv-1",
    version: 1,
    versionControl: 1,
    restringido: false,
    definicion: {
      texto: "Texto legal institucional",
      informacionTratamiento: "Tratamiento de datos personales concurso",
      referenciaAprobacion: "Resolución CD-BIESS-2026-001",
      diasAcceso: 365,
      diasDocumentos: 180,
      diasReportes: 365,
      diasPublicacionNombres: 90,
      diasCorreo: 30,
      diasRevisionConservacion: 730,
      disposicionHistoricos: "revision_humana_sin_borrado_automatico",
      permiteExportacion: true,
      permiteCorreo: true,
      permiteNombresPublicos: true,
    },
    hash: "hash-pol-456",
    aprobadaPor: "user-admin-1",
    publicadaEn: "2026-10-31T20:00:00Z",
  }

  it("obtenerPoliticaVigenteConvocatoria consulta el endpoint público", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      headers: new Headers({ "content-type": "application/json" }),
      json: async () => mockPolitica,
    })
    vi.stubGlobal("fetch", fetchMock)

    const resultado = await obtenerPoliticaVigenteConvocatoria("conv-1")
    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining("/api/v1/public/privacy/competitions/conv-1"),
      expect.objectContaining({ method: "GET" }),
    )
    expect(resultado.id).toBe("pol-123")
    expect(resultado.definicion.referenciaAprobacion).toBe(
      "Resolución CD-BIESS-2026-001",
    )
  })

  it("obtenerPoliticaPorId consulta la versión documental exacta", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      headers: new Headers({ "content-type": "application/json" }),
      json: async () => mockPolitica,
    })
    vi.stubGlobal("fetch", fetchMock)

    const resultado = await obtenerPoliticaPorId("pol-123")
    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining("/api/v1/public/privacy/policies/pol-123"),
      expect.objectContaining({ method: "GET" }),
    )
    expect(resultado.hash).toBe("hash-pol-456")
  })

  it("publicarPoliticaPrivacidad envía la nueva versión al endpoint administrativo", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      headers: new Headers({ "content-type": "application/json" }),
      json: async () => ({ ...mockPolitica, version: 2 }),
    })
    vi.stubGlobal("fetch", fetchMock)

    const payload: PublicarPoliticaDto = {
      versionAnterior: 1,
      definicion: mockPolitica.definicion,
    }

    const resultado = await publicarPoliticaPrivacidad("conv-1", payload)
    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining(
        "/api/v1/admin/privacy/competitions/conv-1/policies",
      ),
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify(payload),
      }),
    )
    expect(resultado.version).toBe(2)
  })

  it("restringirPoliticaPrivacidad aplica la medida administrativa", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      headers: new Headers({ "content-type": "application/json" }),
      json: async () => ({ restringido: true }),
    })
    vi.stubGlobal("fetch", fetchMock)

    const payload: RestringirPoliticaDto = {
      versionAnterior: 1,
      restringido: true,
      motivo: "Revisión técnica cautelar",
    }

    const resultado = await restringirPoliticaPrivacidad("conv-1", payload)
    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining(
        "/api/v1/admin/privacy/competitions/conv-1/restriction",
      ),
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify(payload),
      }),
    )
    expect(resultado.restringido).toBe(true)
  })

  it("listarPoliticasConvocatoria pagina las versiones documentales", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      headers: new Headers({ "content-type": "application/json" }),
      json: async () => ({
        items: [mockPolitica],
        nextCursor: null,
        hasMore: false,
      }),
    })
    vi.stubGlobal("fetch", fetchMock)

    const resultado = await listarPoliticasConvocatoria("conv-1", {
      limit: 10,
      cursor: "cur-1",
    })
    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining(
        "/api/v1/admin/privacy/competitions/conv-1/policies?limit=10&cursor=cur-1",
      ),
      expect.anything(),
    )
    expect(resultado.items).toHaveLength(1)
  })

  it("listarRevisionesRetencion consulta vencimientos de conservación", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      headers: new Headers({ "content-type": "application/json" }),
      json: async () => ({
        items: [
          {
            postulacionId: "sub-1",
            politicaId: "pol-123",
            revisarDesde: "2028-10-31T20:00:00Z",
          },
        ],
        nextCursor: null,
        hasMore: false,
      }),
    })
    vi.stubGlobal("fetch", fetchMock)

    const resultado = await listarRevisionesRetencion("conv-1", { limit: 5 })
    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining(
        "/api/v1/admin/privacy/competitions/conv-1/retention-review?limit=5",
      ),
      expect.anything(),
    )
    expect(resultado.items[0].postulacionId).toBe("sub-1")
  })
})
