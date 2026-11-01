# Contrato de la API

**Sistema de Registro y Postulación al Directorio del BIESS**
Contrato HTTP v1.0

| Campo | Valor |
|---|---|
| Documento | API_CONTRACT v1.0 |
| Base de la API | `/api/v1` |
| Formato | JSON (`application/json; charset=utf-8`) |
| TRD asociado | [`../trd/TRD.md`](../trd/TRD.md) |

> **Estado**: este documento describe el contrato **vigente** (el contexto
> `identity`, `submission`, `document`, `catalog` y los endpoints transversales) y **anticipa** el de los contextos
> que faltan. Los endpoints marcados como *planificado* todavía no existen en
> el servidor: su forma está declarada para que el frontend y el backend se
> desarrollen en paralelo sin sorpresas.

---

## 1. Convenciones

### 1.1 Base y versionado

- Prefijo: `/api/v1`.
- En desarrollo, el navegador llama a `/api/...` y el proxy de Vite reenvía a
  `http://localhost:8080`. En producción, nginx hace lo mismo. **El cliente
  nunca conoce la URL de la API**: siempre usa rutas relativas.

### 1.2 Autenticación

- La sesión viaja en una **cookie `session`** con `HttpOnly`, `SameSite=Lax` y
  `Secure` en producción.
- El token **nunca** se devuelve en el cuerpo de la respuesta ni se guarda en
  `localStorage`: el frontend no lo puede leer.
- Rutas públicas: `GET /healthz`, `GET /readyz`, `/api/v1/auth/register/request`,
  `/register/verify`, `/login`, `/password/forgot` y `/password/reset` (las rutas
  abreviadas comparten el prefijo `/api/v1/auth` y usan `POST`).

### 1.3 Cabeceras

| Cabecera | Dirección | Uso |
|---|---|---|
| `X-Request-ID` | Ambos | Identificador de correlación. Si el cliente lo envía, la API lo conserva; si no, lo genera |
| `Content-Type` | Petición | `application/json` |
| `Accept` | Petición | `application/json` |

### 1.4 Envelope de error

Único para toda la API:

```json
{
  "error": {
    "codigo": "postulacion_no_editable",
    "mensaje": "La postulación ya fue enviada y no admite cambios.",
    "detalle": ["opcional: información adicional"]
  }
}
```

- `codigo` — identificador **estable**; es lo que consume el frontend para
  decidir qué hacer (mensaje, redirección, reintento).
- `mensaje` — texto en español destined al usuario final.
- `detalle` — opcional, array de strings; hoy no se usa.
- Las reglas EF-12 incluyen la ruta del campo o sección en el mensaje, sin valores personales.

### 1.5 Paginación

Dos estrategias sobre el mismo kernel. El modo se elige por los parámetros
recibidos: **si llega `limit` o `cursor`, se usa cursor**; si no, se usa
offset.

#### Offset — listados acotados

| Parámetro | Tipo | Por defecto | Límite |
|---|---|---|---|
| `page` | entero ≥ 1 | `1` | — |
| `pageSize` | entero ≥ 1 | `20` | `100` |

```json
{
  "items": [ /* ... */ ],
  "page": 1,
  "pageSize": 20,
  "total": 137
}
```

#### Cursor (keyset) — listados grandes

| Parámetro | Tipo | Por defecto | Límite |
|---|---|---|---|
| `limit` | entero ≥ 1 | `20` | `100` |
| `cursor` | string opaco | — | — |

```json
{
  "items": [ /* ... */ ],
  "nextCursor": "MjAyNi0wMy0wMVQxMjowMDowMFo",
  "hasMore": true,
  "limit": 20
}
```

El cursor es **opaco**: el cliente lo reenvía tal cual y no lo interpreta. Un
cursor inválido devuelve `400` con el código `cursor_invalido`.

### 1.6 Forma de la respuesta de un listado

Cada listado devuelve **sus elementos en la raíz**, con los metadatos de
paginación como campos hermanos:

```json
{ "items": [ /* ... */ ], "page": 1, "pageSize": 20, "total": 137 }
```

Se evaluó la alternativa de envolverlos en claves explícitas
(`data` / `meta` / `kpis`) y **no se adoptó**, por dos razones:

1. **Cada elemento ya está en un campo con nombre.** `items`, `total` y
   `nextCursor` se leen igual de bien que `data` y `meta`, y el envoltorio
   añade un nivel que el cliente tiene que desenvolver sin información nueva.
2. **La forma es uniforme entre estrategias.** Offset y cursor comparten
   `items`, así que el frontend tiene un solo caso que resolver.

Queda reservado un campo `kpis` para el listado del **dashboard de revisión**
(EF-13), donde los conteos por cargo y por estado acompañan a los elementos y
no son metadatos de paginación. Ese endpoint es planificado (§7), así que la
forma se definirá cuando exista.

### 1.7 Códigos de error

| HTTP | `codigo` | Cuándo ocurre |
|---:|---|---|
| 400 | `cuerpo_invalido` | El JSON no se puede deserializar o trae campos desconocidos |
| 400 | `postulante_invalido` | `applicantId` no es un UUID válido |
| 400 | `identificador_invalido` | El `{id}` de la ruta no es un UUID válido |
| 400 | `contenido_invalido` | El contenido del formulario no es un objeto JSON válido |
| 400 | `paginacion_invalida` | `page` o `pageSize` no son numéricos |
| 400 | `cursor_invalido` | El cursor no es un token válido |
| 401 | `no_autenticado` | La ruta exige sesión y no hay cookie válida |
| 401 | `credenciales_incorrectas` | Cédula o contraseña incorrectas |
| 403 | `acceso_denegado` | La sesión no corresponde al recurso |
| 404 | `postulacion_no_encontrada` | No existe una postulación con ese `{id}` |
| 404 | `ruta_no_encontrada` | La ruta no existe |
| 405 | `metodo_no_permitido` | El método no aplica a esa ruta |
| 409 | `postulacion_no_editable` | La postulación ya fue enviada |
| 409 | `codigo_duplicado` | Ya existe una postulación con ese código |
| 400 | `seccion_invalida` | La sección solicitada no está entre 1 y 9 |
| 413 | `cuerpo_muy_grande` | El cuerpo supera `MAX_REQUEST_BODY_BYTES` (10 MiB) |
| 415 | `tipo_no_soportado` | El adjunto no es un PDF válido |
| 422 | `regla_de_negocio` | Una validación del RF falla (nacionalidad, sector, límites) |
| 429 | `intentos_excedidos` | Cuenta bloqueada 10 minutos tras 3 intentos fallidos |
| 500 | `error_interno` | Error inesperado; el detalle va al log con `request_id` |
| 503 | `base_datos_no_disponible` | La base no responde (readiness) |

