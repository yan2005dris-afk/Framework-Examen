# Paleta institucional del BIESS

Referencia de los colores que la aplicación usa y su origen exacto en el portal
oficial. Es el documento que acompaña a [ADR-009](../decisions/ADR-009-paleta-institucional-biess.md).

## Origen

| Dato | Valor |
|---|---|
| Sitio | <https://www.biess.fin.ec/inicio> |
| Hoja de estilos | `https://www.biess.fin.ec/css.axd?css=/common/themes/biess/style.css` |
| Fecha de extracción | 2026-10-31 |
| Formato en el repo | `oklch()` — el verificador `check-contrato.mjs` rechaza hexadecimales literales en `tokens.css` |

El portal sirve su CSS con la extensión `css.axd`, que concatena varias hojas
(`reset-fonts-grids.css` y `base.css` además de `style.css`). Todos los colores
de este documento salen de `style.css`.

La aplicación **no es** el portal del BIESS: es un simulacro. Se toma la paleta
como identidad visual del simulacro, no el código, las marcas registradas ni el
logotipo oficial del BIESS (que no se replica en la interfaz).

## Colores extraídos y su uso en el portal

| Hex | Uso en `biess.fin.ec` | Token en la app |
|---|---|---|
| `#0060A9` | Cabeceras de tabla, títulos de banner, barras de sección | `--primary` |
| `#005490` | Botón principal (`.lnkGeneral`, `.lnkCredit`) | `--primary-hover` |
| `#005FA9` | Color de enlace (módulos, breadcrumb, FAQ) | — (el portal lo usa como enlace; la app usa `--primary`) |
| `#0071C4` | Botón de enviar y de contacto | `--ring` (foco) |
| `#008EF8` | Glosario: letras inactivas | `--primary-hover` en modo oscuro |
| `#0097CC` | Cifras de estadísticas y día de las fechas | `--brand-cyan` |
| `#1c94c4` | Enlace del datepicker de jQuery UI | `--chart-2` |
| `#003A66` | Menú desplegable, hover del menú, glosario, cabecera de revista | `--brand-900`, `--sidebar` |
| `#24567C` | Separador del menú desplegable | `--sidebar-accent`, `--sidebar-border`, `--chart-4` |
| `#FFD200` | Títulos `h3`/`h4`, cabecera de formulario, pestaña activa | `--brand-yellow` |
| `#FFCC33` | Borde del bloque «importante» | — |
| `#FFCC00` | Título de la galería | — |
| `#F27130` | Subrayado del módulo naranja de la portada | — |
| `#B0D86E` | Fondo del mensaje «confirmado» | `--success` (oscurecido) / `--success-soft` |
| `#FFC95F` | Fondo del mensaje «advertencia» | `--warning` (oscurecido) / `--warning-soft` |
| `#FF8484` | Fondo del mensaje «error» | `--destructive` en modo oscuro |
| `#FFAAAA` | Borde del mensaje «error» | — |
| `#FFD9D9` | Fondo de un campo con error | `--destructive-soft` |
| `#6ACFFF` | Fondo del mensaje «información» | `--brand-cyan` en modo oscuro |
| `#E20E15` | Punto rojo de la firma del pie | `--destructive` |
| `#333333` | Color de enlace y cuerpo de texto | `--foreground` |
| `#666666` | Texto secundario, migas de pan | `--muted-foreground` |
| `#999999` | Enlaces deshabilitados, descargas | — |
| `#CCCCCC` | Borde de los campos de formulario | `--input` |
| `#E0E0E0` | Borde de los bloques | `--border` |
| `#F0F0F0` | Filas alternas de tabla, separadores | `--secondary` |
| `#F1F1F1` | Pie de página | `--muted` |
| `#525252` | Texto gris medio | `--chart-5` |

## Derivados

El portal usa algunos colores solo como fondo. Para texto sobre fondo blanco se
aplicaron las mezclas siguientes (en OKLab, contra blanco):

| Token | Derivado de | Mezcla | Hex | Contraste sobre blanco |
|---|---|---|---|---|
| `--brand-100` | `#0060A9` | 12 % | `#E3ECF6` | 1.19:1 (fondo, no texto) |
| `--accent` | `#0060A9` | 10 % | `#E7EFF7` | 1.16:1 (fondo, no texto) |
| `--brand-yellow-soft` | `#FFD200` | 22 % | `#FFF6D9` | 1.08:1 (fondo, no texto) |
| `--success-soft` | `#B0D86E` | 25 % | `#EBF6DD` | 1.12:1 (fondo, no texto) |
| `--success` | `#B0D86E` | 60 % | `#566B33` | **5.92:1** |
| `--warning-soft` | `#FFC95F` | 30 % | `#FFF0D3` | 1.13:1 (fondo, no texto) |
| `--warning` | `#FFC95F` | 55 % | `#715826` | **6.72:1** |

