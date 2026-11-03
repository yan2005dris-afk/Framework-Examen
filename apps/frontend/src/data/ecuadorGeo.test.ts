import { describe, expect, it } from "vitest"

import { LISTA_PROVINCIAS, PROVINCIAS_ECUADOR } from "./ecuadorGeo"

const PROVINCIAS_ECUADOR_ESPERADAS = [
  "Azuay",
  "Bolívar",
  "Cañar",
  "Carchi",
  "Chimborazo",
  "Cotopaxi",
  "El Oro",
  "Esmeraldas",
  "Galápagos",
  "Guayas",
  "Imbabura",
  "Loja",
  "Los Ríos",
  "Manabí",
  "Morona Santiago",
  "Napo",
  "Orellana",
  "Pastaza",
  "Pichincha",
  "Santa Elena",
  "Santo Domingo de los Tsáchilas",
  "Sucumbíos",
  "Tungurahua",
  "Zamora Chinchipe",
]

describe("catálogo de provincias del Ecuador", () => {
  it("contiene las 24 provincias del país", () => {
    expect(LISTA_PROVINCIAS).toHaveLength(24)
    expect(LISTA_PROVINCIAS).toEqual(PROVINCIAS_ECUADOR_ESPERADAS)
  })

  it("expone la lista de provincias en orden alfabético en español", () => {
    expect(LISTA_PROVINCIAS).toEqual(
      [...LISTA_PROVINCIAS].sort((a, b) => a.localeCompare(b, "es")),
    )
  })

  it("ordena las provincias con ñ según el alfabeto español", () => {
    // El orden por defecto de Array.sort() coloca "Carchi" antes que "Cañar";
    // en español corresponde "Cañar" antes que "Carchi".
    expect(LISTA_PROVINCIAS.indexOf("Cañar")).toBeLessThan(
      LISTA_PROVINCIAS.indexOf("Carchi"),
    )
  })

  it("tiene al menos un cantón en cada provincia", () => {
    for (const [provincia, cantones] of Object.entries(PROVINCIAS_ECUADOR)) {
      expect(
        cantones.length,
        `la provincia ${provincia} no tiene cantones`,
      ).toBeGreaterThan(0)
    }
  })

  it("no repite cantones dentro de una provincia", () => {
    for (const [provincia, cantones] of Object.entries(PROVINCIAS_ECUADOR)) {
      expect(
        new Set(cantones).size,
        `la provincia ${provincia} repite cantones`,
      ).toBe(cantones.length)
    }
  })

  it("incluye los cantones de las ciudades principales", () => {
    expect(PROVINCIAS_ECUADOR["Pichincha"]).toContain("Quito")
    expect(PROVINCIAS_ECUADOR["Guayas"]).toContain("Guayaquil")
    expect(PROVINCIAS_ECUADOR["Azuay"]).toContain("Cuenca")
  })
})
