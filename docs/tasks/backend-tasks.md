# Tareas del backend

**Sistema de Registro y Postulación al Directorio del BIESS**
Lista de trabajo del backend: rúbrica del parcial y cobertura integral del RF v1.4

| Campo | Valor |
|---|---|
| Documento | backend-tasks v1.1 |
| Ámbito | `apps/backend/` |
| Fuente funcional | [RF v1.4](../requerimiento.txt), EF-1 y EF-11 a EF-14 |
| Adaptación de identidad | [ADR-008: OTP por correo](../decisions/ADR-008-registro-y-recuperacion-con-otp.md) |
| Rúbrica | [enunciado del examen práctico](../prd/PRD.md) |

---

## 1. Cómo leer esta lista

Cada tarea tiene:

- **ID** — identificador estable (`T1` a `T28`) para seguirla en el PR y en el
  issues.
- **Criterio** — criterio de la rúbrica que la tarea cubre y los puntos que
  puede sumar.
- **Prioridad** — `P0` (bloquea la rúbrica) · `P1` (importante) ·
  `P2` (deseable).
- **Estado** — `✅` hecho · `🟡` parcial · `⬜` pendiente.

Las **prioridades P0** son las que más puntos mueven. El orden de ejecución
recomendado es: **T1 (Redis) → T2 (identidad) → T3 (formulario) → T4
(adjuntos)**.

---

**Alcance de esta revisión (2026-10-31):** las tareas originales se concentraban en
el parcial. EF-13, EF-14, méritos, psicotécnica y reconsideraciones son obligatorios
para cubrir el RF completo, aunque puedan ejecutarse después de la entrega inicial.
La sección 8 añade esas tareas con prioridad **P1-RF**; los pendientes de identidad,
elegibilidad, permisos y apertura se señalan como **P0-RF**. P2 queda para mejoras no exigidas por
el RF. Una casilla pendiente describe planificación, no una función implementada.

El proyecto usa **OTP por correo** en registro y recuperación: no se requieren
fecha de expedición ni código dactilar. El RF original se conserva sin modificar;
esta adaptación se documenta en ADR-008. La lista de tareas distingue lo literal
del RF, la adaptación aprobada y las definiciones que todavía faltan.

## Corrección transversal del informe T1–T9 (2026-10-31)

Se corrigen OTP por finalidad/desafío/cuenta/correo, consumo y bloqueo atómicos,
TTL, reenvíos, fallo cerrado de identidad sin Redis y propagación de errores.
`00011_mail_outbox.sql` añade correo cifrado durable y reintentos; la cola de
confirmación participa del commit del sello y SMTP se ejecuta después. Se
requiere `MAIL_OUTBOX_KEY`; no se afirma entrega por encolar. Hay integración CI
separada, pero no se da por ejecutada localmente sin variables de pruebas.
Detalle: [IDENTITY_SECURITY.md](../api-contracts/IDENTITY_SECURITY.md) e informe
corregido en la raíz. Las dependencias futuras de T8/T9 no se marcan cerradas.
T23 conserva integración de otras etapas y administración por rol.

## 2. Estado por criterio de la rúbrica

| # | Criterio | Puntos máximos | Cobertura técnica y pendientes |
|---|---|---:|---|
| 1 | Arquitectura e inyección de dependencias | 8 | Base disponible; completar contextos y permisos de las nuevas capacidades |
| 2 | Persistencia y transacciones | 8 | T4/T5 implementadas; ampliar trazabilidad, convocatoria y expediente |
| 3 | Caché, expiración y rate limiting | 6 | T1 disponible; catálogos T7 y consolidación OTP T16 pendientes |
| 4–5 | SPA y consumo asíncrono | 12 | Ver frontend-tasks.md; son criterios compartidos con la integración |
| 6 | Reglas del negocio | 6 | Parcial frente al RF completo: ver matriz de la sección 9 |

Los puntos son los máximos de la rúbrica, no una calificación otorgada ni una
medición de cumplimiento del RF. La existencia de T1–T5 no cierra EF-13 o EF-14.

---

## 3. P0 — Bloqueantes de la rúbrica

### T1 · Cliente Redis: bloqueo de intentos y caché de catálogos

> **Criterio 3 — Cache, Expiración & Rate Limiting (6 pts)**

Es la tarea de mayor retorno: implementa **completo** un criterio de 6 puntos
que hoy está en cero.

**Piezas**

| Pieza | Ubicación | Contenido |
|---|---|---|
| Cliente Redis | `internal/platform/cache/redis.go` | Conexión con `go-redis`, ping, cierre ordenado |
| Puerto de caché | `internal/platform/cache/cache.go` | Interface `Cache` (Get, Set con TTL, Incr, Delete) |
| Contador de intentos | `internal/platform/cache/attempts.go` | `Incr` + TTL atómico, lectura del contador y del bloqueo |
| Caché de catálogos | `internal/platform/cache/catalog.go` | Get-or-set con TTL de 24 h |

**Bloqueo/OTP corregidos; T16 conserva el cierre de integración del ciclo completo**

1. Cada intento fallido de autenticación incrementa
   `identity:login-attempts:{cédula}`.
2. Al llegar a **3 intentos**, se bloquea la cuenta **10 minutos**
   (`identity:lockout:{cédula}` con TTL de 600 s).
3. Con la cuenta bloqueada, la API responde `429` con el mensaje del RF:
   *"registro fallido, inténtelo nuevamente en 10 minutos"*.
4. Pasados los 10 minutos, el bloqueo **desaparece solo** (TTL), sin ningún
   proceso que lo limpie.
5. Un intento **exitoso** reinicia el contador (`DEL`).
6. El estado vive **fuera del proceso**: sobrevive a reinicios y escala
   horizontalmente.

**Claves y TTL**

| Clave | Tipo | TTL |
|---|---|---|
| `identity:login-attempts:{cédula}` | Contador | 10 min |
| `identity:lockout:{cédula}` | Marca | 10 min |
| `catalog:countries` | JSON | 24 h |
| `catalog:provinces:{país}` | JSON | 24 h |
| `catalog:cities:{país}:{provincia}` | JSON | 24 h |

**Criterios de aceptación**

- [x] El servicio de Redis se levanta en `compose.yaml` (desarrollo) y en
      `deploy/compose.yaml` (producción).
