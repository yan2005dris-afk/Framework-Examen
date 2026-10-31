# ADR-003 — PostgreSQL con sqlc y goose

| Campo | Valor |
|---|---|
| ADR | 003 |
| Estado | **Aceptada** |
| Fecha | 2026-10-30 |
| Ámbito | Backend (persistencia) |
| Referencia | [`../database/SCHEMA.md`](../database/SCHEMA.md) |

---

## Contexto

El sistema necesita persistencia relacional con **transacciones atómicas**
para el guardado multi-tabla de una postulación (cabecera, secciones,
documentos y eventos de auditoría). La rúbrica asigna 8 puntos a
"modelado normalizado y uso riguroso de transacciones".

Además, el despliegue de producción no puede depender de que haya un CLI de
migraciones instalado en la máquina.

## Decisión

- **PostgreSQL 17** como base de datos relacional.
- **pgx v5** como driver, con pool de conexiones y gestor de transacciones
  propio (`internal/platform/db`).
- **sqlc v1.31** para generar código Go tipado desde las queries SQL.
- **goose v3** para las migraciones, **embebidas en el binario** con `embed.FS`.

## Alternativas consideradas

### ORM (GORM, Ent, sqlx)

- ✅ Menos SQL escrito, CRUD rápido.
- ❌ GORM: las consultas complejas se resuelven con el DSL o con SQL crudo de
  todos modos, y el **cursor por keyset** (paginación del dashboard) queda
  antinatural.
- ❌ Ent y sqlx: la generación de código introduce un paso más entre el SQL y
  el modelo.
- ❌ El mapeo objeto-relacional oculta las decisiones del esquema (índices,
  `jsonb`, `CHECK`), que es justo lo que la rúbrica evalúa.

### Migraciones con un CLI externo (golang-migrate, Goose como binario)

- ✅ Familiar.
- ❌ En producción hay que garantizar que la versión del CLI coincide con la
  del código; una desalineación rompe el despliegue.
- ❌ Requiere acceso a la base desde el runner.

### sqlc + goose embebidos (elegida)

- ✅ **El SQL es visible y es la fuente de verdad.** Las queries están en
  `sql/queries/*.sql`, a la vista en la revisión.
- ✅ **Tipado en tiempo de compilación**: si una columna cambia y no se
  regenera, el código no compila. Sin sorpresas en runtime.
- ✅ **Las migraciones viajan dentro del binario**: el despliegue ejecuta el
  mismo artefacto que se probó, sin dependencias externas.
- ✅ `sqlc` genera consultas parametrizadas: no hay concatenación de SQL y no
  hay superficie de inyección.
- ✅ `goose` soporta el par `Up`/`Down`, que permite revertir una migración en
  desarrollo.

## Consecuencias

### Positivas

- La generación es **reproducible y verificable en CI**: el pipeline comprueba
  que el código generado está al día; si alguien edita `sqlcgen/` a mano, el
  pipeline falla.
- El esquema se versiona con migraciones **reversibles** y con datos semilla
  idempotentes (`INSERT ... ON CONFLICT DO NOTHING`).
- `pgx` da acceso directo al protocolo PostgreSQL: pool, conexiones
  preparadas y transacciones reales, sin capa intermedia.
- `jsonb` para el contenido del formulario: el RF evoluciona sin migraciones
  por cada campo nuevo.

### Negativas

- Hay que **regenerar** después de tocar el esquema o las queries:
  `make sqlc-generate` y `make db-schema`. Es un paso que se puede olvidar.
- `sqlc` cubre el 80% de los casos; el SQL más complejo (funciones de
  ventana, `CTE`) puede requerir código a mano en el adaptador.
- El paso de generación añade un requisito de herramientas en el entorno de
  desarrollo (`sqlc` instalado o ejecutado con `go run`).

## Reglas

1. **Una migración = un cambio lógico**, con nombre secuencial
   (`00002_add_password_hash.sql`) y sus bloques `Up` / `Down`.
2. **`schema.sql` se regenera, no se edita**: es la entrada de `sqlc`.
3. **`sqlcgen/` no se edita a mano**: se regenera.
4. Las **consultas siempre parametrizadas** (`$1`, `$2`); nunca se concatena
   entrada del usuario en SQL.
5. Las **transacciones** se abren en el caso de uso mediante `TxManager`, y el
   repositorio recibe el `pgx.Tx` como parámetro.

## Referencias

- Migraciones: `apps/backend/sql/migrations/00001_init.sql`
- Queries: `apps/backend/sql/queries/submissions.sql`
- Configuración de sqlc: `apps/backend/sqlc.yaml`
- Gestor de transacciones: `apps/backend/internal/platform/db/db.go`
- Adaptador: `apps/backend/internal/submission/postgres/repository.go`
- Rúbrica, criterio 2 (8 pts): persistencia relacional y transacciones.
