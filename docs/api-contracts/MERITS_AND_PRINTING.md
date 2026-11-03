# T19/T20: copia sellada y revisión de méritos

**Implementación técnica local (2026-10-31).** T19 genera el EF-12 imprimible y
lo adjunta a su confirmación. T20 conserva revisiones, cálculos y resoluciones
versionadas, sin modificar ese envío. No se entrega una matriz institucional:
`docs/requerimiento.txt` exige calificación, pero no fija puntajes ni umbrales.

## Recorrido administrativo

1. Crear una versión de reglas para una convocatoria y sector de T18.
2. Completar la definición y aprobarla con una referencia institucional explícita.
3. Durante la fase **`meritos`**, revisar cada respaldo, requisito e inhabilidad.
4. Calcular desde la revisión vigente y consultar el desglose y los impedimentos.
5. Un administrador publica `avanza` o `excluido`, con motivo. Sólo una publicación
   `avanza` habilita el requisito de méritos para la futura evaluación técnica.

El examen, su agenda y sus intentos siguen en T25–T27; este contrato no los crea.

## T19: descarga y correo

| GET | Acceso |
|---|---|
| `/api/v1/submissions/{id}/print.pdf` | Sesión y titularidad, incluso para administradores |
| `/api/v1/admin/submissions/{id}/print.pdf` | Analista/administrador; lectura auditada |

Devuelven `application/pdf`, `Content-Disposition: attachment`, `no-store` y
`nosniff`. El nombre del archivo contiene sólo el UUID. Un borrador o un hash
inconsistente produce **409**; superar límites técnicos, **413**. No se entregan
bytes si falla la auditoría administrativa. Sin generador disponible: **503**.

La copia incluye código, convocatoria de origen (o legado explícito), hash,
fecha UTC del envío, nueve secciones, todos los registros y datos conservados,
consentimientos y área de firma manuscrita. Los literales y su versión proceden
del JSON sellado, no de la versión actual. Lo ausente se identifica como ausente:
no se completa desde el perfil. No exige firma digital dibujada. Una firma legada
PNG/JPEG válida puede representarse; no se descargan imágenes remotas. Los glifos
no disponibles se muestran como `[U+XXXX]`, no como caracteres en blanco.

El límite es 8 MiB de JSON, 200 páginas y 16 MiB de PDF. El mismo snapshot y la
versión `ef12-print-v1` producen bytes reproducibles dentro de esta versión del
generador. Los metadatos temporales no usan el reloj de la descarga.

Al enviar, la transacción registra un sobre con el snapshot **cifrado en el
outbox de T15**, junto al sello. El worker genera el PDF después del commit y
lo entrega como adjunto MIME al correo registrado. No vuelve a consultar el
borrador ni el perfil. Un error de preparación queda como
`document_preparation_error`, reintentable con la política ya existente; un fallo
SMTP tampoco deshace el sello. La carga cifrada se elimina al terminar su ciclo.
`smtp_accepted` no certifica llegada al buzón. Los trabajos anteriores sin sobre
conservan su correo JSON original: no se reconstruye una copia histórica ficticia.

Los cinco literales siguen mostrándose al guardar mediante la definición y
normalización existentes; guardar y enviar siguen siendo acciones distintas.

## T20: rutas y permisos

Todas requieren sesión. Prefijo administrativo: **`/api/v1/admin/merits`**.

| Método y sufijo | Permiso | Resultado |
|---|---|---|
| `POST /rules` | Analista/administrador | 201, nueva versión inmutable en borrador |
| `GET /rules/{id}` | Analista/administrador | Definición y metadatos; lectura auditada |
| `GET /competitions/{id}/sectors/{sector}/rules` | Analista/administrador | Versiones por cursor descendente, lectura auditada |
| `POST /rules/{id}/approve` | Administrador | Validación estricta y activación explícita |
| `GET /submissions/{id}` | Analista/administrador | Reglas activas y últimas revisión/cálculo/publicación; auditado |
| `POST /submissions/{id}/review` | Analista/administrador | 201, nueva revisión |
| `POST /submissions/{id}/calculate` | Analista/administrador | 201, nuevo cálculo reproducible |
| `POST /submissions/{id}/publish` | Administrador | 201, nueva resolución explícita |

