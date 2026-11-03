import { afterEach, describe, expect, it, vi } from "vitest"
import {
  atenderReconsideracion,
  listarReconsideracionesAdmin,
  listarReconsideracionesPostulante,
  obtenerReconsideracionAdmin,
  obtenerReconsideracionPostulante,
  presentarReconsideracion,
  resolverReconsideracion,
  type EntradaAtenderReconsideracionDto,
  type EntradaPresentarReconsideracionDto,
  type EntradaResolverReconsideracionDto,
  type ReconsideracionDto,
} from "./reconsideracion"

describe("Cliente de Reconsideraciones (reconsideracion.ts - FT21)", () => {
  const fetchOriginal = globalThis.fetch

  afterEach(() => {
    globalThis.fetch = fetchOriginal
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  const mockReconsideracion: ReconsideracionDto = {
    id: "rec-1",
    postulacionId: "sub-1",
    version: 1,
    estado: "recibida",
    resultadoOriginal: {
      id: "res-1",
      postulacionId: "sub-1",
      fase: "meritos",
      version: 1,
      fuenteId: "fue-1",
      reglasId: "reg-1",
      hashReglas: "hr",
      hashEnvio: "he",
      puntaje: 7000,
      maximo: 10000,
      habilitaAvance: false,
      motivo: "Puntaje insuficiente",
      responsableId: "adm-1",
      publicadaEn: "2026-10-31T12:00:00Z",
    },
    reglasPlazo: {
      id: "pol-1",
      convocatoriaId: "conv-1",
      sector: "afiliados",
      tipo: "reconsideracion",
      version: 1,
      definicion: {},
      hash: "hash-pol",
      aprobada: true,
      responsableId: "adm-1",
      creadaEn: "2026-10-31T00:00:00Z",
    },
    motivo: "Se omitió valorar la publicación académica indexada.",
    documentosOriginales: ["doc-1"],
    responsableId: "post-1",
    presentadaEn: "2026-10-31T14:00:00Z",
    cierreVentana: "2026-11-05T18:00:00Z",
    limitePresentacion: "2026-11-02T18:00:00Z",
    limiteTalentoHumano: "2026-11-03T18:00:00Z",
  }

  describe("presentarReconsideracion", () => {
    it("envía solicitud con Idempotency-Key a /reconsiderations/submissions/:id", async () => {
      const entrada: EntradaPresentarReconsideracionDto = {
        fase: "meritos",
        resultadoId: "res-1",
        motivo: "Se omitió valorar la publicación académica indexada.",
        documentosOriginales: ["doc-1"],
      }

      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 201,
        headers: new Headers({ "content-type": "application/json" }),
        json: async () => mockReconsideracion,
      })

      const respuesta = await presentarReconsideracion(
        "sub-1",
        entrada,
        "uuid-idemp-rec",
      )
      expect(respuesta).toEqual(mockReconsideracion)

      const llamada = vi.mocked(globalThis.fetch).mock.calls[0]
      expect(llamada[0]).toContain("/reconsiderations/submissions/sub-1")
      const opciones = llamada[1] as RequestInit
      expect(opciones.method).toBe("POST")
      const cabeceras = new Headers(opciones.headers)
      expect(cabeceras.get("Idempotency-Key")).toBe("uuid-idemp-rec")
    })
  })

  describe("listarReconsideracionesPostulante y obtenerReconsideracionPostulante", () => {
    it("obtiene listado paginado y detalle del postulante", async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: new Headers({ "content-type": "application/json" }),
        json: async () => ({
          items: [mockReconsideracion],
          hasMore: false,
          limit: 10,
        }),
      })

      const listado = await listarReconsideracionesPostulante("sub-1", {
        limit: 10,
      })
      expect(listado.items).toHaveLength(1)

      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: new Headers({ "content-type": "application/json" }),
        json: async () => mockReconsideracion,
      })

      const detalle = await obtenerReconsideracionPostulante("sub-1", "rec-1")
      expect(detalle.id).toBe("rec-1")
    })
  })

  describe("listarReconsideracionesAdmin y obtenerReconsideracionAdmin", () => {
    it("consume endpoints bajo /admin/reconsiderations/", async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: new Headers({ "content-type": "application/json" }),
        json: async () => ({
          items: [mockReconsideracion],
          hasMore: false,
          limit: 10,
        }),
      })

      await listarReconsideracionesAdmin("sub-1")
      expect(globalThis.fetch).toHaveBeenCalledWith(
        expect.stringContaining("/admin/reconsiderations/submissions/sub-1"),
        expect.anything(),
      )

      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: new Headers({ "content-type": "application/json" }),
        json: async () => mockReconsideracion,
      })

      await obtenerReconsideracionAdmin("sub-1", "rec-1")
      expect(globalThis.fetch).toHaveBeenCalledWith(
        expect.stringContaining(
          "/admin/reconsiderations/submissions/sub-1/rec-1",
        ),
        expect.anything(),
      )
    })
  })

  describe("atenderReconsideracion", () => {
    it("envía atención de Analista a /attend con Idempotency-Key", async () => {
      const entrada: EntradaAtenderReconsideracionDto = {
        versionAnterior: 1,
        observacion: "Se revisó el documento y corresponde asignar puntaje.",
        recomendacion: "rectificar",
      }

      const mockAtendida: ReconsideracionDto = {
        ...mockReconsideracion,
        version: 2,
        estado: "atendida",
        atencion: {
          responsableId: "ana-1",
          fecha: "2026-10-31T15:00:00Z",
          observacion: entrada.observacion,
          recomendacion: entrada.recomendacion,
        },
        limiteComision: "2026-11-04T18:00:00Z",
      }

      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: new Headers({ "content-type": "application/json" }),
        json: async () => mockAtendida,
      })

      const res = await atenderReconsideracion(
        "sub-1",
        "rec-1",
        entrada,
        "uuid-idemp-attend",
      )
      expect(res.estado).toBe("atendida")
      expect(res.atencion?.recomendacion).toBe("rectificar")

      expect(globalThis.fetch).toHaveBeenCalledWith(
        expect.stringContaining(
          "/admin/reconsiderations/submissions/sub-1/rec-1/attend",
        ),
        expect.anything(),
      )
    })
  })

  describe("resolverReconsideracion", () => {
    it("envía resolución de Administrador a /resolve con Idempotency-Key", async () => {
      const entrada: EntradaResolverReconsideracionDto = {
        versionAnterior: 2,
        decision: "rectificar",
        motivo: "La Comisión acoge la recomendación de Talento Humano.",
        resultadoRectificadoId: "res-2",
      }

      const mockResuelta: ReconsideracionDto = {
        ...mockReconsideracion,
        version: 3,
        estado: "resuelta",
        resolucion: {
          responsableId: "adm-1",
          fecha: "2026-10-31T16:00:00Z",
          decision: "rectificar",
          motivo: entrada.motivo,
          resultadoVigente: {
            ...mockReconsideracion.resultadoOriginal,
            id: "res-2",
            version: 2,
            puntaje: 8500,
            habilitaAvance: true,
          },
          diferenciaPuntaje: 1500,
        },
      }

      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: new Headers({ "content-type": "application/json" }),
        json: async () => mockResuelta,
      })

      const res = await resolverReconsideracion(
        "sub-1",
        "rec-1",
        entrada,
        "uuid-idemp-resolve",
      )
      expect(res.estado).toBe("resuelta")
      expect(res.resolucion?.decision).toBe("rectificar")
      expect(res.resolucion?.diferenciaPuntaje).toBe(1500)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        expect.stringContaining(
          "/admin/reconsiderations/submissions/sub-1/rec-1/resolve",
        ),
        expect.anything(),
      )
    })
  })
})
