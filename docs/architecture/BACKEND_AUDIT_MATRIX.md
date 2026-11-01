# Matriz de auditoría backend T17–T28

Revisión del esquema y adaptadores vigentes. La migración **00019** añade
`audit_state_events`: actor/rol, recurso/registro/postulación, acción, fase,
antes/después, motivo, fecha y UUID de solicitud. Los logs no sustituyen ese
historial. No se reconstruyen actores ni motivos de operaciones anteriores.

| Operación | Evidencia duradera / enlace | Garantía y límite |
|---|---|---|
| T17 elegibilidad/cierre | `submission_events`, evaluación conservada en postulación; `workflow_events` | Creación/guardado/cierre y evento son atómicos; conserva hash antes/después, no copia el formulario |
| T18 configuración/fases | `competition_versions`, `workflow_events` | Versiones y actor; motivo técnico explícito cuando la operación no pide motivo humano |
| T19 envío/impresión | sello/hash inmutable, `submission_events`, `administrative_access_events`, outbox | El correo falla después del commit sin deshacer el sello; no implica entrega real |
| T20 reglas/revisión/cálculo/publicación | `merit_rule_sets`, `merit_reviews`, `merit_calculations`, `merit_publications`, `workflow_events` | Versiones, responsables, motivo y fuente; fuente anterior por versión/sustitución y proyección uniforme en 00019 |
| T21 psicotécnica/consolidación | `evaluation_records`, `evaluation_commands`, `workflow_events` | Historial append-only e idempotencia; referencias a tres fuentes y sello |
| T22 reconsideración | `reconsiderations`, `evaluation_records`, `workflow_events` | Recibida/atendida/resuelta y responsables distintos; no reemplaza nota sin publicación autorizada |
| T23 notificaciones | `stage_notices`, `mail_delivery_events`, `workflow_events` | Estados/intentos/ciclos; reintento manual con actor y motivo; entrega automática identifica sistema por ausencia de actor |
| T24 listas/exportación | `result_lists`, `result_list_publications`, `administrative_access_events`, `workflow_events` | Fuentes y versiones congeladas; lecturas/exportaciones fallan cerradas si no se registra acceso |
| T25 banco/configuración | `exam_banks`, `exam_configs`, `exam_enrollments`, `workflow_events` | Definiciones inmutables, hash, actor y referencia |
| T26 respuestas | `exam_attempts`, `exam_events` | Matriz/plazo congelados y versión por cambio; 00019 registra actor de la sesión y estado/versiones antes/después de cada respuesta |
| T27 cierre/nota/PDF | `exam_events`, `exam_reports`, `evaluation_records`, outbox, `workflow_events` | Nota/PDF/auditoría/cola atómicos; reloj PostgreSQL, cierre concurrente único |
| T28 privacidad | `privacy_policies`, `privacy_decisions`, `privacy_bindings`, `workflow_events` | Texto/versiones/ID/hash aceptados y motivos de bloqueo; no aceptación retrospectiva de legados |
| Documentos / descargas | `submission_events`, `administrative_access_events`, cola `document_deletions` | Acceso administrativo confirmado antes de URL; carga y referencia atómicas; compensación fallida reintentable |

## Qué está probado

Las integraciones PostgreSQL incluyen fallo de auditoría → rollback, historial
inmutable frente a UPDATE/DELETE/TRUNCATE, revocación concurrente de permisos,
finalización concurrente de examen, publicación versionada, integridad del sello,
y negativa a liberar datos cuando falla la auditoría de lectura. Las pruebas
usan esquemas aislados y aplican todas las migraciones desde vacío.

Los historiales centrales de acceso/rol/postulación/workflow no dependen de una
cascada de cuenta/borrador que los destruya. Las tablas de resultados/evaluación
impiden cambios históricos. La cuenta de aplicación sigue necesitando permisos
mínimos operacionales: un superusuario de PostgreSQL puede alterar triggers y no
se considera protegido contra sí mismo.

## Proyección uniforme de 00019

Los triggers cubren postulaciones, documentos, elegibilidad, versiones de
convocatoria, méritos, evaluación/reconsideración, workflow, notificaciones/outbox,
listas, banco/configuración/intentos/reportes y privacidad. Un fallo al insertar
el diario aborta la misma transacción que estaba cambiando el estado.

La identidad HTTP y el UUID de solicitud viajan por el kernel compartido y se
establecen con `SET LOCAL` **dentro de la transacción**. Crear,
actualizar y eliminar una postulación también usan ese límite transaccional.
No se usa un SELECT para establecer esos metadatos: adquiriría un snapshot antes
de que exportaciones puedan pedir REPEATABLE READ. Los literales son UUID
serializados y roles de una lista cerrada. Las lecturas administrativas confirman acceso y diario antes de entregar datos.
Al terminar la transacción no queda identidad pegada a una conexión reutilizada.
Los trabajos sin solicitud/actor humano se identifican como `sistema`; los
registros explícitos sin perfil vigente conservan `no_registrado`, sin inventar
su rol histórico.

El antes/después es una **proyección de estado**, no otra copia de datos
personales: ID, versión, estado/decisión, aprobación/bloqueo, referencias, hashes y
contadores. Los cambios de contenido EF-12 se distinguen mediante SHA-256, sin
copiar cédula, nombres, correo, respuestas o ciphertext. El detalle de negocio
sigue en sus versiones originales. Los motivos humanos existentes se preservan;
si el contrato no solicita uno, se usa `operacion_tecnica:<acción>`, sin fingir una
justificación humana.

Para inserciones versionadas el estado anterior es la última proyección de la
misma fuente/recurso/fase; para UPDATE/DELETE se usa OLD. La creación tiene antes
nulo y la baja tiene después nulo. No hay claves foráneas que borren el diario por
cascada. UPDATE/DELETE/TRUNCATE están prohibidos. Una baja de borrador conserva su
trazabilidad pseudónima. El diario se inspecciona por SQL administrativo, no se
expone su payload en las rutas del postulante.

La prueba `TestStateJournalCapturesActorBeforeAfterAndRollsBackOnFailure` confirma
atribución/correlación, proyecciones diferentes sin contenido privado, fallo de
inserción → rollback, append-only y conservación tras baja. `audit-evidence.json`
comprueba la cobertura de tablas del recorrido HTTP de demostración.

Para datos previos a 00019 no se inventa un diario retrospectivo: se conservan las
fuentes e historiales anteriores. La fecha de instalación es el inicio verificable
de la proyección uniforme. La cuenta superusuario de la demo no es una defensa
contra alteración administrativa de triggers; producción necesita privilegios
mínimos y controles operativos adicionales.
