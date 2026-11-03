# Auditoría del proyecto frente al examen y al RF v1.4

| Campo | Valor |
|---|---|
| Fecha | 2026-11-02 |
| Rama auditada | `develop` (último commit `1b07a58`) |
| Referencias | `EXAMEN.md` (enunciado y rúbrica) · `REQUERIMIENTO.md` (RF v1.4) |
| Herramientas | CodeGraph (463 archivos, 7.017 nodos, 18.044 aristas), lectura de código y documentación, historial de git |

Cada hallazgo indica la evidencia en el código, **qué dice la documentación del
proyecto sobre por qué se hizo así o por qué no se implementó**, y una
valoración de ese sustento:

- **Justificado**: existe una decisión documentada y coherente.
- **Parcial**: hay sustento, pero incompleto o con un costo no analizado.
- **Pendiente reconocido**: la documentación lo registra como trabajo por hacer, no como decisión.
- **Sin sustento**: no hay documento que lo explique.
- **Desactualizado**: la documentación describe un estado anterior del código.

---

## 1. Verificaciones ejecutadas

| Comprobación | Resultado |
|---|---|
| `go build ./...` | ✅ |
| `go vet ./...` | ✅ |
| `go test ./...` | ✅ todos los paquetes. Las pruebas de integración PostgreSQL se omiten sin `TEST_DATABASE_URL`; la CI sí las ejecuta con servicios PostgreSQL 18 y Redis 7 (`.github/workflows/ci.yml`) |
| `pnpm lint` | ✅ |
| `pnpm test` | ✅ 29 archivos, 356 pruebas |
| `pnpm build` | ✅ |

---

## 2. Puntaje estimado por criterio de la rúbrica

| # | Criterio | Máx. | Estimado | Resumen |
|---|---|---|---|---|
| 1 | Arquitectura Backend & Inyección de Dependencias | 8 | 7–8 | Capas handler → service → repository por bounded context y cableado en `cmd/api/main.go`. Ver H7, H8 y H12 |
| 2 | Persistencia Relacional & Transacciones | 8 | 6–8 | Transacciones reales con `BEGIN/COMMIT/ROLLBACK` y bloqueo de fila. Riesgo: formulario en una columna JSONB (H5) |
| 3 | Cache, Expiración & Rate Limiting (Redis) | 6 | 6 | Bloqueo 3 intentos / 10 min con script Lua atómico; caché de catálogos con TTL 24 h y respaldo PostgreSQL |
| 4 | Frontend SPA | 6 | 6 | Vite 8 + Tailwind 4 + shadcn + TanStack Query, features por dominio, loaders y validaciones |
| 5 | Consumo de API & Manejo Asíncrono | 6 | 5–6 | Guard de sesión con destino y 403 por rol, `ApiError` tipado con 429/`Retry-After`. Respaldo silencioso de catálogos (H3) |
| 6 | Cumplimiento de Reglas del Negocio | 6 | 4.5 | Bloqueo, nacionalidad con cierre en servidor y cascada validada en servidor. **Falta la validación dactilar simulada** (H1) |
| | **Total** | **40** | **~34–38** | |

---

## 3. Lo que cumple (evidencia)

- **Bloqueo de intentos (EF-11, criterio 3):** `internal/platform/cache/attempts.go`.
  `MaxLoginAttempts = 3`, `LockoutDuration = 10 min`, contador y bloqueo en un
  único script Lua (sin carreras), claves por digest de la cédula. Sin Redis el
  login **falla cerrado** (`main.go`, `UnavailableCache`).
- **Caché de catálogos:** `internal/platform/cache/catalog.go`, `CatalogTTL = 24h`,
  con prueba de acierto/fallo/expiración en `internal/catalog/service_test.go`.
- **Transacciones:** `internal/platform/db/db.go` (`TxManager.WithinTx`: `Begin`,
  `Rollback` diferido, `Commit`). `submission/postgres/repository.go` usa
  `Mutate` con bloqueo `FOR UPDATE`; el sello escribe postulación, eventos,
  documentos, outbox de correo y vínculo de privacidad en un solo commit.
