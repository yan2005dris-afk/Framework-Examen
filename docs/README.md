# Documentación del proyecto

**Sistema de Registro y Postulación al Directorio del BIESS**
Módulo inicial de registro y postulación para el Concurso Público de Méritos y
Oposición del Directorio del Banco del Instituto Ecuatoriano de Seguridad Social.

Este directorio es la fuente de verdad documental del proyecto. La documentación
vive aquí —y no en el `README.md` de la raíz ni en `AGENTS.md`— para poder
evolucionar sin duplicar información: el `README.md` de la raíz es la puerta de
entrada y `AGENTS.md` es la guía operativa para agentes de IA.

---

## Mapa de la documentación

```text
docs/
├── README.md                     # este índice
├── requerimiento.txt             # texto del RF del IESS (fuente funcional, versionada)
├── requerimiento.pdf             # PDF original del RF (local, no versionado: regla *.pdf)
├── mockups/                      # mockups HTML de las 36 pantallas (referencia de migración)
│   ├── index.html                # galería: se abre en el navegador, sin servidor
│   ├── README.md                 # mapa pantalla → HU → componente shadcn
│   ├── assets/                   # tokens.css, app.css (contrato de clases), mock.js
│   ├── fixtures/                 # fuente de verdad de los datos del expediente de ejemplo
│   └── scripts/                  # check-consistencia.mjs, check-contrato.mjs
├── prd/
│   └── PRD.md                    # qué se construye y por qué (producto)
├── trd/
│   └── TRD.md                    # cómo se construye (técnico)
├── architecture/
│   └── ARQUITECTURA.md           # vistas del sistema, capas y ciclo de una petición
├── database/
│   └── SCHEMA.md                 # modelo de datos, DDL, índices y semillas
├── api-contracts/
│   └── API_CONTRACT.md           # contrato HTTP: endpoints, errores, paginación
├── decisions/
│   ├── ADR-001-chi-en-vez-de-gin-fiber.md
│   ├── ADR-002-bounded-contexts-y-hexagonal.md
│   ├── ADR-003-postgres-sqlc-goose.md
│   ├── ADR-004-ingles-en-codigo-espanol-en-comentarios.md
│   ├── ADR-005-paginacion-offset-y-cursor.md
│   ├── ADR-006-sesiones-en-base-de-datos.md
│   ├── ADR-007-postgresql-18-y-uuidv7.md
│   └── ADR-008-registro-y-recuperacion-con-otp.md
├── deployment/
│   ├── LOCAL_DEPLOYMENT.md       # entorno de desarrollo
│   └── PRODUCTION_DEPLOYMENT.md  # entorno de producción
└── tasks/
    ├── backend-tasks.md          # trabajo pendiente del backend, mapeado a la rúbrica
    ├── cross-cutting-tasks.md    # tareas transversales de seguridad, contratos y operación
    └── frontend-tasks.md         # trabajo pendiente del frontend, mapeado a la rúbrica
```

---

## Documentos principales

| Documento | Contenido | Audience principal |
|---|---|---|
| [`prd/PRD.md`](prd/PRD.md) | Alcance, actores, 12 historias de usuario, reglas de negocio, trazabilidad RF→HU→código, riesgos | Producto, Evaluación académica |
| [`trd/TRD.md`](trd/TRD.md) | Stack justificado, estructura del proyecto, modelo de datos, API, seguridad, Redis, transacciones, CI | Ingeniería, Evaluación técnica |
| [`architecture/ARQUITECTURA.md`](architecture/ARQUITECTURA.md) | Diagramas C4 del sistema, reglas de dependencias, ciclo de una petición, guía para agregar un contexto | Ingeniería |
| [`database/SCHEMA.md`](database/SCHEMA.md) | ER, DDL por tabla, índices, semillas de catálogos, flujo de migraciones | Ingeniería, DBA |
| [`api-contracts/API_CONTRACT.md`](api-contracts/API_CONTRACT.md) | Envelope de errores, paginación, especificación de cada endpoint con ejemplos | Frontend, QA |
| [`api-contracts/IDENTITY_SECURITY.md`](api-contracts/IDENTITY_SECURITY.md) | Corrección OTP/Redis y notificación durable; configuración y límites | Backend, Frontend, Operaciones |
| [`api-contracts/REVIEW_AUDIT.md`](api-contracts/REVIEW_AUDIT.md) | T8/T9: historial, períodos, nómina, conteos y propuesta de matriz XLSX/PDF | Backend, Frontend, QA |
| [`api-contracts/COMPETITION_LIFECYCLE.md`](api-contracts/COMPETITION_LIFECYCLE.md) | T15–T18: contrato de correo, OTP, elegibilidad, convocatorias y fases para FT14–FT17 | Backend, Frontend, QA |
| [`deployment/LOCAL_DEPLOYMENT.md`](deployment/LOCAL_DEPLOYMENT.md) | Prerrequisitos, arranque con Docker o nativo, variables de entorno, pruebas | Desarrollo |
| [`deployment/PRODUCTION_DEPLOYMENT.md`](deployment/PRODUCTION_DEPLOYMENT.md) | Topología, TLS, secretos, backup, monitoreo, rollback | Operaciones |
| [`tasks/backend-tasks.md`](tasks/backend-tasks.md) | Tareas backend priorizadas y mapeadas a los 6 criterios de la rúbrica | Ingeniería |
| [`tasks/frontend-tasks.md`](tasks/frontend-tasks.md) | Tareas frontend priorizadas y mapeadas a los 6 criterios de la rúbrica | Ingeniería |
| [`mockups/index.html`](mockups/index.html) | Galería de las 36 pantallas del sistema en HTML estático (sin build) | Frontend, Diseño |
| [`mockups/README.md`](mockups/README.md) | Mapa pantalla → historia de usuario → componente shadcn, tokens y reglas del set | Frontend |

