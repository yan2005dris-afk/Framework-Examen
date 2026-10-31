import { describe, expect, it } from "vitest"
import {
  diasHastaCierre,
  estaAbiertaConvocatoria,
  formatearFechaEcuador,
  obtenerEstadoTemporal,
} from "./fechas"

describe("fechas.ts - Utilidades de zona horaria de Ecuador (FT14)", () => {
  it("formatea la fecha en formato legible en español", () => {
    const texto = formatearFechaEcuador("2026-11-15T23:59:00-05:00")
    expect(texto).toContain("15 de noviembre de 2026")
  })

  it("formatea la fecha incluyendo hora en ECT", () => {
    const texto = formatearFechaEcuador("2026-11-15T23:59:00-05:00", {
      incluirHora: true,
    })
    expect(texto).toContain("15 de noviembre de 2026")
    expect(texto).toContain("23:59 (ECT)")
  })

  it("detecta cuando una convocatoria está cerrada por bandera explícita", () => {
    const abierta = estaAbiertaConvocatoria(
      "2020-01-01T00:00:00Z",
      "2030-01-01T00:00:00Z",
      true, // cerradaFlag
      true,
    )
    expect(abierta).toBe(false)
  })

  it("detecta cuando el formulario está deshabilitado", () => {
    const abierta = estaAbiertaConvocatoria(
      "2020-01-01T00:00:00Z",
      "2030-01-01T00:00:00Z",
      false,
      false, // postulacionHabilitada = false
    )
    expect(abierta).toBe(false)
  })

  it("evalúa estado temporal programada, abierta y cerrada", () => {
    const futuro = new Date(Date.now() + 1000 * 60 * 60 * 24 * 10).toISOString()
    const pasado = new Date(Date.now() - 1000 * 60 * 60 * 24 * 10).toISOString()
    const muchoFuturo = new Date(
      Date.now() + 1000 * 60 * 60 * 24 * 30,
    ).toISOString()

    expect(obtenerEstadoTemporal(futuro, muchoFuturo)).toBe("programada")
    expect(obtenerEstadoTemporal(pasado, futuro)).toBe("abierta")
    expect(obtenerEstadoTemporal(pasado, pasado)).toBe("cerrada")
  })

  it("calcula días hasta el cierre", () => {
    const diezDias = new Date(
      Date.now() + 1000 * 60 * 60 * 24 * 10 + 5000,
    ).toISOString()
    expect(diasHastaCierre(diezDias)).toBe(11)

    const pasado = new Date(Date.now() - 10000).toISOString()
    expect(diasHastaCierre(pasado)).toBe(0)
  })
})
