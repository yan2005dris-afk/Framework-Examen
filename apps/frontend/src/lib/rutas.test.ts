import { describe, expect, it } from "vitest"

import type { PostulanteDto } from "@/lib/clientes/auth"
import {
  ACCESO_POR_RUTA,
  RUTAS,
  construirRutaLogin,
  esAnalista,
  leerDestinoLogin,
  puedeAcceder,
  rutaDetallePostulacion,
  rutaInicial,
  rutaSeccion,
  type TipoAcceso,
} from "./rutas"

const POSTULANTE_SIN_ROL: PostulanteDto = {
  id: "11111111-1111-7111-8111-111111111111",
  cedula: "0923456789",
  nombres: "Ana",
  apellidos: "Paredes",
  nombre_completo: "Ana Paredes",
  correo: "ana@example.com",
  metodo_validacion: "fecha_expedicion",
  fecha_registro: "2026-10-31T10:00:00Z",
}

const POSTULANTE: PostulanteDto = { ...POSTULANTE_SIN_ROL, rol: "postulante" }

const ANALISTA: PostulanteDto = {
  ...POSTULANTE_SIN_ROL,
  id: "22222222-2222-7222-8222-222222222222",
  cedula: "1712345678",
  nombres: "Luis",
  apellidos: "Mora",
  nombre_completo: "Luis Mora",
  correo: "luis@example.com",
  rol: "analista",
}

describe("Rutas del portal (rutas.ts)", () => {
  describe("esAnalista", () => {
    it("solo concede el rol cuando la sesión lo declara explícitamente", () => {
      expect(esAnalista(ANALISTA)).toBe(true)
      expect(esAnalista(POSTULANTE)).toBe(false)
    })

    it("trata el rol ausente como postulante aunque la API ya lo envíe", () => {
      expect(esAnalista(POSTULANTE_SIN_ROL)).toBe(false)
      expect(esAnalista(null)).toBe(false)
      expect(esAnalista(undefined)).toBe(false)
    })
  })

  describe("puedeAcceder", () => {
    it("deja pasar a cualquier visitante en las rutas públicas", () => {
      expect(puedeAcceder("publico", null)).toBe(true)
      expect(puedeAcceder("publico", POSTULANTE)).toBe(true)
    })

    it("exige sesión en las rutas con sesión", () => {
      expect(puedeAcceder("con-sesion", null)).toBe(false)
      expect(puedeAcceder("con-sesion", POSTULANTE)).toBe(true)
      expect(puedeAcceder("con-sesion", ANALISTA)).toBe(true)
    })

    it("exige rol de analista en el panel de revisión", () => {
      expect(puedeAcceder("con-rol-analista", null)).toBe(false)
      expect(puedeAcceder("con-rol-analista", POSTULANTE)).toBe(false)
      expect(puedeAcceder("con-rol-analista", POSTULANTE_SIN_ROL)).toBe(false)
      expect(puedeAcceder("con-rol-analista", ANALISTA)).toBe(true)
    })
  })

  describe("rutaInicial", () => {
    it("sin sesión apunta al login", () => {
      expect(rutaInicial(null)).toBe(RUTAS.login)
    })

    it("manda al formulario al postulante, tenga o no postulaciones", () => {
      expect(rutaInicial(POSTULANTE)).toBe(RUTAS.postulacion)
      expect(rutaInicial(POSTULANTE_SIN_ROL)).toBe(RUTAS.postulacion)
    })

    it("nunca manda al postulante a una ruta que le da 403", () => {
      for (const postulante of [POSTULANTE, POSTULANTE_SIN_ROL]) {
        expect(puedeAcceder("con-rol-analista", postulante)).toBe(false)
        expect(rutaInicial(postulante)).not.toBe(RUTAS.dashboard)
      }
    })

    it("manda al panel al analista", () => {
      expect(rutaInicial(ANALISTA)).toBe(RUTAS.dashboard)
    })
  })

  describe("construirRutaLogin", () => {
    it("conserva el destino codificado en la query string", () => {
      expect(construirRutaLogin("/dashboard/postulaciones?page=2")).toBe(
        "/login?destino=%2Fdashboard%2Fpostulaciones%3Fpage%3D2",
      )
    })

    it("no arrastra destinos vacíos, el inicio ni el propio login", () => {
      expect(construirRutaLogin()).toBe(RUTAS.login)
      expect(construirRutaLogin("")).toBe(RUTAS.login)
      expect(construirRutaLogin("   ")).toBe(RUTAS.login)
      expect(construirRutaLogin(RUTAS.inicio)).toBe(RUTAS.login)
      expect(construirRutaLogin(RUTAS.login)).toBe(RUTAS.login)
    })
  })

  describe("leerDestinoLogin", () => {
    it("recupera el destino que dejó el guard", () => {
      const busqueda = construirRutaLogin(rutaSeccion(4)).split("?")[1]
      expect(leerDestinoLogin(busqueda)).toBe(rutaSeccion(4))
    })

    it("devuelve el respaldo cuando no hay destino", () => {
      expect(leerDestinoLogin("")).toBe(RUTAS.inicio)
      expect(leerDestinoLogin("?otro=valor", RUTAS.postulacion)).toBe(
        RUTAS.postulacion,
      )
      expect(leerDestinoLogin("?destino=", RUTAS.postulacion)).toBe(
        RUTAS.postulacion,
      )
    })
  })

  describe("helpers de rutas", () => {
    it("arma rutas con parámetros", () => {
      expect(rutaSeccion(4)).toBe("/postulacion/4")
      expect(rutaDetallePostulacion("abc")).toBe("/dashboard/postulaciones/abc")
    })

    it("declara el acceso de cada prefijo del mapa de rutas", () => {
      const esperado: Record<string, TipoAcceso> = {
        [RUTAS.inicio]: "publico",
        [RUTAS.registro]: "publico",
        [RUTAS.login]: "publico",
        [RUTAS.recuperarContrasena]: "publico",
        [RUTAS.postulacion]: "con-sesion",
        [RUTAS.postulacionRevision]: "con-sesion",
        [RUTAS.postulacionImprimir]: "con-sesion",
        [RUTAS.dashboard]: "con-rol-analista",
        [RUTAS.dashboardPostulaciones]: "con-rol-analista",
      }

      expect(ACCESO_POR_RUTA).toEqual(esperado)
    })
  })
})
