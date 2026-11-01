import { afterEach, describe, expect, it, vi } from "vitest"
import {
  fechaControlEcuadorAISO,
  fechaParaControlEcuador,
  formatearFechaEcuador,
  obtenerEstadoTemporal,
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
