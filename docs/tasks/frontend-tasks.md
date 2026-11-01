# Tareas del frontend

**Sistema de Registro y Postulación al Directorio del BIESS**
Lista de trabajo del frontend: rúbrica del parcial y cobertura integral del RF v1.4

| Campo | Valor |
|---|---|
| Documento | frontend-tasks v1.1 |
| Ámbito | `apps/frontend/` |
| Convenciones | Identificadores y dominio en **español** |
| Fuente funcional | [RF v1.4](../requerimiento.txt), EF-1 y EF-11 a EF-14 |
| Identidad | [OTP por correo, ADR-008](../decisions/ADR-008-registro-y-recuperacion-con-otp.md) |

---

## 1. Alcance y criterio de cobertura

Esta lista conserva los IDs existentes FT1–FT13 y añade FT14–FT28 para cubrir el
RF completo. EF-13 y EF-14 son requisitos funcionales del producto, aunque su
implementación pueda programarse después del módulo inicial del parcial.
Las tareas nuevas son **P1-RF**, salvo identidad, elegibilidad, acceso y fases (**P0-RF**).
P2 queda reservado para mejoras; tener una ruta o un placeholder no cierra una vista.

Registro y recuperación usan **OTP por correo**: no se solicita fecha de expedición
ni código dactilar. El RF original se conserva; la adaptación aprobada está en ADR-008.

| Criterio de rúbrica | Máximo | Pendientes principales |
|---|---:|---|
| SPA, componentes y estado | 6 | Vistas reales y estados de formulario, administración y examen |
| API y asincronía | 6 | Integrar endpoints reales, permisos, documentos y operaciones de cada fase |
| Reglas del negocio | 6 | Matriz de la sección 9; un guard no sustituye las reglas del servidor |

Los puntos son máximos de la rúbrica, no una calificación ni un porcentaje de
cumplimiento del RF. Su criterio 6 es compartido con backend.

---

## 2. Estado actual

| Pieza | Ubicación | Estado |
|---|---|---|
| Estructura de Vite + Tailwind v4 | `apps/frontend/` | ✅ |
| Componentes shadcn/ui | `src/components/ui/` | ✅ (45 componentes) |
| Shell: sidebar, header, tema | `src/components/` | ✅ |
| Panel de revisión FT7 | `src/features/revision/` | Datos del backend; selección por período y conteos cada 30 s |
| Matriz interna FT8 | `/dashboard/reportes` | XLSX/PDF del backend; aprobación institucional pendiente |
| Tipos del dominio EF-12 | `src/types/concurso.ts` | ✅ |
| Validadores puros | `src/utils/validators.ts` | ✅ con tests |
| Catálogo geográfico | `src/data/ecuadorGeo.ts` | ✅ con tests (24 provincias, 221 cantones) |
| Proxy de Vite | `vite.config.ts` | ✅ |
| Vitest + Prettier | `apps/frontend/package.json` | ✅ (64 tests en verde) |
| **Router** | `src/router.tsx` | ✅ (rutas protegidas + guard) |
| **Cliente HTTP** | `src/lib/api.ts` | ✅ (con clientes de dominio y tests) |
| **Vistas de registro y login** | `src/pages/` | ✅ (`registro-view`, `login-view`, `recuperar-contrasena-view`) |
| **Vistas de las 9 secciones** | `src/pages/postulacion-view.tsx` | Estructura inicial; FT5 pendiente |
| **Vista de impresión** | `src/pages/imprimir-view.tsx` | Estructura inicial; FT6 pendiente |

> El set de mockups versionado en `docs/mockups/` es la referencia de flujo, copy
> y lógica de las 9 vistas. Las clases de `docs/mockups/assets/app.css` son el
> contrato de migración a shadcn/ui (mapa pantalla → HU → componente en
> `docs/mockups/README.md`). El `mockups/` original de React ya fue reemplazado.

---

## 3. P0 — Bloqueantes de la rúbrica

### FT1 · Cliente HTTP tipado y manejo asíncrono

> **Criterio 5 — Consumo de API y manejo asíncrono (6 pts)**

Sin esto, la SPA no consume la API y el criterio vale cero.

**Piezas**

| Pieza | Ubicación | Contenido |
|---|---|---|
| Cliente base | `src/lib/api.ts` | `apiGet`, `apiPost`, `apiPut`, `apiDelete` |
| Errores tipados | `src/lib/api-error.ts` | `ErrorApi` con `codigo` y `mensaje` |
| Cliente de dominio | `src/lib/clientes/` | `postulaciones.ts`, `auth.ts`, `catalogos.ts` |
| Configuración | `src/lib/config.ts` | URL base (relativa: `/api/v1`) |

**Reglas**

1. **URLs relativas**: el cliente nunca conoce el host de la API. En
   desarrollo lo resuelve el proxy de Vite; en producción, nginx.
2. **`credentials: 'include'`** en todas las peticiones: la cookie de sesión
   debe viajar.
3. **`async/await` en todas partes**, sin `.then()` encadenados.
4. **Error unificado**: si la respuesta no es 2xx, se lanza un error que
   contiene el `codigo` del envelope, para que la vista decida sin parsear
   texto.
5. **`AbortSignal`** en las peticiones cancelables (búsqueda, cambios de
   vista).

