# Banco, examen teórico y privacidad — T25–T28

**Contrato técnico local.** Implementa el productor real T27 que consume T21,
no un ingreso manual de notas. Los parámetros, banco y políticas legales no se
precargan: deben ser aportados y aprobados por la institución. El frontend no
se modifica en esta entrega.

## Secuencia de integración

1. Administrador publica política de privacidad por convocatoria. El postulante
   debe ver su texto/información de tratamiento y enviar su ID/hash al aceptar
   los cinco puntos originales del EF-12. Sin política vigente no hay nuevo envío.
2. Administrador carga una versión completa del banco y configura escala, cantidad,
   duración, horario y política explícita de intento/unicidad.
3. Publicar méritos T20 y habilitar individualmente al postulante con esa fuente
   exacta y la configuración aprobada.
4. Postulante inicia; el servidor devuelve un intento persistido, su versión,
   vencimiento y reloj actual. Recupera una pregunta por índice y guarda respuestas.
5. Finalizar o vencer usa el mismo cierre transaccional: nota, PDF, expediente,
   auditoría y correo T23. T21/T24 ya pueden consumir esta fuente teórica legítima.

JSON exitoso directo; errores en `error.codigo`. `credentials: include`, protección
CSRF y permisos existentes. Lecturas privadas con `Cache-Control: no-store`.
No almacenar matriz ni reportes en caché pública o persistente del navegador.

## T25 · Banco de preguntas

Sólo **administrador**, con sesión y permiso `preguntas.gestionar`:

| Método/ruta | Uso |
|---|---|
| `POST /api/v1/admin/question-bank/competitions/{period}/sectors/{sector}/versions` | Cargar/reemplazar mediante nueva versión completa |
| `GET /api/v1/admin/question-bank/competitions/{period}/sectors/{sector}/versions` | Metadatos históricos, cursor |
| `GET /api/v1/admin/question-bank/versions/{id}` | Banco y matriz correcta, lectura auditada |
| `POST /api/v1/admin/question-bank/versions/{id}/configurations` | Aprobar parámetros y horario |
| `POST /api/v1/admin/question-bank/submissions/{id}/enable` | Habilitación individual |

El analista no ve la matriz. Las rutas del postulante no devuelven `correcta`,
`matriz`, `hashBanco` ni el banco completo, tampoco tras terminar el examen.
Las preguntas/opciones son texto; el frontend debe escaparlas, no renderizar HTML.

Carga (ejemplo estructural, no banco institucional):

```json
{
  "versionAnterior": 0,
  "referencia": "<documento institucional del banco>",
  "preguntas": [
    {
      "id": "identificador-estable",
      "enunciado": "<pregunta aprobada>",
      "opciones": [
        {"id": "a", "texto": "<opción>"},
        {"id": "b", "texto": "<opción>"}
      ],
      "correcta": "a"
    }
  ]
}
```

El servidor asigna UUIDv7, versión, hash y responsable. No hay edición destructiva:
una corrección envía la versión anterior y una nueva definición completa. IDs de
pregunta y opciones únicos dentro de su alcance, una respuesta correcta existente,
2–8 opciones distintas por pregunta, 1–500 preguntas; cuerpo de hasta 2 MiB.
Son límites técnicos, no parámetros académicos oficiales.

Las listas usan `limit` (máximo 100) y `cursor` ligado a convocatoria/sector/tipo;
respuesta `items`, `hasMore`, `nextCursor`, `limit`.

## T26 · Configuración e inicio

`sector`: `afiliados`, `jubilados` o `empleadores`. El banco pertenece al mismo
período/sector que la postulación. No hay acceso público al módulo de evaluación.

Cuerpo de configuración, todos sus valores relevantes explícitos:

```text
cantidadPreguntas: entero entre 1 y 100, no mayor que el banco
 duracionSegundos: entero entre 1 y 14400
 maximo: entero entre 1 y 1000000000
 umbral: entero entre 0 y maximo
 redondeo: inferior | mas_cercano
 politicaIntentos: un_intento
 politicaUnicidad: orden_preguntas_unico_rechazar_agotamiento
 referenciaAprobacion: acta institucional real
 abreEn, cierraEn: instantes ISO con offset
```

Los límites no son valores por defecto: se rechazan cantidades, duración, escala,
umbral, referencia o política ausentes/incompatibles. Actualmente sólo se admite
**un intento por postulación**; no se ofrece un selector ficticio de múltiples
intentos. Horarios se normalizan a UTC y precisión de microsegundos PostgreSQL;
la UI puede presentarlos en America/Guayaquil.

