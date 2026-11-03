# Alineación visual del frontend con los mockups

**2026-11-03.** Se alineó la estructura compartida del portal con
`docs/mockups`: cabeceras, navegación, ancho de contenido, acceso, formulario y
administración. También se completaron las tablas con modales de las secciones 5–8,
las excepciones de registro/nacionalidad/sector y la composición A4. Es una entrega de alineación visual, **no una declaración de
coincidencia píxel a píxel de los 36 estados**.

## Cambios visibles

| Referencia | Aplicación |
|---|---|
| Cabecera y pie del set | `CabeceraPortal` / `PiePortal`: franja institucional, marca tipográfica, enlaces horizontales activos, cuenta y salida. Sin buscador ficticio ni iniciales fijas |
| 03a, 02a, 04a | Acceso, registro y recuperación con tarjeta de 46 rem, alineación izquierda, bordes suaves y campos de 40 px; sin centrado vertical que desplazaba el formulario |
| 06-01…06-09 | Navegación lateral de 17 rem, separación de 2 rem y contenido flexible. A menos de 60 rem cambia a tira horizontal desplazable, sin tapar los campos |
| 10a…10d | Administración con navegación superior, estadísticas, acciones y tablas. Se conserva gestión de convocatorias/fases/banco en una segunda fila para administradores |
| 10d | Vista previa paginada de matriz con formación, experiencia al envío, publicaciones y horas provenientes de los resúmenes de la API. Sin inventar puntajes, dictámenes o sumas de experiencia solapada |
| 11a/11b y errores recuperables | Marco común, tarjeta y encabezado semántico. Los guards y las decisiones de acceso no cambiaron |
| Componentes compartidos | `.portal-shell` armoniza tarjetas, controles y tablas. Los controles de diálogos portados a `body` también conservan su tamaño. No se editaron primitivas de `components/ui/` |

El tema inicial es claro como el set; se conserva la selección guardada del
usuario y la opción de tema oscuro/sistema. Los ajustes de componentes se limitan
a pantalla, salvo la hoja A4 descrita más abajo. El generador PDF del backend
no se modificó.

## Se conserva el comportamiento real

- OTP por correo y datos personales declarados: no se copió el texto antiguo del
  mockup que atribuye una validación al Registro Civil.
- Convocatoria en la URL, filtros y paginación; caché privada por cuenta/rol.
- Guardado por sección, documentos, cuota, restricciones de fase, envío y hash.
- Exportaciones de la selección completa, diferenciadas de la vista previa
  paginada. Las estadísticas completas permanecen en el dashboard.
- Ningún dato de ejemplo se incorporó al frontend de producción ni como fallback
  ante errores de la API. No se modificaron datos o código del backend.

## Verificación

| Comprobación | Evidencia local |
|---|---|
| Pruebas frontend | 413 aprobadas en 38 archivos; incluye edición sin mutaciones, conflictos locales, validación de registros, excepciones y snapshot imprimible |
| Calidad | Lint, build y formato aprobados |
| Referencias estáticas | Ambos verificadores del set aprobados: 37 documentos incluyendo el índice |
| Navegador de escritorio | Acceso, registro, recuperación, 404, nueve secciones, dashboard, nómina y matriz revisados con datos sintéticos |
| Ancho móvil | Nueve secciones en iframe de 390 px: `scrollWidth = clientWidth = 375 px` (15 px corresponden a la barra de desplazamiento); sin desborde horizontal del documento |
| Integridad de la vista previa | Servidor sólo en loopback, cabecera `X-Data-Origin: synthetic-visual-only`, mutaciones rechazadas con 405 salvo un rechazo 429 sintético de registro; no crea cuentas ni roles reales |

Se corrigió un desborde móvil provocado por texto `sr-only` absolutamente
posicionado fuera del contenedor desplazable. El menú ahora establece su propio
contexto de posición. El ajuste de viewport del navegador no se aplicó, por eso
se verificó el breakpoint con un iframe de ancho real conocido; no se presenta
como prueba sobre un teléfono físico.

