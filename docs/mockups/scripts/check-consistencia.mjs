#!/usr/bin/env node
/**
 * check-consistencia.mjs — Verifica que las pantallas del set rendericen
 * exactamente los valores del fixture único y que el set esté enlazado.
 *
 * Uso:  node docs/mockups/scripts/check-consistencia.mjs
 * Salida: 0 si todo coincide, 1 con la lista de discrepancias.
 *
 * El fixture `fixtures/ef12-2026-004821.json` es la fuente de verdad de los
 * datos de negocio que aparecen en más de una pantalla. Si una pantalla
 * cambia un valor, este script falla: hay que actualizar el fixture, no la
 * pantalla por su cuenta.
 *
 * La lista de pantallas se deriva del directorio, no de una constante: al
 * agregar o quitar una pantalla no hay nada que sincronizar aquí.
 */

import { readFileSync, readdirSync, existsSync } from "node:fs"
import { dirname, join } from "node:path"
import { fileURLToPath } from "node:url"

const raiz = join(dirname(fileURLToPath(import.meta.url)), "..")
const fixturePath = join(raiz, "fixtures", "ef12-2026-004821.json")

if (!existsSync(fixturePath)) {
  console.error(`No se encuentra el fixture: ${fixturePath}`)
  process.exit(1)
}

const fixture = JSON.parse(readFileSync(fixturePath, "utf8"))
const problemas = []

/** Todas las pantallas del set, incluidas en el índice. */
const pantallas = readdirSync(raiz)
  .filter((f) => f.endsWith(".html"))
  .sort()

const htmlPorPantalla = new Map(
  pantallas.map((p) => [p, readFileSync(join(raiz, p), "utf8")]),
)

/** Valores que no pueden aparecer en ninguna pantalla (obsoletos). */
const obsoletos = [
  {
    valor: "1713849201",
    motivo: "cedula invalida con el algoritmo modulo 10 (el fixture usa 1713849204)",
  },
  {
    valor: "1712345678",
    motivo: "cedula de la pareja invalida (el fixture usa 1712345675)",
  },
  {
    valor: "d4a1f8",
    motivo: "hash de otro expediente (el fixture usa 9f2c41b7)",
  },
  {
    valor: "BIESS-CPMO-2026-00428",
    motivo: "codigo de postulacion anterior (el fixture usa EF12-2026-004821)",
  },
  {
    valor: "clave-incorrecta",
    motivo: "la contrasena no se precarga en el DOM",
  },
]

/** 1. Cada pantalla del fixture debe contener sus valores esperados. */
for (const [pantalla, esperados] of Object.entries(fixture.pantallas)) {
  const html = htmlPorPantalla.get(pantalla)
  if (html === undefined) {
    problemas.push(`${pantalla}: no existe en el set`)
    continue
  }
  for (const valor of esperados) {
    if (!html.includes(valor)) {
      problemas.push(`${pantalla}: falta el valor del fixture "${valor}"`)
    }
  }
}

/** 2. Ninguna pantalla puede contener un valor obsoleto. */
for (const [pantalla, html] of htmlPorPantalla) {
  for (const { valor, motivo } of obsoletos) {
    if (html.includes(valor)) {
      problemas.push(`${pantalla}: contiene el valor obsoleto "${valor}" (${motivo})`)
    }
  }
}

/** 3. La fecha de envio debe ser una sola en todo el set. */
const fechas = new Set()
for (const html of htmlPorPantalla.values()) {
  for (const m of html.matchAll(/\b(\d{2}\/\d{2}\/\d{4})\s+(\d{2}:\d{2})\b/g)) {
    fechas.add(m[0])
  }
}
const fechaEsperada = `${fixture.envio.fecha}`
if (fechas.size > 0 && !fechas.has(fechaEsperada)) {
  problemas.push(
    `fecha de envio incoherente: se encontro ${[...fechas].join(", ")} y el fixture fija ${fechaEsperada}`,
  )
}

/** 4. Todo enlace interno debe apuntar a un archivo que existe. */
for (const [pantalla, html] of htmlPorPantalla) {
  for (const m of html.matchAll(/(?:href|src)="([^"]+)"/g)) {
    const destino = m[1]
    if (/^(https?:|mailto:|tel:|#|data:)/.test(destino)) continue
    const limpio = destino.split("#")[0].split("?")[0]
    if (limpio === "") continue
    if (!existsSync(join(raiz, limpio))) {
      problemas.push(`${pantalla}: enlace roto -> ${destino}`)
    }
  }
}

/** 5. Toda pantalla debe estar enlazada desde el indice (sin pantallas huerfanas). */
const indice = htmlPorPantalla.get("index.html") ?? ""
const sinEnlace = pantallas.filter(
  (p) => p !== "index.html" && !indice.includes(`"${p}"`),
)
for (const p of sinEnlace) {
  problemas.push(`index.html: la pantalla ${p} no está enlazada desde el índice`)
}

/** 6. Sin estilos ni scripts en línea: el set es HTML plano + assets/mock.js. */
for (const [pantalla, html] of htmlPorPantalla) {
  if (/\sstyle="/.test(html)) {
    problemas.push(`${pantalla}: atributo style en línea (los estilos van en app.css)`)
  }
  if (/<script(?![^>]*\bsrc=)[^>]*>[\s\S]*?\S[\s\S]*?<\/script>/.test(html)) {
    problemas.push(`${pantalla}: script en línea (solo se permite assets/mock.js)`)
  }
  if (/\son[a-z]+="/.test(html)) {
    problemas.push(`${pantalla}: manejador de evento en línea`)
  }
  if (/href="javascript:/.test(html)) {
    problemas.push(`${pantalla}: enlace javascript: (no hay lógica fuera de mock.js)`)
  }
}

/** 7. Toda pantalla necesita un enlace de vuelta al indice. */
for (const [pantalla, html] of htmlPorPantalla) {
  if (pantalla === "index.html") continue
  if (!/Galería/.test(html)) {
    problemas.push(`${pantalla}: falta el enlace de vuelta a la galería`)
  }
}

if (problemas.length > 0) {
  console.error(`check-consistencia: ${problemas.length} discrepancia(s)\n`)
  for (const p of problemas) console.error(`  - ${p}`)
  process.exit(1)
}

console.log(
  `check-consistencia: OK — ${pantallas.length} pantallas (${Object.keys(fixture.pantallas).length} con valores del fixture) coinciden con ${fixture.codigoUnico}`,
)
