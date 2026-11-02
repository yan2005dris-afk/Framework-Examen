import { describe, expect, it } from "vitest"

import { crearQueryClient, vaciarDatosDeSesion } from "./query-client"

describe("vaciarDatosDeSesion", () => {
  it("descarta los datos de la cuenta anterior", () => {
    const cliente = crearQueryClient()
    cliente.setQueryData(["postulaciones", "actual", null], {
      items: [{ code: "EF12-2026-CUENTA-A" }],
    })
    cliente.setQueryData(["catalogos", "paises"], ["Ecuador"])

    vaciarDatosDeSesion(cliente)

    expect(
      cliente.getQueryData(["postulaciones", "actual", null]),
    ).toBeUndefined()
    expect(cliente.getQueryCache().getAll()).toHaveLength(0)
  })
})
