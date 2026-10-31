import { describe, expect, it, vi } from "vitest"
import { createElement } from "react"
import { renderToStaticMarkup } from "react-dom/server"
import { MemoryRouter, Route, Routes } from "react-router"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { PanelRevision } from "./panel-revision"
import { DashboardPostulacionDetalleView } from "@/pages/dashboard-postulacion-detalle-view"
import { clavesRevision, opcionesConteos, opcionesNomina } from "./queries"
import { busquedaRevision } from "./modelo"
import { ErrorApi } from "@/lib/api-error"
import type { FiltrosRevision } from "@/lib/clientes/revision"

vi.mock("@/features/auth/sesion-context", () => ({
  useSesion: () => ({ postulante: { id: "qa", rol: "analista" } }),
}))

const id = "01900000-0000-7000-8000-000000000002"
const filtros: FiltrosRevision = {
  periodo: "01900000-0000-7000-8000-000000000001",
  sector: "",
  estado: "",
}
const cuenta = "qa:analista"

function prepararCliente() {
  const cliente = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  cliente.setQueryData([...clavesRevision.cuenta(cuenta), "convocatorias"], {
    pages: [
      {
        items: [
          {
            id: filtros.periodo,
            codigo: "QA-2024",
            titulo: "Período de prueba",
            cerrada: true,
            apertura: "2024-01-01T00:00:00Z",
            cierre: "2024-03-01T00:00:00Z",
          },
        ],
        hasMore: false,
        limit: 50,
      },
    ],
    pageParams: [undefined],
  })
  cliente.setQueryData(opcionesConteos(cuenta, filtros).queryKey, {
    consultadoEn: "2024-03-01T15:00:00Z",
    actualizarEnSegundos: 30,
    conteos: [
      {
        sector: "afiliados",
        status: "submitted",
        postulantes: 7,
        postulaciones: 9,
      },
    ],
  })
  cliente.setQueryData(opcionesNomina(cuenta, filtros).queryKey, {
    pages: [
      {
        items: [
          {
            id,
            codigo: "QA-001",
            convocatoria: "QA-2024",
            hash: "a".repeat(64),
            titulos: [],
            experiencias: [],
            publicaciones: 0,
            horasCapacitacion: 0,
            nombres: "Persona sintética",
            apellidos: "Histórica DEMO",
            sector: "afiliados",
            status: "submitted",
            submittedAt: "2024-03-01T15:00:00Z",
          },
        ],
        hasMore: true,
        nextCursor: "opaco",
        limit: 20,
      },
    ],
    pageParams: [undefined],
  })
  return cliente
}

function renderizar(
  cliente: QueryClient,
  vista: "inicio" | "nomina" | "reportes",
  busqueda = busquedaRevision(filtros),
) {
  return renderToStaticMarkup(
    createElement(
      QueryClientProvider,
      { client: cliente },
      createElement(
        MemoryRouter,
        { initialEntries: [`/dashboard${busqueda}`] },
        createElement(PanelRevision, { vista }),
      ),
    ),
  )
}

