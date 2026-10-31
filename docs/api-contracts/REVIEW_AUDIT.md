# Auditoría y revisión administrativa — T8/T9

Implementación backend del 2026-10-31. Fuente funcional: `docs/requerimiento.txt`,
EF-1 (trazabilidad) y EF-13.3 (conteos, históricos, certificados y matriz).
La SPA correspondiente sigue en FT7/FT8; aprobar/publicar fases corresponde a T18.

## 1. Historial privado

| Método | Ruta | Acceso |
|---|---|---|
| GET | `/api/v1/submissions/{id}/events?limit=20&cursor=…` | Sólo titular, incluso para un borrador retirado |
| GET | `/api/v1/admin/submissions/{id}/events?limit=20&cursor=…` | Analista/administrador, sólo expediente enviado |

Respuesta `CursorPage<Entry>`: `items`, `limit`, `hasMore`, `nextCursor` opcional.
Cada entrada tiene `id`, `actorId` opcional, `rol`, `accion`, `recursoId`,
`tipoRecurso`, `resultado`, `createdAt`. No devuelve payloads, nombres, cédulas,
correos, operadores, motivos libres, claves de objetos ni URLs firmadas.

Se unen guardados/envíos/bajas, altas y bajas de documentos, accesos administrativos,
eventos de negocio y cambios de rol de la cuenta titular. Estos últimos son
eventos de cuenta, no cambios de estado del EF-12: la tabla histórica no guardaba
el rol del actor, por lo que se entrega `rol=no_registrado` sin inventarlo.
Los eventos automáticos/operativos sin actor identificable omiten `actorId`.
Los nuevos cambios de expediente/adjuntos capturan el rol vigente al escribir.
Los eventos anteriores a T8 mantienen `rol=no_registrado`; no se reconstruye
un rol histórico a partir de la cuenta actual.
La consulta administrativa registra a su vez `historial.consultado` después de
leer: aparecerá al refrescar, no altera las páginas ya recorridas.

El cursor queda ligado al expediente y ordena por `(created_at DESC, id DESC)`.
Límite por defecto 20, máximo 100; sin `OFFSET` ni `count(*)` de paginación.
Parámetros desconocidos, repetidos, vacíos o mal codificados se rechazan con 400.
Un cursor de otro expediente se rechaza. Las rutas propias no se convierten en
administrativas por tener un rol de personal interno.

### Conservación e integración

La migración `00009` elimina las cascadas del historial: borrar un borrador/cuenta
no elimina sus eventos. No reconstruye eventos ya perdidos antes de migrar.
Una cuenta con expedientes sellados no puede borrarse mediante cascada ordinaria.
Los cuatro historiales rechazan UPDATE/DELETE/TRUNCATE.
No hay purga automática ni un plazo legal inventado. Es una política técnica
conservadora provisional; T28 y la institución deben aprobar el período, la
minimización, el archivo y el procedimiento autorizado de eliminación.
Las protecciones se aplican a DML ordinario; un propietario/superusuario de base
puede alterar triggers. Usar credenciales de ejecución con privilegios mínimos.

`audit.WorkflowRecorder.RecordWorkflow(ctx, event)` es el puerto de T17/T18 y
T20–T27. Exige el contexto creado con `db.WithTransaction`; valida acciones y
metadatos cerrados, y el error debe propagarse para revertir cambio y evento.
No existe un endpoint público para fabricar eventos. Se admiten:

`fase.publicada`, `fase.habilitada`, `fase.cerrada`, `elegibilidad.verificada`,
`meritos.revisados`, `meritos.calificados`, `evaluacion.iniciada`,
`evaluacion.finalizada`, `reconsideracion.presentada`, `reconsideracion.resuelta`,
`resultado.publicado`.

El puerto y su tabla están disponibles, pero no certifican operaciones de módulos
que todavía no existen. Su instrumentación real se realizará al implementar esas
tareas. Los cambios de rol actuales conservan su trigger transaccional de T6.

## 2. Nómina y conteos por período

