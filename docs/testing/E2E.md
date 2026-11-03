# Pruebas de extremo a extremo (E2E)

Este documento describe las dos suites E2E del portal: la de **interfaz**
(Playwright, navegador) y la de **API** (Go, HTTP real). Comparten el mismo
conjunto de datos sembrado, de modo que una prueba de interfaz y una de API
hablan del mismo postulante.

---

## 1. Por qué dos suites

| Suite | Cubre | Corre contra |
|---|---|---|
| **Interfaz** (`apps/frontend/e2e/`) | Lo que ve la persona: navegación, formularios, mensajes, permisos visibles | El stack de desarrollo (Vite + API + Postgres + Redis + Mailpit + S3) |
| **API** (`apps/backend/e2e/`) | El contrato HTTP: envelope de error, códigos de estado, cabeceras, paginación, RBAC, idempotencia | El router real sobre un esquema de PostgreSQL aislado |

No son intercambiables:

- La suite de API es **rápida y determinista**: no abre navegador, levanta el
  router de producción en un `httptest.Server` y usa un esquema de base de
  datos aislado que se destruye al terminar.
- La suite de interfaz necesita el stack completo y comprueba lo que un
  humano o un revisor de examen vería.

---

## 2. El seed: `internal/e2edata`

El paquete `apps/backend/internal/e2edata` siembra un conjunto determinista y
**idempotente**:

- **Cuentas** por rol: administrador, analista y cuatro postulantes con
  expedientes en estados distintos (enviado, borrador, cerrado).
- **Convocatoria** con su control aprobado y publicado.
- **Política de privacidad** vigente y el **vínculo de aceptación** de cada
  expediente.
- **Expedientes** con contenido canónico y sello SHA-256.

### Por qué vive en Go

El seed debe hablar el mismo idioma que el dominio. Si el backend no acepta
un documento (una opción inválida, un literal fuera del RF, un hash que no
coincide con el canónico), la suite de interfaz tampoco debe poder sembrarlo:
si no, el test mide un dato que la aplicación jamás aceptaría.

El seed construye el formulario con los **structs del dominio**
(`submission.Form`, `privacy.Rules`) y calcula el sello con
`submission.HashContent`, el mismo resumen que usa el servidor.

### Ejecutarlo

```bash
cd apps/backend
DATABASE_URL="postgres://postgres@localhost:5432/biess?sslmode=disable" go run ./cmd/e2e-seed
```

El comando imprime el conjunto en JSON. La suite de interfaz lo consume
automáticamente mediante el `globalSetup` de Playwright, que escribe el
archivo en `apps/frontend/e2e/.estado/seed.json`.

### Idempotencia

Repetir la siembra no duplica cuentas ni expedientes. El seed respeta las
invariantes del dominio, que son estrictas:

- Una **política publicada es inmutable**: si el contenido cambia, se publica
  una versión nueva en vez de reescribir la anterior.
- Un **expediente enviado o cerrado es inmutable**: sólo el borrador se
  realinea.
- El **vínculo de privacidad no se reescribe**: es evidencia de aceptación.
- Una cuenta tiene **un solo expediente por convocatoria**.

### Cuentas sembradas

| Rol | Cédula | Correo | Expediente |
|---|---|---|---|
| administrador | `2407612650` | `e2e-administrador@biess.local` | — |
| analista | `2402249987` | `e2e-analista@biess.local` | — |
| postulante | `2406425922` | `e2e-aprobado@biess.local` | enviado y sellado |
| postulante | `2402740035` | `e2e-minimo@biess.local` | borrador editable |
| postulante | `2403401355` | `e2e-cerrado@biess.local` | cerrado por no elegibilidad |
| postulante | `2408380695` | `e2e-reconsideracion@biess.local` | enviado |

Contraseña compartida: `TestPassword.2026*` (variable `E2E_PASSWORD`).

---

## 3. Suite de API (`apps/backend/e2e/`)

### Andamiaje

`e2e/harness.go` levanta:

