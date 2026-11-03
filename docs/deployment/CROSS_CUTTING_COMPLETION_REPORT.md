# Entrega transversal: verificación técnica provisional

**Fecha:** 2026-11-02 · **Ámbito:** desarrollo y datos sintéticos.

Se corrigieron controles que estaban marcados como terminados sin cumplir todo
su criterio. XT-01/02/04/05/06/08 cuentan con implementación y evidencia local;
XT-07/09 se cierran únicamente bajo el **perfil técnico provisional autorizado**.
**XT-03 queda parcial:** no hay una barrera exhaustiva de compatibilidad de todos
los endpoints. No se certifica producción, aprobación institucional ni CI remoto.

## Cambios que afectan al uso

- Mutaciones sin un único `Origin` permitido: 403, sin ejecutar la operación.
  Login revoca sesiones anteriores e inserta la nueva de forma atómica; si falla,
  todo revierte. Un error de DB al consultar sesión es 503, no un logout falso.
- OTP limita también por correo/finalidad, además de cédula y desafío. Redis es
  obligatorio para seguridad; sus fallos no desactivan controles.
- Readiness distingue DB escribible, Redis, almacenamiento, outbox y listener de
  descargas obligatorios. SMTP es degradable porque el trabajo queda en cola.
- Las URLs de documentos conservan `{url, expiresAt}`, pero son capacidades
  opacas: **no se entrega una firma S3 al navegador**. Un listener privado vuelve
  a comprobar sesión, rol, titularidad y privacidad y audita cada GET/HEAD.
  También revalida después de leer/verificar el PDF, antes de responder.
- La migración **00020** almacena sólo hashes de capacidades, su vínculo a sesión
  y versión, y un diario append-only de acceso para los tres roles. No modifica
  el sello histórico ni reescribe migraciones ya desplegadas.
- Hay gauges cerrados de colas y cinco reglas de alerta; no se instaló colector,
  agenda ni canal de notificación. Los workflows incorporan servicios de prueba
  y contrato HTTP real con API/Mailpit; no se ejecutaron remotamente.
- El simulacro cifra DB, objetos y credenciales operativas con autenticación,
  restaura desde el descifrado verificado y compara todas las tablas y hashes.
  Después invalida las sesiones de la copia restaurada.
- Reejecutar la demostración crea otro recurso sintético si un aviso de prueba
  venció; conserva el original y su historial, sin revivirlo ni borrarlo.
- La prueba Redis real usa un correo sintético único por corrida: reutilizar uno
  fijo agotaba correctamente la nueva cuota compartida al repetir la suite. Se
  corrigió el aislamiento del test, no se relajó ni borró el control de seguridad.

## Verificación local

| Comprobación | Resultado |
|---|---|
| `Test-Demo.ps1` | 669 pruebas Go, cero omitidas; gofmt, vet y build aprobados |
| `Test-Privacy.ps1` | 120 pruebas, cero omitidas; once categorías y cero infracciones detectadas en logs sintéticos |
| Frontend | 356 pruebas / 29 archivos; lint, build y formato aprobados |
| `Invoke-Demo.ps1` | Recorrido HTTP con servicios reales; capacidad emitida denegada tras bloqueo, no revive al reabrir, se invalida al cerrar sesión |
| Revocación durante lectura | Prueba con PostgreSQL real y proveedor HTTP de prueba: no libera PDF si la sesión se elimina mientras responde el proveedor |
| Fallos controlados | Redis/DB/RustFS y SMTP recuperados; liveness/readiness diferenciados; ráfaga real produjo 69 respuestas 429 en nginx |
| sqlc | Regeneración idéntica, sin archivos desactualizados |
| nginx | Configuraciones de demo y producción aprobadas con `nginx -t` en contenedores locales; no se desplegó producción |
| Prometheus | `promtool check rules`: cinco reglas válidas |
| Workflows / Compose | YAML válido; no equivale a una ejecución verde de GitHub Actions |
| Salud final | Seis dependencias `ok`, estado `ready` |

