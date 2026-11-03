import { afterEach, describe, expect, it, vi } from "vitest"
import {
  fechaControlEcuadorAISO,
  fechaParaControlEcuador,
  formatearFechaEcuador,
  obtenerEstadoTemporal,
  diasHastaCierre,
  estaAbiertaConvocatoria,
} from "./fechas"

afterEach(() => vi.useRealTimers())
describe("Fechas de Ecuador independientes del equipo", () => {
  it("convierte UTC sin cortar la Z como si fuese hora de Ecuador", () => {
    expect(fechaParaControlEcuador("2026-11-01T02:30:00Z")).toBe(
      "2026-10-31T21:30",
    )
    expect(fechaControlEcuadorAISO("2026-10-31T21:30")).toBe(
      "2026-11-01T02:30:00.000Z",
    )
  })
  it.each([
    "",
    "2026-02-30T10:00",
    "2026-13-01T08:00",
    "2026-11-01",
    "2026-01-01T25:00",
    "texto",
  ])("rechaza %s sin toISOString sobre fecha inválida", (fecha) =>
    expect(fechaControlEcuadorAISO(fecha)).toBeNull(),
  )
  it("mantiene el día correcto cerca de medianoche y muestra zona explícita", () => {
    expect(
      formatearFechaEcuador("2026-11-01T02:30:00Z", { incluirHora: true }),
    ).toBe("31 de octubre de 2026 · 21:30 (ECT)")
  })
  it("nunca declara una fecha inválida como abierta", () => {
    expect(obtenerEstadoTemporal("no", "sí")).toBe("cerrada")
    vi.useFakeTimers()
    vi.setSystemTime(new Date("2026-10-31T15:00:00Z"))
    expect(
      obtenerEstadoTemporal(
        "2027-01-01T00:00:00Z",
        "2027-02-01T00:00:00Z",
        true,
      ),
    ).toBe("programada")
  })
})

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