El snapshot administrativo y los PDF de respaldo se consultan mediante las rutas
existentes de T6/T4. Nunca se devuelven claves de almacenamiento desde méritos.
La escritura comprueba el rol vigente bajo bloqueo y confirma evidencia más
evento de historial en una sola transacción; sin auditoría revierte todo.

Para el titular: `GET /api/v1/merits/submissions/{id}` devuelve
`{ "publicado": false, "habilitadoFaseTecnica": false, "resultado": null }`
hasta existir una resolución. Luego `resultado` contiene la publicación, su
desglose, impedimentos, hashes, versión, responsable y motivo. No expone revisiones
ni reglas en borrador. No hay consulta pública ni de expedientes ajenos.

### Reglas: definición sin valores oficiales por defecto

Crear: `{ "convocatoriaId": "<uuid>", "sector": "afiliados", "definicion": {...} }`.
Sectores: `afiliados`, `jubilados`, `empleadores`, habilitados en la convocatoria.
Cada creación asigna una versión consecutiva; para corregir una definición se crea
otra versión. Aprobar: `{ "referenciaAprobacion": "<acta o resolución real>" }`.
El administrador debe aportar una referencia, no se genera ni verifica su
autenticidad documental automáticamente. Se conservan aprobador y fecha.

Una definición aprobable exige:

| Campo | Contrato |
|---|---|
| `criterios` | 1–50 objetos con `codigo`, `nombre`, `metrica`, `filtro` opcional, `tamanoUnidad`, `centipuntosPorUnidad`, `maximoCentipuntos`, `pesoPuntosBase`, `requiereEvidencia` |
| `requisitos` | 1–100 pares `codigo`/`nombre` del checklist institucional |
| `inhabilidades` | Las 13 entradas únicas `pregunta: q1…q13` y `respuestaAdmitida: Sí/No`; **q1 exige Sí** por su sentido positivo |
| `maximoTotalCentipuntos`, `umbralCentipuntos` | Enteros explícitos, umbral entre cero y máximo; sin valores implícitos |
| `redondeo` | `inferior` o `mas_cercano` (mitades hacia arriba) |
| `experiencias` | `unir_intervalos` o `sumar_registros`; nunca inferido |
| `computoDias` | `dias_transcurridos` o `dias_inclusivos` |
| `desempates` | Códigos únicos de criterios en orden y cierre `resolucion_manual` |

Métricas soportadas: `education_count` (filtro de nivel EF-12), `experience_days`
(General/Dirección), `publication_count` (tipo EF-12), `training_hours` (tipo de
evento). Filtro vacío incluye todos los tipos de esa métrica. Una futura matriz
que requiera otra fórmula necesita ampliar y verificar el motor; no se fuerza a
una aproximación. No hay puntajes ingresados por el navegador.

Una unidad monetaria no interviene: **100 centipuntos = 1 punto** y los pesos
suman **10000 puntos base**. Para cada criterio se calcula exactamente
`redondear(min(unidades × centipuntosPorUnidad / tamanoUnidad, máximo) × peso / 10000)`.
Se redondea una vez por criterio, después de aplicar el máximo y el peso. El
máximo total debe coincidir con la suma de los máximos ponderados y redondeados.
El motor `meritos-v1` usa enteros/racionales, no flotantes.

El cálculo cuenta sólo registros con documento revisado `valido`, vinculado por
sección, registro, documento y SHA-256 al envío. Las experiencias actuales terminan
en la fecha del envío en America/Guayaquil, nunca en el día de consulta. Se ignora
la duración numérica declarada y se computan días desde las fechas selladas.

Los desempates devuelven el vector de valores según el orden aprobado, **no un
ranking ni un ganador automático**. Una igualdad restante exige resolución manual.

### Revisión, cálculo y publicación

Revisión (`versionAnterior: 0` para la primera):

