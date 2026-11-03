# Alineación visual del frontend con los mockups

**2026-11-03.** Se alineó la estructura compartida del portal con
`docs/mockups`: cabeceras, navegación, ancho de contenido, acceso, formulario y
administración. Es una entrega de alineación visual, **no una declaración de
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
a pantalla: no se cambió la hoja A4 ni el generador de documentos sellados.

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
| Pruebas frontend | 399 aprobadas en 36 archivos; cinco pruebas nuevas de cabecera, enlaces, estado activo, convocatoria y avance |
| Calidad | Lint, build y formato aprobados |
| Referencias estáticas | Ambos verificadores del set aprobados: 37 documentos incluyendo el índice |
| Navegador de escritorio | Acceso, registro, recuperación, 404, nueve secciones, dashboard, nómina y matriz revisados con datos sintéticos |
| Ancho móvil | Nueve secciones en iframe de 390 px: `scrollWidth = clientWidth = 375 px` (15 px corresponden a la barra de desplazamiento); sin desborde horizontal del documento |
| Integridad de la vista previa | Servidor sólo en loopback, cabecera `X-Data-Origin: synthetic-visual-only`, mutaciones rechazadas con 405; no crea cuentas ni roles reales |

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

## Alcance pendiente y revisión por unidades

- Formación, experiencia, publicaciones y capacitación conservan la edición en
  línea existente. Falta completar su conversión a tablas con diálogos de edición
  iguales a los estados específicos del set.
- No se revalidaron todos los modales, errores OTP, variantes de envío ni una
  impresión A4 en esta entrega. No se afirma regresión visual automatizada de los
  36 estados ni cumplimiento completo de accesibilidad.
- La matriz usa los campos disponibles del contrato; no reproduce un dictamen
  ilustrativo del mockup ni columnas que la API no aporta.

Unidades de revisión/reversión, sin commits creados: (1) marco compartido, acceso
y estilos; (2) navegación del formulario y sus pruebas; (3) administración y
matriz; (4) fixtures/evidencia/documentación. Cada unidad conserva sus pruebas y
no requiere revertir trabajo concurrente del backend. Los estilos y las rutas
pueden revisarse antes que los cambios de indentación del formulario.