- **Nacionalidad ecuatoriana:** `internal/eligibility/service.go:42` cierra la
  postulación (`status = closed`) **antes** de validar el formulario; el frontend
  muestra el bloqueo en `seccion-1-personal.tsx`.
- **Cascada validada en servidor:** `internal/submission/rules.go:144` rechaza
  provincia inexistente y cantón que no pertenece a la provincia.
- **Mensajes literales del RF:** `internal/identity/errors.go` («sus credenciales
  de ingreso son incorrectas», «la información ingresada es incorrecta»,
  «registro fallido, inténtelo nuevamente en 10 minutos»).
- **Rutas protegidas:** `components/guard-sesion.tsx` (carga sin parpadeo,
  `/login?destino=…`, vista 403 por rol) y layouts por rol.
- **Feedback de bloqueo:** `features/auth/alerta-bloqueo.tsx` con cuenta regresiva
  y `lib/api-error.ts` (`esBloqueoIntentos`, `retryAfterSeg`).
- **RF más allá del parcial:** EF-12 completo (9 secciones, adjuntos y cuotas,
  calculadora de experiencia, 5 consentimientos literales, código único y hash),
  EF-13 (dashboard, histórico y matriz XLSX/PDF) y EF-14 (banco de preguntas,
  sorteo, temporizador, PDF de resultado).

---

## 4. Hallazgos y su sustento documental

### H1 — No existe la validación dactilar / fecha de expedición simulada

- **Impacto:** criterio 6 (nombra «validación dactilar simulada») y módulo EF-11 del
  enunciado («simulación de validación dactilar y fecha de expedición»). Estimado: −1,5 pts.
- **Evidencia:** el registro y la recuperación usan OTP por correo
  (`internal/identity/service.go`, `POST /api/v1/auth/register/request|verify`).
- **Sustento documentado:**
  - `docs/decisions/ADR-008-registro-y-recuperacion-con-otp.md` (aceptada
    2026-10-31, «Decisión del responsable del proyecto»): *«El RF original exige
    validar la fecha de expedición o el código dactilar mediante Registro Civil. El
    proyecto no dispone de acceso a ese servicio.»* Decide explícitamente: *«No
    solicitar ni simular la validación de fecha de expedición o código dactilar.»*
  - `docs/prd/PRD.md:99` y riesgo R-01 (`PRD.md:404`): «No hay acceso a los
    sistemas del IESS… Registro Civil no se utiliza: módulo 10 + OTP».
  - `docs/tasks/backend-tasks.md:848`: «La decisión de no usar Registro Civil ya
    está resuelta por ADR-008: no queda como bloqueo ni se vuelve a exigir».
  - `docs/trd/TRD.md` §6.1 y `AGENTS.md` repiten la regla.
- **Historial:** `3dc98b0` (2026-10-31, T2) implementó la identidad con OTP; ese
  mismo commit marcó en las tareas como hecho «`POST /auth/register` valida la
  fecha de expedición o el código», aunque el código hacía otra cosa. `ad72830`
  formalizó ADR-008 sin más explicación en el mensaje.
- **Valoración: Parcial.** El motivo (sin acceso a Registro Civil) está
  documentado, pero:
  1. Ningún documento explica por qué **no se simuló**, cuando para Historial
     Laboral sí se usa una fuente simulada rotulada (T17, `PRD.md:274`,
     `internal/eligibility/simulated/`).
  2. La decisión se evaluó contra el RF, no contra la rúbrica: ningún documento
     menciona el costo en el criterio 6.
- **Acción propuesta:** añadir un paso previo al OTP (cédula + fecha de expedición
  **o** código dactilar de 10 dígitos) verificado por un puerto `CivilRegistry`
  con adaptador `simulated` en el servidor, igual que la elegibilidad. Los fallos
  cuentan para el bloqueo de 3 intentos / 10 min, con los mensajes literales del
  RF, y la UI lo rotula «Simulación — sin conexión real con Registro Civil».
  Revisar ADR-008 (estado «Modificada») o sustituirla por ADR-010. **Acordarlo con
  el equipo**: revierte una decisión registrada como del responsable del proyecto.

### H2 — La cascada del frontend no consume la API para provincias y cantones

