import { afterEach, describe, expect, it, vi } from "vitest"
import {
  consolidarExpediente,
  descargarEvidenciaAdmin,
  descargarEvidenciaPostulante,
  obtenerEvaluacionPostulante,
  obtenerExpedienteGradesAdmin,
  obtenerHistorialEvaluacionPostulante,
  obtenerHistorialGradesAdmin,
  publicarPsicotecnica,
  registrarPsicotecnica,
  validarYConvertirPdfBase64,
  LIMITE_EVIDENCIA_BYTES,
  type CasoEvaluacionDto,
  type EntradaConsolidacionDto,
  type EntradaPsicotecnicaDto,
  type EntradaPublicarPsicotecnicaDto,
  type RegistroPsicotecnicoDto,
  type ResultadoPublicadoDto,
} from "./evaluacion"

describe("Cliente de Evaluación y Oposición (evaluacion.ts - FT20)", () => {
  const fetchOriginal = globalThis.fetch

  afterEach(() => {
    globalThis.fetch = fetchOriginal
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  describe("validarYConvertirPdfBase64", () => {
    it("valida límite de 2 MiB", async () => {
      expect(LIMITE_EVIDENCIA_BYTES).toBe(2 * 1024 * 1024)
      const archivoGrande = new File(
        [new Uint8Array(LIMITE_EVIDENCIA_BYTES + 10)],
        "grande.pdf",
        { type: "application/pdf" },
      )
      await expect(validarYConvertirPdfBase64(archivoGrande)).rejects.toThrow(
        /supera el límite de 2 MiB/i,
      )
    })

    it("valida que el archivo sea PDF", async () => {
      const archivoTxt = new File(["hola"], "archivo.txt", {
        type: "text/plain",
      })
      await expect(validarYConvertirPdfBase64(archivoTxt)).rejects.toThrow(
        /solo se admiten documentos en formato pdf/i,
      )
    })

    it("convierte un PDF a base64 puro sin cabecera data URL", async () => {
      const contenido = "Contenido de prueba"
      const archivo = new File([contenido], "informe.pdf", {
        type: "application/pdf",
      })

      const resultado = await validarYConvertirPdfBase64(archivo)
      expect(resultado.nombre).toBe("informe.pdf")
      expect(resultado.pdfBase64).toBeDefined()
      expect(resultado.pdfBase64).not.toContain("data:application/pdf;base64,")
    })
  })

  describe("obtenerEvaluacionPostulante", () => {
    it("llama a GET /evaluations/submissions/:id", async () => {
      const mockCaso: CasoEvaluacionDto = {
        sector: "afiliados",
        expediente: {
          resultados: {
            meritos: {
              id: "res-1",
              postulacionId: "sub-1",
              fase: "meritos",
              version: 1,
              fuenteId: "fue-1",
              reglasId: "reg-1",
              hashReglas: "hash1",
              hashEnvio: "hash2",
              puntaje: 8500,
              maximo: 10000,
              habilitaAvance: true,
              motivo: "Aprobado",
              responsableId: "adm-1",
              publicadaEn: "2026-10-31T20:00:00Z",
            },
          },
        },
        fuentesPublicadas: {},
        consultadoEn: "2026-10-31T20:05:00Z",
      }

      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: new Headers({ "content-type": "application/json" }),
        json: async () => mockCaso,
      })

      const respuesta = await obtenerEvaluacionPostulante("sub-1")
      expect(respuesta).toEqual(mockCaso)
      expect(globalThis.fetch).toHaveBeenCalledWith(
        expect.stringContaining("/evaluations/submissions/sub-1"),
        expect.objectContaining({ method: "GET" }),
      )
    })
  })

  describe("obtenerHistorialEvaluacionPostulante", () => {
    it("envía parámetros de cursor y limit", async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: new Headers({ "content-type": "application/json" }),
        json: async () => ({
          items: [
            {
              id: "hist-1",
              accion: "meritos.publicada",
              fecha: "2026-10-31T00:00:00Z",
            },
          ],
          hasMore: false,
          limit: 10,
        }),
      })

      const respuesta = await obtenerHistorialEvaluacionPostulante("sub-1", {
        limit: 10,
        cursor: "pos-1",
      })
      expect(respuesta.items).toHaveLength(1)
      expect(globalThis.fetch).toHaveBeenCalledWith(
        expect.stringContaining("cursor=pos-1"),
        expect.anything(),
      )
    })
  })

  describe("descargarEvidenciaPostulante y descargarEvidenciaAdmin", () => {
    it("obtiene URL temporal de descarga", async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: new Headers({ "content-type": "application/json" }),
        json: async () => ({
          url: "https://storage.local/evidencia.pdf?signature=xyz",
          expiraEn: "2026-10-31T20:10:00Z",
        }),
      })

      const respuesta = await descargarEvidenciaPostulante("sub-1", "evi-1")
      expect(respuesta.url).toContain("storage.local")

      const respuestaAdmin = await descargarEvidenciaAdmin("sub-1", "evi-1")
      expect(respuestaAdmin.url).toContain("storage.local")
    })
  })

  describe("obtenerExpedienteGradesAdmin y obtenerHistorialGradesAdmin", () => {
    it("consume las rutas auditadas bajo /admin/grades/", async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: new Headers({ "content-type": "application/json" }),
        json: async () => ({
          sector: "afiliados",
          expediente: { resultados: {} },
          fuentesPublicadas: {},
          consultadoEn: "2026-10-31T20:00:00Z",
        }),
      })

      await obtenerExpedienteGradesAdmin("sub-1")
      expect(globalThis.fetch).toHaveBeenCalledWith(
        expect.stringContaining("/admin/grades/submissions/sub-1"),
        expect.objectContaining({ method: "GET" }),
      )

      await obtenerHistorialGradesAdmin("sub-1")
      expect(globalThis.fetch).toHaveBeenCalledWith(
        expect.stringContaining("/admin/grades/submissions/sub-1/history"),
        expect.objectContaining({ method: "GET" }),
      )
    })
  })

  describe("registrarPsicotecnica", () => {
    it("envía payload y cabecera Idempotency-Key", async () => {
      const entrada: EntradaPsicotecnicaDto = {
        versionAnterior: 0,
        reglasId: "reg-1",
        evaluadaEn: "2026-10-31T18:00:00Z",
        puntaje: 450,
        conclusion: "Apto con observaciones",
        motivo: "Evaluación psicométrica completada",
        respaldo: {
          nombre: "informe.pdf",
          pdfBase64: "JVBERi0xLjQK...",
        },
      }

      const mockRespuesta: RegistroPsicotecnicoDto = {
        id: "psi-1",
        postulacionId: "sub-1",
        version: 1,
        reglasId: "reg-1",
        hashReglas: "h1",
        hashEnvio: "h2",
        instrumento: "Test-1",
        versionInstrumento: "v1",
        puntaje: 450,
        maximo: 500,
        conclusion: "Apto con observaciones",
        evaluadaEn: "2026-10-31T18:00:00Z",
        respaldo: {
          id: "evi-1",
          postulacionId: "sub-1",
          responsableId: "adm-1",
          nombre: "informe.pdf",
          tamano: 1024,
          sha256: "sha-1",
          creadaEn: "2026-10-31T18:05:00Z",
        },
        motivo: "Evaluación psicométrica completada",
        responsableId: "adm-1",
        registradaEn: "2026-10-31T18:05:00Z",
      }

      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 201,
        headers: new Headers({ "content-type": "application/json" }),
        json: async () => mockRespuesta,
      })

      const res = await registrarPsicotecnica("sub-1", entrada, "uuid-idemp-1")
      expect(res).toEqual(mockRespuesta)

      const llamada = vi.mocked(globalThis.fetch).mock.calls[0]
      expect(llamada[0]).toContain(
        "/admin/grades/submissions/sub-1/psychometric",
      )
      const opciones = llamada[1] as RequestInit
      expect(opciones.method).toBe("POST")
      const cabeceras = new Headers(opciones.headers)
      expect(cabeceras.get("Idempotency-Key")).toBe("uuid-idemp-1")
    })
  })

  describe("publicarPsicotecnica", () => {
    it("publica versión oficial con Idempotency-Key", async () => {
      const entrada: EntradaPublicarPsicotecnicaDto = {
        versionAnterior: 0,
        fuenteId: "psi-1",
        reglasId: "reg-1",
        motivo: "Publicación oficial de psicotécnica",
      }

      const mockResultado: ResultadoPublicadoDto = {
        id: "res-psi-1",
        postulacionId: "sub-1",
        fase: "oposicion_psicotecnica",
        version: 1,
        fuenteId: "psi-1",
        reglasId: "reg-1",
        hashReglas: "hr",
        hashEnvio: "he",
        puntaje: 450,
        maximo: 500,
        habilitaAvance: true,
        motivo: "Publicación oficial",
        responsableId: "adm-1",
        publicadaEn: "2026-10-31T19:00:00Z",
      }

      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 201,
        headers: new Headers({ "content-type": "application/json" }),
        json: async () => mockResultado,
      })

      const res = await publicarPsicotecnica("sub-1", entrada, "uuid-idemp-pub")
      expect(res).toEqual(mockResultado)
      expect(globalThis.fetch).toHaveBeenCalledWith(
        expect.stringContaining(
          "/admin/grades/submissions/sub-1/psychometric/publish",
        ),
        expect.anything(),
      )
    })
  })

  describe("consolidarExpediente", () => {
    it("envía fuentes de méritos, teórica y psicotécnica", async () => {
      const entrada: EntradaConsolidacionDto = {
        versionAnterior: 0,
        reglasId: "reg-1",
        motivo: "Consolidación final de calificaciones",
        fuentesEsperadas: {
          meritos: "res-mer-1",
          oposicion_teorica: "res-teo-1",
          oposicion_psicotecnica: "res-psi-1",
        },
      }

      const mockConsolidado: ResultadoPublicadoDto = {
        id: "res-final-1",
        postulacionId: "sub-1",
        fase: "resultados",
        version: 1,
        fuenteId: "00000000-0000-0000-0000-000000000000",
        reglasId: "reg-1",
        hashReglas: "hr",
        hashEnvio: "he",
        puntaje: 8850,
        maximo: 10000,
        habilitaAvance: true,
        motivo: "Consolidación final",
        responsableId: "adm-1",
        publicadaEn: "2026-10-31T20:00:00Z",
        desglose: [
          {
            fase: "meritos",
            resultadoId: "res-mer-1",
            puntajeOrigen: 9000,
            maximoOrigen: 10000,
            pesoPuntosBase: 4000,
            aporte: 3600,
          },
        ],
      }

      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 201,
        headers: new Headers({ "content-type": "application/json" }),
        json: async () => mockConsolidado,
      })

      const res = await consolidarExpediente("sub-1", entrada)
      expect(res.fase).toBe("resultados")
      expect(res.puntaje).toBe(8850)
    })
  })
})
