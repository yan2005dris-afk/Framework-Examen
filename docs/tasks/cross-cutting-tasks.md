# Tareas transversales

**Sistema de Registro y Postulación al Directorio del BIESS**  
Plan transversal para seguridad, contratos, operación e integración del RF v1.4

| Campo | Valor |
|---|---|
| Documento | cross-cutting-tasks v1.1 |
| Ámbito | `apps/frontend/`, `apps/backend/`, `.github/`, `compose.yaml` y `deploy/` |
| Relación | Complementa `frontend-tasks.md` y `backend-tasks.md` |
| Estado | Cierre técnico provisional verificado; XT-03 conserva una brecha explícita de compatibilidad exhaustiva |
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

### XT-01 · Seguridad de sesión, CSRF y cookies [✅ hecho]

**Origen:** EF-1, RNF de seguridad, `T2`, `T6`, `FT2` y `FT3`.  
**Prioridad:** P0-X.

Definir y aplicar una política única para las sesiones basadas en cookies. La
autenticación no debe depender únicamente de que el navegador oculte la cookie.

**Backend**

- [x] Mantener las cookies con `HttpOnly`, `SameSite` y `Secure` según el entorno.
- [x] Definir protección CSRF para `POST`, `PUT`, `PATCH` y `DELETE`.
- [x] Validar `Origin` o el mecanismo CSRF elegido en todas las rutas mutantes.
- [x] No aceptar `Origin` o `Referer` arbitrarios como prueba de confianza.
- [x] Rotar la sesión después de login y de recuperación de contraseña.
- [x] Revocar sesiones anteriores en los eventos definidos por la política de seguridad.
- [x] Aplicar expiración absoluta y, si corresponde, expiración por inactividad.
- [x] No aceptar tokens de sesión en query string, cuerpo o `localStorage`.

**Frontend**

- [x] Enviar el mecanismo CSRF requerido por el backend en todas las mutaciones.
- [x] No persistir tokens, códigos OTP ni datos personales sensibles en
      `localStorage` o `sessionStorage`.
- [x] Tratar `401` y `403` de forma diferenciada.
- [x] Evitar reintentos automáticos de mutaciones no idempotentes.

**Criterios de aceptación**

- [x] Una mutación sin protección CSRF recibe un error estable y no modifica datos.
- [x] Un token de sesión anterior al login o reset deja de ser válido según la política.
- [x] Las cookies tienen atributos correctos en desarrollo y producción.
- [x] Existen pruebas HTTP para origen válido, origen inválido, sesión expirada y
      rotación de sesión.

**Dependencias:** `T2`, `T6`, `FT1`–`FT3`.  
**Esfuerzo estimado:** 2 días.

### XT-02 · Consolidación segura de OTP y rate limiting [✅ hecho]

**Origen:** `T1`, `T2`, `T16`, `FT3` y `FT16`.  
**Prioridad:** P0-X.

Separar el OTP de registro y recuperación como desafíos distintos y garantizar
que los controles de seguridad permanezcan activos cuando Redis no esté disponible.

**Criterios técnicos**

- [x] Vincular cada desafío a cédula, correo, finalidad y un identificador de desafío.
- [x] Invalidar el desafío anterior al reenviar.
- [x] Consumir el OTP de forma atómica y de uso único.
- [x] Aplicar TTL, límite de intentos y límite de reenvíos desde el servidor.
- [x] No permitir utilizar un OTP de registro para recuperación, ni al contrario.
- [x] No revelar la existencia de una cuenta durante recuperación.
- [x] Separar el caché opcional de catálogos del almacenamiento obligatorio de seguridad.
- [x] Si Redis es indispensable y no está disponible, rechazar claramente los flujos
      que dependen de él; no continuar con controles desactivados.
- [x] Limitar por cédula, correo, desafío e IP según la política aprobada.

**Criterios de aceptación**

- [x] Dos verificaciones concurrentes no pueden consumir el mismo OTP.
- [x] Un reenvío invalida el desafío anterior.
- [x] El vencimiento y los intentos restantes provienen de la API.
- [x] Los límites sobreviven al reinicio y a múltiples instancias.
- [x] Existen pruebas de concurrencia, expiración, reenvío, bloqueo y Redis no disponible.

**Dependencias:** `T1`, `T2`, `T16`, `FT3`, `FT16`.  
**Esfuerzo estimado:** 2 días.

### XT-03 · Contratos y compatibilidad frontend/backend [🟡 parcial]

**Origen:** `T3`, `T4`, `T6`–`T9`, `FT1`, `FT4`–`FT8` y `FT16`–`FT27`.  
**Prioridad:** P0-X.

Establecer una única fuente verificable para DTO, estados, errores, paginación,
consentimientos, documentos y notificaciones.

