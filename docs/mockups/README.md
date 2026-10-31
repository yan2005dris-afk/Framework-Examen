# Set de mockups — Portal de Postulación BIESS/IESS

Este directorio es la **referencia de diseño y de contrato de migración** para las
vistas del frontend en React 19 + Tailwind v4 + shadcn/ui. Son HTML estático: se
abre `index.html` en el navegador, sin servidor, sin build y sin dependencias.

Son **36 pantallas** más el índice. Todas comparten un solo expediente de ejemplo
(`EF12-2026-004821`), de modo que ningún dato de negocio se contradice entre
pantallas.

- Fuente funcional: `docs/requerimiento.txt` (RF v1.4 del IESS).
- Contrato HTTP: `docs/api-contracts/API_CONTRACT.md`.
- Historias de usuario: `docs/prd/PRD.md` (HU-01 … HU-12).
- Tokens: `assets/tokens.css`. Clases: `assets/app.css`. Interactividad: `assets/mock.js`.

## Reglas del set

1. **Los nombres de clase de `assets/app.css` son el contrato de migración.** Cada
   clase corresponde a un componente de shadcn/ui (ver el mapa de abajo). No se
   inventan clases ni estilos: lo que no está en `app.css` no existe.
2. **Cero colores literales fuera de `assets/tokens.css`.** Las pantallas solo
   consumen `var(--token)`.
3. **Cero lógica de negocio.** Los mockups no validan, no calculan y no guardan.
   Las únicas interacciones son las dos mejoras progresivas de `assets/mock.js`:
   `[data-password-toggle]` (mostrar/ocultar contraseña) y
   `[data-dialog-open]` / `[data-dialog-close]` (diálogos de demo).
4. **Accesibilidad por defecto:** `<html lang="es">`, enlace de salto, `<label for>`
   en cada control, `<th scope>` en las tablas y botones con texto visible.
5. **Formato:** 2 espacios, comillas dobles, sin punto y coma.
6. El avance del wizard usa una sola regla en todo el set: `Avance: N de 9
   secciones` con N = secciones **completadas**, y `data-level="N"` con N entre 0
   y 9 (el ancho de la barra se deriva de ese valor sobre 9).
7. **Un solo expediente de ejemplo.** Todos los datos de negocio (código, hash,
   fecha de envío, adjuntos, registros, familiares, capacitación y las 13
   respuestas de inhabilidades) viven en `fixtures/ef12-2026-004821.json`. Para
   comprobar que ninguna pantalla se contradiga:
   `node scripts/check-consistencia.mjs`.
8. Los **identificadores de catálogo** (`EF-12 · 1.1.6`, `HU-03`) no son estados:
   llevan la clase `tag` (mono, gris), nunca `badge`. Un `badge` significa
   estado de negocio y solo hay cuatro: success, warning (accionable),
   info y danger.

## Mapa pantalla → historia de usuario → componente shadcn

### Acceso y registro (EF-11)

| Pantalla | HU | Componentes shadcn |
|---|---|---|
| `01-portal.html` | HU-01, HU-03 | Button, Card, Badge, Separator, Tabs |
| `02a-registro-identidad.html` | HU-01 | Input, Label, RadioGroup, FieldSet, Alert, Button |
| `02b-registro-credenciales.html` | HU-01 | Input, Label, Checkbox, Progress, Alert, Button |
| `02c-registro-exito.html` | HU-01 | Alert, Button, Card |
| `02d-registro-bloqueado.html` | HU-02 | Alert, Table, Button |
| `03a-login.html` | HU-03 | Input, Label, Checkbox, Button |
| `03b-login-error.html` | HU-03 | Alert, Input, Button |
| `04a-recuperar-identidad.html` | HU-04 | Input, Label, RadioGroup, Button |
| `04b-recuperar-correo-enviado.html` | HU-04 | Alert, Card, Button |
| `04c-recuperar-nueva-contrasena.html` | HU-04 | Input, Label, Progress, Button |
| `04d-recuperar-exito.html` | HU-04 | Alert, Button |

### Panel del postulante y formulario EF-12

