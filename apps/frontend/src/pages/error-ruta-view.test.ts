import { createElement } from "react"
import { renderToStaticMarkup } from "react-dom/server"
import { MemoryRouter } from "react-router"
import { describe, expect, it } from "vitest"

import { PantallaErrorRuta } from "./error-ruta-view"

const DETALLE =
  "Failed to execute 'removeChild' on 'Node': The node to be removed is not a child of this node."

function render(mostrarDetalle: boolean): string {
  return renderToStaticMarkup(
    createElement(
      MemoryRouter,
      null,
      createElement(PantallaErrorRuta, { detalle: DETALLE, mostrarDetalle }),
    ),
  )
}

describe("pantalla de error de las rutas", () => {
  it("ofrece recargar y volver al inicio con un mensaje en español", () => {
    const html = render(false)
    expect(html).toContain("Ocurrió un problema al mostrar esta página")
    expect(html).toContain("Recargar la página")
    expect(html).toContain("Volver al inicio")
  })

  it("no muestra el error técnico fuera de desarrollo", () => {
    expect(render(false)).not.toContain("removeChild")
  })

  it("muestra el detalle técnico en desarrollo", () => {
    expect(render(true)).toContain("removeChild")
  })
})
