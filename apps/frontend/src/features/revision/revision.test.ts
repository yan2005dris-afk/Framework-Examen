import { afterEach, describe, expect, it, vi } from "vitest"
import { createElement } from "react"
import { renderToStaticMarkup } from "react-dom/server"
import { QueryClient } from "@tanstack/react-query"
import { apiGetBlob, configurarManejadorNoAutorizado } from "@/lib/api"
import {
  exportarMatrizRevision,
  listarRevision,
  obtenerEnlaceDocumentoRevision,
  obtenerConteosRevision,
  parametrosRevision,
  TIPOS_MATRIZ,
  type FiltrosRevision,
} from "@/lib/clientes/revision"
import { opcionesConteos, opcionesNomina } from "./queries"
import {
  busquedaRevision,
  duracionHistorica,
  enlaceTemporalValido,
  fechaRevision,
  leerFiltrosRevision,
} from "./modelo"
import { ExpedienteSellado } from "./expediente-sellado"
import { ACCESO_POR_RUTA, puedeAcceder, RUTAS } from "@/lib/rutas"

const PERIODO = "01900000-0000-7000-8000-000000000001"
const filtros: FiltrosRevision = {
  periodo: PERIODO,
  sector: "afiliados",
  estado: "submitted",
}
const json = (valor: unknown, status = 200) =>
  new Response(JSON.stringify(valor), {
    status,
    headers: { "Content-Type": "application/json" },
  })

afterEach(() => {
  vi.unstubAllGlobals()
  configurarManejadorNoAutorizado(null)
  vi.useRealTimers()
})

describe("Filtros compartidos FT7 / FT8", () => {
  it("no selecciona una convocatoria ni legado por defecto", () => {
    expect(leerFiltrosRevision(new URLSearchParams()).filtros.periodo).toBe("")
    expect(() => parametrosRevision({ ...filtros, periodo: "" })).toThrow()
  })
  it("conserva selección al navegar, sin arrastrar cursores", () => {
    const params = new URLSearchParams(
      `${busquedaRevision(filtros)}&cursor=anterior`,
    )
    expect(leerFiltrosRevision(params)).toEqual({ filtros })
    expect(busquedaRevision(leerFiltrosRevision(params).filtros)).not.toContain(
      "cursor",
    )
  })
  it.each([
    "convocatoria=mal",
    "convocatoria=legado&convocatoria=otro",
    "sector=indefinido",
    "estado=draft",
    "estado=approved",
    "estado=under_review",
  ])("rechaza URL inválida %s", (busqueda) => {
    expect(
      leerFiltrosRevision(new URLSearchParams(busqueda)).error,
    ).toBeTruthy()
  })
  it("separa de forma explícita el legado", () => {
    expect(
      parametrosRevision({ periodo: "legado", sector: "", estado: "" }),
    ).toEqual({ sinConvocatoria: true, sector: undefined, status: undefined })
    expect(parametrosRevision(filtros)).toEqual({
      convocatoriaId: PERIODO,
      sector: "afiliados",
      status: "submitted",
    })
  })
  it("reportes hereda protección de analistas; una cuenta sin sesión no accede", () => {
    expect(ACCESO_POR_RUTA[RUTAS.dashboardReportes]).toBe("con-rol-analista")
    expect(puedeAcceder(ACCESO_POR_RUTA[RUTAS.dashboardReportes], null)).toBe(
      false,
    )
  })
})

