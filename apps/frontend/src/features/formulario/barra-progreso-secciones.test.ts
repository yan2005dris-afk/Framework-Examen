import { createElement } from "react"
import { renderToStaticMarkup } from "react-dom/server"
import { MemoryRouter } from "react-router"
import { describe, expect, it } from "vitest"
import { BarraProgresoSecciones } from "./barra-progreso-secciones"

describe("navegación del formulario según el mockup", () => {
  it("conserva nueve rutas y la convocatoria; marca el paso activo, no una página distinta", () => {
    const html = renderToStaticMarkup(
      createElement(
        MemoryRouter,
        null,
        createElement(BarraProgresoSecciones, {
          seccionActiva: 2,
          convocatoriaId: "periodo-prueba",
          seccionesCompletadas: { 1: true, 2: false, 3: false },
        }),
      ),
    )
    expect(html.match(/<a /g)).toHaveLength(9)
    expect(html.match(/aria-current="step"/g)).toHaveLength(1)
    for (let n = 1; n <= 9; n++)
      expect(html).toContain(`/postulacion/${n}?convocatoria=periodo-prueba`)
    expect(html).toContain("Avance: 1 de 9 secciones")
    expect(html).toContain("Domicilio y contacto")
    expect(html).toContain("overflow-x-auto")
    expect(html).toContain('class="relative flex')
  })
  it("el avance cuenta completadas, no el índice del paso seleccionado", () => {
    const html = renderToStaticMarkup(
      createElement(
        MemoryRouter,
        null,
        createElement(BarraProgresoSecciones, {
          seccionActiva: 9,
          seccionesCompletadas: {},
        }),
      ),
    )
    expect(html).toContain("Avance: 0 de 9 secciones")
  })
})
