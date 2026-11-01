# AGENTS.md

**Framework-Examen** — portal de postulación al concurso BIESS/IESS (Ecuador):
registro, login, formulario de postulación en 9 secciones (EF-12), revisión, envío con
hash criptográfico, impresión y dashboard. Monorepo pnpm: SPA React + API Go.

Este archivo es la definición del proyecto para cualquier agente de IA: léelo antes de
tocar código y actualízalo cuando cambie una convención.

## Estructura

- `apps/frontend/` — SPA: React 19 + Vite 8 + Tailwind v4 + shadcn/ui
- `apps/backend/` — API: Go 1.27 (chi + pgx/Postgres)
- `docs/mockups/` — set de mockups HTML estático (36 pantallas) que es la **referencia
  de diseño y contrato de migración** a shadcn/ui. Se versiona. Los nombres de clase
  de `docs/mockups/assets/app.css` son el contrato: cada clase corresponde a un
  componente de shadcn. Guía de migración en `docs/mockups/README.md`. Los datos de
  negocio del expediente de ejemplo viven en `docs/mockups/fixtures/` y se validan
  con `node docs/mockups/scripts/check-consistencia.mjs` (datos y enlaces) y
  `node docs/mockups/scripts/check-contrato.mjs` (clases, etiquetas y a11y).

## Stack

### Frontend

- React 19 + TypeScript 6 + Vite 8
- Tailwind CSS 4 (plugin `@tailwindcss/vite`, sin `tailwind.config.js`)
- shadcn/ui — estilo `radix-nova`, base color `neutral`, iconos Lucide
- Radix UI, `cmdk` (command palette), `react-day-picker`, `recharts`, `sonner` (toasts),
  `next-themes` (tema claro/oscuro), `cn` (merge de clases Tailwind)
- **Vitest 5** (tests unitarios) + **Prettier 3** (formato; ignora `src/components/ui/`)
- pnpm 12 como package manager (workspace) — lockfile `pnpm-lock.yaml`

### Backend

- Go 1.27 — `chi` v5 (routing), `pgx` v5 (Postgres), `uuid`
- PostgreSQL 18 — los identificadores son **UUIDv7** (`uuid.NewV7()` en Go,
  `DEFAULT uuidv7()` en la base). La imagen Docker guarda los datos en
  `/var/lib/postgresql/<major>/<cluster>`: el volumen se monta en
  `/var/lib/postgresql`, no en `.../data`
- `sqlc` v1.31 (SQL → Go tipado) + `goose` v3 (migraciones embebidas en el binario)
- Módulo Go **independiente**: `apps/backend/` no es paquete pnpm, no lo instales con pnpm
- Config por entorno: `PORT`, `APP_ENV`, `APP_URL`, `DATABASE_URL`, `DB_REQUIRED`,
  `STORAGE_DIR`, `CORS_ALLOWED_ORIGINS`, `SESSION_TTL`, `MAX_REQUEST_BODY_BYTES`, `LOG_LEVEL`
- Identidad: registro y recuperación mediante **OTP por correo** (`otp_correo`),
  cédula por módulo 10. No solicitar fecha de expedición ni código dactilar ni
  presentar los datos como verificados por Registro Civil. Ver ADR-008.
- Correo por SMTP: `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD`, `SMTP_FROM`, `SMTP_TLS`
  (en desarrollo apunta a Mailpit)

## Setup commands

Desde la raíz del repo:

- Instalar deps: `pnpm install`
- Frontend dev:   `pnpm dev`      (puerto 5173)
- Frontend build: `pnpm build`
- Frontend lint:  `pnpm lint`
- Frontend test:  `pnpm test`     (Vitest)
- Frontend formato: `pnpm format` / `pnpm format:check`
- Backend build:  `cd apps/backend && go build ./...`
- Backend tests:  `cd apps/backend && go test ./...`
- Stack completo: `docker compose up -d` — api, frontend, postgres y mailpit
- Mockups (datos y enlaces):  `node docs/mockups/scripts/check-consistencia.mjs`
- Mockups (contrato y a11y):   `node docs/mockups/scripts/check-contrato.mjs`

El compose de desarrollo vive en `compose.yaml` (raíz). El de producción en
`deploy/compose.yaml` y lee sus variables de `deploy/.env`:

```bash
docker compose -f deploy/compose.yaml up -d
```

