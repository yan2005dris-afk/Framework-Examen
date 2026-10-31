# T15–T18: correo, OTP, elegibilidad y calendario

Implementación técnica del backend, sin despliegue ni cambios en datos reales.
El contrato consumible por FT14–FT17 está en
[COMPETITION_LIFECYCLE.md](../api-contracts/COMPETITION_LIFECYCLE.md).

## Operación y migración

`00012_competition_lifecycle.sql` añade controles y versiones de convocatoria,
evidencias append-only y estado `closed`. No modifica las migraciones ya existentes,
no asigna convocatorias al legado y no habilita períodos al actualizar.

1. Aplicar las migraciones por el procedimiento habitual del backend.
2. Crear y configurar un período desde una cuenta administradora. El servidor
   comprueba nuevamente el rol dentro de la transacción, no solo al abrir la sesión.
3. Aprobar/publicar/habilitar explícitamente y definir la fase `postulacion`.
   Actualizar mediante su versión vigente; 409 indica conflicto de edición.
4. Configurar el proveedor controlado si no existe acceso autorizado al IESS.
   `ELIGIBILITY_SIMULATION_FILE` apunta a un JSON operacional privado, por ejemplo:

   ```json
   {"01900000-0000-7000-8000-000000000001":{"afiliados":true,"jubilados":false}}
   ```

   El UUID debe corresponder a una cuenta **de pruebas**. Es información sintética,
   no evidencia oficial. Campos ausentes no se interpretan como falsos; valores
   nulos, sectores inválidos, archivos demasiado grandes o JSON mal formado se rechazan.
   En contenedores, montar el archivo explícitamente y configurar su ruta interna;
   no ubicarlo entre adjuntos de postulantes ni versionar datos operacionales.
5. Conservar `MAIL_OUTBOX_KEY` estable. Configurar SMTP antes de ofrecer OTP.
   Para reenvíos, `OTP_RESEND_DELAY=1m` y `OTP_MAX_SENDS=3` son los valores por defecto.

## Garantías y límites

- SQL valida la ventana después de obtener bloqueos y antes de aceptar escrituras.
  La apertura es inclusiva y el cierre exclusivo, con zona de presentación
  `America/Guayaquil`. Los instantes se normalizan a microsegundos para concordar
  con PostgreSQL; estas convenciones necesitan validación institucional.
- El contenido enviado sigue siendo inmutable. No se recalifica un histórico con
  el perfil actual ni se mezcla el cierre por elegibilidad con una fase o mérito.
- Un cierre se confirma y audita antes de responder 409. Un fallo de proveedor,
  persistencia o auditoría revierte la operación y no se declara como inelegibilidad.
  Las nuevas postulaciones del mismo período no pueden evadir un cierre anterior.
- No hay reapertura administrativa automática: debe definirse y aprobarse su política.
- La cola conserva cifrado, leases y reintentos. La identidad de la operación no
  cambia por un vencimiento nuevo. Un error al cerrar SMTP después de aceptar DATA
  no obliga a reenviar; un corte antes de registrar esa aceptación todavía puede
  producir duplicados. `smtp_accepted` nunca significa llegada certificada al buzón.
- La cuota OTP conserva su ventana original. La espera por cuota agotada no se
  confunde con el cooldown, y los fallos concurrentes no reinician un bloqueo activo.

## Verificación

Se ejecutó `go test ./... -count=1` con PostgreSQL 18 temporal en esquemas aislados
por prueba y Redis 7 real (`REDIS_TEST_URL`), validando TTL, monotonicidad, no renovación
de ventana y consumo único concurrente (`TestRedisSecurityIntegrationTTLAndAtomicity`).
SMTP se validó mediante servidor TCP de pruebas. Los tests cubren también rollback de la
cola, reintentos, idempotencia, consumo único OTP, plazos de adjuntos, cierres
persistentes y rechazo de escrituras desde una pestaña anterior al cierre.

Además: `go vet ./...`, `go build ./...`, `gofmt` y regeneración con sqlc v1.31.1.
El entorno Windows local tiene `CGO_ENABLED=0`; `-race` queda para CI con soporte CGO.

Durante la validación final se resolvieron los siguientes ajustes:
1. Las comparaciones de fechas de fase normalizan a microsegundos para concordar con
   el redondeo de PostgreSQL.
2. La consulta de deduplicación de la cola de correo requirió un tipo explícito en SQL
   para el parámetro reutilizado en `INSERT SELECT` y el filtro.
3. El fixture de formulario completo (`serviceFixture(t, true)` en `service_test.go`)
   se actualizó con evidencia de evaluación de elegibilidad para cumplir las reglas
   estrictas de lectura/cálculo de avance de T17.
4. Se validó la integración del frontend (FT14–FT15) con el contrato estricto T18:
   cero respuestas ficticias, manejo honesto de convocatorias no publicadas o cerradas,
   y control de concurrencia optimista (`version`).
5. Se concluyó la batería de pruebas y se retiraron los contenedores temporales de QA
   (`framework-t16-redis-qa-20261031` y `framework-t15-t18-qa-20261031`).

## Unidades de cambio y reversión

- **T15:** correo/outbox, sus queries y pruebas. Puede revertirse separado del
  calendario; no borra la cola existente ni requiere recrear datos.
- **T16:** caché de controles, identidad, configuración y pruebas. Conservar los
  valores previos de configuración si se revierte; no vaciar Redis ni revocar
  sesiones ajenas como parte de una reversión técnica.
- **T17/T18:** contextos `eligibility`/`competition`, integración de submission/document,
  migración 00012 y código generado forman una unidad de invariantes. No retirar
  sus controles mientras existan cierres. El Down rechaza perder su semántica;
  una reversión requiere plan explícito, nunca purgar evidencias o históricos.

Los archivos del frontend y la rama compartida pertenecen al trabajo concurrente
FT14/FT15. No se modifican ni se revierten desde esta implementación.
