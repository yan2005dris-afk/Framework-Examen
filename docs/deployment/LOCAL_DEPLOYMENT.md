# Despliegue local

**Sistema de Registro y Postulación al Directorio del BIESS**
Guía del entorno de desarrollo

| Campo | Valor |
|---|---|
| Documento | LOCAL_DEPLOYMENT v1.0 |
| Compose de desarrollo | `compose.yaml` (raíz del repo) |
| Compose de producción | `deploy/compose.yaml` (ver [PRODUCTION_DEPLOYMENT.md](PRODUCTION_DEPLOYMENT.md)) |

---

## 1. Prerrequisitos

| Herramienta | Versión | Para qué |
|---|---|---|
| Go | 1.27.3 o superior | Compilar y probar el backend |
| Node.js | 22 LTS | Frontend |
| pnpm | 12 | Gestor de paquetes del workspace |
| Docker + Compose v2 | Reciente | PostgreSQL, Redis, RustFS, Mailpit y los servicios del proyecto |
| PostgreSQL (opcional) | 18+ | Solo si se trabaja **sin** Docker |
| Redis (opcional) | 7 | Solo si se trabaja **sin** Docker. Bloqueo de intentos de login y caché de catálogos |

> **Nota**: sin Redis la API arranca, pero el inicio de sesión y el registro
> fallan cerrados (no se pueden contar los intentos) y los catálogos se leen
> directamente de PostgreSQL.

Verificación rápida:

```bash
go version        # go1.27.3 o superior
node --version    # v22.x
pnpm --version    # 12.x
docker compose version
```

---

## 2. Opción A — Todo con Docker (recomendada)

Levanta API, frontend, PostgreSQL, Redis, RustFS y Mailpit. Es la forma más
rápida de tener el entorno completo.

```bash
# 1. Instalar las dependencias del frontend (el compose usa el lockfile)
pnpm install

# 2. Levantar el stack
docker compose up -d

# 3. Ver el estado
docker compose ps
```