---

## 2. Endpoints transversales

### 2.1 `GET /healthz`

Sonda de vida del proceso. No toca la base de datos.

```bash
curl -s http://localhost:8080/healthz
```

```json
{
  "status": "ok",
  "version": "dev",
  "time": "2026-04-01T12:00:00Z"
}
```

### 2.2 `GET /readyz`

Sonda de disponibilidad: comprueba que PostgreSQL responde.

```json
{ "status": "ready" }
```

Cuando la base no está disponible: `503` con `base_datos_no_disponible`.

---

## 3. Contexto `submission` (vigente)

Rutas registradas por `internal/submission/handler.go`:

| Método | Ruta | Descripción |
|---|---|---|
| `POST` | `/api/v1/submissions` | Abre una postulación en estado `draft` |
| `GET` | `/api/v1/submissions` | Lista las postulaciones de un postulante |
| `GET` | `/api/v1/submissions/{id}` | Obtiene una postulación |
| `PUT` | `/api/v1/submissions/{id}` | Reemplaza el contenido de una postulación editable |
| `DELETE` | `/api/v1/submissions/{id}` | Elimina una postulación no enviada |
| `POST` | `/api/v1/submissions/{id}/sections/{n}` | Guarda parcialmente una de las nueve secciones y devuelve el progreso |
| `POST` | `/api/v1/submissions/{id}/submit` | Valida, sella y notifica el envío |

### 3.1 `POST /api/v1/submissions`

Crea una postulación en estado `draft`. El contenido se **canoniza** (claves
ordenadas) antes de persistirse.

**Cuerpo**

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `applicantId` | string (UUID) | no | Se toma de la sesión; si se incluye debe coincidir con ella |
| `code` | string | no | Código visible; si se omite, la API lo genera (`EF12-AAAA-XXXXXX`) |
| `content` | objeto JSON | sí | Secciones del formulario EF-12 |

```bash
curl -X POST http://localhost:8080/api/v1/submissions \
  -H 'Content-Type: application/json' \
  -d '{
    "applicantId": "11111111-1111-1111-1111-111111111111",
    "code": "EF12-2026-A1B2C3",
    "content": {
      "seccion1": { "nombres": "Ana", "apellidos": "Pérez", "nacionalidad": "Ecuatoriana" },
      "seccion2": { "pais": "Ecuador", "provincia": "Pichincha", "canton": "Quito" }
    }
  }'
```

**Respuesta `201`**

```json
{
  "id": "044d3694-4ca2-4d87-aa08-ed9e35482710",
  "applicantId": "11111111-1111-1111-1111-111111111111",
  "code": "EF12-2026-A1B2C3",
  "status": "draft",
  "content": {
    "seccion1": { "apellidos": "Pérez", "nacionalidad": "Ecuatoriana", "nombres": "Ana" },
    "seccion2": { "canton": "Quito", "pais": "Ecuador", "provincia": "Pichincha" }
  },
  "createdAt": "2026-04-01T12:00:00Z",
  "updatedAt": "2026-04-01T12:00:00Z"
}
```

> Nótese que `content` llega con las **claves ordenadas**: esa es la
> canonización que garantiza un hash estable.

**Errores**

| HTTP | `codigo` | Causa |
|---:|---|---|
| 400 | `cuerpo_invalido` | JSON malformado o campo desconocido |
| 400 | `postulante_invalido` | `applicantId` ausente o no UUID |
| 400 | `contenido_invalido` | `content` vacío o no es un objeto JSON |
| 409 | `codigo_duplicado` | Ya existe una postulación con ese `code` |

### 3.2 `GET /api/v1/submissions`

Lista las postulaciones del usuario autenticado. `applicantId` es opcional y, si se incluye, debe coincidir con la sesión; de otro modo responde `403 acceso_denegado`.

```bash
# Offset
curl -s 'http://localhost:8080/api/v1/submissions?applicantId=11111111-1111-1111-1111-111111111111&page=1&pageSize=20'

# Cursor
curl -s 'http://localhost:8080/api/v1/submissions?applicantId=11111111-1111-1111-1111-111111111111&limit=20'
```

**Respuesta `200` (offset)**

```json
{
  "items": [
    {
      "id": "044d3694-4ca2-4d87-aa08-ed9e35482710",
      "applicantId": "11111111-1111-1111-1111-111111111111",
      "code": "EF12-2026-A1B2C3",
      "status": "draft",
      "content": { "seccion1": {} },
      "createdAt": "2026-04-01T12:00:00Z",
      "updatedAt": "2026-04-01T12:00:00Z"
    }
  ],
  "page": 1,
  "pageSize": 20,
  "total": 1
}
```

**Respuesta `200` (cursor)**

```json
{
  "items": [ /* ... */ ],
  "nextCursor": "MjAyNi0wMy0wMVQxMjowMDowMFo",
  "hasMore": false,
  "limit": 20
}
```

**Errores**

| HTTP | `codigo` | Causa |
|---:|---|---|
| 400 | `postulante_invalido` | Falta `applicantId` o no es un UUID |
| 400 | `paginacion_invalida` | `page` o `pageSize` no numéricos |
| 400 | `cursor_invalido` | El cursor no es un token válido |

### 3.3 `GET /api/v1/submissions/{id}`

```bash
curl -s http://localhost:8080/api/v1/submissions/044d3694-4ca2-4d87-aa08-ed9e35482710
```

**Respuesta `200`**: el mismo objeto de postulación. Cuando la postulación ya
fue enviada, `hash` y `submittedAt` aparecen con valor.

**Errores**: `400 identificador_invalido` · `404 postulacion_no_encontrada`.

### 3.4 `PUT /api/v1/submissions/{id}`

Reemplaza el contenido del formulario de una postulación **editable**.

**Cuerpo**

```json
{
  "content": {
    "seccion5": { "titulos": [] }
  }
}
```

**Respuesta `200`**: la postulación actualizada, con `updatedAt` nuevo y el
contenido canonizado.

