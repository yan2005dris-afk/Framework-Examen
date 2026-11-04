import { QueryClient } from "@tanstack/react-query"
import { describe, expect, it } from "vitest"

import type { PostulacionDto } from "@/lib/clientes/postulaciones"
import type { PaginaCursor } from "@/types/api"
import { resolverPostulacionActual } from "./modelo"
import { olvidarPostulacion, postulacionKeys } from "./queries"

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

function borrador(id: string, convocatoriaId: string): PostulacionDto {
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

describe("olvidarPostulacion", () => {
  // Reproduce el fallo de «Descartar borrador»: con la copia vieja en caché, la
  // pantalla volvía a encontrar el borrador eliminado, fijaba su convocatoria en
  // la URL y terminaba creando otro.
  it("la postulación eliminada deja de resolverse desde la caché", () => {
    const cliente = new QueryClient()
    const descartado = borrador("descartado", "conv-a")
    const otra = borrador("otra", "conv-b")
    const pagina = (items: PostulacionDto[]): PaginaCursor<PostulacionDto> => ({
      items,
      hasMore: false,
      limit: 50,
    })
    cliente.setQueryData(
      postulacionKeys.actual(undefined),
      pagina([descartado]),
    )
    cliente.setQueryData(postulacionKeys.actual("conv-a"), pagina([descartado]))
    cliente.setQueryData(postulacionKeys.actual("conv-b"), pagina([otra]))
    cliente.setQueryData(postulacionKeys.detalle("descartado"), descartado)

    const cacheada = cliente.getQueryData<PaginaCursor<PostulacionDto>>(
      postulacionKeys.actual(undefined),
    )
    expect(
      resolverPostulacionActual(undefined, cacheada?.items ?? []).estado,
    ).toBe("encontrada")

    olvidarPostulacion(cliente, "descartado")

    const sinParametro = cliente.getQueryData<PaginaCursor<PostulacionDto>>(
      postulacionKeys.actual(undefined),
    )
    expect(
      resolverPostulacionActual(undefined, sinParametro?.items ?? []),
    ).toEqual({
      estado: "sin_postulacion",
    })
    expect(
      cliente.getQueryData<PaginaCursor<PostulacionDto>>(
        postulacionKeys.actual("conv-a"),
      )?.items,
    ).toEqual([])
    expect(
      cliente.getQueryData<PaginaCursor<PostulacionDto>>(
        postulacionKeys.actual("conv-b"),
      )?.items,
    ).toEqual([otra])
    expect(
      cliente.getQueryData(postulacionKeys.detalle("descartado")),
    ).toBeUndefined()
  })
})
