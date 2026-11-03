#!/usr/bin/env node
/**
 * check-contrato.mjs — Comprueba el contrato de migración del set:
 *   · toda clase usada en el HTML existe en assets/app.css
 *   · no hay identificadores duplicados dentro de una pantalla
 *   · cada control de formulario tiene etiqueta asociada
 *   · cada tabla declara <th scope>
 *   · el enlace de salto y <html lang="es"> están presentes
 *
 * Uso: node docs/mockups/scripts/check-contrato.mjs
 */

import { readFileSync, readdirSync } from "node:fs"
import { dirname, join } from "node:path"
import { fileURLToPath } from "node:url"

const raiz = join(dirname(fileURLToPath(import.meta.url)), "..")
const css = readFileSync(join(raiz, "assets", "app.css"), "utf8")
const pantallas = readdirSync(raiz)
  .filter((f) => f.endsWith(".html"))
  .sort()

/** Clases declaradas en app.css (selector simple, sin_chain ni pseudo). */
const clasesCss = new Set()
for (const m of css.matchAll(/\.(-?[_a-zA-Z][\w-]*)/g)) clasesCss.add(m[1])

/** Clases que no son de app.css pero forman parte del contrato de shadcn/Tailwind. */
const permitidas = new Set([
  "sr-only",
  "dark",
  "group",
  "peer",
])

const problemas = []

for (const pantalla of pantallas) {
  const html = readFileSync(join(raiz, pantalla), "utf8")

  // 1. Clases inexistentes
  const usadas = new Set()
  for (const m of html.matchAll(/class="([^"]+)"/g)) {
    for (const c of m[1].trim().split(/\s+/)) if (c) usadas.add(c)
  }
  for (const c of usadas) {
    if (!clasesCss.has(c) && !permitidas.has(c)) {
      problemas.push(`${pantalla}: clase "${c}" no existe en app.css`)
    }
  }

  // 2. Identificadores duplicados
  const ids = [...html.matchAll(/\sid="([^"]+)"/g)].map((m) => m[1])
  const repetidos = ids.filter((id, i) => ids.indexOf(id) !== i)
  for (const id of [...new Set(repetidos)]) {
    problemas.push(`${pantalla}: id duplicado "${id}"`)
  }

  // 3. Controles de formulario sin etiqueta
  // Un <label> que envuelve al control ya lo asocia (asociación implícita).
  for (const m of html.matchAll(/<(input|select|textarea)\b([^>]*)>/g)) {
    const attrs = m[2]
    const tipo = (attrs.match(/type="([^"]+)"/) || [])[1] || ""
    if (["hidden", "submit", "button", "reset"].includes(tipo)) continue
    const id = (attrs.match(/\sid="([^"]+)"/) || [])[1]
    const aria = (attrs.match(/aria-label(?:ledby)?="([^"]+)"/) || [])[1]
    // ¿El control está dentro de un <label>…</label>? (asociación implícita)
    const antes = html.slice(0, m.index)
    const abreLabel = antes.lastIndexOf("<label")
    const cierraLabel = antes.lastIndexOf("</label>")
    const dentroDeLabel = abreLabel > cierraLabel
    const tieneLabel =
      (id && html.includes(`for="${id}"`)) ||
      (aria && ids.includes(aria)) ||
      dentroDeLabel
    if (!tieneLabel) {
      problemas.push(`${pantalla}: control <${m[1]}> sin etiqueta asociada`)
    }
  }

  // 4. Cabeceras de tabla con scope
  for (const m of html.matchAll(/<th\b([^>]*)>/g)) {
    if (!/\bscope=/.test(m[1])) {
      problemas.push(`${pantalla}: <th> sin atributo scope`)
    }
  }

  // 5. Idioma y enlace de salto
  if (!/<html[^>]+lang="es"/.test(html)) {
    problemas.push(`${pantalla}: falta lang="es"`)
  }
  if (!/class="skip-link"/.test(html) && !/href="#main"/.test(html)) {
    problemas.push(`${pantalla}: falta el enlace de salto al contenido`)
  }
}

// 6. app.css no debe tener colores literales (todo pasa por tokens.css).
// Los comentarios se quitan antes: mentioning un color en una nota no es usarlo.
const sinComentarios = (texto) => texto.replace(/\/\*[\s\S]*?\*\//g, "")

const literales = [...sinComentarios(css).matchAll(/#[0-9a-fA-F]{3,8}\b/g)].map((m) => m[0])
if (literales.length > 0) {
  problemas.push(
    `app.css: ${literales.length} color(es) literal(es) fuera de tokens.css: ${[...new Set(literales)].join(", ")}`,
  )
}

// 7. tokens.css tampoco: sus valores deben ser oklch() u otra función de color.
const tokens = readFileSync(join(raiz, "assets", "tokens.css"), "utf8")
const literalesTokens = [
  ...sinComentarios(tokens).matchAll(/:\s*#[0-9a-fA-F]{3,8}\b/g),
]
if (literalesTokens.length > 0) {
  problemas.push(`tokens.css: ${literalesTokens.length} color(es) literal(es)`)
}

if (problemas.length > 0) {
  console.error(`check-contrato: ${problemas.length} problema(s)\n`)
  for (const p of problemas) console.error(`  - ${p}`)
  process.exit(1)
}

console.log(
  `check-contrato: OK — ${pantallas.length} pantallas cumplen el contrato (${clasesCss.size} clases en app.css, 0 colores literales)`,
)
