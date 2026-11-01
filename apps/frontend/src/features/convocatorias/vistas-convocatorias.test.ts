import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { createElement, type ReactElement } from "react"
import { renderToStaticMarkup } from "react-dom/server"
import { MemoryRouter } from "react-router"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { PortadaView } from "@/pages/portada-view"
import { AdminConvocatoriasView } from "@/pages/admin-convocatorias-view"
import { AdminFasesView } from "@/pages/admin-fases-view"
import { clavesConvocatorias } from "./queries"
import type {
  ConvocatoriaAdmin,
  ConvocatoriaPublica,
} from "@/types/convocatoria"

vi.mock("@/features/auth/sesion-context", () => ({
  useSesion: () => ({
    estado: "anonimo",
    postulante: { id: "admin-qa", rol: "administrador" },
  }),
}))
vi.mock("@/features/auth/use-ruta-inicial", () => ({
  useRutaInicial: () => ({ ruta: null, calculando: false }),
}))

export const PUBLICA: ConvocatoriaPublica = {
  id: "01931580-0000-7000-8000-000000000001",
  codigo: "QA-NO-OFICIAL",
  titulo: "Concurso de prueba",
  apertura: "2026-10-01T05:00:00Z",
  cierre: "2026-12-01T05:00:00Z",
  sectores: ["afiliados", "jubilados", "empleadores"],
  fases: [
    {
      codigo: "postulacion",
      titulo: "Recepción QA",
      apertura: "2026-10-01T05:00:00Z",
      cierre: "2026-11-15T05:00:00Z",
      publicada: true,
      habilitada: true,
    },
  ],
  ef12Disponible: true,
  zonaHoraria: "America/Guayaquil",
  limites: "[apertura,cierre)",
}
const ADMIN: ConvocatoriaAdmin = {
  ...PUBLICA,
  version: 7,
  inicioAprobado: true,
  publicada: true,
  ef12Habilitado: true,
}

function cliente(publica: ConvocatoriaPublica | null = PUBLICA) {
  const q = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  q.setQueryData(clavesConvocatorias.publicas, {
    pages: [{ items: publica ? [publica] : [], hasMore: false, limit: 20 }],
    pageParams: [undefined],
  })
  if (publica)
    q.setQueryData(clavesConvocatorias.detallePublico(publica.id), publica)
  q.setQueryData(clavesConvocatorias.adminLista("admin-qa:administrador"), {
    pages: [
      { items: [ADMIN], hasMore: true, nextCursor: "cursor-opaco", limit: 20 },
    ],
    pageParams: [undefined],
  })
  q.setQueryData(
    clavesConvocatorias.adminDetalle("admin-qa:administrador", ADMIN.id),
    ADMIN,
  )
  return q
}
function renderizar(
  elemento: ReactElement,
  publica: ConvocatoriaPublica | null = PUBLICA,
  ruta = "/",
  q = cliente(publica),
) {
  return renderToStaticMarkup(
    createElement(
      QueryClientProvider,
      { client: q },
      createElement(MemoryRouter, { initialEntries: [ruta] }, elemento),
    ),
  )
}
beforeEach(() => {
  vi.useFakeTimers()
  vi.setSystemTime(new Date("2026-10-31T16:00:00Z"))
})
afterEach(() => vi.useRealTimers())

describe("FT14 portal desde convocatorias reales", () => {
  it("presenta cronograma, sectores y acciones sólo del período seleccionado", () => {
    const html = renderizar(createElement(PortadaView))
    for (const texto of [
      "QA-NO-OFICIAL",
      "Postulaciones abiertas",
      "Afiliados",
      "Jubilados",
      "Empleadores",
      "Recepción QA",
      "America/Guayaquil",
      "OTP",
    ])
      expect(html).toContain(texto)
    expect(html).toContain(`/registro?convocatoria=${PUBLICA.id}`)
    expect(html).not.toContain("Registro Civil en línea")
    expect(html).not.toContain("2026 – 2030")
    expect(html).toContain('rel="noopener noreferrer"')
  })
  it("cierra por fecha exacta de la fase aunque la respuesta aún conserve true", () => {
    vi.setSystemTime(new Date(PUBLICA.fases[0].cierre))
    const html = renderizar(createElement(PortadaView))
    expect(html).toContain("Recepción cerrada")
    expect(html).not.toContain("Iniciar postulación")
  })
  it("distingue fase futura y deshabilitación", () => {
    const futura = {
      ...PUBLICA,
      fases: [{ ...PUBLICA.fases[0], apertura: "2026-11-10T05:00:00Z" }],
    }
    expect(renderizar(createElement(PortadaView), futura)).toContain(
      "Apertura programada",
    )
    expect(
      renderizar(createElement(PortadaView), {
        ...PUBLICA,
        ef12Disponible: false,
      }),
    ).toContain("Recepción deshabilitada")
  })
  it("no inventa convocatoria vacía ni declara un error de red como cierre", () => {
    const html = renderizar(createElement(PortadaView), null)
    expect(html).toContain("Sin convocatorias publicadas")
    expect(html).not.toContain("QA-NO-OFICIAL")
    expect(html).not.toContain("Iniciar postulación")
  })
  it("no habilita acciones con información almacenada hace más de 45 segundos", () => {
    const q = cliente()
    q.setQueryData(clavesConvocatorias.detallePublico(PUBLICA.id), PUBLICA, {
      updatedAt: Date.now() - 46_000,
    })
    const html = renderizar(createElement(PortadaView), PUBLICA, "/", q)
    expect(html).toContain("Disponibilidad por confirmar")
    expect(html).not.toContain("Iniciar postulación")
  })
  it("un enlace a período desconocido no se sustituye por el último de la lista", () => {
    const html = renderizar(
      createElement(PortadaView),
      PUBLICA,
      "/?convocatoria=otro-periodo",
    )
    expect(html).not.toContain("Iniciar postulación")
  })
})

describe("FT15 administración", () => {
  it("muestra histórico paginado sin denominar al número cargado como total global", () => {
    const html = renderizar(createElement(AdminConvocatoriasView))
    expect(html).toContain("Histórico de convocatorias")
    expect(html).toContain("Cargar más períodos")
    expect(html).toContain("Períodos cargados")
    expect(html).not.toContain("Total Convocatorias")
  })
  it("exige seleccionar período antes de configurar, sin tomar el primero", () => {
    const html = renderizar(createElement(AdminFasesView))
    expect(html).toContain("Seleccione el período que desea administrar")
    expect(html).not.toContain("Editar configuración</")
  })
  it("separa aprobación, visibilidad, recepción y fases, sin controles de escritura inmediata", () => {
    const html = renderizar(
      createElement(AdminFasesView),
      PUBLICA,
      `/admin/fases?convocatoria=${PUBLICA.id}`,
    )
    for (const texto of [
      "Versión 7",
      "Inicio del concurso",
      "Portal público",
      "Formulario EF-12",
      "Editar configuración",
      "Cronograma de fases",
      "Enlace para portales",
    ])
      expect(html).toContain(texto)
    expect(html).not.toContain('role="switch"')
    expect(html).toContain('target="_blank"')
    expect(html).not.toContain("En curso")
  })
})