- El **mismo router que el binario de producción**, construido por
  `internal/platform/appkit`. La suite no reimplementa el cableado de
  contextos: una ruta nueva queda cubierta sin duplicar código.
- Un **esquema de PostgreSQL aislado** (`internal/testutil.Postgres`), que
  aplica las migraciones y se destruye al terminar.
- **Redis real**, porque los controles de identidad (OTP, intentos) fallan
  cerrados sin él.
- **Almacenamiento local temporal** para los adjuntos.
- Un **buzón en memoria** que captura los correos, de modo que la suite lee
  el código OTP sin depender de Mailpit.

### Ejecutarla

```bash
cd apps/backend
E2E_TEST_DATABASE_URL="postgres://postgres@localhost:5432/biess?sslmode=disable" \
REDIS_TEST_URL="redis://localhost:6379/0" \
go test ./e2e/ -v
```

Sin esas variables la suite se omite (`t.Skip`), con el mismo criterio que
las pruebas de integración de PostgreSQL.

### Qué cubre

| Fichero | Cobertura |
|---|---|
| `identidad_test.go` | Healthchecks, `X-Request-ID`, envelope de error, 401 sin sesión, CSRF, login, cookie HttpOnly, sesión, cierre, bloqueo por intentos y `Retry-After` |
| `autorizacion_test.go` | RBAC por rol (postulante, analista, administrador), catálogos geográficos y la cascada país→provincia, titularidad del expediente, paginación por offset y por cursor |
| `postulacion_test.go` | Guardado por sección, nacionalidad, sellado y hash, inmutabilidad del enviado, impresión PDF, progreso, auditoría, rutas públicas |
| `registro_test.go` | Registro con OTP real, código inválido, cédula inválida, recuperación de contraseña y protección anti-oráculo |

---

## 4. Suite de interfaz (`apps/frontend/e2e/`)

### Fixtures compartidos (`e2e/fixtures/`)

| Fichero | Qué aporta |
|---|---|
| `datos.ts` | Cuentas, expedientes y convocatoria sembrados; lectura del archivo de seed |
| `sesion.ts` | Login por API y por navegador, lectura del OTP desde Mailpit, generador de PDF |
| `cedulas.ts` | Cédulas ecuatorianas válidas (módulo 10, provincia 1–24, tercer dígito < 6) y correos únicos |

`global-setup.ts` ejecuta el seed antes de la corrida y publica el archivo de
estado.

### Ejecutarla

```bash
cd apps/frontend
pnpm test:e2e
```

Requiere el stack de desarrollo en pie (`docker compose up -d`).

### Reglas de los specs

- **No hay datos hardcodeados**: las cuentas y los expedientes vienen del
  seed. Un spec que necesita una cuenta nueva la registra por la UI, pero
  con cédula válida y leyendo el OTP con `leerOTP`.
- **Una cuenta, un expediente**: el seed no puede sembrar dos expedientes
  para la misma cuenta en la misma convocatoria; el dominio lo rechaza.
- **Un test que descubre un bug no se maquilla**: se marca `test.skip` con un
  comentario que dice qué falla y dónde, y el bug queda reportado. Cambiar
  la aplicación excede el alcance de la suite.

---

## 5. Correcciones de infraestructura que habilitaron la suite

- **Mailpit en la red del compose.** Estaba con `network_mode: host` mientras
  la API vivía en la red del bridge: `SMTP_HOST: localhost` apuntaba al propio
  contenedor y ningún correo llegaba. Ahora Mailpit publica `8025` y `1025` y
  la API le habla por su nombre de servicio.
- **`appkit`**. El router estaba cableado dentro de `cmd/api/main.go`; sin
  extraerlo, la suite de API habría tenido que duplicar cien líneas de
  cableado y habría divergido del binario real.

---

## 6. Qué no cubre la suite

- El módulo de oposición completo (cálculo de méritos publicado, banco de
  preguntas versionado, configuración, matrícula, attempt y calificación) no
  tiene seed. El escenario del examen queda fuera hasta que exista.
- La exportación a XLSX y la firma manuscrita dependen de datos que el seed
  no publica.
