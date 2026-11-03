import { createElement } from "react"
import { renderToStaticMarkup } from "react-dom/server"
import { describe, expect, it, vi } from "vitest"
import casos from "../../../../backend/internal/submission/testdata/form_cases.json"
import {
  confirmarEdicionRegistro,
  duracionExperienciaImprimible,
} from "./modelo"
import { validarRegistroFormulario } from "./validacion"
import type { Seccion5Data } from "./seccion-5-academica"
import type { Seccion6Data } from "./seccion-6-experiencia"
import { Seccion5Academica } from "./seccion-5-academica"
import { Seccion6Experiencia } from "./seccion-6-experiencia"
import { Seccion7Publicaciones } from "./seccion-7-publicaciones"
import { Seccion8Capacitacion } from "./seccion-8-capacitacion"
import { HojaPostulacion } from "./hoja-postulacion"
import type { PostulacionDto } from "@/lib/clientes/postulaciones"
import { calcularDuracionExperiencia } from "@/helpers/experiencia"

describe("confirmación de registros del modal", () => {
  const original = {
    id: "uno",
    titulo: "Original",
    documentoPdf: { id: "pdf", nombre: "respaldo.pdf" },
  }
  it("no muta la lista ni el respaldo al editar", () => {
    const lista = [structuredClone(original)]
    const borrador = structuredClone(original)
    borrador.titulo = "Actualizado"
    const salida = confirmarEdicionRegistro(lista, borrador, original)
    expect(lista[0]).toEqual(original)
    expect(salida[0].titulo).toBe("Actualizado")
    expect(salida[0].documentoPdf).toEqual(original.documentoPdf)
    borrador.documentoPdf.nombre = "Cambio posterior"
    expect(salida[0].documentoPdf.nombre).toBe("respaldo.pdf")
  })
  it("agrega un registro sólo al confirmar y conserva los demás", () => {
    const nuevo = { ...original, id: "dos" }
    expect(confirmarEdicionRegistro([original], nuevo, null)).toEqual([
      original,
      nuevo,
    ])
  })
  it("rechaza un registro eliminado, alterado o un ID nuevo duplicado", () => {
    expect(() => confirmarEdicionRegistro([], original, original)).toThrow(
      /cambió/,
    )
    expect(() =>
      confirmarEdicionRegistro(
        [{ ...original, titulo: "Otra pestaña" }],
        original,
        original,
      ),
    ).toThrow(/cambió/)
    expect(() => confirmarEdicionRegistro([original], original, null)).toThrow(
      /cambió/,
    )
  })
})

describe("validación de registros sin exigir PDF antes de su carga", () => {
  const publicacion = {
    id: "p",
    titulo: "Publicación de prueba",
    tipo: "Publicación",
    medio: "Revista de prueba",
    fechaPublicacion: "2020-01-01",
    relacionCampo: "Sí",
  }
  const capacitacion = {
    id: "c",
    nombre: "Curso de prueba",
    tipo: "Curso",
    institucion: "Institución de prueba",
    fechaInicio: "2020-01-01",
    fechaFin: "2020-01-02",
    horas: 8,
  }
  for (const [numero, registro] of [
    [5, casos.base.seccion5.titulos[0]],
    [6, casos.base.seccion6.experiencias[0]],
    [7, publicacion],
    [8, capacitacion],
  ] as const) {
    it(`acepta campos completos de sección ${numero} y rechaza vacíos`, () => {
      expect(validarRegistroFormulario(numero, registro)).toBeNull()
      expect(validarRegistroFormulario(numero, { id: "nuevo" })).not.toBeNull()
    })
  }
  it("rechaza rangos invertidos y horas no positivas", () => {
    expect(
      validarRegistroFormulario(8, { ...capacitacion, fechaFin: "2019-01-01" }),
    ).not.toBeNull()
    expect(
      validarRegistroFormulario(8, { ...capacitacion, horas: 0 }),
    ).not.toBeNull()
  })
})

describe("tablas compactas y hoja inmutable", () => {
  it("las cuatro secciones no muestran editores en línea al abrir", () => {
    const vistas = [
      createElement(Seccion5Academica, {
        datos: {
          ...casos.base.seccion5,
          titulos: casos.base.seccion5.titulos.map((t) => ({
            ...t,
            tipo: "Nacional",
          })),
        } satisfies Seccion5Data,
        onChange: vi.fn(),
      }),
      createElement(Seccion6Experiencia, {
        datos: {
          ...casos.base.seccion6,
          experiencias: casos.base.seccion6.experiencias.map((e) => ({
            ...e,
            tipo: "General",
          })),
        } satisfies Seccion6Data,
        onChange: vi.fn(),
      }),
      createElement(Seccion7Publicaciones, {
        datos: { cuentaConPublicaciones: "Sí", publicaciones: [] },
        onChange: vi.fn(),
      }),
      createElement(Seccion8Capacitacion, {
        datos: casos.base.seccion8,
        onChange: vi.fn(),
      }),
    ]
    for (const vista of vistas) {
      const html = renderToStaticMarkup(vista)
      expect(html).toContain("<table")
      expect(html).not.toContain("<input")
    }
  })
  it("la duración de un empleo vigente queda cortada al envío en Ecuador", () => {
    vi.useFakeTimers()
    try {
      const registro = { fechaInicio: "2019-01-01", actualmenteLaborando: true }
      vi.setSystemTime(new Date("2030-01-01T00:00:00Z"))
      const a = duracionExperienciaImprimible(registro, "2020-01-01T03:00:00Z")
      vi.setSystemTime(new Date("2040-01-01T00:00:00Z"))
      expect(
        duracionExperienciaImprimible(registro, "2020-01-01T03:00:00Z"),
      ).toBe(a)
      expect(a).toBe(
        calcularDuracionExperiencia("2019-01-01", "2019-12-31", false).texto,
      )
      expect(duracionExperienciaImprimible(registro)).toBe(
        "Fecha de envío no disponible",
      )
    } finally {
      vi.useRealTimers()
    }
  })
  it("imprime las nueve secciones, cinco aceptaciones y hash recibido, sin cambiar el snapshot", () => {
    const documento: PostulacionDto = {
      id: "s",
      applicantId: "a",
      code: "POSTULACION-PRUEBA",
      status: "submitted",
      content: structuredClone(casos.base),
      hash: "a".repeat(64),
      submittedAt: "2026-10-31T21:00:00Z",
      createdAt: "2026-10-30T00:00:00Z",
      updatedAt: "2026-10-31T21:00:00Z",
      progress: { completed: 9, total: 9, percentage: 100, sections: [] },
    }
    const antes = structuredClone(documento)
    const html = renderToStaticMarkup(
      createElement(HojaPostulacion, { postulacion: documento }),
    )
    expect(html.match(/class="print-block"/g)).toHaveLength(10)
    expect(html).toContain("a".repeat(64))
    expect(html).toContain("POSTULACION-PRUEBA")
    expect(html).toContain("Firma manuscrita del postulante")
    expect(html).not.toContain("Resolución CD-BIESS")
    expect(documento).toEqual(antes)
  })
})
