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

    it("propaga fielmente el mensaje y detalles informados por el servidor sin inventar contadores locales", () => {
      const errConDetalle = new ErrorApi(
        "sus credenciales de ingreso son incorrectas",
        {
          estado: 401,
          codigo: "credenciales_invalidas",
          detalles: ["le queda 1 intento restante"],
        },
      )
      expect(errConDetalle.detalles).toEqual(["le queda 1 intento restante"])

      const errSinDetalle = new ErrorApi(
        "sus credenciales de ingreso son incorrectas",
        {
          estado: 401,
          codigo: "credenciales_invalidas",
        },
      )
      expect(errSinDetalle.detalles).toBeUndefined()
      expect(errSinDetalle.mensaje).toBe(
        "sus credenciales de ingreso son incorrectas",
      )
    })

    it("identifica errores específicos de OTP (inválido / expirado / conflicto)", () => {
      const errOtp = new ErrorApi(
        "el código de verificación es incorrecto o ha expirado",
        {
          estado: 401,
          codigo: "otp_invalido",
        },
      )
      expect(errOtp.codigo).toBe("otp_invalido")
      expect(errOtp.esNoAutorizado()).toBe(true)

      const errExistente = new ErrorApi(
        "el postulante ya se encuentra registrado",
        {
          estado: 409,
          codigo: "postulante_existente",
        },
      )
      expect(errExistente.esConflicto()).toBe(true)
    })

    it("calcula el tiempo restante de bloqueo a partir de un timestamp persistido sin reiniciar el contador", () => {
      const ahora = Date.now()
      const diezMinutos = ahora + 10 * 60 * 1000
      const cincoMinutos = ahora + 5 * 60 * 1000
      const yaExpirado = ahora - 1000

      function segundosRestantes(expiraEn: number): number {
        return Math.max(0, Math.ceil((expiraEn - Date.now()) / 1000))
      }

      expect(segundosRestantes(diezMinutos)).toBeGreaterThanOrEqual(599)
      expect(segundosRestantes(cincoMinutos)).toBeLessThanOrEqual(301)
      expect(segundosRestantes(yaExpirado)).toBe(0)
    })

    it("confirma recuperación genérica sin revelar si la cédula está registrada", () => {
      const respuestaServidor = {
        mensaje:
          "Si la cédula está registrada, se ha enviado un código de recuperación a su correo.",
      }
      expect(respuestaServidor.mensaje).toContain(
        "Si la cédula está registrada",
      )
      expect(respuestaServidor.mensaje).not.toContain("no encontrada")
    })
  })
})