**Errores**

| HTTP | `codigo` | Causa |
|---:|---|---|
| 400 | `cuerpo_invalido` | JSON malformado |
| 400 | `contenido_invalido` | `content` vacío o no es un objeto |
| 404 | `postulacion_no_encontrada` | No existe |
| 409 | `postulacion_no_editable` | La postulación ya fue enviada (RN-06) |

### 3.5 `DELETE /api/v1/submissions/{id}`

Elimina una postulación que **todavía no fue enviada**.

**Respuesta `204`**: sin cuerpo.

**Errores**: `404 postulacion_no_encontrada` · `409 postulacion_no_editable`.

---

## 4. Guardado parcial y envío (`submission`, vigente)

Todas las rutas de postulaciones exigen cookie de sesión. Solo su propietario
puede consultar, guardar, enviar o eliminar el recurso (`403 acceso_denegado`
para otra sesión). La cédula de `seccion1`, si se informa, debe ser la de la sesión.
Los identificadores de postulaciones y postulantes se representan como UUIDv7.

### Definición del EF-12 y vínculo con la cuenta OTP (T3)

`GET /api/v1/submissions/form-definition` requiere sesión y devuelve la versión
`EF12-v1.4-otp`, opciones de captura (incluidos todos los países del RF), nueve
secciones con campos obligatorios/opcionales/condicionales, provincias/cantones,
las trece preguntas literales, límites y los cinco textos del consentimiento.
Es el contrato embebido en `internal/submission/definition.json`; el validador
puro TypeScript utiliza el mismo archivo. El cliente lo consume mediante
`obtenerDefinicionFormulario`.

Al crear sin `content`, se abre un borrador vacío y se precargan nombres,
apellidos, cédula y correo principal desde la cuenta OTP. Cédula y correo principal
se contrastan con la cuenta en cada escritura, incluido el envío; no se permite
sustituirlos por otros. Los nombres pueden corregirse como datos declarados.
Nacimiento, nacionalidad y estado civil deben completarse; no se infieren.
`verificadoRegistroCivil` se normaliza a `false` en los nuevos guardados.

Los borradores permiten campos pendientes. Los IDs de registros declarados deben
ser estables, únicos dentro de su lista y de hasta 255 caracteres; se exige un ID
no vacío al enviar. La selección de parentesco y país se contrasta con el catálogo.
Un empleo marcado como actual no admite una fecha de finalización contradictoria.
Se conserva el contrato de requeridos existente: correo 2 es opcional; pareja y
organización dependen de su respuesta; capacitación admite una lista declarada
vacía. Firma digital dibujada no es obligatoria. `definition.json` expone el detalle
por campo, sin inventar requisitos para los vacíos del RF.

`content.consentimiento` incluye `version` y cinco `puntos` desde el guardado,
con las aceptaciones que declare el usuario. Guardar no acepta automáticamente
ningún punto. El servidor elimina un `aceptadoEn` suministrado en borrador y solo
lo asigna al sellar, sustituyendo versión/textos por sus literales vigentes.
Una lectura de una postulación enviada conserva su contenido y hash originales.

La elegibilidad laboral controlada por el servidor y el cierre persistente de T17
siguen separados de esta validación de captura. Las referencias reales a PDFs se
comprueban al enviar mediante T4; un validador local no acredita que un objeto exista.

### 4.1 `POST /api/v1/submissions/{id}/sections/{n}`

`n` es un entero de **1 a 9**. El cuerpo es directamente un objeto de la sección,
sin envoltorio `content`. Ejemplo para la sección 2:

```json
{
  "pais": "Ecuador",
  "provincia": "Pichincha",
  "canton": "Quito"
}
```

Se mezclan los campos con la versión almacenada: los omitidos se conservan,
los objetos anidados se mezclan y las listas se reemplazan completas. `null`
elimina un campo de la sección. Por ejemplo, para cambiar la respuesta 1.9.7:
`{"q7":"No","q7Detalles":null}`. Las demás secciones no cambian.

El guardado admite campos pendientes. Los valores presentes inválidos se
rechazan con `422 regla_de_negocio`, sin persistir el cambio. Los campos
obligatorios pendientes no cuentan como completos y bloquean el envío final.
Las nueve secciones usan las claves y opciones de
`apps/frontend/src/types/concurso.ts`; la API rechaza campos desconocidos y tipos
incorrectos. Las fechas del formulario son `AAAA-MM-DD`, sin hora.

La respuesta `200` es la postulación completa (§3) e incluye `progress`:

```json
{
  "total": 9,
  "completed": 2,
  "percentage": 22,
  "sections": [
    { "number": 1, "complete": true },
    { "number": 2, "complete": true },
    { "number": 3, "complete": false },
    { "number": 4, "complete": false },
    { "number": 5, "complete": false },
    { "number": 6, "complete": false },
    { "number": 7, "complete": false },
    { "number": 8, "complete": false },
    { "number": 9, "complete": false }
  ]
}
```

Este objeto aparece en el campo `progress` de todas las respuestas de
postulaciones. El porcentaje es `floor(completed * 100 / 9)`; los cinco
consentimientos se comprueban por separado al enviar.

| Sección | Clave | Validación principal |
|---:|---|---|
| 1 | `seccion1` | Nacionalidad `Ecuatoriana`, cédula válida, género, fecha y estado civil; pareja si está casado o en unión de hecho |
| 2 | `seccion2` | País → provincia → cantón; en Ecuador se valida pertenencia al catálogo del formulario; en el extranjero, formato y longitud; teléfonos y correos |
| 3 | `seccion3` | Sí/No; si responde Sí, al menos un familiar completo de IESS o BIESS |
| 4 | `seccion4` | Sector `afiliados`, `jubilados` o `empleadores`, elegibilidad afirmativa y nombre de organización si declara respaldo |
| 5 | `seccion5` | Al menos un título, nivel, tipo, área de estudio y registro SENESCYT |
| 6 | `seccion6` | Al menos una experiencia, fechas ordenadas y funciones de hasta 1000 caracteres Unicode |
| 7 | `seccion7` | Sí/No; las publicaciones declaradas llevan tipo, medio, fecha y relación con el campo |
| 8 | `seccion8` | Lista explícita, que puede estar vacía; eventos completos, horas positivas y fechas ordenadas |
| 9 | `seccion9` | Las trece respuestas Sí/No; con `q7 = Sí`, institución, cargo, fecha de fin y motivo |