Desde `apps/frontend/` también funcionan `pnpm dev`, `pnpm build`, `pnpm lint`, `pnpm preview`.

### Backend (desde `apps/backend/`)

- `make help` — lista los targets disponibles
- `make run` — API en desarrollo (puerto 8080)
- `go run ./cmd/api storage` — servidor local de objetos (8081); comparte `STORAGE_DIR`, `STORAGE_PUBLIC_URL` y `STORAGE_SIGNING_KEY` con la API
- `make docker-up` / `make docker-down` — PostgreSQL local en Docker
- `make db-migrate` / `make db-status` / `make db-down` / `make db-reset` — migraciones goose
- `make sqlc-generate` — regenera `internal/platform/sqlcgen` desde `sql/queries`
- `make db-schema` — regenera `sql/queries/schema.sql` desde las migraciones
- `make catalog-seed` — regenera la migración `00008` y `sql/seeds/catalogs.sql` desde la definición/geografía EF-12 (no reescribir migraciones ya desplegadas)
- `make test` / `make vet` / `make fmt` / `make tidy`

### Componentes shadcn

```bash
pnpm dlx shadcn@latest add <componente>          # agregar
pnpm dlx shadcn@latest add <componente> --overwrite  # regenerar sobreescribiendo
```

`components.json` es la fuente de verdad de aliases, estilo y variables CSS.

## Project layout

`apps/frontend/src/`:

- `components/ui/` — componentes shadcn. **No editar a mano**: se regeneran con el CLI.
  Es una hoja: no importa features ni vistas
- `components/` — componentes propios (`app-sidebar.tsx`, `app-header.tsx`,
  `theme-toggle.tsx`, `theme-provider.tsx`, `guard-sesion.tsx`)
- `features/<dominio>/` — **una feature por dominio de negocio**. Cada feature
  agrupa lo suyo: `modelo.ts` (lógica pura del dominio), `validacion.ts`
  (validación pura del dominio), `queries.ts` (TanStack Query), `constantes.ts`
  y la UI. Los tests van al lado del módulo
- `layouts/` — guard por layout (`layout-admin`, `layout-postulante`,
  `layout-publico`)
- `pages/` — vistas (formularios y pantallas de ruta)
- `lib/` — transporte e infraestructura: `api.ts`, `api-error.ts`, `config.ts`,
  `rutas.ts`, `query-client.ts`, `query-provider.tsx`, `uuid.ts` y
  `lib/clientes/<dominio>.ts`. **Cero reglas de negocio**
- `utils/` — utilidades genéricas y copy-pasteable entre proyectos
  (`validators.ts`: cédula módulo 10, fortaleza de contraseña, email;
  `fechas.ts`: zona America/Guayaquil)
- `helpers/` — lógica de este portal compartida entre dos o más features
  (`experiencia.ts`: duración de la experiencia laboral EF-12)
- `types/` — tipos compartidos (`api.ts`, `concurso.ts`, `convocatoria.ts`)
- `data/` — `ecuadorGeo.ts` (provincias/cantones), `mockInitialData.ts`
- `hooks/use-mobile.ts`
- `assets/images/` — imágenes

**Dónde va la lógica pura y las validaciones** (regla de tres preguntas):

1. ¿Es puro? (sin React, sin `fetch`, sin estado, sin DOM) → si no, es hook,
   query o componente
2. ¿Habla el vocabulario de un dominio? → `features/<dominio>/modelo.ts`, o
   `features/<dominio>/validacion.ts` si su trabajo es rechazar datos
3. ¿No habla vocabulario de dominio y lo usan dos o más features? → `helpers/`;
   si además es genérico y reutilizable entre proyectos → `utils/`

El mensaje de error viaja con la validación (no se escribe en el JSX). Si una
feature necesita lógica de otra, esa lógica sube a la capa compartida: no se
importa de feature a feature.

`eslint.config.js` hace cumplir estas fronteras con `no-restricted-imports`:
las features no importan features ajenas (salvo `auth`, que es un primitivo
transversal, y los módulos compartidos ya listados), las capas compartidas no
dependen de features ni de vistas, y `components/ui` es una hoja.

`apps/backend/`:

- `cmd/api/` — binario de la API: cableado de contextos + apagado ordenado
- `cmd/migrate/` — runner de migraciones (goose embebido en `sql/migrations`)
- `internal/platform/` — transversal: `config`, `db`, `httpx`, `logger`, `mail`, `storage`, `sqlcgen`
- `internal/shared/domain/` — kernel compartido: errores base y paginación
- `internal/<contexto>/` — **un bounded context por carpeta** (screaming architecture):
  `identity/`, `submission/`, `document/`, …