- [x] Mantener `API_CONTRACT.md` actualizado por cada endpoint.
- [x] Documentar request, response, códigos HTTP, códigos de error y campos opcionales.
- [x] Definir los estados de postulación, documento, fase, notificación y evaluación.
- [x] Definir el envelope de errores y su comportamiento para `401`, `403`, `409`,
      `422`, `429` y `5xx`.
- [x] Documentar paginación por offset y cursor, incluyendo el formato opaco del cursor.
- [x] Versionar cambios incompatibles del contrato.
- [x] Alinear nombres de campos EF-12 entre `definition.json`, Go y TypeScript.
- [x] Probar que los cinco consentimientos, referencias de documentos y hashes
      mantienen la misma forma en ambos clientes.
- [x] Añadir pruebas de contrato automatizadas para endpoints críticos.

**Criterios de aceptación**

- [x] El frontend puede generar sus tipos o validaciones desde un contrato mantenido.
- [x] Una respuesta de error del backend se presenta correctamente sin parsear texto libre.
- [x] Los contratos cubren registro, OTP, sesión, postulación, documentos, revisión,
      exportaciones y notificaciones.
- [ ] Toda modificación incompatible de cualquier endpoint falla en CI antes de integrarse.
      Hay contratos críticos y paridad Go/TypeScript; falta una barrera exhaustiva
      de compatibilidad de todos los DTO/rutas (no se afirma que las pruebas actuales la sustituyan).

**Dependencias:** todas las tareas que expongan endpoints.  
**Esfuerzo estimado:** 2 días iniciales y mantenimiento continuo.

### XT-04 · Estados, transiciones e idempotencia de procesos [✅ hecho]

**Origen:** `T3`, `T5`, `T9`, `T18`, `T20`–`T27`, `FT5`–`FT8` y `FT14`–`FT27`.  
**Prioridad:** P0-X.

Definir las máquinas de estado y las operaciones idempotentes para evitar que
reintentos, pestañas abiertas o carreras cambien ilegalmente el expediente.

- [x] Separar estado de postulación, convocatoria, fase, elegibilidad, evaluación,
      reconsideración, publicación y notificación.
- [x] Documentar las transiciones válidas y el actor autorizado para cada una.
- [x] Rechazar guardados, adjuntos, envíos y evaluaciones fuera de plazo en el servidor.
- [x] Hacer idempotentes el sellado, finalización de examen, reenvío y generación
      de notificaciones.
- [x] Usar bloqueo o control de versión para carreras entre cierre y envío.
- [x] Evitar doble creación por reintentos de red o doble clic.
- [x] Conservar el estado anterior, nuevo estado, actor, fecha y motivo en auditoría.

**Criterios de aceptación**

- [x] Cada transición inválida devuelve un error estable y no altera datos.
- [x] Dos solicitudes concurrentes producen un solo resultado definitivo.
- [x] Una pestaña antigua no puede modificar una convocatoria o fase cerrada.
- [x] Las pruebas cubren reintentos, carreras y recuperación tras desconexión.

**Dependencias:** `T5`, `T6`, `T8`, `T18`, `T20`–`T27`.  
**Esfuerzo estimado:** 3 días iniciales y aplicación por módulo.

## 4. P1-X — Operación, privacidad e integración

### XT-05 · Notificaciones persistentes y reintentos [✅ hecho]

**Origen:** `T3`, `T15`, `T19`, `T23`, `T27`, `FT6`, `FT22` y `FT27`.  
**Prioridad:** P1-X.

Separar el resultado de negocio de la entrega del correo. Sellar una postulación
no debe deshacerse por un fallo SMTP, pero tampoco debe presentarse como notificada.

- [x] Crear un registro persistente de notificación con evento, destinatario,
      plantilla, versión, estado y número de intentos.
- [x] Encolar la notificación dentro de la transacción que confirma el evento.
- [x] Enviar después del commit con reintentos y backoff.
- [x] Garantizar idempotencia para que un reintento no duplique el correo lógico.
- [x] Persistir estados `pending`, `sending`, `sent`, `failed` y `retrying` o
      equivalentes documentados.
- [x] Registrar errores sanitizados sin OTP ni datos sensibles.
- [x] Permitir reintento autorizado de notificaciones fallidas.
- [x] Entregar al frontend el estado real de notificación.

**Criterios de aceptación**

- [x] Un fallo SMTP no revierte el sellado ni se presenta como envío exitoso.
- [x] Un reinicio no pierde notificaciones pendientes.
- [x] Existe un único correo lógico por evento; SMTP es **al menos una vez**.
      Un ACK perdido puede duplicar la entrega física: no se promete exactamente
      una vez ni se confunde `smtp_accepted` con recepción en el buzón.
