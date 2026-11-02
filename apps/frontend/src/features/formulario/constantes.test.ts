import { describe, expect, it } from "vitest"

import { AREAS_EF12 } from "./constantes"
import { validarSeccionEF12 } from "./validacion"

describe("áreas del RF 1.5.1.4 y 1.6.6", () => {
  it("ofrece exactamente las cinco áreas del requerimiento", () => {
    expect(AREAS_EF12).toEqual([
      "Finanzas",
      "Mercado de Capitales",
      "Administración de Empresas",
      "Economía",
      "Derecho o materias afines",
    ])
  })

  it("cada área de la lista es aceptada en la experiencia profesional", () => {
    for (const area of AREAS_EF12) {
      const seccion6 = {
        experiencias: [
          {
            id: "exp-1",
            tipo: "General",
            cargo: "Analista",
            institucion: "Institución de prueba",
            fechaInicio: "2020-01-01",
            fechaFin: "2022-01-01",
            actualmenteLaborando: false,
            actividades: area,
            descripcionFunciones: "Funciones de prueba",
          },
        ],
      }
      expect(validarSeccionEF12(seccion6, 6).esValido, area).toBe(true)
    }
  })
})