La duración se calcula en el servidor: `duracionCalculada` del cliente se
reemplaza. Para un empleo actual se usa la fecha de Ecuador (UTC−5), y al enviar
se vuelve a calcular y queda congelada en el sello. Go y TypeScript ejecutan
los mismos casos de prueba para años, meses y días, incluidos febrero y bisiestos.

**Límites de integración:** `verificacionIessCumplida` es actualmente un estado
declarado por el cliente. Se valida la regla y se rechaza el estado negativo
con el mensaje del RF, pero **todavía no se consulta al IESS**. Del mismo modo,
`verificadoRegistroCivil` no acredita una consulta al Registro Civil: no existe
esa integración en el alcance actual (ADR-008). EF-12 precarga datos de la cuenta
OTP y recoge el resto como declaraciones del postulante. El catálogo
geográfico replica la referencia local del frontend; su persistencia y caché
corresponden a T7. Los respaldos se cargan mediante el contexto `document` (sección 4.3). Al enviar,
se comprueban sus identificadores y se normalizan sus metadatos desde PostgreSQL;
un nombre y tamaño declarados por el cliente no acreditan un archivo.

### 4.2 `POST /api/v1/submissions/{id}/submit`

Puede enviarse con cuerpo vacío o `{}` si el consentimiento ya está guardado en
`content.consentimiento` mediante `PUT`. También acepta las cinco declaraciones
en la misma petición:

```json
{
  "consentimiento": {
    "aceptacionTratamientoDatos": true,
    "declaracionVeracidad": true,
    "aceptacionNormasConcurso": true,
    "autorizacionPublicacionNombre": true,
    "constanciaEntregaFisica": true
  }
}
```

Valida las nueve secciones completas y las cinco aceptaciones. Guarda en
`content.consentimiento` los cinco textos literales del RF, `version` y
`aceptadoEn` asignados por el servidor. La firma digitalizada y `fechaFirma`
son opcionales: el RF exige entregar el impreso con firma manuscrita.

La respuesta `200` contiene la postulación completa con `status: submitted`,
`hash` SHA-256 de 64 caracteres y `submittedAt`. El JSON se canoniza
**recursivamente**, también dentro de listas; cambiar el orden de claves no
cambia el hash. Recalcular SHA-256 sobre el contenido canónico recuperado debe
dar el mismo valor.

Guardar y enviar se ejecutan con `SELECT ... FOR UPDATE` y commit transaccional.
Un segundo envío responde `409` y no vuelve a notificar. `PUT`, el guardado por
secciones y `DELETE` también responden `409` después del sello; el SQL impide
modificar o borrar una fila enviada aunque compitan peticiones.

Después del commit se envía un correo con código, fecha, hash y contenido
sellado al correo **registrado en la cuenta**, no al destinatario aportado en el
formulario. Si SMTP falla o no está configurado, el sello se conserva y la API
responde `200`; se registra un aviso sin datos personales. La entrega es de mejor
esfuerzo, sin cola persistente ni reintentos automáticos (T15).

| HTTP | `codigo` | Causa |
|---:|---|---|
| 400 | `cuerpo_invalido` | JSON malformado, múltiples raíces o campos desconocidos del envoltorio |
| 400 | `contenido_invalido` | Contenido no tipado, sección no objeto o campos desconocidos |
| 400 | `seccion_invalida` | `n` no es un entero de 1 a 9 |
| 401 | `no_autenticado` / `sesion_invalida` | Falta sesión o expiró |
| 403 | `acceso_denegado` | Otra cuenta o cédula distinta de la sesión |
| 404 | `postulacion_no_encontrada` | No existe la postulación |
| 409 | `postulacion_no_editable` | Ya está sellada |
| 422 | `regla_de_negocio` | Valor inválido, sección incompleta al enviar o consentimiento faltante |

Las pruebas de PostgreSQL están en `internal/submission/postgres/repository_integration_test.go`.
Se habilitan con `SUBMISSION_TEST_DATABASE_URL` contra una base de pruebas; crean y
eliminan un esquema aislado. Cubren concurrencia, abortos, doble sellado, hash
reproducible desde JSONB y rechazo de escrituras obsoletas.

### 4.3 Adjuntos PDF (`document`, vigente)

Todas las rutas requieren cookie de sesión y titularidad. La carga y la baja
requieren una postulación editable; la descarga sigue disponible después del envío.

| Método | Ruta | Respuesta |
|---|---|---|
| `POST` | `/api/v1/submissions/{id}/documents` | `201`, metadatos del PDF |
| `GET` | `/api/v1/submissions/{id}/documents` | `200`, array de metadatos (sin paginación) |
| `GET` | `/api/v1/documents/{id}/download` | `200`, `{ "url": "https://...", "expiresAt": "..." }` |
| `DELETE` | `/api/v1/documents/{id}` | `204`, baja lógica confirmada y borrado del objeto encolado |

**Carga:** `multipart/form-data`, exactamente un archivo en `file` y los campos:

| Campo | Tipo | Regla |
|---|---|---|
| `section` | Entero | 4, 5, 6, 7 u 8 |
| `recordId` | Texto | ID estable del título/experiencia/publicación/capacitación previamente guardado; vacío u omitido para sección 4 |
| `file` | Archivo | MIME declarado `application/pdf` y contenido detectado como PDF (`%PDF-`) |

Guardar primero el registro de la sección con un `id` no vacío. La sección 4
requiere `cuentaConRespaldoOrganizacion: "Sí"`. Solo se admite un respaldo por
registro; para reemplazarlo, eliminar el anterior y volver a cargar.

```bash
curl -b cookies.txt -F 'section=5' -F 'recordId=titulo-1' \
  -F 'file=@titulo.pdf;type=application/pdf' \
  http://localhost:8080/api/v1/submissions/UUID/documents
```

Ejemplo de metadatos (la clave del almacenamiento nunca se devuelve):

```json
{
  "id": "019c0000-0000-7000-8000-000000000001",
  "submissionId": "019c0000-0000-7000-8000-000000000002",
  "section": 5,
  "recordId": "titulo-1",
  "name": "titulo.pdf",
  "mime": "application/pdf",
  "sizeBytes": 123456,
  "sha256": "<64 caracteres hexadecimales>",
  "createdAt": "2026-10-31T12:00:00Z"
}
```

