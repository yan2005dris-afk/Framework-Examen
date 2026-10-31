# Tareas transversales

**Sistema de Registro y Postulación al Directorio del BIESS**  
Plan transversal para seguridad, contratos, operación e integración del RF v1.4

| Campo | Valor |
|---|---|
| Documento | cross-cutting-tasks v1.0 |
| Ámbito | `apps/frontend/`, `apps/backend/`, `.github/`, `compose.yaml` y `deploy/` |
| Relación | Complementa `frontend-tasks.md` y `backend-tasks.md` |
| Estado | Todas las tareas permanecen pendientes hasta cumplir sus criterios |
| Prioridad | `P0-X` bloquea seguridad o consistencia; `P1-X` es importante para operación; `P2-X` es deseable |

---

## 1. Cómo leer esta lista

Estas tareas no sustituyen FT1–FT28 ni T1–T28. Cubren preocupaciones que
atraviesan ambos planes y que no deben atribuirse únicamente a frontend o
backend.

- **ID** — identificador estable `XT-01` a `XT-09`.
- **Criterio** — área transversal que se valida en varios módulos.
- **Prioridad** — `P0-X`, `P1-X` o `P2-X`.
- **Estado** — `✅` hecho · `🟡` parcial · `⬜` pendiente.

Una tarea transversal solo se cierra cuando existe implementación, contrato
documentado, pruebas y evidencia de integración en los módulos afectados.

## 2. Cobertura transversal

| Área | Tareas relacionadas | Riesgo si falta |
|---|---|---|
| Seguridad de sesión | XT-01, XT-02 | Secuestro de sesión, abuso de OTP o bypass de controles |
| Contratos API | XT-03 | Frontend y backend interpretan estados o errores de forma distinta |
| Estados de negocio | XT-04 | Carreras, duplicados y operaciones fuera de fase |
| Notificaciones | XT-05 | El sistema confirma operaciones que no fueron notificadas |
| Integración continua | XT-06 | Fallos solo visibles con servicios externos |
| Privacidad | XT-07 | Exposición de datos personales en logs, exportaciones o URLs |
| Operación | XT-08, XT-09 | Incidentes sin diagnóstico, recuperación o restauración verificable |

## 3. P0-X — Seguridad y consistencia

### XT-01 · Seguridad de sesión, CSRF y cookies

**Origen:** EF-1, RNF de seguridad, `T2`, `T6`, `FT2` y `FT3`.  
**Prioridad:** P0-X.

Definir y aplicar una política única para las sesiones basadas en cookies. La
autenticación no debe depender únicamente de que el navegador oculte la cookie.

**Backend**

- [ ] Mantener las cookies con `HttpOnly`, `SameSite` y `Secure` según el entorno.
- [ ] Definir protección CSRF para `POST`, `PUT`, `PATCH` y `DELETE`.
- [ ] Validar `Origin` o el mecanismo CSRF elegido en todas las rutas mutantes.
- [ ] No aceptar `Origin` o `Referer` arbitrarios como prueba de confianza.
- [ ] Rotar la sesión después de login y de recuperación de contraseña.
- [ ] Revocar sesiones anteriores en los eventos definidos por la política de seguridad.
- [ ] Aplicar expiración absoluta y, si corresponde, expiración por inactividad.
- [ ] No aceptar tokens de sesión en query string, cuerpo o `localStorage`.

**Frontend**

- [ ] Enviar el mecanismo CSRF requerido por el backend en todas las mutaciones.
- [ ] No persistir tokens, códigos OTP ni datos personales sensibles en
      `localStorage` o `sessionStorage`.
- [ ] Tratar `401` y `403` de forma diferenciada.
- [ ] Evitar reintentos automáticos de mutaciones no idempotentes.

**Criterios de aceptación**

- [ ] Una mutación sin protección CSRF recibe un error estable y no modifica datos.
- [ ] Un token de sesión anterior al login o reset deja de ser válido según la política.
- [ ] Las cookies tienen atributos correctos en desarrollo y producción.
- [ ] Existen pruebas HTTP para origen válido, origen inválido, sesión expirada y
      rotación de sesión.

