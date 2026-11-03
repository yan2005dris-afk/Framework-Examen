# ADR-009 — Paleta institucional del BIESS extraída del CSS del portal oficial

| Campo | Valor |
|---|---|
| Estado | Aceptada |
| Fecha | 2026-10-31 |
| Ámbito | Sistema de diseño del frontend y del set de mockups |
| Fuente | Hoja de estilos del portal oficial `biess.fin.ec` (31-10-2026) |
| Referencia | [`docs/design/PALETA-BIESS.md`](../design/PALETA-BIESS.md) |

## Contexto

El repositorio ya declaraba una paleta "institucional" en los tokens del set de
mockups, documentada como `#0A3871` / `#072146` del portal IESS/BIESS. Ese par
no aparece en el CSS del portal del BIESS: son valores tomados de otra fuente
(navy de la interfaz del IESS) y nunca se verificaron contra
`biess.fin.ec`, que es el sitio que la aplicación simula. Mientras tanto el
frontend usaba la paleta neutra por defecto de shadcn, con `--primary` casi
negro.

El resultado eran tres paletas coexistiendo —neutra en la app, navy no
verificada en los mockups, y ninguna institucional— y ninguna fuente de
verdad para el color.

## Decisión

- La paleta de la aplicación y del set de mockups se extrae del CSS del portal
  oficial del BIESS (`/common/themes/biess/style.css`), no de una impresión ni
  de otra institution. La tabla completa, con el uso de cada color en el
  portal y el contraste verificado, está en
  [`docs/design/PALETA-BIESS.md`](../design/PALETA-BIESS.md).
- La identidad se compone de **azul institucional**: `#0060A9` en barras y
  cabeceras, `#003A66` en navegación, `#005490` en el botón principal,
  `#0097CC` en la marca del producto.
- El amarillo `#FFD200` del portal se conserva como token disponible
  (`--brand-yellow`), pero **no se aplica a la interfaz**: los títulos y las
  cabeceras usan el azul institucional. Es una decisión de diseño del equipo
  (el remarko amarillo en los encabezados se consideróVisualmente pesado).
- Los estados de éxito y advertencia se derivan de los verdes y ámbar del
  portal, oscurecidos para cumplir 4.5:1 como texto: los originales son
  fondos de mensaje y no llegan al mínimo.
- El modo oscuro no se invierte: es una derivación de la misma paleta (azul
  `#003A66` hacia el negro), porque el portal oficial no publica una variante
  oscura.
- `apps/frontend/src/index.css` y `docs/mockups/assets/tokens.css` declaran los
  mismos valores, token por token. Ninguno de los dos admite hexadecimales
  literales: se expresan en `oklch()` y el hex original queda en el comentario,
  como fuente de trazabilidad.
- La aplicación es un **simulacro**: adopta la paleta como identidad visual,
  sin replicar el logotipo ni las marcas registradas del BIESS.

## Alternativas descartadas

- **Mantener la paleta neutra de shadcn.** Implicaba que el simulacro no se
  pareciera al sitio al que simula, y dejaba el token `--primary` como casi
  negro mientras el portal usa azul.
- **Conservar `#0A3871` / `#072146`.** No están en el CSS de `biess.fin.ec`; son
  el navy de otra interfaz del IESS. Mantenerlos habría dejado la paleta sin
  fuente verificable, que es el problema que esta decisión resuelve.
- **Usar el amarillo como `--accent` (hover de menús y listas).** Satura la
  interfaz: el amarillo del portal se usa en títulos, no en cada estado hover.
- **Pintar los títulos con el amarillo del portal.** El portal sí lo hace en sus
  `h3`/`h4`, pero el remarko-yellow sobre los encabezados de las tarjetas se
  vio pesado en la interfaz y no se mantiene. El token queda disponible.
- **Replicar el markup del portal.** El portal es HTML clásico con CSS de 2010;
  la app es React con shadcn/ui. Se toma el color, no la estructura.

## Consecuencias y pendientes

El re-skin es centralizado: los componentes consumen tokens semánticos, así que
la paleta se cambia en dos archivos. Los mockups y el frontend quedan alineados
y el contrato de clases del set (`app.css`) no cambia.

Los tokens que el set de mockups declara y el frontend todavía no expone
(`--on-primary`, `--brand-200`, `--brand-300`, `--brand-ink`, los de
impresión y los `--*-strong`) siguen pendientes de pasar al frontend; sus
valores ya son coherentes con la paleta BIESS.

El amarillo queda sin uso en la interfaz: si en el futuro se quiere aplicar
—por ejemplo a un chip de estado o a una marca— los tokens
`--brand-yellow`, `--brand-yellow-soft` y `--on-brand-yellow` ya están
declarados en los dos archivos.

`--brand-cyan` (`#0097CC`) queda como acento y para series de gráficas: como
color de texto de cuerpo no alcanza el 4.5:1.