Los límites usan **MiB**: sección 4 hasta **4 194 304 bytes**; secciones 5–8 hasta
**2 097 152 bytes**; total por cuenta, entre todas sus postulaciones, hasta
**52 428 800 bytes**. El servidor cuenta bytes reales y admite el límite exacto.
El envoltorio multipart está acotado a 4 MiB + 64 KiB y también se aplica
`MAX_REQUEST_BODY_BYTES` (10 MiB por defecto). Se rechazan campos desconocidos,
archivos múltiples y cuerpos chunked que exceden el límite.

La carga confirma metadatos y referencia del formulario en una transacción.
La referencia JSON utiliza `id`, `nombre`, `tamano` (bytes decimales como texto),
`fechaSubida` y `sha256`. La clave aleatoria es
`{submission_id}/{section}/{uuidv4}.pdf`; el ID del documento en PostgreSQL es UUIDv7.
La cuota se serializa bloqueando al postulante y luego la postulación. Cargas,
bajas y envío comparten el bloqueo de la postulación. Una carga rechazada se
compensa eliminando el objeto; si el resultado de COMMIT es incierto, primero se
consulta si existen sus metadatos para evitar borrar un objeto confirmado.

**Envío:** exige respaldos reales para organización (cuando corresponde), títulos,
experiencias, publicaciones y capacitaciones declaradas. Comprueba documento,
postulación, sección y registro; reemplaza los metadatos enviados por los de la
base de datos antes de calcular el SHA-256 canónico. Si se retiró un registro del
formulario, debe eliminarse su respaldo antes de enviar. Faltantes, referencias
falsas o respaldos sin registro producen `422 regla_de_negocio`. El guardado de
borradores sigue permitiendo campos pendientes.

**Descarga:** la API entrega JSON con una URL de **300 segundos**, sin proxy del
binario ni redirección. Es una capacidad de lectura: quien la posea puede usarla
hasta que expire; no registrar ni compartir su query string. Las respuestas de
listado y URL llevan `Cache-Control: no-store`. El servidor de objetos fuerza PDF,
`Content-Disposition: attachment` y ausencia de caché. La baja física se procesa
al arrancar y cada minuto; los enlaces emitidos pueden funcionar hasta el borrado
o su expiración. Los fallos de almacenamiento conservan la baja en PostgreSQL,
incluidas las cascadas al eliminar borradores.

| HTTP | Código | Causa |
|---|---|---|
| 400 | `cuerpo_invalido` | Multipart malformado, campos desconocidos o número de archivos incorrecto |
| 400 | `identificador_invalido` / `seccion_invalida` / `nombre_invalido` | Parámetros inválidos |
| 401 | `no_autenticado` / `sesion_invalida` | Sesión ausente o expirada |
| 403 | `acceso_denegado` | Documento o postulación de otra cuenta |
| 404 | `documento_no_encontrado` / `postulacion_no_encontrada` | Recurso inexistente |
| 409 | `postulacion_no_editable` | Carga o baja después del envío |
| 409 | `documento_duplicado` | Ya existe un PDF para ese registro |
| 413 | `archivo_muy_grande` / `cuota_documentos_excedida` | Límite de archivo/multipart o cuota total |
| 413 | `cuerpo_muy_grande` | Límite global de petición anterior al handler |
| 415 | `tipo_no_soportado` | MIME declarado o firma binaria diferente de PDF |
| 422 | `registro_invalido` | Registro inexistente o sin ID estable |
| 500 | `error_interno` | Fallo de persistencia/almacenamiento; sin datos sensibles |

**Almacenamiento local:** `STORAGE_DRIVER=local`, `STORAGE_DIR`,
`STORAGE_PUBLIC_URL` y `STORAGE_SIGNING_KEY` (mínimo 32 bytes). Ejecutar un segundo
proceso con `go run ./cmd/api storage` (`STORAGE_PORT=8081`). Ambos procesos deben
compartir directorio y clave; `compose.yaml` ya lo configura. El servidor solo
sirve GET/HEAD con firma HMAC y vencimiento; no publica directorios ni rutas sin
firma. Las aperturas usan `os.Root` para impedir escapes por rutas o symlinks.

**S3:** `STORAGE_DRIVER=s3`, `S3_ENDPOINT` (URL con bucket incluido), `S3_REGION`,
`S3_ACCESS_KEY`, `S3_SECRET_KEY` y opcional `S3_SESSION_TOKEN`. Para un host de
lectura distinto, definir `S3_PUBLIC_ENDPOINT`. Ejemplos de endpoint:
`https://bucket.s3.region.amazonaws.com` o `http://minio:9000/bucket`. El bucket
privado debe existir; se requieren permisos PutObject/GetObject/DeleteObject.
No se aprovisiona el bucket ni se refrescan credenciales STS automáticamente.
Las credenciales se inyectan por entorno; en producción se usa HTTPS.

**Pruebas:** reglas y HTTP sin red; firma SigV4 contra el vector público oficial
AWS; transporte S3 y respuestas 503 con doble HTTP. PostgreSQL optativo:
`DOCUMENT_TEST_DATABASE_URL` (PostgreSQL 18, esquema aislado). Simulación Postman:
`postman simulation run postman/simulations/document-storage/document-storage.sim.yaml --port 4011 --no-history`;
después ejecutar `go test ./internal/platform/storage -run TestS3Simulation` con
`S3_SIMULATION_ENDPOINT=http://127.0.0.1:4011/s3/bucket`. El doble no valida IAM ni
SigV4: no sustituye una comprobación con el bucket real.

### 4.4 Endpoint pendiente

| Método | Ruta | Descripción |
|---|---|---|
| `GET` | `/api/v1/submissions/{id}/print` | Formulario imprimible con espacio de firma |

---

## 5. Contexto `identity` (vigente: OTP por correo)

El proyecto utiliza módulo 10 + OTP por correo para registro y recuperación,
según [ADR-008](../decisions/ADR-008-registro-y-recuperacion-con-otp.md). No solicita
fecha de expedición ni código dactilar y no certifica identidad civil.

