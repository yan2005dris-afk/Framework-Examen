# Tareas del frontend

**Sistema de Registro y Postulación al Directorio del BIESS**
Lista de trabajo del frontend, priorizada y mapeada a la rúbrica de 40 puntos

| Campo | Valor |
|---|---|
| Documento | frontend-tasks v1.0 |
| Ámbito | `apps/frontend/` |
| Convenciones | Identificadores y dominio en **español** |

---

## 1. Estado por criterio de la rúbrica

| # | Criterio | Puntos | Estado | Tareas |
|---|---|---:|---|---|
| 4 | Frontend SPA: reactividad, componentes y estado | 6 | 🟡 **2/6** | FT2, FT3, FT5 |
| 5 | Consumo de API y manejo asíncrono | 6 | 🟡 **2/6** | **FT1**, FT4, FT6 |
| 6 | Cumplimiento de las reglas del negocio (RF IESS) | 6 | 🟡 **1/6** | FT7, FT8 |
| | **Total frontend** | **18** | **5/18** | |

---

## 2. Estado actual

| Pieza | Ubicación | Estado |
|---|---|---|
| Estructura de Vite + Tailwind v4 | `apps/frontend/` | ✅ |
| Componentes shadcn/ui | `src/components/ui/` | ✅ (45 componentes) |
| Shell: sidebar, header, tema | `src/components/` | ✅ |
| Dashboard (mockup migrado) | `src/pages/dashboard-view.tsx` | 🟡 con datos mock |
| Tipos del dominio EF-12 | `src/types/concurso.ts` | ✅ |
| Validadores puros | `src/utils/validators.ts` | ✅ con tests |
| Catálogo geográfico | `src/data/ecuadorGeo.ts` | ✅ con tests (24 provincias, 221 cantones) |
| Proxy de Vite | `vite.config.ts` | ✅ |
| Vitest + Prettier | `apps/frontend/package.json` | ✅ (64 tests en verde) |
| **Router** | — | ⬜ |
| **Cliente HTTP** | `src/lib/api.ts` | ✅ (con clientes de dominio y tests) |
| **Vistas de registro y login** | — | ⬜ |
| **Vistas de las 9 secciones** | — | ⬜ |
| **Vista de impresión** | — | ⬜ |

> El `mockups/` original es la referencia de flujo, copy y lógica de las 9
> vistas. Está en `.gitignore` y se elimina cuando termine la migración.

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
| Router | `src/router.tsx` | React Router v7 |
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
| `/dashboard` | Panel del analista | Con rol |
| `/dashboard/postulaciones` | Listado de revisión | Con rol |
| `/dashboard/postulaciones/:id` | Detalle | Con rol |
| `*` | 404 | Público |

**Criterios de aceptación**

- [ ] Sin sesión, las rutas protegidas redirigen a `/login` y conservan el
      destino para volver tras entrar.
- [ ] Con sesión, `/postulacion` es la ruta por defecto del postulante que
      aún no ha enviado ninguna postulación (HU-03, criterio 5).
- [ ] Las rutas del dashboard exigen rol; un postulante que las abre recibe
      `403`.
- [ ] El guard muestra un estado de carga mientras consulta la sesión, sin
      parpadeo de la pantalla de login.
- [ ] La navegación interna no recarga la página.

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

**Flujo de registro (HU-01 y HU-02)**

1. **Paso 1 — Identidad**: cédula (10 dígitos) y **fecha de expedición**
   (DD/MM/AAAA) **o** **código dactilar** (10 dígitos).
2. Validación con `validarCedulaEcuatoriana` (módulo 10, ya implementado).
3. Si falla: *"la información ingresada es incorrecta"*.
4. **Paso 2 — Cuenta**: correo y contraseña.
5. Validación de fortaleza con `evaluarFortalezaPassword` (ya implementado):
   ≥ 8 caracteres, mayúscula, minúscula, número y símbolo.
6. **Paso 3 — Confirmación**: resumen y aceptación.
7. Tras 3 intentos fallidos, la API responde `429`; la vista muestra
   *"registro fallido, inténtelo nuevamente en 10 minutos"* y **deshabilita el
   formulario** hasta que expire el bloqueo.

**Flujo de login (HU-03)**

- Cédula y contraseña.
- Credenciales incorrectas → *"sus credenciales de ingreso son incorrectas"*.
- Bloqueo activo → mensaje de 10 minutos.
- Éxito → redirección según `mustCompleteApplication`.

**Criterios de aceptación**

- [ ] El formulario da feedback **inmediato** por campo (al perder el foco) y
      en el envío.
- [ ] Hay un estado de carga en el botón de envío, deshabilitado para evitar
      el doble envío.
- [ ] Los mensajes del RF son literales.
- [ ] El formulario es navegable por teclado y los errores se anuncian a
      lectores de pantalla.
