# Notificaciones de etapa y listas de elegibles — T23/T24

**Entrega técnica local, 2026-10-31.** Completa el aviso de resultados de méritos,
consulta de entregas y reintentos autorizados; añade listas versionadas sobre
fuentes definitivas. No configura valores oficiales, no implementa el examen T27
ni sustituye la política institucional de privacidad T28.

## Recorrido recomendado

1. Publicar los resultados mediante T20/T21/T27; sus commits encolan el aviso.
2. Consultar entregas del expediente; ante un fallo definitivo, el administrador
   puede solicitar reintento con motivo y contador vigente, antes del vencimiento.
3. Con postulación cerrada, reconsideraciones terminadas/resueltas y resultados
   habilitados, generar una lista interna con orden y referencia institucional.
4. Revisar entradas y CSV. Publicar con versión anterior y referencia de aprobación.
   **Privada por defecto**; hacerla pública exige referencia de privacidad y
   consentimiento sellado reconocido para todos los nombres elegibles.

Las respuestas exitosas son JSON directo, no un objeto `data`. Los errores
conservan el envelope `error.codigo` del contrato general. Todas estas lecturas
usan `Cache-Control: no-store`. Las listas paginan con `limit` (máximo 100) y
`cursor` opaco; respuesta `items`, `nextCursor`, `hasMore`, `limit`.

## T23 · Entregas sin pérdida del evento

### Productores conectados

| Evento confirmado | Aviso y contenido |
|---|---|
| Envío EF-12 | Confirmación existente T19, sobre cifrado y PDF del mismo sello; `ef12-print-v1` |
| Publicación de requisitos/méritos | `resultado-etapa-v2`; no se notifica una revisión o cálculo provisional |
| Publicación psicotécnica / consolidación | `resultado-etapa-v2` |
| Cierre teórico definitivo | Puerto interno de T27, `resultado-etapa-v2`; productor real pendiente |
| Resolución de reconsideración | `reconsideracion-v2` |
| Publicación de lista T24 | `lista-elegibles-v1`, un aviso por expediente de la selección |

Los avisos de etapa anuncian la actualización y remiten al portal con sesión;
no contienen notas, motivos de exclusión, informes psicotécnicos ni URLs firmadas
que caduquen mientras esperan en cola. Destinatario: correo registrado al confirmar
el evento, no el correo declarado en el formulario ni un campo HTTP. Reintentar
conserva ese destinatario, cuerpo, recurso y `Message-ID`, sin regenerar el sello.
El PDF/reporte específico del examen debe implementarlo T27; esta entrega no lo
simula ni afirma adjuntarlo.

La cola AES-GCM y el evento comparten la transacción. Si no se puede encolar,
no se confirma la publicación/resolución. SMTP se ejecuta después del commit.
Un fallo SMTP no revierte la nota ni crea otra postulación.

### Rutas

| Método/ruta | Acceso |
|---|---|
| `GET /api/v1/notifications/submissions/{id}` | Titular |
| `GET /api/v1/notifications/submissions/{id}/{mail}/history` | Titular |
| `GET /api/v1/admin/notifications/submissions/{id}` | Analista/administrador, auditado |
| `GET /api/v1/admin/notifications/submissions/{id}/{mail}/history` | Analista/administrador, auditado |
| `POST /api/v1/admin/notifications/submissions/{id}/{mail}/retry` | Administrador vigente bloqueado, auditado en transacción |

`mail` es el `id` del listado de entregas, no el identificador del resultado.
La consulta se limita al expediente: no permite recorrer correos OTP ni otras
cuentas. Se incluyen la confirmación EF-12 y los avisos de etapa vinculados.
La migración vincula avisos T21/T22 anteriores sin cambiar su carga.

Cada entrega expone `id`, `accion`, `plantilla`, `estado`, `intentosCiclo`,
`intentosTotales`, `reenvios`, `codigoError`, `creadaEn`, `actualizadaEn`.
No expone destinatario, asunto, cuerpo, cifrado, OTP ni error SMTP libre.
El historial agrega inicio/fin de intento y reintento manual, con códigos seguros.
Los totales previos a la migración se conservan; no se inventan eventos antiguos.

Reintento:

```json
{ "reenviosAnteriores": 0, "motivo": "Restablecido el canal institucional" }
```

