# Tareas del backend

**Sistema de Registro y Postulación al Directorio del BIESS**
Lista de trabajo del backend, priorizada y mapeada a la rúbrica de 40 puntos

| Campo | Valor |
|---|---|
| Documento | backend-tasks v1.0 |
| Ámbito | `apps/backend/` |
| Rúbrica | [enunciado del examen práctico](../prd/PRD.md) |

---

## 1. Cómo leer esta lista

Cada tarea tiene:

- **ID** — identificador estable (`BE-xx`) para seguirla en el PR y en el
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

## 2. Estado por criterio de la rúbrica

| # | Criterio | Puntos | Estado | Tareas |
|---|---|---:|---|---|
| 1 | Arquitectura backend e inyección de dependencias | 8 | ✅ **8/8** | T6, T7 |
| 2 | Persistencia relacional y transacciones | 8 | 🟡 **7/8** | T5, T8 |
| 3 | Caché, expiración y rate limiting (Redis) | 6 | ✅ **6/6** | T1 |
| 4 | Frontend SPA (ver [frontend-tasks.md](frontend-tasks.md)) | 6 | — | — |
| 5 | Consumo de API y manejo asíncrono (ver [frontend-tasks.md](frontend-tasks.md)) | 6 | — | — |
| 6 | Cumplimiento de las reglas del negocio (RF IESS) | 6 | ✅ **6/6** | T2, T3, T4 |
| | **Total backend** | **28** | **27/28** | |

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

**Comportamiento exigido por el EF-11**

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

Implementa el EF-11 completo.

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

- [x] `POST /api/v1/auth/register` valida la fecha de expedición o el código
      dactilar y devuelve `201`.
- [x] `POST /api/v1/auth/login` devuelve la cookie `session` con `HttpOnly` y
      `SameSite=Lax`; en producción añade `Secure`.
- [x] El token **no** se persiste en claro: en `sessions` solo su SHA-256.
- [x] El login responde `429` con `intentos_excedidos` cuando hay bloqueo.
- [x] `GET /api/v1/auth/session` responde `200` con el perfil o `401`.
- [x] `POST /api/v1/auth/password/forgot` envía el correo con el enlace
      (vía `platform/mail`).
- [x] Existe un middleware de autenticación (`RequireSession`) reutilizable
      por los demás contextos.
- [x] Tests: módulo 10, fortaleza de contraseña, y el flujo de login con
      bloqueo tras 3 intentos.

**Dependencias**: T1 (para el bloqueo). **Esfuerzo estimado**: 3 días.

---

### T3 · Contexto de postulación: las 9 secciones del EF-12

> **Criterio 6 — Reglas del negocio (6 pts)**

**Backend implementado (2026-10-31)**: modelo tipado, validación, progreso, guardado parcial, sello y confirmación SMTP. La verificación externa IESS/Registro Civil y los documentos reales aún dependen de sus integraciones.

**Piezas**

| Pieza | Ubicación | Contenido |
|---|---|---|
| Secciones | `internal/submission/sections.go` | Tipos de las 9 secciones del EF-12 |
| Reglas | `internal/submission/rules.go` | Validaciones del RF |
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
- [ ] Paridad completa con la validación de las vistas del frontend. El backend ya aplica las reglas; duración y fechas imposibles cuentan con casos compartidos Go/Vitest. Las vistas EF-12 y su validador completo siguen pendientes de migración.
- [x] Los 5 puntos de consentimiento se guardan y son verificables.
- [x] La postulación enviada es inmutable: `PUT` y `DELETE` devuelven `409`.
- [x] Tests de cada regla con los casos de borde del RF.

**Verificación**: pruebas de reglas, guardado concurrente, doble envío, rollback, consentimiento, destinatario registrado y estados HTTP con dobles en memoria. Las pruebas PostgreSQL optativas se ejecutan con `SUBMISSION_TEST_DATABASE_URL` y crean un esquema aislado por ejecución. En este entorno se omiten por falta de URL; SMTP real y `-race` requieren sockets habilitados y CGO, respectivamente.

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

