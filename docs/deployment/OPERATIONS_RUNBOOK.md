# Operación y alertas del perfil transversal

## Disponibilidad

`/healthz` sólo comprueba que el proceso responde. `/readyz` comprueba en paralelo
con límite de dos segundos: **PostgreSQL escribible**, Redis realmente usado por
seguridad, bucket privado de storage y esquema/clave del outbox, más listener de descargas. Si falla uno de
ellos responde 503; SMTP caído devuelve 200 con `status: degraded` porque la cola
puede aceptar trabajo durable sin entregar inmediatamente.

La sonda SMTP hace handshake/NOOP sin enviar mensajes; no prueba credenciales ni
entrega. S3 usa HEAD firmado del bucket sin leer PDFs; no demuestra todos los
permisos de escritura. En modo local se verifica el directorio de storage; la
sonda no certifica que un servidor local separado esté accesible al navegador.
Si Redis no pudo conectarse durante arranque, el adaptador queda cerrado: corregir
la dependencia y reiniciar API, no sustituirlo por un caché sin seguridad.

Las respuestas sólo entregan nombres cerrados y `ok`/`no_disponible`; no incluyen
hosts, URLs firmadas, consultas ni credenciales. `Cache-Control: no-store`.

## Incidentes

| Señal | Acción | Evitar |
|---|---|---|
| Redis no disponible | Revisar servicio/red/credenciales; recuperar; verificar OTP y readiness. Reiniciar API si falló su conexión inicial | Noop para seguridad o borrar bloqueos para reintentar |
| SMTP degradado / backlog | Recuperar SMTP, comprobar worker/leases y estados; reintento autorizado sólo si agotado | Marcar entregado, reenviar aceptados o prometer exactamente una entrega |
| Storage no disponible | Recuperar bucket/credenciales, verificar HEAD y round-trip; dejar bajas en cola | Confirmar un PDF inexistente o perder compensaciones |
| PostgreSQL read-only | Detener mutaciones por readiness, revisar primary/rol y recuperación antes de reabrir | Promover un nodo sin coordinar, ignorar escrituras fallidas |
| Migración falla | Mantener fuera de tráfico; registrar versión y restaurar en otra copia | Down destructivo con evidencia histórica |
| Cola de bajas crece | Revisar permisos S3/timeout, reintentos y confirmación de objetos ausentes | Borrar fila de tarea sin borrar objeto |

## Métricas y alertas

El puerto separado `METRICS_ADDR` debe estar en red privada. Los contadores e
histogramas tienen etiquetas cerradas; los gauges de backlog sólo permiten
`outbox`, `failed_mail`, `object_deletions`, `upload_intents`.
`deploy/monitoring/alerts.yml` define umbrales provisionales: API/métricas caídas
2 min, correo agotado 5 min, >100 pendientes 10 min, bajas/cargas vencidas 15 min.

El colector debe configurar job `backend` para scrape y un blackbox probe
`backend-readiness` contra `/readyz`. Estos archivos **no instalan un colector ni
un canal de alertas**; operaciones debe cargar reglas, verificar `promtool` y probar
notificación antes de producción. No se afirma que alguien recibe alertas por
crear un YAML. No imprimir URLs firmadas ni cuerpos en diagnósticos.

`X-Request-ID` válido UUID permite buscar la solicitud en logs. El adaptador W3C
local no exporta OTLP ni crea automáticamente trazas distribuidas. Los 429 de
nginx se miden en su proxy; los 429 de Go en las métricas HTTP de la API.


## Descargas revocables

`DOWNLOAD_PUBLIC_URL` debe apuntar al proxy público con `/downloads`; el listener
8082 queda privado. La API sirve capacidades opacas, no URLs firmadas S3. No pasar
esa ruta por CDN, caché o logs de request URI, ni habilitar redirects/rangos sin
conservar la validación. La caída de DB, pérdida de sesión, rol o política deniega
la descarga incluso si el enlace aún no expiró. El stream privado de storage se
verifica y se vuelve a autorizar antes de los bytes.

Durante el despliegue desde firmas S3 directas antiguas, cerrar emisión vieja y
rotar credenciales de firma (coordinando RustFS/API) o esperar el máximo TTL fuera
de tráfico. Ninguna rotación borra objetos. Ese paso es obligatorio antes de
prometer que todos los enlaces previos son revocables; no se ejecutó en producción.