```json
{
  "versionAnterior": 0,
  "reglasId": "<uuid-versión-aprobada>",
  "documentos": [
    {"seccion": 5, "registroId": "<id-sellado>", "documentoId": "<uuid-sellado>", "resultado": "valido", "observacion": ""}
  ],
  "requisitos": [{"codigo": "<codigo-aprobado>", "cumple": true, "observacion": ""}],
  "inhabilidades": [{"pregunta": "q1", "verificada": true, "observacion": ""}],
  "motivo": "<fundamento de la revisión>"
}
```

El ejemplo abreviado debe completarse con **todos** los registros de secciones
4–8, todos los requisitos aprobados y las **13** preguntas. Para sección 4 el
registro de organización es `registroId: ""`. Un respaldo rechazado usa
`invalido` con observación; un registro sin documento usa `no_presentado`, ID vacío
y observación. Un requisito negativo o una pregunta no verificada exige observación.
Duplicados, IDs ajenos, SHA incorrecto y revisión incompleta se rechazan.

Calcular: `{ "revisionId": "<uuid-vigente>", "versionAnterior": 0, "motivo": "..." }`.
Se conserva `versionMotor`, hashes del envío/reglas, revisión fuente, unidades,
puntaje por criterio, total, máximo, umbral e impedimentos. Un requisito no cumplido,
pregunta no verificada/respuesta no admitida, evidencia requerida ausente o nota
inferior al umbral impide publicar `avanza`.

Publicar: `{ "calculoId": "<uuid-vigente>", "versionAnterior": 0,
"decision": "avanza", "motivo": "..." }`. También admite `excluido` con motivo.
Se verifica nuevamente el cálculo contra revisión, reglas y motor vigentes.
Una revisión nueva o recálculo **no sustituye** la resolución publicada.
Para corregirla se envían además `sustituyeId` y la versión anterior de la
publicación, con motivo explícito. La resolución previa permanece inmutable.

Las escrituras requieren convocatoria aprobada/publicada y fase `meritos`
publicada/habilitada dentro de sus fechas y las del período, intervalo
`[apertura,cierre)`. **No requieren reabrir EF-12.** El legado sin convocatoria
permanece de sólo lectura, sin reasignación inventada.

### Paginación y errores

La lista de reglas admite únicamente `limit` (máximo 100) y `cursor` opaco, ligado
a convocatoria/sector. Devuelve `items`, `hasMore`, `nextCursor`; no usa OFFSET.
Los cuerpos son JSON estrictos, sin campos desconocidos ni datos adicionales,
máximo 128 KiB (definición de reglas, 64 KiB).

| HTTP | Código estable de méritos |
|---:|---|
| 401/403 | Sesión/permisos o `meritos_prohibidos` |
| 404 | `meritos_no_encontrados` |
| 409 | `reglas_meritos_pendientes`, `version_meritos_conflicto`, `fase_meritos_cerrada` |
| 422 | `meritos_invalidos`, `evidencia_meritos_invalida` |
| 503 | `meritos_no_disponibles` (incluye fallo de auditoría; sin datos parciales) |

Ante conflicto, volver a consultar; no repetir una publicación ciegamente.
Un nuevo envío de POST no garantiza idempotencia: las versiones esperadas evitan
duplicar revisiones/cálculos/resoluciones. Las aprobaciones repetidas devuelven 409.

## Persistencia y operación

La migración **00013** incorpora reglas, cabeceras y registros append-only de
revisiones, cálculos y publicaciones. Las claves compuestas impiden enlazar
cabeceras/resultados de expedientes distintos. Reglas aprobadas e historiales
rechazan UPDATE/DELETE/TRUNCATE. No tiene semilla de puntajes ni modifica datos
históricos. Su `Down` se niega a eliminar evidencias si ya existen datos; requiere
un plan de reversión específico, no vaciar las tablas.

Aplicar con el runner de migraciones habitual antes de desplegar la API. **Esta
sesión sólo migró esquemas de una base temporal, no la base de la aplicación.**
FT6/FT19 deben consumir las rutas; no se implementó aquí su interfaz. Las reglas
oficiales, su aprobación y la verificación operativa institucional siguen pendientes.