Cada banco admite una configuración inmutable. Cambiar parámetros exige otra
versión del banco y aprobación: no modifica ni extiende intentos anteriores.

### Unicidad y agotamiento

El motor usa aleatoriedad criptográfica para seleccionar preguntas sin repetición
y permutar sus opciones. La unicidad garantizada es el **orden de IDs de preguntas
entre los intentos de la misma configuración congelada**, persistido con restricción
única en PostgreSQL. No es una promesa de infinitas variantes ni de contenido
universalmente distinto entre diferentes bancos/configuraciones.

La capacidad es `n! / (n-k)!`; `k=1` tiene sólo `n` órdenes posibles. Colisiones
se vuelven a sortear y, si es necesario, se busca un orden libre en el espacio
finito. Si se agotó, devuelve `asignaciones_examen_agotadas`: no duplica examen ni
abre otro intento silenciosamente. Ampliar el banco requiere decisión institucional.

### Habilitación

`POST .../submissions/{id}/enable`:

```json
{"configuracionId":"<UUID>","resultadoMeritosId":"<UUID publicación vigente T20>"}
```

Exige administrador vigente bloqueado, resultado `avanza`, sello y sector correctos.
Repetir la misma habilitación no la duplica; otra distinta da 409. El calendario
T18 y el horario deben permitir el examen al iniciar. No basta la bandera de UI.
La respuesta identifica `fase: evaluacion_tecnica`. Retirar/reconfigurar un intento
individual no es una operación disponible: se conserva el historial de un intento.

### Rutas del postulante

| Método/ruta | Cuerpo/resultado |
|---|---|
| `POST /api/v1/exams/submissions/{id}/start` | `{}`; inicia o recupera el mismo intento |
| `GET /api/v1/exams/submissions/{id}` | Estado/reloj/resultado definitivo si terminó |
| `GET /api/v1/exams/submissions/{id}/questions/{index}` | Una pregunta, opciones y selección guardada; índice empieza en 1 |
| `PUT /api/v1/exams/submissions/{id}/answers` | Guardado individual, control de versión |
| `POST /api/v1/exams/submissions/{id}/finish` | `{}`; cierre idempotente, sin puntaje del cliente |
| `GET /api/v1/exams/submissions/{id}/report.pdf` | PDF definitivo almacenado, titularidad y privacidad |

Cualquier inicio concurrente, recarga, reconexión o nueva sesión conserva intento,
orden, respuestas y vencimiento. La asignación y el plazo se confirman **antes**
de responder al navegador. Sin habilitación devuelve 404; méritos no vigentes,
ventana cerrada o fase no habilitada bloquean el inicio/guardado.

El inicio se rechaza si no queda el tiempo completo de duración dentro de la
ventana configurada y fase T18; no se recorta una prueba sin aprobación.

Respuesta de estado: `id`, `postulacionId`, `version`, `estado` (`active`/`finished`),
`iniciadaEn`, `venceEn`, `servidorEn`, `segundosRestantes`, `cantidadPreguntas`,
`respondidas` y `resultado` sólo cuando existe. No contiene matriz o respuestas.

Guardar:

```json
{"versionAnterior":1,"preguntaId":"<ID asignado>","opcionId":"<ID asignado>"}
```

Se comprueba pertenencia de pregunta/opción y versión bajo bloqueo. Repetir una
selección que ya está guardada no crea otra versión. Un cambio con versión
obsoleta da 409; volver a consultar y no sobrescribir a ciegas.
El frontend puede navegar por índices, pero nunca recibe el banco completo.

**El tiempo es del servidor después de adquirir bloqueos.** Una petición iniciada
antes del límite pero procesada después no guarda una respuesta tardía. La UI debe
sincronizar su contador con `servidorEn` y tratar `venceEn` como límite exclusivo.

Bloqueos de copiar/pegar y botón Atrás son parte del frontend FT26, no controles
que el backend pueda certificar contra captura de pantalla o un cliente modificado.
Esta entrega no afirma implementar esos bloqueos visuales.

## T27 · Cierre, calificación y PDF

El worker revisa vencimientos cada segundo y recupera intentos tras reinicios.
Lectura/inicio/guardado tardío también pueden confirmar el vencimiento; una respuesta
tardía devuelve `examen_cerrado` **después de confirmar el cierre**, no se pierde la
nota por revertirla con el error. El worker y cierre manual comparten el bloqueo.

Sólo entra lo guardado antes del límite. El motor `examen-v1` calcula:

`puntaje = redondear(maximo × aciertos / cantidadPreguntas)`

Pesos iguales por pregunta, no respondida/incorrecta aporta cero, sin penalización
inventada. Ese algoritmo y su referencia deben aprobarse con la escala; otra
ponderación requiere ampliar el contrato. `mas_cercano` redondea mitades arriba;
aritmética entera exacta. El servidor compara con su matriz congelada, nunca con
una respuesta correcta o nota enviada por HTTP.

El commit incluye estado final, respuestas, aciertos, resultado, PDF, SHA-256,
evento auditable y correo T23. `RecordFinalTheory` se invoca dentro de esa misma
transacción, no mediante un ingreso HTTP de nota. Una carrera produce una sola
nota/PDF/actuación y un aviso lógico. Una caída/fallo de PDF, auditoría o cola
revierte todo y el cierre vuelve a intentarse; nunca confirma una nota sin su aviso.

El PDF `examen-reporte-v1` se conserva **en PostgreSQL**, hasta 4 MiB, inmutable,
y viaja adjunto en la cola cifrada. Esta decisión evita una ventana entre escritura
S3 y confirmación de la nota. No consume la cuota de adjuntos declarados del
postulante. Se conservan banco, reglas, sello, tiempos, fórmula, aciertos y nota;
no publica la matriz correcta. Los bytes son reproducibles para el mismo cierre.

Administración: analista/administrador pueden consultar, con auditoría:

- `GET /api/v1/admin/grades/submissions/{id}/exam`
- `GET /api/v1/admin/grades/submissions/{id}/exam/report.pdf`

El PDF responde `Content-Type: application/pdf`, `Cache-Control: no-store` y
`X-Content-SHA256`. No hay URL pública de reportes. El correo se envía al registrado
mediante T23, con reintentos y Message-ID estable; `smtp_accepted` no es entrega
al buzón. T28 puede impedir enviar o descargar sin cambiar la nota histórica.

## T28 · Política versionada y aceptación sellada

No se incluye texto jurídico, plazo legal ni política institucional de ejemplo
en producción. El responsable debe aportar texto, información de tratamiento,
referencia de aprobación y tiempos por canal. No confundir una opción configurable
con aprobación legal automática.

### Rutas

| Método/ruta | Acceso |
|---|---|
| `GET /api/v1/public/privacy/competitions/{id}` | Política vigente y revisión de control, sin sesión |
| `GET /api/v1/public/privacy/policies/{id}` | Versión documental histórica exacta |
| `POST /api/v1/admin/privacy/competitions/{id}/policies` | Administrador publica nueva versión aprobada |
| `GET /api/v1/admin/privacy/competitions/{id}/policies` | Administrador, historial paginado/auditado |
| `POST /api/v1/admin/privacy/competitions/{id}/restriction` | Administrador, restricción/restitución auditada |
| `GET /api/v1/admin/privacy/competitions/{id}/retention-review` | Administrador, revisiones vencidas paginadas/auditadas |

Publicar: `versionAnterior` de control y `definicion`:

- `texto`, `informacionTratamiento`, `referenciaAprobacion`: textos institucionales
  obligatorios. Transportados como texto, no HTML autorizado.
- `diasAcceso`, `diasDocumentos`, `diasReportes`, `diasPublicacionNombres`,
  `diasCorreo`, `diasRevisionConservacion`: enteros explícitos 1–36500; límites
  técnicos, no plazos por defecto. Días transcurridos desde aceptación del sello,
  instantes con cierre exclusivo; no se infieren feriados ni términos legales.
- `permiteExportacion`, `permiteCorreo`, `permiteNombresPublicos`: permisos
  explícitos; ausentes se interpretan como false, nunca como autorización.
- `disposicionHistoricos`: única opción soportada
  `revision_humana_sin_borrado_automatico`.

Respuesta: `id`, `convocatoriaId`, versión documental `version`, `hash`,
`definicion`, `aprobadaPor`, `publicadaEn`. La consulta vigente también devuelve
`versionControl` y `restringido`. Restricciones incrementan el control, no editan
el documento aceptado: usar siempre `versionControl` al modificar.

Al enviar EF-12 se agregan a su `consentimiento`, **junto a las cinco aceptaciones
existentes**, estos campos:

```json
{"politicaPrivacidadId":"<UUID vigente>","hashPoliticaPrivacidad":"<hash vigente>"}
```

