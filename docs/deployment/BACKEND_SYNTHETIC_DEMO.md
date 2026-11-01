# Backend: demostración sintética y recuperación

**Ámbito:** desarrollo local. Ningún resultado es institucional ni una prueba de
producción. Parámetros: [PARAMETROS-SINTETICOS](../PARAMETROS-SINTETICOS.md).
La corrida no toca los servicios habituales usados por el frontend.

## Ejecutar

Desde la raíz, en PowerShell con Go y Docker disponibles:

```powershell
./scripts/backend-demo/Start-Demo.ps1
./scripts/backend-demo/Invoke-Demo.ps1
./scripts/backend-demo/Invoke-Demo.ps1 -Faults
./scripts/backend-demo/Test-Demo.ps1
./scripts/backend-demo/Backup-Restore-Demo.ps1
```

Ejecutar los comandos **en serie**: no correr pruebas mientras `-Faults` detiene
servicios. Un `.env` nuevo habilita la excepción QA autorizada (`DEMO_FAST_EXPIRY=1`)
y el dataset `demo`; la corrida local verificada se conserva como `verified-v3`.
El inicio compila binarios Linux y reutiliza la imagen local
`framework-examen-api:latest`. Si falta: `docker compose build api`.
El contrato `TestS3RuntimeRoundTrip` aprovisiona el bucket privado de QA y verifica
PUT/GET/DELETE con SigV4 real; no se crea un bucket de producción.

| Servicio propio | Dirección local |
|---|---|
| API / liveness / readiness | `http://localhost:18080` |
| nginx, límite general | `http://localhost:18081` |
| Métricas privadas | `http://localhost:19090/metrics` |
| Mailpit | `http://localhost:18025` |
| PostgreSQL | `localhost:15432`, base `framework_synthetic_demo` |
| Redis de API / integración | `localhost:16379`, DB 0 / DB 15 |
| RustFS privado | `http://localhost:19000/synthetic-demo` |

El proyecto Compose es `framework-backend-demo`; los puertos sólo escuchan en
loopback. Los contenedores y volúmenes tienen etiqueta `data.origin=synthetic-demo`.
No incluye otra SPA: el agente de frontend conserva su entorno habitual.

## Estado y protección de datos

`Start-Demo.ps1` genera secretos aleatorios en `deploy/demo/.env`, ignorado por Git.
No imprimir ese archivo, el estado de cuentas ni volcados del entorno Docker.
El bootstrap aplica hasta 00010 en una base nueva, crea un legado ficticio sin
convocatoria/consentimiento y migra luego hasta **00019**. No desactiva triggers
ni reescribe un envío existente. En bases ya inicializadas omite ese bootstrap.

El cargador exige `DEMO_CONFIRM=synthetic-demo` y rechaza cualquier URL distinta de
`localhost:15432/framework_synthetic_demo`. No recibe destinos arbitrarios.

Los artefactos privados viven en `apps/backend/data/synthetic-demo/`, también
ignorado. `DEMO_DATASET` permite subdirectorios alfanuméricos; la corrida verificada
usa `verified-v3`. El `state.json` contiene contraseñas ficticias generadas: no se
comparte ni versiona. Conserva IDs y progreso para reanudar sin duplicar expedientes.
Los bloqueos OTP no se borran: se respeta `Retry-After`; Mailpit se filtra por fecha
para no reutilizar un código de una emisión anterior.

Para una **nueva corrida**, elegir `DEMO_DATASET`, `DEMO_POSTGRES_VOLUME` y
`DEMO_OBJECT_VOLUME` nuevos en el `.env` privado. Cambiar estos nombres conserva
los volúmenes previos; no ejecutar `down -v`. Los volúmenes originales de la
primera corrida y su respaldo se conservaron. No mezclar un estado de cuentas
anterior con otra base vacía.

`DEMO_FAST_EXPIRY=1` habilita la configuración QA de 5 segundos autorizada; sólo
se usa para el escenario de vencimiento. Sin esa opción, el cargador deja el
intento de 3600 segundos abierto y debe reanudarse tras vencer. Nunca modifica
fechas de un intento existente ni fabrica la nota faltante.