```ts
// Forma objetivo
export async function obtenerPostulaciones(
  params: { applicantId: string; page?: number },
  signal?: AbortSignal,
): Promise<Pagina<Postulacion>> {
  return apiGet<Pagina<Postulacion>>("/submissions", { params, signal })
}
```

**Criterios de aceptación**

- [x] Todas las llamadas pasan por el cliente único; no hay `fetch` suelto en
      los componentes.
- [x] El error que llega a la vista trae `codigo` y `mensaje`.
- [x] Un `401` en una ruta protegida dispara la redirección al login (vía callback configurable).
- [x] Test del cliente con un servidor simulado: éxito, 400, 401, 500 y
      error de red.

**Esfuerzo estimado**: 1,5 días.

---

### FT2 · Router con rutas protegidas

> **Criterio 5 — Rutas protegidas (6 pts)**

**Piezas**

| Pieza | Ubicación | Contenido |
|---|---|---|
| Router | `src/router.tsx` | React Router 8 (`createBrowserRouter`) |
| Guard de sesión | `src/components/guard-sesion.tsx` | Consulta la sesión y redirige |
| Layout | `src/layouts/layout-postulante.tsx` | Shell con sidebar y header |
| Layout | `src/layouts/layout-admin.tsx` | Shell del dashboard |

**Rutas**

| Ruta | Vista | Acceso |
|---|---|---|
| `/` | Portada | Público |
| `/registro` | Registro (EF-11) | Público |
| `/login` | Login (EF-11) | Público |
| `/recuperar-contrasena` | Recuperación (EF-11) | Público |
| `/postulacion` | Formulario EF-12 | Con sesión |
| `/postulacion/:seccion` | Sección concreta | Con sesión |
| `/postulacion/revision` | Revisión antes de enviar | Con sesión |
| `/postulacion/imprimir` | Vista imprimible (HU-11) | Con sesión |
| `/seguimiento` | Seguimiento de fases y resultados (FT22) | Con sesión |
| `/reconsideracion` | Reconsideraciones (FT21) | Con sesión |
| `/evaluacion` | Portal evaluación técnica EF-14 (FT25) | Con sesión |
| `/evaluacion/examen` | Examen interactivo EF-14 (FT26) | Con sesión |
| `/evaluacion/resultado` | Resultados evaluación EF-14 (FT27) | Con sesión |
| `/dashboard` | Panel del analista | Con rol analista |
| `/dashboard/reportes` | Matriz comparativa interna (FT8) | Con rol analista |
| `/dashboard/postulaciones` | Listado de revisión | Con rol analista |
| `/dashboard/postulaciones/:id` | Detalle | Con rol analista |
| `/admin/convocatorias` | Gestión de convocatorias (FT15) | Con rol admin |
| `/admin/fases` | Fases y cronograma (FT15) | Con rol admin |
| `/admin/preguntas` | Banco de preguntas EF-14 (FT24) | Con rol admin |
| `*` | 404 | Público |

**Criterios de aceptación**

- [x] Sin sesión, las rutas protegidas redirigen a `/login` y conservan el
      destino para volver tras entrar.
- [x] Con sesión, `/postulacion` es la ruta por defecto del postulante que
      aún no ha enviado ninguna postulación (HU-03, criterio 5).
- [x] Las rutas del dashboard exigen rol; un postulante que las abre recibe
      `403`.
- [x] El guard muestra un estado de carga mientras consulta la sesión, sin
      parpadeo de la pantalla de login.
- [x] La navegación interna no recarga la página.
- [x] Extender permisos al rol `administrador` (Comisión de Selección) y distinguirlo
      del analista; mantener el rechazo de acceso cuando falte autorización (T6).
- [x] Añadir rutas de convocatoria, fases, méritos, reconsideraciones y EF-14 según
      FT14–FT27. La admisión a examen exige elegibilidad y horario del servidor.


**Esfuerzo estimado**: 1,5 días.

---

### FT3 · Vistas de registro, login y recuperación (EF-11)

> **Criterio 6 — Reglas del negocio (6 pts) · Criterio 4 — Feedback visual**

**Piezas**

| Pieza | Ubicación | Contenido |
|---|---|---|
| Vista de registro | `src/pages/registro-view.tsx` | Wizard de 3 pasos |
| Vista de login | `src/pages/login-view.tsx` | Cédula + contraseña |
| Recuperación | `src/pages/recuperar-contrasena-view.tsx` | Solicitud y restablecimiento |
| Componentes | `src/features/auth/` | Formularios, hooks y validaciones |

**Flujo de registro adaptado con OTP (HU-01 y HU-02, ADR-008)**

1. Solicitar cédula de diez dígitos y correo; validar módulo 10 y formato de correo.
2. Solicitar el envío con `POST /auth/register/request`.
3. Mostrar ingreso del OTP de seis dígitos y completar nombres, apellidos y contraseña.
4. Confirmar con `POST /auth/register/verify`; no crear la cuenta solo por enviar el correo.
5. Informar OTP inválido/expirado, permitir reenvío conforme al servidor y respetar
   el bloqueo de tres intentos y diez minutos; no reiniciarlo al recargar la página.