No sustituyen los cinco literales por una aceptación genérica. El servidor
comprueba política activa no restringida, hash exacto, cinco aceptaciones y
coincidencia con el contenido sellado. La vinculación y fecha/revisión se confirman
en la transacción del envío. No hay una ruta para fabricar aceptación retroactiva
sobre expedientes ya sellados. Otra política posterior no cambia el consentimiento
anterior; la restricción vigente de la convocatoria sí bloquea las salidas.

**Cambio de integración importante:** la composición real exige esta aceptación
para nuevos envíos. No se debe inventar una política para mantener verde una
pantalla vieja. El frontend FT28 debe mostrar la política y remitir ID/hash.
Si la versión cambió durante la edición, recibe 409 y debe pedir aceptación de
la vigente. Los borradores siguen editables; no se certifica una aceptación antes
del envío ni se presentan datos OTP como identidad civil verificada.

### Acceso, salidas y conservación

En la composición de la API, las lecturas de expediente, listados, exportaciones,
PDF y descargas temporales se controlan por la vinculación sellada y canal. Los
listados completos se rechazan si incluyen un expediente sin autorización; no se
truncan ocultando registros. La lista pública T24 revisa la política en cada
consulta, incluso histórica, además del literal de nombres de T24.

La comprobación SMTP ocurre antes de preparar/entregar adjuntos; OTP no contiene
expediente y conserva su flujo separado. Fallar privacidad no altera una nota:
el aviso queda reintentable según T23, sin enviar datos por un canal no permitido.

Para una URL de documento de cinco minutos se exige margen de cinco minutos
antes de caducar su autorización. Una URL ya emitida no es revocable de inmediato
por esta capa; puede seguir funcionando hasta su vencimiento. No se promete
revocación instantánea de capacidades S3/locales ya entregadas.

Restricción:

```json
{"versionAnterior":1,"restringido":true,"motivo":"<decisión del responsable>"}
```

Se conserva actor, fecha y motivo, sin borrar políticas, formularios, resultados,
PDF ni registros auditables. La restitución explícita usa `restringido: false`
y la revisión de control vigente. Publicar otro texto no levanta una restricción.

Al llegar `diasRevisionConservacion`, la bandeja devuelve `postulacionId`,
`politicaId`, `revisarDesde`. **No ejecuta eliminaciones ni interpreta la fecha
como autorización de purga.** El responsable debe definir/aprobar la disposición
material y las excepciones de conservación antes de implementar un borrado que
pueda afectar evidencia protegida por triggers. Esa decisión institucional y su
procedimiento de ejecución permanecen pendientes, no resueltos por código.

### Legado y operación

Expedientes previos sin vinculación verificable no reciben una aceptación
inventada: sus salidas protegidas quedan bloqueadas. La institución debe decidir
la transición del legado; no se reescribe su hash. Aplicar las migraciones no
siembra políticas o preguntas y no pone en marcha un concurso por defecto.

Los adaptadores puros usados en tests pueden inyectar puertos sin T28 para aislar
reglas; `cmd/api` conecta siempre binder, guard, sender y gate público. Agregar una
nueva salida de datos exige incorporar su canal a esos controles, no reutilizar
un puerto interno como endpoint sin autorización.

## Errores

| HTTP | Examen | Privacidad |
|---:|---|---|
| 401/403 | Sesión / `examen_prohibido` | Sesión / `privacidad_prohibida` |
| 404 | `examen_no_encontrado` | `politica_no_encontrada` |
| 409 | `version_examen_conflicto`, `examen_cerrado`, `examen_no_habilitado`, `asignaciones_examen_agotadas` | `privacidad_pendiente`, `version_privacidad_conflicto` |
| 422 | `examen_invalido` | `privacidad_invalida` |
| 503 | `examen_no_disponible` | `privacidad_no_disponible` |

Cuerpos estrictos: 8 KiB para parámetros/respuestas; 1 KiB para inicio/cierre;
2 MiB para banco y 160000 bytes para publicación de política. Campos desconocidos,
JSON adicional, notas de cliente y destinatarios arbitrarios se rechazan.

## Verificación y límites

Pruebas unitarias, HTTP/httptest y PostgreSQL 18 temporal: asignaciones finitas,
respuestas persistidas, carreras, notas y PDF reproducibles, adjunto de cola,
inmutabilidad, rol vigente, privacidad por canal, versiones y rollback de auditoría.
PDF sintético renderizado/inspeccionado con Poppler. No se afirma examen institucional,
prueba visual de FT24–FT28, validación jurídica, SMTP a buzón ni despliegue.