- [x] `REDIS_URL` en la configuración; la API arranca **degradada** si Redis
      no está disponible (fail-open en el login, fail-loud en el arranque con
      `DB_REQUIRED`).
- [x] `IncrementAttempts` es **atómico**: usa `INCR` + `EXPIRE` en una
      transacción o `pipeline`, nunca dos comandos sueltos.
- [x] Los tests usan un servidor Redis efímero o un doble de prueba; no
      requieren Redis para probar las reglas puras.
- [x] La caché de catálogos reduce las consultas a PostgreSQL: se puede
      observar con `INFO commandstats` en Redis.

**Dependencias**: ninguna. **Esfuerzo estimado**: 2 días.

---

### T2 · Contexto de identidad: registro, login y sesión

> **Criterio 6 — Reglas del negocio (6 pts) · Criterio 5 — Rutas protegidas**

Base de autenticación implementada con OTP. La consolidación del desafío y sus
controles corresponde a T16; no depende de acceso al Registro Civil.

**Piezas**

| Pieza | Ubicación | Contenido |
|---|---|---|
| Modelo | `internal/identity/` | `IDCard` (ya existe), `Applicant`, `Session` |
| Errores | `internal/identity/errors.go` | Errores con código estable |
| Servicio | `internal/identity/service.go` | `Register`, `Login`, `Logout`, `RestorePassword` |
| Hash de contraseña | `internal/platform/password/argon2.go` | Argon2id (64 MiB, t=3, p=2), sal de 16 bytes |
| Repositorio | `internal/identity/repository.go` + `postgres/` | Postulantes, sesiones y recuperación |
| Handler | `internal/identity/handler.go` | Rutas `/api/v1/auth/*` |
| Migración | `sql/migrations/00003_identity.sql` | `password_hash`, tabla `sessions`, tabla `password_resets` |

**Reglas que debe cumplir**

| Regla | Origen | Comportamiento |
|---|---|---|
| RN-01 | EF-11 | Cédula válida por módulo 10 (ya implementado en `NewIDCard`) |
| RN-02 | RF | Contraseña: ≥ 8 caracteres, mayúscula, minúscula, número y símbolo |
| RN-03 | EF-11 | 3 intentos fallidos → bloqueo de 10 minutos (con T1) |
| RN-04 | RF | Argon2id con sal aleatoria; nunca en claro |
| RN-08 | RF | Token de sesión en cookie `HttpOnly`; nunca en `localStorage` |

**Mensajes al usuario (literales del RF)**

| Situación | Mensaje |
|---|---|
| Datos de identidad incorrectos | *"la información ingresada es incorrecta"* |
| Credenciales incorrectas | *"sus credenciales de ingreso son incorrectas"* |
| Cuenta bloqueada | *"registro fallido, inténtelo nuevamente en 10 minutos"* |

**Criterios de aceptación**

- [x] `POST /api/v1/auth/register/request` recibe cédula y correo y envía un OTP.
- [x] `POST /api/v1/auth/register/verify` verifica el OTP y crea la cuenta (`201`)
      con método `otp_correo` y rol inicial `postulante`.
- [x] `POST /api/v1/auth/login` devuelve la cookie `session` con `HttpOnly` y
      `SameSite=Lax`; en producción añade `Secure`.
- [x] El token **no** se persiste en claro: en `sessions` solo su SHA-256.
- [x] El login responde `429` con `intentos_excedidos` cuando hay bloqueo.
- [x] `GET /api/v1/auth/session` responde `200` con el perfil o `401`.
- [x] `POST /api/v1/auth/password/forgot` envía un OTP al correo registrado;
      `POST /api/v1/auth/password/reset` lo recibe con la contraseña nueva.
- [x] Existe un middleware de autenticación (`RequireSession`) reutilizable
      por los demás contextos.
- [x] Tests: módulo 10, fortaleza de contraseña, y el flujo de login con
      bloqueo tras 3 intentos.

**Dependencias**: T1 (para el bloqueo). **Esfuerzo estimado**: 3 días.

---

### T3 · Contexto de postulación: las 9 secciones del EF-12

> **Criterio 6 — Reglas del negocio (6 pts)**

**Backend T3 completado (2026-10-31)**: nueve secciones tipadas, precarga desde
la cuenta OTP, definición compartida de catálogos/campos, validación Go/TypeScript,
progreso, guardado parcial, consentimientos al guardar y sello transaccional.
La validación de elegibilidad y el cierre persistente pertenecen a T17; las vistas
que consumen estas capacidades se implementan en FT5/FT6.

**Piezas**

| Pieza | Ubicación | Contenido |
|---|---|---|
| Secciones | `internal/submission/sections.go` | Tipos de las 9 secciones del EF-12 |
| Reglas | `internal/submission/rules.go`, `records.go` | Validación parcial/final e IDs estables |
| Contrato de captura | `internal/submission/definition.json`, `definition.go` | Países, opciones, nueve secciones y trece preguntas |
| Cuenta OTP | `internal/submission/account.go` | Precarga y vínculo de cédula/correo con la cuenta |
| Servicio | `internal/submission/service.go` | `SaveSection`, `Submit` |
| Hash y envío | `internal/submission/submission.go` | `MarkAsSent` ya existe |

**Reglas del RF que deben validarse en el servidor**

| Regla | Origen | Comportamiento |
|---|---|---|
| Nacionalidad no ecuatoriana → cierra la postulación | EF-12 1.1.6 | `422 regla_de_negocio`; no permite enviar |
| Cascada País → Provincia → Cantón | EF-12 1.2.1–1.2.3 | Si el país es Ecuador, provincia y cantón deben existir en el catálogo |
| Si el país es otro, provincia y cantón son texto libre | EF-12 1.2.3 | Se valida el formato, no la existencia |
| Elegibilidad por sector | EF-12 1.4.1–1.4.3 | Afiliado / jubilado / empleador; mensaje del RF y cierre automático |
| Funciones de la experiencia: máx. 1000 caracteres | EF-12 1.6 | Validación de longitud |
| Duración calculada en años, meses y días | EF-12 1.6 | Misma regla que el frontend, verificada por test |
| Inhabilidades: 13 respuestas Sí/No | EF-12 1.9 | Las 13 son obligatorias; si 1.9.7 es "Sí", se piden institución, cargo, fecha de fin y motivo |
| Consentimiento: 5 puntos literales | EF-12 1.10 | El envío exige haber aceptado los 5 puntos |
| RN-05 | TRD | Hash SHA-256 sobre el JSON canónico (ya implementado) |
| RN-06 | TRD | Una postulación enviada no se edita ni se elimina (ya implementado) |

