# ADR-005 — Paginación por offset y por cursor sobre un kernel compartido

| Campo | Valor |
|---|---|
| ADR | 005 |
| Estado | **Aceptada** |
| Fecha | 2026-10-30 |
| Ámbito | Backend (API) |
| Referencia | [`../api-contracts/API_CONTRACT.md` §1.5](../api-contracts/API_CONTRACT.md) |

---

## Contexto

El sistema tiene dos tipos de listados con requisitos opuestos:

1. **"Mis postulaciones"**: la lista personal del postulante. Es corta (una o
   pocas postulaciones), el usuario quiere ver páginas numeradas y el total.
2. **Dashboard de revisión**: el listado del analista de Talento Humano sobre
   **todas** las postulaciones del concurso. Con la Disposición Transitoria
   Primera (30 días) y un concurso nacional, el volumen puede llegar a
   **decenas o cientos de miles de filas**.

El `OFFSET` degrada: `OFFSET 100000` obliga a PostgreSQL a leer y descartar
100 000 filas. Y el `count(*)` sobre toda la tabla, en cada página, es un
escaneo completo que se encarece con el tiempo.

## Decisión

Se implementan **dos estrategias de paginación sobre un kernel compartido**,
con un contrato HTTP distinto para cada una:

| Estrategia | Parámetros | Respuesta | Uso |
|---|---|---|---|
| **Offset** | `page`, `pageSize` | `{ items, page, pageSize, total }` | Listados acotados |
| **Cursor (keyset)** | `limit`, `cursor` | `{ items, nextCursor, hasMore, limit }` | Listados grandes |

El kernel vive en `internal/shared/domain/pagination.go` y los parsers en
`internal/platform/httpx/pagination.go`: **ningún handler parsea la query
string a mano**.

### Reglas

1. **Límite duro de 100 elementos por página** (`MaxPageSize`), en ambas
   estrategias. Por defecto son 20 (`DefaultPageSize`).
2. **El cursor es opaco**: es base64url de la clave de orden del contexto. El
   cliente lo reenvía, no lo interpreta. Si el contexto cambia su clave de
   orden, el cursor cambia de forma incompatible y el cliente debe empezar
   desde el inicio.
3. **La clave de orden vive en el contexto**, no en el kernel. El kernel solo
   sabe codificar y decodificar una secuencia de valores.
4. **El modo se elige por los parámetros recibidos**: si llega `limit` o
   `cursor`, se usa cursor; si no, se usa offset.
5. Los parámetros no numéricos son **error del cliente** (`400`); los valores
   fuera de rango se ajustan al límite.

### Clave de orden del contexto `submission`

`(updated_at DESC, id DESC)`. El `id` desempata las filas con la misma
fecha, de modo que la paginación **nunca omite ni repite** registros:

```sql
WHERE applicant_id = $1
  AND (updated_at < $2 OR (updated_at = $2 AND id < $3))
ORDER BY updated_at DESC, id DESC
LIMIT $4
```

## Alternativas consideradas

### Solo offset

- ✅ Simple, una sola ruta de código.
- ❌ Se degrada con el volumen del dashboard: `OFFSET` alto + `count(*)` en
  cada página.
- ❌ Al insertar o borrar filas, las páginas se desplazan y el usuario ve
  registros repetidos o saltados.

### Solo cursor

- ✅ Escala sin degradación.
- ❌ Pierde el "ir a la página 7" y el total, que el usuario espera en una
  lista personal.
- ❌ No sirve para los controles de paginación por número que usan los
  componentes de tabla del frontend.

### Cargar todo y paginar en memoria

- ✅ Trivial de implementar.
- ❌ Carga cientos de miles de filas en memoria por petición; inviable con el
  volumen del concurso.

### Ambas, con un kernel común (elegida)

- ✅ Cada listado usa la estrategia que le corresponde.
- ✅ Un solo kernel: los parsers, los límites y las respuestas se comportan
  igual en todos los contextos.
- ✅ La transición es natural: un contexto que crece pasa de offset a cursor
  sin cambiar el kernel.

## Consecuencias

### Positivas

- El dashboard no degrada con cientos de miles de postulaciones (RNF-02).
- El cliente tiene una API simple en ambos casos y nunca necesita construir
  un `WHERE` complejo.
- El límite duro de 100 protege la base de un `pageSize=100000` malicioso o
  descuidado.
- El cursor opaco mantiene la clave de orden **dentro del backend**: si
  mañana cambia a `(sector, updated_at)`, el contrato HTTP no cambia.

### Negativas

- **Dos formas de respuesta**: el frontend necesita saber cuál usar. Se
  resuelve por la presencia de `limit`/`cursor`, que es explícita.
- El cursor **no es estable frente a cambios en la clave de orden**: si el
  contexto cambia su orden, los cursores emitidos antes dejan de ser válidos.
- El total (`count(*)`) sigue siendo caro en la estrategia offset. Se acepta
  porque esa estrategia se reserva para listados acotados.
- El cursor no permite saltar a una página arbitraria. Es el precio de la
  escalabilidad, y es aceptable en un listado de revisión.

## Referencias

- Kernel: `apps/backend/internal/shared/domain/pagination.go`
- Parsers: `apps/backend/internal/platform/httpx/pagination.go`
- Queries keyset: `apps/backend/sql/queries/submissions.sql`
- Contrato HTTP: [`../api-contracts/API_CONTRACT.md`](../api-contracts/API_CONTRACT.md)
- RNF-01 y RNF-02 del PRD.