---

## Decisiones de arquitectura (ADR)

Los ADR registran las decisiones con contexto, alternativas descartadas y
consecuencias. Son cortos y se leen en minutos.

| ADR | Decisión | Estado |
|---|---|---|
| [ADR-001](decisions/ADR-001-chi-en-vez-de-gin-fiber.md) | chi v5 en lugar de Gin o Fiber | Aceptada |
| [ADR-002](decisions/ADR-002-bounded-contexts-y-hexagonal.md) | Bounded contexts con puertos y adaptadores | Aceptada |
| [ADR-003](decisions/ADR-003-postgres-sqlc-goose.md) | PostgreSQL + sqlc + goose | Aceptada |
| [ADR-004](decisions/ADR-004-ingles-en-codigo-espanol-en-comentarios.md) | Identificadores en inglés, comentarios en español | Aceptada |
| [ADR-005](decisions/ADR-005-paginacion-offset-y-cursor.md) | Paginación offset y cursor sobre un kernel compartido | Aceptada |
| [ADR-006](decisions/ADR-006-sesiones-en-base-de-datos.md) | Sesiones en PostgreSQL con cookie `HttpOnly` | Aceptada |
| [ADR-007](decisions/ADR-007-postgresql-18-y-uuidv7.md) | PostgreSQL 18 y UUIDv7 como estrategia de identificadores | Aceptada |
| [ADR-008](decisions/ADR-008-registro-y-recuperacion-con-otp.md) | Registro y recuperación con OTP por correo, sin Registro Civil | Aceptada |

---

## Alcance funcional y adaptación vigente

`requerimiento.txt` conserva el RF v1.4 original. Las tareas de backend y frontend
incluyen su cobertura integral, también EF-13/EF-14 y los procesos generales de
méritos, psicotécnica y reconsideraciones. Registro y recuperación se adaptan a
OTP por correo (ADR-008): fecha de expedición y código dactilar no son requisitos
del proyecto. Las matrices de las tareas separan planificación de implementación.

## Cómo leer la documentación

- **¿Qué tiene que hacer el sistema?** → [`prd/PRD.md`](prd/PRD.md)
- **¿Cómo está construido?** → [`trd/TRD.md`](trd/TRD.md) y
  [`architecture/ARQUITECTURA.md`](architecture/ARQUITECTURA.md)
- **¿Cómo se guarda?** → [`database/SCHEMA.md`](database/SCHEMA.md)
- **¿Cómo se consume la API?** → [`api-contracts/API_CONTRACT.md`](api-contracts/API_CONTRACT.md)
- **¿Por qué está hecho así?** → [`decisions/`](decisions/)
- **¿Cómo lo levanto?** → [`deployment/LOCAL_DEPLOYMENT.md`](deployment/LOCAL_DEPLOYMENT.md)
- **¿Qué falta?** → [`tasks/`](tasks/)
- **¿Cómo se ve cada pantalla?** → [`mockups/index.html`](mockups/index.html) (abre
  `docs/mockups/index.html` en el navegador) y la guía de migración en
  [`mockups/README.md`](mockups/README.md)

## Convenciones de los documentos

- Idioma: **español**. Los identificadores del código, en cambio, siguen la
  convención del proyecto: inglés en Go, español en el dominio del frontend.
- Los documentos marcan explícitamente lo que está **implementado** y lo que
  está **pendiente**. La sección de tareas es la lista viva de lo pendiente.
- Las referencias al código usan rutas reales del repositorio; si una ruta está
  marcada como *planificada*, aún no existe en el árbol.

---

**Ver también**: [`../README.md`](../README.md) (raíz) ·
[`../AGENTS.md`](../AGENTS.md) (guía para agentes de IA)


### Entrega backend T19/T20

- [Contrato de impresión y méritos](api-contracts/MERITS_AND_PRINTING.md): rutas,
  permisos, reglas sin puntajes oficiales por defecto y correcciones versionadas.


### Entrega frontend FT14/FT15

- [Portal y gestión de convocatorias](tasks/FT14-FT15.md): pantallas, límites del
  contrato T18, confirmaciones y evidencia local de escritorio/móvil.

### Entrega backend T21/T22

- [Contrato de evaluación y reconsideraciones](api-contracts/EVALUATION_RECONSIDERATION.md):
  registro psicotécnico, expediente consolidado, plazos por actor y correcciones.
  Sin valores institucionales precargados; el resultado teórico depende de T27.