> **Estado (2026-11-02): corregido en el árbol de trabajo, sin commit.** La sección 2 usa `useCatalogosProvincias` y `useCatalogosCantones`, con estados de carga.

- **Evidencia:** `features/formulario/seccion-2-contacto.tsx:64` usa
  `PROVINCIAS_ECUADOR` de `data/ecuadorGeo.ts`. Los países sí vienen de la API
  (`useCatalogosPaises`). Los hooks `useCatalogosProvincias` y
  `useCatalogosCantones` (`features/postulaciones/queries.ts:108,120`) existen
  pero no tienen consumidores. En la demo, la caché Redis de provincias y
  cantones no se ejercita desde la interfaz.
- **Sustento documentado:**
  - `docs/tasks/frontend-tasks.md:665` (EF-12 1.2): «Catálogos backend completos;
    **integración de vistas pendiente**».
  - `docs/prd/PRD.md:373`: «🟡 Parcial (solo frontend)».
  - `docs/database/SCHEMA.md` §3: la geografía del backend es «idéntica a
    `apps/frontend/src/data/ecuadorGeo.ts`», así que los datos no divergen.
- **Historial:** la sección 2 se creó en `8fb265b` (FT5, 2026-10-31) con el
  catálogo local; los endpoints de catálogo llegaron después (T7).
- **Valoración: Pendiente reconocido.** No es una decisión: la documentación lo
  marca como integración pendiente.
- **Acción propuesta:** sustituir `PROVINCIAS_ECUADOR` por los dos hooks, con
  estados de carga y error en los `Select`.

### H3 — El cliente de catálogos cae en silencio al catálogo local

> **Estado (2026-11-02): corregido en el árbol de trabajo, sin commit.** El cliente propaga el error; la sección 2 usa la lista local solo como respaldo, con aviso y botón «Reintentar».

- **Evidencia:** `lib/clientes/catalogos.ts` captura cualquier error (`catch {}`)
  y devuelve `PAISES`, `LISTA_PROVINCIAS` o `PROVINCIAS_ECUADOR` sin avisar al
  usuario ni registrar el fallo. Una caída de la API o una sesión vencida pasan
  inadvertidas.
- **Sustento documentado:** solo el JSDoc del propio archivo: «Intenta consultar
  la caché de la API y si no responde o está desconectada, recurre al catálogo
  local». No hay ADR ni tarea que lo respalde. La decisión FT14/FT15 de
  `AGENTS.md` va en sentido contrario para otros datos: sin conexión o con datos
  viejos, la UI **bloquea** escrituras en lugar de seguir con datos locales.
- **Valoración: Sin sustento** (intención de diseño no documentada, que además
  contradice el criterio aplicado en FT14/FT15).
- **Acción propuesta:** mantener el respaldo, pero exponerlo (por ejemplo, un
  aviso «Mostrando catálogo local; no se pudo consultar el servidor») y no
  ocultar errores 401/403.

### H4 — El README no declara Redis como prerrequisito

- **Evidencia:** la tabla de prerrequisitos de `README.md` lista Go, Node.js,
  pnpm, Docker y PostgreSQL, pero no Redis ni su versión. El enunciado lo exige.
- **Sustento documentado:** ninguno a favor. La causa es documentación
  desactualizada: `docs/deployment/LOCAL_DEPLOYMENT.md:23-27` todavía dice
  «Redis (opcional) 7 — Pendiente de implementar… la versión de Redis no es
  requisito todavía» y su tabla de estado (línea 342) marca Redis como
  «❌ Pendiente». Redis ya está implementado, en `compose.yaml` y en la CI
  (`redis:7-alpine`).
- **Valoración: Desactualizado.**
- **Acción propuesta:** añadir «Redis | 7 | bloqueo de intentos y caché de
  catálogos» al README y corregir `LOCAL_DEPLOYMENT.md`.

### H5 — El formulario de postulación se guarda en una columna JSONB

- **Impacto:** criterio 2 pide «modelado normalizado y transacciones para
  inserciones multi-tabla en el guardado de la postulación».