Se usaron PostgreSQL, Redis, RustFS, Mailpit y nginx del proyecto separado
`framework-backend-demo`. Las simulaciones HTTP embebidas son pruebas Go, **no
ejecuciones de Postman**. No se reconstruyó ni detuvo el stack compartido del
frontend. Fue necesario iniciar Docker Desktop, que estaba apagado al retomar.

### Restauración observada

Simulacro `backup-20261102202820`, sólo datos sintéticos:

- Migración 20, seis sellos y 24 objetos cotejados por SHA-256; PDFs de examen,
  auditoría, consentimientos y cargas cifradas de outbox incluidos.
- Manifiestos lógicos de todas las tablas idénticos **antes** de invalidar las
  sesiones de la copia. Ningún cambio en la base fuente para esa invalidación.
- Recuperación desde archivos cifrados autenticados: **32,589 segundos**.
- Objetivos provisionales: RTO 1800 s, RPO 86400 s. El ensayo cumple ese objetivo
  local de tiempo; no mide un RPO diario ni el tiempo de recuperación productivo.
- Conserva `.enc`, manifiestos y evidencia; retiró sólo los nuevos temporales en
  claro. Respaldos históricos no cifrados siguen privados y requieren revisión.

La evidencia detallada queda en `apps/backend/data/synthetic-demo/verified-v3/`,
ignorada por Git. `quality.json`, `privacy-evidence.json`, `fault-evidence.json` y
los manifiestos registran la corrida; `recovery-timing.json` incluye commit,
árbol de trabajo con cambios y hash del binario. Los números no son evidencia
inmutable de versiones futuras.

## Qué sigue pendiente

1. **XT-03:** contrato de máquina y barrera exhaustiva de compatibilidad de todos
   los DTO/rutas. Hay definición de formulario compartida y contratos críticos;
   no se declara cobertura total ni se oculta el criterio pendiente.
2. **Producción:** aprobar retención/excepciones legales, reglas institucionales,
   RPO/RTO y responsables; habilitar agenda, copia externa y custodia independiente
   de claves; desplegar y probar colector/canal de alertas; ejecutar CI remoto.
3. **Cutover de descargas:** cerrar emisión antigua y rotar credenciales de firma
   o esperar el TTL fuera de tráfico antes de prometer revocación de todos los
   enlaces previos. No se ejecutó ninguna rotación en producción.
4. **Límite de revocación:** bloquea nuevas peticiones y vuelve a comprobar antes
   de responder; no retira bytes ya entregados ni una transferencia que ya empezó
   a enviarlos. Copias locales de PDFs/exportaciones quedan bajo custodia humana.
5. `-race` no se certifica en este Windows sin compilador CGO; el workflow Linux
   lo solicita. SMTP sigue siendo al menos una vez, no entrega física exactamente
   una vez ni confirmación de llegada al buzón.

## Repetir sin tocar datos de trabajo

```powershell
./scripts/backend-demo/Start-Demo.ps1
./scripts/backend-demo/Test-Demo.ps1
./scripts/cross-cutting/Test-Privacy.ps1
./scripts/backend-demo/Invoke-Demo.ps1
./scripts/backend-demo/Invoke-Demo.ps1 -Faults
./scripts/backend-demo/Backup-Restore-Demo.ps1
```

Antes de repetir el último paso, liberar el puerto 19010 deteniendo sólo el
contenedor restaurado identificado y etiquetado en su evidencia; no borrar
volúmenes, copias ni bases anteriores.

**Referencias:** [tareas](../tasks/cross-cutting-tasks.md) ·
[contrato](../api-contracts/API_CONTRACT.md) ·
[retención](../privacy/RETENTION_POLICY.md) ·
[operación](OPERATIONS_RUNBOOK.md) · [recuperación](RECOVERY_RUNBOOK.md).
