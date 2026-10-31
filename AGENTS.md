# AGENTS.md

**Framework-Examen** — portal de postulación al concurso BIESS/IESS (Ecuador):
registro, login, formulario de postulación en 9 secciones (EF-12), revisión, envío con
hash criptográfico, impresión y dashboard. Monorepo pnpm: SPA React + API Go.

Este archivo es la definición del proyecto para cualquier agente de IA: léelo antes de
tocar código y actualízalo cuando cambie una convención.

## Estructura

- `apps/frontend/` — SPA: React 19 + Vite 8 + Tailwind v4 + shadcn/ui
- `apps/backend/` — API: Go 1.27 (chi + pgx/Postgres)
- `mockups/` — mockup original. Referencia de flujo, copy y lógica de las 9 vistas.
  **Temporal y local**: está en `.gitignore` (no se versiona) y se elimina cuando
  termine la migración de vistas a shadcn/ui.

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
- `make test` / `make vet` / `make fmt` / `make tidy`

### Componentes shadcn

```bash
pnpm dlx shadcn@latest add <componente>          # agregar
pnpm dlx shadcn@latest add <componente> --overwrite  # regenerar sobreescribiendo
```

`components.json` es la fuente de verdad de aliases, estilo y variables CSS.

## Project layout

`apps/frontend/src/`:

- `components/ui/` — componentes shadcn. **No editar a mano**: se regeneran con el CLI
- `components/` — componentes propios (`app-sidebar.tsx`, `app-header.tsx`,
  `theme-toggle.tsx`, `theme-provider.tsx`)
- `pages/` — vistas (`dashboard-view.tsx`)
- `types/concurso.ts` — tipos del dominio (formulario EF-12 de 9 secciones)
- `utils/validators.ts` — cédula módulo 10, fortaleza de contraseña, email
- `data/` — `ecuadorGeo.ts` (provincias/cantones), `mockInitialData.ts`
- `lib/utils.ts` — re-export de `cn`
- `hooks/use-mobile.ts`
- `assets/images/` — imágenes

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
  `database/`, `api-contracts/`, `decisions/`, `deployment/` y `tasks/`.
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
- Comentarios JSDoc en las funciones de `src/utils/`
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
- `mockups/` — referencia local del mockup original

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

| 2026-10-31 | Definición EF-12 embebida y casos de validación compartidos Go/TypeScript | `internal/submission/definition.json` es la fuente de opciones/tipos de captura; los cambios deben conservar paridad con `src/utils/validar-formulario.ts` mediante `form_cases.json`. Los contenedores frontend y CI incluyen estos recursos. La cuenta OTP aporta la precarga sin certificar identidad civil. |