**Dependencias:** `T2`, `T6`, `FT1`–`FT3`.  
**Esfuerzo estimado:** 2 días.

### XT-02 · Consolidación segura de OTP y rate limiting

**Origen:** `T1`, `T2`, `T16`, `FT3` y `FT16`.  
**Prioridad:** P0-X.

Separar el OTP de registro y recuperación como desafíos distintos y garantizar
que los controles de seguridad permanezcan activos cuando Redis no esté disponible.

**Criterios técnicos**

- [ ] Vincular cada desafío a cédula, correo, finalidad y un identificador de desafío.
- [ ] Invalidar el desafío anterior al reenviar.
- [ ] Consumir el OTP de forma atómica y de uso único.
- [ ] Aplicar TTL, límite de intentos y límite de reenvíos desde el servidor.
- [ ] No permitir utilizar un OTP de registro para recuperación, ni al contrario.
- [ ] No revelar la existencia de una cuenta durante recuperación.
- [ ] Separar el caché opcional de catálogos del almacenamiento obligatorio de seguridad.
- [ ] Si Redis es indispensable y no está disponible, rechazar claramente los flujos
      que dependen de él; no continuar con controles desactivados.
- [ ] Limitar por cédula, correo, desafío e IP según la política aprobada.

**Criterios de aceptación**

- [ ] Dos verificaciones concurrentes no pueden consumir el mismo OTP.
- [ ] Un reenvío invalida el desafío anterior.
- [ ] El vencimiento y los intentos restantes provienen de la API.
- [ ] Los límites sobreviven al reinicio y a múltiples instancias.
- [ ] Existen pruebas de concurrencia, expiración, reenvío, bloqueo y Redis no disponible.

**Dependencias:** `T1`, `T2`, `T16`, `FT3`, `FT16`.  
**Esfuerzo estimado:** 2 días.

### XT-03 · Contratos y compatibilidad frontend/backend

**Origen:** `T3`, `T4`, `T6`–`T9`, `FT1`, `FT4`–`FT8` y `FT16`–`FT27`.  
**Prioridad:** P0-X.

Establecer una única fuente verificable para DTO, estados, errores, paginación,
consentimientos, documentos y notificaciones.

- [ ] Mantener `API_CONTRACT.md` actualizado por cada endpoint.
- [ ] Documentar request, response, códigos HTTP, códigos de error y campos opcionales.
- [ ] Definir los estados de postulación, documento, fase, notificación y evaluación.
- [ ] Definir el envelope de errores y su comportamiento para `401`, `403`, `409`,
      `422`, `429` y `5xx`.
- [ ] Documentar paginación por offset y cursor, incluyendo el formato opaco del cursor.
- [ ] Versionar cambios incompatibles del contrato.
- [ ] Alinear nombres de campos EF-12 entre `definition.json`, Go y TypeScript.
- [ ] Probar que los cinco consentimientos, referencias de documentos y hashes
      mantienen la misma forma en ambos clientes.
- [ ] Añadir pruebas de contrato automatizadas para endpoints críticos.

**Criterios de aceptación**

- [ ] El frontend puede generar sus tipos o validaciones desde un contrato mantenido.
- [ ] Una respuesta de error del backend se presenta correctamente sin parsear texto libre.
- [ ] Los contratos cubren registro, OTP, sesión, postulación, documentos, revisión,
      exportaciones y notificaciones.
- [ ] Una modificación incompatible falla en CI antes de integrarse.

**Dependencias:** todas las tareas que expongan endpoints.  
**Esfuerzo estimado:** 2 días iniciales y mantenimiento continuo.

### XT-04 · Estados, transiciones e idempotencia de procesos

**Origen:** `T3`, `T5`, `T9`, `T18`, `T20`–`T27`, `FT5`–`FT8` y `FT14`–`FT27`.  
**Prioridad:** P0-X.

Definir las máquinas de estado y las operaciones idempotentes para evitar que
reintentos, pestañas abiertas o carreras cambien ilegalmente el expediente.

- [ ] Separar estado de postulación, convocatoria, fase, elegibilidad, evaluación,
      reconsideración, publicación y notificación.