Sólo admite `failed`, carga todavía disponible y vencimiento original vigente.
Rechaza `pending`, `processing`, `smtp_accepted`, vencidos, OTP y `payload_invalid`.
El contador es optimista: dos solicitudes concurrentes no duplican la actuación;
la segunda recibe 409 y debe volver a consultar. No permite cambiar el destinatario.

Límites técnicos: cinco intentos automáticos por ciclo, hasta tres reintentos
manuales dentro de la vida original del aviso (siete días para estas etapas).
No son plazos legales de reconsideración. `intentosTotales` nunca se reinicia;
`intentosCiclo` sí, para el backoff. Se retiene la carga cifrada de un fallo sólo
hasta vencer. Aceptación, expiración o reemplazo la eliminan. Un fallo al auditar
el reintento revierte tanto el cambio de estado como su historial.

**SMTP es al menos una vez**: una aceptación seguida de caída antes de confirmar
el estado puede producir otro envío con el mismo Message-ID. `smtp_accepted` no
certifica llegada al buzón ni lectura del mensaje.

## T24 · Lista interna reproducible

Prefijo administrativo: **`/api/v1/admin/results`**.

| Método/ruta bajo el prefijo | Acceso |
|---|---|
| `POST /competitions/{period}/sectors/{sector}/lists` | Analista/administrador, generación auditada |
| `GET /competitions/{period}/sectors/{sector}/lists` | Analista/administrador, versiones internas |
| `GET /competitions/{period}/sectors/{sector}/publications` | Analista/administrador, versiones de publicación |
| `GET /lists/{id}` | Metadatos, reglas, versión y hash; no entradas completas |
| `GET /lists/{id}/entries` | Entradas paginadas del snapshot interno |
| `GET /lists/{id}/export.csv` | CSV de esa versión, selección completa acotada y auditada |
| `GET /publications/{id}` | Metadatos de publicación, aprobación y versión sustituida |
| `POST /lists/{id}/publish` | Exclusivamente administrador vigente, auditado |

Los dos POST requieren **`Idempotency-Key: <UUID>`**. Misma clave, actor, ruta y
contenido recuperan el recurso original, incluso si luego cerró la fase. Cambiar
cualquiera de ellos da 409. Devuelven 200 con metadatos; una clave nueva solicita
una actuación nueva, sujeta a controles de fuentes y versión.

Generar:

```json
{
  "reglas": {
    "orden": ["resultados", "meritos"],
    "referenciaOrden": "<acta institucional que aprueba el orden>"
  }
}
```

No hay orden institucional precargado. `orden` comienza con `resultados` y puede
incluir una vez cada componente numérico: `meritos`, `oposicion_teorica`,
`oposicion_psicotecnica`. Se compara puntuación relativa a su máximo con enteros
exactos, descendente. No puede usarse un concepto psicotécnico como puntaje.
Quienes mantienen igualdad reciben el mismo `puesto` y `empate: true`; el UUID
sólo estabiliza la presentación, **no resuelve el empate ni adjudica un cargo**.
Un desempate institucional distinto requiere ampliar y aprobar las reglas.

`sector` identifica el cargo/representación disponible en EF-12: `afiliados`,
`jubilados`, `empleadores`. No se inventa un catálogo adicional de cargos.

### Qué bloquea la generación/publicación

- Convocatoria/fase de resultados sin aprobación, publicación, habilitación o
  fuera de `[apertura,cierre)`, o postulación aún abierta.
- Fase publicada de reconsideración todavía no terminada, o solicitudes recibidas
  o atendidas sin resolución, aunque sus plazos hayan vencido.
- Falta de méritos publicados; para quienes avanzaron, falta de oposición y
  consolidación definitivas con política aprobada y fuentes actuales.
- Fuentes de otra fase, sello o convocatoria, desglose/total distinto al motor,
  consolidación anterior a una corrección, o persona duplicada en la selección.
- Selección superior a 10000 expedientes, 32 MiB de presupuesto o 20 s de operación.
  No se trunca en silencio. No se promete rendimiento de producción con esos límites.

Una exclusión publicada en méritos se conserva como no elegible: no se inventa
una nota teórica/psicotécnica para completarla. Una fuente ausente para alguien
que avanzó no se convierte en cero. Hasta conectar T27, estos expedientes reales
seguirán pendientes de una fuente legítima.