6. Confirmar el registro y garantizar que el primer acceso autenticado lleve al EF-12.
   La API actual puede emitir sesión al verificar; no entrar en bucles entre login y formulario.

**Login y recuperación (HU-03 y HU-04)**

- Login con cédula y contraseña; mensajes literales del RF y feedback de bloqueo.
- Recuperación: pedir cédula, llamar a `/auth/password/forgot` y mostrar confirmación
  genérica. El OTP se entrega al correo registrado, sin permitir cambiar el destinatario.
- Pedir OTP y nueva contraseña para `/auth/password/reset`; después volver al login.
- No pedir fecha de expedición ni código dactilar. El correo verificado no se presenta
  como una certificación de identidad emitida por Registro Civil.

**Criterios de aceptación**

- [x] El formulario da feedback **inmediato** por campo (al perder el foco) y
      en el envío.
- [x] Hay un estado de carga en el botón de envío, deshabilitado para evitar
      el doble envío.
- [x] Los mensajes del RF son literales.
- [x] El formulario es navegable por teclado y los errores se anuncian a
      lectores de pantalla.
- [x] Mostrar intentos restantes y vencimiento solo si los informa el servidor;
      no inventar contadores o un tiempo nuevo en cada recarga (FT16/T16).

**Esfuerzo estimado**: 3 días.

---

## 4. P1 — Importantes

### FT4 · Estado remoto con TanStack Query

> **Criterio 4 — Estado sólido (6 pts) · Criterio 5 — Manejo asíncrono**

| Pieza | Ubicación | Contenido |
|---|---|---|
| Proveedor | `src/lib/query-provider.tsx` | `QueryClient` con reintentos y `staleTime` |
| Hooks | `src/features/<contexto>/queries.ts` | `usePostulaciones`, `useGuardarSeccion`, … |
| Estados | `src/components/estados/` | `Cargando`, `Vacio`, `Error` reutilizables |

- [x] Cada vista maneja **loading / success / error** de forma uniforme.
- [x] Los errores de red se reintentan con retroceso exponencial; los 4xx no.
- [x] El caché de React Query invalida la postulación tras guardar.
- [x] Un `toast` (sonner) informa del resultado de cada acción.

**Esfuerzo estimado**: 1,5 días.

### FT5 · Formulario EF-12: las 9 secciones

> **Criterio 6 — Reglas del negocio (6 pts) · Criterio 4 — Componentes**

El mayor bloque del frontend. El set de mockups en `docs/mockups/` es la
referencia de flujo y copy.

| Sección | Contenido | Elementos clave |
|---|---|---|
| 1 | Información personal | Nombres, cédula, género, nacimiento, nacionalidad, estado civil, pareja |
| 2 | Domiciliaria y contacto | Cascada País → Provincia → Cantón, 2 correos, 2 teléfonos, dirección |
| 3 | Información adicional | Familiares en segundo grado de consanguinidad / cuarto de afinidad |
| 4 | Selección del cargo | Sector + respaldo de organización (PDF 4 MB) |
| 5 | Formación académica | Nivel, institución, título, área, SENESCYT, documento (PDF 2 MB) |
| 6 | Experiencia profesional | Tipo, cargo, fechas, funciones (máx. 1000), **duración calculada**, documento |
| 7 | Publicaciones | Título, tipo, medio, fecha, relación con el campo, documento |
| 8 | Capacitación | Tipo, nombre, institución, fechas, horas, documento |
| 9 | Inhabilidades | 13 preguntas Sí/No; si 1.9.7 es "Sí", pedir más datos |

**Detalle exigible además de la tabla**

- Sección 1: precargar nombres, apellidos y cédula desde la cuenta OTP; completar
  género, nacimiento, nacionalidad y estado civil de forma declarada. Si casado/a
  o unión de hecho, mostrar nombre completo y cédula de pareja. No mostrar sello
  de verificación de Registro Civil sin una consulta real.
- Sección 2: precargar correo 1 desde la cuenta; conservar correo 2, celular,
  teléfono domiciliario, calle principal, número, calle secundaria, sector y referencia.
- Sección 3: listas exactas de parentesco e institución IESS/BIESS, área y cargo;
  permitir varios familiares con agregar/editar/eliminar y guardar información.
- Sección 4: cargos y mensajes literales por Afiliados/Jubilados/Empleadores;
  pregunta de organización correspondiente a cada sector, nombre y PDF condicional.
- Secciones 5–8: todos los campos y opciones de EF-12 1.5–1.8, fechas SENESCYT,
  actividades de experiencia y horas de capacitación; publicaciones se oculta al responder No.
- Sección 9: las trece preguntas literales (1.9.1 a 1.9.13), sus trece respuestas
  y los campos adicionales de 1.9.7; no resumirlas ni invertir su sentido.
- Usar todos los países del RF y los catálogos de T7; no depender solo de una lista
  reducida de países ni crear opciones distintas entre interfaz y servidor.

**Comportamiento común**

- **Guardar**, **Editar**, **Eliminar** y **Guardar información** en cada
  sección.
- **Guardar progreso**: el borrador persiste y se recupera al reingresar
  (HU-09).
- Barra de progreso con las 9 secciones y su estado (completa, parcial,
  pendiente).

**Criterios de aceptación**