- [ ] Documentar las transiciones válidas y el actor autorizado para cada una.
- [ ] Rechazar guardados, adjuntos, envíos y evaluaciones fuera de plazo en el servidor.
- [ ] Hacer idempotentes el sellado, finalización de examen, reenvío y generación
      de notificaciones.
- [ ] Usar bloqueo o control de versión para carreras entre cierre y envío.
- [ ] Evitar doble creación por reintentos de red o doble clic.
- [ ] Conservar el estado anterior, nuevo estado, actor, fecha y motivo en auditoría.

**Criterios de aceptación**

- [ ] Cada transición inválida devuelve un error estable y no altera datos.
- [ ] Dos solicitudes concurrentes producen un solo resultado definitivo.
- [ ] Una pestaña antigua no puede modificar una convocatoria o fase cerrada.
- [ ] Las pruebas cubren reintentos, carreras y recuperación tras desconexión.

**Dependencias:** `T5`, `T6`, `T8`, `T18`, `T20`–`T27`.  
**Esfuerzo estimado:** 3 días iniciales y aplicación por módulo.

## 4. P1-X — Operación, privacidad e integración

### XT-05 · Notificaciones persistentes y reintentos

**Origen:** `T3`, `T15`, `T19`, `T23`, `T27`, `FT6`, `FT22` y `FT27`.  
**Prioridad:** P1-X.

Separar el resultado de negocio de la entrega del correo. Sellar una postulación
no debe deshacerse por un fallo SMTP, pero tampoco debe presentarse como notificada.

- [ ] Crear un registro persistente de notificación con evento, destinatario,
      plantilla, versión, estado y número de intentos.
- [ ] Encolar la notificación dentro de la transacción que confirma el evento.
- [ ] Enviar después del commit con reintentos y backoff.
- [ ] Garantizar idempotencia para que un reintento no duplique el correo lógico.
- [ ] Persistir estados `pending`, `sending`, `sent`, `failed` y `retrying` o
      equivalentes documentados.
- [ ] Registrar errores sanitizados sin OTP ni datos sensibles.
- [ ] Permitir reintento autorizado de notificaciones fallidas.
- [ ] Entregar al frontend el estado real de notificación.

**Criterios de aceptación**

- [ ] Un fallo SMTP no revierte el sellado ni se presenta como envío exitoso.
- [ ] Un reinicio no pierde notificaciones pendientes.
- [ ] Una notificación se entrega como máximo una vez por evento lógico, según
      la garantía definida por el adaptador SMTP.
- [ ] Existen pruebas de rollback, reintento, duplicado y fallo permanente.

**Dependencias:** `T5`, `T8`, `T19`, `T23`, `FT6`, `FT22`.  
**Esfuerzo estimado:** 3 días.

### XT-06 · Integración, CI y pruebas con servicios reales

**Origen:** criterios de verificación de `T1`–`T9`, `FT1`–`FT6` y convenciones de
desarrollo.  
**Prioridad:** P1-X.

Convertir las pruebas que hoy dependen de variables externas en una matriz
reproducible de CI y desarrollo local.

- [ ] Levantar PostgreSQL, Redis, SMTP de prueba y almacenamiento S3/RustFS en CI.
- [ ] Ejecutar migraciones sobre una base vacía y sobre una base con datos previos.
- [ ] Ejecutar pruebas de idempotencia de semillas y rollback de transacciones.
- [ ] Ejecutar pruebas de contrato HTTP contra el backend real.
- [ ] Ejecutar pruebas de carga/concurrencia para OTP, cuota, envío y examen.
- [ ] Mantener secretos y credenciales de CI fuera del repositorio.
- [ ] Separar pruebas unitarias, integración y E2E con comandos identificables.
- [ ] Publicar artefactos de cobertura, logs sanitizados y diagnósticos de fallos.
- [ ] Hacer obligatorios `gofmt`, `go vet`, `go test`, lint, build y formato frontend
      en los cambios correspondientes.

**Criterios de aceptación**