## Qué comprueba el recorrido

- Registro/OTP real en Redis y Mailpit, inicio de sesión y roles iniciales
  auditados con operador/motivo sintético.
- Convocatoria publicada, política aprobada y aceptación ID/hash en el sello.
- Guardado de nueve secciones, borrador reanudable, documentos PDF válidos,
  impresión EF-12, cierre por elegibilidad no evadible y envío inmutable.
- Revisión documentada, cálculo y publicación de méritos; checklist sin puntos
  si quedan requisitos o inhabilidades sin confirmar.
- Instrumento `PSICO-DEMO-1`, umbral 60, publicación administrativa y consolidación
  de fuentes 40/30/30; el cliente no introduce una nota teórica libre.
- Banco de 20 preguntas neutrales, diez asignadas, examen principal de una hora,
  respuesta versionada, finalización idempotente, PDF y vencimiento corto separado.
- Reconsideración de titular → analista → administrador; tres estados simultáneos
  aislados en jubilados, sin bloquear la lista de afiliados.
- Lista interna/CSV y proyección pública sólo de nombres consentidos.
- Rechazos HTTP 401/403/409/422/413; `-Faults` cubre 429 y 503 reales.

La notificación `pending` de fixture tiene su próximo intento pospuesto 24 horas.
La `failed` usa una dirección deliberadamente inválida: pasa por el compositor
SMTP y cinco intentos del worker. Sólo para este fixture se adelanta la agenda de
reintentos en su base sintética; **no** se modifican plazos de resultados/exámenes.
El resto de avisos se acepta en Mailpit; eso no demuestra entrega a un buzón real.

## Observabilidad y límites

`METRICS_ADDR` abre un servidor independiente, loopback por defecto. No se añade
una ruta pública de métricas al proxy. Los contadores e histogramas Prometheus
sólo admiten componentes, operaciones y resultados de una lista cerrada; no hay
etiquetas con IDs, cédulas, correos, nombres, cuerpos o claves de objetos.

Ejemplo de latencia p95 por componente:

```promql
histogram_quantile(0.95, sum by (le, component) (rate(backend_operation_duration_seconds_bucket[5m])))
```

Se registran HTTP, OTP, outbox (despacho y entrega), almacenamiento, transacciones,
exportaciones, migraciones y limpiezas. Los 429 del propio nginx se verifican en su
respuesta/log; no llegan al contador HTTP de Go. No confundir ambas mediciones.

**Adaptador de trazas elegido:** JSON estructurado con identificadores W3C
`trace_id`, `span_id`, cabecera `Traceparent` y `X-Request-ID` UUID. Se registran
plantillas de rutas, nunca query strings o URLs con identificadores personales.
Es un adaptador local, **no** un SDK OpenTelemetry, ni exportador OTLP, ni trazado
distribuido de suboperaciones. Debe integrarse con un colector antes de afirmar
trazas distribuidas. No se propaga `baggage` del cliente.

Redis tiene límites de conexión/lectura/escritura/pool y de operación (3 s).
No se repite automáticamente un script de seguridad tras una respuesta de red
ambigua. S3 limita cada petición a 10 s; SMTP usa un contexto de 15 s en el worker.
Los handlers propagan `HTTP_REQUEST_TIMEOUT`; el servidor añade sus límites de
lectura/escritura. Esto no interrumpe por fuerza código que ignore el contexto.

La purga corre al iniciar y cada hora: lote de hasta 1000 sesiones con más de
30 días expiradas; no elimina sesiones activas ni recién expiradas. El volumen
por hora debe vigilarse para evitar acumulación sobre ese ritmo. Las bajas
confirmadas son durables; una compensación de carga fallida también se encola si
RustFS no responde. Un cierre abrupto entre escribir un objeto y registrar sus
metadatos aún necesita inventario de huérfanos: no se afirma recuperación total
de esa ventana. No se borran expedientes o documentos por antigüedad.

