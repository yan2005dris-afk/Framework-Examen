# ADR-006 — Sesiones en PostgreSQL con cookie HttpOnly

| Campo | Valor |
|---|---|
| ADR | 006 |
| Estado | **Aceptada** |
| Fecha | 2026-10-30 |
| Ámbito | Backend (seguridad) y Frontend (cliente) |
| Referencia | [`../trd/TRD.md` §6](../trd/TRD.md) |

---

## Contexto

El postulante inicia sesión con **cédula y contraseña** (HU-03) y la sesión
debe proteger el formulario EF-12. Requisitos que se cruzan:

- La contraseña nunca se almacena en claro (RN-04): hash **Argon2id** con sal
  aleatoria.
- La sesión debe **caducar** (TTL configurable) y poder **revocarse** al
  cerrar sesión.
- El token **no debe ser accesible desde JavaScript** (RN-08): si el
  frontend lo guardara en `localStorage`, cualquier XSS podría robarlo.
- La API debe poder escalar a **varias instancias** sin sesiones pegadas a un
  proceso (RNF-03).

## Decisión

- **Sesiones almacenadas en PostgreSQL**, en la tabla `sessions`.
- **Token opaco aleatorio de 256 bits** (32 bytes) en base64url.
- En la base se guarda **solo el SHA-256 del token**, nunca el token en
  claro: una lectura no autorizada de la tabla no permite suplantar a nadie.
- El token viaja en una **cookie `HttpOnly`**:
  `HttpOnly; SameSite=Lax; Secure` (en producción) + `Path=/`.
- La **consulta de sesión es un `SELECT` por petición**: con el índice único
  sobre `token_hash` es un acceso O(log n) y no requiere una ida y vuelta al
  cliente.
- El cliente **no interpreta** el token: solo reenvía la cookie.

## Alternativas consideradas

### JWT (token firmado)

- ✅ Sin estado: no requiere consulta a la base en cada petición.
- ✅ Se puede verificar localmente.
- ❌ **Revocación difícil**: un JWT válido lo es hasta que expira, aunque el
  usuario cierre sesión o un administrador bloquee la cuenta. En un concurso
  con datos sensibles, la revocación inmediata es un requisito, no una
  comodidad.
- ❌ Se puede **usar como contraseña** si se filtra: el token no se puede
  reemitir ni invalidar.
- ❌ Su tamaño (varios cientos de bytes) viaja en **cada** petición.
- ❌ Almacenado en `localStorage` (lo habitual con JWT), es vulnerable a XSS.

### Sesiones en memoria del proceso

- ✅ Simple, sin esquema de base de datos.
- ❌ Se pierden al reiniciar: el usuario cae de la sesión sin aviso.
- ❌ No escalan: con dos instancias, la sesión existe en una sola.
- ❌ El enunciado penaliza explícitamente los "estados temporales en memoria
  volátil de la aplicación" (criterio 3 de la rúbrica).

### Sesiones en Redis

- ✅ Rápido y con TTL nativo.
- ❌ Redis es el almacén de **efímeros** (bloqueo de intentos y caché de
  catálogos). Meter ahí las sesiones mezcla dos ida y vuelta con ciclos de
  vida distintos.
- ❌ Una pérdida de Redis cierra todas las sesiones de golpe.
- ❌ La sesión es un dato que debe **durables** en el concurso: conviene que
  viva junto al resto del dato transaccional, en la base.

### Sesiones en PostgreSQL (elegida)

- ✅ **Revocación inmediata**: borrar la fila cierra la sesión.
- ✅ Sobrevive a reinicios y escala horizontalmente.
- ✅ Auditoría: la tabla registra IP y agente de usuario, y permite listar
  sesiones activas de un postulante.
- ✅ Reutiliza la infraestructura ya presente (pool, transacciones, backup).
- ✅ La cookie `HttpOnly` protege el token frente a XSS.

## Consecuencias

### Positivas

- Cerrar sesión es un `DELETE` verificable, no una expiración diferida.
- Un administrador puede invalidar todas las sesiones de un postulante.
- La cookie `HttpOnly` + `SameSite=Lax` + `Secure` da una defensa en capas
  frente a XSS y CSRF.
- El token nunca se persiste en claro: el mayor riesgo (filtración de la
  base) no compromete a los usuarios.

### Negativas

- **Una consulta a la base por petición autenticada.** Se mitiga con el
  índice único sobre `token_hash` y con un pool bien dimensionado. Para el
  volumen del parcial es perfectamente aceptable.
- La tabla `sessions` **crece**: hay que purgarla. Se resuelve con una
  tarea periódica que borra las filas expiradas.
- La cookie `SameSite=Lax` **no** bloquea el envío de cookies en peticiones
  cross-site de nivel superior (un `<form>` desde otro dominio). Como el
  único método mutante es `POST` con `Content-Type: application/json`, y
  los formularios HTML no pueden enviar `application/json`, el riesgo real es
  bajo. Para `multipart` (subida de PDF) habría que revisar el
  `SameSite`.
- Requiere la tabla `sessions`, que **todavía no existe**: es parte del
  trabajo pendiente del contexto `identity`.

## Parámetros

| Parámetro | Valor por defecto | Variable |
|---|---|---|
| Vigencia de la sesión | 24 h | `SESSION_TTL` |
| Longitud del token | 32 bytes (256 bits) | — |
| Algoritmo del hash | SHA-256 | — |
| Cookie | `HttpOnly; SameSite=Lax` (+ `Secure` en producción) | — |

## Referencias

- Tabla planificada: `sessions` en [`../database/SCHEMA.md`](../database/SCHEMA.md)
- Requisitos: RN-04, RN-08, RNF-03, RNF-04 del PRD
- Contexto de identidad: `apps/backend/internal/identity/`
- Rúbrica, criterio 5 (6 pts): enrutamiento protegido por sesión.
