# ADR-004 — Identificadores en inglés, comentarios en español

| Campo | Valor |
|---|---|
| ADR | 004 |
| Estado | **Aceptada** |
| Fecha | 2026-10-30 |
| Ámbito | Todo el proyecto |
| Referencia | [`../../AGENTS.md`](../../AGENTS.md) |

---

## Contexto

El proyecto maneja dos idiomas con roles distintos:

- El **dominio del negocio** es el Requerimiento Funcional del IESS, en
  español: "postulación", "cédula", "sección", "inhabilidades", "provincia".
- El **código** debe ser legible para un equipo técnico y interoperable con el
  ecosistema de Go y con las librerías de terceros.

Además, el enunciado del examen y el proyecto usan un vocabulario propio
(`submission`, `applicant`, `service`, `repository`) que es en inglés.

## Decisión

### Backend (Go)

- **Identificadores en inglés**: funciones, métodos, tipos, campos, variables y
  constantes.
- **Comentarios, mensajes de error y documentación en español**, con bloques
  JSDoc (`/** … */`) que incluyen `@param` y `@returns`.

```go
/**
 * CanonicalContent normaliza el contenido del formulario a un JSON estable:
 * con las claves ordenadas, el mismo formulario siempre produce el mismo
 * hash, sin importar el orden en que llegue del cliente.
 *
 * @param {json.RawMessage} content - Contenido crudo del formulario.
 * @returns {json.RawMessage} Contenido canónico.
 * @throws {Error} ErrInvalidContent si el JSON no es un objeto válido.
 */
func CanonicalContent(content json.RawMessage) (json.RawMessage, error) {
```

### Frontend (TypeScript)

- **Identificadores y dominio en español**: `esValida`,
  `calcularDuracionExperiencia`, `seccion1`, `cumpleMayuscula`, `esValidoEmail`.
- Esto refleja que el frontend modela directamente el formulario EF-12: los
  nombres hablan el idioma del postulante.

## Alternativas consideradas

### Todo en español

- ✅ Coherente con el RF y con el lector del dominio.
- ❌ Choca con la stdlib de Go y con las librerías: `errors.New`, `json.Marshal`,
  `http.StatusOK`. Un mixto forzado produce inconsistencias.
- ❌ Los términos del RF ("inhabilidades", "sección") no mapean de forma natural
  a nombres de tipos y métodos.

### Todo en inglés

- ✅ Coherente con Go y con el ecosistema.
- ❌ El equipo y la evaluación leen el RF en español: el código se vuelve
  opaco para quien conoce el negocio.
- ❌ Pierde la trazabilidad entre el requisito (`EF-12 punto 1.9`) y el código.

### Inglés en identificadores + español en comentarios (elegida)

- ✅ El código es idiomático en Go y las librerías se usan sin fricción.
- ✅ Los comentarios explican el **porqué** en el idioma del RF, que es donde
  está el conocimiento de negocio.
- ✅ El frontend, que sí modela el formulario, habla en español.

## Consecuencias

### Positivas

- Los comentarios JSDoc son la documentación de la API interna: se leen en
  español y explican la regla, no la mecánica.
- El vocabulario del RF es rastreable: `ErrNotEditable` documenta *"la
  postulación ya fue enviada y no admite cambios"*.
- Quien lee el RF puede revisar el código sin translator.

### Negativas

- **Un idioma por archivo no es lo habitual**: se ve en los archivos de Go,
  donde el nombre del símbolo es inglés y el comentario español. Es una
  decisión consciente, no un descuido.
- La documentación generada por `godoc` sale en español, lo que es inusual en
  un proyecto Go. Aceptado: el público del proyecto es el equipo del
  concurso.
- Hay que vigilar que los mensajes de error al usuario **siempre** salgan en
  español (el frontend los muestra tal cual).

## Verificación

- Los mensajes de error del backend son la prueba: `"la cédula debe tener
  exactamente 10 dígitos numéricos"`, `"El cuerpo de la petición no es
  válido."`.
- El CI ejecuta `gofmt` y `go vet`, que no verifican el idioma; la revisión
  humana del PR es el control.

## Referencias

- Convenciones del proyecto: [`../../AGENTS.md`](../../AGENTS.md)
- Ejemplo de dominio: `apps/backend/internal/submission/submission.go`
- Ejemplo de frontend: `apps/frontend/src/utils/validators.ts`
