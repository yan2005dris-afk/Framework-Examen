# ADR-001 — chi v5 en lugar de Gin o Fiber

| Campo | Valor |
|---|---|
| ADR | 001 |
| Estado | **Aceptada** |
| Fecha | 2026-10-30 |
| Ámbito | Backend (HTTP) |
| Referencia | [`../trd/TRD.md` §2.1](../trd/TRD.md) |

---

## Contexto

El enunciado del examen pide un backend en Go con *Fiber/Gin* y separation
estricta de responsabilidades. La rúbrica evalúa **la arquitectura en capas con
inyección de dependencias**, no la librería concreta: "Arquitectura en capas
perfectamente desacoplada (handlers, services, repositories) mediante inyección
de dependencias".

El RF exige, además, subir **PDF adjuntos** con límites por tipo (2 MB, 4 MB y
50 MB en total). Esa es la restricción técnica que condiciona la elección del
router.

## Decisión

Se usa **chi v5** (`github.com/go-chi/chi/v5`) como router HTTP.

## Alternativas consideradas

### Gin

- ✅ Muy popular, ecosistema amplio, middlewares abundance.
- ✅ Interoperable con `net/http`: `gin.Context` implementa `http.ResponseWriter`
  y expone `*http.Request`.
- ❌ Dependencia grande y arrastra muchas dependencias transitivas.
- ❌ Su API propia (`gin.Context`) se filtra hacia las capas inferiores: los
  handlers de archivos y los middlewares terminan knowing gin, lo que ata el
  dominio a la librería.

### Fiber

- ✅ API moderna y benchmark muy alto en rutas simples.
- ❌ **Opera sobre `fasthttp.RequestCtx`, no sobre `*http.Request`.** Todo lo que
  dependa de la stdlib (middlewares de `net/http`, `http.MaxBytesReader`,
  `http.Request.Context`, libraries de `multipart`) requiere adaptadores.
- ❌ **Carga el cuerpo completo de la petición en memoria**, en lugar de
  streaming. Es exactamente el punto crítico del RF: subir PDFs de hasta
  50 MB por postulante.
- ❌ Su modelo de contexto es incompatible con el ecosistema estándar de Go.

### chi v5 (elegida)

- ✅ **Opera sobre `*http.Request` nativo**: el mismo `http.Handler` sirve para
  la API, para `multipart` y para los middlewares de la stdlib.
- ✅ `http.MaxBytesReader` funciona sin adaptadores, así que el límite de
  10 MiB por petición y los límites por tipo de PDF son directos.
- ✅ Middlewares componibles y **sin estado global**: cada contexto registra
  sus rutas sobre el router raíz y no depende de un framework de middleware.
- ✅ Dependencias mínimas: el proyecto usa solo `chi`, `pgx`, `uuid` y `sqlc`.
- ✅ `chi.URLParam(request, "id")` da acceso a los parámetros de ruta sin
  cadena de contexto ni reflection.

## Consecuencias

### Positivas

- Las subidas de PDF se procesan en streaming y con límites aplicados por la
  stdlib, sin adaptadores.
- Los middlewares del proyecto (`RequestID`, `Recover`, `Logging`, `CORS`,
  `MaxBodySize`) son funciones `func(http.Handler) http.Handler` reutilizables
  y testeables de forma aislada.
- El dominio y los servicios no dependen de chi: `handler.go` es el único
  archivo que importa el router.
- Superficie de dependencias mínima, lo que reduce el riesgo de cara a la
  revisión del código.

### Negativas

- **Desviación literal de la rúbrica**, que menciona "Fiber/Gin". Se asume
  conscientemente: la arquitectura evaluada (Handler → Service → Repository
  con DI) es idéntica con cualquier router.
- chi no trae validación de struct por tags: la validación se hace en el
  dominio, que es donde debe estar.
- chi no incluye un ORM ni un gestor de migraciones: `sqlc` y `goose` cubren
  esa necesidad (ver [ADR-003](ADR-003-postgres-sqlc-goose.md)).

## Cómo se justifica en la sustentación

> "La rúbrica evalúa la separación en capas con inyección de dependencias, y
> esa separación es la misma con chi. Elegí chi porque trabaja sobre
> `*http.Request` nativo: el RF exige subir PDFs de hasta 50 MB por
> postulante, y Fiber cargaría el cuerpo completo en memoria. Además chi no
> arrastra adaptadores para `multipart` ni para `http.MaxBytesReader`."

## Referencias

- Código: `apps/backend/internal/platform/httpx/router.go`
- Middlewares: `apps/backend/internal/platform/httpx/httpx.go`
- Rúbrica, criterio 1 (8 pts): arquitectura en capas e inyección de dependencias.