**Criterios de aceptación**

- [x] `POST /api/v1/submissions/{id}/sections/{n}` guarda una sección y
      devuelve el progreso (9 secciones, con cuáles están completas).
- [x] `POST /api/v1/submissions/{id}/submit` sella: hash, `submittedAt`,
      estado `submitted` y correo de confirmación.
- [x] Validador puro del frontend en `src/utils/validar-formulario.ts` y reglas Go
      contrastados con 67 casos de negocio compartidos de las nueve secciones.
      La conexión de ese validador a las vistas corresponde a FT5.
- [x] Los 5 puntos de consentimiento se guardan y son verificables.
- [x] La postulación enviada es inmutable: `PUT` y `DELETE` devuelven `409`.
- [x] Tests de cada regla con los casos de borde del RF.
- [x] Precargar nombres/apellidos/cédula y correo principal desde la cuenta OTP.
      Cédula y correo principal no se sustituyen por los de otra cuenta mediante
      Create/PUT/SaveSection/Submit. Los nombres siguen siendo declarativos.
      No se inventan nacimiento/nacionalidad ni se acepta certificación civil del cliente.
- [x] Mantener países del RF, parentescos, instituciones, niveles, áreas, tipos y
      trece preguntas literales en `definition.json`; usarlos en Go y TypeScript.
      `GET /api/v1/submissions/form-definition` los entrega con sesión. La
      persistencia/caché del contexto global de catálogos sigue en T7.
- [x] Publicar el contrato vigente de campos obligatorios, opcionales y condicionales:
      borradores incompletos, correo secundario opcional, pareja/remoción condicionales,
      lista de capacitación vacía declarada y firma digital no exigida. No se añadieron
      edades, puntajes ni una décima sección para resolver vacíos del RF por suposición.
- [x] Entregar y persistir los cinco literales/versiones también al guardar. El servidor
      descarta textos/versiones/fechas de aceptación inventados y solo fecha la
      aceptación al sellar. La representación visual e impresión siguen en T19/FT6.


**Verificación de este cierre**: tests de los módulos Go submission/document/identity
correctos y 68 tests Vitest del validador EF-12 correctos, incluidos los mismos 67
casos de negocio que usa Go. Cubren campos pendientes, condicionales, catálogos,
fechas, IDs duplicados, límites Unicode, declaraciones y formas JSON inválidas.
Las pruebas de cuenta/HTTP cubren precarga, sustitución de identidad/correo,
certificación civil falsa y conservación del snapshot. No se ejecutaron servicios
externos ni pruebas PostgreSQL sin sus variables de integración.

**Dependencias**: T2 (sesión). **Esfuerzo estimado**: 4 días.

---

### T4 · Contexto de documentos: adjuntos PDF

> **Criterio 6 — Reglas del negocio (6 pts) · Criterio 2 — Persistencia**

**Backend implementado (2026-10-31)**: carga PDF, metadatos transaccionales, cuota por cuenta, respaldos verificables al enviar, descarga temporal local/S3 (RustFS) y cola persistente de bajas.

**Límites del RF**

| Documento | Tamaño máximo |
|---|---|
| Respaldo de organización (sección 4) | 4 MB |
| Formación académica (5), experiencia (6), publicaciones (7), capacitación (8) | 2 MB |
| **Total por postulante** | 50 MB |

**Piezas**

| Pieza | Ubicación | Contenido |
|---|---|---|
| Modelo | `internal/document/document.go` | Metadatos del adjunto |
| Servicio | `internal/document/service.go` | Validación de MIME, tamaño por tipo y total |
| Puerto | `internal/document/repository.go` | Persistencia de metadatos |
| Handler | `internal/document/handler.go` | `multipart` con `chi` (ventaja de ADR-001) |
| Migración | `sql/migrations/00004_documents.sql` | Tabla `documents` |
| Almacenamiento | `internal/platform/storage/` | Interface + backend local + adaptador S3 compatible con RustFS |

**Criterios de aceptación**

- [x] Se acepta solo `application/pdf`; el resto devuelve `415`.
- [x] El tamaño por tipo y el total de 50 MB se verifican en el servidor.
- [x] La clave de almacenamiento es **no adivinable**:
      `{submission_id}/{section}/{uuid}.pdf`.
- [x] Se guarda el SHA-256 del archivo para verificar su integridad.
- [x] La descarga usa una URL prefirmada de corta duración; la API no hace
      proxy del binario.
- [x] Los adjuntos de una postulación enviada no se pueden eliminar.
- [x] Test del límite total con varios archivos.

**Verificación**: pasan las pruebas de reglas, HTTP, cuota concurrente con dobles, referencias y firma SigV4 oficial compatible con RustFS/S3. RustFS está integrado en `compose.yaml` como servicio de almacenamiento de objetos S3 (`rustfs/rustfs:latest` en puertos 9000 y 9001). Los tests PostgreSQL se habilitan con `DOCUMENT_TEST_DATABASE_URL` y fueron validados con base de datos real. El SMTP preexistente falla en local por sockets bloqueados y `-race` requiere CGO.

**Dependencias**: T3. **Esfuerzo estimado**: 3 días.

---

## 4. P1 — Importantes

### T5 · Transacción real en el guardado multi-tabla

> **Criterio 2 — Persistencia y transacciones (8 pts)**

T3 ya utiliza `TxManager.WithinTx` y bloqueo de fila para guardar y sellar la postulación. T5 amplía esa transacción al guardado **multi-tabla** de documentos y eventos cuando existan.

- [x] `SaveSubmission` inserta cabecera, secciones, documentos y eventos de
      auditoría dentro de **una** transacción.
- [x] Un fallo intermedio provoca `ROLLBACK` y **no** deja datos huérfanos.
- [x] Test de integración: se fuerza un error al insertar documentos y se
      comprueba que la cabecera tampoco queda.

**Verificación**: prueba de integración multi-tabla en `apps/backend/internal/document/postgres/submission_integration_test.go` (`TestSaveSubmissionCommitsAllTablesOrRollsBackOnDocumentFailure`) validada contra PostgreSQL en Docker: comprueba la inserción atómica de cabecera, 2 documentos y 1 evento de auditoría en `submission_events`, y verifica que un fallo en la inserción de documentos revierte cabecera, documentos y eventos sin dejar datos huérfanos.

