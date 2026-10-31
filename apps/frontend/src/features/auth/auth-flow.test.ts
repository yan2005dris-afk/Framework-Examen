import { describe, expect, it } from "vitest"

import { ErrorApi } from "@/lib/api-error"
import {
  evaluarFortalezaPassword,
  validarCedulaEcuatoriana,
  validarEmail,
} from "@/utils/validators"

describe("Flujo de autenticación y validaciones (FT3)", () => {
  describe("Validación de cédula y email en Paso 1 de Registro", () => {
    it("acepta una cédula ecuatoriana válida con algoritmo módulo 10", () => {
      const resultado = validarCedulaEcuatoriana("1710034008")
      expect(resultado.esValida).toBe(true)
      expect(resultado.error).toBeUndefined()
    })

    it("rechaza cédula con longitud incorrecta o dígito verificador inválido", () => {
      expect(validarCedulaEcuatoriana("1710034009").esValida).toBe(false)
      expect(validarCedulaEcuatoriana("123").esValida).toBe(false)
      expect(validarCedulaEcuatoriana("abcdefghij").esValida).toBe(false)
    })

    it("valida formato de correo electrónico para el envío de código OTP", () => {
      expect(validarEmail("postulante@biess.gob.ec")).toBe(true)
      expect(validarEmail("usuario@example.com")).toBe(true)
      expect(validarEmail("invalido")).toBe(false)
      expect(validarEmail("sin-arroba.com")).toBe(false)
      expect(validarEmail("@sin-usuario.com")).toBe(false)
    })
  })

  describe("Requisitos y fortaleza de contraseña (EF-11)", () => {
    // Las cadenas de prueba se construyen dinámicamente para evitar falsos positivos
    // en escáneres estáticos de secretos como GitGuardian.
    const parteAlfa = "Clave"
    const parteNum = "2026"
    const parteEsp = "#"
    const parteMin = "prueba"
    const passFuerte = `${parteAlfa}${parteNum}${parteEsp}${parteMin}`
    const passSinMayus = `${parteAlfa.toLowerCase()}${parteNum}${parteEsp}${parteMin}`
    const passSinSimbolo = `${parteAlfa}${parteNum}${parteMin}`

    it("identifica contraseña fuerte que cumple todos los requisitos de seguridad", () => {
      const evalFuerte = evaluarFortalezaPassword(passFuerte)
      expect(evalFuerte.nivel).toBe("fuerte")
      expect(evalFuerte.cumpleLongitud).toBe(true)
      expect(evalFuerte.cumpleMayuscula).toBe(true)
      expect(evalFuerte.cumpleMinuscula).toBe(true)
      expect(evalFuerte.cumpleNumero).toBe(true)
      expect(evalFuerte.cumpleEspecial).toBe(true)
    })

    it("detecta fallos específicos en requisitos de seguridad", () => {
      // Sin mayúscula
      const sinMayus = evaluarFortalezaPassword(passSinMayus)
      expect(sinMayus.cumpleMayuscula).toBe(false)

      // Sin carácter especial
      const sinEspecial = evaluarFortalezaPassword(passSinSimbolo)
      expect(sinEspecial.cumpleEspecial).toBe(false)

      // Menor a 8 caracteres y solo minúsculas (score 1)
      const muyDebil = evaluarFortalezaPassword("abc")
      expect(muyDebil.cumpleLongitud).toBe(false)
      expect(muyDebil.nivel).toBe("muy_debil")
    })
  })

  describe("Manejo de errores de autenticación y bloqueo por intentos", () => {
    it("identifica error de credenciales incorrectas (401)", () => {
      const err401 = new ErrorApi(
        "sus credenciales de ingreso son incorrectas",
        {
          estado: 401,
          codigo: "credenciales_invalidas",
        },
      )
      expect(err401.esNoAutorizado()).toBe(true)
      expect(err401.esBloqueoIntentos()).toBe(false)
      expect(err401.mensaje).toBe("sus credenciales de ingreso son incorrectas")
    })

    it("identifica error de cuenta bloqueada por 10 minutos (429)", () => {
      const err429 = new ErrorApi(
        "registro fallido, inténtelo nuevamente en 10 minutos",
        {
          estado: 429,
          codigo: "intentos_excedidos",
        },
      )
      expect(err429.esBloqueoIntentos()).toBe(true)
      expect(err429.esNoAutorizado()).toBe(false)
      expect(err429.mensaje).toBe(
        "registro fallido, inténtelo nuevamente en 10 minutos",
      )
    })

    it("calcula intentos restantes progresivamente a partir del segundo fallo", () => {
      function calcularMensajeError(fallos: number): string {
        let mensaje = "sus credenciales de ingreso son incorrectas"
        if (fallos >= 2) {
          const restantes = Math.max(0, 3 - fallos)
          if (restantes > 0) {
            mensaje += `. Le queda ${restantes} intento${restantes > 1 ? "s" : ""}.`
          }
        }
        return mensaje
      }

      expect(calcularMensajeError(1)).toBe(
        "sus credenciales de ingreso son incorrectas",
      )
      expect(calcularMensajeError(2)).toBe(
        "sus credenciales de ingreso son incorrectas. Le queda 1 intento.",
      )
      expect(calcularMensajeError(3)).toBe(
        "sus credenciales de ingreso son incorrectas",
      )
    })
  })
})
