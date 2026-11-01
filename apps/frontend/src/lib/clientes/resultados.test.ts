import { afterEach, describe, expect, it, vi } from "vitest"
import {
  descargarCsvListaAdmin,
  generarListaResultados,
  listarListasAdmin,
  listarPublicacionesListasAdmin,
  obtenerEntradasListaAdmin,
  obtenerListaMetadataAdmin,
  obtenerResultadosPublicosVigentes,
  publicarListaAdmin,
  type EntradaGenerarListaDto,
  type EntradaPublicarListaDto,
  type ListaResultadosMetadataDto,
  type PublicacionListaMetadataDto,
} from "./resultados"

describe("Cliente de Resultados y Listas de Elegibles (resultados.ts - FT23)", () => {
  const fetchOriginal = globalThis.fetch

  afterEach(() => {
    globalThis.fetch = fetchOriginal
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  const mockLista: ListaResultadosMetadataDto = {
    id: "list-1",
    convocatoriaId: "conv-1",
    sector: "afiliados",
    version: 1,
    reglas: {
      orden: ["resultados", "meritos"],
      referenciaOrden: "Acta 001-2026",
    },
    hashFuentes: "hash-fuentes-1",
    responsableId: "adm-1",
    creadaEn: "2026-10-31T20:00:00Z",
  }

  const mockPublicacion: PublicacionListaMetadataDto = {
    id: "pub-1",
    listaId: "list-1",
    convocatoriaId: "conv-1",
    sector: "afiliados",
    version: 1,
    referenciaAprobacion: "Resolución CESD-2026-005",
    publica: true,
    referenciaPrivacidad: "Política aprobada T28",
    responsableId: "adm-1",
    publicadaEn: "2026-10-31T20:30:00Z",
  }

  describe("generarListaResultados", () => {
    it("envía solicitud con reglas y cabecera Idempotency-Key", async () => {
      const entrada: EntradaGenerarListaDto = {
        reglas: {
          orden: ["resultados", "meritos"],
          referenciaOrden: "Acta 001-2026",
        },
      }

      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: new Headers({ "content-type": "application/json" }),
        json: async () => mockLista,
      })

      const res = await generarListaResultados(
        "conv-1",
        "afiliados",
        entrada,
        "uuid-idemp-list",
      )
      expect(res).toEqual(mockLista)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        expect.stringContaining(
          "/admin/results/competitions/conv-1/sectors/afiliados/lists",
        ),
        expect.objectContaining({
          method: "POST",
          headers: expect.anything(),
        }),
      )
    })
  })

  describe("listarListasAdmin, listarPublicacionesListasAdmin y obtenerListaMetadataAdmin", () => {
    it("consume endpoints de consulta interna", async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: new Headers({ "content-type": "application/json" }),
        json: async () => ({
          items: [mockLista],
          hasMore: false,
          limit: 10,
        }),
      })

      const listas = await listarListasAdmin("conv-1", "afiliados")
      expect(listas.items).toHaveLength(1)

      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: new Headers({ "content-type": "application/json" }),
        json: async () => ({
          items: [mockPublicacion],
          hasMore: false,
          limit: 10,
        }),
      })

      const pubs = await listarPublicacionesListasAdmin("conv-1", "afiliados")
      expect(pubs.items).toHaveLength(1)

      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: new Headers({ "content-type": "application/json" }),
        json: async () => mockLista,
      })

      const meta = await obtenerListaMetadataAdmin("list-1")
      expect(meta.id).toBe("list-1")
    })
  })

  describe("obtenerEntradasListaAdmin y descargarCsvListaAdmin", () => {
    it("obtiene entradas y descarga binaria CSV", async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: new Headers({ "content-type": "application/json" }),
        json: async () => ({
          items: [
            {
              postulacionId: "sub-1",
              codigo: "EF12-2026-004821",
              hashEnvio: "h1",
              nombres: "Carlos Alberto",
              apellidos: "Mendoza",
              elegible: true,
              puesto: 1,
              empate: false,
              consentimientoNombre: true,
              resultados: {},
            },
          ],
          hasMore: false,
          limit: 10,
        }),
      })

      const entradas = await obtenerEntradasListaAdmin("list-1")
      expect(entradas.items).toHaveLength(1)
      expect(entradas.items[0].elegible).toBe(true)

      const blobMock = new Blob(["version,puesto,nombres\n1,1,Carlos"], {
        type: "text/csv",
      })
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: new Headers({ "content-type": "text/csv" }),
        blob: async () => blobMock,
      })

      const blobRes = await descargarCsvListaAdmin("list-1")
      expect(blobRes).toBeDefined()
    })
  })

  describe("publicarListaAdmin", () => {
    it("envía solicitud de publicación oficial con Idempotency-Key", async () => {
      const entrada: EntradaPublicarListaDto = {
        versionPublicadaAnterior: 0,
        referenciaAprobacion: "Resolución CESD-2026-005",
        publica: true,
        referenciaPrivacidad: "Política aprobada T28",
      }

      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: new Headers({ "content-type": "application/json" }),
        json: async () => mockPublicacion,
      })

      const res = await publicarListaAdmin("list-1", entrada)
      expect(res.publica).toBe(true)
      expect(globalThis.fetch).toHaveBeenCalledWith(
        expect.stringContaining("/admin/results/lists/list-1/publish"),
        expect.objectContaining({ method: "POST" }),
      )
    })
  })

  describe("obtenerResultadosPublicosVigentes", () => {
    it("consume ruta pública sin sesión con nombres en orden alfabético", async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: new Headers({ "content-type": "application/json" }),
        json: async () => ({
          id: "pub-1",
          convocatoriaId: "conv-1",
          sector: "afiliados",
          version: 1,
          publicadaEn: "2026-10-31T20:30:00Z",
          orden: "alfabetico_nombres_sin_puntajes",
          nombres: {
            items: ["Mendoza Carlos Alberto"],
            hasMore: false,
            limit: 20,
          },
        }),
      })

      const res = await obtenerResultadosPublicosVigentes("conv-1", "afiliados")
      expect(res.orden).toBe("alfabetico_nombres_sin_puntajes")
      expect(res.nombres.items).toContain("Mendoza Carlos Alberto")
      expect(globalThis.fetch).toHaveBeenCalledWith(
        expect.stringContaining(
          "/public/results/competitions/conv-1/sectors/afiliados",
        ),
        expect.anything(),
      )
    })
  })
})
