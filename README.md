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
| Docker + Compose v2 | Reciente | Levanta el stack completo con los servicios de abajo |
| PostgreSQL | 18 | Base de datos (la imagen trae `uuidv7()` nativo) |
| Redis | 7 | Bloqueo de intentos de login (3 fallos, 10 min) y caché de catálogos (24 h) |
| RustFS (compatible con S3) | Reciente | Almacenamiento de los PDF de respaldo |
| Mailpit | Reciente | Captura el correo de desarrollo (códigos OTP y avisos) |

Con Docker, PostgreSQL, Redis, RustFS y Mailpit vienen en `compose.yaml`: no
hace falta instalarlos aparte.

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
| Redis | `localhost:6379` |
| RustFS: API S3 / consola | http://localhost:9000 · http://localhost:9001 |
| Bandeja de correo (Mailpit) | http://localhost:8025 |

Al arrancar, la API aplica las migraciones, incluidos los catálogos de países,
provincias y cantones. Las cuentas de prueba y la convocatoria **no** se crean
solas: ver [Datos de prueba](#datos-de-prueba-solo-desarrollo).

### Datos de prueba (solo desarrollo)

> [!WARNING]
> Estos pasos son **solo para desarrollo**. Crean cuentas con una contraseña
> conocida, una convocatoria ficticia abierta y resultados de elegibilidad
> inventados. **Nunca** los ejecutes contra la base de producción. Ver
> [En producción](#en-producción).

Con el stack levantado y la API ya en marcha (`docker compose ps`), desde la
raíz del repositorio:

**1. Cuentas y convocatoria.** Primero `users.sql`: la política de privacidad
de la convocatoria la firma el administrador de prueba.

```bash
# bash / Git Bash
docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U postgres -d biess < apps/backend/sql/seeds/users.sql
docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U postgres -d biess < apps/backend/sql/seeds/competitions.sql
```

```powershell
# PowerShell: el operador < no existe; cmd /c conserva las tildes
cmd /c 'docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U postgres -d biess < apps\backend\sql\seeds\users.sql'
cmd /c 'docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U postgres -d biess < apps\backend\sql\seeds\competitions.sql'
```

`users.sql` se puede repetir sin duplicar datos. `competitions.sql` solo hace
falta la primera vez: cuando la convocatoria ya tiene postulaciones, el sistema
protege su historial y la semilla termina con «La convocatoria con expedientes
conserva su período e identidad» sin modificar nada.

| Rol | Cédula | Correo | Contraseña |
|---|---|---|---|
| Postulante | `1710034008` | `postulante@biess.local` | `Password.2026*` |
| Analista | `1710034016` | `analista@biess.local` | `Password.2026*` |
| Administrador | `0910034008` | `admin@biess.local` | `Password.2026*` |

El login pide la **cédula**, no el correo.

**2. Bucket de documentos.** Sin él, subir un PDF responde
`503 almacenamiento_no_disponible`. Se crea una vez desde la consola de RustFS
(http://localhost:9001, usuario y contraseña `rustfsadmin`) con el nombre
`biess`, o por línea de comandos. La red se llama `<carpeta>_default` según el
nombre de la carpeta del repositorio (`docker network ls` la muestra):

```bash
docker run --rm --network framework-examen_default \
  -e AWS_ACCESS_KEY_ID=rustfsadmin -e AWS_SECRET_ACCESS_KEY=rustfsadmin -e AWS_DEFAULT_REGION=us-east-1 \
  amazon/aws-cli:2.27.50 --endpoint-url http://rustfs:9000 s3api create-bucket --bucket biess
```

**3. Elegibilidad simulada del IESS.** El RF verifica con Historial Laboral si
el postulante es afiliado, jubilado o empleador. Aquí se simula con un archivo
JSON por cuenta y sector; sin él, guardar la sección 4 responde
`503 elegibilidad_no_disponible`.

```bash
# Identificadores de las cuentas (cambian en cada instalación)
docker compose exec postgres psql -U postgres -d biess -Atc "SELECT id, cedula FROM applicants"
```

Crear `apps/backend/data/eligibility.json` (la carpeta está montada en el
contenedor como `/app/data` y no se versiona):

```json
{
  "UUID-DE-LA-CUENTA": { "afiliados": true, "jubilados": true, "empleadores": true }
}
```

Con `false` en un sector se prueba el cierre por no elegibilidad. Después,
crear un `.env` en la raíz con
`ELIGIBILITY_SIMULATION_FILE=/app/data/eligibility.json` y reiniciar la API con
`docker compose up -d api`.

Los datos sobreviven a `docker compose up -d --build` y a `docker compose down`;
**`docker compose down -v` los borra** y hay que repetir estos pasos.

### En producción

| Dato | Cómo se crea |
|---|---|
| Catálogos (países, provincias, cantones, opciones del formulario) | Automáticamente, con las migraciones al arrancar la API |
| Primer administrador | La persona se registra en el portal (OTP por correo) y un operador le asigna el rol con `go run ./cmd/applicant-role -cedula … -role administrador -operator … -reason …` (desde `apps/backend`, con `DATABASE_URL`). El cambio queda auditado |
| Convocatorias | Desde el panel de administración, con aprobación y publicación explícitas |
| Elegibilidad | Proveedor real del IESS; la simulación no se configura |

`users.sql`, `competitions.sql` y `eligibility.json` no se usan en producción.

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
| [`docs/mockups/`](docs/mockups/) | Mockups HTML de las 36 pantallas (referencia de diseño y contrato de migración) |

Para ver las pantallas, abre [`docs/mockups/index.html`](docs/mockups/index.html)
en el navegador: es HTML estático, sin servidor ni build. La guía de migración
(mapa pantalla → historia de usuario → componente shadcn) está en
[`docs/mockups/README.md`](docs/mockups/README.md).

Para las convenciones del repositorio y el flujo de trabajo, ver
[`AGENTS.md`](AGENTS.md).

---

## Entregables del examen

| Entregable | Ubicación |
|---|---|
| Repositorio con historial y ramas ordenadas | Este repositorio (`develop` → `feature/*` → PR) |
| Prerrequisitos y ejecución local | Este README + [`docs/deployment/LOCAL_DEPLOYMENT.md`](docs/deployment/LOCAL_DEPLOYMENT.md) |
| Variables de entorno de ejemplo | `.env.example` del backend y de `deploy/` |
| Script DDL | [`apps/backend/sql/queries/schema.sql`](apps/backend/sql/queries/schema.sql): esquema completo, generado desde las migraciones de [`sql/migrations/`](apps/backend/sql/migrations/) |
| Datos semilla de catálogos (división política) | [`apps/backend/sql/seeds/catalogs.sql`](apps/backend/sql/seeds/catalogs.sql): DDL y semilla autónoma de 194 países, 24 provincias y 221 cantones |
| Datos de prueba | [`users.sql`](apps/backend/sql/seeds/users.sql) y [`competitions.sql`](apps/backend/sql/seeds/competitions.sql): solo desarrollo, ver [Datos de prueba](#datos-de-prueba-solo-desarrollo) |
| Documentación funcional y técnica | [`docs/`](docs/) |