**Esfuerzo estimado**: 1 día.

### T6 · Middleware de autenticación y de los tres roles

> **RF EF-1: actores y permisos; EF-13 y EF-14. Prioridad P0-RF.**

- [x] `RequireSession` resuelve la cookie y adjunta el postulante al contexto.
- [x] Persistencia de roles `postulante` y `analista` y campo `rol` en sesión.
- [x] Restricción ampliada con `administrador` (Comisión de Selección) y política
      explícita por operación en `identity/permissions.go`, con denegación por defecto.
- [x] `RequireRole` / `RequirePermission` y protección de `/api/v1/admin/` por
      recurso: revisión, documentos, fases, preguntas, calificaciones y reconsideraciones.
      Los handlers disponibles también exigen su permiso específico; el guard SPA
      no participa en la decisión de autorización del servidor.
- [x] Permitir al personal autorizado consultar formularios enviados y obtener
      URLs temporales de certificados, manteniendo la restricción de propietario
      en las rutas del postulante y auditando resultado, actor, recurso y fecha.
- [x] Asignación fuera del registro público: PATCH administrativo con motivo,
      comprobación del rol vigente bajo bloqueo y auditoría transaccional. La CLI
      permite el alta inicial con operador/motivo. Un trigger registra también
      cambios SQL directos; el historial rechaza UPDATE, DELETE y TRUNCATE.

**Implementación T6 completada (2026-10-31)**. Migración `00007_authorization.sql`,
SQLC regenerado, tres roles en sesión y permisos compartidos entre HTTP y servicios.
Analista y administrador consultan expedientes/certificados y tienen acceso base a
calificaciones/reconsideraciones; fases, banco de preguntas y asignación de roles
exigen administrador. El registro OTP siempre crea `postulante`, incluso si se
inyecta `rol`/`role` en el JSON. Una cookie existente usa el rol vigente de PostgreSQL.

Rutas implementadas: `GET /api/v1/admin/submissions/{id}`, su lista `/documents`,
`GET /api/v1/admin/documents/{id}/download` y
`PATCH /api/v1/admin/applicants/{id}/role`. Solo se entregan snapshots enviados,
sin alterar su contenido ni abrir borradores ajenos. Las URLs duran cinco minutos;
si falla la auditoría, no se entrega el formulario, los metadatos ni la URL.

**Alcance pendiente en sus tareas**: T9 incorpora listas y filtrado histórico por
convocatoria; T18/T20–T22/T25 implementan las operaciones de fases, calificaciones,
reconsideraciones y preguntas, usando estos permisos y controles específicos de
estado/plazo. La política implementada no acredita aprobación funcional por los
responsables: su validación y los detalles de resolución por actor siguen en dichas
tareas. T8 conserva la consulta paginada, conservación y auditoría del resto del flujo.

**Verificación**: tests Go de identidad, postulación, documentos, auditoría y los
comandos afectados correctos. Cubren matriz de roles, revocación, escalamiento,
registro OTP, rutas administrativas/propietario, snapshot y fallo de auditoría.
El test PostgreSQL de restricciones, historial inmutable y rollback del cambio de
rol es optativo con `AUTHORIZATION_TEST_DATABASE_URL`; no se ejecutó sin esa variable.

**Dependencias**: T2. **Frontend**: FT2, FT7, FT15, FT24–FT25.

### T7 · Contexto de catálogo con caché

> **Criterio 3 — Caché de catálogos (parte de los 6 pts)**

- [x] `GET /api/v1/catalogs/countries`, `/provinces`, `/cities`.
- [x] Se sirven desde Redis con TTL de 24 h; ante un fallo de Redis, se
      responde desde PostgreSQL (fail-open).
- [x] La tabla `catalogs` incluye todos los países enumerados en EF-12 1.2.1,
      las 24 provincias y su relación con ciudades/cantones. Conservar el catálogo
      nacional de 221 cantones y documentar diferencias con la lista parcial del RF;
      no reducir países a la antigua semilla de 18. Usar una migración idempotente.
- [x] **Entregable del examen**: existe un `.sql` con el DDL completo y los
      datos semilla de la división política, listo para ejecutarse sobre una
      base vacía. El DDL vive en `apps/backend/sql/migrations/`; el seed, en
      una migración idempotente (`INSERT ... ON CONFLICT DO NOTHING`) y, en
      forma plana, en `apps/backend/sql/seeds/`.

- [x] Publicar los catálogos de dominio de EF-12: parentescos, IESS/BIESS, niveles
      de estudio, áreas, tipo Nacional/Extranjera, experiencia General/Dirección,
      publicaciones y Curso/Seminario/Taller, sin opciones inventadas.

**T7 completada (2026-10-31)**: contexto `internal/catalog/` con puertos, servicio,
HTTP y adaptador SQLC. Catálogos protegidos por sesión, disponibles para los tres
roles; listas bajo `paises`/`provincias`/`cantones` compatibles con el cliente existente.
Se admiten las rutas con query y las rutas geográficas con segmentos que ese
cliente ya utiliza. La cascada exige
padres válidos y devuelve listas vacías para geografía extranjera de captura libre.
`GET /api/v1/catalogs/options/{name}` entrega los doce grupos bajo `opciones`.

`00008_catalogs.sql` y `sql/seeds/catalogs.sql` contienen los 194 países del RF,
24 provincias, 221 cantones y 47 opciones de dominio. La semilla es idempotente,
usa UUIDv7 y referencias a padres; el SQL autónomo incluye DDL y puede ejecutarse
en una base vacía de PostgreSQL 18. `make catalog-seed` regenera ambos desde la
definición EF-12 y la geografía existente. La extracción de `Bután` se corrigió
para no tratar la cabecera de página como parte de un país.

**Verificación**: tests Go de `catalog`, `platform/cache` y `submission` correctos,
incluida la compilación mediante tests de los comandos afectados. Cubren hit/miss,
expiración exacta de 24 h, fallos de lectura/escritura Redis, caché corrupta,
cancelación, fuente persistente fallida, filtros, roles y coincidencia de la semilla
con el RF. Las pruebas PostgreSQL de migración/SQL autónomo, idempotencia, UUIDv7,
orden, padres y cobertura se habilitan con `CATALOG_TEST_DATABASE_URL`; no se
ejecutaron sin esa variable. No se iniciaron servicios ni se cambió código frontend.

