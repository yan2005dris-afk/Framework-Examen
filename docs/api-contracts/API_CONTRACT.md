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
> `submission` y los endpoints transversales) y **anticipa** el de los contextos
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
- Rutas públicas: `GET /healthz`, `GET /readyz`, `POST /api/v1/auth/register`,
  `POST /api/v1/auth/login`, `POST /api/v1/auth/password/forgot`.

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

### 3.1 `POST /api/v1/submissions`

Crea una postulación en estado `draft`. El contenido se **canoniza** (claves
ordenadas) antes de persistirse.

**Cuerpo**

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `applicantId` | string (UUID) | sí | Postulante dueño de la postulación |
| `code` | string | sí | Código visible; si se omite, la API lo genera (`EF12-AAAA-XXXXXX`) |
| `content` | objeto JSON | sí | Secciones del formulario EF-12 |

```bash
curl -X POST http://localhost:8080/api/v1/submissions \
  -H 'Content-Type: application/json' \
  -d '{
    "applicantId": "11111111-1111-1111-1111-111111111111",
    "code": "EF12-2026-A1B2C3",
    "content": {
      "seccion1": { "nombres": "Ana", "apellidos": "Pérez", "nacionalidad": "ecuatoriana" },
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
    "seccion1": { "apellidos": "Pérez", "nacionalidad": "ecuatoriana", "nombres": "Ana" },
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

Lista las postulaciones de un postulante. **Obligatorio**: `applicantId`.

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
    "seccion5": { "nivel": "superior", "institucion": "Universidad Central del Ecuador" }
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

## 4. Contexto `submission` (planificado)

Estos endpoints completan el EF-12. El formato de petición y de respuesta es
igual al anterior; se marcan como planificados porque el servidor aún no los
expone.

| Método | Ruta | Descripción | Requisito |
|---|---|---|---|
| `POST` | `/api/v1/submissions/{id}/submit` | Sella la postulación: calcula el hash, pasa a `submitted` y notifica por correo | RN-05, RN-06 |
| `POST` | `/api/v1/submissions/{id}/documents` | Adjunta un PDF (multipart) validando MIME, tamaño por tipo y total | HU-07 |
| `GET` | `/api/v1/submissions/{id}/documents` | Lista los adjuntos de la postulación | HU-07 |
| `DELETE` | `/api/v1/documents/{id}` | Elimina un adjunto de una postulación editable | HU-07 |
| `GET` | `/api/v1/submissions/{id}/print` | Vista imprimible del formulario con el espacio de firma | HU-11 |

### 4.1 `POST /api/v1/submissions/{id}/submit`

```json
{
  "id": "044d3694-4ca2-4d87-aa08-ed9e35482710",
  "status": "submitted",
  "hash": "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
  "submittedAt": "2026-04-01T13:30:00Z"
}
```

El hash es el SHA-256 del contenido **canónico**; recalcularlo sobre el
contenido almacenado debe dar el mismo valor.

---

## 5. Contexto `identity` (planificado)

| Método | Ruta | Descripción | Requisito |
|---|---|---|---|
| `POST` | `/api/v1/auth/register` | Registra al postulante validando cédula y fecha de expedición/código dactilar | HU-01, HU-02 |
| `POST` | `/api/v1/auth/login` | Inicia sesión con cédula y contraseña | HU-03 |
| `POST` | `/api/v1/auth/logout` | Cierra la sesión | HU-03 |
| `GET` | `/api/v1/auth/session` | Devuelve la sesión vigente | RNF-03 |
| `POST` | `/api/v1/auth/password/forgot` | Envía el enlace de recuperación por correo | HU-04 |
| `POST` | `/api/v1/auth/password/reset` | Define la contraseña nueva | HU-04 |

### 5.1 `POST /api/v1/auth/login`

```json
{
  "cedula": "1710034008",
  "password": "UnaClaveSegura1!"
}
```

**Respuesta `200`**: cookie `session` + cuerpo con el perfil.

```json
{
  "applicant": {
    "id": "11111111-1111-1111-1111-111111111111",
    "cedula": "1710034008",
    "nombres": "Ana",
    "apellidos": "Pérez",
    "correo": "ana.perez@example.com"
  },
  "mustCompleteApplication": true
}
```

`mustCompleteApplication` es `true` cuando el postulante todavía no ha
enviado ninguna postulación: el frontend lo lleva al formulario (HU-03,
criterio 5).

**Errores**

| HTTP | `codigo` | Causa |
|---:|---|---|
| 400 | `credenciales_invalidas` | Cédula con formato inválido |
| 401 | `credenciales_incorrectas` | Cédula o contraseña incorrectas |
| 429 | `intentos_excedidos` | 3 intentos fallidos: cuenta bloqueada 10 minutos |

> El mensaje al usuario es el del RF: *"sus credenciales de ingreso son
> incorrectas"*. El código `credenciales_incorrectas` es estable para el
> frontend; el mensaje exacto puede cambiar sin romper el contrato.

---

## 6. Contexto `catalog` (planificado)

| Método | Ruta | Descripción | Caché |
|---|---|---|---|
| `GET` | `/api/v1/catalogs/countries` | Lista de países | `catalog:countries` (24 h) |
| `GET` | `/api/v1/catalogs/provinces?country=EC` | Provincias del país | `catalog:provinces:{country}` (24 h) |
| `GET` | `/api/v1/catalogs/cities?country=EC&province=Pichincha` | Cantones de la provincia | `catalog:cities:{country}:{province}` (24 h) |

Todas devuelven un array JSON plano y se sirven desde Redis cuando la caché
está tibia.

---

## 7. Contexto `review` (planificado)

| Método | Ruta | Descripción |
|---|---|---|
| `GET` | `/api/v1/review/submissions?status=&sector=&limit=&cursor=` | Listado para revisión (paginación por cursor) |
| `GET` | `/api/v1/review/stats` | Conteos por cargo y por estado para el dashboard |
| `POST` | `/api/v1/review/submissions/{id}/phase` | Habilita una fase del concurso |

---

## 8. Reglas transversales del contrato

1. **Todo cuerpo es JSON** salvo los adjuntos, que usan `multipart/form-data`.
2. **Campos desconocidos: `400`.** El decoder usa `DisallowUnknownFields`, de
   modo que un error de tipeo del frontend se detecta en vez de ignorarse en
   silencio.
3. **Los identificadores son UUID v4** generados por la aplicación.
4. **Las fechas son RFC 3339 en UTC** (`2026-04-01T12:00:00Z`).
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