- [x] La sección 2 implementa la **cascada**: si el país es Ecuador, provincia
      y cantón se eligen de la lista; si no, se escriben a mano.
- [x] Si la nacionalidad no es **ecuatoriana**, el sistema **no permite
      continuar** y cierra la postulación (HU-06).
- [x] La duración de la experiencia se muestra en **años, meses y días**
      (usando `calcularDuracionExperiencia`, ya implementado y con test).
- [x] La sección 9 exige las 13 respuestas y pide los 4 campos extra cuando
      1.9.7 es "Sí".
- [x] Los adjuntos validan el tamaño **antes** de subir y muestran el límite
      vigente (2 MB / 4 MB).
- [x] El guardado es optimista con reversión si la API falla, o pesimista con
      feedback claro (elegir una estrategia y aplicarla igual en todas las
      secciones).

**Esfuerzo estimado**: 8 días (la mayor tarea del proyecto).

### FT6 · Revisión, envío e impresión

> **Criterio 6 (HU-10 y HU-11)**

**Piezas**

| Pieza | Ubicación | Contenido |
|---|---|---|
| Revisión | `src/pages/revision-postulacion-view.tsx` | Resumen de las 9 secciones |
| Consentimiento | `src/features/postulacion/consentimiento.tsx` | Los 5 puntos literales del RF |
| Impresión | `src/pages/imprimir-view.tsx` | Formulario completo con firma manuscrita |

**Criterios de aceptación**

- [x] **Guardar Postulación** muestra los cinco puntos literales de EF-12 1.10.1;
      también están visibles en revisión/envío y constan en el formulario final.
- [x] Sin marcar todos los puntos, el envío está deshabilitado.
- [x] Tras enviar, la postulación queda **bloqueada**: no se edita y se
      muestra el hash.
- [x] La vista de impresión incluye el espacio de **firma manuscrita**, los
      nombres, los apellidos y la cédula (HU-11).
- [x] La vista de impresión tiene estilos de impresión (`@media print`) que
      ocultan la navegación.
- [x] Imprimir el formulario íntegro sellado: nueve secciones, código único,
      consentimientos, nombres/apellidos/cédula y espacio para firma en papel.
      No exigir una firma dibujada digitalmente como sustituto del requisito manuscrito.
- [x] Informar el envío del formulario completo al correo registrado y mostrar el
      estado de notificación que entregue la API; no prometer entrega por confirmar el sello (T19/T23).


**Esfuerzo estimado**: 2,5 días.

---

### FT7 · Dashboard e históricos reales del personal autorizado

**Origen:** EF-13.3. **Prioridad:** P1-RF.

- [x] Consumir conteos reales por cargo/sector; consulta automática cada 30 s con el
      panel visible y conectado, además de actualización manual y al reconectar.
- [x] Selección explícita de convocatoria (incluidas cerradas) o legado sin convocatoria,
      filtros de cargo/sector y estado enviado, y paginación por cursor hasta el final.
- [x] Consultar EF-12 sellado íntegro, sus nueve secciones y consentimiento original,
      certificados con enlaces temporales y duración por experiencia en años, meses y días.
- [x] Excluir borradores, identificar envío/histórico y explicar que no constituye
      resultado de fase. No mezclar perfiles o formularios de otra convocatoria.
- [ ] Integrar resultados reales de fases cuando T18/T20 los expongan; no hay estados
      ficticios ni operaciones de calificación en esta entrega.
- [ ] Validación integrada en navegador con servicios reales y aceptación institucional
      de la latencia propuesta. Las comprobaciones locales no sustituyen esa evidencia.

**Backend consumido:** T6, T8–T9. **Alcance pendiente:** T18 y resultados posteriores.
**Evidencia y uso:** [Contrato de revisión y auditoría](../api-contracts/REVIEW_AUDIT.md).

### FT8 · Matriz comparativa en Excel y PDF

**Origen:** EF-13.3. **Prioridad:** P1-RF.

- [x] Solicitar `.xlsx` y PDF autorizados al backend con la misma convocatoria,
      cargo/sector y estado de FT7; exportar toda la selección, no la página cargada.
- [x] Mostrar generación, descarga iniciada, cancelación, errores y reintento; evitar
      doble solicitud, cancelar al cambiar filtros y revocar las URLs binarias locales.
- [x] Respetar los errores 401/403/413; no convertir JSON de error en archivo.
- [x] Rotular EF13-TECNICA-v1 como propuesta técnica interna sin puntajes ni lista pública
      de elegibles. Ambos formatos son generados por el mismo contrato backend.
- [ ] Acordar columnas con Talento Humano y completar la validación integrada en navegador.
      T24/publicación de listas no se presume implementado.

**Backend consumido:** T9. **Dependencias:** FT7; T24 queda para publicación posterior.

---

## 5. P2 — Deseables

| ID | Tarea | Beneficio |
|---|---|---|
| FT9 | Pruebas E2E con Playwright | Detectar regresiones del recorrido |
| FT10 | Modo offline con aviso de conexión | Cada caída de red |
| FT11 | Internacionalización (es/en) | Escalabilidad futura |
| FT12 | Medir el bundle y cargar el formulario por secciones | Rendimiento en equipos lentos |
| FT13 | Atajos de teclado y navegación completa por teclado | Accesibilidad |

---