describe("Presentación administrativa FT7 / FT8 (renderizado sin navegador)", () => {
  it("requiere selección explícita antes de mostrar conteos, nómina o exportación", () => {
    const cliente = prepararCliente()
    try {
      const html = renderizar(cliente, "nomina", "")
      expect(html).toContain("Seleccione una convocatoria")
      expect(html).not.toContain("Persona sintética")
      expect(html).not.toContain("Descargar Excel")
      expect(html).not.toContain("Última consulta:")
    } finally {
      cliente.clear()
    }
  })
  it("muestra conteos reales por sector y diferencia personas de expedientes", () => {
    const cliente = prepararCliente()
    try {
      const html = renderizar(cliente, "inicio")
      expect(html).toContain(">7</p>")
      expect(html).toContain("9")
      expect(html).toContain("expedientes sellados")
      expect(html).toContain("cada 30 s")
      expect(html).toContain("(cerrada)")
      expect(html).not.toContain("Código dactilar")
    } finally {
      cliente.clear()
    }
  })
  it("conserva filtros en enlaces al expediente y permite continuar la paginación", () => {
    const cliente = prepararCliente()
    try {
      const html = renderizar(cliente, "nomina")
      expect(html).toContain("Histórica DEMO")
      expect(html).toContain(
        `/dashboard/postulaciones/${id}?convocatoria=${filtros.periodo}`,
      )
      expect(html).toContain("Cargar más expedientes")
      expect(html).toContain("Enviado y sellado")
      expect(html).toContain("Descargar Excel")
      expect(html).toContain("Descargar PDF")
    } finally {
      cliente.clear()
    }
  })
  it("una selección sin filas muestra vacío, no una nómina de muestra", () => {
    const cliente = prepararCliente()
    cliente.setQueryData(opcionesNomina(cuenta, filtros).queryKey, {
      pages: [{ items: [], hasMore: false, limit: 20 }],
      pageParams: [undefined],
    })
    try {
      const html = renderizar(cliente, "nomina")
      expect(html).toContain("No hay expedientes en esta selección")
      expect(html).not.toContain("Histórica DEMO")
    } finally {
      cliente.clear()
    }
  })
  it("un fallo de conteos no se representa como cero ni como dato fresco", () => {
    const cliente = prepararCliente()
    cliente
      .getQueryCache()
      .find({ queryKey: opcionesConteos(cuenta, filtros).queryKey })
      ?.setState({
        status: "error",
        error: new ErrorApi("No autorizado", { estado: 403 }),
      })
    try {
      const html = renderizar(cliente, "inicio")
      expect(html).toContain("Conteos no disponibles")
      expect(html).not.toContain("Última consulta:")
      expect(html).not.toContain(">7</p>")
    } finally {
      cliente.clear()
    }
  })
  it("oculta filas anteriores si falla la autorización al cargar más", () => {
    const cliente = prepararCliente()
    cliente
      .getQueryCache()
      .find({ queryKey: opcionesNomina(cuenta, filtros).queryKey })
      ?.setState({
        status: "error",
        error: new ErrorApi("Permiso revocado", { estado: 403 }),
        fetchMeta: { fetchMore: { direction: "forward" } },
      })
    try {
      const html = renderizar(cliente, "nomina")
      expect(html).toContain("Permiso revocado")
      expect(html).not.toContain("Histórica DEMO")
    } finally {
      cliente.clear()
    }
  })
  it("rotula la propuesta técnica sin inventar méritos o elegibles", () => {
    const cliente = prepararCliente()
    try {
      const html = renderizar(cliente, "reportes")
      expect(html).toContain("EF13-TECNICA-v1")
      expect(html).toContain("pendiente de aprobación institucional")
      expect(html).toContain("No es una lista pública de elegibles")
      expect(html).toContain("10.000")
      expect(html).not.toMatch(/<input|Publicar resultados/)
    } finally {
      cliente.clear()
    }
  })
  it("un enlace inválido no monta consultas del expediente", () => {
    const cliente = prepararCliente()
    try {
      const html = renderToStaticMarkup(
        createElement(
          QueryClientProvider,
          { client: cliente },
          createElement(
            MemoryRouter,
            { initialEntries: ["/dashboard/postulaciones/invalido"] },
            createElement(
              Routes,
              null,
              createElement(Route, {
                path: "/dashboard/postulaciones/:id",
                element: createElement(DashboardPostulacionDetalleView),
              }),
            ),
          ),
        ),
      )
      expect(html).toContain("Enlace de expediente no válido")
      expect(
        cliente
          .getQueryCache()
          .findAll()
          .some((q) => q.queryKey.includes("expediente")),
      ).toBe(false)
    } finally {
      cliente.clear()
    }
  })
})