**Dependencias**: T1. **Frontend**: FT5. **Esfuerzo estimado**: 1,5 días.

### T8 · Completar auditoría de expediente y accesos

> **RF EF-1: trazabilidad completa; EF-13: consulta histórica.**

- [x] Tabla `submission_events` y eventos `guardada`/`enviada` disponibles desde T5.
- [x] Guardados, envío y cambios de adjuntos registran eventos en la transacción.
- [x] Restringir modificaciones de eventos y definir conservación del historial;
      registrar `consultada` y descargas administrativas con actor, fecha y resultado.
- [x] Proveer tabla y puerto transaccional cerrado para fases, elegibilidad, méritos,
      evaluaciones, reconsideraciones y publicación, sin payload libre; conservar
      el trigger de permisos existente y unir sus metadatos al historial.
- [x] Publicar historial paginado para el propio expediente y para personal autorizado.

**Base T8 implementada (2026-10-31)**: `00009_audit_history.sql` hace append-only
los eventos del expediente y evita su pérdida por cascadas. Audita bajas y
cambios de adjuntos en la misma transacción, y enlaza descargas administrativas
con el expediente. Historial propio/administrativo con cursor por fecha/UUID,
titularidad, permiso de revisión y registro obligatorio de accesos internos.
Conservación técnica sin purga automática, pendiente del plazo/procedimiento que
apruebe la institución en T28. No equivale a una política legal ya aprobada.

- [ ] Instrumentar el puerto de negocio al implementar los módulos futuros
      T17/T18/T20–T27; una lista de acciones admitidas no acredita que existan.

Contrato, límites y pruebas: [`REVIEW_AUDIT.md`](../api-contracts/REVIEW_AUDIT.md).

**Dependencias**: T5, T6. **Frontend**: FT7, FT19–FT23, FT27.

### T9 · Revisión, dashboard e históricos del concurso

> **RF EF-13. Obligatoria para el RF completo; no es una mejora opcional.**

- [x] Listar postulaciones por convocatoria, cargo/sector y estado, con cursor.
- [x] Exponer conteos frescos por cargo/sector con propuesta de polling cada 30 s;
      la actualización de la pantalla corresponde a FT7 y la latencia institucional
      queda sujeta a aprobación, no se anuncia como garantía de tiempo real.
- [x] Al consultar una convocatoria cerrada, devolver su nómina completa y los EF-12
      sellados de ese período, sin mezclarlos con convocatorias posteriores.
- [x] Incluir certificados descargables con autorización y duración de cada experiencia
      en años, meses y días, usando el dato del expediente histórico correspondiente.
- [x] Exportar matriz comparativa técnica en `.xlsx` y PDF con la misma selección
      y filtros, sin limitarla a la página visible.

**Backend T9 implementado (2026-10-31)**: contexto `review/` y rutas
`/api/v1/admin/competitions`, `/admin/submissions`, `/admin/submissions/stats` y
`/admin/submissions/export.xlsx|pdf` bajo `/api/v1`. `00010` registra períodos y
protege su asociación, sellos y certificados. Exige `convocatoriaId` o selección
explícita de legado `sinConvocatoria=true`; no asigna expedientes existentes a una
convocatoria ficticia. Sólo entrega snapshots enviados, verifica el hash y nunca
actualiza nombres o duraciones con datos posteriores. Reutiliza detalle y URLs
temporales de T6/T4. XLSX sin fórmulas y PDF con fuentes embebidas; lectura de
exportación `REPEATABLE READ READ ONLY`, con límite de filas/bytes y sin truncado.

La gestión y publicación de convocatorias reales permanece en T18. La matriz
`EF13-TECNICA-v1` es una propuesta interna, no calificación/lista oficial de
elegibles. Ver [`REVIEW_AUDIT.md`](../api-contracts/REVIEW_AUDIT.md).

**Dependencias**: T6, T8, T18. **Frontend**: FT7–FT8.

---

## 5. P2 — Deseables

| ID | Tarea | Beneficio |
|---|---|---|
| T10 | Observabilidad: métricas Prometheus y trazas OpenTelemetry | Evaluar RNF-01 en datos reales |
| T11 | Pruebas E2E del recorrido registro → formulario → envío | Detectar regresiones completas |
| T12 | Operación y recuperación de RustFS/S3 (base disponible desde T4) | Copias y restauración de los objetos |
| T13 | Tarea programada de purga de sesiones expiradas | Evita que la tabla crezca |
| T14 | Rate limiting general por IP en nginx | Complementa el bloqueo por cédula |

---

## 6. Orden de ejecución y relación con la rúbrica

1. Sobre T1–T7 disponibles, endurecer OTP: T16.
2. Configurar convocatorias y políticas (T18/T28), cerrar elegibilidad T17 y
   formulario final T19, con la base de notificaciones persistentes T15/T23.
3. Completar auditoría/revisión e implementar requisitos y méritos: T8–T9, T20.
4. Incorporar banco y examen T25–T27; consolidar oposición psicotécnica T21,
   reconsideraciones T22 y listas T24 conforme a las dependencias declaradas.

El cumplimiento se comprueba por criterios de aceptación y trazabilidad de la
sección 9. No se asignan puntos automáticamente por completar una tarea; los
criterios 4–6 requieren integración con las tareas del frontend.

---

## 7. Convenciones para las tareas nuevas

1. **Cada capacidad** sigue la estructura del contexto `submission`:
   `service.go`, `handler.go`, `repository.go`, `dto.go`, `errors.go`,
   `postgres/`.
2. **Las reglas puras se prueban sin base de datos**: si una regla necesita
   PostgreSQL, es que no es una regla pura.
3. **Identificadores en inglés, comentarios en español** con JSDoc
   (`@param`, `@returns`).
4. **Cada endpoint** actualiza [`../api-contracts/API_CONTRACT.md`](../api-contracts/API_CONTRACT.md)
   y su tabla de errores.
5. **Antes de abrir el PR**: `gofmt -l .`, `go build ./...`, `go vet ./...`,
   `go test ./... -race` en verde.

---

## 8. Tareas añadidas para cubrir el RF completo

Todas las casillas de esta sección están pendientes. Los nombres de capacidades
son propuestas de organización; no indican que ya existan endpoints o tablas.

