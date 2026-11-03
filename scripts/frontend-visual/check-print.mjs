/** Contrato estático del papel: Vitest omite por defecto el contenido de CSS. */
import { readFileSync } from "node:fs";
import assert from "node:assert/strict";
const raiz = new URL("../../", import.meta.url);
const leer = (ruta) => readFileSync(new URL(ruta, raiz), "utf8");
const actual = leer("apps/frontend/src/index.css");
const referencia = leer("docs/mockups/assets/tokens.css");
const css = leer("apps/frontend/src/impresion.css");
const tokens = [...referencia.matchAll(/(--print-[\w-]+):\s*([^;]+);/g)].slice(
  0,
  8,
);
assert.equal(tokens.length, 8);
for (const [, token, valor] of tokens)
  assert.ok(actual.includes(`${token}: ${valor};`), token);
for (const regla of [
  "size: A4",
  "margin: 18mm 16mm",
  "counter(page)",
  "counter(pages)",
])
  assert.ok(css.includes(regla), regla);
console.log(
  "Impresión: 8 tokens idénticos al mockup, A4, márgenes y folio por página verificados.",
);