Capturas, **sólo presentación con fixtures**, no evidencia de operaciones reales:

- [Acceso alineado](evidence/acceso-alineado-2026-11-03.jpg)
- [Matriz alineada](evidence/matriz-alineada-2026-11-03.jpg)

## Repetir la revisión sin tocar expedientes

Terminal 1:

```powershell
node scripts/frontend-visual/serve-fixtures.mjs --rol=postulante
```

Terminal 2:

```powershell
$env:VITE_DEV_API_TARGET="http://127.0.0.1:5180"
pnpm --dir apps/frontend dev --host 127.0.0.1 --port 5175 --strictPort
```

Abrir `/postulacion/1?convocatoria=019a4000-0000-7000-8000-000000000001`.
La revisión móvil está en `http://127.0.0.1:5180/__qa/mobile`.
Para administración, reiniciar sólo el servidor de fixtures con
`--rol=administrador` y recargar la página. También admite `analista` y `anonimo`.
No dejar esa variable apuntando a fixtures al volver al trabajo contra la API;
no usar este servidor en producción. No prueba login, guardado, envío ni SMTP.

## Tablas y modales completados

Las secciones de formación, experiencia, publicaciones y capacitación muestran
una tabla compacta y acciones Agregar, Editar, Eliminar y Adjuntar PDF. El diálogo
valida un registro completo sin exigir un PDF previo: primero tiene que existir el
registro al que se vincula el documento. Cancelar, Escape y el cierre del editor
no escriben en el formulario; confirmar sólo actualiza el estado local.
**Guardar borrador** conserva la sección en la API, como antes.

El respaldo tiene un diálogo separado, según el anexo de modales del set. Cargar
o quitar un PDF sí ejecuta la operación existente del servidor y se avisa de ello;
cerrar no revierte una operación confirmada. Se mantienen identificadores,
cuota, límite de 2 MB, permisos y descargas existentes. La eliminación de un
registro es local hasta guardar y no llama silenciosamente al borrado del PDF.
Declarar que no hay publicaciones no borra una lista existente.

Verificado en navegador con fixtures: cuatro diálogos, cancelar título y
experiencia, rechazar un título vacío, guardar título/publicación/horas de
capacitación, abrir respaldo y cancelar eliminación. No se subieron documentos
ni se guardó/envió un expediente real. La entrada nativa de fecha por el adaptador
del navegador no actualizó React al usar `fill`; no se contabiliza como alta
completa probada en navegador. Los casos completos se validaron con pruebas
unitarias y se probaron ediciones sobre registros fechados del fixture.

## Excepciones completadas

| Referencia | Comportamiento real conservado |
|---|---|
| 02d | Pantalla dedicada con cuenta regresiva y barra. Usa `Retry-After`; sólo el vencimiento se conserva en `sessionStorage`, sin cédula, correo ni OTP. Recargar no renueva el plazo. La API mantiene la autorización final |
| 06-01b | La selección extranjera muestra una advertencia diferenciada del cierre. Volver a editar conserva la declaración; Siguiente permanece bloqueado hasta corregirla |
| 06-04b | Un estado `closed` recibido de la API muestra una pantalla propia sin acciones de reapertura; reproduce motivo/origen del servidor y distingue la simulación controlada |

Se comprobaron los dos cierres y la advertencia local en navegador. El escenario
429 mostró 10:00 y, tras recargar, 09:53: no volvió a diez minutos. No se inventan
historiales de intentos, certificaciones de Registro Civil ni plazos de reclamo.

## Hoja A4 completada

`HojaPostulacion` reproduce el formulario consultado con cabecera partida,
datos en dos columnas, tablas compactas, trece preguntas, cinco declaraciones,
firma y hash completo. Respeta los ocho tokens de papel del set, A4 y márgenes
18 × 16 mm; el folio se genera en el margin box de `@page`. Cabecera del portal,
navegación, botones y avisos no se imprimen. La duración de trabajos vigentes se
corta a la fecha civil de envío en Ecuador, no al día de impresión.

