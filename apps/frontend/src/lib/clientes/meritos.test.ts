import { afterEach, describe, expect, it, vi } from "vitest"
import {
  calcularMeritos,
  centipuntosAPuntos,
  crearRevisionMeritos,
  listarReglasMeritos,
  obtenerExpedienteMeritosAdmin,
  obtenerResultadoMeritosPostulante,
  publicarMeritos,
  type CalculoMeritosDto,
  type ExpedienteMeritosAdminDto,
  type PublicacionMeritosDto,
  type ResultadoMeritosPostulanteDto,
  type RevisionInput,
  type RevisionMeritosDto,
} from "./meritos"

describe("Cliente de Méritos (meritos.ts - Contrato T20/FT19)", () => {
  const fetchOriginal = globalThis.fetch

  afterEach(() => {
    globalThis.fetch = fetchOriginal
    vi.restoreAllMocks()
  })

  it("centipuntosAPuntos convierte a puntos con 2 decimales", () => {
    expect(centipuntosAPuntos(100)).toBe("1.00")
    expect(centipuntosAPuntos(4550)).toBe("45.50")
    expect(centipuntosAPuntos(0)).toBe("0.00")
  })

  it("obtenerResultadoMeritosPostulante consulta GET /merits/submissions/{id}", async () => {
    const mockRes: ResultadoMeritosPostulanteDto = {
      publicado: true,
      habilitadoFaseTecnica: true,
      resultado: {
        id: "pub-1",
        postulacionId: "sub-1",
        version: 1,
        calculoId: "calc-1",
        decision: "avanza",
        motivo: "Cumple con el umbral reglamentario",
        evaluacion: {
          totalCentipuntos: 4500,
          maximoCentipuntos: 5000,
          umbralCentipuntos: 3500,
          puedeAvanzar: true,
          desglose: [],
          impedimentos: [],
          desempate: [],
          desempateManual: false,
        },
        hashEnvio: "hash-env",
        reglasId: "rules-1",
        hashReglas: "hash-reg",
        responsableId: "admin-1",
        publicadaEn: "2026-10-31T20:00:00Z",
      },
    }

    globalThis.fetch = vi.fn().mockImplementation(() =>
      Promise.resolve(
        new Response(JSON.stringify(mockRes), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      ),
    )

    const res = await obtenerResultadoMeritosPostulante("sub-1")
    expect(res.publicado).toBe(true)
    expect(res.habilitadoFaseTecnica).toBe(true)
    expect(res.resultado?.decision).toBe("avanza")
    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/merits/submissions/sub-1"),
      expect.objectContaining({ method: "GET" }),
    )
  })

  it("obtenerExpedienteMeritosAdmin consulta GET /admin/merits/submissions/{id}", async () => {
    const mockCase: ExpedienteMeritosAdminDto = {
      reglasActivas: undefined,
      revision: undefined,
      calculo: undefined,
      publicacion: undefined,
    }

    globalThis.fetch = vi.fn().mockImplementation(() =>
      Promise.resolve(
        new Response(JSON.stringify(mockCase), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      ),
    )

    const res = await obtenerExpedienteMeritosAdmin("sub-1")
    expect(res).toBeDefined()
    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/admin/merits/submissions/sub-1"),
      expect.objectContaining({ method: "GET" }),
    )
  })

  it("crearRevisionMeritos envía POST /admin/merits/submissions/{id}/review", async () => {
    const input: RevisionInput = {
      versionAnterior: 0,
      reglasId: "rule-uuid",
      documentos: [],
      requisitos: [],
      inhabilidades: [],
      motivo: "Revisión técnica inicial",
    }

    const mockRevision: RevisionMeritosDto = {
      id: "rev-1",
      postulacionId: "sub-1",
      hashEnvio: "hash-env",
      version: 1,
      revision: input,
      responsableId: "admin-1",
      creadaEn: "2026-10-31T20:00:00Z",
    }

    globalThis.fetch = vi.fn().mockImplementation(() =>
      Promise.resolve(
        new Response(JSON.stringify(mockRevision), {
          status: 201,
          headers: { "Content-Type": "application/json" },
        }),
      ),
    )

    const res = await crearRevisionMeritos("sub-1", input)
    expect(res.id).toBe("rev-1")
    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/admin/merits/submissions/sub-1/review"),
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify(input),
      }),
    )
  })

  it("calcularMeritos envía POST /admin/merits/submissions/{id}/calculate", async () => {
    const mockCalculo: CalculoMeritosDto = {
      versionMotor: "meritos-v1",
      id: "calc-1",
      postulacionId: "sub-1",
      version: 1,
      revisionId: "rev-1",
      reglasId: "rules-1",
      hashReglas: "hr",
      hashEnvio: "he",
      evaluacion: {
        totalCentipuntos: 4000,
        maximoCentipuntos: 5000,
        umbralCentipuntos: 3500,
        puedeAvanzar: true,
        desglose: [],
        impedimentos: [],
        desempate: [],
        desempateManual: false,
      },
      motivo: "Cálculo oficial",
      responsableId: "admin-1",
      creadoEn: "2026-10-31T20:00:00Z",
    }

    globalThis.fetch = vi.fn().mockImplementation(() =>
      Promise.resolve(
        new Response(JSON.stringify(mockCalculo), {
          status: 201,
          headers: { "Content-Type": "application/json" },
        }),
      ),
    )

    const res = await calcularMeritos("sub-1", {
      revisionId: "rev-1",
      versionAnterior: 0,
      motivo: "Cálculo oficial",
    })
    expect(res.evaluacion.totalCentipuntos).toBe(4000)
    expect(res.versionMotor).toBe("meritos-v1")
  })

  it("publicarMeritos envía POST /admin/merits/submissions/{id}/publish", async () => {
    const mockPub: PublicacionMeritosDto = {
      id: "pub-1",
      postulacionId: "sub-1",
      version: 1,
      calculoId: "calc-1",
      decision: "avanza",
      motivo: "Aprobado por el comité",
      evaluacion: {
        totalCentipuntos: 4000,
        maximoCentipuntos: 5000,
        umbralCentipuntos: 3500,
        puedeAvanzar: true,
        desglose: [],
        impedimentos: [],
        desempate: [],
        desempateManual: false,
      },
      hashEnvio: "he",
      reglasId: "rules-1",
      hashReglas: "hr",
      responsableId: "admin-1",
      publicadaEn: "2026-10-31T20:00:00Z",
    }

    globalThis.fetch = vi.fn().mockImplementation(() =>
      Promise.resolve(
        new Response(JSON.stringify(mockPub), {
          status: 201,
          headers: { "Content-Type": "application/json" },
        }),
      ),
    )

    const res = await publicarMeritos("sub-1", {
      calculoId: "calc-1",
      versionAnterior: 0,
      decision: "avanza",
      motivo: "Aprobado por el comité",
    })
    expect(res.decision).toBe("avanza")
    expect(res.id).toBe("pub-1")
  })

  it("listarReglasMeritos consulta las versiones con cursor opaco", async () => {
    globalThis.fetch = vi.fn().mockImplementation(() =>
      Promise.resolve(
        new Response(
          JSON.stringify({
            items: [],
            hasMore: false,
            limit: 10,
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        ),
      ),
    )

    const res = await listarReglasMeritos("comp-1", "afiliados", {
      limit: 10,
      cursor: "cursor-opaco",
    })
    expect(res.items).toEqual([])
    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.stringContaining(
        "/admin/merits/competitions/comp-1/sectors/afiliados/rules",
      ),
      expect.objectContaining({ method: "GET" }),
    )
  })
})