| Método | Ruta | Comportamiento actual |
|---|---|---|
| `POST` | `/api/v1/auth/register/request` | Recibe cédula y correo, envía OTP y devuelve `200` con mensaje |
| `POST` | `/api/v1/auth/register/verify` | Verifica OTP, crea la cuenta con `otp_correo` y devuelve `201` con `postulante`; puede emitir cookie de sesión |
| `POST` | `/api/v1/auth/login` | Cédula y contraseña; `200` con `postulante` y cookie de sesión |
| `POST` | `/api/v1/auth/logout` | Revoca sesión y limpia cookie |
| `GET` | `/api/v1/auth/session` | Perfil actual dentro de `postulante`, incluido `rol`, o `401` |
| `POST` | `/api/v1/auth/password/forgot` | Solicita cédula y envía OTP al correo registrado; mensaje genérico |
| `POST` | `/api/v1/auth/password/reset` | Verifica OTP y actualiza contraseña; invalida sesiones anteriores |

### 5.1 Registro y recuperación

Los cuerpos vigentes corresponden a `internal/identity/dto.go`:

```json
{ "cedula": "1710034008", "correo": "cuenta@example.test" }
```

Es el cuerpo de `register/request`. El OTP tiene seis dígitos y TTL actual de diez
minutos. `register/verify` recibe:

```json
{
  "cedula": "1710034008",
  "correo": "cuenta@example.test",
  "codigo_otp": "123456",
  "nombres": "Persona",
  "apellidos": "Prueba",
  "password": "UnaClaveSegura1!"
}
```

Recuperación: `password/forgot` recibe `{ "cedula": "1710034008" }`.
`password/reset` recibe `cedula`, `codigo_otp` y `new_password`; el destinatario
no forma parte del cuerpo. No se utiliza un enlace de restablecimiento como flujo vigente.

La base OTP está implementada. T16 debe completar vinculación a correo/finalidad,
consumo atómico y controles de reenvío; esta descripción no afirma que esos
controles pendientes ya estén garantizados. TTL del OTP y bloqueo de diez minutos
son conceptos diferentes.

### 5.2 Login y sesión

```json
{ "cedula": "1710034008", "password": "UnaClaveSegura1!" }
```

Respuesta de login: cookie `session` y perfil bajo la clave **`postulante`**.
`GET /auth/session` utiliza la misma clave y `ApplicantResponseDTO`, cuyos campos
incluyen `id`, `cedula`, `nombres`, `apellidos`, `nombre_completo`, `correo`,
`rol`, `metodo_validacion` y `fecha_registro`.

El contrato actual no devuelve `mustCompleteApplication`. La interfaz determina
su ruta inicial desde sesión/rol y completa la integración del primer acceso al
formulario en FT2/FT3. Los roles persistidos son `postulante`, `analista` y
`administrador`. El backend aplica T6; la integración de las pantallas administrativas
continúa en FT15. La sesión consulta el rol vigente; no lo toma del cuerpo, una
cabecera del cliente ni una copia guardada al crear la cookie.

| HTTP | `codigo` | Situación |
|---:|---|---|
| 400 | `cuerpo_invalido` | JSON que no puede decodificarse |
| 401 | `otp_invalido` | OTP incorrecto, usado o expirado |
| 401 | `credenciales_invalidas` | Error de cédula o contraseña en login |
| 401 | `sin_sesion` / `sesion_expirada` | Falta cookie o sesión expirada en `GET /auth/session` |
| 401 | `no_autenticado` / `sesion_invalida` | Rechazo del middleware de rutas protegidas |
| 409 | `postulante_existente` | Cuenta ya registrada |
| 422 | `informacion_invalida` / `password_debil` | Datos de registro o fortaleza inválidos, según operación |
| 429 | `intentos_excedidos` | Bloqueo temporal de intentos |

Los mensajes del RF se mantienen para credenciales y bloqueo; los errores de OTP
se expresan como tales. La confirmación genérica de recuperación no acredita
que el mensaje SMTP se haya entregado.

### 5.3 Autorización administrativa (T6)

`RequireSession` resuelve la cookie; `RequireRole` / `RequirePermission` rechazan
con `401` la ausencia de identidad y con `403 acceso_denegado` un rol sin permiso.
Roles, permisos y recursos administrativos desconocidos se deniegan por defecto.
La misma política se comprueba en los servicios de lectura y asignación de roles.

| Operación / prefijo bajo `/api/v1/admin` | Postulante | Analista TH | Administrador/Comisión |
|---|---|---|---|
| Consultar formularios enviados (`submissions`) | No | Sí | Sí |
| Consultar certificados / emitir descarga (`documents`) | No | Sí | Sí |
| Gestión base de calificaciones (`grades`) | No | Sí | Sí |
| Gestión base de reconsideraciones (`reconsiderations`) | No | Sí | Sí |
| Gestionar fases (`phases`) | No | No | Sí |
| Gestionar banco de preguntas (`question-bank`) | No | No | Sí |
| Cambiar roles (`applicants`) | No | No | Sí |

Los prefijos de fases/preguntas/calificaciones/reconsideraciones tienen política de
acceso preparada, pero sus operaciones de negocio siguen en T18/T20–T22/T25. Esta
matriz expresa el acceso base implementado: no reemplaza los controles de fase,
plazo, expediente ni la distinción entre atención de Talento Humano y resolución
de Comisión que debe aplicar T22. La aprobación funcional corresponde a los
responsables del concurso; no se afirma que ya se haya obtenido.

| Método | Ruta implementada | Respuesta |
|---|---|---|
| `GET` | `/api/v1/admin/submissions/{id}` | `Response` EF-12 con contenido y hash almacenados |
| `GET` | `/api/v1/admin/submissions/{id}/documents` | Lista de metadatos de certificados |
| `GET` | `/api/v1/admin/documents/{id}/download` | `{ "url": "…", "expiresAt": "…" }`, duración de cinco minutos |
| `PATCH` | `/api/v1/admin/applicants/{id}/role` | Perfil seguro `ApplicantResponseDTO` actualizado |

Las lecturas requieren `submitted`, hash y fecha de envío. No enriquecen el snapshot
con datos actuales de la cuenta ni permiten consultar borradores ajenos. La selección
por convocatoria/histórico se implementa en T9; estas rutas resuelven por UUID.
Las rutas habituales `/submissions` y `/documents` mantienen su comprobación de
propietario para todos los roles, sin una excepción administrativa de escritura.