**Verificación local:** Chrome headless 156 generó un PDF de **3 páginas A4**
(594.96 × 841.92 pt) con un registro por cada sección de méritos. Se renderizaron
las tres páginas con Poppler y se inspeccionaron todas: sin páginas vacías,
recortes visibles ni hash truncado; firma y hash permanecen juntos. Una primera
prueba detectó una página blanca al terminar el elemento con página nombrada:
se corrigió asignando la página a todo el cuerpo de esta vista, no sólo al artículo.
El número de páginas depende del contenido; no se fuerza el número del expediente
ilustrativo ni se alteran los literales para hacerlos caber.

```powershell
node scripts/frontend-visual/check-print.mjs
```

Este verificador comprueba tokens, A4, márgenes y contadores sin depender del
procesamiento de CSS de Vitest. Los PDFs/perfiles temporales permanecen ignorados
en `apps/frontend/test-results/print-qa/`.

Capturas locales de **datos sintéticos**, no evidencia de aceptación institucional.
`docs/design/evidence/` está ignorado por Git; las imágenes se entregan localmente,
no se añaden al historial del repositorio:

- [Modal de formación](evidence/formacion-modal-2026-11-03.jpg)
- [Sector no elegible](evidence/sector-no-elegible-2026-11-03.jpg)
- [Bloqueo temporal](evidence/bloqueo-registro-2026-11-03.jpg)
- [A4, primera página](evidence/impresion-a4-p1-2026-11-03.png)
- [A4, firma y hash](evidence/impresion-a4-p3-2026-11-03.png)

Para reproducir cada estado, reiniciar **sólo** el servidor de fixtures con
`--escenario=borrador|enviada|nacionalidad|sector|bloqueo`. En bloqueo usar
`--rol=anonimo`; sólo `POST /api/v1/auth/register/request` devuelve un 429
controlado, nunca crea la cuenta ni envía correo. El resto de mutaciones sigue
rechazado. Cada escenario es independiente del backend real.

## Límites de la comprobación

- No se afirma regresión visual automatizada de los 36 estados, accesibilidad
  completa ni impresión idéntica en todos los navegadores/impresoras.
- Se midieron las cuatro tablas en iframe de 390 px: documento de 375 px,
  contenedor desplazable de 292 px y tabla de 640 px, sin desborde del documento; el adaptador
  no pudo accionar los botones dentro de ese iframe. La interacción completa de
  modales en un teléfono físico queda para QA manual, no se presenta como aprobada.
- El backend, SMTP, carga real de documentos y persistencia no se revalidaron en
  esta entrega visual. Se conservan los callbacks y contratos existentes.
- La matriz usa campos del contrato, no un dictamen ilustrativo del mockup.

## Unidades de revisión y reversión

Sin commits creados. Mantener pruebas y documentación con cada comportamiento:

| Unidad | Límite de reversión, sin tocar backend u otras pantallas |
|---|---|
| Tablas y modales | `editor-registros.tsx`, secciones 5–8, `confirmarEdicionRegistro` y validación del registro; pruebas de edición en `edicion-modal.test.ts` |
| Excepciones | `excepcion-postulacion.tsx`, integración en sección 1 y formulario, `pantalla-bloqueo-registro.tsx`, integración de registro/alerta; `excepcion-postulacion.test.ts` |
| Impresión | `hoja-postulacion.tsx`, `impresion.css`, `imprimir-view.tsx`, tokens `--print-*`, duración al envío y su prop de experiencia; pruebas de hoja/duración y `check-print.mjs` |
| Evidencia aislada | Escenarios de `serve-fixtures.mjs`, capturas y este informe; no son dependencias de producción |

Validación conjunta: `pnpm test`, `pnpm lint`, `pnpm build`,
`pnpm format:check`, ambos verificadores del set y `check-print.mjs`.
