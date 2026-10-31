import { describe, expect, it, vi } from "vitest"
import { createElement } from "react"
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
  ResumenConvocatoriaAdmin,
} from "@/types/convocatoria"

let mockSesion = {
  estado: "anonimo",
  postulante: null as {
    id: string
    cedula: string
    nombres: string
    apellidos: string
    correo: string
    rol: string
  } | null,
}

vi.mock("@/features/auth/sesion-context", () => ({
  useSesion: () => mockSesion,
}))

vi.mock("@/features/auth/use-ruta-inicial", () => ({
  useRutaInicial: () => ({
    ruta: "/admin/convocatorias",
    calculando: false,
  }),
}))

const MOCK_PUBLICA_ACTIVA: ConvocatoriaPublica = {
  id: "01931580-0000-7000-8000-000000000001",
  codigo: "BIESS-DIR-2026",
  titulo:
    "Concurso Público de Méritos y Oposición para la designación de los Miembros del Directorio del BIESS",
  apertura: "2026-10-15T00:00:00-05:00",
  cierre: "2026-11-15T23:59:59-05:00",
  sectores: ["afiliados", "jubilados", "empleadores"],
  fases: [
    {
      codigo: "postulacion",
      titulo: "Recepción de postulaciones (EF-12)",
      apertura: "2026-10-15T00:00:00-05:00",
      cierre: "2026-11-15T23:59:59-05:00",
      publicada: true,
      habilitada: true,
    },
  ],
  ef12Disponible: true,
  zonaHoraria: "America/Guayaquil",
  limites: "[apertura,cierre)",
}

const MOCK_RESUMEN_ADMIN: ResumenConvocatoriaAdmin = {
  id: "01931580-0000-7000-8000-000000000001",
  codigo: "BIESS-DIR-2026",
  titulo: "Concurso Público de Méritos y Oposición BIESS",
  apertura: "2026-10-15T00:00:00-05:00",
  cierre: "2026-11-15T23:59:59-05:00",
  cerrada: false,
}

const MOCK_DETALLE_ADMIN: ConvocatoriaAdmin = {
  id: "01931580-0000-7000-8000-000000000001",
  codigo: "BIESS-DIR-2026",
  titulo: "Concurso Público de Méritos y Oposición BIESS",
  apertura: "2026-10-15T00:00:00-05:00",
  cierre: "2026-11-15T23:59:59-05:00",
  version: 2,
  inicioAprobado: true,
  publicada: true,
  ef12Habilitado: true,
  sectores: ["afiliados", "jubilados", "empleadores"],
  fases: [
    {
      codigo: "postulacion",
      titulo: "Recepción de postulaciones (EF-12)",
      apertura: "2026-10-15T00:00:00-05:00",
      cierre: "2026-11-15T23:59:59-05:00",
      publicada: true,
      habilitada: true,
    },
  ],
}

function prepararCliente(
  convocatoriaPublica: ConvocatoriaPublica | null = MOCK_PUBLICA_ACTIVA,
) {
  const cliente = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })

  cliente.setQueryData(clavesConvocatorias.publica, convocatoriaPublica)

  cliente.setQueryData([...clavesConvocatorias.adminLista, undefined], {
    items: [MOCK_RESUMEN_ADMIN],
    hasMore: false,
    limit: 20,
  })

  cliente.setQueryData(
    clavesConvocatorias.adminDetalle(MOCK_DETALLE_ADMIN.id),
    MOCK_DETALLE_ADMIN,
  )

  return cliente
}

function renderizarConContexto(
  elemento: React.ReactElement,
  convocatoria: ConvocatoriaPublica | null = MOCK_PUBLICA_ACTIVA,
  ruta = "/",
) {
  const cliente = prepararCliente(convocatoria)
  return renderToStaticMarkup(
    createElement(
      QueryClientProvider,
      { client: cliente },
      createElement(MemoryRouter, { initialEntries: [ruta] }, elemento),
    ),
  )
}

describe("Vistas de Portada y Administración (Contrato T18 sin datos ficticios)", () => {
  it("PortadaView renderiza la convocatoria pública aprobada y los tres sectores sociales", () => {
    const html = renderizarConContexto(createElement(PortadaView))

    // Título y código oficial
    expect(html).toContain("Concurso Público de Méritos y Oposición")
    expect(html).toContain("BIESS-DIR-2026")

    // Sectores
    expect(html).toContain("Afiliados")
    expect(html).toContain("Jubilados")
    expect(html).toContain("Empleadores")
    expect(html).toContain("1 principal y 1 alterno")

    // Estado abierto
    expect(html).toContain("Postulaciones Abiertas")

    // Enlaces oficiales IESS y BIESS
    expect(html).toContain("https://www.biess.fin.ec")
    expect(html).toContain("https://www.iess.gob.ec")
  })

  it("PortadaView muestra estado de convocatoria no admite postulaciones cuando ef12Disponible es false", () => {
    const cerrada: ConvocatoriaPublica = {
      ...MOCK_PUBLICA_ACTIVA,
      ef12Disponible: false,
      cierre: "2026-10-01T00:00:00Z",
    }
    const html = renderizarConContexto(createElement(PortadaView), cerrada)

    expect(html).toContain("Convocatoria no admite postulaciones")
    expect(html).toContain("Convocatoria Cerrada")
    expect(html).toContain("Postulaciones no disponibles")
  })

  it("PortadaView muestra honestamente estado sin convocatoria cuando el servidor no tiene concursos aprobados", () => {
    const html = renderizarConContexto(createElement(PortadaView), null)

    expect(html).toContain("Sin convocatoria activa para postulación")
    expect(html).toContain(
      "Actualmente no existe una convocatoria pública aprobada",
    )
    expect(html).toContain("Postulaciones no disponibles")
  })

  it("AdminConvocatoriasView muestra listado institucional de convocatorias", () => {
    mockSesion = {
      estado: "autenticado",
      postulante: {
        id: "admin-1",
        cedula: "0910034008",
        nombres: "DAMIAN",
        apellidos: "ADMIN",
        correo: "admin@biess.local",
        rol: "administrador",
      },
    }

    const html = renderizarConContexto(
      createElement(AdminConvocatoriasView),
      MOCK_PUBLICA_ACTIVA,
      "/admin/convocatorias",
    )

    expect(html).toContain("Gestión de Convocatorias")
    expect(html).toContain("BIESS-DIR-2026")
    expect(html).toContain("Nueva convocatoria")
    expect(html).toContain("Total Convocatorias")
  })

  it("AdminFasesView renderiza la versión optimista y controles de aprobación y EF-12", () => {
    const html = renderizarConContexto(
      createElement(AdminFasesView),
      MOCK_PUBLICA_ACTIVA,
      `/admin/fases?convocatoria=${MOCK_DETALLE_ADMIN.id}`,
    )

    expect(html).toContain("Fases y Configuración (T18)")
    expect(html).toContain("v2")
    expect(html).toContain("Inicio Aprobado")
    expect(html).toContain("Publicada en Portal")
    expect(html).toContain("Formulario EF-12")
    expect(html).toContain("postulacion")
    expect(html).toContain("Control de concurrencia optimista")
  })
})
