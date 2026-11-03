# ADR-002 — Bounded contexts con puertos y adaptadores

| Campo | Valor |
|---|---|
| ADR | 002 |
| Estado | **Aceptada** |
| Fecha | 2026-10-30 |
| Ámbito | Backend (organización del código) |
| Referencia | [`../architecture/ARQUITECTURA.md`](../architecture/ARQUITECTURA.md) |

---

## Contexto

El sistema tiene cuatro dominios claramente separados: **identidad**
(registro, autenticación, sesiones), **postulación** (el formulario EF-12),
**documentos** (adjuntos PDF) y **revisión** (dashboard del administrador). El
RF los describe con datos y reglas propios.

El enunciado exige "organización en capas limpias mediante Inyección de
Dependencias (Handler/Controller → Service/Use Case → Repository)". La
pregunta es **dónde** se cortan las capas, porque de eso depende que el
dominio sea realmente independiente de la infraestructura.

## Decisión

Se adopta **arquitectura de bounded contexts con puertos y adaptadores**
(hexagonal), organizada así:

```text
internal/
├── platform/          # transversal: config, db, httpx, logger, mail, cache, storage
├── shared/domain/     # kernel: errores base y paginación (sin negocio)
├── identity/          # contexto: postulante, registro, login, sesión
├── submission/        # contexto: formulario EF-12
├── document/          # contexto: adjuntos PDF
├── catalog/           # contexto: catálogos geográficos
└── review/            # contexto: administración y dashboard
```

Dentro de cada contexto, cada capacidad se compone de archivos con
responsabilidad única, **sin** carpetas `domain/`, `application/` e
`infrastructure/`:

```text
internal/<contexto>/
├── service.go     # casos de uso
├── handler.go     # adaptador HTTP
├── repository.go  # puerto (interface)
├── dto.go         # objetos de la capa HTTP
├── errors.go      # errores del dominio
└── postgres/      # adaptador de persistencia
```

Reglas de dependencia:

1. Las **reglas puras** (entidades, canonización, hash, validaciones) no
   importan nada de infraestructura.
2. Los **casos de uso** dependen solo de interfaces declaradas en su propio
   paquete.
3. Los **adaptadores** implementan esos puertos y traducen errores de
   infraestructura a errores de dominio.
4. `shared/domain/` solo contiene primitivas transversales.
5. `platform/` no contiene reglas de negocio.
6. Los contextos **no comparten entidades**: se referencian por identificador.

## Alternativas consideradas

### Capas técnicas (`api/`, `service/`, `repository/`)

- ✅ Familiar y fácil de explicar.
- ❌ Todo el dominio del concurso queda disperso en las tres carpetas: el
  formulario EF-12, la cédula y el dashboard quedan en archivos distintos del
  mismo contexto.
- ❌ "Shared" y "utils" se convierten en cajones de sastre: cualquier código
  acaba dependiendo de "lo común".
- ❌ Un cambio en el RF obliga a tocar archivos de tres carpetas.

### Clean Architecture strict (domain/application/infrastructure en cada módulo)

- ✅ Desacoplamiento máximo, muy testeable.
- ❌ Para un proyecto de este tamaño, **tres carpetas por módulo** son ruido:
  el reparto de archivos ya garantiza la misma separación.
- ❌ Mucha ceremonia en módulos pequeños (un solo archivo de dominio acaba en
  una carpeta propia).

### Carpeta por feature (la elegida)

- ✅ Las carpetas **dicen qué hace el sistema**: `identity/`, `submission/`,
  `catalog/`.
- ✅ Cada contexto tiene su propio modelo y sus propias reglas; no hay
  entidades compartidas que se filtren.
- ✅ La granularidad por archivo (`service.go`, `handler.go`, …) da el mismo
  desacoplamiento que la opción anterior, con menos carpetas.
- ✅ El cableado queda en un único lugar: `cmd/api/main.go`.
- ✅ Escala bien: añadir `document/` es crear una carpeta, no reorganizar el
  árbol.

## Consecuencias

### Positivas

- El contexto `submission` es una **referencia completa**: cualquier otro
  contexto se implementa copiando su forma, no su código.
- Las reglas de negocio se prueban **sin base de datos ni servidor**.
- Un cambio en el RF (por ejemplo, las 9 secciones) toca **un solo contexto**.
- La separación es visible en el árbol de archivos, lo que hace la arquitectura
  comprobable en la sustentación sin necesidad de explicarla.

### Negativas

- Duplicación menor de tipos primitives entre contextos (p. ej. el `ID` de
  postulante): se acepta a cambio de no compartir el modelo.
- `shared/domain/` puede crecer sin control si no se vigila: hoy contiene
  solo errores base y paginación, y debe seguir así.
- La frontera entre contextos requiere un contrato explícito cuando uno
  consulta datos de otro (p. ej. `review` necesita saber si un postulante
  tiene postulación enviada): hoy se resuelve por `applicant_id`.

## Nota sobre la interoperabilidad con el IESS

**Actualización 2026-10-31:** Registro Civil no se simula mediante fecha de
expedición o código dactilar. El proyecto adopta **OTP por correo + módulo 10**
para registro y recuperación ([ADR-008](ADR-008-registro-y-recuperacion-con-otp.md)).
El OTP verifica el acceso al correo; no equivale a certificar datos de identidad.
Los datos restantes se declaran en EF-12.

Historial Laboral conserva un puerto independiente y una fuente simulada en el
servidor para elegibilidad mientras no haya acceso real; su consolidación está
pendiente en T17. La simulación no debe depender de un booleano del cliente.

## Referencias

- Contexto de referencia: `apps/backend/internal/submission/`
- Cableado: `apps/backend/cmd/api/main.go`
- Errores base: `apps/backend/internal/shared/domain/errors.go`
- Rúbrica, criterio 1 (8 pts): arquitectura en capas e inyección de dependencias.
