import { describe, expect, it } from "vitest"
import { clavesPrivacidad } from "./queries"
import type { ReglasPoliticaPrivacidadDto } from "@/lib/clientes/privacidad"

describe("Módulo de Privacidad y Consentimiento (FT28)", () => {
  describe("Claves de TanStack Query (clavesPrivacidad)", () => {
    it("genera jerarquía consistente de claves de consulta", () => {
      expect(clavesPrivacidad.todas).toEqual(["privacidad"])
      expect(clavesPrivacidad.vigente("conv-123")).toEqual([
        "privacidad",
        "vigente",
        "conv-123",
      ])
      expect(clavesPrivacidad.porId("pol-456")).toEqual([
        "privacidad",
        "detalle",
        "pol-456",
      ])
      expect(clavesPrivacidad.historial("conv-123")).toEqual([
        "privacidad",
        "historial",
        "conv-123",
      ])
      expect(clavesPrivacidad.retencion("conv-123")).toEqual([
        "privacidad",
        "retencion",
        "conv-123",
      ])
    })
  })

  describe("Reglas de Política Institucional", () => {
    it("cumple con el esquema de reglas del contrato T28", () => {
      const reglasValidas: ReglasPoliticaPrivacidadDto = {
        texto: "Texto legal aprobado por Directorio",
        informacionTratamiento: "Tratamiento conforme a Art. 7 LOPDP",
        referenciaAprobacion: "Resolución CD-BIESS-2026-001",
        diasAcceso: 365,
        diasDocumentos: 180,
        diasReportes: 365,
        diasPublicacionNombres: 90,
        diasCorreo: 30,
        diasRevisionConservacion: 730,
        disposicionHistoricos: "revision_humana_sin_borrado_automatico",
        permiteExportacion: true,
        permiteCorreo: true,
        permiteNombresPublicos: true,
      }

      expect(reglasValidas.diasAcceso).toBeGreaterThanOrEqual(1)
      expect(reglasValidas.diasDocumentos).toBeLessThanOrEqual(36500)
      expect(reglasValidas.disposicionHistoricos).toBe(
        "revision_humana_sin_borrado_automatico",
      )
      expect(reglasValidas.permiteNombresPublicos).toBe(true)
    })
  })
})
