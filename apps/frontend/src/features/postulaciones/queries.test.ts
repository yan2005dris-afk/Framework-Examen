import { describe, expect, it } from "vitest"

import { postulacionKeys } from "./queries"

describe("postulacionKeys", () => {
  it("genera claves estables para listados, detalles y catálogos", () => {
    expect(postulacionKeys.todas).toEqual(["postulaciones"])
    expect(postulacionKeys.listas()).toEqual(["postulaciones", "lista"])
    expect(postulacionKeys.lista({ page: 1, pageSize: 10 })).toEqual([
      "postulaciones",
      "lista",
      { page: 1, pageSize: 10 },
    ])
    expect(postulacionKeys.detalles()).toEqual(["postulaciones", "detalle"])
    expect(postulacionKeys.detalle("123-abc")).toEqual([
      "postulaciones",
      "detalle",
      "123-abc",
    ])
    expect(postulacionKeys.miPostulacion()).toEqual([
      "postulaciones",
      "mi-postulacion",
    ])
    expect(postulacionKeys.definicion()).toEqual(["formulario-definicion"])
    expect(postulacionKeys.paises()).toEqual(["catalogos", "paises"])
    expect(postulacionKeys.provincias("Ecuador")).toEqual([
      "catalogos",
      "provincias",
      "Ecuador",
    ])
    expect(postulacionKeys.cantones("Guayas", "Ecuador")).toEqual([
      "catalogos",
      "cantones",
      "Ecuador",
      "Guayas",
    ])
  })
})