describe("Cliente autorizado y descarga binaria", () => {
  it.each([
    {},
    { consultadoEn: "2024-01-01", conteos: null },
    {
      consultadoEn: "2024-01-01",
      conteos: [
        {
          sector: "afiliados",
          status: "submitted",
          postulantes: -1,
          postulaciones: 0,
        },
      ],
    },
  ])("no interpreta conteos incompletos como cero: %j", async (respuesta) => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(json(respuesta)))
    await expect(obtenerConteosRevision(filtros)).rejects.toMatchObject({
      codigo: "conteos_invalidos",
    })
  })
  it("acepta un conteo vacío real con fecha de consulta", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        json({
          consultadoEn: "2024-01-01T00:00:00Z",
          actualizarEnSegundos: 30,
          conteos: [],
        }),
      ),
    )
    await expect(obtenerConteosRevision(filtros)).resolves.toMatchObject({
      conteos: [],
    })
  })
  it("mantiene bytes, credenciales y señal; no pasa opciones internas a fetch", async () => {
    const bytes = new Uint8Array([80, 75, 0, 255, 128])
    const fetch = vi
      .fn()
      .mockResolvedValue(
        new Response(bytes, { headers: { "Content-Type": TIPOS_MATRIZ.xlsx } }),
      )
    vi.stubGlobal("fetch", fetch)
    const signal = new AbortController().signal
    const blob = await exportarMatrizRevision(filtros, "xlsx", signal)
    expect(new Uint8Array(await blob.arrayBuffer())).toEqual(bytes)
    const [ruta, opciones] = fetch.mock.calls[0]
    expect(ruta).toBe(
      `/api/v1/admin/submissions/export.xlsx?convocatoriaId=${PERIODO}&sector=afiliados&status=submitted`,
    )
    expect(opciones.credentials).toBe("include")
    expect(opciones.signal).toBe(signal)
    expect(opciones.headers.get("Accept")).toBe(TIPOS_MATRIZ.xlsx)
    expect(opciones).not.toHaveProperty("formatoRespuesta")
  })
  it("ambos archivos usan exactamente los mismos filtros, sin cursor ni limit", async () => {
    const fetch = vi.fn().mockImplementation((ruta: string) =>
      Promise.resolve(
        new Response("archivo", {
          headers: {
            "Content-Type": ruta.includes(".pdf")
              ? TIPOS_MATRIZ.pdf
              : TIPOS_MATRIZ.xlsx,
          },
        }),
      ),
    )
    vi.stubGlobal("fetch", fetch)
    await exportarMatrizRevision(filtros, "xlsx")
    await exportarMatrizRevision(filtros, "pdf")
    const urls = fetch.mock.calls.map(
      ([ruta]) => new URL(ruta, "https://local.test"),
    )
    expect(urls[0].search).toBe(urls[1].search)
    for (const url of urls) {
      expect(url.searchParams.has("cursor")).toBe(false)
      expect(url.searchParams.has("limit")).toBe(false)
    }
  })
  it.each([401, 403, 413, 500])(
    "no convierte el error HTTP %s en archivo",
    async (estado) => {
      const noAutorizado = vi.fn()
      configurarManejadorNoAutorizado(noAutorizado)
      vi.stubGlobal(
        "fetch",
        vi
          .fn()
          .mockResolvedValue(
            json(
              { error: { codigo: "rechazo", mensaje: "Operación rechazada" } },
              estado,
            ),
          ),
      )
      await expect(
        exportarMatrizRevision(filtros, "pdf"),
      ).rejects.toMatchObject({ codigo: "rechazo", estado })
      expect(noAutorizado).toHaveBeenCalledTimes(estado === 401 ? 1 : 0)
    },
  )
  it.each(["application/json", "text/html", "application/octet-stream"])(
    "rechaza el MIME %s aunque la respuesta sea 200",
    async (mime) => {
      vi.stubGlobal(
        "fetch",
        vi
          .fn()
          .mockResolvedValue(
            new Response("no es matriz", { headers: { "Content-Type": mime } }),
          ),
      )
      await expect(
        exportarMatrizRevision(filtros, "xlsx"),
      ).rejects.toMatchObject({ codigo: "archivo_invalido" })
    },
  )
  it("rechaza archivo vacío", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockResolvedValue(
          new Response("", { headers: { "Content-Type": TIPOS_MATRIZ.pdf } }),
        ),
    )
    await expect(exportarMatrizRevision(filtros, "pdf")).rejects.toMatchObject({
      codigo: "archivo_invalido",
    })
  })
  it("propaga cancelación sin presentarla como error de red", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockRejectedValue(new DOMException("Cancelado", "AbortError")),
    )
    await expect(
      apiGetBlob("/admin/submissions/export.pdf"),
    ).rejects.toMatchObject({ name: "AbortError" })
  })
  it("consulta nómina administrativa y codifica el cursor opaco", async () => {
    const fetch = vi
      .fn()
      .mockResolvedValue(json({ items: [], hasMore: false, limit: 20 }))
    vi.stubGlobal("fetch", fetch)
    await listarRevision(filtros, "opaco+/=")
    const url = new URL(fetch.mock.calls[0][0], "https://local.test")
    expect(url.pathname).toBe("/api/v1/admin/submissions")
    expect(url.searchParams.get("cursor")).toBe("opaco+/=")
    expect(url.searchParams.get("limit")).toBe("20")
  })
  it("pide una capacidad temporal al servidor, no el binario del documento", async () => {
    const fetch = vi.fn().mockResolvedValue(
      json({
        url: "https://storage.test/pdf",
        expiresAt: "2030-01-01T00:00:00Z",
      }),
    )
    vi.stubGlobal("fetch", fetch)
    await obtenerEnlaceDocumentoRevision(PERIODO)
    expect(fetch.mock.calls[0][0]).toBe(
      `/api/v1/admin/documents/${PERIODO}/download`,
    )
  })
})