- **Evidencia:** `submissions.content JSONB` guarda las 9 secciones. Familiares,
  títulos, experiencias, publicaciones y capacitaciones no tienen tablas hijas.
  Las columnas consultables (código, estado, postulante, hash) sí son propias, y
  el sello sí es multi-tabla.
- **Sustento documentado:**
  - `docs/decisions/ADR-003-postgres-sqlc-goose.md:77`: «`jsonb` para el
    contenido del formulario: el RF evoluciona sin migraciones por cada campo nuevo».
  - `docs/trd/TRD.md` §4.1: «Los datos que sí se consultan… tienen columna propia».
  - `docs/database/SCHEMA.md:177`: «no se fuerza una migración por cada campo nuevo».
  - `AGENTS.md` (decisión 2026-10-31): el hash del envío se calcula sobre el JSON
    canónico, lo que favorece guardar el documento completo.
- **Valoración: Parcial.** La decisión está justificada técnicamente, pero ningún
  documento analiza su efecto en el criterio 2. ADR-003 incluso afirma que es
  «justo lo que la rúbrica evalúa», sin abordar la palabra «normalizado».
- **Acción propuesta:** o bien preparar la defensa para la sustentación (snapshot
  sellado inmutable + hash canónico, validación tipada en Go, columnas propias
  para lo que se consulta, transacción multi-tabla real en el sello), o bien
  normalizar al menos las secciones repetibles en tablas hijas escritas en la
  misma transacción.

### H6 — Catálogos en una sola tabla `catalogs` con `kind`

- **Evidencia:** `sql/migrations/00008_catalogs.sql`. No hay tablas separadas
  `paises`, `provincias` y `cantones`.
- **Sustento documentado:** `docs/database/SCHEMA.md` §3: relaciones reales por
  `parent_id` UUID (provincia → Ecuador, cantón → provincia), `CHECK` por tipo,
  clave única `(kind, parent_key, value)` que hace idempotente la semilla y
  distingue cantones homónimos de provincias distintas. `AGENTS.md` (T7) fija la
  semilla reproducible.
- **Valoración: Justificado.** Conviene explicarlo en la sustentación como
  jerarquía autorreferenciada con integridad referencial.

### H7 — Los handlers dependen del servicio concreto (`*Service`)

- **Evidencia:** `submission/handler.go:24`, `catalog/handler.go:12` y el resto.
- **Sustento documentado:** `docs/decisions/ADR-002` exige interfaces entre
  **casos de uso y adaptadores de persistencia** (regla 2); no dice nada sobre la
  frontera handler → servicio.
- **Valoración: Parcial.** Es idiomático en Go y no viola ADR-002, pero no está
  documentado. Un evaluador puede leerlo como «dependencias acopladas
  directamente» (nivel «Bueno» del criterio 1).
- **Acción propuesta:** documentarlo en ADR-002 (los handlers son el borde y se
  prueban con el servicio real y repositorios falsos), o declarar interfaces por
  handler si se quiere blindar el criterio.

### H8 — `identity.Service` depende de tipos concretos de infraestructura

- **Evidencia:** `internal/identity/service.go:46-47` declara
  `attempts *cache.AttemptsStore` y `otpStore *cache.OTPStore`, tipos del paquete
  `internal/platform/cache`.
- **Sustento documentado:** ninguno. **Contradice** ADR-002 regla 2 («Los casos
  de uso dependen solo de interfaces declaradas en su propio paquete») y
  `AGENTS.md` («El dominio no importa infraestructura»). El resto de contextos sí
  lo cumple (por ejemplo, `catalog.Service` recibe interfaces `Repository` y `Cache`).
- **Valoración: Sin sustento** (incumple la regla propia del proyecto).
- **Acción propuesta:** declarar en `identity/` las interfaces `AttemptsStore` y
  `OTPStore` con los métodos usados y recibirlas en `ServiceDependencies`.

### H9 — La IP del cliente se toma del primer valor de `X-Forwarded-For`

- **Evidencia:** `internal/identity/handler.go:347` usa `parts[0]` de
  `X-Forwarded-For`. `deploy/nginx.conf:40` añade la IP real al **final** con
  `$proxy_add_x_forwarded_for`, así que el primer valor lo controla el cliente y
  puede falsificarse. La IP queda registrada en la sesión.
