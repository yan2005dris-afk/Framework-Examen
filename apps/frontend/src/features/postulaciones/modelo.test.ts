import { describe, expect, it } from "vitest"

import type { PostulacionDto } from "@/lib/clientes/postulaciones"
import {
  debeCrearBorrador,
  quitarPostulacionDePagina,
  resolverPostulacionActual,
} from "./modelo"

function postulacion(id: string, convocatoriaId?: string): PostulacionDto {
  return {
    id,
    applicantId: "cuenta-1",
    code: `EF12-${id}`,
    status: "draft",
    content: {},
    createdAt: "2026-11-01T00:00:00Z",
    updatedAt: "2026-11-01T00:00:00Z",
    progress: { total: 9, completed: 0, percentage: 0, sections: [] },
    convocatoriaId,
  }
}

describe("resolverPostulacionActual", () => {
  const a = postulacion("a", "conv-a")
  const b = postulacion("b", "conv-b")

  it("con convocatoria en la URL abre la postulación de esa convocatoria", () => {
    expect(resolverPostulacionActual("conv-b", [b])).toEqual({
      estado: "encontrada",
      postulacion: b,
      convocatoriaId: "conv-b",
    })
  })

  it("con convocatoria en la URL y sin postulación allí, informa cuál falta", () => {
    expect(resolverPostulacionActual("conv-c", [])).toEqual({
      estado: "sin_postulacion",
      convocatoriaId: "conv-c",
    })
  })

  it("nunca abre la postulación de otra convocatoria", () => {
    expect(resolverPostulacionActual("conv-c", [a]).estado).toBe(
      "sin_postulacion",
    )
  })

  it("sin parámetro y con una sola postulación, usa esa y su convocatoria", () => {
    expect(resolverPostulacionActual(undefined, [a])).toEqual({
      estado: "encontrada",
      postulacion: a,
      convocatoriaId: "conv-a",
    })
  })

  it("sin parámetro y con varias, pide elegir en vez de tomar la primera", () => {
    expect(resolverPostulacionActual(undefined, [a, b])).toEqual({
      estado: "elegir",
      postulaciones: [a, b],
    })
  })

  it("sin parámetro y sin postulaciones, no hay convocatoria resuelta", () => {
    expect(resolverPostulacionActual(undefined, [])).toEqual({
      estado: "sin_postulacion",
    })
  })
})

describe("debeCrearBorrador", () => {
  const base = {
    convocatoriaId: "conv-a",
    abierta: true,
    creacionLibre: true,
    borradorDescartado: false,
  }

  it("crea el borrador de una convocatoria abierta sin postulación", () => {
    expect(debeCrearBorrador(base)).toBe(true)
  })

  it("no lo recrea después de descartarlo", () => {
    expect(debeCrearBorrador({ ...base, borradorDescartado: true })).toBe(false)
  })

  it("no crea sin convocatoria, con la convocatoria cerrada ni con otra creación en curso", () => {
    expect(debeCrearBorrador({ ...base, convocatoriaId: undefined })).toBe(
      false,
    )
    expect(debeCrearBorrador({ ...base, abierta: false })).toBe(false)
    expect(debeCrearBorrador({ ...base, creacionLibre: false })).toBe(false)
  })
})

describe("quitarPostulacionDePagina", () => {
  const a = postulacion("a", "conv-a")
  const b = postulacion("b", "conv-b")

  it("quita la postulación eliminada de una página o un cursor", () => {
    expect(
      quitarPostulacionDePagina(
        { items: [a, b], hasMore: false, limit: 50 },
        "a",
      ),
    ).toEqual({ items: [b], hasMore: false, limit: 50 })
    expect(
      quitarPostulacionDePagina(
        { items: [a], page: 1, pageSize: 10, total: 1 },
        "a",
      ).items,
    ).toEqual([])
  })

  it("devuelve los mismos datos si no la contiene o no es una página", () => {
    const pagina = { items: [b], hasMore: false, limit: 50 }
    expect(quitarPostulacionDePagina(pagina, "a")).toBe(pagina)
    expect(quitarPostulacionDePagina(a, "a")).toBe(a)
    expect(quitarPostulacionDePagina(undefined, "a")).toBeUndefined()
  })
})