### T15 · Entrega fiable del correo (base compartida con T23)

**Prioridad:** P1-RF. Implementación técnica disponible; no equivale a completar
notificaciones de fases/resultados de T23.

- [x] Cola persistente cifrada, alta dentro del commit del sello, leases y reintentos
      acotados; retirar cargas vencidas y conservar metadatos de seguimiento.
- [x] Deduplicación por operación/recurso o desafío, incluso si cambia el vencimiento;
      `Message-ID` estable en los reintentos y prevención de reenvío tras un QUIT roto.
- [x] Distinguir pendiente, aceptación SMTP, fallo, expiración y desafío reemplazado.
      La aceptación SMTP no certifica llegada al buzón ni garantiza exactamente una entrega.

### T16 · Consolidar registro y recuperación con OTP

**Origen:** EF-11.4–6 adaptado por ADR-008. **Prioridad:** P0-RF.

- [x] Vincular desafío a cédula, correo y finalidad y consumirlo atómicamente una vez.
- [x] Invalidar el anterior al reenviar; configurar cooldown y cuota de emisiones
      mediante `OTP_RESEND_DELAY` y `OTP_MAX_SENDS`, manteniendo TTL de diez minutos.
- [x] Tres fallos y diez minutos de bloqueo desde su activación; no renovar la ventana
      por fallos concurrentes. Informar TTL efectivo con `Retry-After` / `X-Retry-At`.
- [x] Recuperar solo al correo registrado, conservar respuesta no enumerativa también
      para desafíos ausentes y revocar sesiones al confirmar la nueva contraseña.
- [x] Registrar `otp_correo`, sin certificar identidad civil ni exigir acceso a Registro Civil.

**Dependencias:** T1–T2. **Frontend:** FT3, FT16.

### T17 · Elegibilidad por sector y cierre de postulación

**Origen:** EF-12 1.1.6 y 1.4.1–1.4.3. **Prioridad:** P0-RF.

- [x] Puerto de situación laboral y adaptador de simulación controlada en servidor,
      mediante archivo operacional por UUID/sector; sin aceptación por defecto ni bandera del cliente.
- [x] Evidencia persistida por cuenta, convocatoria y sector, con resultado, fecha y origen.
      La ausencia de dato o un fallo técnico devuelve 503 y no produce inelegibilidad.
- [x] Confirmar cierre por nacionalidad extranjera o condición no elegible; impedir
      guardar, adjuntar, enviar, eliminar o evadirlo creando otro borrador del mismo período.
- [x] Estado `closed` separado del envío y de las fases, con evidencia e historial
      inmutables. Esta versión no reabre cierres automáticamente.
- [x] Conservar nombre y PDF real obligatorios al completar/enviar si se declara
      respaldo de organización/federación; OTP no acredita elegibilidad.
- [ ] Aprobar la política institucional de eventual revisión administrativa e integrar
      proveedor real IESS cuando exista autorización/acceso. La simulación no lo sustituye.

**Dependencias:** T3–T4, T18. **Frontend:** FT5, FT17.

### T18 · Convocatorias, fases y calendario de apertura/cierre

**Origen:** EF-1, EF-11.1 y EF-13.1–2. **Prioridad:** P0-RF.

- [x] Persistir convocatoria, sectores, fases, aprobación, publicación y habilitación
      del EF-12; crear deshabilitada y conservar cada versión de configuración.
- [x] Exigir convocatoria explícita para nuevas postulaciones; preservar el legado
      solo para lectura, sin reasignaciones ni habilitación automática.
- [x] Portal público solo de períodos aprobados/publicados y fases publicadas;
      administración exclusiva de administrador, con versión optimista, auditoría y rol vigente bloqueado.
- [x] Aplicar ventana, aprobación, fase y sector en PostgreSQL para altas, guardados,
      bajas, adjuntos y envíos, incluso cuando la pestaña quedó abierta.
- [x] Implementar zona `America/Guayaquil`, instantes con offset, precisión de
      microsegundos y límites `[apertura,cierre)`; conservar históricos de períodos cerrados.
- [x] Aceptación del frontend contra el contrato T18 completada (FT14–FT15 sin respuestas ficticias);
      validación de límites con el responsable institucional al fijar fechas oficiales.

**Dependencias:** T6, T8. **Frontend:** FT14–FT15, FT25.
**Contrato compartido:** [COMPETITION_LIFECYCLE.md](../api-contracts/COMPETITION_LIFECYCLE.md).

### T19 · Formulario final legible, impresión y correo completo

**Origen:** EF-12 1.10.1–1.10.3 y notas finales. **Prioridad:** P1-RF.

- [x] Producir una representación íntegra y legible del snapshot sellado para
      impresión/descarga, con código único, nueve secciones, cinco literales,
      nombres, apellidos, cédula y apartado de firma manuscrita.
- [x] Reutilizar ese snapshot para el formulario remitido al correo registrado.
      Hoy SMTP ya incluye el JSON completo; completar su presentación imprimible
      (por ejemplo PDF), sin regenerarlo desde un borrador posterior.
- [x] Mostrar los literales al guardar, no solo al enviar; guardar y enviar siguen
      siendo operaciones distintas. No exigir una firma digital dibujada.
- [x] Mantener hash y fecha reproducibles y autorización de acceso a la copia final.
      El fallo del correo no revierte el sello: su entrega se gestiona mediante T23.

**Estado técnico:** implementado localmente; PDF autenticado y adjunto desde cola cifrada.
Ver [contrato T19/T20](../api-contracts/MERITS_AND_PRINTING.md). Integración visual y despliegue no incluidos.

**Dependencias:** T3–T5, T23. **Frontend:** FT6.

### T20 · Revisión de requisitos, inhabilidades y calificación de méritos

**Origen:** EF-1, verificación de requisitos y calificación automática. **Prioridad:** P1-RF.

- [x] Registrar revisión de requisitos y respaldos por el personal autorizado,
      observaciones, resultado y motivo de avance o exclusión en el expediente.
- [x] Aplicar reglas de inhabilidades conforme a la definición aprobada; no usar
      «cualquier Sí excluye», porque la pregunta 1.9.1 tiene sentido positivo.
- [x] Parametrizar y versionar criterios, ponderaciones, máximos, umbrales y
      reglas de desempate de méritos cuando sean entregados por el responsable.