## 6. Orden de ejecución y cierre funcional

1. Sobre FT1–FT3 disponibles, consolidar OTP FT16 y permisos/apertura FT14–FT15.
2. Completar FT5/FT6 y documentos FT18, integrar elegibilidad FT17 y políticas FT28.
3. Implementar administración real FT7–FT8, méritos FT19 y seguimiento FT22.
4. Incorporar EF-14 con FT24–FT27 y consolidar expediente FT20,
   reconsideraciones FT21 y publicación FT23 según sus dependencias.

La matriz de la sección 9 define cobertura de planificación. Cada funcionalidad
solo se cierra cuando la interfaz consume su capacidad backend y se cumplen sus
criterios; no se deducen puntos ni cumplimiento a partir de placeholders.

---

## 7. Convenciones para las tareas nuevas

1. **Identificadores en español**, en el dominio y en los componentes:
   `esValida`, `seccion4`, `cargarPostulacion`.
2. **Los componentes de shadcn/ui no se editan a mano**: se regeneran con
   `pnpm dlx shadcn@latest add <componente>`.
3. **Las reglas puras van en `src/utils/`** con test en `src/utils/*.test.ts`.
4. **Nada de `any`**: TypeScript estricto, `import type` para los imports
   solo de tipo.
5. **Estilos solo con clases de Tailwind**; nada de CSS en línea salvo valores
   dinámicos.
6. **Antes de abrir el PR**: `pnpm format:check`, `pnpm lint`, `pnpm test` y
   `pnpm build` en verde.

---

## 8. Tareas añadidas para cubrir el RF completo

Todas las casillas nuevas permanecen pendientes. Las pantallas pueden dividirse
por feature al implementarlas; esta lista define comportamientos y dependencias.

### FT14 · Portal del concurso, publicación y apertura

**Origen:** EF-11.1–3 y EF-1 restricciones temporales. **Prioridad:** P0-RF.

- [x] Mostrar convocatoria y fases publicadas, fechas y estado abierto/cerrado,
      utilizando los datos del servidor y la zona horaria definida por T18.
- [x] Ofrecer iniciar sesión y registrarse. Proporcionar el enlace para integración
      en portales IESS/BIESS con apertura en nueva pestaña; la publicación externa
      requiere coordinación institucional y no se presume realizada desde este repo.
- [x] Ante cierre o deshabilitación, actualizar la pantalla y explicar por qué no
      se puede postular, incluso si el usuario dejó abierto el formulario.

**Estado técnico (2026-10-31):** implementado y verificado localmente.
Publicación en sitios externos y despliegue institucional no realizados.
Ver [entrega FT14/FT15](FT14-FT15.md).

**Backend:** T18. **Dependencias:** FT2–FT3.

### FT15 · Administración de convocatorias, fases y permisos

**Origen:** EF-1 actores; EF-13.1–2. **Prioridad:** P0-RF.

- [x] Incorporar el rol administrador junto con analista y postulante; menús y rutas
      deben reflejar permisos del servidor, sin permitir asignarse un rol en registro.
- [x] Permitir al rol autorizado publicar fases, aprobar/programar apertura y cierre
      y habilitar EF-12, con confirmación de cambios y estado de publicación.
- [x] Separar calendario/fase, estado del formulario y resultados de evaluación;
      mostrar histórico y no mezclar convocatorias activas y cerradas.
- [x] Presentar rechazos de autorización/plazo de la API y conservar su trazabilidad.

**Estado técnico (2026-10-31):** implementado sobre contrato T18.
Las fechas generales se programan al crear el período; sus fases se editan con
confirmación y versión. No se inventa un endpoint para modificar las fechas base.
Ver [entrega FT14/FT15](FT14-FT15.md).

**Backend:** T6, T18. **Dependencias:** FT2, FT4.

### FT16 · Integrar el ciclo completo de OTP por correo

**Origen:** EF-11 adaptado por ADR-008. **Prioridad:** P0-RF; complementa FT3.

- [x] Consumir request/verify de registro y forgot/reset de recuperación; vincular
      el paso visible al desafío, correo y finalidad de la operación correspondiente.
- [x] Mostrar expiración, OTP incorrecto, uso previo, bloqueo y reenvío conforme
      al servidor; no generar códigos ni verificar su validez en el navegador.
- [x] Mantener cédula/correo vinculados entre pasos y reiniciar el desafío si cambia
      el correo; no permitir elegir otro destinatario durante recuperación.
- [x] Precargar datos disponibles de la cuenta en EF-12 y dejar el resto declarativo,
      sin pedir fecha de expedición ni código dactilar ni mostrar verificación civil.

**Backend:** T2, T16. **Dependencias:** FT3, FT5.

### FT17 · Elegibilidad por sector y cierre automático del formulario

**Origen:** EF-12 1.1.6 y 1.4.1–1.4.3. **Prioridad:** P0-RF.

- [x] Consultar el resultado de afiliación/jubilación/empleador del servidor al
      seleccionar sector y mostrar los mensajes literales correspondientes.
- [x] Informar el origen simulado cuando se use el adaptador local; no convertir
      un checkbox del postulante en una verificación de Historial Laboral.
- [x] Mostrar cierre por nacionalidad no admitida o sector no elegible y abandonar
      el flujo editable; al recargar, conservar el estado de cierre del servidor.