### T6 · Middleware de autenticación y de roles

> **Criterio 5 — Rutas protegidas (6 pts)**

- [ ] `RequireSession` resuelve la cookie y adjunta el postulante al
      contexto.
- [ ] `RequireRole` para los endpoints del dashboard (Analista TH,
      Administrador).
- [ ] El servicio de revisión solo es accesible con rol.

**Dependencias**: T2. **Esfuerzo estimado**: 1 día.

### T7 · Contexto de catálogo con caché

> **Criterio 3 — Caché de catálogos (parte de los 6 pts)**

- [ ] `GET /api/v1/catalogs/countries`, `/provinces`, `/cities`.
- [ ] Se sirven desde Redis con TTL de 24 h; ante un fallo de Redis, se
      responde desde PostgreSQL (fail-open).
- [ ] La tabla `catalogs` se puebla con la semilla del RF (24 provincias,
      221 cantones, 18 países) en una migración idempotente.
- [ ] **Entregable del examen**: existe un `.sql` con el DDL completo y los
      datos semilla de la división política, listo para ejecutarse sobre una
      base vacía. El DDL vive en `apps/backend/sql/migrations/`; el seed, en
      una migración idempotente (`INSERT ... ON CONFLICT DO NOTHING`) y, en
      forma plana, en `apps/backend/sql/seeds/`.

**Dependencias**: T1. **Esfuerzo estimado**: 1,5 días.

### T8 · Auditoría: `submission_events`

> **Criterio 2 — Modelo normalizado (parte de los 8 pts)**

- [ ] Tabla `submission_events` append-only: `guardada`, `enviada`,
      `consultada`.
- [ ] Se escribe en la misma transacción que el cambio que la origina.
- [ ] `GET /api/v1/submissions/{id}/events` para el historial (EF-13).

**Dependencias**: T5. **Esfuerzo estimado**: 1 día.

### T9 · Contexto de revisión y dashboard

> **Criterio 2 y 6 (parcial EF-13)**

- [ ] Listado de postulaciones con filtros (estado, sector) y paginación por
      **cursor**.
- [ ] Conteos por cargo y por estado, cacheados en Redis.
- [ ] Exportación de la matriz comparativa a `.xlsx` y PDF.

**Esfuerzo estimado**: 3 días.

---

## 5. P2 — Deseables

| ID | Tarea | Beneficio |
|---|---|---|
| T10 | Observabilidad: métricas Prometheus y trazas OpenTelemetry | Evaluar RNF-01 en datos reales |
| T11 | Pruebas E2E del recorrido registro → formulario → envío | Detectar regresiones completas |
| T12 | Migración de los documentos a S3/MinIO | Resiliencia del almacenamiento |
| T13 | Tarea programada de purga de sesiones expiradas | Evita que la tabla crezca |
| T14 | Rate limiting general por IP en nginx | Complementa el bloqueo por cédula |
| T15 | Reintento exponencial en el cliente de correo | Entregabilidad de las notificaciones |

---

## 6. Cobertura de la rúbrica: plan de puntos

| Criterio | Hoy | Con T1–T4 | Con todo |
|---|---:|---:|---:|
| 1. Arquitectura e ID | 8 | 8 | 8 |
| 2. Persistencia y transacciones | 5 | 7 | 8 |
| 3. Redis | 0 | 6 | 6 |
| 6. Reglas del negocio | 2 | 5 | 6 |
| **Subtotal backend** | **15/28** | **26/28** | **28/28** |

Con T1–T4, el backend pasa de **15 a 26 de 28 puntos posibles**. Es la
prioridad absoluta de la hoja de ruta.

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

**Ver también**: [frontend-tasks.md](frontend-tasks.md) ·
[`../architecture/ARQUITECTURA.md`](../architecture/ARQUITECTURA.md) ·
[`../api-contracts/API_CONTRACT.md`](../api-contracts/API_CONTRACT.md)
