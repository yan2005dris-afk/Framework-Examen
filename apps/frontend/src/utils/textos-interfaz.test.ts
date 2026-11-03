import ts from "typescript"
import { describe, expect, it } from "vitest"

// Se analizan nodos de texto, no comentarios ni nombres internos del contrato.
const fuentes = import.meta.glob<string>(
  ["../**/*.tsx", "../**/*.ts", "!../**/*.test.ts", "!../components/ui/**"],
  { query: "?raw", import: "default", eager: true },
)
const referenciaInterna =
  /\b(?:EF[-‐‑–]?\d+(?:\.\d+)*\b(?!-\d{4}-)|EF\d+-TECNICA-v\d+|FT\d+(?:-FT?\d+)?|T[1-9]\d*|HU[-‐‑–]\d+|RNF?[-‐‑–]\d+)\b/i

function buscarReferencias(codigo: string, archivo: string): string[] {
  const fuente = ts.createSourceFile(
    archivo,
    codigo,
    ts.ScriptTarget.Latest,
    true,
    archivo.endsWith(".tsx") ? ts.ScriptKind.TSX : ts.ScriptKind.TS,
  )
  const hallazgos: string[] = []
  function visitar(nodo: ts.Node) {
    if (
      ts.isJsxText(nodo) ||
      ts.isStringLiteralLike(nodo) ||
      nodo.kind === ts.SyntaxKind.TemplateHead ||
      nodo.kind === ts.SyntaxKind.TemplateMiddle ||
      nodo.kind === ts.SyntaxKind.TemplateTail
    ) {
      const texto = (nodo as ts.JsxText | ts.StringLiteralLike).text
      if (referenciaInterna.test(texto)) {
        const { line } = fuente.getLineAndCharacterOfPosition(nodo.getStart())
        hallazgos.push(`${archivo}:${line + 1}: ${texto.trim()}`)
      }
    }
    ts.forEachChild(nodo, visitar)
  }
  visitar(fuente)
  return hallazgos
}

describe("Textos de interfaz sin referencias al plan de desarrollo", () => {
  it("revisa títulos, mensajes, etiquetas accesibles y textos de todas las vistas", () => {
    expect(Object.keys(fuentes).length).toBeGreaterThan(50)
    const hallazgos = Object.entries(fuentes).flatMap(([archivo, codigo]) =>
      buscarReferencias(codigo, archivo),
    )
    expect(hallazgos).toEqual([])
  })

  it("detecta referencias en JSX, atributos, mensajes y plantillas", () => {
    const codigo = `
      const mensaje = "Pendiente T27"
      const nombre = \`matriz-EF13-TECNICA-v1\`
      const vista = <section aria-label="Formulario EF-12"><h1>Resultados (FT23)</h1><p>Imprimir (HU-11)</p></section>
    `
    expect(buscarReferencias(codigo, "ejemplo.tsx")).toHaveLength(5)
  })

  it("conserva identificadores, comentarios y códigos de expediente", () => {
    const codigo = `
      // T24: mantener el contrato EF-12 del servidor.
      const ef12Disponible = true
      const codigoPostulacion = "EF12-2026-004821"
      const fecha = "2026-01-01T00:00:00Z"
    `
    expect(buscarReferencias(codigo, "ejemplo.ts")).toEqual([])
  })
})