| Pantalla | HU | Componentes shadcn |
|---|---|---|
| `05-panel-postulante.html` | HU-05, HU-09 | Card, Badge, Progress, Table, Button |
| `06-01-informacion-personal.html` | HU-05, HU-06 | Input, Label, Select, FieldSet, Alert, Button |
| `06-01b-nacionalidad-no-ecuatoriana.html` | HU-06 | Alert, Card, Button |
| `06-02-domicilio-contacto.html` | HU-05, HU-06 | Select, Input, Label, Alert, Button |
| `06-03-informacion-adicional.html` | HU-05, HU-07 | Input, Label, Select, Table, Button |
| `06-04-seleccion-cargo.html` | HU-08 | RadioGroup, Input, Label, Alert, Button |
| `06-04b-cargo-no-elegible.html` | HU-08 | Alert, Card, Button |
| `06-05-formacion-academica.html` | HU-05, HU-07 | Select, Input, Label, Table, Button |
| `06-06-experiencia-profesional.html` | HU-05, HU-07 | Select, Input, Textarea, Label, Table, Button |
| `06-07-publicaciones.html` | HU-05, HU-07 | Select, Input, Label, Table, Button |
| `06-08-capacitacion.html` | HU-05, HU-07 | Select, Input, Label, Table, Button |
| `06-09-inhabilidades.html` | HU-05 | RadioGroup, Input, Label, FieldSet, Button |
| `06-anexos-modales.html` | HU-07, HU-10 | Dialog, Input, Label, Alert, Button, Table |
| `07-revision-consentimiento.html` | HU-09, HU-10 | Table, Checkbox, Alert, Button |
| `07b-revision-documentos-pendientes.html` | HU-07, HU-09 | Table, Badge, Alert, Button |
| `08-confirmacion-envio.html` | HU-10 | Card, Badge, Alert, Button |
| `09-imprimir.html` | HU-11 | Card, Table, Separator |

### Administración (EF-13) y estados del sistema

| Pantalla | HU | Componentes shadcn |
|---|---|---|
| `10a-panel-admin.html` | HU-12 | Card, Statistic, Table, Badge, Button |
| `10b-postulaciones.html` | HU-12 | Input, Select, Table, Button, Pagination |
| `10b-postulaciones-vacia.html` | HU-12 | Card, Button, EmptyState |
| `10c-detalle-postulacion.html` | HU-12 | Card, Table, Button |
| `10d-matriz.html` | HU-12 | Table, Button |
| `11a-403.html` | HU-12 | Alert, Button |
| `11b-404.html` | HU-12 | Alert, Button |
| `11c-sesion-expirada.html` | HU-12 | Alert, Button |

## Tokens

`assets/tokens.css` declara **68 tokens** (`:root` + variante `.dark`). El frontend
(`apps/frontend/src/index.css`) usa los mismos nombres y los expone a utilidades
Tailwind v4 con `@theme inline { --color-X: var(--X) }`.

**Compartidos con el frontend (45):** `--background`, `--foreground`, `--card`,
`--card-foreground`, `--popover`, `--popover-foreground`, `--primary`,
`--primary-foreground`, `--primary-hover`, `--secondary`, `--secondary-foreground`,
`--muted`, `--muted-foreground`, `--accent`, `--accent-foreground`, `--destructive`,
`--destructive-soft`, `--border`, `--input`, `--ring`, `--radius`, `--radius-sm`,
`--radius-md`, `--radius-lg`, `--radius-xl`, `--font-sans`, `--font-mono`,
`--shadow-sm`, `--shadow-md`, `--shadow-lg`, `--space-1`, `--space-2`, `--space-3`,
`--space-4`, `--space-5`, `--space-6`, `--space-8`, `--space-10`, `--space-12`,
`--space-16`, `--brand-100`, `--brand-900`, `--success`, `--success-soft`,
`--warning`, `--warning-soft`.

**Solo en el set (23):** `--on-primary`, `--on-foreground`, `--on-foreground-muted`,
`--on-brand-900`, `--brand-200`, `--brand-300`, `--brand-ink`, `--success-strong`,
`--success-on-dark`, `--warning-strong`, `--destructive-strong`,
`--destructive-hover`, `--muted-foreground-strong`, `--overlay` y los nueve de
impresión (`--print-ink`, `--print-ink-strong`, `--print-heading`, `--print-rule`,
`--print-table-head`, `--print-table-foot`, `--print-paper`, `--print-folio`).
Están declarados en `assets/tokens.css` y quedan pendientes de pasar al frontend:
son los que usan las cabeceras oscuras, los estados "sobre fondo de color" y la
hoja imprimible.

Este set usa la paleta institucional del BIESS (`#0A3871` / `#072146`); el frontend
sigue en la paleta neutral actual. Adoptar la paleta institucional es una decisión
de marca de una sola edición, no de cada componente.

## Verificación

Hay dos scripts, sin dependencias (solo Node):