- [x] Calcular puntaje desde evidencias verificadas, conservar desglose reproducible
      y emitir el resultado de la fase que habilitará o no la evaluación técnica.
- [x] Auditar cambios y recalcular solo mediante una acción autorizada; no alterar
      silenciosamente una resolución o un resultado previamente publicado.

**Dependencias:** T6, T8–T9, T18. **Frontend:** FT19.
**Estado técnico:** motor y revisión implementados, probados con datos sintéticos.
No existe una matriz oficial precargada; la activación institucional sigue pendiente.

**Definición pendiente:** el RF no contiene la matriz de puntuación ni los umbrales;
no fijar valores inventados en código.

### T21 · Calificaciones de oposición psicotécnica y expediente consolidado

**Origen:** EF-1, oposición teórica y psicotécnica. **Prioridad:** P1-RF.

- [ ] Registrar resultados psicotécnicos según el instrumento/formato aprobado,
      con responsable, fecha y evidencia; el RF no detalla su instrumento de aplicación.
- [ ] Incorporar la nota teórica calculada por T27, sin sustituirla por una nota
      ingresada libremente desde la interfaz.
- [ ] Consolidar méritos y oposición conforme a reglas aprobadas, con historial
      y correcciones autorizadas; evitar contar dos veces el mismo resultado.
- [ ] Exponer el expediente propio y el administrativo con sus permisos específicos.

**Dependencias:** T18, T20, T27. **Frontend:** FT20, FT22.

### T22 · Reconsideraciones y control de plazos por actor

**Origen:** EF-1, gestión de reconsideraciones. **Prioridad:** P1-RF.

- [ ] Registrar solicitud, motivos y respaldos, vincularlos al resultado de una fase
      y controlar su presentación por el postulante dentro de la ventana aplicable.
- [ ] Modelar recepción, atención de Talento Humano y resolución de la Comisión
      con responsables, estados, fecha límite y trazabilidad de cada actuación.
- [ ] Parametrizar el cómputo de términos/plazos por actor y calendario aprobado;
      rechazar acciones fuera de plazo y evitar duplicados ante reintentos.
- [ ] Conservar resultado original, resolución y efecto sobre puntajes/avance,
      notificando cada decisión y habilitando su consulta en el expediente.

**Dependencias:** T6, T8, T18, T20–T21, T23. **Frontend:** FT21.
**Definición pendiente:** duración, días hábiles/calendario y reglas aplicables no
están detallados en este RF; requieren parámetros aprobados.

### T23 · Notificaciones automáticas por etapa y entrega persistente

**Origen:** EF-1, EF-12 1.10.3 y EF-14 1.4. **Prioridad:** P1-RF; absorbe T15.

- [ ] Encolar la notificación dentro de la transacción que confirma el evento;
      enviar por SMTP después del commit con reintentos persistentes e idempotencia.
- [ ] Cubrir envío de postulación, resultados de requisitos/méritos/oposición,
      reconsideraciones y publicación de resultados según el flujo aprobado.
- [ ] Usar destinatario registrado y plantillas versionadas; adjuntar o enlazar
      de forma autorizada el formulario/reporte cuando corresponda (T19/T27).
- [ ] Registrar entrega/fallo sin OTP ni datos sensibles en logs y permitir reintento
      autorizado. Un fallo de entrega no cambia una nota ni duplica una postulación.

**Dependencias:** T5, T8, correo existente. **Frontend:** FT6, FT22, FT27.

### T24 · Listas de elegibles y publicación de resultados

**Origen:** EF-1, reportes/listas de elegibles; EF-12 1.10.1.4. **Prioridad:** P1-RF.

- [ ] Generar listas por convocatoria/cargo desde resultados aprobados y
      reconsideraciones resueltas, con versión, fecha y orden definido por las reglas.
- [ ] Gestionar publicación por el rol competente y conservar versiones históricas.
- [ ] Distinguir reportes internos de listas públicas: no publicar documentos,
      contactos o datos personales no autorizados; respetar el consentimiento del RF.
- [ ] Vincular reportes a los mismos resultados del expediente y de la matriz T9.

**Dependencias:** T6, T9, T18, T20–T23, T28. **Frontend:** FT8, FT23.

### T25 · Banco de preguntas y matriz de respuestas correctas

**Origen:** EF-14 1.1. **Prioridad:** P1-RF.

- [ ] Permitir al administrador cargar y gestionar preguntas de opción múltiple,
      opciones, respuestas correctas y versiones del banco.
- [ ] Validar estructura, opciones válidas y disponibilidad de preguntas suficientes
      para la cantidad parametrizada; definir el formato de carga con el responsable.
- [ ] Mantener respuestas correctas fuera de las respuestas HTTP del examen activo.
- [ ] Congelar la versión de preguntas/respuestas empleada en cada evaluación,
      aunque después se edite el banco, y auditar la gestión administrativa.

**Dependencias:** T6, T8, T18. **Frontend:** FT24.

### T26 · Habilitación y aplicación de la evaluación teórica

**Origen:** EF-14, apertura y puntos 1.2–1.3.4. **Prioridad:** P1-RF.

- [ ] Admitir solo a quienes superaron méritos y están habilitados por el administrador
      en fecha/hora programadas; reflejar la fase «Evaluación Técnica» en el expediente.
      El público general no puede consultar el examen.
- [ ] Parametrizar cantidad de preguntas y duración. Al iniciar, seleccionar al azar
      preguntas y orden, persistir la asignación y el vencimiento antes de responder.
- [ ] Validar que banco y parámetros permiten la unicidad de exámenes solicitada;
      definir cómo resolver agotamiento de combinaciones sin prometer unicidad imposible.
- [ ] Al reintentar inicio o recuperar sesión, mantener el mismo intento, orden,
      respuestas y tiempo restante, según la política de intentos aprobada.
- [ ] Guardar respuestas por pregunta y exigir que pertenezcan al intento vigente;
      no confiar en tiempo, elegibilidad, puntaje o IDs arbitrarios del navegador.
- [ ] Cerrar al vencer el reloj del servidor y conservar respuestas hasta el plazo,
      incluso si se pierde la conexión o se cierra la pantalla; rechazar respuestas tardías.

**Dependencias:** T18, T20, T25. **Frontend:** FT25–FT26.
**Definición pendiente:** número de preguntas, duración y política de intentos.