- [ ] Un checkout limpio puede ejecutar la suite documentada con los servicios de prueba.
- [ ] Las migraciones y semillas pasan desde una base vacía.
- [ ] Un fallo de integración identifica el servicio y la etapa afectada.
- [ ] CI no depende de servicios personales o variables locales no documentadas.

**Dependencias:** `T1`–`T9`, `FT1`–`FT8`, infraestructura de desarrollo.  
**Esfuerzo estimado:** 3 días.

### XT-07 · Privacidad, retención y protección de datos

**Origen:** EF-1, `T8`, `T28`, `FT6`, `FT22`, `FT23` y `FT28`.  
**Prioridad:** P1-X.

Aplicar una política uniforme al tratamiento de cédulas, nombres, correos,
documentos, resultados y exportaciones.

- [ ] Definir qué datos pueden aparecer en logs, errores, auditoría y métricas.
- [ ] Redactar o truncar cédulas, correos y nombres en logs técnicos.
- [ ] No registrar OTP, contraseñas, tokens ni contenido de documentos.
- [ ] Definir retención, purga y excepciones legales para sesiones, documentos,
      expedientes, auditoría, exportaciones y notificaciones.
- [ ] Aplicar autorización y expiración a todas las URLs temporales.
- [ ] Evitar guardar exportaciones o URLs temporales en almacenamiento persistente
      del navegador.
- [ ] Separar listados públicos, reportes internos y expediente administrativo.
- [ ] Versionar la política de privacidad y los cinco consentimientos.
- [ ] Documentar eliminación lógica, eliminación física y trazabilidad de cada una.

**Criterios de aceptación**

- [ ] Una revisión de logs no encuentra OTP, tokens, contraseñas ni PDFs.
- [ ] Las URLs temporales caducan y no permiten acceso tras revocación.
- [ ] Una exportación solo contiene columnas autorizadas para su actor.
- [ ] Existe una prueba o checklist verificable para cada categoría de dato.

**Dependencias:** `T4`, `T6`, `T8`, `T23`, `T28`.  
**Esfuerzo estimado:** 2 días.

### XT-08 · Observabilidad, health checks y operación

**Origen:** RNF operativos, `T10`, `T14`, `T23` y despliegues de desarrollo/
producción.  
**Prioridad:** P1-X.

Permitir distinguir una API viva de una API lista para autenticar, postular,
notificar o descargar documentos.

- [ ] Implementar endpoints de liveness y readiness separados.
- [ ] Indicar dependencias degradadas: PostgreSQL, Redis, SMTP y almacenamiento.
- [ ] Añadir métricas de latencia, errores, rate limiting, OTP, colas, storage y
      exportaciones sin datos personales.
- [ ] Correlacionar logs mediante `X-Request-ID`.
- [ ] Medir fallos de migración, correo, documentos, transacciones y tareas de limpieza.
- [ ] Definir alertas para cola de notificaciones, bajas pendientes y expiración de
      dependencias.
- [ ] Documentar procedimientos para Redis caído, SMTP caído, storage caído y DB read-only.
- [ ] Aplicar límites de tiempo a handlers y operaciones externas.

**Criterios de aceptación**

- [ ] Readiness falla cuando una dependencia obligatoria no está disponible.
- [ ] Liveness continúa funcionando durante una dependencia degradada.
- [ ] Las métricas no contienen cédulas, correos, tokens ni nombres.
- [ ] Un incidente puede rastrearse desde la respuesta hasta los logs mediante request ID.

**Dependencias:** `XT-02`, `XT-05`, `XT-06`, `T10`.  
**Esfuerzo estimado:** 2 días.

### XT-09 · Backups, restauración y recuperación ante fallos

**Origen:** persistencia de `T5`, documentos de `T4`, auditoría de `T8` y
operación de PostgreSQL/RustFS.  
**Prioridad:** P1-X.

Verificar que los datos del expediente y sus documentos puedan restaurarse
manteniendo integridad, referencias y auditoría.

