import { describe, expect, it } from "vitest"

import type { PostulacionDto } from "@/lib/clientes/postulaciones"
import { resolverPostulacionActual } from "./modelo"

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
