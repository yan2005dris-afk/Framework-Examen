import { describe, expect, it } from "vitest"

import definicion from "../../../../backend/internal/submission/definition.json"
import { aplicarValoresIniciales, consentimientoParaEnvio } from "./modelo"
import { validarSeccionEF12 } from "./validacion"

describe("aplicarValoresIniciales", () => {
  it("completa el país y la nacionalidad de un borrador recién creado", () => {
    const creado = {
      seccion1: { cedula: "1710034008", nacionalidad: "" },
      seccion2: { correo1: "postulante@biess.local", pais: "" },
    }
    const resultado = aplicarValoresIniciales(creado)
    expect(resultado.seccion1).toMatchObject({ nacionalidad: "Ecuatoriana" })
    expect(resultado.seccion2).toMatchObject({
      pais: "Ecuador",
      correo1: "postulante@biess.local",
    })
  })

  it("nunca reemplaza un valor ya elegido", () => {
    const resultado = aplicarValoresIniciales({
      seccion2: { pais: "Colombia" },
    })
    expect(resultado.seccion2).toEqual({ pais: "Colombia" })
  })

  it("no inventa secciones que no llegaron ni modifica el original", () => {
    const original = { seccion3: { tieneFamiliaresIessBiess: "No" } }
    const resultado = aplicarValoresIniciales(original)
    expect(resultado).toEqual(original)
    expect(resultado).not.toBe(original)
  })

  it("la sección 2 resultante pasa la validación sin volver a elegir el país", () => {
    const { seccion2 } = aplicarValoresIniciales({
      seccion2: { correo1: "postulante@biess.local", pais: "" },
    })
    expect(validarSeccionEF12(seccion2, 2).esValido).toBe(true)
  })
})

describe("consentimientoParaEnvio", () => {
  const camposContrato = Object.keys(definicion.tipos.Consent)
  const enPantalla = {
    aceptacionTratamientoDatos: true,
    declaracionVeracidad: true,
    aceptacionNormasConcurso: true,
    autorizacionPublicacionNombre: true,
    constanciaEntregaFisica: true,
    versionPolitica: 2,
    referenciaAprobacion: "Resolución de prueba",
  }

  it("envía sólo campos del contrato Consent del servidor", () => {
    const enviado = consentimientoParaEnvio(enPantalla, {
      id: "politica-1",
      hash: "b".repeat(64),
    })
    for (const campo of Object.keys(enviado))
      expect(camposContrato, campo).toContain(campo)
    expect(enviado).toMatchObject({
      politicaPrivacidadId: "politica-1",
      hashPoliticaPrivacidad: "b".repeat(64),
    })
  })

  it("sin política vigente envía sólo los cinco puntos", () => {
    expect(Object.keys(consentimientoParaEnvio(enPantalla))).toHaveLength(5)
  })
})