- **Sustento documentado:** ninguno. `PRODUCTION_DEPLOYMENT.md` solo trata
  `X-Forwarded-Proto` para la cookie `Secure`.
- **Mitigación existente:** el bloqueo de intentos es por cédula, no por IP, así
  que no puede eludirse por esta vía.
- **Valoración: Sin sustento.**
- **Acción propuesta:** preferir `X-Real-IP` (que nginx fija con
  `$remote_addr`) o tomar el último salto de confianza.

### H10 — El bloqueo del login muestra el mensaje del registro

- **Evidencia:** `ErrAccountLocked = "registro fallido, inténtelo nuevamente en 10
  minutos"` se usa en el login.
- **Sustento documentado:** `docs/prd/PRD.md:153` y `docs/tasks/backend-tasks.md:98,168`
  asignan ese literal al caso «Cuenta bloqueada». El RF solo define el texto para
  el registro y no da uno para el login.
- **Valoración: Justificado** (se reutiliza el único literal que da el RF).

### H11 — Restos del modelo dactilar en el esquema y en los tipos del frontend

- **Evidencia:** `00001_init.sql:8` (`metodo_validacion DEFAULT 'fecha_expedicion'`),
  `lib/clientes/auth.ts:63` y `types/concurso.ts:136`
  (`"fecha_expedicion" | "codigo_dactilar"`).
- **Sustento documentado:** `docs/database/SCHEMA.md:510-514` explica el
  `DEFAULT` del esquema: es DDL histórico, el servicio asigna `otp_correo` de forma
  explícita y no implica validación con Registro Civil. Los tipos del frontend no
  tienen explicación.
- **Valoración:** esquema **Justificado**; tipos del frontend **Sin sustento**.
  Si se implementa H1, estos valores vuelven a tener uso.

### H12 — chi en lugar de Fiber/Gin

- **Evidencia:** `go.mod` usa `go-chi/chi/v5`. El enunciado dice «Go - Fiber/Gin».
- **Sustento documentado:** `docs/decisions/ADR-001-chi-en-vez-de-gin-fiber.md`:
  chi opera sobre `*http.Request` nativo, interoperable con la stdlib y con
  `multipart` para los PDF; Fiber obliga a `fasthttp` y a cargar el cuerpo en
  memoria. Incluye un guion para la sustentación. También `TRD.md` §2.1 y riesgo R-02.
- **Valoración: Justificado.**

### H13 — `ARQUITECTURA.md` y el PRD describen Redis y la cascada como pendientes

- **Evidencia:** `docs/architecture/ARQUITECTURA.md:336` («Implementación
  pendiente… criterio 3, 6 pts») y tabla de estado (línea 399: «Redis… Pendiente»).
  También describe claves `identity:login-attempts:{cédula}` que no coinciden con
  las reales (`identity:auth:{digest}:attempts|lock`). `PRD.md:373` marca la
  cascada como «solo frontend» aunque el backend la valida.
- **Sustento documentado:** ninguno. Último cambio del archivo: `5184ac1`
  (2026-10-31), anterior a la implementación de Redis.
- **Valoración: Desactualizado.** Riesgo en la sustentación si el evaluador lee
  la documentación.
- **Acción propuesta:** actualizar la sección 9 y la tabla de estado.

### H14 — Las tareas enlazan un enunciado que no está en el repositorio

- **Evidencia:** `docs/tasks/backend-tasks.md:12` enlaza «enunciado del examen
  práctico» a `docs/prd/PRD.md`, que no contiene el texto ni la rúbrica literal.
- **Sustento documentado:** ninguno. El enunciado completo vive ahora en
  `EXAMEN.md`, que está en `.gitignore`.
- **Valoración: Sin sustento** (menor).
- **Acción propuesta:** corregir el enlace o incluir la rúbrica en el PRD.

### H15 — Entregables: `.env.example` y script `.sql`

- **Evidencia:** no hay `.env.example` en la raíz y no hay un único `.sql` con
  DDL y semillas juntos.