4. Cargar los **datos de prueba** (cuentas, convocatoria, bucket de documentos
   y elegibilidad simulada): ver la sección
   [Datos de prueba del README](../../README.md#datos-de-prueba-solo-desarrollo).
   Sin ellos no hay con qué iniciar sesión ni a qué postular.

| Servicio | URL | Puerto |
|---|---|---|
| Frontend (Vite) | http://localhost:5173 | 5173 |
| API Go | http://localhost:8080 | 8080 |
| PostgreSQL | `localhost:5432/biess` | 5432 |
| Redis | `localhost:6379` | 6379 |
| RustFS (API S3) | http://localhost:9000 | 9000 |
| RustFS (consola) | http://localhost:9001 | 9001 |
| Mailpit (bandeja) | http://localhost:8025 | 8025 |
| Mailpit (SMTP) | `localhost:1025` | 1025 |

### 2.1 Cómo funciona el proxy

En desarrollo **no hay nginx**. El propio Vite hace de proxy:

```ts
// apps/frontend/vite.config.ts
server: {
  proxy: {
    "/api": { target: env.VITE_DEV_API_TARGET, changeOrigin: true },
  },
}
```

El navegador llama a `/api/v1/...` y Vite reenvía a `http://api:8080`. El
resultado es que existe **un solo origen**, igual que en producción: no hay
CORS ni cookies cross-origin, y el código del frontend no cambia entre
entornos.

### 2.2 Correo

Mailpit captura **todo** el correo que envía la API, sin enviarlo a nadie. Se
abarca la bandeja en http://localhost:8025. La API apunta a él con:

```yaml
SMTP_HOST: mailpit
SMTP_PORT: "1025"
SMTP_FROM: no-reply@biess.local
```

### 2.3 Base de datos

PostgreSQL en desarrollo usa **autenticación `trust`** (sin contraseña): es una
base local, aislada del exterior, y evita escribir secretos en el compose.

```bash
# Entrar a la consola de PostgreSQL
docker compose exec postgres psql -U postgres -d biess

# Ver las tablas
docker compose exec postgres psql -U postgres -d biess -c '\dt'

# Ver los datos semilla
docker compose exec postgres psql -U postgres -d biess -c 'SELECT * FROM applicants;'
```

---

## 3. Opción B — Backend nativo, base en Docker

Útil para iterar rápido sobre Go, con recarga automática.

```bash
# 1. Solo la base de datos y el correo
docker compose up -d postgres mailpit

# 2. Migraciones
cd apps/backend
make db-migrate

# 3. API en modo desarrollo
make run
```

La API queda en http://localhost:8080. El frontend sigue levantándose con
`docker compose up -d frontend`, que ya apunta a `http://api:8080`; si se
prefiere el frontend nativo, hay que cambiar `VITE_DEV_API_TARGET` a
`http://localhost:8080` y levantar Vite con `pnpm dev`.

### 3.1 Tareas del Makefile

Desde `apps/backend`:

| Target | Qué hace |
|---|---|
| `make help` | Lista los targets disponibles |
| `make run` | Arranca la API en desarrollo |
| `make build` | Compila el binario en `bin/api` |
| `make test` | Ejecuta los tests |
| `make vet` | Ejecuta `go vet` |
| `make fmt` | Formatea con `gofmt` |
| `make tidy` | Ordena el `go.mod` |
| `make sqlc-generate` | Regenera `internal/platform/sqlcgen` |
| `make db-schema` | Regenera `sql/queries/schema.sql` desde las migraciones |
| `make db-migrate` | Aplica las migraciones pendientes |
| `make db-status` | Muestra el estado de las migraciones |
| `make db-down` | Revierte la última migración |
| `make db-reset` | **DESTRUCTIVO**: recrea el esquema |
| `make docker-up` / `make docker-down` | Levanta o apaga el compose de desarrollo |

### 3.2 Migraciones

```bash
cd apps/backend
make db-migrate      # aplica lo pendiente
make db-status       # qué versión está aplicada
make db-down         # revierte la última
```

El runner es `cmd/migrate`, con las migraciones **embebidas** en el binario:
no hace falta el CLI de goose.

```bash
# Equivalente manual, con otra base de datos
DATABASE_URL="postgres://postgres@localhost:5432/biess?sslmode=disable" \
  go run ./cmd/migrate

# Ver el estado
DATABASE_URL="postgres://postgres@localhost:5432/biess?sslmode=disable" \
  go run ./cmd/migrate -status
```

---

## 4. Frontend

```bash
# Desde la raíz del repo
pnpm install
pnpm dev          # http://localhost:5173

# Scripts disponibles
pnpm build        # compila a apps/frontend/dist
pnpm lint         # ESLint
pnpm test         # Vitest
pnpm format       # Prettier (escribe)
pnpm format:check # Prettier (verifica, no escribe)
```

También funciona dentro de `apps/frontend/`:

```bash
cd apps/frontend
pnpm dev
pnpm build
pnpm lint
```

### 4.1 Componentes shadcn/ui

`src/components/ui/` **no se edita a mano**: se regenera con el CLI.

```bash
pnpm dlx shadcn@latest add <componente>
pnpm dlx shadcn@latest add <componente> --overwrite
```

`components.json` es la fuente de verdad de aliases, estilo y variables CSS.

---

## 5. Variables de entorno

### 5.1 Backend

`apps/backend/.env.example` es la plantilla. Se copia a `.env`, que **nunca**
se versiona:

```bash
cp apps/backend/.env.example apps/backend/.env
```

| Variable | Por defecto | Descripción |
|---|---|---|
| `APP_ENV` | `development` | `development` \| `production` |
| `PORT` | `8080` | Puerto de escucha |
| `APP_URL` | `http://localhost:5173` | URL pública del frontend (enlaces de correo) |
| `DATABASE_URL` | — | Cadena de conexión a PostgreSQL |
| `DB_REQUIRED` | `false` | Si es `true`, la API no arranca sin base de datos |
| `REDIS_URL` | `redis://localhost:6379/0` | Bloqueo de intentos y caché de catálogos (`redis://redis:6379/0` en Docker) |
| `STORAGE_DRIVER` | `s3` | `s3` (RustFS en desarrollo) o almacenamiento local |
| `S3_ENDPOINT` / `S3_PUBLIC_ENDPOINT` | `http://localhost:9000/biess` | Endpoint del bucket `biess`; hay que crearlo una vez |
| `S3_ACCESS_KEY` / `S3_SECRET_KEY` | `rustfsadmin` | Credenciales de RustFS en desarrollo |
| `STORAGE_DIR` | `./data/uploads` | Directorio de adjuntos con almacenamiento local |
| `ELIGIBILITY_SIMULATION_FILE` | vacío | JSON de elegibilidad simulada; vacío = la sección 4 responde 503. Solo desarrollo |
| `MAIL_OUTBOX_KEY` | — | Clave base64 de 32 bytes para cifrar la cola de correo |
| `OTP_RESEND_DELAY` / `OTP_MAX_SENDS` | `1m` / `3` | Espera entre reenvíos y envíos máximos en 10 min |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:5173` | Lista separada por comas; vacío = denegar todo |
| `SESSION_TTL` | `24h` | Vigencia de la sesión |
| `HTTP_READ_TIMEOUT` | `15s` | Timeout de lectura |
| `HTTP_WRITE_TIMEOUT` | `30s` | Timeout de escritura |
| `HTTP_IDLE_TIMEOUT` | `60s` | Timeout de conexiones inactivas |
| `HTTP_REQUEST_TIMEOUT` | `30s` | Timeout por petición |
| `MAX_REQUEST_BODY_BYTES` | `10485760` | Máximo del cuerpo (10 MiB) |
| `LOG_LEVEL` | `info` | `debug` \| `info` \| `warn` \| `error` |
| `SMTP_HOST` | — | Servidor SMTP (`mailpit` en desarrollo) |
| `SMTP_PORT` | `1025` | Puerto SMTP |
| `SMTP_USER` / `SMTP_PASSWORD` | — | Credenciales SMTP |
| `SMTP_FROM` | `no-reply@biess.local` | Remitente |
| `SMTP_TLS` | `false` | STARTTLS |

### 5.2 Frontend

| Variable | Por defecto | Descripción |
|---|---|---|
| `VITE_DEV_API_TARGET` | `http://api:8080` | Destino del proxy de Vite (solo desarrollo) |

En producción **no hay variables del frontend**: la SPA se compila y nginx
sirve los archivos estáticos. `/api` se resuelve en el mismo origen.

---

## 6. Pruebas

### 6.1 Backend

```bash
cd apps/backend
gofmt -l .          # debe imprimir nada
go build ./...
go vet ./...
go test ./... -race
go test ./... -cover
```

Las pruebas de las reglas puras (cédula módulo 10, canonización y hash del
contenido, estados de la postulación, paginación) **no necesitan base de
datos**.

### 6.2 Frontend

```bash
pnpm format:check
pnpm lint
pnpm test
pnpm build
```

### 6.3 Smoke test manual

Requiere los [datos de prueba](../../README.md#datos-de-prueba-solo-desarrollo):
las rutas de postulación y catálogos exigen sesión.

```bash
# Sondas de la API
curl -s http://localhost:8080/healthz
curl -s http://localhost:8080/readyz

# Iniciar sesión con el postulante de prueba (guarda la cookie de sesión)
curl -s -c sesion.txt -H 'Content-Type: application/json' \
  -d '{"cedula":"1710034008","password":"Password.2026*"}' \
  http://localhost:8080/api/v1/auth/login

# Catálogos (servidos desde Redis tras la primera consulta)
curl -s -b sesion.txt http://localhost:8080/api/v1/catalogs/provinces/Ecuador
curl -s -b sesion.txt http://localhost:8080/api/v1/catalogs/cities/Ecuador/Guayas

# Postulaciones del usuario y convocatorias publicadas
curl -s -b sesion.txt 'http://localhost:8080/api/v1/submissions?limit=20'
curl -s http://localhost:8080/api/v1/competitions
```

### 6.4 Pruebas de punta a punta

```bash
pnpm --filter frontend test:e2e   # Playwright
```

---

## 7. Problemas frecuentes

| Síntoma | Causa probable | Solución |
|---|---|---|
| `connection refused` en `/readyz` | La API no ve PostgreSQL | `docker compose ps` y revisa `DATABASE_URL` |
| `no such table: submissions` | Faltan las migraciones | `make db-migrate` |
| La SPA no carga datos | El proxy de Vite no apunta a la API | Revisa `VITE_DEV_API_TARGET` |
| `pnpm: command not found` | pnpm no está instalado | `corepack enable && corepack prepare pnpm@12 --activate` |
| Los adjuntos no aparecen | `STORAGE_DIR` no existe o no es escribible | Crea el directorio o corrige la variable |
| `cuerpo_muy_grande` (413) | El archivo supera `MAX_REQUEST_BODY_BYTES` | Sube el límite **y** el de nginx en producción |
| El login responde 401 con las credenciales de prueba | No se cargaron las semillas, o se escribió el correo en lugar de la cédula | Cargar los [datos de prueba](../../README.md#datos-de-prueba-solo-desarrollo); el login pide la cédula |
| «La recepción no está disponible» en el formulario | No hay convocatoria abierta | Cargar `competitions.sql` (datos de prueba) o crear una desde el panel |
| `competitions.sql` falla con «La convocatoria con expedientes conserva su período e identidad» | La convocatoria ya tiene postulaciones y su historial está protegido | Nada que corregir: la semilla solo hace falta la primera vez y no modificó datos |
| `503 almacenamiento_no_disponible` al subir un PDF | Falta el bucket `biess` en RustFS | Crearlo (ver datos de prueba) |
| `503 elegibilidad_no_disponible` al guardar la sección 4 | `ELIGIBILITY_SIMULATION_FILE` vacío o la cuenta no figura en el JSON | Configurar la elegibilidad simulada (ver datos de prueba) |
| Pruebas de integración fallan con `no se pudo crear el esquema aislado` | Otro PostgreSQL ocupa el puerto 5432 del equipo | Detener ese servicio o ejecutar las pruebas dentro de la red de Docker |

### Depurar un contenedor

```bash
docker compose logs -f api
docker compose logs -f frontend
docker compose exec api /app/api healthcheck
```

### Reiniciar todo

```bash
docker compose down          # conserva los volúmenes
docker compose down -v       # borra también los datos (DESTRUCTIVO)
```

---

## 8. Qué falta para el entorno completo

| Elemento | Estado |
|---|---|
| PostgreSQL 18 | ✅ Disponible |
| Mailpit (SMTP de pruebas) | ✅ Disponible |
| Redis (caché y bloqueo de intentos) | ✅ Disponible |
| RustFS / S3 (adjuntos en objetos) | ✅ Disponible; el bucket `biess` se crea una vez |
| E2E (Playwright) | ✅ Disponible (`pnpm --filter frontend test:e2e`) |
| Datos de prueba automáticos | ⚠️ Manuales a propósito: ver [datos de prueba](../../README.md#datos-de-prueba-solo-desarrollo) |

El detalle está en [`../tasks/backend-tasks.md`](../tasks/backend-tasks.md) y
[`../tasks/frontend-tasks.md`](../tasks/frontend-tasks.md).

---

## 9. Actualizar desde PostgreSQL 17

Si el stack ya estaba levantado con PostgreSQL 17, el directorio de datos del
volumen **no** lo abre PostgreSQL 18. Además, desde la versión 18 las imágenes
Docker guardan los datos en `/var/lib/postgresql/<major>/<cluster>`: el
compose monta el volumen en `/var/lib/postgresql`, no en `.../data`.

En desarrollo el camino corto es recrear la base y volver a aplicar las
migraciones. Los datos de desarrollo son semilla y se regeneran:

```bash
# 1. Volcado por si acaso
docker compose exec -T postgres pg_dump -U postgres -d biess > /tmp/biess.sql

# 2. Recrear el contenedor de PostgreSQL con el layout nuevo
docker compose up -d --force-recreate postgres

# 3. Aplicar las migraciones sobre el cluster nuevo
cd apps/backend && make db-migrate
```

Si el arranque falla con un aviso sobre *"PostgreSQL data in
/var/lib/postgresql/data"*, el volumen todavía tiene el cluster viejo: hay
que vaciarlo o usar un volumen nuevo.

El procedimiento de producción, con `pg_upgrade` y con volcado lógico, está
en [PRODUCTION_DEPLOYMENT.md §6.1](PRODUCTION_DEPLOYMENT.md).

---

**Ver también**: [PRODUCTION_DEPLOYMENT.md](PRODUCTION_DEPLOYMENT.md) ·
[`../architecture/ARQUITECTURA.md`](../architecture/ARQUITECTURA.md) ·
[`../api-contracts/API_CONTRACT.md`](../api-contracts/API_CONTRACT.md)
