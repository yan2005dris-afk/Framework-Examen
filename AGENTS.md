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
- pnpm 12 como package manager (workspace) — lockfile `pnpm-lock.yaml`

### Backend

- Go 1.27 — `chi` v5 (routing), `pgx` v5 (Postgres), `uuid`
- Config por entorno: `PORT`, `APP_ENV`, `APP_URL`, `DATABASE_URL`, `DB_REQUIRED`,
  `STORAGE_DIR`, `CORS_ALLOWED_ORIGINS`, `SESSION_TTL`

## Setup commands

Desde la raíz del repo:

- Instalar deps: `pnpm install`
- Frontend dev:   `pnpm dev`      (puerto 5173)
- Frontend build: `pnpm build`
- Frontend lint:  `pnpm lint`
- Backend build:  `cd apps/backend && go build ./...`
- Backend tests:  `cd apps/backend && go test ./...`

Desde `apps/frontend/` también funcionan `pnpm dev`, `pnpm build`, `pnpm lint`, `pnpm preview`.

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

`apps/backend/internal/`:

- `config/` — carga de configuración desde el entorno
- `database/` — conexión y acceso a Postgres
- `httpapi/` — respuestas y helpers HTTP

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
  proyecto sí las cumple.

## Testing instructions

- No hay framework de tests configurado todavía.
- Antes de abrir una PR: `pnpm lint` y `pnpm build` deben pasar en verde.
- La lógica de negocio va acompañada de tests en su módulo: Vitest en frontend,
  `_test.go` en backend.

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