- **Sustento documentado:** `README.md` (sección «Entregables del examen») ubica
  ambos: `apps/backend/.env.example` y `deploy/.env.example`, y
  `apps/backend/sql/` (migraciones y seeds). `sql/queries/schema.sql` contiene el
  DDL completo **con** los `INSERT` de catálogos, y `sql/seeds/catalogs.sql` es
  autónomo y transaccional (`SCHEMA.md` §3).
- **Valoración: Justificado** por la estructura de monorepo. Conviene que el
  README nombre expresamente `schema.sql` y `seeds/catalogs.sql` como «el archivo
  `.sql` del entregable».

### H16 — Secciones del RF sin desarrollar

- **Evidencia:** EF-12 anuncia «Formación adicional» y «Referencias Personales»,
  pero sus apartados 1.1–1.10 no las detallan; el proyecto implementa las 9
  secciones desarrolladas.
- **Sustento documentado:** `docs/tasks/backend-tasks.md:845`: «Aclarar su
  alcance antes de crear una décima sección». También la diferencia entre las
  ciudades del RF y el catálogo nacional usado (221 cantones).
- **Valoración: Justificado** (ambigüedad del RF registrada como pregunta abierta).

### H17 — Interoperabilidad con el IESS simulada

- **Evidencia:** `internal/eligibility/simulated/` (Historial Laboral para
  afiliados, jubilados y empleadores).
- **Sustento documentado:** `PRD.md:84,274,404` (R-01), `AGENTS.md` (T17): el
  adaptador se identifica como simulación y la falta de dato es 503, no un cierre.
- **Valoración: Justificado.** Es el precedente que se puede usar para H1.

---

## 5. Prioridad sugerida para subir puntos

1. **Añadir la simulación dactilar o de fecha de expedición, rotulada como
   simulación, y revisar ADR-008** (H1). Puede sumar unos 1,5 pts en el criterio 6.
   Requiere acuerdo del equipo.
2. **Conectar la cascada a `useCatalogosProvincias` y `useCatalogosCantones`, y
   mostrar un aviso cuando se use el catálogo local** (H2, H3). Hace visible en la
   demo la caché Redis de catálogos y mejora el criterio 5.
3. **Agregar Redis al README** (H4) y corregir `LOCAL_DEPLOYMENT.md` y
   `ARQUITECTURA.md` (H13). Es un entregable explícito del enunciado.
4. **Preparar la defensa del JSONB, o normalizar al menos las secciones
   repetibles en tablas hijas** (H5).

Mejoras menores, sin impacto directo en la rúbrica pero baratas:

- Interfaces para `AttemptsStore` y `OTPStore` en `identity/` (H8): cumple la
  regla propia de ADR-002 y refuerza el criterio 1.
- `X-Real-IP` en lugar del primer valor de `X-Forwarded-For` (H9).
- Corregir el enlace al enunciado en `backend-tasks.md` (H14) y nombrar el `.sql`
  del entregable en el README (H15).

## 6. Resumen del sustento

| Hallazgo | Tema | Valoración |
|---|---|---|
| H1 | Validación dactilar simulada ausente | Parcial |
| H2 | Cascada con catálogo local | Pendiente reconocido |
| H3 | Respaldo silencioso de catálogos | Sin sustento |
| H4 | Redis fuera del README | Desactualizado |
| H5 | Formulario en JSONB | Parcial |
| H6 | Tabla única `catalogs` | Justificado |
| H7 | Handlers → `*Service` concreto | Parcial |
| H8 | `identity.Service` → tipos de `platform/cache` | Sin sustento |
| H9 | IP desde `X-Forwarded-For[0]` | Sin sustento |
| H10 | Mensaje de bloqueo reutilizado en login | Justificado |
| H11 | Restos de `fecha_expedicion`/`codigo_dactilar` | Justificado (esquema) / Sin sustento (frontend) |
| H12 | chi en vez de Fiber/Gin | Justificado |
| H13 | Documentación de Redis y cascada desactualizada | Desactualizado |
| H14 | Enlace al enunciado inexistente | Sin sustento |
| H15 | `.env.example` y `.sql` repartidos | Justificado |
| H16 | Secciones del RF sin desarrollar | Justificado |
| H17 | Interoperabilidad IESS simulada | Justificado |