- [ ] El contador de intentos fallidos se muestra desde el segundo fallo
      ("le quedan 2 intentos").

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

- [ ] Cada vista maneja **loading / success / error** de forma uniforme.
- [ ] Los errores de red se reintentan con retroceso exponencial; los 4xx no.
- [ ] El caché de React Query invalida la postulación tras guardar.
- [ ] Un `toast` (sonner) informa del resultado de cada acción.

**Esfuerzo estimado**: 1,5 días.

### FT5 · Formulario EF-12: las 9 secciones

> **Criterio 6 — Reglas del negocio (6 pts) · Criterio 4 — Componentes**

El mayor bloque del frontend. El `mockups/` original es la referencia de flujo
y copy.

| Sección | Contenido | Elementos clave |
|---|---|---|
| 1 | Información personal | Nombres, cédula, género, nacimiento, nacionalidad, estado civil, pareja |
| 2 | Domiciliaria y contacto | Cascada País → Provincia → Cantón, 2 correos, 2 teléfonos, dirección |
| 3 | Información adicional | Familiares en 2.º grado / 4.º de afinidad |
| 4 | Selección del cargo | Sector + respaldo de organización (PDF 4 MB) |
| 5 | Formación académica | Nivel, institución, título, área, SENESCYT, documento (PDF 2 MB) |
| 6 | Experiencia profesional | Tipo, cargo, fechas, funciones (máx. 1000), **duración calculada**, documento |
| 7 | Publicaciones | Título, tipo, medio, fecha, relación con el campo, documento |
| 8 | Capacitación | Tipo, nombre, institución, fechas, horas, documento |
| 9 | Inhabilidades | 13 preguntas Sí/No; si 1.9.7 es "Sí", pedir más datos |

**Comportamiento común**

- **Guardar**, **Editar**, **Eliminar** y **Guardar información** en cada
  sección.
- **Guardar progreso**: el borrador persiste y se recupera al reingresar
  (HU-09).
- Barra de progreso con las 9 secciones y su estado (completa, parcial,
  pendiente).

**Criterios de aceptación**

- [ ] La sección 2 implementa la **cascada**: si el país es Ecuador, provincia
      y cantón se eligen de la lista; si no, se escriben a mano.
- [ ] Si la nacionalidad no es **ecuatoriana**, el sistema **no permite
      continuar** y cierra la postulación (HU-06).
- [ ] La duración de la experiencia se muestra en **años, meses y días**
      (usando `calcularDuracionExperiencia`, ya implementado y con test).
- [ ] La sección 9 exige las 13 respuestas y pide los 4 campos extra cuando
      1.9.7 es "Sí".
- [ ] Los adjuntos validan el tamaño **antes** de subir y muestran el límite
      vigente (2 MB / 4 MB).
- [ ] El guardado es optimista con reversión si la API falla, o pesimista con
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

- [ ] "Enviar Postulación" muestra los **5 puntos** de consentimiento y
      declaración jurada, literales del RF.
- [ ] Sin marcar todos los puntos, el envío está deshabilitado.
- [ ] Tras enviar, la postulación queda **bloqueada**: no se edita y se
      muestra el hash.
- [ ] La vista de impresión incluye el espacio de **firma manuscrita**, los
      nombres, los apellidos y la cédula (HU-11).
- [ ] La vista de impresión tiene estilos de impresión (`@media print`) que
      ocultan la navegación.

**Esfuerzo estimado**: 2,5 días.

---

## 5. P2 — Deseables

| ID | Tarea | Beneficio |
|---|---|---|
| FT7 | Dashboard del analista con datos reales | EF-13 (fuera del parcial) |
| FT8 | Exportación de la matriz a `.xlsx` y PDF | EF-13 |
| FT9 | Pruebas E2E con Playwright | Detectar regresiones del recorrido |
| FT10 | Modo offline con aviso de conexión | Cada caída de red |
| FT11 | Internacionalización (es/en) | Escalabilidad futura |
| FT12 | Medir el bundle y cargar el formulario por secciones | Rendimiento en equipos lentos |
| FT13 | Atajos de teclado y navegación completa por teclado | Accesibilidad |

---

## 6. Cobertura de la rúbrica: plan de puntos

| Criterio | Hoy | Con FT1–FT3 | Con FT1–FT6 |
|---|---:|---:|---:|
| 4. SPA, componentes y estado | 2 | 4 | 6 |
| 5. Consumo de API y asincronía | 0 | 4 | 6 |
| 6. Reglas del negocio | 1 | 3 | 6 |
| **Subtotal frontend** | **3/18** | **11/18** | **18/18** |

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

**Ver también**: [backend-tasks.md](backend-tasks.md) ·
[`../architecture/ARQUITECTURA.md`](../architecture/ARQUITECTURA.md) ·
[`../api-contracts/API_CONTRACT.md`](../api-contracts/API_CONTRACT.md)
