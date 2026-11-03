import { createElement } from "react"
import { renderToStaticMarkup } from "react-dom/server"
import { MemoryRouter } from "react-router"
import { describe, expect, it } from "vitest"
import { CabeceraPortal, PiePortal } from "./cabecera-portal"

describe("marco visual del portal", () => {
  it("mantiene el enlace activo, la identidad y el salto al contenido sin códigos del plan", () => {
    const html = renderToStaticMarkup(
      createElement(
        MemoryRouter,
        null,
        createElement(CabeceraPortal, {
          destino: "/",
          enlaces: [
            {
              titulo: "Formulario",
              ruta: "/postulacion?convocatoria=periodo",
              activo: true,
            },
          ],
          cuenta: { nombre: "Persona Prueba", detalle: "Postulante" },
        }),
      ),
    )
    expect(html).toContain('href="#contenido-portal"')
    expect(html).toContain('href="/postulacion?convocatoria=periodo"')
    expect(html).toContain('aria-current="page"')
    expect(html).toContain("Directorio del BIESS")
    expect(html).toContain("Persona Prueba")
    expect(html).not.toMatch(/EF-\d+|FT\d+|HU-\d+/)
    expect(html).not.toContain('data-slot="sidebar"')
  })
  it("oculta el cierre de sesión si no existe una acción autorizada", () => {
    const html = renderToStaticMarkup(
      createElement(
        MemoryRouter,
        null,
        createElement(CabeceraPortal, { destino: "/", enlaces: [] }),
      ),
    )
    expect(html).not.toContain(">Salir<")
  })
  it("excluye el pie de la impresión", () => {
    expect(renderToStaticMarkup(createElement(PiePortal))).toContain(
      "print:hidden",
    )
  })
})
