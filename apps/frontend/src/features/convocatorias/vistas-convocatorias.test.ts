import { describe, expect, it, vi } from "vitest"
import { createElement } from "react"
import { renderToStaticMarkup } from "react-dom/server"
import { MemoryRouter } from "react-router"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"

import { PortadaView } from "@/pages/portada-view"
import { AdminConvocatoriasView } from "@/pages/admin-convocatorias-view"
import { AdminFasesView } from "@/pages/admin-fases-view"
import { clavesConvocatorias } from "./queries"
import { CONVOCATORIA_OFICIAL_BASE } from "@/lib/clientes/convocatorias"

vi.mock("@/features/auth/sesion-context", () => ({
  useSesion: () => ({
    estado: "autenticado",
    postulante: {
      id: "admin-1",
      cedula: "0910034008",
      nombres: "DAMIAN",
      apellidos: "ADMIN",
      correo: "admin@biess.local",
      rol: "administrador",
    },
  }),
}))

vi.mock("@/features/auth/use-ruta-inicial", () => ({
  useRutaInicial: () => ({
    ruta: "/admin/convocatorias",
    calculando: false,
  }),
}))

function prepararCliente(
  convocatoriaPersonalizada = CONVOCATORIA_OFICIAL_BASE,
) {
  const cliente = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })

  cliente.setQueryData(clavesConvocatorias.publica, convocatoriaPersonalizada)

  cliente.setQueryData([...clavesConvocatorias.adminLista, undefined], {
    items: [convocatoriaPersonalizada],
    hasMore: false,
    limit: 20,
  })

  return cliente
}

function renderizarConContexto(
  elemento: React.ReactElement,
  convocatoria = CONVOCATORIA_OFICIAL_BASE,
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

describe("Vistas de Portada y Administración (FT14 y FT15)", () => {
  it("PortadaView renderiza el título oficial, período y los tres sectores sociales", () => {
    const html = renderizarConContexto(createElement(PortadaView))

    // Título y resolución
    expect(html).toContain("Concurso Público de Méritos y Oposición")
    expect(html).toContain("Directorio del BIESS")
    expect(html).toContain("Resolución No. C.D. 701")

    // Sectores
    expect(html).toContain("Afiliados")
    expect(html).toContain("Jubilados")
    expect(html).toContain("Empleadores")
    expect(html).toContain("1 principal y 1 alterno")

    // 4 pasos
    expect(html).toContain("Paso 1")
    expect(html).toContain("Registro de identidad")
    expect(html).toContain("Formulario EF-12")
    expect(html).toContain("Envío y sellado criptográfico")

    // Enlaces oficiales IESS y BIESS
    expect(html).toContain("https://www.biess.fin.ec")
    expect(html).toContain("https://www.iess.gob.ec")
    expect(html).toContain('target="_blank"')
    expect(html).toContain('rel="noopener noreferrer"')
  })

  it("PortadaView muestra estado de postulación abierta", () => {
    const html = renderizarConContexto(createElement(PortadaView))
    expect(html).toContain("Postulaciones Abiertas")
  })

  it("PortadaView muestra alerta prominente cuando la convocatoria está cerrada", () => {
    const cerrada = {
      ...CONVOCATORIA_OFICIAL_BASE,
      cerrada: true,
      estado: "cerrada" as const,
      cierre: "2026-10-01T00:00:00Z",
    }
    const html = renderizarConContexto(createElement(PortadaView), cerrada)

    expect(html).toContain("Convocatoria no admite postulaciones")
    expect(html).toContain("El plazo de recepción de postulaciones concluyó")
    expect(html).toContain("Convocatoria Cerrada")
  })

  it("AdminConvocatoriasView muestra tabla de convocatorias y métricas", () => {
    const html = renderizarConContexto(
      createElement(AdminConvocatoriasView),
      CONVOCATORIA_OFICIAL_BASE,
      "/admin/convocatorias",
    )

    expect(html).toContain("Gestión de Convocatorias")
    expect(html).toContain("BIESS-DIR-2026")
    expect(html).toContain("Nueva convocatoria")
    expect(html).toContain("Total Convocatorias")
    expect(html).toContain("Convocatorias Activas")
  })

  it("AdminFasesView renderiza las 6 fases reglamentarias y el switch de habilitación EF-12", () => {
    const html = renderizarConContexto(
      createElement(AdminFasesView),
      CONVOCATORIA_OFICIAL_BASE,
      "/admin/fases?convocatoria=01931580-0000-7000-8000-000000000001",
    )

    expect(html).toContain("Fases y Cronograma Oficial")
    expect(html).toContain("Admisión y Recepción del Formulario EF-12")
    expect(html).toContain("Cronograma Oficial de Fases (1 a 6)")

    // Verificación de las 6 fases
    expect(html).toContain("Fase 1: Convocatoria pública y postulación")
    expect(html).toContain(
      "Fase 2: Verificación de requisitos mínimos e inhabilidades",
    )
    expect(html).toContain("Fase 3: Valoración y calificación de méritos")
    expect(html).toContain(
      "Fase 4: Examen técnico de conocimientos (Oposición)",
    )
    expect(html).toContain("Fase 5: Impugnación ciudadana y reconsideración")
    expect(html).toContain("Fase 6: Designación oficial y posesión")
  })
})
