import { describe, expect, it } from "vitest"

import casosDuracion from "../../../backend/internal/submission/testdata/duration_cases.json"
import { calcularDuracionExperiencia } from "./experiencia"

describe("calcularDuracionExperiencia", () => {
  it.each(casosDuracion)(
    "coincide con Go: $start / $end",
    ({ start, end, anios, meses, dias, texto }) => {
      expect(calcularDuracionExperiencia(start, end)).toEqual({
        anios,
        meses,
        dias,
        texto,
      })
    },
  )

  it("rechaza fechas de calendario imposibles", () => {
    expect(calcularDuracionExperiencia("2024-02-30", "2024-03-02").texto).toBe(
      "Fecha no válida",
    )
  })
  it("calcula años, meses y días entre dos fechas", () => {
    const resultado = calcularDuracionExperiencia("2020-01-01", "2021-03-15")

    expect(resultado.anios).toBe(1)
    expect(resultado.meses).toBe(2)
    expect(resultado.dias).toBe(14)
    expect(resultado.texto).toBe("1 año, 2 meses, 14 días")
  })

  it("devuelve cero días cuando falta la fecha de inicio", () => {
    const resultado = calcularDuracionExperiencia("")

    expect(resultado.anios).toBe(0)
    expect(resultado.dias).toBe(0)
  })

  it("marca la fecha como inválida cuando el rango está invertido", () => {
    const resultado = calcularDuracionExperiencia("2021-03-15", "2020-01-01")

    expect(resultado.texto).toBe("Fecha no válida")
  })
})
