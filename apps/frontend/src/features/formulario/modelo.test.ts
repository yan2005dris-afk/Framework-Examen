import { describe, expect, it } from "vitest"

import { aplicarValoresIniciales } from "./modelo"
import { validarSeccionEF12 } from "./validacion"

describe("aplicarValoresIniciales", () => {
  it("completa el país y la nacionalidad de un borrador recién creado", () => {
    const creado = {
      seccion1: { cedula: "1710034008", nacionalidad: "" },
      seccion2: { correo1: "postulante@biess.local", pais: "" },
    }
    const resultado = aplicarValoresIniciales(creado)
    expect(resultado.seccion1).toMatchObject({ nacionalidad: "Ecuatoriana" })
    expect(resultado.seccion2).toMatchObject({
      pais: "Ecuador",
      correo1: "postulante@biess.local",
    })
  })

  it("nunca reemplaza un valor ya elegido", () => {
    const resultado = aplicarValoresIniciales({
      seccion2: { pais: "Colombia" },
    })
    expect(resultado.seccion2).toEqual({ pais: "Colombia" })
  })

  it("no inventa secciones que no llegaron ni modifica el original", () => {
    const original = { seccion3: { tieneFamiliaresIessBiess: "No" } }
    const resultado = aplicarValoresIniciales(original)
    expect(resultado).toEqual(original)
    expect(resultado).not.toBe(original)
  })

  it("la sección 2 resultante pasa la validación sin volver a elegir el país", () => {
    const { seccion2 } = aplicarValoresIniciales({
      seccion2: { correo1: "postulante@biess.local", pais: "" },
    })
    expect(validarSeccionEF12(seccion2, 2).esValido).toBe(true)
  })
})
