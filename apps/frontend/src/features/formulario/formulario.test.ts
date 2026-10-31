import { describe, expect, it } from "vitest"

import {
  CONSENTIMIENTO_DEFAULT,
  PREGUNTAS_INHABILIDADES,
  PUNTOS_CONSENTIMIENTO_EF12,
  SECCIONES_EF12_INFO,
} from "./constantes"
import { calcularDuracionExperiencia } from "@/utils/validators"

describe("Formulario EF-12 (FT5 / FT6)", () => {
  describe("BarraProgresoSecciones", () => {
    it("declara exactamente las 9 secciones oficiales del EF-12", () => {
      expect(SECCIONES_EF12_INFO).toHaveLength(9)
      expect(SECCIONES_EF12_INFO.map((s) => s.numero)).toEqual([
        1, 2, 3, 4, 5, 6, 7, 8, 9,
      ])
    })
  })

  describe("Inhabilidades (Sección 9)", () => {
    it("incluye las 13 preguntas literales del RF sin resumir", () => {
      expect(PREGUNTAS_INHABILIDADES).toHaveLength(13)
      expect(PREGUNTAS_INHABILIDADES[0].numero).toBe("1.9.1")
      expect(PREGUNTAS_INHABILIDADES[6].numero).toBe("1.9.7")
      expect(PREGUNTAS_INHABILIDADES[6].tieneDetalle).toBe(true)
      expect(PREGUNTAS_INHABILIDADES[12].numero).toBe("1.9.13")
    })
  })

  describe("Cálculo de duración en experiencia (Sección 6)", () => {
    it("calcula años, meses y días correctamente para un período cerrado", () => {
      const duracion = calcularDuracionExperiencia("2020-01-01", "2021-03-15")
      expect(duracion.anios).toBe(1)
      expect(duracion.meses).toBe(2)
      expect(duracion.dias).toBe(14)
      expect(duracion.texto).toContain("1 año")
    })

    it("maneja el cálculo en tiempo real si actualmente labora allí", () => {
      const duracion = calcularDuracionExperiencia("2020-01-01", "", true)
      expect(duracion.anios).toBeGreaterThanOrEqual(4)
      expect(duracion.texto).toBeDefined()
    })
  })

  describe("Consentimiento y Declaración Jurada (FT6 - EF-12 1.10.1)", () => {
    it("contiene exactamente los 5 puntos literales exigidos por el RF", () => {
      expect(PUNTOS_CONSENTIMIENTO_EF12).toHaveLength(5)
      expect(PUNTOS_CONSENTIMIENTO_EF12.map((p) => p.numero)).toEqual([
        1, 2, 3, 4, 5,
      ])
    })

    it("inicia con todos los consentimientos desmarcados por defecto", () => {
      expect(CONSENTIMIENTO_DEFAULT.aceptacionTratamientoDatos).toBe(false)
      expect(CONSENTIMIENTO_DEFAULT.declaracionVeracidad).toBe(false)
      expect(CONSENTIMIENTO_DEFAULT.aceptacionNormasConcurso).toBe(false)
      expect(CONSENTIMIENTO_DEFAULT.autorizacionPublicacionNombre).toBe(false)
      expect(CONSENTIMIENTO_DEFAULT.constanciaEntregaFisica).toBe(false)
    })

    it("incluye los textos literales clave en cada punto legal", () => {
      const [p1, p2, p3, p4, p5] = PUNTOS_CONSENTIMIENTO_EF12
      expect(p1.texto).toContain(
        "Ley Orgánica de Protección de Datos Personales",
      )
      expect(p2.texto).toContain("corresponde a la verdad")
      expect(p3.texto).toContain("descalificación del proceso")
      expect(p4.texto).toContain("página web del IESS - BIESS")
      expect(p5.texto).toContain("firma manuscrita")
    })
  })
})