La generación y publicación bloquean el control común de T18 para no mezclarse
con nuevas postulaciones, correcciones o reconsideraciones durante la selección.
Publicar vuelve a consultar y comparar el hash de fuentes; una revisión concurrente
no actualiza silenciosamente el borrador. El coste de este bloqueo está acotado
por el presupuesto/timeout y requiere medición antes de uso masivo.

### Contenido y relación con T9

Cada entrada interna incluye `postulacionId`, `codigo`, `hashEnvio`, nombres,
`elegible`, `puesto`, `empate`, consentimiento reconocido, resultados de fase con
sus UUID, escalas y reglas, y referencias a reconsideraciones resueltas. Nombres
y cargo proceden del snapshot sellado, no del perfil actual.

El CSV conserva versión, convocatoria/sector, postulación, hash del sello, nombres,
elegibilidad/puesto/empate, UUID de las cuatro fuentes, sus puntajes/máximos
explícitos (vacío si no aplica) y hash de la lista. Se
neutralizan prefijos de fórmula; no contiene contactos, cédula ni adjuntos.
Se enlaza con la matriz T9 por **`postulacionId` + `hashEnvio`**, y con T20/T21
por los UUID de resultados. T9 sigue siendo matriz de captura, no una calificación
ni una lista oficial; no se modifica su contrato ni se recalculan notas al exportar.

## Publicación privada o pública

Publicar una primera versión interna:

```json
{
  "versionPublicadaAnterior": 0,
  "referenciaAprobacion": "<acta institucional>",
  "publica": false
}
```

Para corregir: enviar la versión publicada vigente y `sustituyeId` de su
publicación. Se requiere otro borrador; ni el anterior ni su publicación cambian.
Cada publicación encola un aviso por expediente dentro del mismo commit.

`publica: true` exige también `referenciaPrivacidad` no vacía y autorización de
nombre en cada elegible: versión reconocida RF v1.4, literal cuarto exacto,
aceptación y fecha en el contenido sellado. No basta una bandera HTTP ni el
perfil vigente. Si falta, toda la publicación pública se rechaza; no se entrega
una lista incompleta ocultando casos sin advertencia.

La referencia de privacidad es una **aprobación humana explícita y trazable**,
no una verificación automática de legalidad ni la implementación completa de T28.
La institución debe aportar/aprobar su política; retiro, conservación y otras
reglas futuras siguen en T28. No se preautoriza publicación pública al instalar.

### Rutas públicas sin sesión

- `GET /api/v1/public/results/competitions/{period}/sectors/{sector}`: publicación
  vigente, sólo si ésta es explícitamente pública.
- `GET /api/v1/public/results/publications/{id}`: versión histórica pública exacta.

Proyección pública distinta: `id` de publicación, `convocatoriaId`, `sector`,
`version`, `publicadaEn`, `orden` y **`nombres` paginados alfabéticamente**.
No expone código de postulación, UUID de postulante/expediente, hash individual,
puesto, puntaje, exclusión, resolución, responsable, contacto, cédula o PDF.

Si una nueva publicación es interna, la ruta de vigente no vuelve a presentar la
pública anterior como actual. El histórico público conserva su versión explícita.
El cursor público está ligado al UUID de publicación: si cambió la vigente entre
páginas, reiniciar la consulta o continuar por la ruta histórica devuelta.

## Errores y límites compartidos

| HTTP | Notificaciones | Listas |
|---:|---|---|
| 401/403 | Sesión / `notificacion_prohibida` | Sesión / `lista_prohibida` |
| 404 | `notificacion_no_encontrada` | `lista_no_encontrada`, también para una versión interna en ruta pública |
| 409 | `reintento_no_admitido` | `fuentes_lista_pendientes`, `version_lista_conflicto`, `fase_lista_cerrada`, `publicacion_nombre_no_autorizada` |
| 413 | — | `lista_demasiado_grande` |
| 422 | `notificacion_invalida` | `lista_invalida` |
| 503 | `notificacion_no_disponible` | `lista_no_disponible` |

Cuerpos estrictos de hasta 8 KiB; campos desconocidos y JSON adicional se rechazan.
Las lecturas administrativas no entregan datos si falla su auditoría.

**Verificación:** pruebas Go unitarias/HTTP y PostgreSQL temporal con datos
sintéticos. No constituye despliegue, prueba visual del frontend, entrega a buzón,
validación de política institucional ni prueba de un examen T27 real.
