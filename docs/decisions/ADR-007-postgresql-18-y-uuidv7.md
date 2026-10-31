# ADR-007 — PostgreSQL 18 y UUIDv7 como estrategia de identificadores

| Campo | Valor |
|---|---|
| ADR | 007 |
| Estado | **Aceptada** |
| Fecha | 2026-10-31 |
| Ámbito | Backend (persistencia) y datos |
| Referencia | [`../trd/TRD.md` §4](../trd/TRD.md) · [`../database/SCHEMA.md`](../database/SCHEMA.md) |

---

## Contexto

El motor de datos era PostgreSQL 17 con identificadores **UUID v4**, que
`uuid.New()` produce de forma aleatoria. Eso tiene dos efectos conocidos sobre
una tabla con clave primaria `UUID`:

1. **Fragmentación del índice.** El valor aleatorio no tiene correlación con
   la posición física de la fila, así que cada inserción en el B-tree del
   PRIMARY KEY cae en una página distinta. Con millones de filas eso produce
   divisiones de página, escrituras ampificadas y una caché menos efectiva.
2. **Desempate arbitrario en la paginación por cursor.** La clave de orden es
   `(updated_at DESC, id DESC)`. Con UUID v4, el `id` es un valor aleatorio:
   entre dos filas con la misma `updated_at`, el orden no significa nada.

PostgreSQL 18 (publicado el 25 de septiembre de 2025) trae `uuidv7()` de forma
nativa: un UUID cuyo encabezado de 48 bits es una marca de tiempo en
milisegundos, con el resto aleatorio. La versión 19 estaba en *release
candidate* al tomar esta decisión, así que 18 es la estable vigente.

El proyecto también arrastra una consecuencia operativa: **las imágenes
Docker de PostgreSQL 18 cambiaron el layout del directorio de datos**. Ya no
es `/var/lib/postgresql/data`, sino `/var/lib/postgresql/<major>/<cluster>`, y
el montaje del volumen debe hacerse en `/var/lib/postgresql`. Sin ese cambio,
la imagen se niega a arrancar y avisa de que parece un intento de actualizar
la imagen sin actualizar la base.

## Decisión

1. **El motor es PostgreSQL 18** (`postgres:18-alpine` en desarrollo y en
   producción), con el volumen montado en `/var/lib/postgresql`.
2. **Los identificadores son UUIDv7.** La aplicación los genera con
   `uuid.NewV7()` de `github.com/google/uuid`, que ya estaba en la
   dependencias: no se añade ninguna librería.
3. **La base declara el mismo valor por defecto**: `DEFAULT uuidv7()` en las
   columnas `id`. Es la red de seguridad para los caminos de inserción que no
   fijan el identificador (semillas, importaciones, SQL manual).
4. **El índice de listado por postulante incluye el identificador**:
   `(applicant_id, updated_at DESC, id DESC)`. Con UUIDv7 ordenados en el
   tiempo, el desempate del cursor deja de ser arbitrario y se resuelve
   dentro del índice.

## Alternativas consideradas

### UUID v4 con PostgreSQL 17 (el estado anterior)

- ✅ Sin cambios de motor, sin procedimiento de actualización de la base.
- ❌ Fragmentación del índice del PRIMARY KEY.
- ❌ Desempate de la paginación por cursor sin significado.
- ❌ Sin `uuid_extract_timestamp()`: la marca de tiempo de creación no se
  puede recuperar del identificador.

### UUIDv7 generado por la aplicación, sin tocar el motor

- ✅ El beneficio de indexación se consigue igual.
- ❌ La versión 18 seguiría sin usarse: se pierde la función nativa y el
  `DEFAULT uuidv7()` de la base.
- ❌ Un despliegue con una base administrada fuera del proyecto quedaría
  atado a la versión que soporte la función.

### Identificadores `bigint` con secuencia