`--success` y `--warning` son los únicos dos tokens donde el color del portal
**no** se usa tal cual: el verde `#B0D86E` (1.63:1) y el ámbar `#FFC95F` (1.52:1)
del sitio son fondos de mensaje, y como texto no llegarían al 4.5:1 de WCAG AA.

## Contraste verificado (WCAG 2.2)

| Par | Ratio | Umbral |
|---|---|---|
| `#333333` sobre `#FFFFFF` | 12.63:1 | AA (4.5:1) |
| `#666666` sobre `#FFFFFF` | 5.74:1 | AA (4.5:1) |
| `#0060A9` sobre `#FFFFFF` | 6.47:1 | AA (4.5:1) |
| `#FFFFFF` sobre `#0060A9` | 6.47:1 | AA (4.5:1) |
| `#FFFFFF` sobre `#003A66` | 11.69:1 | AA (4.5:1) |
| `#000000` sobre `#FFD200` | 14.47:1 | AAA (7:1) |
| `#0071C4` sobre `#FFFFFF` | 5.05:1 | AA (4.5:1) |
| `#E20E15` sobre `#FFFFFF` | 4.88:1 | AA (4.5:1) |
| `#566B33` sobre `#EBF6DD` | 5.29:1 | AA (4.5:1) |
| `#715826` sobre `#FFF0D3` | 5.97:1 | AA (4.5:1) |
| `#0097CC` sobre `#FFFFFF` | 3.33:1 | Solo texto grande (3:1) y elementos gráficos |

`#0097CC` y `#1c94c4` quedan reservados a cifras grandes, series de gráficas y
bordes: no deben usarse como color de texto de cuerpo.

En modo oscuro, los pares equivalentes se verificaron sobre `#00050E` y
`#000D1D`: texto 14.3:1, primario 6.1:1, amarillo 14.1:1, verde 12.5:1, ámbar
13.4:1 y rojo 8.6:1.

## Modo oscuro

El portal oficial no publica una variante oscura. El bloque `.dark` de la app
es una **derivación** de la misma paleta (azul `#003A66` mezclado hacia el
negro), no una inversión de los tokens claros: se conserva el amarillo y el
celeste, y el azul principal sube a `#0097CC` para mantener el contraste sobre
fondo oscuro.

## Dónde se aplica cada cosa

| Elemento | Token | Referencia en el portal |
|---|---|---|
| Barra de navegación lateral | `--sidebar: #003A66` | Menú superior azul marino |
| Marca del producto (escudo) | `--sidebar-primary: #0097CC` | Cifras destacadas del portal |
| Botón primario | `--primary` / `--primary-hover` | `.lnkGeneral` / `.lnkCredit` |
| Encabezado de tabla | `--primary` + `--primary-foreground` | `thead` de `.tabla-agencias` |
| Enlace dentro de la app | `--primary` | Enlaces `#005FA9` |
| Estado correcto | `--success` sobre `--success-soft` | Mensaje `.confirm` |
| Estado de error | `--destructive` sobre `--destructive-soft` | Mensaje `.errormsg` |

El amarillo `#FFD200` **no se aplica a la interfaz**: queda disponible como
token (`--brand-yellow`, `--brand-yellow-soft`, `--on-brand-yellow`) porque
forma parte de la paleta del portal, pero los títulos y las cabeceras de la
app usan el azul institucional y el navy. Decisión de diseño del equipo, no
una lectura de la fuente.

## Dónde viven los valores

| Archivo | Contenido |
|---|---|
| `apps/frontend/src/index.css` | Tokens del frontend + su exposición a Tailwind v4 (`@theme inline`) |
| `docs/mockups/assets/tokens.css` | Los mismos valores para el set de mockups (fuente única de color del set) |

Los dos archivos se mantienen en el mismo valor token por token. El script
`docs/mockups/scripts/check-contrato.mjs` verifica que `tokens.css` no
introduzca colores literales; la paridad entre ambos archivos es una
responsabilidad de revisión, no una comprobación automática.
