# Framework-Examen

**Sistema de Registro y Postulación al Directorio del BIESS**
Módulo inicial de registro y postulación para el Concurso Público de Méritos y
Oposición del Directorio del Banco del Instituto Ecuatoriano de Seguridad Social
(IESS).

SPA en React + microservicio transaccional en Go, con PostgreSQL y Redis,
separados en capas con inyección de dependencias.

---

## Prerrequisitos

| Herramienta | Versión | Para qué |
|---|---|---|
| Go | 1.27.3+ | Backend |
| Node.js | 22 LTS | Frontend |
| pnpm | 12 | Gestor de paquetes del workspace |
| Docker + Compose v2 | Reciente | PostgreSQL, Redis, Mailpit y el stack completo |

---

## Puesta en marcha

### Con Docker (recomendado)

```bash
pnpm install
docker compose up -d
```

| Servicio | URL |
|---|---|
| Frontend (Vite) | http://localhost:5173 |
| API Go | http://localhost:8080 |
| PostgreSQL | `localhost:5432/biess` |
| Bandeja de correo (Mailpit) | http://localhost:8025 |

### Sin Docker

```bash
# Base de datos
cd apps/backend && make docker-up && make db-migrate

# API
cd apps/backend && make run

# Frontend
pnpm dev
```

Detalle completo, troubleshooting y variables de entorno en
[`docs/deployment/LOCAL_DEPLOYMENT.md`](docs/deployment/LOCAL_DEPLOYMENT.md).

---

## Estructura

```text
apps/
├── backend/     # API Go (chi v5 + pgx + sqlc + goose)
└── frontend/    # SPA React (Vite + Tailwind v4 + shadcn/ui)
deploy/          # compose y nginx de producción
compose.yaml     # compose de desarrollo
docs/            # documentación completa del proyecto
```

---

## Comandos

| Tarea | Comando |
|---|---|
| Instalar dependencias | `pnpm install` |
| Frontend en desarrollo | `pnpm dev` |
| Lint del frontend | `pnpm lint` |
| Tests del frontend | `pnpm test` |
| Formato del frontend | `pnpm format` · `pnpm format:check` |
| Build del frontend | `pnpm build` |
| Build del backend | `cd apps/backend && go build ./...` |
| Tests del backend | `cd apps/backend && go test ./... -race` |
| Migraciones | `cd apps/backend && make db-migrate` |
| Regenerar sqlc | `cd apps/backend && make sqlc-generate` |
| Stack completo | `docker compose up -d` |

Ayuda completa del backend: `cd apps/backend && make help`.

---

## Variables de entorno

| Archivo | Uso |
|---|---|
| [`apps/backend/.env.example`](apps/backend/.env.example) | Plantilla del backend (SMTP incluido) |
| [`deploy/.env.example`](deploy/.env.example) | Plantilla de producción |

Los archivos `.env` reales **nunca** se versionan. Sin configuración, la API
arranca con valores por defecto aptos para desarrollo.

---

## Documentación

La documentación completa vive en [`docs/`](docs/):

| Documento | Contenido |
|---|---|
| [`docs/README.md`](docs/README.md) | Índice de toda la documentación |
| [`docs/prd/PRD.md`](docs/prd/PRD.md) | Requisitos de producto: alcance, historias, reglas de negocio |
| [`docs/trd/TRD.md`](docs/trd/TRD.md) | Requisitos técnicos: stack, modelo de datos, API, seguridad |
| [`docs/architecture/ARQUITECTURA.md`](docs/architecture/ARQUITECTURA.md) | Vistas del sistema, capas y ciclo de una petición |
| [`docs/database/SCHEMA.md`](docs/database/SCHEMA.md) | Modelo de datos, DDL, índices y semillas |
| [`docs/api-contracts/API_CONTRACT.md`](docs/api-contracts/API_CONTRACT.md) | Contrato HTTP: endpoints, errores, paginación |
| [`docs/decisions/`](docs/decisions/) | Decisiones de arquitectura (ADR) |
| [`docs/deployment/`](docs/deployment/) | Despliegue local y en producción |
| [`docs/tasks/`](docs/tasks/) | Trabajo pendiente, mapeado a la rúbrica |

Para las convenciones del repositorio y el flujo de trabajo, ver
[`AGENTS.md`](AGENTS.md).

---

## Entregables del examen

| Entregable | Ubicación |
|---|---|
| Repositorio con historial y ramas ordenadas | Este repositorio (`develop` → `feature/*` → PR) |
| Prerrequisitos y ejecución local | Este README + [`docs/deployment/LOCAL_DEPLOYMENT.md`](docs/deployment/LOCAL_DEPLOYMENT.md) |
| Variables de entorno de ejemplo | `.env.example` del backend y de `deploy/` |
| Script DDL y datos semilla | [`apps/backend/sql/`](apps/backend/sql/) — migraciones goose y seeds |
| Documentación funcional y técnica | [`docs/`](docs/) |
