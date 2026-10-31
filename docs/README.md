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
| [`api-contracts/REVIEW_AUDIT.md`](api-contracts/REVIEW_AUDIT.md) | T8/T9: historial, períodos, nómina, conteos y propuesta de matriz XLSX/PDF | Backend, Frontend, QA |
| [`deployment/LOCAL_DEPLOYMENT.md`](deployment/LOCAL_DEPLOYMENT.md) | Prerrequisitos, arranque con Docker o nativo, variables de entorno, pruebas | Desarrollo |
| [`deployment/PRODUCTION_DEPLOYMENT.md`](deployment/PRODUCTION_DEPLOYMENT.md) | Topología, TLS, secretos, backup, monitoreo, rollback | Operaciones |
| [`tasks/backend-tasks.md`](tasks/backend-tasks.md) | Tareas backend priorizadas y mapeadas a los 6 criterios de la rúbrica | Ingeniería |
| [`tasks/frontend-tasks.md`](tasks/frontend-tasks.md) | Tareas frontend priorizadas y mapeadas a los 6 criterios de la rúbrica | Ingeniería |

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
