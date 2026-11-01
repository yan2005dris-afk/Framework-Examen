import { afterEach, describe, expect, it, vi } from "vitest"
import {
  abrirDocumentoSeguro,
  calcularCuotaUtilizada,
  eliminarDocumento,
  formatearTamanoBytes,
  limitePorSeccion,
  listarDocumentos,
  obtenerEnlaceDescarga,
  subirDocumento,
  LIMITE_ORGANIZACION_BYTES,
  LIMITE_REGISTRO_BYTES,
  LIMITE_TOTAL_POSTULANTE_BYTES,
  type DocumentoDto,
} from "./documentos"

describe("Cliente de Documentos y Cuotas (documentos.ts - Contrato T4/FT18)", () => {
  const fetchOriginal = globalThis.fetch

  afterEach(() => {
    globalThis.fetch = fetchOriginal
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  it("calcula límites por sección correctamente", () => {
    expect(LIMITE_TOTAL_POSTULANTE_BYTES).toBe(50 * 1024 * 1024) // 50 MiB
    expect(limitePorSeccion(4)).toBe(LIMITE_ORGANIZACION_BYTES) // 4 MiB
    expect(limitePorSeccion(5)).toBe(LIMITE_REGISTRO_BYTES) // 2 MiB
    expect(limitePorSeccion(6)).toBe(LIMITE_REGISTRO_BYTES)
    expect(limitePorSeccion(7)).toBe(LIMITE_REGISTRO_BYTES)
    expect(limitePorSeccion(8)).toBe(LIMITE_REGISTRO_BYTES)
    expect(() => limitePorSeccion(1)).toThrow()
    expect(() => limitePorSeccion(9)).toThrow()
  })

  it("formatea tamaños en KiB y MiB", () => {
    expect(formatearTamanoBytes(512 * 1024)).toBe("512.0 KiB")
    expect(formatearTamanoBytes(2 * 1024 * 1024)).toBe("2.0 MiB")
  })

  it("calcula la cuota utilizada sumando documentos", () => {
    const docs: DocumentoDto[] = [
      {
        id: "doc-1",
        submissionId: "sub-1",
        section: 5,
        recordId: "rec-1",
        name: "titulo.pdf",
        mime: "application/pdf",
        sizeBytes: 1024 * 1024,
        sha256: "abc",
        createdAt: "2026-10-31T00:00:00Z",
      },
      {
        id: "doc-2",
        submissionId: "sub-1",
        section: 6,
        recordId: "rec-2",
        name: "experiencia.pdf",
        mime: "application/pdf",
        sizeBytes: 2 * 1024 * 1024,
        sha256: "def",
        createdAt: "2026-10-31T00:00:00Z",
      },
    ]

    const cuota = calcularCuotaUtilizada(docs)
    expect(cuota.bytes).toBe(3 * 1024 * 1024)
    expect(cuota.porcentaje).toBe(6) // 3 / 50 = 6%
    expect(cuota.formateado).toContain("3.0 MiB de 50.0 MiB")
    expect(cuota.excedida).toBe(false)
  })

  it("subirDocumento envía FormData con section, recordId y file", async () => {
    const mockDoc: DocumentoDto = {
      id: "doc-123",
      submissionId: "sub-456",
      section: 5,
      recordId: "tit-1",
      name: "diploma.pdf",
      mime: "application/pdf",
      sizeBytes: 1000,
      sha256: "hash-sha256",
      createdAt: "2026-10-31T12:00:00Z",
    }

    const mockFetch = vi.fn().mockImplementation(() =>
      Promise.resolve(
        new Response(JSON.stringify(mockDoc), {
          status: 201,
          headers: { "Content-Type": "application/json" },
        }),
      ),
    )
    globalThis.fetch = mockFetch

    const archivo = new File(["%PDF-1.7 contenido"], "diploma.pdf", {
      type: "application/pdf",
    })
    const res = await subirDocumento("sub-456", 5, archivo, "tit-1")

    expect(res).toEqual(mockDoc)
    expect(mockFetch).toHaveBeenCalledTimes(1)
    const [url, init] = mockFetch.mock.calls[0]
    expect(String(url)).toContain("/submissions/sub-456/documents")
    expect(init?.method).toBe("POST")
    expect(init?.body).toBeInstanceOf(FormData)
    const formData = init?.body as FormData
    expect(formData.get("section")).toBe("5")
    expect(formData.get("recordId")).toBe("tit-1")
    expect((formData.get("file") as File).name).toBe(archivo.name)
    expect((formData.get("file") as File).size).toBe(archivo.size)
  })

  it("listarDocumentos llama GET /submissions/{id}/documents", async () => {
    const mockDocs: DocumentoDto[] = []
    globalThis.fetch = vi.fn().mockImplementation(() =>
      Promise.resolve(
        new Response(JSON.stringify(mockDocs), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      ),
    )

    const res = await listarDocumentos("sub-789")
    expect(res).toEqual([])
    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/submissions/sub-789/documents"),
      expect.objectContaining({ method: "GET" }),
    )
  })

  it("eliminarDocumento llama DELETE /documents/{id}", async () => {
    globalThis.fetch = vi
      .fn()
      .mockImplementation(() =>
        Promise.resolve(new Response(null, { status: 204 })),
      )

    await eliminarDocumento("doc-del-1")
    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/documents/doc-del-1"),
      expect.objectContaining({ method: "DELETE" }),
    )
  })

  it("obtenerEnlaceDescarga llama GET /documents/{id}/download", async () => {
    globalThis.fetch = vi.fn().mockImplementation(() =>
      Promise.resolve(
        new Response(
          JSON.stringify({
            url: "https://storage.local/doc.pdf?signature=abc",
            expiresAt: "2026-10-31T20:00:00Z",
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        ),
      ),
    )

    const res = await obtenerEnlaceDescarga("doc-dl-1")
    expect(res.url).toBe("https://storage.local/doc.pdf?signature=abc")
    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/documents/doc-dl-1/download"),
      expect.objectContaining({ method: "GET" }),
    )
  })

  it("abrirDocumentoSeguro solicita enlace y abre en ventana con noopener", async () => {
    globalThis.fetch = vi.fn().mockImplementation(() =>
      Promise.resolve(
        new Response(
          JSON.stringify({
            url: "https://storage.local/signed-url",
            expiresAt: "2026-10-31T20:00:00Z",
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        ),
      ),
    )

    const openSpy = vi.fn()
    vi.stubGlobal("window", { open: openSpy })

    await abrirDocumentoSeguro("doc-safe-1")
    expect(openSpy).toHaveBeenCalledWith(
      "https://storage.local/signed-url",
      "_blank",
      "noopener,noreferrer",
    )
  })
})
