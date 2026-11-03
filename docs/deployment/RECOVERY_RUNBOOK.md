# Recuperación provisional de PostgreSQL y objetos

**Perfil:** `tecnico-provisional-v1`, autorizado para QA el 2026-11-01.
**Objetivos**, no mediciones ni compromiso de producción: RPO 24 h, RTO 30 min.
Responsable funcional propuesto: administrador del sistema; operador de respaldo:
operaciones; aprobación/legal: institución. Falta asignar personas reales antes
 de producción. No se conecta ningún almacenamiento o correo real.

## Ruta rápida

```powershell
./scripts/backend-demo/Start-Demo.ps1
./scripts/backend-demo/Backup-Restore-Demo.ps1
```

El script sólo opera la base `framework_synthetic_demo` y sus contenedores
etiquetados. Detiene brevemente su API para una captura coherente. No detiene el
stack compartido del frontend ni borra backups anteriores. Si el puerto 19010 está
ocupado por un simulacro previo, detener sólo el contenedor indicado por su
`restore-evidence.txt` después de comprobar `data.origin=synthetic-demo`.

## Frecuencia, conservación y claves

| Elemento | Perfil técnico |
|---|---|
| Frecuencia | Diario y antes de una migración significativa; el operador debe activar su agenda local. No se creó silenciosamente una tarea programada |
| Retención | Siete días de capturas verificadas; conservar una copia independiente y todo hold. Revisión antes de purgar, nunca borrar el último respaldo válido |
| Cifrado | AES-256-GCM, bloques de 1 MiB, subclave por archivo HMAC-SHA256 con sal aleatoria de 32 bytes, índice y tamaño autenticados, marcador final autenticado |
| Clave | `BACKUP_ENCRYPTION_KEY`, 32 bytes base64; se genera en `.env` ignorado, **fuera** del backup. Para recuperación real: custodia independiente/offline con acceso controlado |
| Cargas operativas | Clave de outbox y credenciales S3 dentro de `recovery.env.enc`; nunca guardar la clave de backup dentro de este archivo |
| Versiones | Registrar commit/binario, versión goose, objetos y hashes. Restaurar primero con versiones compatibles, no migrar a ciegas una copia antigua |
| Destino | Archivo cifrado en volumen distinto y copia fuera del equipo como requisito antes de producción; el simulacro local no prueba off-site |

No se usa una contraseña humana como clave AES. Una clave perdida hace
irrecuperable el respaldo. La clave se recibe por entorno, no argumentos ni logs.
El lector rechaza alteraciones, truncamiento, reordenamiento y datos adicionales;
no confirma el destino hasta verificar todo el flujo.

## Qué hace el simulacro

1. Valida sellos, objetos, PDFs y capacidad de descifrar el outbox de origen.
2. Captura PostgreSQL con `pg_dump -Fc` y el volumen coherente de RustFS.
3. Cifra dump, objetos y credenciales. Descifra a temporales privados y compara
   hashes. **La restauración usa los archivos descifrados**, no los originales.
4. Crea otra base `synthetic_restore_<fecha>`, otro volumen y otro RustFS.
5. Compara los manifiestos lógicos de **todas las tablas** y los SHA-256 de cada
   objeto referenciado/PDF; valida sellos y cargas cifradas. Incluye migraciones,
   consentimientos, auditoría y notificaciones pendientes.
6. Tras comparar el snapshot completo, invalida todas las sesiones de la copia
   restaurada. Los manifiestos `before/restored` prueban el snapshot **antes** de
   esta invalidación deliberada; `recovery-timing.json` registra la acción.
7. Mide tiempo de captura/restauración/verificación e informa desviación del RTO.
   No afirma haber medido un RPO diario sin observar una agenda de capturas.
8. Retira sólo temporales nuevos de ese simulacro tras verificar el cifrado y la
   recuperación. Conserva `.enc`, manifiestos y evidencia; en fallo preserva
   archivos para diagnóstico y reactiva la API de origen.

Las carpetas de pruebas históricas anteriores al cifrado no se convierten ni se
borran automáticamente. Pueden contener dumps o claves en claro: mantenerlas
restringidas, no copiarlas fuera y revisarlas con el operador. Sólo los nuevos
simulacros declaran `restoreFromEncrypted=true`.

## Recuperación operativa y reapertura

- Aislar entorno y SMTP; preparar credenciales/clave desde custodia autorizada.
- Verificar hashes de `.enc` y recuperar usando `backup-crypt decrypt` a un destino
  **nuevo**, nunca sobrescribir la copia ni extraer sobre la base actual.
- Restaurar imagen compatible de PostgreSQL 18 y RustFS y aplicar la versión
  goose registrada. Las migraciones posteriores se ensayan en otra copia.
- Comparar sellos, PDFs, referencias, diario, outbox y hash de todas las tablas.
- Invalidar sesiones, reaplicar restricciones/bloqueos posteriores al backup,
  revisar outbox (SMTP puede aceptar una entrega cuyo ACK se perdió: al menos una
  vez, deduplicación lógica, no exactamente una vez).
- Abrir tráfico sólo con `/readyz` y comprobación funcional correctos; documentar
  pérdida de datos observada, tiempo total y responsable de autorización.

## Registro por simulacro

`recovery-timing.json` contiene inicio/fin UTC, segundos medidos, objetivos,
confirmación de restauración desde cifrado y sesiones invalidadas.
`archive-sha256.json` contiene hashes de archivos cifrados.
`before.json`/`restored.json` contienen hashes agregados y recuentos, no contenido.
`restore-evidence.txt` identifica base/volumen/contenedor para evitar borrar el
entorno equivocado. Todo queda en la carpeta QA ignorada.

**No aprobado aún para producción:** responsables nominados, agenda efectiva,
retención/excepciones legales, custodia/rotación de claves, copia off-site y RPO/RTO
institucionales. El usuario autorizó el perfil técnico provisional, no estas
aprobaciones ni una eliminación sobre datos reales.