- [x] Existen pruebas de rollback, reintento, duplicado y fallo permanente.

**Dependencias:** `T5`, `T8`, `T19`, `T23`, `FT6`, `FT22`.  
**Esfuerzo estimado:** 3 días.

### XT-06 · Integración, CI y pruebas con servicios reales [✅ hecho]

**Origen:** criterios de verificación de `T1`–`T9`, `FT1`–`FT6` y convenciones de
desarrollo.  
**Prioridad:** P1-X.

Convertir las pruebas que hoy dependen de variables externas en una matriz
reproducible de CI y desarrollo local.

- [x] Levantar PostgreSQL, Redis, SMTP de prueba y almacenamiento S3/RustFS en CI.
- [x] Ejecutar migraciones sobre una base vacía y sobre una base con datos previos.
- [x] Ejecutar pruebas de idempotencia de semillas y rollback de transacciones.
- [x] Ejecutar pruebas de contrato HTTP contra el backend real.
- [x] Ejecutar pruebas de carga/concurrencia para OTP, cuota, envío y examen.
- [x] Mantener secretos y credenciales de CI fuera del repositorio.
- [x] Separar pruebas unitarias, integración y E2E con comandos identificables.
- [x] Publicar artefactos de cobertura, logs sanitizados y diagnósticos de fallos.
- [x] Hacer obligatorios `gofmt`, `go vet`, `go test`, lint, build y formato frontend
      en los cambios correspondientes.

**Criterios de aceptación**

- [x] Un checkout limpio puede ejecutar la suite documentada con los servicios de prueba.
- [x] Las migraciones y semillas pasan desde una base vacía.
- [x] Un fallo de integración identifica el servicio y la etapa afectada.
- [x] CI no depende de servicios personales o variables locales no documentadas.

**Dependencias:** `T1`–`T9`, `FT1`–`FT8`, infraestructura de desarrollo.  
**Esfuerzo estimado:** 3 días.

### XT-07 · Privacidad, retención y protección de datos [✅ perfil técnico provisional]

**Origen:** EF-1, `T8`, `T28`, `FT6`, `FT22`, `FT23` y `FT28`.  
**Prioridad:** P1-X.

Aplicar una política uniforme al tratamiento de cédulas, nombres, correos,
documentos, resultados y exportaciones.

- [x] Definir qué datos pueden aparecer en logs, errores, auditoría y métricas.
- [x] Redactar o truncar cédulas, correos y nombres en logs técnicos.
- [x] No registrar OTP, contraseñas, tokens ni contenido de documentos.
- [x] Definir retención, purga y excepciones legales para sesiones, documentos,
      expedientes, auditoría, exportaciones y notificaciones.
- [x] Aplicar autorización y expiración a todas las URLs temporales.
- [x] Evitar guardar exportaciones o URLs temporales en almacenamiento persistente
      del navegador.
- [x] Separar listados públicos, reportes internos y expediente administrativo.
- [x] Versionar la política de privacidad y los cinco consentimientos.
- [x] Documentar eliminación lógica, eliminación física y trazabilidad de cada una.

**Criterios de aceptación**

- [x] Una revisión de logs no encuentra OTP, tokens, contraseñas ni PDFs.
- [x] Las URLs temporales caducan y no permiten acceso tras revocación.
- [x] Una exportación solo contiene columnas autorizadas para su actor.
- [x] Existe una prueba o checklist verificable para cada categoría de dato.

**Dependencias:** `T4`, `T6`, `T8`, `T23`, `T28`.  
**Esfuerzo estimado:** 2 días.

**Alcance del cierre:** perfil técnico autorizado por el usuario, no política legal.
[Retención y checklist](../privacy/RETENTION_POLICY.md). La revocación inmediata
se aplica a nuevas peticiones mediante capacidades controladas; no retira bytes
ya entregados. No se purgan sellos/auditoría por antigüedad.

### XT-08 · Observabilidad, health checks y operación [✅ hecho]

**Origen:** RNF operativos, `T10`, `T14`, `T23` y despliegues de desarrollo/
producción.  
**Prioridad:** P1-X.

Permitir distinguir una API viva de una API lista para autenticar, postular,
notificar o descargar documentos.

- [x] Implementar endpoints de liveness y readiness separados.
- [x] Indicar dependencias degradadas: PostgreSQL, Redis, SMTP y almacenamiento.
- [x] Añadir métricas de latencia, errores, rate limiting, OTP, colas, storage y
      exportaciones sin datos personales.
- [x] Correlacionar logs mediante `X-Request-ID`.
- [x] Medir fallos de migración, correo, documentos, transacciones y tareas de limpieza.
- [x] Definir alertas para cola de notificaciones, bajas pendientes y expiración de
      dependencias.