- [x] Tratar indisponibilidad técnica como error recuperable, no como descalificación.
- [x] Mostrar la pregunta y respaldo de organización/federación propios de cada sector.

**Backend:** T17. **Dependencias:** FT5, FT14, FT18.

### FT18 · Adjuntos reales por registro y cuota total

**Origen:** EF-1 restricciones y EF-12 1.4–1.8. **Prioridad:** P1-RF.

- [x] Añadir cliente `documentos` con carga multipart, listado, baja y solicitud de
      URL temporal; guardar primero el registro con ID estable para asociar su PDF.
- [x] Integrar organización, títulos, experiencias, publicaciones y cursos: nombre,
      tamaño, estado de carga/error, reemplazo y descarga del respaldo correspondiente.
- [x] Mostrar límites 4/2 MiB y total 50 MiB, contabilizando lo que informa el servidor;
      la validación local no reemplaza la cuota/MIME comprobados por la API.
- [x] Conservar referencias devueltas por la API y resolver archivos de registros
      retirados antes del envío; impedir borrados y cargas tras el sello.
- [x] Abrir URLs temporales sin guardarlas en almacenamiento persistente del navegador
      y solicitar una nueva si expiraron. No descargar pasando el binario por la API.

**Backend:** T4. **Dependencias:** FT1, FT5–FT6.

### FT19 · Revisión de requisitos, inhabilidades y méritos

**Origen:** EF-1, verificación y calificación de méritos. **Prioridad:** P1-RF.

- [x] Mostrar evidencia, trece declaraciones, observaciones y dictamen de revisión,
      con acciones solo para el rol y la fase autorizados.
- [x] Mostrar criterios/versiones aprobados y el cálculo del servidor con desglose;
      no calcular la nota definitiva ni decidir aptitud solo desde la interfaz.
- [x] Gestionar avance o exclusión, motivación y publicación del resultado de méritos.
- [x] Explicar resultados y sus efectos sobre la habilitación de evaluación técnica.

**Backend:** T9, T20. **Dependencias:** FT7, FT15, FT18.

### FT20 · Resultados de oposición psicotécnica y expediente

**Origen:** EF-1, oposición teórica y psicotécnica. **Prioridad:** P1-RF.

- [x] Proporcionar al personal autorizado el registro del resultado psicotécnico
      según instrumento y formato definidos, con evidencia y responsable.
- [x] Mostrar nota teórica recibida de T27 y resultados consolidados del servidor,
      con historial de correcciones y versiones publicadas.
- [x] Permitir al postulante consultar únicamente su expediente y sus resultados
      cuando sean publicados, sin acceder a documentos o notas de otros usuarios.

**Backend:** T21, T27. **Dependencias:** FT7, FT15, FT19.

### FT21 · Solicitud, atención y resolución de reconsideraciones

**Origen:** EF-1, reconsideraciones y plazos por actor. **Prioridad:** P1-RF.

- [x] Permitir al postulante presentar motivos y respaldos contra el resultado
      habilitado; mostrar plazo y estado de su solicitud con datos del servidor.
- [x] Incorporar bandejas de atención de Talento Humano y de resolución de la Comisión,
      con expediente, responsables, vencimiento y acciones autorizadas por estado.
- [x] Mostrar resolución y efecto en calificaciones/avance, sin sobrescribir la
      evidencia del resultado original ni permitir envíos duplicados.
- [x] No permitir actuar fuera de plazo aunque la pantalla haya quedado abierta;
      no fijar días o términos legales que el RF no especifica.

**Backend:** T22. **Dependencias:** FT15, FT19–FT20, FT22.

### FT22 · Seguimiento de fases, resultados y notificaciones

**Origen:** EF-1, comunicaciones automáticas por etapa. **Prioridad:** P1-RF.

- [x] Mostrar al postulante el estado real de cada fase y las resoluciones publicadas,
      con acceso a su reporte, convocatoria y próximos pasos disponibles.
- [x] Distinguir operación confirmada, notificación pendiente, entregada o fallida
      según el backend; no deducir entrega del correo a partir de un toast de éxito.
- [x] Proporcionar al personal autorizado consulta/reintento de notificaciones
      fallidas cuando T23 lo permita, sin exponer destinatarios a otros postulantes.

**Backend:** T21–T23. **Dependencias:** FT4, FT15.

### FT23 · Listas de elegibles y publicación de resultados

**Origen:** EF-1 reportes/listas; EF-12 1.10.1.4. **Prioridad:** P1-RF.

- [x] Mostrar listas por convocatoria/cargo y versión publicada con el orden del
      servidor, y permitir al rol competente revisar/publicar según su autorización.
- [x] Separar vista interna de publicación pública, respetando consentimiento y
      campos autorizados; no exponer certificados o contactos del expediente.
- [x] Permitir consultar versiones históricas y los reportes correspondientes.

**Backend:** T24. **Dependencias:** FT8, FT15, FT19–FT22, FT28.

### FT24 · Gestión administrativa del banco de preguntas

**Origen:** EF-14 1.1. **Prioridad:** P1-RF.

- [x] Permitir cargar y gestionar preguntas/opciones/respuestas correctas con el
      formato que se acuerde, mostrando validaciones de estructura y banco insuficiente.