### T27 · Finalización, calificación, PDF y correo de la evaluación

**Origen:** EF-14 1.3.3–1.4. **Prioridad:** P1-RF.

- [ ] Unificar «Finalizar y Calificar» y vencimiento automático en una operación
      idempotente; una carrera entre ambos produce una única nota definitiva.
- [ ] Comparar respuestas con la matriz congelada y calcular la nota conforme a
      la escala/ponderación aprobadas, sin aceptar la calificación del cliente.
- [ ] Confirmar cierre, respuestas finales, nota, expediente, auditoría y evento de
      notificación en una transacción; conservar el desglose de cálculo.
- [ ] Generar reporte PDF, almacenarlo para administración y habilitar su consulta
      por el postulante; notificar automáticamente al correo registrado mediante T23.

**Dependencias:** T5, T8, T23, T25–T26. **Frontend:** FT20, FT27.
**Definición pendiente:** escala y regla de cálculo de la nota no detalladas en el RF.

### T28 · Políticas de privacidad y tratamiento de datos

**Origen:** EF-1, precondiciones y restricciones; EF-12 1.10. **Prioridad:** P1-RF.

- [ ] Publicar la política de privacidad y la información de tratamiento aprobadas
      por la institución, con versión/fecha, sin inventar textos jurídicos o plazos.
- [ ] Vincular la versión aplicable al expediente y conservar los cinco consentimientos
      literales ya implementados; no sustituirlos por una aceptación genérica.
- [ ] Definir con el responsable acceso, conservación y eliminación de datos,
      documentos y reportes, preservando los registros que deban mantenerse.
- [ ] Aplicar esas reglas a exportación, correo, URLs temporales y listas públicas.

**Dependencias:** T3–T4, T6, T8. **Frontend:** FT6, FT28.

---

## 9. Matriz de cobertura y definiciones pendientes

**Cobertura de planificación, no certificación de implementación.** La matriz
incluye lo explícito de EF-11–EF-14 y los procesos que EF-1 describe sin detallar.

| Fuente del RF | Requisito o adaptación | Backend | Frontend | Situación |
|---|---|---|---|---|
| EF-1, actores | Postulante, Analista TH y Administrador/Comisión | T6 | FT2, FT15 | Tres roles y permisos backend disponibles; integración administrativa de interfaz pendiente |
| EF-1, privacidad | Política, tratamiento y consentimiento informado | T28, T3 | FT28, FT6 | Consentimientos base; política e integración pendientes |
| EF-11.1–3 | Fase aprobada, portal institucional, nueva pestaña y acceso | T18 | FT14–FT15 | Planificado |
| EF-11.4–6, ADR-008 | Registro, login y recuperación con OTP por correo | T1–T2, T16 | FT3, FT16 | Base OTP y vistas disponibles; consolidación T16/FT16 pendiente |
| EF-12, primer ingreso | Entrada obligatoria al formulario | T2, T18 | FT2–FT3 | Guard base; cerrar flujo funcional |
| EF-12 1.1 | Datos personales/cónyuge y nacionalidad; precarga de cuenta OTP | T3, T17 | FT5, FT16–FT17 | Tipado base; autocompletado civil sustituido por datos declarados |
| EF-12 1.2 | Países completos, cascada, correo registrado y dirección | T3, T7 | FT5 | Catálogos backend completos; integración de vistas pendiente |
| EF-12 1.3 | Familiares y opciones exactas de parentesco/institución | T3, T7 | FT5 | Opciones backend disponibles; integración visual pendiente |
| EF-12 1.4 | Cargo, elegibilidad, mensajes, cierre y organización | T3–T4, T17–T18 | FT5, FT17–FT18 | Sustituir elegibilidad declarada por fuente del servidor |
| EF-12 1.5–1.8 | Títulos, experiencia/calculadora, publicaciones y capacitación | T3–T4, T7 | FT5, FT18 | Backend base; vistas y adjuntos integrados pendientes |
| EF-12 1.9 | Trece respuestas, detalles de remoción y revisión | T3, T20 | FT5, FT19 | Revisión administrativa backend disponible; interfaz y matriz institucional pendientes |
| EF-1, restricciones | PDF 2/4 MiB y total 50 MiB | T4 | FT18 | Backend disponible; consumo frontend pendiente |
| EF-12, notas | Guardado parcial, CRUD de registros y código único | T3–T5 | FT5 | Backend base; interfaz funcional pendiente |
| EF-12 1.10 | Literales al guardar, firma manuscrita, impresión y correo completo | T3, T19, T23 | FT6, FT28 | PDF sellado y adjunto en cola disponibles; integración visual pendiente |
| EF-13.1–2 | Publicar fases y habilitar EF-12 | T6, T18 | FT15 | Planificado |
| EF-13.3 | Conteos por cargo en tiempo real e históricos íntegros/certificados | T8–T9, T18 | FT7 | Parcial: shell/mock; falta administración real |
| EF-13.3 | Matriz comparativa Excel/PDF | T9 | FT8 | Planificado, obligatorio para RF completo |
| EF-1 | Verificar requisitos y calcular méritos | T20 | FT19 | Motor configurable y revisión backend disponibles; falta matriz institucional aprobada |
| EF-1 | Registrar oposición psicotécnica y consolidar expediente | T21 | FT20 | Planificado; falta instrumento/reglas |
| EF-1 | Reconsideraciones y plazos exactos por actor | T22 | FT21 | Planificado; faltan términos y calendario aprobados |
| EF-1 | Notificar resultados de cada etapa | T23 | FT22 | Correo base; flujo y reintentos persistentes pendientes |
| EF-1 | Reportes, listas de elegibles y trazabilidad completa | T8, T24 | FT7–FT8, FT23 | Planificado sobre auditoría base de T5 |
| EF-14, apertura/1.1 | Módulo restringido y banco de preguntas | T6, T25 | FT24–FT25 | Planificado |
| EF-14 1.2 | Méritos aprobados, fecha/hora, azar, cantidad y orden distinto | T18, T20, T26 | FT25–FT26 | Planificado; parámetros pendientes |
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

**Ver también**: [frontend-tasks.md](frontend-tasks.md) ·
[`../architecture/ARQUITECTURA.md`](../architecture/ARQUITECTURA.md) ·
[`../api-contracts/API_CONTRACT.md`](../api-contracts/API_CONTRACT.md)