describe("Privacidad, cursores y actualización", () => {
  it("aísla caché por cuenta, rol y filtros y no retiene datos al desmontar", () => {
    const base = opcionesNomina("cuenta:analista", filtros)
    expect(base.gcTime).toBe(0)
    expect(base.initialPageParam).toBeUndefined()
    expect(base.queryKey).not.toEqual(
      opcionesNomina("otra:analista", filtros).queryKey,
    )
    expect(base.queryKey).not.toEqual(
      opcionesNomina("cuenta:administrador", filtros).queryKey,
    )
    expect(base.queryKey).not.toEqual(
      opcionesNomina("cuenta:analista", { ...filtros, sector: "jubilados" })
        .queryKey,
    )
  })
  it("consulta conteos cada 30 s sin polling en segundo plano", () => {
    const opciones = opcionesConteos("cuenta:analista", filtros)
    expect(opciones.refetchInterval).toBe(30_000)
    expect(opciones.refetchIntervalInBackground).toBe(false)
    expect(opciones.refetchOnReconnect).toBe("always")
  })
  it("recorre páginas con cursor del servidor y empieza de nuevo al cambiar sector", async () => {
    const cliente = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    })
    const fetch = vi.fn().mockImplementation((ruta: string) => {
      const url = new URL(ruta, "https://local.test")
      const siguiente = url.searchParams.has("cursor")
      return Promise.resolve(
        json({
          items: [],
          hasMore: !siguiente,
          nextCursor: siguiente ? undefined : "cursor-opaco",
          limit: 20,
        }),
      )
    })
    vi.stubGlobal("fetch", fetch)
    try {
      const pagina = await cliente.fetchInfiniteQuery({
        ...opcionesNomina("cuenta:analista", filtros),
        pages: 2,
      })
      expect(pagina.pageParams).toEqual([undefined, "cursor-opaco"])
      await cliente.fetchInfiniteQuery(
        opcionesNomina("cuenta:analista", {
          ...filtros,
          sector: "empleadores",
        }),
      )
      expect(
        new URL(fetch.mock.calls[2][0], "https://local.test").searchParams.has(
          "cursor",
        ),
      ).toBe(false)
    } finally {
      cliente.clear()
    }
  })
})