Las respuestas administrativas usan `Cache-Control: no-store`. Cada lectura de
personal autorizado registra actor, rol, acción, UUID del recurso, fecha y resultado
(`permitido`/`rechazado`) en `administrative_access_events`, incluidos recursos
inexistentes y borradores rechazados. Si falla el registro, se responde `500` sin
entregar datos ni URL. El evento de descarga acredita emisión de enlace, no que el
archivo haya sido descargado desde RustFS. Los rechazos previos de sesión/rol usan
el error HTTP habitual. No se almacenan contenido, nombres, cédulas ni URLs firmadas
en estos eventos. T8 incorpora consulta paginada y política de conservación.

El cambio de rol acepta exclusivamente:

```json
{ "rol": "analista", "motivo": "Asignación de revisión del concurso" }
```

El motivo es obligatorio, de hasta 500 caracteres y sin controles. JSON desconocido
o concatenado produce `400`; rol/motivo inválidos, `422 cambio_rol_invalido`; cuenta
inexistente, `404 postulante_no_encontrado`. Se bloquean actor y destino en orden
estable, se vuelve a comprobar que el actor siga siendo administrador y se confirma
el cambio con su historial en una transacción. Si falla el trigger de auditoría,
se revierte el rol. El registro OTP carece de campos para asignar roles y crea
siempre `postulante`.

La migración `00007_authorization.sql` amplía la restricción de roles y crea
`applicant_role_events`. Un trigger registra cambios de rol, también por SQL directo,
con valor anterior/nuevo, actor si existe, operador y motivo. Los dos historiales
rechazan `UPDATE`, `DELETE` y `TRUNCATE`; no dependen de cascadas al borrar cuentas.

Para el alta inicial de un administrador, desde `apps/backend/`, con `DATABASE_URL`
configurada y migraciones aplicadas:

```powershell
go run ./cmd/applicant-role -cedula "CEDULA_EXISTENTE" -role administrador -operator "operador-local" -reason "Alta inicial de Comisión"
```

La CLI es una herramienta operativa con credenciales de base de datos, no una ruta
pública. Exige operador y motivo; no imprime cédulas ni datos de la cuenta.

---

## 6. Contexto `catalog` (T7 implementado)

| Método | Ruta | Descripción | Caché |
|---|---|---|---|
| `GET` | `/api/v1/catalogs/countries` | Lista de países | `catalog:countries` (24 h) |
| `GET` | `/api/v1/catalogs/provinces?country=EC` | Provincias del país | `catalog:provinces:{country}` (24 h) |
| `GET` | `/api/v1/catalogs/cities?country=EC&province=Pichincha` | Cantones de la provincia | `catalog:cities:{country}:{province}` (24 h) |
| `GET` | `/api/v1/catalogs/options/{name}` | Opciones de dominio del EF-12 | `catalog:options:{name}` (24 h) |
| `GET` | `/api/v1/catalogs/provinces/{country}` | Ruta compatible con el cliente existente | Misma clave canónica de provincias |
| `GET` | `/api/v1/catalogs/cities/{country}/{province}` | Ruta compatible con el cliente existente | Misma clave canónica de cantones |

Requieren sesión y admiten los tres roles. Devuelven arrays de nombres bajo las
claves que ya consume `clientes/catalogos.ts`: `paises`, `provincias`, `cantones`.
Las opciones de dominio usan `opciones`. No se exponen IDs internos:

```json
{ "opciones": ["IESS", "BIESS"] }
```

El contrato planificado usaba arrays planos, pero el cliente implementado espera
estas claves y rutas con segmentos. T7 conserva esa compatibilidad y también
publica las rutas con filtros. Ambas variantes comparten servicio, validación y
caché; no hay que cambiar el cliente existente para empezar a consumirlas.

Países: los 194 del RF EF-12 1.2.1, en su orden original. Se eliminó la cabecera de
página que había quedado mezclada con `Bután` en la extracción del documento.
No se ofrece la antigua muestra de 18 países ni la opción inventada `Otro país`.
Provincias/cantones: las 24/221 entradas del catálogo existente, con su relación
persistente y sus etiquetas compatibles con los formularios actuales.

`country` es obligatorio para provincias/cantones, como filtro o segmento. Acepta `EC` o `Ecuador` para
Ecuador; los demás países se indican por su nombre del catálogo, no por un código
ISO supuesto. Se normalizan espacios exteriores y mayúsculas/minúsculas.
`province` es obligatorio para cantones y debe ser una provincia ecuatoriana válida
si el país es Ecuador. Para países extranjeros válidos, provincias/cantones
devuelven listas vacías (`{"provincias":[]}` / `{"cantones":[]}`): esos campos se
capturan manualmente, como exige el RF. La provincia
se convierte a su etiqueta canónica antes de construir la clave de caché.

`name` admite: `generos`, `estadosCiviles`, `respuestas`, `parentescos`,
`instituciones`, `sectores`, `niveles`, `tiposInstitucion`, `areas`,
`tiposExperiencia`, `tiposPublicacion` y `tiposEvento`. Sus valores y orden coinciden
con `submission/definition.json`; no añaden opciones al RF.

PostgreSQL es la fuente persistente y Redis usa TTL de 24 horas. Un miss, error de
lectura/escritura, Redis desactivado o JSON cacheado inválido no bloquea la consulta
a PostgreSQL. Las entradas inválidas se reemplazan si Redis permite escribir.
Una lectura de caché válida evita consultar PostgreSQL para esa lista; una caída
de la base sin entrada cacheada produce un error controlado, sin cachear el fallo.

| HTTP | `codigo` | Situación |
|---:|---|---|
| 400 | `filtro_catalogo_invalido` | Filtros ausentes, desconocidos, duplicados o país/provincia inválidos |
| 401 / 403 | Códigos de sesión / `acceso_denegado` | Sesión o rol no admitidos |
| 404 | `catalogo_no_encontrado` | Nombre de catálogo de dominio desconocido |
| 503 | `catalogo_no_disponible` | Falló la fuente persistente sin lista utilizable en caché |

En las rutas con segmentos no se admiten filtros adicionales, para evitar padres
contradictorios entre ruta y query string.

Los endpoints no usan paginación: entregan listas de selección acotadas completas.
El SQL y las diferencias de cobertura respecto al RF están en
[`../database/SCHEMA.md`](../database/SCHEMA.md). La integración visual sigue en FT5.

---

## 7. Auditoría y contexto `review` (T8/T9)