- ✅ Es lo más compacto y lo más rápido de indexar.
- ❌ El identificador **se puede adivinar**: un postulante vería el de otro
  con solo incrementarlo.
- ❌ Obliga a exponer el identificador como un dato sensible y a gestionar un
  segundo plano de seguridad para lo que debería ser un identificador opaco.

### Identificadores generados solo por la base (`DEFAULT uuidv7()`)

- ✅ Una sola fuente de verdad.
- ❌ El dominio no puede conocer su propio identificador antes de
  persistir, lo que rompe la independencia del dominio: una entidad con
  identificador vacío es una entidad incompleta.
- ❌ La arquitectura del proyecto ([ADR-002](ADR-002-bounded-contexts-y-hexagonal.md))
  exige que el dominio se pueda probar sin base de datos.

### PostgreSQL 18 con UUIDv7 en ambos lados (elegida)

- ✅ La base y la aplicación hablan el mismo formato, sin divergencia.
- ✅ El dominio conserva su identificador desde el primer momento: la
  entidad se prueba sin base de datos.
- ✅ La base sigue siendo la autoridad para cualquier inserción que no fije
  el identificador.
- ✅ El `DEFAULT` cubre el caso de la importación masiva sin cambiar el
  contrato del dominio.

## Consecuencias

### Positivas

- El índice del PRIMARY KEY se mantiene compacto: las inserciones nuevas caen
  al final del B-tree.
- La clave de orden `(updated_at, id)` es **consistente con el orden de
  creación**, lo que hace que la paginación por cursor sea estable y
  resoluble por índice.
- El identificador **lleva dentro la marca de tiempo de creación**: se puede
  recuperar con `uuid_extract_timestamp(id)` sin una columna adicional, lo
  que sirve para diagnóstico y para particionado futuro.
- `uuidv7()` es nativo: no hay que calcularlo a mano ni depender de una
  extensión.

### Negativas

- **La migración 00002 exige PostgreSQL 18.** Aplicarla sobre 17 falla con
  `function uuidv7() does not exist`. El motor se actualiza **antes** de
  correr las migraciones; el procedimiento está en
  [`../deployment/PRODUCTION_DEPLOYMENT.md`](../deployment/PRODUCTION_DEPLOYMENT.md).
- **La actualización de la base mayor requiere un procedimiento**: un
  directorio de datos de PostgreSQL 17 no lo abre un PostgreSQL 18. Hay que
  usar `pg_upgrade` o un volcado lógico.
- **El layout del volumen cambió**: el montaje va en `/var/lib/postgresql`.
  Un despliegue existente que montaba en `.../data` debe migrar el volumen o
  empezar con uno nuevo.
- El UUIDv7 **revela la marca de tiempo de creación** de una fila. No es un
  dato sensible en este dominio (la cédula y el nombre lo son, y no están en
  el identificador), pero conviene saberlo.
- `github.com/google/uuid` no expone un comparador ni un extractor de
  marca de tiempo para UUIDv7: el test del orden y del timestamp los
  implementa el propio proyecto sobre los bytes del UUID.

## Verificación

```sql
-- La versión 7 está en el nibble de versión del UUID
SELECT substring(id::text, 15, 1) FROM submissions LIMIT 1;   -- 7

-- La marca de tiempo de creación coincide con created_at
SELECT uuid_extract_timestamp(id) = created_at FROM submissions;
```

## Referencias

- Código: `apps/backend/internal/submission/service.go` (`newIdentifier`)
- Test: `apps/backend/internal/submission/identifier_test.go`
- Migración: `apps/backend/sql/migrations/00002_uuidv7.sql`
- Composición: `compose.yaml`, `deploy/compose.yaml`
- [PostgreSQL 18 — notas de la versión](https://www.postgresql.org/docs/18/release-18.html)
- [Tipos UUID en PostgreSQL 18](https://www.postgresql.org/docs/18/datatype-uuid.html)