describe("Histórico íntegro, sin datos actuales ni ejecución de contenido", () => {
  it("conserva duración almacenada con años, meses y días", () => {
    expect(
      duracionHistorica(
        {
          duracionCalculada: { anios: 5, meses: 2, dias: 0, texto: "antiguo" },
        },
        "2024-01-01T00:00:00Z",
      ),
    ).toBe("5 años, 2 meses, 0 días (al sello)")
  })
  it("corta una experiencia vigente en la fecha de envío de Ecuador, no hoy", () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date("2040-01-01T00:00:00Z"))
    expect(
      duracionHistorica(
        { fechaInicio: "2023-01-01", actualmenteLaborando: true },
        "2024-01-02T02:00:00Z",
      ),
    ).toBe("1 años, 0 meses, 0 días (calculada con fechas del envío; legado)")
  })
  it("no transforma fechas ausentes o invertidas en experiencia válida", () => {
    expect(duracionHistorica({}, "2024-01-01")).toBe("Duración no disponible")
    expect(
      duracionHistorica(
        { fechaInicio: "2024-01-01", fechaFin: "2020-01-01" },
        "2024-01-01",
      ),
    ).toBe("Duración no disponible")
  })
  it("renderiza nueve secciones, consentimientos guardados y campos legados sin editar", () => {
    const contenido: Record<string, unknown> = Object.fromEntries(
      Array.from({ length: 9 }, (_, i) => [
        `seccion${i + 1}`,
        { dato: `Sección histórica ${i + 1}` },
      ]),
    )
    contenido.seccion1 = {
      nombres: '<script>alert("x")</script>',
      campoLegado: "Conservado",
    }
    contenido.seccion6 = {
      experiencias: [
        {
          id: "exp-1",
          cargo: "Cargo histórico",
          duracionCalculada: { anios: 2, meses: 3, dias: 4 },
        },
      ],
    }
    contenido.consentimiento = {
      version: "anterior",
      puntos: ["Literal aceptado original"],
      declaracionVeracidad: true,
    }
    const html = renderToStaticMarkup(
      createElement(ExpedienteSellado, {
        contenido,
        enviadoEn: "2024-01-01T15:00:00Z",
      }),
    )
    expect(html.match(/<summary/g)).toHaveLength(10)
    for (const texto of [
      "Conservado",
      "Cargo histórico",
      "2 años, 3 meses, 4 días",
      "Literal aceptado original",
      "anterior",
    ])
      expect(html).toContain(texto)
    expect(html).toContain("&lt;script&gt;")
    expect(html).not.toContain("<script>")
    expect(html).not.toMatch(/<input|<textarea|<select/)
  })
  it("no carga una URL remota declarada como firma", () => {
    const html = renderToStaticMarkup(
      createElement(ExpedienteSellado, {
        contenido: {
          consentimiento: {
            firmaManuscritaDataUrl: "https://externo.test/identificador",
          },
        },
        enviadoEn: "2024-01-01",
      }),
    )
    expect(html).not.toContain("https://externo.test")
    expect(html).not.toContain("<img")
  })
  it("muestra la zona del concurso", () => {
    expect(fechaRevision("2024-01-02T02:00:00Z")).toContain("1 ene")
    expect(fechaRevision("no-fecha")).toBe("No disponible")
  })
  it.each([
    "javascript:alert(1)",
    "data:text/html,malicioso",
    "https://usuario:clave@storage.test/archivo",
    "/relativo",
  ])("rechaza capacidad insegura %s", (url) => {
    expect(
      enlaceTemporalValido({ url, expiresAt: "2030-01-01T00:00:00Z" }, 0),
    ).toBe(false)
  })
  it("requiere enlace HTTP(S) vigente", () => {
    expect(
      enlaceTemporalValido(
        {
          url: "https://storage.test/a?firma=temporal",
          expiresAt: "2030-01-01T00:00:00Z",
        },
        0,
      ),
    ).toBe(true)
    expect(
      enlaceTemporalValido(
        { url: "https://storage.test/a", expiresAt: "2020-01-01T00:00:00Z" },
        Date.parse("2024-01-01"),
      ),
    ).toBe(false)
  })
})