- `internal/<contexto>/<capacidad>/` — cada capacidad trae `service.go` (casos de uso),
  `handler.go` (adaptador HTTP), `repository.go` (puerto), `dto.go`, `errors.go` y
  `postgres/` (adaptador de persistencia)
- `sql/queries/` — esquema (`schema.sql`) y queries de sqlc
- `sql/migrations/` — migraciones goose embebidas

Ejemplo de referencia completo: `internal/submission/` (CRUD de postulaciones).

Raíz del repo:

- `compose.yaml` — compose de **desarrollo**: api, frontend (Vite), postgres y mailpit
- `deploy/compose.yaml` — compose de **producción** (lee `deploy/.env`)
- `deploy/nginx.conf` — reverse proxy de producción: sirve la SPA y reenvía `/api`
- `apps/frontend/Dockerfile` — etapas `dev` (Vite) y `prod` (nginx)
- `.github/workflows/ci.yml` — CI: formato, lint y tests de frontend y backend
- `docs/` — documentación del proyecto: `prd/`, `trd/`, `architecture/`,
  `database/`, `api-contracts/`, `decisions/`, `design/`, `deployment/` y `tasks/`.
  El índice es `docs/README.md`; la fuente funcional del RF está en
  `docs/requerimiento.txt`

## Code style

- Comillas dobles, sin punto y coma, indentación de 2 espacios (formato del preset nova)
- TypeScript estricto: sin `any`, tipos explícitos en props y estado
- `verbatimModuleSyntax` activo → los imports solo de tipo usan `import type`
- `noUnusedLocals` y `noUnusedParameters` activos
- Alias `@/` → `apps/frontend/src/`
- **El dominio usa español en los identificadores** (`esValida`, `calcularDuracionExperiencia`,
  `seccion1`, `cumpleMayuscula`) — mantener ese idioma en lo nuevo
- Tailwind para estilos: clases utilitarias, sin CSS en línea salvo valores dinámicos
- Comentarios JSDoc en las funciones de `src/utils/`, `src/helpers/` y
  `src/features/*/modelo.ts` / `validacion.ts`
- **Tests en `src/**/*.test.ts` con Vitest** (entorno node para lógica pura).
  `src/components/ui/` se excluye de Prettier: lo regenera el CLI de shadcn
- En desarrollo el frontend habla con la API por el proxy de Vite (`/api` →
  `VITE_DEV_API_TARGET`): un solo origen, sin CORS. En producción hace lo mismo
  nginx (`deploy/nginx.conf`)

### Go (backend)

- **Identificadores en inglés** (funciones, métodos, tipos, campos) y **comentarios y
  mensajes al usuario en español**, con bloques JSDoc (`/** … */` + `@param`/`@returns`)
- Archivos por responsabilidad, sin `package` genérico: `service.go`, `handler.go`,
  `repository.go`, `dto.go`, `errors.go`
- **El dominio no importa infraestructura**: los puertos (interfaces) viven junto al
  dominio y los adaptadores de persistencia en `postgres/`
- Sin `any` implícito ni panics en la capa HTTP: los errores de dominio se traducen a
  respuestas JSON con código estable
- **Paginación reutilizable** en `internal/shared/domain/pagination.go`, con dos
  estrategias sobre un mismo kernel:
  - `PageRequest` / `Page[T]` — offset (`page`, `pageSize`), para listados acotados
  - `CursorRequest` / `CursorPage[T]` — cursor por keyset (`limit`, `cursor`), para
    listados grandes donde `OFFSET` y el `count(*)` se degradan
  - Los parsers están en `internal/platform/httpx/pagination.go`: **ningún handler
    parsea la query string a mano**. Límite duro de 100 elementos por página y el
    cursor es opaco (el cliente no lo interpreta)
- `gofmt` obligatorio; `go vet ./...` y `go test ./...` en verde antes de la PR

### Gotchas verificados