- [x] Mostrar versiones del banco y advertir el efecto de editar solo futuras
      evaluaciones; los exámenes iniciados conservan su versión.
- [x] Restringir esta interfaz al administrador y evitar incluir respuestas correctas
      en las vistas o peticiones de un postulante durante un examen activo.

**Backend:** T6, T25. **Dependencias:** FT15.

### FT25 · Programación y habilitación de la evaluación técnica

**Origen:** EF-14, apertura y punto 1.2. **Prioridad:** P1-RF.

- [x] Permitir al administrador definir fecha/hora, duración y cantidad de preguntas
      y habilitar exclusivamente a quienes superaron la fase de méritos.
- [x] Mostrar elegibilidad y estados no habilitado/programado/disponible/vencido
      desde la API; mostrar «Evaluación Técnica» al habilitarse la fase. El módulo
      permanece oculto al público general.
- [x] Gestionar los rechazos por horario, fase o banco insuficiente antes de publicar
      la programación, sin inventar valores no acordados por la institución.

**Backend:** T18, T20, T25–T26. **Dependencias:** FT15, FT19, FT24.

### FT26 · Aplicación de examen con preguntas individuales y reloj

**Origen:** EF-14 1.2–1.3.4. **Prioridad:** P1-RF.

- [ ] Mostrar «Iniciar Evaluación» solo cuando la API lo permita; recuperar el intento
      y su orden persistido, con una pregunta a la vez y guardado de respuestas.
- [ ] Mostrar cuenta regresiva superior calculada desde el vencimiento del servidor;
      recargar o cambiar el reloj local no reinicia el tiempo.
- [ ] Aplicar los controles de navegación Atrás y copiar/pegar solicitados por el RF
      dentro de la pantalla; la autorización y el vencimiento se siguen imponiendo en servidor.
- [ ] Incluir «Finalizar y Calificar» y al llegar a 00:00 cerrar el flujo y solicitar
      su finalización con las respuestas guardadas; el servidor también cierra sin navegador.
- [ ] Resolver desconexión y reconexión sin crear otro examen ni reordenar preguntas;
      mostrar qué respuestas quedaron confirmadas y no admitir envíos posteriores al plazo.

**Backend:** T26–T27. **Dependencias:** FT4, FT25.

### FT27 · Nota, reporte PDF y cierre de evaluación

**Origen:** EF-14 1.3.5–1.4. **Prioridad:** P1-RF.

- [ ] Tras cierre manual o automático, mostrar la nota calculada por el servidor y
      su incorporación al expediente; la SPA no manda una nota definitiva.
- [ ] Mostrar generación/disponibilidad/error del reporte PDF y descargar la copia
      autorizada; el módulo administrativo conserva acceso al mismo resultado.
- [ ] Informar estado de la notificación automática al correo registrado y tratar
      reintentos de finalización como consulta del mismo resultado, sin duplicar notas.

**Backend:** T23, T27. **Dependencias:** FT22, FT26.

### FT28 · Política de privacidad y consentimiento informado

**Origen:** EF-1 precondiciones y EF-12 1.10. **Prioridad:** P1-RF.

- [ ] Presentar la política institucional vigente con versión y fecha desde registro
      y formulario, utilizando el contenido aprobado de T28.
- [ ] Mantener los cinco literales del RF en revisión e impresión; no reemplazarlos
      por un único checkbox genérico ni añadir autorizaciones obligatorias no definidas.
- [ ] Limitar la presentación de datos en listados, reportes y comunicaciones al
      destinatario y rol correspondientes, conforme a la política aprobada.

**Backend:** T28. **Dependencias:** FT3, FT6.

---

## 9. Matriz de cobertura y definiciones pendientes

**Cobertura de planificación, no certificación de implementación.** La matriz
incluye lo explícito de EF-11–EF-14 y los procesos que EF-1 describe sin detallar.

