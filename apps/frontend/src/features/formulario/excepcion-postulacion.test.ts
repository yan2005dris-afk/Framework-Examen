import { createElement } from "react"
import { renderToStaticMarkup } from "react-dom/server"
import { MemoryRouter } from "react-router"
import { describe, expect, it } from "vitest"
import { ExcepcionPostulacion } from "./excepcion-postulacion"
import { PantallaBloqueoRegistro } from "@/features/auth/pantalla-bloqueo-registro"

describe("pantallas de excepción sin evidencia inventada", () => {
  it("una declaración incorrecta no se presenta como cierre definitivo", () => {
    const html = renderToStaticMarkup(
      createElement(
        MemoryRouter,
        null,
        createElement(ExcepcionPostulacion, {
          tipo: "nacionalidad",
          valor: "Extranjera",
          cerrada: false,
          onCorregir: () => {},
        }),
      ),
    )
    expect(html).toContain("Pendiente de corrección")
    expect(html).toContain("Volver a editar mis datos")
    expect(html).not.toContain("Cerrada por el servidor")
  })
  it("un cierre del servidor no muestra acciones para reabrir o corregir", () => {
    const html = renderToStaticMarkup(
      createElement(
        MemoryRouter,
        null,
        createElement(ExcepcionPostulacion, {
          tipo: "sector",
          valor: "afiliados",
          cerrada: true,
          mensaje: "No elegible",
          origen: "Simulación controlada del servidor",
        }),
      ),
    )
    expect(html).toContain("Cerrada por el servidor")
    expect(html).toContain("No elegible")
    expect(html).not.toContain("Volver a editar mis datos")
  })
  it("el bloqueo no inventa un historial ni certifica identidad civil", () => {
    const html = renderToStaticMarkup(
      createElement(
        MemoryRouter,
        null,
        createElement(PantallaBloqueoRegistro, {
          expiraEn: new Date(Date.now() + 600000),
          onExpirar: () => {},
        }),
      ),
    )
    expect(html).toContain("Registro bloqueado temporalmente")
    expect(html).toContain("Tiempo restante de bloqueo")
    expect(html).not.toContain("<table")
    expect(html).not.toContain("coincide con el Registro Civil")
  })
})