| Método | Ruta | Descripción |
|---|---|---|
| `GET` | `/api/v1/submissions/{id}/events` | Historial paginado del titular |
| `GET` | `/api/v1/admin/submissions/{id}/events` | Historial administrativo auditado |
| `GET` | `/api/v1/admin/competitions` | Períodos activos y cerrados, paginados |
| `GET` | `/api/v1/admin/submissions?convocatoriaId=&sector=&status=&limit=&cursor=` | Nómina sellada con cursor ligado a filtros |
| `GET` | `/api/v1/admin/submissions/stats?convocatoriaId=&sector=` | Conteos frescos y recomendación de refresco |
| `GET` | `/api/v1/admin/submissions/export.xlsx?convocatoriaId=&sector=` | Matriz técnica completa, sin paginación |
| `GET` | `/api/v1/admin/submissions/export.pdf?convocatoriaId=&sector=` | Misma selección en PDF |

Contrato completo, filtros de legado, formatos, seguridad, límites y propuesta de
columnas/frecuencia: **[REVIEW_AUDIT.md](REVIEW_AUDIT.md)**. Reemplaza las rutas
`/review/*` que sólo estaban planificadas; no se añade un alias que eluda T6.
Las operaciones de fases siguen en T18, no en las lecturas de T9.

---

## 8. Reglas transversales del contrato

1. **Todo cuerpo es JSON** salvo los adjuntos, que usan `multipart/form-data`.
2. **Campos desconocidos: `400`.** El decoder usa `DisallowUnknownFields`, de
   modo que un error de tipeo del frontend se detecta en vez de ignorarse en
   silencio.
3. **Los identificadores son UUIDv7** generados por la aplicación y por PostgreSQL 18.
4. **Las marcas de tiempo son RFC 3339 en UTC** (`2026-04-01T12:00:00Z`); las fechas civiles del formulario son `AAAA-MM-DD`.
5. **`hash` y `submittedAt` solo aparecen cuando la postulación fue enviada**
   (`omitempty`).
6. **Toda respuesta lleva `X-Request-ID`**, incluso las de error.
7. **Ninguna respuesta expone datos internos**: ni SQL, ni rutas del servidor,
   ni el detalle de un panic.

---

## 9. Cliente HTTP del frontend

El frontend consume la API con un cliente único (`src/lib/api.ts`):

- **Rutas relativas** (`/api/v1/...`): el mismo origen funciona en desarrollo
  y en producción.
- **`credentials: 'include'`**: la cookie de sesión viaja en cada petición.
- **`async/await`**: todas las funciones son `async` y lanzan un error
  tipado cuando la respuesta no es `2xx`, leyendo el campo `error.codigo`.
- **Rutas protegidas**: antes de renderizar una vista que exige sesión, se
  consulta `GET /api/v1/auth/session`; si responde `401`, el guard redirige al
  login.

---

**Ver también**: [`../trd/TRD.md`](../trd/TRD.md) ·
[`../database/SCHEMA.md`](../database/SCHEMA.md) ·
[`../architecture/ARQUITECTURA.md`](../architecture/ARQUITECTURA.md)


## Corrección vigente de identidad/notificaciones (2026-10-31)

[IDENTITY_SECURITY.md](IDENTITY_SECURITY.md) actualiza el contrato de OTP:
`desafio_id`/`expira_en`, aislamiento por finalidad y correo, límites, seguridad
no degradable, y estado de cola de correo. Prevalece sobre ejemplos anteriores
que hablen de OTP sin finalidad o afirmen entrega inmediata. La respuesta EF-12
puede incluir `notificacion` fuera del contenido sellado. Las rutas no cambian.


## Copia imprimible y méritos (T19/T20, 2026-10-31)

[MERITS_AND_PRINTING.md](MERITS_AND_PRINTING.md) define la descarga PDF del sello,
el adjunto desde el outbox cifrado, revisión de respaldos, matrices versionadas,
aprobación, cálculos y publicación explícita de resoluciones. No añade puntajes
institucionales por defecto ni habilita un examen desde un cálculo provisional.

## Evaluación y reconsideraciones (T21/T22, 2026-10-31)

[EVALUATION_RECONSIDERATION.md](EVALUATION_RECONSIDERATION.md) define políticas
versionadas y aprobadas, informe psicotécnico, consolidación de fuentes publicadas,
expediente propio/administrativo, respaldos, idempotencia y flujo de reconsideración
con plazos por actor. No permite cargar una nota teórica desde el navegador;
la fuente definitiva debe producirla T27. No incluye valores oficiales por defecto.

## Notificaciones y listas (T23/T24, 2026-10-31)

[NOTIFICATIONS_AND_RESULTS.md](NOTIFICATIONS_AND_RESULTS.md) define consultas de
entrega, reintento administrativo, generación/exportación de listas, controles de
fuentes/reconsideraciones y versiones públicas mínimas con consentimiento.
No añade notas teóricas simuladas ni aprobación de privacidad por defecto.

## Banco, evaluación y privacidad (T25–T28)

[EXAMS_AND_PRIVACY.md](EXAMS_AND_PRIVACY.md) define el productor real de nota T27,
su PDF/adjunto, horarios/intentos/asignación finita y políticas de tratamiento.
Los nuevos envíos requieren aceptar ID/hash de política vigente; no se fabrica
aceptación para el legado ni se precargan textos legales o parámetros académicos.


### Extensiones optativas verificadas para QA sintético

- En un criterio de méritos, `unidadesCompletas: true` redondea la cantidad hacia
  abajo a bloques de `tamanoUnidad` antes del cálculo. Ausente/false conserva el
  cálculo proporcional histórico. No se precarga una duración legal del año.
- En reglas de oposición de modo puntaje, `umbralPsicotecnico` opcional debe estar
  entre cero y el máximo. Si se define, exige puntaje igual/superior **y** una
  conclusión admitida para habilitar avance. Ausente conserva versiones previas;
  no se permite en modo concepto. Usar otra versión aprobada, no editar la vigente.
- Un fallo de escritura del almacén de documentos responde **503** con código
  `almacenamiento_no_disponible`, sin confirmar metadatos ni exponer error remoto.
  La compensación se encola si el borrado inmediato tampoco está disponible.
- `X-Request-ID` sólo conserva UUID válido no nulo; otros valores se reemplazan.
  `Traceparent` identifica una traza local nueva, no garantiza propagación OTLP.