| Método | Ruta | Respuesta |
|---|---|---|
| GET | `/api/v1/admin/competitions?limit=20&cursor=…` | `CursorPage<Competition>` de períodos, incluidos cerrados |
| GET | `/api/v1/admin/submissions?convocatoriaId=UUID&sector=afiliados&status=submitted&limit=20&cursor=…` | `CursorPage<Applicant>` |
| GET | `/api/v1/admin/submissions/stats?convocatoriaId=UUID&sector=afiliados` | Filtros, `consultadoEn`, `actualizarEnSegundos: 30`, conteos |
| GET | `/api/v1/admin/submissions/export.xlsx?convocatoriaId=UUID&sector=afiliados` | Libro Excel privado |
| GET | `/api/v1/admin/submissions/export.pdf?convocatoriaId=UUID&sector=afiliados` | PDF privado |

Todas exigen sesión y `revision.consultar` (analista o administrador), sin acceso
público. La auditoría debe confirmarse antes de entregar datos o bytes. Un fallo
de registro produce 500 sin contenido ni archivo. `Cache-Control: no-store` en
respuestas y errores de estas operaciones.

### Filtros y legado

- Debe indicarse **exactamente una** selección: `convocatoriaId=UUID` o
  `sinConvocatoria=true`. La segunda recupera únicamente filas sin período.
- `sector` opcional: `afiliados`, `jubilados`, `empleadores`. En el EF-12 vigente
  el cargo de representación se expresa por sector, no por un cargo laboral.
- `status` opcional: sólo `submitted`. Los borradores y `in_review` no se exponen
  a personal revisor; esto conserva la política de T6.
- Los cursores quedan ligados a todos los filtros; orden estable por
  `(created_at DESC, id DESC)`. Ninguna página mezcla períodos implícitamente.
- `limit` y `cursor` sólo se admiten en los listados. No en stats/exportaciones.
  Al exportar se usa la selección completa, no la página visible.
- Un período inexistente devuelve 404. Un resultado vacío válido devuelve
  `items: []` / `conteos: []`; una caída de PostgreSQL no se presenta como cero.

`Applicant` devuelve código, nombre/apellidos del sello, sector, estado, fecha de
envío, hash, formación resumida, experiencia por ID con años/meses/días y documento
referenciado, cantidad de publicaciones y horas de capacitación. No consulta el
perfil actual ni expone contactos/cédula en el listado. Se verifica el hash del
JSON canónico antes de proyectar o exportar; un sello alterado/ininterpretable
produce `409 expediente_no_integro`, no se omite silenciosamente de la nómina.

Se conserva la duración guardada al enviar. Para sellos antiguos sin duración,
se calcula con las fechas históricas; una experiencia entonces vigente termina
en la fecha del envío en Ecuador, nunca en la fecha actual de consulta. No se
suman períodos solapados ni se deduce un puntaje de méritos.

Cada conteo agrupa por sector y estado y contiene `postulantes` (personas distintas)
y `postulaciones` (expedientes). No cuenta borradores ni expedientes sin sello.
La consulta se ejecuta en PostgreSQL en cada petición. La **propuesta técnica**
para FT7 es polling cada 30 s mientras el dashboard esté visible y en línea;
retomar/refrescar al recuperar foco y detenerlo al salir. No es SSE/WebSocket ni
una garantía de latencia institucional: aprobar esa frecuencia con el requirente.

### Registro de períodos y relación con T18

`00010` crea `competitions` con UUIDv7, código único, título, apertura, cierre y
fecha de creación. No genera una convocatoria ficticia ni migra automáticamente
expedientes existentes. `POST /api/v1/submissions` admite `convocatoriaId` opcional
durante la transición; si se indica, el período debe existir y estar abierto.
El ID de período no se puede cambiar después de crear el expediente. No se deriva
del JSON ni se acepta en actualizaciones. La respuesta EF-12 incluye ese ID.

T18 debe crear/configurar períodos reales y hacer obligatoria su selección donde
corresponda, además de gestionar aprobación, publicación, fases y calendario.
No hay un endpoint de escritura de convocatorias añadido como atajo en T9.
Los expedientes anteriores permanecen en la selección explícita de legado.

El código/título/período se protege después de recibir expedientes. Los EF-12
enviados y sus metadatos de certificados rechazan modificaciones/bajas, también
por SQL ordinario; se bloquean los TRUNCATE de esas tablas. El cierre impide nuevas
altas/envíos asociados. La nómina cerrada sigue accesible por su propio UUID.
El registro de períodos es infraestructura histórica, **no** una fase aprobada.

