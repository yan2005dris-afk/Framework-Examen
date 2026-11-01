import { afterEach, describe, expect, it, vi } from "vitest"
import {
  configurarVersionBanco,
  crearVersionBancoPreguntas,
  descargarReporteExamenPdf,
  finalizarExamen,
  habilitarPostulanteExamen,
  iniciarExamen,
  listarVersionesBancoPreguntas,
  obtenerEstadoExamen,
  obtenerPreguntaExamen,
  obtenerVersionBancoPreguntas,
  responderPreguntaExamen,
  type BancoPreguntasMetadataDto,
  type ConfiguracionExamenDto,
  type EntradaBancoPreguntasDto,
  type ReglasConfiguracionExamenDto,
  type VistaExamenDto,
} from "./preguntas"

describe("Cliente de Banco de Preguntas y Evaluación (preguntas.ts - FT24-FT27)", () => {
  const fetchOriginal = globalThis.fetch

  afterEach(() => {
    globalThis.fetch = fetchOriginal
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  const mockBanco: BancoPreguntasMetadataDto = {
    id: "bank-1",
    convocatoriaId: "conv-1",
    sector: "afiliados",
    version: 1,
    definicion: {
      versionAnterior: 0,
      referencia: "Resolución CD-001",
      preguntas: [
        {
          id: "q1",
          enunciado: "¿Cuál es el principio fundamental?",
          opciones: [
            { id: "a", texto: "Opción A" },
            { id: "b", texto: "Opción B" },
          ],
          correcta: "a",
        },
      ],
    },
    hash: "hash-bank-123",
    responsableId: "admin-1",
    creadaEn: "2026-10-31T20:00:00Z",
  }

  const mockConfig: ConfiguracionExamenDto = {
    id: "config-1",
    bancoId: "bank-1",
    convocatoriaId: "conv-1",
    sector: "afiliados",
    reglas: {
      cantidadPreguntas: 20,
      duracionSegundos: 3600,
      maximo: 100,
      umbral: 70,
      redondeo: "mas_cercano",
      politicaIntentos: "un_intento",
      politicaUnicidad: "orden_preguntas_unico_rechazar_agotamiento",
      referenciaAprobacion: "Acta de Consejo 2026",
      abreEn: "2026-11-01T08:00:00Z",
      cierraEn: "2026-11-01T20:00:00Z",
    },
    hash: "hash-config-123",
    creadaEn: "2026-10-31T20:30:00Z",
    responsableId: "admin-1",
  }

  const mockVista: VistaExamenDto = {
    id: "attempt-1",
    postulacionId: "sub-1",
    version: 1,
    estado: "active",
    iniciadaEn: "2026-11-01T08:05:00Z",
    venceEn: "2026-11-01T09:05:00Z",
    servidorEn: "2026-11-01T08:05:01Z",
    segundosRestantes: 3599,
    cantidadPreguntas: 20,
    respondidas: 0,
  }

  describe("crearVersionBancoPreguntas y listarVersionesBancoPreguntas (FT24)", () => {
    it("crea una nueva versión completa e inmutable del banco de reactivos", async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: new Headers({ "content-type": "application/json" }),
        json: async () => mockBanco,
      })

      const entrada: EntradaBancoPreguntasDto = {
        versionAnterior: 0,
        referencia: "Resolución CD-001",
        preguntas: mockBanco.definicion.preguntas,
      }

      const res = await crearVersionBancoPreguntas(
        "conv-1",
        "afiliados",
        entrada,
      )
      expect(res.version).toBe(1)
      expect(res.definicion.preguntas).toHaveLength(1)
      expect(globalThis.fetch).toHaveBeenCalledWith(
        expect.stringContaining(
          "/admin/question-bank/competitions/conv-1/sectors/afiliados/versions",
        ),
        expect.objectContaining({ method: "POST" }),
      )
    })

    it("lista versiones históricas del banco", async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: new Headers({ "content-type": "application/json" }),
        json: async () => ({
          items: [mockBanco],
          hasMore: false,
          limit: 10,
        }),
      })

      const res = await listarVersionesBancoPreguntas("conv-1", "afiliados")
      expect(res.items).toHaveLength(1)
      expect(res.items[0].hash).toBe("hash-bank-123")
    })

    it("obtiene el detalle y matriz auditada del banco por ID", async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: new Headers({ "content-type": "application/json" }),
        json: async () => mockBanco,
      })

      const res = await obtenerVersionBancoPreguntas("bank-1")
      expect(res.id).toBe("bank-1")
      expect(res.definicion.preguntas[0].correcta).toBe("a")
    })
  })

  describe("configurarVersionBanco y habilitarPostulanteExamen (FT25)", () => {
    it("congela la configuración aprobada de evaluación técnica", async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: new Headers({ "content-type": "application/json" }),
        json: async () => mockConfig,
      })

      const reglas: ReglasConfiguracionExamenDto = mockConfig.reglas
      const res = await configurarVersionBanco("bank-1", reglas)
      expect(res.bancoId).toBe("bank-1")
      expect(res.reglas.cantidadPreguntas).toBe(20)
      expect(globalThis.fetch).toHaveBeenCalledWith(
        expect.stringContaining(
          "/admin/question-bank/versions/bank-1/configurations",
        ),
        expect.objectContaining({ method: "POST" }),
      )
    })

    it("habilita individualmente a un postulante que superó méritos", async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: new Headers({ "content-type": "application/json" }),
        json: async () => ({ fase: "evaluacion_tecnica" }),
      })

      const res = await habilitarPostulanteExamen("sub-1", {
        configuracionId: "config-1",
        resultadoMeritosId: "merit-1",
      })

      expect(res.fase).toBe("evaluacion_tecnica")
      expect(globalThis.fetch).toHaveBeenCalledWith(
        expect.stringContaining(
          "/admin/question-bank/submissions/sub-1/enable",
        ),
        expect.objectContaining({ method: "POST" }),
      )
    })
  })

  describe("Ciclo del examen del postulante (FT25-FT27)", () => {
    it("obtiene el estado y reloj del examen activo", async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: new Headers({ "content-type": "application/json" }),
        json: async () => mockVista,
      })

      const res = await obtenerEstadoExamen("sub-1")
      expect(res.estado).toBe("active")
      expect(res.segundosRestantes).toBe(3599)
    })

    it("inicia o reanuda el examen oficial", async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: new Headers({ "content-type": "application/json" }),
        json: async () => mockVista,
      })

      const res = await iniciarExamen("sub-1")
      expect(res.id).toBe("attempt-1")
      expect(globalThis.fetch).toHaveBeenCalledWith(
        expect.stringContaining("/exams/submissions/sub-1/start"),
        expect.objectContaining({ method: "POST" }),
      )
    })

    it("obtiene una pregunta individual por índice", async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: new Headers({ "content-type": "application/json" }),
        json: async () => ({
          id: "q1",
          enunciado: "¿Cuál es el principio?",
          opciones: [
            { id: "a", texto: "A" },
            { id: "b", texto: "B" },
          ],
          indice: 1,
          versionIntento: 1,
        }),
      })

      const res = await obtenerPreguntaExamen("sub-1", 1)
      expect(res.indice).toBe(1)
      expect(res.opciones).toHaveLength(2)
      expect(globalThis.fetch).toHaveBeenCalledWith(
        expect.stringContaining("/exams/submissions/sub-1/questions/1"),
        expect.anything(),
      )
    })

    it("guarda una respuesta individual con control de versión", async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: new Headers({ "content-type": "application/json" }),
        json: async () => ({
          ...mockVista,
          version: 2,
          respondidas: 1,
        }),
      })

      const res = await responderPreguntaExamen("sub-1", {
        versionAnterior: 1,
        preguntaId: "q1",
        opcionId: "a",
      })

      expect(res.version).toBe(2)
      expect(res.respondidas).toBe(1)
      expect(globalThis.fetch).toHaveBeenCalledWith(
        expect.stringContaining("/exams/submissions/sub-1/answers"),
        expect.objectContaining({ method: "PUT" }),
      )
    })

    it("finaliza y califica en el servidor", async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: new Headers({ "content-type": "application/json" }),
        json: async () => ({
          ...mockVista,
          estado: "finished",
        }),
      })

      const res = await finalizarExamen("sub-1")
      expect(res.estado).toBe("finished")
    })

    it("descarga el reporte PDF del examen", async () => {
      const blobEsperado = new Blob(["%PDF-1.4"], { type: "application/pdf" })
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: new Headers({ "content-type": "application/pdf" }),
        blob: async () => blobEsperado,
      })

      const res = await descargarReporteExamenPdf("sub-1")
      expect(res.type).toBe("application/pdf")
    })
  })
})