`DATA_ORIGIN=synthetic-demo`, sólo fuera de producción, marca PDFs, correos y CSV
como **DATOS SINTÉTICOS - NO VÁLIDO**. El modo normal no añade esa marca.

## Fallos controlados

`Invoke-Demo.ps1 -Faults` sólo puede detener los contenedores de su proyecto:

| Dependencia | Aserción |
|---|---|
| Redis | OTP responde 503; no hay fallback inseguro |
| PostgreSQL | liveness 200, readiness 503; vuelve a estar listo |
| RustFS | carga 503 sin documento confirmado; descarga real tras volver |
| SMTP | aceptación durable pendiente, fallo, reinicio API y reintento en Mailpit |
| nginx | ráfaga concurrente produce 429 por IP |

Cada dependencia se recupera en `defer` incluso si falla una aserción. Readiness
actual comprueba **PostgreSQL**, no certifica disponibilidad de SMTP/S3/Redis;
por eso se prueban aparte. Los errores de proveedores no se copian a logs con
credenciales ni a respuestas de usuario.

## Respaldo y restauración comprobables

El script detiene sólo la API sintética, valida sellos/objetos y hace `pg_dump`
formato custom. Detiene brevemente su RustFS para archivar un volumen coherente.
Después crea una **base nueva** `synthetic_restore_<fecha>` y un **volumen y
contenedor nuevos** de objetos (puerto 19010). No restaura sobre la base fuente.

`cmd/demo-verify` compara los SHA-256 lógicos de todas las tablas (incluye
migraciones, auditoría, outbox y reportes), valida cada sello, cada PDF de examen almacenado y descarga todos los
objetos referenciados para cotejar SHA-256. Descifra también todas las cargas de
outbox todavía presentes, sin imprimirlas, con la clave conservada. Los manifiestos anterior/restaurado
deben ser idénticos. También se guardan hashes de los archivos de respaldo.

La clave de cifrado del outbox y las credenciales S3 se conservan en un
`recovery.env` **privado** junto al respaldo. Sin esa clave no se pueden recuperar
cargas pendientes cifradas. No publicar el archivo ni restaurar la cola contra un
SMTP real. El script deja el contenedor restaurado disponible para inspección;
antes de repetir hay que liberar su puerto 19010 deteniendo **sólo** ese contenedor
identificado en `restore-evidence.txt`, sin borrar volúmenes ni backups.

## Evidencia y límites de cierre

El directorio privado contiene `state.json`, PDFs, banco JSON, CSV, `quality.json`,
`go-test.jsonl`, `fault-evidence.json` `audit-evidence.json` y carpetas `backup-*`. Las pruebas Go se
activan con URL explícita y crean/eliminan sus propios esquemas, no datos de la
aplicación. La simulación S3 usa un servidor HTTP embebido de Go: no se presenta
como ejecución de Postman ni reemplaza el round-trip real contra RustFS.

La detección `-race` se intentó, pero Windows no tiene `gcc` disponible para CGO.
No se presenta como aprobada. Los tests de concurrencia de negocio sí se ejecutan
contra PostgreSQL y Redis reales. La regeneración sqlc debe ser idempotente.

### Interpretación del cierre

- La auditoría uniforme rige desde 00019; no se atribuye retrospectivamente el
  legado creado bajo 00010. Su acceso privado se bloquea por falta de aceptación.
- La vigencia de nuevas aceptaciones de la política de la demo queda acotada por
  la convocatoria de 2026. El modelo de política no añade un calendario legal
  independiente; las ventanas posteriores siguen la retención desde el sello.
- Retención es política/ventanas y revisión humana, no prueba de borrado automático
  ni cumplimiento jurídico. Las sesiones sí se purgan conforme al lote descrito.
- No hay producción, correo externo, reglas institucionales ni prueba de espera
  real de 60 minutos. La excepción QA autorizada comprueba 5 s reales.
- `-race` permanece no ejecutable por falta de GCC/CGO; no se informa como verde.

La evidencia sólo representa la versión y fecha de su corrida. Conservar el
respaldo/estado correspondiente y no presentar resultados antiguos como validación
de código modificado después.
