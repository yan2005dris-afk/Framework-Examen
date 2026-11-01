import { describe, expect, it } from "vitest"
import {
  estadoRecepcion,
  estadoFase,
  validarConfiguracion,
  enlacePublicoConvocatoria,
  detalleErrorConvocatoria,
} from "./modelo"
import { ErrorApi } from "@/lib/api-error"
import type {
  ConfiguracionConvocatoria,
  ConvocatoriaPublica,
} from "@/types/convocatoria"

const c: ConvocatoriaPublica = {
  id: "periodo-qa",
  codigo: "QA",
  titulo: "Sintético",
  apertura: "2026-10-01T00:00:00Z",
  cierre: "2026-12-01T00:00:00Z",
  sectores: ["afiliados"],
  ef12Disponible: true,
  zonaHoraria: "America/Guayaquil",
  limites: "[apertura,cierre)",
  fases: [
    {
      codigo: "postulacion",
      titulo: "Recepción QA",
      apertura: "2026-10-15T05:00:00Z",
      cierre: "2026-11-15T05:00:00Z",
      publicada: true,
      habilitada: true,
    },
  ],
}
const ahora = Date.parse("2026-10-31T15:00:00Z")
function config(): ConfiguracionConvocatoria {
  return {
    version: 3,
    inicioAprobado: true,
    publicada: true,
    ef12Habilitado: true,
    sectores: ["afiliados"],
    fases: structuredClone(c.fases),
  }
}

describe("Límites de recepción FT14", () => {
  it.each([
    ["2026-09-30T00:00:00Z", "programada"],
    ["2026-10-10T00:00:00Z", "programada"],
    ["2026-10-15T05:00:00Z", "abierta"],
    ["2026-11-15T04:59:59.999Z", "abierta"],
    ["2026-11-15T05:00:00Z", "cerrada"],
    ["2026-12-01T00:00:00Z", "cerrada"],
  ])("%s => %s", (fecha, resultado) => {
    expect(estadoRecepcion(c, Date.parse(fecha))).toBe(resultado)
  })
  it("nunca deriva habilitación sólo desde la fecha", () => {
    expect(estadoRecepcion({ ...c, ef12Disponible: false }, ahora)).toBe(
      "suspendida",
    )
    expect(estadoRecepcion(c, ahora, false)).toBe("sin_confirmar")
    expect(estadoRecepcion(undefined, ahora)).toBe("sin_confirmar")
    expect(estadoRecepcion({ ...c, apertura: "inválida" }, ahora)).toBe(
      "sin_confirmar",
    )
    expect(estadoRecepcion({ ...c, fases: [] }, ahora)).toBe("suspendida")
    expect(
      estadoRecepcion(
        { ...c, fases: [{ ...c.fases[0], habilitada: false }] },
        ahora,
      ),
    ).toBe("suspendida")
  })
  it("fase futura o finalizada no se presenta como en curso por habilitada=true", () => {
    expect(estadoFase(c.fases[0], Date.parse("2026-10-01T00:00:00Z"))).toBe(
      "Programada",
    )
    expect(estadoFase(c.fases[0], Date.parse(c.fases[0].cierre))).toBe(
      "Finalizada",
    )
  })
})
describe("Validación preventiva de configuración T18", () => {
  it("acepta configuración coherente y conserva la versión", () => {
    const v = config()
    expect(validarConfiguracion(v, c.apertura, c.cierre)).toBeNull()
    expect(v.version).toBe(3)
  })
  it.each([
    "sector",
    "aprobacion",
    "fase_ef12",
    "duplicada",
    "fecha",
    "fuera",
    "publicacion",
    "titulo",
  ])("rechaza %s sin enviar", (caso) => {
    const v = config()
    switch (caso) {
      case "sector":
        v.sectores = []
        break
      case "aprobacion":
        v.inicioAprobado = false
        break
      case "fase_ef12":
        v.fases = []
        break
      case "duplicada":
        v.fases.push({ ...v.fases[0] })
        break
      case "fecha":
        v.fases[0].cierre = v.fases[0].apertura
        break
      case "fuera":
        v.fases[0].cierre = "2027-01-01T00:00:00Z"
        break
      case "publicacion":
        v.fases[0].publicada = false
        break
      case "titulo":
        v.fases[0].titulo = " "
        break
    }
    expect(validarConfiguracion(v, c.apertura, c.cierre)).toBeTruthy()
  })
  it("mantiene HTTP y código de conflicto/permisos para trazabilidad", () => {
    expect(
      detalleErrorConvocatoria(
        new ErrorApi("Cambió la versión", {
          estado: 409,
          codigo: "version_conflicto",
        }),
      ),
    ).toContain("HTTP 409 · version_conflicto")
    expect(
      detalleErrorConvocatoria(
        new ErrorApi("Sin permiso", { estado: 403, codigo: "acceso_denegado" }),
      ),
    ).toContain("403")
  })
  it("el enlace exacto no arrastra filtros, sesiones ni rutas administrativas", () => {
    expect(
      enlacePublicoConvocatoria(
        "https://portal.example.test/admin/fases?token=secreto",
        "uuid-qa",
      ),
    ).toBe("https://portal.example.test/?convocatoria=uuid-qa")
  })
})