- **TypeScript 7 no es compatible con typescript-eslint.** El proyecto fija
  `typescript@~6.0.3`. No subir a 7.x hasta que typescript-eslint dé soporte
  ([issue #10940](https://github.com/typescript-eslint/typescript-eslint/issues/10940)).
- **`baseUrl` fue eliminado en TypeScript 6+.** Los `paths` van **sin** `baseUrl`, y deben
  estar en `tsconfig.json` **y** `tsconfig.app.json`. El CLI de shadcn lee el
  `tsconfig.json` raíz: si el alias no está ahí, escribe los componentes en una carpeta
  literal llamada `@/`.
- **ESLint** desactiva dos reglas solo para `src/components/ui/**` y `src/hooks/**`
  (`react-refresh/only-export-components`, `react-hooks/set-state-in-effect`), porque el
  código que genera shadcn no cumple las reglas nuevas de React Compiler. El resto del
  proyecto sí las cumple. La única excepción propia: los módulos `src/features/**/*-context.tsx`
  pueden exportar su hook de consumo junto al proveedor, porque la regla se configura con
  `allowExportNames` (al crear otro contexto, agregá el hook a esa lista en vez de apagar la regla).

## Testing instructions

- El frontend usa Vitest para lógica pura y casos de duración compartidos con Go (`internal/submission/testdata/duration_cases.json`).
- Antes de abrir una PR: `pnpm lint` y `pnpm build` deben pasar en verde.
- La lógica de negocio va acompañada de tests en su módulo: Vitest en frontend,
  `_test.go` en backend (`go test ./...`).
- Las reglas puras del backend se prueban sin base de datos: `internal/identity/`
  (cédula módulo 10) y `internal/submission/` (contenido canónico, estados, hash).

## Flujo de trabajo (git flow)

`main` es la rama de producción, `develop` es la de integración. **Todo el trabajo diario
ocurre sobre `develop`.**

| Rama | Propósito | Nace desde |
|---|---|---|
| `main` | Código en producción. Solo recibe merges de `release/*` y `hotfix/*`. | — |
| `develop` | Rama de trabajo por defecto. | `main` |
| `feature/*` | Nueva funcionalidad. | `develop` |
| `release/*` | Preparación de release. | `develop` |
| `hotfix/*` | Corrección urgente en producción. | `main` |
| `support/*` | Mantenimiento de versiones antiguas. | `main` |

Prefijos configurados en `.git/config` (`gitflow.prefix.*`): `feature/`, `release/`,
`hotfix/`, `support/`. El prefijo de tags de versión está **vacío**: los tags se crean
como `0.1.0`, no `v0.1.0`.

```bash
git flow feature start <nombre>    # nueva feature desde develop
git flow feature finish <nombre>   # fusiona a develop y borra la rama
git flow release start <version>   # desde develop
git flow release finish <version>  # fusiona a main + tag, y release/* -> develop
git flow hotfix start <version>    # desde main
git flow hotfix finish <version>   # fusiona a main + tag, y a develop
```

### Reglas de ramas

- **`develop` es la rama de trabajo**: los cambios se comitean directamente sobre ella.
- `main` solo cambia por `release/*` o `hotfix/*`.
- Las ramas `feature/*` se crean **desde `develop`** cuando convenga aislar un trabajo
  grande, pero no son obligatorias.
- Antes de comitear: `git pull origin develop` para estar al día.

## Convenciones de commits

- [Conventional Commits](https://www.conventionalcommits.org/): `feat:`, `fix:`, `docs:`,
  `refactor:`, `chore:`, `test:`. Scope opcional: `feat(frontend): ...`.
- Asunto en imperativo, ≤ 72 caracteres, sin punto final.
- Un commit = un cambio lógico. No mezcles refactor con funcionalidad.

## PR & commit conventions

- Rama de PR: `feature/<descripcion-corta>` en kebab-case, 3-4 palabras máximo.
- PR hacia `develop`. Hacia `main` solo las de release/hotfix.
- Título de PR con conventional commits, igual que los commits.
- Abrir con `gh pr create` una vez que lint y build estén en verde.

## Security

- Nunca commitear secretos: `.env*` está en `.gitignore` y solo `.env.example` se versiona.
- `GEMINI_API_KEY`, `APP_URL` y `DATABASE_URL` se inyectan en runtime — no hardcodearlos.
- La app maneja **datos personales sensibles** (cédula, nombres, documentos PDF de
  postulantes). No loguearlos, no mandarlos a servicios externos sin necesidad, y no
  los subas a archivos versionados.

### Qué nunca se versiona

Definido en `.gitignore`:

- `node_modules/`, `dist/`, `coverage/`, `*.tsbuildinfo`
- Lockfiles de otros gestores (`package-lock.json`, `yarn.lock`, `bun.lock`).
  **Sí** se versiona `pnpm-lock.yaml`, que es el lockfile de este proyecto.
- Secretos: `.env*` (salvo `.env.example`), `*.pem`, `*.key`, `*.p12`
- Documentos y adjuntos de postulantes: `*.pdf`, `*.doc(x)`, `*.xls(x)`, `*.ppt(x)`,
  `apps/backend/data/`, `uploads/`, `storage/`
- `/mockups/` (raíz) — el mockup original en React ya no está en el repo; la regla de
  `.gitignore` se conserva como red de seguridad para que una copia local nunca se
  versione por accidente. El set vigente es `docs/mockups/`.

## Decisiones

Bitácora de decisiones del proyecto. **Actualiza esta sección** cuando se tome una
decisión que afecte cómo se trabaja en el repo.

| Fecha | Decisión | Motivo |
|---|---|---|
| 2026-10-30 | git flow con `main` / `develop` | El proyecto se trabaja sobre `develop`; `main` queda para producción. |
| 2026-10-30 | `main` como rama de producción (no `master`) | Convenciones actuales del ecosistema. |
| 2026-10-30 | pnpm como package manager (workspace `apps/*`) | Lockfile único para frontend; Bun queda solo en `mockups/`. |
| 2026-10-30 | shadcn/ui con preset `radix-nova` y primitivas Radix | Alineado con shadcn-admin; base neutral + iconos Lucide. |
| 2026-10-30 | TypeScript fijado en `~6.0.3` | typescript-eslint no soporta TS 7.0; con ESLint activo la 6.x es la última compatible. |
| 2026-10-30 | `mockups/` se conserva como referencia temporal | Fuente de las 9 vistas y del copy pendientes de migrar a shadcn/ui. |
| 2026-10-31 | El set de mockups se versiona en `docs/mockups/` (HTML estático) y el `mockups/` de la raíz se elimina | El set nuevo reemplaza al mockup original de React: sin build ni dependencias, versionable y con las clases de `app.css` como contrato de migración a shadcn/ui. El mockup original ya no aporta nada y se iba a borrar igual. |
| 2026-10-30 | Trabajo directo sobre `develop`, sin rama obligatoria | El equipo integra directo; `feature/*` queda como opción para trabajos grandes. |
| 2026-10-30 | Backend en Go con `chi` v5 (en vez de Gin o Fiber) | chi opera sobre `*http.Request` nativo: interoperable con la stdlib y con `multipart`, clave para los PDF adjuntos; Fiber obliga a `fasthttp.RequestCtx` y a cargar el cuerpo entero en memoria. |
| 2026-10-30 | Backend organizado por bounded contexts (`internal/<contexto>/<capacidad>`) con `platform/` transversal | Screaming architecture: las carpetas dicen qué hace el sistema. Referencia: `CotizacionCamisas/apps/backend`. |
| 2026-10-30 | `sqlc` para el acceso a datos y `goose` para migraciones embebidas | SQL visible y tipado; las migraciones viajan dentro del binario, sin depender del CLI en producción. |
| 2026-10-30 | En Go: identificadores en inglés y comentarios en español con JSDoc | Código legible para el equipo sin perder el idioma del dominio. |
| 2026-10-30 | El hash del envío se calcula sobre el JSON canónico (claves ordenadas) | El mismo formulario debe producir el mismo hash sin importar el orden de las claves. |
| 2026-10-30 | Paginación con dos estrategias sobre un kernel compartido (offset y cursor por keyset) | Offset para listados acotados ("mis postulaciones") y cursor para el dashboard de revisión, donde el `OFFSET` y el `count(*)` se degradan con cientos de miles de postulaciones. El cursor es opaco y la clave de orden vive en el contexto, no en el kernel. |

| 2026-10-31 | EF-12 tipado con JSON compatible con frontend, guardado parcial por sección y sello con bloqueo de fila | Los borradores admiten campos pendientes; el envío exige las nueve secciones y cinco consentimientos. `SELECT FOR UPDATE` serializa cambios y envío; el hash canoniza objetos anidados. |
| 2026-10-31 | Confirmación SMTP posterior al commit del envío | Un fallo de correo no deshace un sello confirmado. El destinatario es la cuenta registrada; los reintentos persistentes quedan para T15. |

| 2026-10-31 | Documentos PDF con cuota por postulante y referencia confirmada en la misma transacción | Bloquear postulante y luego postulación serializa la cuota de 50 MiB entre todas sus postulaciones; el envío verifica cada respaldo y sella su SHA-256. |
| 2026-10-31 | Descargas temporales desde servidor local separado o S3; cola persistente de bajas | La API no hace proxy de PDFs. El modo local firma URLs HMAC de cinco minutos; S3 usa SigV4. Los borrados y las cascadas generan tareas reintentables dentro de PostgreSQL. |

| 2026-10-31 | Registro y recuperación con OTP por correo, sin validación dactilar ni fecha de expedición | No existe acceso a Registro Civil. Se valida módulo 10 y acceso al correo; los demás datos personales son declarados. El RF original se conserva y la adaptación se documenta en ADR-008. |

| 2026-10-31 | Definición EF-12 embebida y casos de validación compartidos Go/TypeScript | `internal/submission/definition.json` es la fuente de opciones/tipos de captura; los cambios deben conservar paridad con `src/features/formulario/validacion.ts` mediante `form_cases.json`. Los contenedores frontend y CI incluyen estos recursos. La cuenta OTP aporta la precarga sin certificar identidad civil. |

| 2026-10-31 | Tres roles y política administrativa explícita en `identity/permissions.go` | Toda ruta administrativa vive en `/api/v1/admin/`, usa sesión y permiso por operación; los permisos desconocidos se deniegan. Las rutas del postulante conservan titularidad. La lectura administrativa solo entrega snapshots enviados y URLs temporales tras registrar auditoría. |
| 2026-10-31 | Cambios de rol auditados por trigger y confirmados en transacción | El registro OTP siempre crea postulantes. El PATCH de roles exige administrador vigente bajo bloqueo; la CLI de alta inicial exige operador y motivo. Los historiales de acceso y rol rechazan modificación/borrado. T8 completa retención y consulta del historial. |

| 2026-10-31 | Catálogos persistidos con semilla reproducible y caché Redis prescindible | T7 conserva 194 países RF, 24 provincias, 221 cantones del catálogo existente y opciones EF-12. `cmd/catalog-seed` genera migración y SQL autónomo desde la definición/geografía. Redis usa 24 h; errores y entradas inválidas consultan PostgreSQL. Las listas usan las claves `paises`/`provincias`/`cantones`/`opciones`; se admiten filtros y las rutas geográficas con segmentos del cliente existente. `EC` es alias HTTP de Ecuador. |

| 2026-10-31 | Historial append-only sin cascadas y eventos de negocio dentro del commit | T8 conserva eventos al retirar borradores/cuentas, pagina por fecha/UUID y audita accesos administrativos. `WorkflowRecorder` exige transacción; los futuros módulos deben instrumentarlo. Sin purga automática hasta aprobar retención en T28. |
| 2026-10-31 | Revisión T9 por período explícito, sin asignar una convocatoria ficticia al legado | `competitions` prepara históricos de T18. La relación de origen es inmutable y `sinConvocatoria=true` separa el legado. Sólo se consulta el sello enviado y se conserva su duración histórica; las descargas reutilizan T4/T6. |
| 2026-10-31 | Matriz técnica versionada y conteos frescos, sin puntajes inventados | `review/` propone columnas desde EF-13 y polling de 30 s, sujetos a aprobación institucional. XLSX/PDF comparten filtros; la exportación usa una instantánea de lectura y falla sin archivo parcial al superar 10000 filas/32 MiB/limitaciones de formato. T18 y FT7/FT8 mantienen su alcance. |

| 2026-10-31 | Identidad falla cerrada sin Redis; OTP por finalidad/desafío/cuenta/correo con scripts atómicos | La corrección de INFORME-BACKEND-T1-T9 sustituye el consumo Get/Delete y el pipeline de intentos. Catálogos sí conservan fallback a PostgreSQL. Reenvío 60 s, tres emisiones/10 min y tres intentos por desafío; el TTL de la ventana no se renueva. |
| 2026-10-31 | Cola de correo cifrada y confirmación durable dentro del sello | `MAIL_OUTBOX_KEY` es base64 de 32 bytes aleatorios estables, inyectada fuera del repo. `mail_outbox` usa AES-GCM, leases, reintentos y vencimiento; SMTP ocurre después del commit. `smtp_accepted` no significa llegada al buzón. No aceptar sellos que no puedan registrar su confirmación. Contrato: `docs/api-contracts/IDENTITY_SECURITY.md`. |
| 2026-10-31 | El set de mockups tiene una sola fuente de verdad (`docs/mockups/fixtures/`) y dos verificadores sin dependencias (`check-consistencia.mjs`, `check-contrato.mjs`) | La revisión y la crítica de diseño Detectaron que el set mostraba tres versiones del mismo expediente (hash, fechas y adjuntos distintos). Un fixture único más dos scripts hace que la contradicción sea imposible de reintroducir: si un valor cambia, el script falla en vez de dejar dos pantallas discrepantes. |
| 2026-10-31 | En los mockups, el folio de la hoja imprimible se genera en el margin box de `@page` | Es el único contexto donde los contadores `page` y `pages` devuelven el número real de página; con `position: fixed` el contador queda en 0. |
| 2026-10-31 | FT7/FT8 comparten filtros administrativos en la URL y solo consumen snapshots sellados | Período o legado explícito; conteos cada 30 s visibles/en línea. Caché privada por cuenta/rol con `gcTime: 0`; blobs y enlaces temporales no se persisten. La matriz proviene del backend y no es una lista oficial de elegibles. |

| 2026-10-31 | T18 separa registro de períodos y controles versionados; ninguna convocatoria se habilita por defecto | Aprobación, publicación, sector y fase se comprueban bajo bloqueo en PostgreSQL para toda escritura. Zona America/Guayaquil y ventana [apertura,cierre), con precisión de microsegundos; validación institucional pendiente. El legado conserva solo lectura. |
| 2026-10-31 | T17 verifica elegibilidad mediante puerto del servidor y confirma cierres antes de responder 409 | Las banderas del cliente no acreditan elegibilidad. El adaptador controlado se identifica como simulación; dato ausente/fallo técnico es 503, no cierre. Estado closed y evidencia append-only, sin reapertura automática ni evasión mediante otro borrador del mismo período. |
| 2026-10-31 | T15/T16 conservan identidad lógica del correo y TTL efectivo de los controles | La cola deduplica por recurso/finalidad o desafío y conserva Message-ID; SMTP sigue siendo al menos una vez. Reenvío configurable, tres fallos y bloqueo de diez minutos sin renovación por concurrencia; Retry-After procede del TTL de Redis. |

| 2026-10-31 | T19 imprime y adjunta el snapshot sellado mediante un sobre cifrado, preparado después del commit | El mismo generador `ef12-print-v1` valida el hash y conserva los literales guardados. SMTP nunca representa un borrador ni consulta un perfil posterior; preparación fallida queda reintentable sin deshacer el sello. |
| 2026-10-31 | T20 versiona reglas aprobadas, revisión, cálculo y publicación por separado | El RF no contiene puntajes ni umbrales: no hay matriz oficial precargada. `meritos-v1` puntúa sólo respaldos verificados, con aritmética exacta y días al envío; q1 es positiva. La fase `meritos`, el rol vigente y auditoría se comprueban en transacción. Recalcular no sustituye una resolución publicada; corregir exige referencia y versión previas. |

| 2026-10-31 | FT14/FT15 usan período explícito, fechas Ecuador y confirmación versionada | Portal guiado por `docs/mockups/01-portal.html` y componentes del set; el cronograma procede de T18, no de fechas ilustrativas. Las ediciones capturan su versión de origen y no se reintentan ciegamente ante 409. La disponibilidad se refresca cada 30 s; sin conexión, fallo o dato de más de 45 s bloquea nuevas escrituras en UI sin declarar cierre definitivo. El servidor conserva la autorización final. |

| 2026-10-31 | T21 separa informe psicotécnico, publicación y consolidación de tres fuentes definitivas | `evaluation/` versiona instrumento/escalas/pesos aprobados, usa racionales exactos y conserva cada corrección. T27 sólo aporta resultados por puerto interno en su transacción; no existe ruta HTTP de nota teórica libre ni resultado simulado por defecto. |
| 2026-10-31 | T22 congela términos por actor y requiere actuaciones distintas de postulante, analista y administrador | No hay plazos oficiales en el RF. La política aprobada define días, feriados y eventual límite de fase. Reloj PostgreSQL tras los bloqueos, claves idempotentes y evidencia inmutable preservan el original; rectificar exige otra publicación autorizada. Los respaldos del postulante comparten la cuota T4 y las decisiones encolan correo dentro del commit. |

| 2026-10-31 | T23 identifica avisos de etapa por fuente/expediente y conserva el historial de entrega | `notice/` comparte la cola cifrada con T19/T21/T22 y la publicación T20/T24. Reintentar exige administrador vigente, motivo y contador; mantiene destinatario/Message-ID y no revive aceptados, OTP o vencidos. Cinco intentos por ciclo, tres reintentos manuales y expiración original son límites operacionales, no plazos legales. |
| 2026-10-31 | T24 congela listas internas y publica sólo nombres autorizados, mediante proyección separada | `results/` valida fuentes actuales del sello y reconsideraciones resueltas tras el cierre. El orden necesita referencia aprobada; empates comparten puesto y no adjudican cargos. La lista pública es opt-in, sin notas, contactos o documentos; exige literal sellado de nombres y referencia humana de privacidad, sin sustituir T28. |

| 2026-10-31 | T25–T27 congelan matriz, horario y un intento por postulación en `exam/` | Banco/configuración nuevos son versiones inmutables; azar criptográfico y orden único por configuración con agotamiento explícito. El reloj PostgreSQL posterior a los bloqueos manda; cierre manual/worker comparten nota, PDF BYTEA inmutable, auditoría y outbox en un commit. No hay nota HTTP libre ni matriz correcta en rutas de postulante. |
| 2026-10-31 | T28 requiere política aprobada y aceptación ID/hash dentro del sello; ninguna política se siembra | `privacy/` conserva cinco literales, vínculo inmutable, ventanas y restricciones por canal, gate de listas públicas y control antes de SMTP. Sin vínculo, las salidas del legado fallan cerradas; no se inventa aceptación ni se reescribe hash. La fecha de revisión solicita decisión humana, no autoriza borrar evidencia. Textos/plazos y disposición material requieren aprobación institucional. |
| 2026-10-31 | La paleta sale del CSS del portal oficial del BIESS, no de una impresión | `docs/design/PALETA-BIESS.md` fija el origen, los hex y el contraste verificado: azul institucional `#0060A9`, navegación `#003A66`, botón `#005490`, celeste `#0097CC`. El amarillo `#FFD200` queda como token disponible, sin aplicarse a la interfaz: los encabezados usan el azul institucional. El frontend y los mockups declaran el mismo valor token por token en `oklch()`; el hex queda solo en el comentario. Los estados de éxito y advertencia se oscurecen porque los del portal son fondos de mensaje. El modo oscuro es derivación de la paleta, no inversión. La app es un simulacro: toma el color, no las marcas. |
| 2026-10-31 | Demostraciones backend aisladas mediante `scripts/backend-demo/` y proyecto `framework-backend-demo` | No detener el stack del frontend ni reutilizar sus datos; secretos/fixtures quedan en rutas ignoradas. `DATA_ORIGIN` marca salidas sólo en desarrollo. El perfil QA de 5 s autorizado usa otra versión; el principal conserva 3600 s. |
| 2026-10-31 | Métricas privadas y trazas locales explícitas, sin prometer OTLP | `METRICS_ADDR` separado del proxy; etiquetas cerradas, rutas parametrizadas, UUID de solicitud y trazas W3C locales. Redis limita operaciones/pool a 3 s y no repite scripts tras errores de red ambiguos. |
| 2026-10-31 | Diario de estados uniforme desde migración 00019, sin backfill de actores | Triggers guardan proyecciones antes/después y motivo técnico/humano en el commit; identidad/UUID de solicitud son locales a la transacción. No duplicar formularios personales en el diario ni inferir actores históricos. |
| 2026-11-01 | El frontend agrupa por feature y separa la lógica pura por capa, con fronteras verificadas por ESLint | `features/<dominio>/modelo.ts` y `validacion.ts` alojan la lógica pura del dominio; `helpers/` guarda la regla de este portal compartida entre features y `utils/` lo genérico. El backend Go es el dominio: el frontend sólo replica lo necesario para la UX (`features/formulario/validacion.ts` frente a `form_cases.json`). `lib/` es transporte sin reglas de negocio. `no-restricted-imports` bloquea feature→feature, shared→feature y `components/ui`→feature; `auth` queda como primitivo transversal y los módulos compartidos ya enumerados son la excepción, no un patrón. |
