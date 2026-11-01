# Evaluación y reconsideraciones — T21/T22

**Contrato de implementación local, 2026-10-31.** El RF exige estas capacidades,
pero no define instrumento psicotécnico, escalas, pesos o términos por actor.
No hay reglas oficiales precargadas. Este contrato no implementa el examen T27.

## Permisos y rutas

Todas requieren sesión; las operaciones POST de expediente exigen
**`Idempotency-Key: <UUID>`**. La misma clave, actor, ruta y contenido recuperan
el recibo original (200); una actuación nueva devuelve 201. Reutilizar la clave
con otro contenido da 409. Un reintento confirmado no vuelve a contar plazos ni
duplica correos, incluso si la ventana cerró después del primer commit.

| Método y ruta | Acceso |
|---|---|
| `GET /api/v1/evaluations/submissions/{id}` | Titular: resultados publicados; no registro psicotécnico provisional |
| `GET /api/v1/evaluations/submissions/{id}/history` | Titular, historial paginado |
| `GET /api/v1/evaluations/submissions/{id}/evidence/{evidence}/download` | Titular; informe institucional sólo tras su publicación |
| `GET /api/v1/admin/grades/submissions/{id}` | Analista/administrador, lectura auditada del expediente |
| `GET /api/v1/admin/grades/submissions/{id}/history` | Analista/administrador, auditado |
| `GET /api/v1/admin/grades/submissions/{id}/evidence/{evidence}/download` | Analista/administrador, auditado |
| `POST /api/v1/admin/grades/submissions/{id}/psychometric` | Analista/administrador, registrar informe |
| `POST /api/v1/admin/grades/submissions/{id}/psychometric/publish` | Administrador, publicar versión exacta |
| `POST /api/v1/admin/grades/submissions/{id}/consolidate` | Administrador, consolidar fuentes definitivas |
| `POST /api/v1/reconsiderations/submissions/{id}` | Titular con rol postulante |
| `GET /api/v1/reconsiderations/submissions/{id}` | Titular, lista paginada de sus solicitudes |
| `GET /api/v1/reconsiderations/submissions/{id}/{appeal}` | Titular, solicitud y sus actuaciones |
| `GET /api/v1/admin/reconsiderations/submissions/{id}` | Analista/administrador, lista auditada |
| `GET /api/v1/admin/reconsiderations/submissions/{id}/{appeal}` | Analista/administrador, detalle auditado |
| `POST /api/v1/admin/reconsiderations/submissions/{id}/{appeal}/attend` | **Analista** de Talento Humano |
| `POST /api/v1/admin/reconsiderations/submissions/{id}/{appeal}/resolve` | **Administrador**, Comisión |

No existe ruta para cargar una nota teórica desde el navegador. T27 debe invocar
el puerto interno `RecordFinalTheory` dentro de la transacción que confirma su
nota calculada. Hasta existir esa fuente definitiva, la consolidación devuelve
`resultado_definitivo_pendiente`, no cero ni una calificación simulada.

El historial devuelve `id`, `accion`, `fecha` y `detalle` con el snapshot de cada
versión/actuación. La lectura administrativa permite reconstruir correcciones;
el historial propio omite `detalle` de registros psicotécnicos provisionales.
Los resultados publicados y las actuaciones de su reconsideración sí tienen detalle.

Las respuestas GET usan `no-store`; las descargas devuelven `url` y `expiraEn`
(cinco minutos), nunca claves de objetos ni proxy de bytes.

## Reglas aprobadas y versionadas

Prefijo **`/api/v1/admin/grades`**:

- `POST /policies`: analista/administrador prepara una versión nueva.
  Cuerpo: `convocatoriaId`, `sector`, `tipo` (`oposicion`/`reconsideracion`),
  `definicion`. El servidor asigna `id`, `version`, `hash` y responsable.
- `GET /policies/{policy}`: lectura auditada.
- `GET /competitions/{period}/sectors/{sector}/policies/{kind}`: lista por cursor.
- `POST /policies/{policy}/approve`: administrador, cuerpo
  `{ "referenciaAprobacion": "<acta o resolución real>" }`.

Una aprobación valida la definición completa y activa esa versión. No cambia el
contenido; una corrección necesita otra versión y aprobación explícita. Se guardan
aprobador y fecha. No se verifica automáticamente la autenticidad de un acta.

### Definición de oposición

`definicion.oposicion` contiene:

| Campo | Regla |
|---|---|
| `instrumento`, `versionInstrumento` | Referencias explícitas del instrumento aprobado |
| `modoPsicotecnico` | `puntaje` o `concepto` |
| `maximoPsicotecnico` | Entero positivo en modo puntaje; `null` en concepto |
| `conclusiones`, `conclusionesAdmitidas` | Catálogo explícito y subconjunto admitido; sin «apto» por defecto |
| `maximoTeorico` | Escala esperada del resultado definitivo de T27 |
| `pesoMeritos`, `pesoTeorica`, `pesoPsicotecnico` | Enteros no negativos que suman 10000 puntos base |
| `maximoTotal`, `umbral` | Escala total explícita y umbral entre cero y máximo |
| `redondeo` | `inferior` o `mas_cercano` (mitades hacia arriba) |

En modo concepto el peso psicotécnico debe ser cero, pero la conclusión admitida
sigue siendo requisito de avance. No se convierte un concepto en puntaje inventado.

El motor `consolidacion-v1` calcula, por componente:
`redondear(maximoTotal × puntajeFuente / maximoFuente × peso / 10000)`.
Usa enteros y racionales exactos. La suma de máximos ponderados/redondeados debe
coincidir con el máximo total aprobado. Cada fuente entra una sola vez.

### Definición de reconsideraciones

`definicion.reconsideracion` contiene tres términos: `postulante`,
`talentoHumano`, `comision`. Cada uno declara:

- `cantidad`: entero positivo, sin valor por defecto.
- `unidad`: `minutos_transcurridos`, `dias_calendario` o `dias_habiles`.
- `incluyeDiaInicial`: explícito para cómputos por día.
- `horaLimite`: `HH:MM` para días; vacío en minutos transcurridos.

`calendario` declara `zonaHoraria: "America/Guayaquil"`,
`diasLaborables` (domingo=0…sábado=6), `feriados` (fechas ISO explícitas) y
`finVentana`: **`exigir_termino_completo`** o **`limitar_a_fase`**.
No se infieren feriados ni días legales. La primera opción rechaza una actuación
si el calendario no permite conceder el término completo; la segunda permite
limitarlo al cierre de la fase por decisión explícita de la política aprobada.

La presentación se cuenta desde la publicación del resultado o la apertura de
reconsideraciones, la posterior. Talento Humano se cuenta desde la presentación;
Comisión desde la atención. Se congelan política y fechas calculadas en la
solicitud. Aprobar otro calendario no reescribe los plazos ya registrados.
Las actuaciones requieren además la fase T18 publicada/habilitada y su ventana
vigente. Apertura inclusiva; plazo y cierre exclusivos. Manda el reloj de la base
después de adquirir los bloqueos, no el reloj ni la hora enviada por el cliente.

## T21: registro, publicación y consolidación

Registrar psicotécnica recibe: `versionAnterior` (0 inicialmente), `reglasId`,
`evaluadaEn` (no futura ni anterior al envío), `puntaje` (entero o null según modo),
`conclusion`, `motivo`, `respaldo: { nombre, pdfBase64 }`.
El informe registra instrumento/versión de la política y SHA-256 calculado por el
servidor. Una corrección crea otra versión; no modifica el registro anterior.

Publicar recibe `versionAnterior` de la publicación, `fuenteId` del registro
psicotécnico exacto, `reglasId`, `motivo` y, al corregir, `sustituyeId` de la
publicación previa. No basta editar un registro para cambiar un resultado público.
Ambas operaciones requieren fase `oposicion_psicotecnica`.

Consolidar requiere fase `resultados` y recibe `versionAnterior`, `reglasId`,
`motivo`, `sustituyeId` cuando corresponde, y
`fuentesEsperadas: { meritos, oposicion_teorica, oposicion_psicotecnica }`, cuyos
valores son UUID de publicaciones definitivas. `fuenteId` no se utiliza aquí.
Si las fuentes cambiaron, hay conflicto, no recálculo silencioso.

El expediente vincula publicaciones T20, nota definitiva T27 y psicotécnica
publicada al mismo sello. Guarda fuentes, escalas, pesos, aportes, total, versión
de motor y reglas. Un componente excluido impide avanzar aunque el total supere
el umbral. Cambiar un resultado fuente no sobrescribe una consolidación publicada:
debe confirmarse otra versión con referencia a la anterior.

## T22: actuaciones y efecto