```bash
node scripts/check-consistencia.mjs   # los datos del expediente y los enlaces
node scripts/check-contrato.mjs      # el contrato de migración a shadcn/ui
```

`check-consistencia.mjs` valida el set completo y falla si algo se desincroniza.
La lista de pantallas se deriva del directorio, no de una constante. Comprueba:

1. que cada pantalla con datos de negocio contenga exactamente los valores del
   fixture;
2. que ningún archivo contenga un valor obsoleto (cédulas inválidas, el hash del
   expediente anterior, el código anterior, la contraseña precargada);
3. que exista una sola fecha de envío en todo el set;
4. que no haya enlaces internos rotos;
5. que toda pantalla esté enlazada desde `index.html` (sin pantallas huérfanas);
6. que no haya estilos, scripts, manejadores ni enlaces `javascript:` en línea;
7. que toda pantalla tenga su enlace de vuelta a la galería.

`check-contrato.mjs` vigila el contrato de la regla 1: toda clase usada en el
HTML existe en `assets/app.css`, no hay identificadores duplicados, cada control
tiene etiqueta (explícita o por asociación implícita), cada `<th>` declara
`scope`, toda pantalla declara `lang="es"` y su enlace de salto, y ni `app.css` ni
`tokens.css` tienen colores literales.

## Cobertura

**Cubierto por el set:** las 9 secciones del EF-12 con su stepper y su estado;
registro, bloqueo, login y recuperación (EF-11) con los literales del RF; hash
SHA-256 del contenido canónico, código único, envío sellado e impresión con firma
manuscrita; adjuntos PDF con sus límites; catálogo completo de países y provincias;
estado terminal por nacionalidad (1.1.6) y por no elegibilidad de sector (1.4.1.2);
dashboard de administración con matriz y exportación; los estados de formulario
foco, error, deshabilitado, cargando y vacío.

**Estados representados:** el expediente de ejemplo `EF12-2026-004821` aparece en dos
momentos distintos, y ambos son coherentes entre sí:

| Momento | Pantallas | Secciones | Adjuntos |
|---|---|---|---|
| Borrador en curso | `05-panel-postulante` | 4 de 9 | 6 · 14,2 MB de 50 MB |
| Expediente enviado | `07`, `07b`, `08`, `09`, `10b`, `10c`, `10d` | 9 de 9 | 8 · 18,4 MB de 50 MB |

Las pantallas `06-01` … `06-09` documentan **la primera pasada** por el asistente:
la sección N muestra N−1 secciones completadas, y el paso N aparece marcado como
«Completada» a partir de `06-02`. El panel `05` es la foto de ese mismo recorrido
**detrás de la sección 4**, así que su CTA «Continuar en la sección 5» reanuda donde
corresponde en lugar de devolver al inicio.

`07b` es la variante con el envío bloqueado por dos documentos obligatorios
pendientes (RF 1.6.8 y 1.7.1.6). `10b-postulaciones-vacia` es el estado vacío del
listado de administración.

**Inhabilidades:** el postulante responde "Sí" en 1.9.1 (afirmación de
elegibilidad), 1.9.7 (removido por un Organismo de Control) y 1.9.12
(responsabilidades con la Contraloría). El RF **no** declara descalificación
automática: la matriz de administración muestra "2 declaradas · 1.9.7, 1.9.12" y
el dictamen "Observado", y el detalle del expediente reproduce el detalle
declarado de la 1.9.7.

**Impresión:** la hoja de `09-imprimir.html` ocupa 4 páginas A4. El folio se genera
en el margin box de `@page` (`@bottom-center`), que es el único contexto donde los
contadores `page` y `pages` devuelven el número real; un `position: fixed` con
`counter(page)` queda en «Página 0 de 0».

**Responsive y objetivos táctiles:** por debajo de 60 rem el stepper deja de flotar
y pasa a ser una tira horizontal desplazable, de modo que no tapa los campos de la
sección. Las tablas anchas viven dentro de `.table-wrap` (con scroll horizontal) y
los controles de opción miden 24 × 24 px para cumplir el objetivo mínimo de
WCAG 2.2 (SC 2.5.8); los enlaces que viven dentro de una frase quedan cubiertos
por la excepción «inline» de la misma norma.

**Fuera del set (pendiente de diseño):** las fechas del proceso y los plazos, que
el RF deja a la resolución de la Comisión; la duración del enlace de recuperación y
el tiempo de espera entre reenvíos, que ninguna fuente del proyecto fija; y la
"Fase 3 · Entrevista", que aparece en el panel pero no figura en el RF v1.4.