El detalle íntegro y las descargas usan las rutas existentes de T6/T4:
`GET /admin/submissions/{id}`, `GET /admin/submissions/{id}/documents`,
`GET /admin/documents/{id}/download` (todas bajo `/api/v1`). El enlace de descarga
vence en cinco minutos y va directo al almacenamiento; la API no retransmite PDF.
El evento acredita emisión del enlace, no la descarga efectiva. El respaldo físico
de los objetos sigue requiriendo la política de backups de T12.

## 3. Propuesta de matriz `EF13-TECNICA-v1`

El RF exige Excel/PDF pero no fija columnas. Se propone a Talento Humano:

1. Código de expediente.
2. Nombres y apellidos del formulario sellado.
3. Cargo de representación / sector.
4. Fecha de envío UTC, etiquetada expresamente.
5. Formación declarada: nivel y título.
6. Experiencias individualizadas por ID, tipo, cargo y duración en años/meses/días.
7. Cantidad de publicaciones declaradas.
8. Horas de capacitación declaradas.

Convocatoria, estado `submitted`, filtros, versión, fecha de consulta y cantidad
de expedientes constan en la cabecera. No se incluyen cédula, correo, domicilio,
declaraciones sensibles ni URLs firmadas. No se presenta como lista de elegibles,
calificación, orden de mérito ni formato oficialmente aprobado.

Ambos formatos usan la misma proyección y filtros. XLSX separa las experiencias
en una segunda hoja (con hash de expediente para referencia); PDF repite cabecera
y divide filas largas entre páginas. El libro usa celdas de texto `inlineStr`,
sin macros/fórmulas ejecutables aun si un dato empieza por `=`. Texto XML escapado.
PDF usa fpdf y fuentes Go embebidas, sin depender de fuentes instaladas en el host.

Exportación completa en transacción `REPEATABLE READ READ ONLY`: se comprueba el
presupuesto y se carga una única instantánea. Máximos: 10 000 expedientes,
32 MiB de JSON de origen y 20 s de procesamiento. Si se superan filas/bytes o el
límite de texto de una celda Excel, responde 413 sin archivo parcial. No se
truncan nombres, experiencias ni la nómina. Para volúmenes mayores se necesita
un trabajo asíncrono posterior, no descargas paginadas disfrazadas de exportación.

## 4. Errores y pruebas

| HTTP | Código | Caso |
|---|---|---|
| 400 | `filtro_revision_invalido` | Período ausente/ambiguo, sector/estado no permitido, parámetros inválidos |
| 400 | `cursor_invalido` | Token inválido, de otro expediente/listado o con filtros distintos |
| 401/403 | Sesión / `acceso_denegado` | Falta de sesión, titularidad o permiso |
| 404 | `convocatoria_no_encontrada` | UUID de período inexistente |
| 409 | `convocatoria_no_disponible` | Alta/envío asociado a período no admitido |
| 409 | `expediente_no_integro` | Sello o estructura histórica incoherente |
| 413 | `exportacion_demasiado_grande` | Presupuesto excedido, sin archivo parcial |
| 500 | `historial_no_disponible` / `revision_no_disponible` | Persistencia/auditoría/generación fallida |

Tests unitarios/HTTP cubren titularidad, roles, fallos de auditoría, cursores,
filtros, integridad, duración histórica, XML/fórmulas, PDF largo y cancelación.
La integración PostgreSQL es optativa en esquemas aislados:

```powershell
# Desde apps/backend, con una base PostgreSQL 18 EXCLUSIVAMENTE de pruebas:
$env:AUDIT_TEST_DATABASE_URL = "URL_DE_PRUEBAS"
$env:REVIEW_TEST_DATABASE_URL = "URL_DE_PRUEBAS"
go test ./internal/audit/... ./internal/review/...
```

Sin esas variables se omite la integración, no se inicia PostgreSQL ni Docker.
Las migraciones se aplican al arrancar la API según la configuración existente;
agregar archivos de migración no equivale a aplicarlas en un entorno desplegado.