- [ ] Definir frecuencia, retención y cifrado de backups de PostgreSQL.
- [ ] Definir respaldo y restauración del almacenamiento de objetos.
- [ ] Relacionar backup de metadatos con versión de objetos y hashes SHA-256.
- [ ] Documentar recuperación de migraciones y compatibilidad de versiones.
- [ ] Probar restauración en un entorno aislado, sin exponer datos reales.
- [ ] Definir RPO/RTO con el responsable del sistema.
- [ ] Verificar que la restauración preserve sesiones revocadas, expedientes sellados,
      eventos de auditoría y notificaciones pendientes según la política.
- [ ] Documentar el procedimiento y registrar cada simulacro de recuperación.

**Criterios de aceptación**

- [ ] Se puede restaurar una base y sus objetos desde cero.
- [ ] Los documentos restaurados coinciden con sus hashes almacenados.
- [ ] Los expedientes sellados conservan su contenido, hash y trazabilidad.
- [ ] El simulacro mide el tiempo real de recuperación y documenta desviaciones.

**Dependencias:** `T4`, `T5`, `T8`, `XT-06`, `XT-07`.  
**Esfuerzo estimado:** 3 días iniciales y simulacros periódicos.

## 5. Orden de ejecución

1. **XT-01 y XT-02:** cerrar sesiones, CSRF, OTP y rate limiting.
2. **XT-03 y XT-04:** fijar contratos y máquinas de estado antes de ampliar módulos.
3. **XT-05:** persistir notificaciones y estados visibles para ambos clientes.
4. **XT-06:** automatizar la matriz de integración en CI.
5. **XT-07 y XT-08:** aplicar privacidad y observabilidad sobre los flujos reales.
6. **XT-09:** probar restauración y recuperación antes de producción.

## 6. Matriz de relación con los planes existentes

| Capacidad | Backend | Frontend | Transversal |
|---|---|---|---|
| Sesión, cookies y autorización | T2, T6 | FT1–FT3, FT15 | XT-01 |
| Registro, recuperación y OTP | T1, T2, T16 | FT3, FT16 | XT-02 |
| DTO, errores y estados HTTP | T3–T9, T18–T28 | FT1, FT4–FT8, FT16–FT27 | XT-03 |
| Sellado, fases y evaluación | T3, T18, T20–T27 | FT5–FT6, FT17, FT19–FT27 | XT-04 |
| Confirmación y resultados por correo | T19, T23, T27 | FT6, FT22, FT27 | XT-05 |
| Pruebas y despliegue | T1–T9, T10–T14 | FT1–FT13 | XT-06 |
| Datos personales y publicación | T4, T6, T8, T23, T28 | FT6, FT7–FT8, FT23, FT28 | XT-07 |
| Diagnóstico y disponibilidad | T10, T14, T23 | FT4, FT22 | XT-08 |
| Continuidad y recuperación | T4, T5, T8, T12–T13 | FT18, FT22, FT27 | XT-09 |

## 7. Definiciones que requieren aprobación

- Política CSRF y dominios/orígenes confiables por entorno.
- Garantía de entrega de correo: al menos una vez, como máximo una vez o entrega
  lógica idempotente.
- Estados oficiales de notificación y tiempo máximo de reintentos.
- Matriz de transiciones de convocatoria, fase, postulación y evaluación.
- Límites de rate limiting por operación, cédula, correo e IP.
- Retención, eliminación y excepciones legales de expedientes y documentos.
- RPO/RTO, frecuencia de backups y responsables de recuperación.
- Dependencias obligatorias para readiness en cada entorno.

## 8. Criterio de cierre transversal

Las tareas XT no se cierran por tener código aislado. Cada una debe contar con:

- implementación en los módulos afectados;
- contrato o decisión documentada;
- pruebas unitarias y de integración apropiadas;
- logs y errores sanitizados;
- validación en CI o evidencia operativa;
- actualización de `frontend-tasks.md`, `backend-tasks.md` y
  `docs/api-contracts/API_CONTRACT.md` cuando cambie el alcance.

**Ver también:** [`frontend-tasks.md`](frontend-tasks.md) ·
[`backend-tasks.md`](backend-tasks.md) ·
[`../api-contracts/API_CONTRACT.md`](../api-contracts/API_CONTRACT.md) ·
[`../../AGENTS.md`](../../AGENTS.md)
