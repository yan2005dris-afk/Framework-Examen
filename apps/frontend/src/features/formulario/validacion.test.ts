import { describe, expect, it } from "vitest"
import compartidos from "../../../../backend/internal/submission/testdata/form_cases.json"
import type { DocumentoDto } from "@/lib/clientes/documentos"
import { referenciaDocumento } from "./modelo"
import {
  validarCedulaPareja,
  validarCorreoSecundario,
  validarFormularioEF12,
  validarSeccionEF12,
} from "./validacion"

interface Caso {
  nombre: string
  completo: boolean
  valido: boolean
  borrador?: boolean
  contenido?: unknown
  parches?: { ruta: string; valor: unknown }[]
  borrar?: string[]
}

function aplicarCambio(
  base: unknown,
  ruta: string,
  valor: unknown,
  borrar = false,
) {
  const partes = ruta.split(".")
  let actual = base as Record<string, unknown>
  for (const parte of partes.slice(0, -1))
    actual = actual[parte] as Record<string, unknown>
  const campo = partes.at(-1)!
  if (borrar) delete actual[campo]
  else actual[campo] = valor
}

describe("paridad Go/TypeScript del EF-12", () => {
  const casos: Caso[] = compartidos.casos
  for (const caso of casos) {
    it(caso.nombre, () => {
      const contenido: unknown = Object.hasOwn(caso, "contenido")
        ? caso.contenido
        : structuredClone(caso.borrador ? {} : compartidos.base)
      for (const cambio of caso.parches ?? [])
        aplicarCambio(contenido, cambio.ruta, cambio.valor)
      for (const ruta of caso.borrar ?? [])
        aplicarCambio(contenido, ruta, undefined, true)
      const resultado = validarFormularioEF12(
        contenido,
        caso.completo,
        new Date(compartidos.ahora),
      )
      expect(resultado.esValido, JSON.stringify(resultado.errores)).toBe(
        caso.valido,
      )
    })
  }

  it("valida una sección completa sin exigir las otras ocho ni el consentimiento", () => {
    expect(
      validarSeccionEF12(
        compartidos.base.seccion1,
        1,
        true,
        new Date(compartidos.ahora),
      ).esValido,
    ).toBe(true)
    expect(validarSeccionEF12({}, 10).esValido).toBe(false)
  })
})

describe("validaciones de campo extraídas de las secciones", () => {
  it("no exige cédula de la pareja si el campo está vacío", () => {
    expect(validarCedulaPareja(undefined)).toBeNull()
    expect(validarCedulaPareja("")).toBeNull()
  })

  it("rechaza una cédula de la pareja que no cumple el módulo 10", () => {
    const resultado = validarCedulaPareja("1710034009")

    expect(resultado?.esValida).toBe(false)
    expect(resultado?.error).toContain("dígito verificador")
  })

  it("acepta una cédula de la pareja válida", () => {
    expect(validarCedulaPareja("1710034008")?.esValida).toBe(true)
  })

  it("no exige correo de confirmación si el campo está vacío", () => {
    expect(validarCorreoSecundario(undefined)).toBeNull()
    expect(validarCorreoSecundario("")).toBeNull()
  })

  it("rechaza un correo de confirmación con formato inválido", () => {
    expect(validarCorreoSecundario("correo sin arroba")).toBe("Correo inválido")
  })

  it("acepta un correo de confirmación válido", () => {
    expect(validarCorreoSecundario("maria.fernandez@example.com")).toBeNull()
  })
})

describe("referencia de respaldo PDF tras la subida", () => {
  const subido: DocumentoDto = {
    id: "01a19cfc-42aa-79ef-a877-28f0a8a7fbe3",
    submissionId: "01a19ce2-6213-7dc3-b8ec-7a7a8da4a62c",
    section: 4,
    recordId: "",
    name: "respaldo-prueba.pdf",
    mime: "application/pdf",
    sizeBytes: 965,
    sha256: "a".repeat(64),
    createdAt: "2026-11-02T16:04:18Z",
  }
  const seccion4 = {
    sector: "afiliados",
    cuentaConRespaldoOrganizacion: "Sí",
    nombreOrganizacion: "Organización de prueba",
  }

  it("cumple el contrato: la sección se puede guardar después de subir el PDF", () => {
    const resultado = validarSeccionEF12(
      { ...seccion4, documentoRespaldoPdf: referenciaDocumento(subido) },
      4,
    )
    expect(resultado.esValido, JSON.stringify(resultado.errores)).toBe(true)
  })

  it("un campo fuera del contrato impide guardar la sección", () => {
    const resultado = validarSeccionEF12(
      {
        ...seccion4,
        documentoRespaldoPdf: {
          ...referenciaDocumento(subido),
          sizeBytes: subido.sizeBytes,
        },
      },
      4,
    )
    expect(resultado.esValido).toBe(false)
  })
})