- [x] Documentar procedimientos para Redis caído, SMTP caído, storage caído y DB read-only.
- [x] Aplicar límites de tiempo a handlers y operaciones externas.

**Criterios de aceptación**

- [x] Readiness falla cuando una dependencia obligatoria no está disponible.
- [x] Liveness continúa funcionando durante una dependencia degradada.
- [x] Las métricas no contienen cédulas, correos, tokens ni nombres.
- [x] Un incidente puede rastrearse desde la respuesta hasta los logs mediante request ID.

**Dependencias:** `XT-02`, `XT-05`, `XT-06`, `T10`.  
**Esfuerzo estimado:** 2 días.

### XT-09 · Backups, restauración y recuperación ante fallos [✅ perfil técnico provisional]

**Origen:** persistencia de `T5`, documentos de `T4`, auditoría de `T8` y
operación de PostgreSQL/RustFS.  
**Prioridad:** P1-X.

Verificar que los datos del expediente y sus documentos puedan restaurarse
manteniendo integridad, referencias y auditoría.

- [x] Definir frecuencia, retención y cifrado de backups de PostgreSQL.
- [x] Definir respaldo y restauración del almacenamiento de objetos.
- [x] Relacionar backup de metadatos con versión de objetos y hashes SHA-256.
- [x] Documentar recuperación de migraciones y compatibilidad de versiones.
- [x] Probar restauración en un entorno aislado, sin exponer datos reales.
- [x] Definir RPO/RTO con el responsable del sistema.
- [x] Verificar que la restauración preserve sesiones revocadas, expedientes sellados,
      eventos de auditoría y notificaciones pendientes según la política.
- [x] Documentar el procedimiento y registrar cada simulacro de recuperación.

**Criterios de aceptación**

- [x] Se puede restaurar una base y sus objetos desde cero.
- [x] Los documentos restaurados coinciden con sus hashes almacenados.
- [x] Los expedientes sellados conservan su contenido, hash y trazabilidad.
- [x] El simulacro mide el tiempo real de recuperación y documenta desviaciones.

**Dependencias:** `T4`, `T5`, `T8`, `XT-06`, `XT-07`.  
**Esfuerzo estimado:** 3 días iniciales y simulacros periódicos.

**Alcance del cierre:** RPO 24 h / RTO 30 min como objetivos provisionales
autorizados, no SLA institucional. Restauración aislada desde archivos cifrados,
comparación íntegra y posterior invalidación de sesiones. Agenda efectiva,
custodia independiente, copia externa y responsables nominados siguen como
requisitos de producción: [procedimiento](../deployment/RECOVERY_RUNBOOK.md).

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

## 9. Evidencia y condiciones de entrega (2026-11-02)

La evidencia reproducible se resume en el
[informe de cierre técnico](../deployment/CROSS_CUTTING_COMPLETION_REPORT.md).
No se declara ejecución remota de GitHub Actions ni despliegue de producción.

| Área | Implementación / prueba principal |
|---|---|
| XT-01 | Origin estricto; login revoca sesiones anteriores en un commit; fallo de DB devuelve 503 sin borrar cookie; contrato HTTP con API/Mailpit |
| XT-02 | Lua atómico y cuota adicional por correo/finalidad; PostgreSQL/Redis reales y fallo cerrado |
| XT-03 | Contrato mantenido, definición compartida y contratos críticos; pendiente protección exhaustiva contra incompatibilidad |
| XT-04 / XT-05 | Bloqueos/versiones, diario transaccional y outbox cifrado; eventos lógicos únicos, SMTP al menos una vez |
| XT-06 | Suite con servicios reales, migraciones aisladas y workflows preparados; ejecución remota pendiente |
| XT-07 | Matriz de once categorías, checklist/logs y capacidades revocables; 00020 exige cutover de firmas antiguas |
| XT-08 | Readiness completo, gauges de colas, cinco reglas validadas con promtool y runbook; no hay colector/canal desplegado |
| XT-09 | Captura coherente, cifrado autenticado, restauración desde cifrado, manifiestos idénticos y sesiones invalidadas |

### Pendientes que no se ocultan

- XT-03: contrato de máquina / comprobación exhaustiva de compatibilidad de todos
  los endpoints, más allá de la cobertura crítica existente.
- Producción: aprobación institucional de conservación/excepciones, responsables,
  RPO/RTO y reglas de negocio; ejecutar CI remoto, desplegar colector/canal y
  agenda/copia externa, verificar custodia de claves.
- Descargas: retirar firmas S3 antiguas rotando credenciales o esperando su TTL
  sin tráfico. El código nuevo no invalida retroactivamente esas firmas.
- Windows no dispone del compilador CGO requerido para `-race`; no se declara
  aprobado localmente. El workflow Linux sí lo solicita.
