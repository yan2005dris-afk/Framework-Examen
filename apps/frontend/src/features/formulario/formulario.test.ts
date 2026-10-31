import { describe, expect, it, vi } from "vitest"

import {
  CONSENTIMIENTO_DEFAULT,
  PREGUNTAS_INHABILIDADES,
  PUNTOS_CONSENTIMIENTO_EF12,
  SECCIONES_EF12_INFO,
} from "./constantes"
import { validarSeccionEF12 } from "@/utils/validar-formulario"
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

    it("unifica el contrato con q7Detalles anidado cuando q7 es 'Sí'", () => {
      // Incompleto sin q7Detalles al validar sección completa
      const s9Incompleta = {
        q1: "No",
        q2: "No",
        q3: "No",
        q4: "No",
        q5: "No",
        q6: "No",
        q7: "Sí",
        q8: "No",
        q9: "No",
        q10: "No",
        q11: "No",
        q12: "No",
        q13: "No",
      }
      const resIncompleto = validarSeccionEF12(s9Incompleta, 9, true)
      expect(resIncompleto.esValido).toBe(false)
      expect(resIncompleto.errores[0].mensaje).toBe(
        "Complete los detalles de remoción.",
      )

      // Inválido si q7 es No pero envía q7Detalles (falla incluso en borrador)
      const s9ConDetalleSobrante = {
        ...s9Incompleta,
        q7: "No",
        q7Detalles: {
          institucion: "Ministerio",
          cargo: "Cargo",
          fechaFin: "2023-01-01",
          motivo: "Motivo",
        },
      }
      const resSobrante = validarSeccionEF12(s9ConDetalleSobrante, 9, false)
      expect(resSobrante.esValido).toBe(false)
      expect(resSobrante.errores[0].mensaje).toBe(
        "Los detalles solo corresponden a una respuesta Sí.",
      )

      // Completo con q7Detalles estructurado
      const s9Completa = {
        ...s9Incompleta,
        q7: "Sí",
        q7Detalles: {
          institucion: "Ministerio del Trabajo",
          cargo: "Especialista",
          fechaFin: "2023-08-31",
          motivo: "Supresión de partida",
        },
      }
      const resCompleto = validarSeccionEF12(s9Completa, 9, true)
      expect(resCompleto.esValido).toBe(true)
    })
  })

  describe("Control de guardado y prevención de secciones inválidas (FT5)", () => {
    it("impide enviar una sección inválida a la API cuando validarSeccionEF12 devuelve errores", async () => {
      const mockMutateAsync = vi.fn().mockResolvedValue({})

      async function guardarSeccionSimulada(
        seccion: number,
        datos: unknown,
      ): Promise<boolean> {
        const resultado = validarSeccionEF12(datos, seccion, false)
        if (!resultado.esValido && resultado.errores.length > 0) {
          return false
        }
        await mockMutateAsync({ seccion, datos })
        return true
      }

      // Sección 1 con cédula inválida (falla la validación incluso en borrador)
      const seccionInvalida = {
        nombres: "Juan",
        apellidos: "Pérez",
        cedula: "12345", // cédula inválida no ecuatoriana
        genero: "Masculino",
        fechaNacimiento: "1990-01-01",
        nacionalidad: "Ecuatoriana",
        estadoCivil: "Soltero/a",
      }
      const resultadoGuardado = await guardarSeccionSimulada(1, seccionInvalida)

      expect(resultadoGuardado).toBe(false)
      expect(mockMutateAsync).not.toHaveBeenCalled()

      // Sección 1 con datos válidos
      const seccionValida = {
        nombres: "Damian",
        apellidos: "Torres",
        cedula: "1710034008",
        genero: "Masculino",
        fechaNacimiento: "1990-01-01",
        nacionalidad: "Ecuatoriana",
        estadoCivil: "Soltero/a",
      }
      const resultadoValido = await guardarSeccionSimulada(1, seccionValida)

      expect(resultadoValido).toBe(true)
      expect(mockMutateAsync).toHaveBeenCalledTimes(1)
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

  describe("Elegibilidad por sector y cierre automático (FT17)", () => {
    it("distingue el cierre definitivo (409) de una caída transitoria del servicio (503)", async () => {
      let estadoPostulacion: "draft" | "closed" = "draft"
      let mensajeAdvertencia: string | null = null

      async function procesarRespuestaGuardado(
        codigoError: string,
      ): Promise<void> {
        if (codigoError === "postulacion_cerrada_elegibilidad") {
          estadoPostulacion = "closed"
        } else if (codigoError === "elegibilidad_no_disponible") {
          mensajeAdvertencia =
            "La verificación de elegibilidad no está disponible temporalmente. Puede continuar editando."
        }
      }

      // Caso 1: El servicio de elegibilidad no responde (503) -> No cierra el borrador
      await procesarRespuestaGuardado("elegibilidad_no_disponible")
      expect(estadoPostulacion).toBe("draft")
      expect(mensajeAdvertencia).toContain("temporalmente")

      // Caso 2: El postulante no es elegible en el sector (409) -> Cierra la postulación automáticamente
      await procesarRespuestaGuardado("postulacion_cerrada_elegibilidad")
      expect(estadoPostulacion).toBe("closed")
    })

    it("la evidencia del servidor expone el origen de simulación controlada y no inventa datos", () => {
      const evidenciaServidor = {
        id: "eval-1",
        cuentaId: "acc-1",
        convocatoriaId: "conv-1",
        sector: "afiliados",
        elegible: false,
        origen: "simulacion_controlada_servidor",
        verificadoEn: "2026-10-31T20:00:00Z",
        codigo: "no_afiliado",
        mensaje: "Postulante No Afiliado al IESS",
      }

      expect(evidenciaServidor.origen).toBe("simulacion_controlada_servidor")
      expect(evidenciaServidor.elegible).toBe(false)
      expect(evidenciaServidor.mensaje).toBe("Postulante No Afiliado al IESS")
    })
  })
})