| Fuente del RF | Requisito o adaptación | Backend | Frontend | Situación |
|---|---|---|---|---|
| EF-1, actores | Postulante, Analista TH y Administrador/Comisión | T6 | FT2, FT15 | Tres roles y permisos backend disponibles; integración administrativa de interfaz pendiente |
| EF-1, privacidad | Política, tratamiento y consentimiento informado | T28, T3 | FT28, FT6 | Consentimientos base; política e integración pendientes |
| EF-11.1–3 | Fase aprobada, portal institucional, nueva pestaña y acceso | T18 | FT14–FT15 | Planificado |
| EF-11.4–6, ADR-008 | Registro, login y recuperación con OTP por correo | T1–T2, T16 | FT3, FT16 | Implementado: ciclo completo OTP con cooldown, desafío del servidor y bloqueo |
| EF-12, primer ingreso | Entrada obligatoria al formulario | T2, T18 | FT2–FT3 | Guard base; cerrar flujo funcional |
| EF-12 1.1 | Datos personales/cónyuge y nacionalidad; precarga de cuenta OTP | T3, T17 | FT5, FT16–FT17 | Implementado: precarga OTP y cierre por nacionalidad |
| EF-12 1.2 | Países completos, cascada, correo registrado y dirección | T3, T7 | FT5 | Catálogos backend completos; integración de vistas pendiente |
| EF-12 1.3 | Familiares y opciones exactas de parentesco/institución | T3, T7 | FT5 | Opciones backend disponibles; integración visual pendiente |
| EF-12 1.4 | Cargo, elegibilidad, mensajes, cierre y organización | T3–T4, T17–T18 | FT5, FT17–FT18 | Implementado: elegibilidad por sector del servidor, cierre automático y persistencia |
| EF-12 1.5–1.8 | Títulos, experiencia/calculadora, publicaciones y capacitación | T3–T4, T7 | FT5, FT18 | Implementado: vistas y adjuntos integrados con límites y cuotas |
| EF-12 1.9 | Trece respuestas, detalles de remoción y revisión | T3, T20 | FT5, FT19 | Implementado: declaraciones, revisión técnica y dictamen administrativo |
| EF-1, restricciones | PDF 2/4 MiB y total 50 MiB | T4 | FT18 | Implementado: cliente multipart, límites 4/2 MiB y cuota total 50 MiB |
| EF-12, notas | Guardado parcial, CRUD de registros y código único | T3–T5 | FT5 | Backend base; interfaz funcional pendiente |
| EF-12 1.10 | Literales al guardar, firma manuscrita, impresión y correo completo | T3, T19, T23 | FT6, FT28 | Sello y correo con JSON disponibles; presentación final pendiente |
| EF-13.1–2 | Publicar fases y habilitar EF-12 | T6, T18 | FT15 | Planificado |
| EF-13.3 | Conteos por cargo en tiempo real e históricos íntegros/certificados | T8–T9, T18 | FT7 | Integración FT7 disponible; pendientes T18/fases y validación integrada |
| EF-13.3 | Matriz comparativa Excel/PDF | T9 | FT8 | Integración XLSX/PDF disponible; aprobación institucional y validación integrada pendientes |
| EF-1 | Verificar requisitos y calcular méritos | T20 | FT19 | Implementado: revisión técnica, cálculo oficial meritos-v1 y publicación de resolución |
| EF-1 | Registrar oposición psicotécnica y consolidar expediente | T21 | FT20 | Implementado: registro psicotécnico y consolidación de expediente con notas oficiales |
| EF-1 | Reconsideraciones y plazos exactos por actor | T22 | FT21 | Implementado: bandejas de recepción, análisis y resolución con plazos del servidor |
| EF-1 | Notificar resultados de cada etapa | T23 | FT22 | Implementado: seguimiento de etapas para postulante, estados auditados y reintentos administrativos |
| EF-1 | Reportes, listas de elegibles y trazabilidad completa | T8, T24 | FT7–FT8, FT23 | Implementado: matriz preliminar EF-13, generación reproducible de listas, CSV y nómina pública ciudadana |
| EF-14, apertura/1.1 | Módulo restringido y banco de preguntas | T6, T25 | FT24–FT25 | Implementado: banco inmutable, matriz auditada, validaciones y configuración |
| EF-14 1.2 | Méritos aprobados, fecha/hora, azar, cantidad y orden distinto | T18, T20, T26 | FT25–FT26 | Implementado: habilitación individual, programación y verificación de elegibilidad |
| EF-14 1.3.1–1.3.4 | Pregunta individual, reloj, controles UI y cierre al vencer | T26–T27 | FT26 | Planificado |
| EF-14 1.3.5–1.4 | Calificar, guardar nota, PDF, correo y copia administrativa | T21, T23, T27 | FT20, FT27 | Planificado; escala de nota pendiente |

### Definiciones que el RF no permite resolver por suposición

- Fórmula y máximos de méritos, umbrales de avance, ponderación de oposición,
  instrumento psicotécnico y reglas de desempate: T20–T21.
- Duración y cantidad de preguntas, escala de calificación, intentos permitidos y
  criterio de unicidad cuando se agota el banco: T25–T27.
- Plazos de reconsideración de cada actor, cómputo de días y calendario: T22.
- Columnas de matriz/listas, permisos precisos por actor y latencia de «tiempo real»:
  T6, T9 y T24. El RF los solicita pero no fija todos esos detalles.
- EF-12 menciona al inicio «Referencias Personales», pero sus apartados 1.1–1.9
  no desarrollan esa sección. Aclarar su alcance antes de crear una décima sección.
- Resolver campos opcionales/asteriscos frente a «todos los campos llenos», y la
  diferencia entre las ciudades enumeradas en el RF y el catálogo nacional usado.
- La decisión de no usar Registro Civil ya está resuelta por ADR-008: no queda
  como bloqueo ni se vuelve a exigir fecha de expedición/código dactilar. El RF
  original se conserva para trazabilidad y el OTP se documenta como adaptación.

**Pendiente de implementación** no significa **requisito opcional**: FT7/FT8 y
T16–T28/FT14–FT28 cubren funciones del RF que faltaban o estaban insuficientemente
definidas en las listas anteriores. Tareas P2 de offline, idiomas, métricas o
atajos no sustituyen estos requisitos.

---

**Ver también**: [backend-tasks.md](backend-tasks.md) ·
[`../architecture/ARQUITECTURA.md`](../architecture/ARQUITECTURA.md) ·
[`../api-contracts/API_CONTRACT.md`](../api-contracts/API_CONTRACT.md)
