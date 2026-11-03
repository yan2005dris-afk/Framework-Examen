# Cierre técnico del backend sintético

**Fecha:** 2026-10-31, America/Guayaquil. **Estado:** demostración técnica verificada;
no producción ni aprobación institucional. Se trabajó sólo en backend/operación,
preservando los cambios concurrentes y el stack del frontend.

## Resultado

- Corrida `verified-v3`, proyecto `framework-backend-demo`, API en
  `http://localhost:18080`, sin datos de postulantes reales.
- Migraciones **00001–00019** desde vacío. Legado ficticio creado bajo 00010 y
  migrado conservando su hash, sin inventar aceptación de privacidad.
- EF-12 completo, borrador reanudable, PDFs válidos y reemplazo/descarga temporal
  con URL de 300 segundos y SHA-256 comprobado.
- Méritos con checklist y unidades completas; psicotécnica con umbral 60;
  consolidación **40/30/30** confirmada por el usuario.
- Banco principal 20/10, examen de **3600 segundos**, nota calculada por servidor,
  finalización idempotente, PDF y correo. Vencimiento real en configuración
  **QA de 5 segundos**, independiente y autorizada: respuesta parcial → nota 10.
- Reconsideración completa y fixtures simultáneos recibida/atendida/resuelta en
  jubilados; lista interna y publicación consentida de afiliados.
- Notificaciones pendientes/fallidas/aceptadas, PDF recibido en Mailpit idéntico
  al descargado por HTTP. No se afirma entrega a buzones externos.

## Operación y auditoría

La nueva migración 00019 conserva actor, rol, recurso, antes/después, motivo,
fecha y solicitud dentro del commit. Usa proyecciones/hashes, no otra copia del
formulario. Incluye creación y respuestas de examen; las lecturas administrativas
registran acceso antes de liberar datos. Se probaron rollback por fallo de diario,
append-only y conservación tras baja de borrador.

Se corrigieron dos fallos detectados durante ejecución: Redis podía agotar el
plazo HTTP con reintentos/pool; ahora limita operaciones y no repite scripts tras
resultados de red ambiguos. Un SELECT de configuración de auditoría impedía pedir
REPEATABLE READ en exportaciones; se sustituyó por SET LOCAL sin adquirir snapshot.
La caída de S3 ahora entrega 503 y conserva la compensación de la carga fallida.

Hay métricas Prometheus privadas, trazas locales W3C/JSON explícitas, purga por
lotes de sesiones y límite general de nginx. Se comprobaron caídas y recuperación
de Redis/PostgreSQL/RustFS/SMTP, reinicio de API con correo pendiente y 429 real.

## Calidad y recuperación

| Comprobación | Resultado |
|---|---|
| `go test ./... -count=1` con servicios reales | **608 aprobadas, 0 omitidas** |
| Formato, `go vet`, `go build` | Aprobados |
| sqlc regenerado sin variaciones | Aprobado |
| Migraciones desde vacío | Hasta 00019 |
| nginx de demo y producción (`nginx -t`) | Aprobados; producción no desplegada |
| Códigos negativos | 401, 403, 409, 422, 429, 413, 503 |
| Escaneo de secretos y exclusión de QA | Aprobados |
| `-race` | **No disponible:** falta GCC/CGO en Windows |

Última restauración: `backup-20261031232325`. Se restauró en **otra base y otro
volumen/contenedor**, sin sobrescribir el origen. Verificados **6 sellos, 24
objetos SHA-256, 4 PDFs de examen y 2 cargas cifradas recuperables del outbox**.
Los manifiestos de todas las tablas, incluidos auditoría/outbox/migraciones,
coincidieron exactamente. Los backups previos no se eliminaron.

## Límites

- Parámetros, textos y resultados sintéticos, marcados **DATOS SINTÉTICOS - NO VÁLIDO**.
- El adaptador de trazas no es un SDK OpenTelemetry ni exporta OTLP.
- No se esperaron 60 minutos: el vencimiento medido fue el QA autorizado de 5 s.
- No hay purga automática de expedientes ni certificación legal de retenciones.
- Auditoría uniforme desde 00019; no hay atribución retrospectiva del legado.
- Readiness comprueba PostgreSQL; los otros servicios se verificaron por separado.
- Un cierre abrupto entre escribir un objeto y guardar metadatos todavía requiere
  inventario operativo de huérfanos; no se promete recuperación universal.

## Repetir y consultar evidencia

[Guía de ejecución y recuperación](BACKEND_SYNTHETIC_DEMO.md) ·
[Matriz de auditoría](../architecture/BACKEND_AUDIT_MATRIX.md) ·
[Parámetros](../PARAMETROS-SINTETICOS.md).

La evidencia privada vive en `apps/backend/data/synthetic-demo/verified-v3/`:
`quality.json`, `go-test.jsonl`, `state.json`, `audit-evidence.json`,
`fault-evidence.json`, PDFs/CSV y `backup-*`. **No publicar esa carpeta:** contiene
credenciales ficticias, claves de recuperación y volcados. Este informe sólo
contiene resultados agregados y no reemplaza esa evidencia fechada.