Presentar recibe `fase` (`meritos`, `oposicion_teorica`,
`oposicion_psicotecnica`, `resultados`), `resultadoId` de la publicación vigente,
`motivo`, `documentosOriginales` (UUID de PDF del EF-12 propio, opcional) y
`respaldo` opcional con PDF adicional. La combinación expediente/fase/resultado
admite una única solicitud; otra clave no elude esa unicidad.

Estados y cuerpos:

1. **`recibida`**: conserva resultado original, política, fecha de presentación,
   límite de presentación y límite de Talento Humano.
2. **`atendida`**: analista envía `versionAnterior`, `observacion` y
   `recomendacion` (`confirmar`/`rectificar`). Guarda responsable/fecha y calcula
   límite de Comisión. El administrador no suplanta esta actuación del analista.
3. **`resuelta`**: administrador envía `versionAnterior`, `decision`, `motivo` y
   `resultadoRectificadoId` sólo para `rectificar`.

Confirmar exige que el resultado vigente siga siendo el original. Rectificar
exige otra publicación autorizada/calculada más reciente, del mismo expediente,
fase y sello. **La resolución no acepta puntajes arbitrarios ni calcula una nota
nueva.** El responsable debe realizar antes la corrección por el flujo propio
de la fase (T20/T21/T27), con su publicación y permisos. Si ese flujo requiere
reabrir formalmente una fase, se hace explícitamente mediante T18.

Se conserva original, resolución y resultado vigente. Se informa diferencia de
puntaje sólo si ambas versiones tienen la misma escala; en otros casos se
conservan ambas escalas sin una resta engañosa. El avance es el de la publicación
rectificada, nunca una bandera aportada por el solicitante.

## Archivos, auditoría y correo

Los PDF adicionales tienen límite técnico de 2 MiB por archivo. `pdfBase64` es
base64 del archivo, no una URL ni data-URL. El servidor comprueba firma PDF,
terminación, tamaño y nombre seguro; no afirma análisis antivirus ni validez
jurídica automática. Los respaldos de postulantes comparten la cuota de 50 MiB
con T4; los informes institucionales no consumen esa cuota personal.

Los objetos se guardan fuera del JSON sellado. Un intento durable previo permite
recoger objetos huérfanos si hay fallo o caída antes del commit. El limpiador sólo
retira intentos no confirmados vencidos; no purga evidencia histórica.

El commit incluye registro inmutable, cabecera, recibo idempotente y auditoría.
Publicaciones y resoluciones incluyen una notificación cifrada en el outbox de
T15. Si no se puede encolar, no se confirma la decisión. SMTP ocurre después y sus
fallos no revierten resultados ni duplican la actuación. El mensaje anuncia una
actualización; los puntajes/informes sensibles se consultan con sesión. No se
afirma llegada al buzón a partir de `smtp_accepted`.

## Paginación y errores

Las listas admiten `limit` (máximo 100) y `cursor` opaco ligado a expediente/tipo
de lista o convocatoria/sector/tipo de reglas. Respuesta: `items`, `hasMore`,
`nextCursor`, `limit`. No se utiliza OFFSET para los historiales.

Cuerpos JSON estrictos: 128 KiB para reglas/actuaciones y hasta 4 MiB para peticiones
con PDF base64, sujeto al límite de archivo. Campos desconocidos y JSON adicional
se rechazan; no hay campo de nota teórica editable.

| HTTP | Código |
|---:|---|
| 401/403 | Sesión/permisos o `evaluacion_prohibida` |
| 404 | `evaluacion_no_encontrada` |
| 409 | `reglas_evaluacion_pendientes`, `version_evaluacion_conflicto`, `plazo_reconsideracion_vencido`, `fase_evaluacion_cerrada`, `estado_reconsideracion_invalido`, `resultado_definitivo_pendiente` |
| 413 | `cuota_respaldos_excedida` |
| 422 | `evaluacion_invalida` |
| 503 | `evaluacion_no_disponible` |

Reintentar fallos de conexión con la **misma clave y contenido**. Ante un conflicto
real, consultar el expediente, revisar el cambio y emitir una actuación nueva,
no sustituir silenciosamente la versión anterior.

## Pendientes institucionales y de integración

- Instrumento, formato, escalas, pesos, umbrales y calendarios deben ser aportados
  y aprobados; el RF no fija sus valores.
- T27 debe conectar su cierre calculado al puerto interno. Hasta entonces no hay
  nota teórica legítima para consolidar un expediente real.
- Esta entrega no modifica pantallas del frontend, no aplica migraciones a la base
  de aplicación y no constituye despliegue ni aprobación institucional.
