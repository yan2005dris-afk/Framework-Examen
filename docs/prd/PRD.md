# Documento de Requisitos del Producto (PRD)

**Sistema de Registro y Postulación al Directorio del BIESS**
Concurso Público de Méritos y Oposición — Directorio del Banco del Instituto
Ecuatoriano de Seguridad Social (BIESS)

| Campo | Valor |
|---|---|
| Documento | PRD v1.1 |
| Fuente funcional | `docs/requerimiento.txt` (texto del Requerimiento Funcional v1.4, 26/03/2026; el PDF original se conserva localmente) |
| Marco legal | Resolución No. C.D. 701 — Disposición Transitoria Primera |
| Ámbito del parcial | Módulo inicial de registro y postulación (EF-11, EF-12 y parte de EF-13) |
| Asignatura | Frameworks de Programación Web |
| Estado | En construcción — módulo inicial y backlog integral del RF diferenciados |

---

## 1. Resumen ejecutivo y objetivos

El Instituto Ecuatoriano de Seguridad Social (IESS) requiere un **módulo web
integral** para automatizar el Concurso Público de Méritos y Oposición por el
que se seleccionan los Miembros Principales y Alternos del Directorio del BIESS,
en representación de los sectores **Afiliados, Jubilados y Empleadores**.

Gestionar hoy la recepción, verificación, calificación y notificación de cientos
de postulantes con hojas de cálculo y correos sueltos es inviable desde el
punto de vista operativo, multiplica el error humano y deja vacíos de
auditoría frente a los órganos de control.

### 1.1 Objetivos principales

1. **Digitalizar el ingreso** de postulantes con verificación de identidad y
   control de intentos fallidos (EF-11).
2. **Capturar el formulario EF-12** completo en 9 secciones, con guardado de
   progreso, adjuntos PDF y validación de reglas de elegibilidad.
3. **Asegurar la trazabilidad** de cada postulación: código único, historial de
   estados, hash criptográfico del contenido enviado y notificaciones
   automáticas.
4. **Dotar a Talento Humano** de un módulo de administración con dashboard en
   tiempo real y exportación de la matriz comparativa (EF-13).

### 1.2 Objetivos técnicos (evaluables en la rúbrica)

| # | Objetivo | Cómo se evidencia |
|---|---|---|
| T1 | Arquitectura en capas con inyección de dependencias | Handler → Service → Repository por bounded context (`internal/<contexto>/`) |
| T2 | Persistencia relacional con transacciones atómicas | PostgreSQL + `sqlc` + migraciones `goose` |
| T3 | Caché y control de intentos con Redis | Redis con TTL: bloqueo de 10 min y caché de catálogos |
| T4 | SPA modular y reactiva | React 19 + Vite + Tailwind v4 + shadcn/ui, estado con TanStack Query |
| T5 | Consumo asíncrono con rutas protegidas | Cliente HTTP tipado, `async/await`, interceptor de sesión |
| T6 | Reglas de negocio del RF cumplidas en servidor | Validación en dominio, no solo en el formulario |

---

## 2. Contexto y justificación

- **Origen**: Requerimiento FuncIONAL v1.4 emitido por la Subdirección Nacional
  de Tecnologías de la Información (IESS), a solicitud de la Subdirección
  Nacional de Gestión de Talento Humano.
- **Obligación**: la Disposición Transitoria Primera de la Resolución C.D. 701
  otorga **30 días** desde su suscripción para publicar el formulario de
  postulación en los portales institucionales.
- **Plazo del proyecto**: el examen prático evalúa el **módulo inicial**
  (registro + postulación). Los módulos de banco de preguntas y evaluación
  teórica (EF-14) se documentan aquí pero **no forman parte de la entrega**.

---

## 3. Alcance

### 3.1 Dentro del alcance del parcial

| Código | Especificación funcional | Estado |
|---|---|---|
| EF-11 | Registrar postulantes y autenticar su acceso | Base OTP y vistas de autenticación disponibles; consolidación T16/FT16 pendiente (ADR-008) |
| EF-12 | Formulario de postulación (9 secciones) | Parcial (CRUD y dominio listos; falta el flujo completo de secciones) |
| EF-13 | Módulo de administración | Parcial (dashboard en el mockup; falta matriz y exportación) |

### 3.2 Fuera del alcance del parcial

| Código | Especificación funcional | Motivo |
|---|---|---|
| EF-14 | Banco de preguntas y evaluación teórica | No pertenece al módulo inicial evaluado |
| — | Interoperabilidad real con Registro Civil e Historial Laboral del IESS | Registro Civil sustituido por módulo 10 + OTP (ADR-008). Historial Laboral se abstrae con una fuente simulada de servidor pendiente de T17 |
| — | Notificaciones a cientos de postulantes en producción | El envío SMTP queda implementado; la campaña masiva es del módulo de administración |

---

### 3.3 Cobertura del producto completo y adaptación de identidad

Las exclusiones del parcial no convierten EF-13, EF-14, méritos, oposición
psicotécnica, reconsideraciones y listas de elegibles en mejoras opcionales.
Su planificación integral está en las secciones 8–9 de
[backend-tasks.md](../tasks/backend-tasks.md) y
[frontend-tasks.md](../tasks/frontend-tasks.md). El RF no proporciona todas las
fórmulas, umbrales ni duraciones; esas definiciones deben ser entregadas por el
responsable antes de fijar valores en la implementación.

Por decisión del proyecto, el registro y recuperación usan **OTP por correo**,
sin fecha de expedición ni código dactilar. Se valida la estructura de la cédula
por módulo 10 y la posesión del correo por OTP; no se certifica identidad civil.
Los datos personales que no estén en la cuenta se declaran en EF-12. La fuente
`requerimiento.txt` conserva la redacción original y
[ADR-008](../decisions/ADR-008-registro-y-recuperacion-con-otp.md) registra la adaptación.

---

## 4. Actores y permisos

| Actor | Descripción | Accesos |
|---|---|---|
| **Postulante** | Persona natural que se registra y postula | EF-11, EF-12, consulta y descarga de su postulación |
| **Analista TH** | Analista de la Subdirección Nacional de Gestión de Talento Humano | Habilita fases, revisa postulaciones, genera matriz |
| **Administrador** | Comisión de Selección | Publica fases, habilita la evaluación técnica, administra el banco de preguntas |
| **Sistema** | Procesos automáticos | Validaciones, bloqueos, notificaciones, hash y sellado |

---

## 5. Historias de usuario

### HU-01 — Registro con cédula y OTP por correo

**Como** postulante **quiero** registrar mi cuenta validando mi cédula y el acceso
a mi correo **para** utilizar el portal sin depender de Registro Civil.

**Criterios de aceptación**

1. Solicitar cédula de diez dígitos y correo; validar módulo 10 y sintaxis de correo.
2. Enviar OTP de seis dígitos con vigencia actual de diez minutos mediante
   `POST /api/v1/auth/register/request`.
3. Completar nombres, apellidos, contraseña y OTP con
   `POST /api/v1/auth/register/verify`; crear la cuenta solo tras verificar el desafío.
4. El desafío debe vincular cédula, correo y finalidad de registro; consumo de uso
   único, bloqueo y reenvío se completan en T16. No solicitar fecha de expedición
   ni código dactilar, ni presentar OTP como certificación de Registro Civil.
5. Persistir `metodo_validacion = otp_correo` y rol inicial `postulante`.
6. Precargar en EF-12 los datos disponibles de la cuenta; los restantes, incluidos
   género, nacimiento, nacionalidad, estado civil y pareja, se declaran en el formulario.

> **RF de origen:** EF-11.4 y EF-12 1.1. **Adaptación aprobada:** ADR-008.
> El TTL del OTP es una decisión técnica; no debe confundirse con el bloqueo del RF.

### HU-02 — Bloqueo temporal tras intentos fallidos

**Como** postulante **quiero** que el sistema bloquee mi registro tras tres
intentos fallidos **para** proteger la identidad de los usuarios y descartar
intentos automatizados.

**Criterios de aceptación**

1. El postulante dispone de **3 oportunidades** de registro.
2. Al tercer intento fallido se muestra
   *"registro fallido, inténtelo nuevamente en 10 minutos"*.
3. Pasados **10 minutos** el sistema se refresca y permite reintentar.
4. El contador se reinicia tras un intento exitoso.
5. El estado del bloqueo se conserva **fuera del proceso de la aplicación**
   (Redis con TTL de 10 minutos) para que sobreviva a reinicios y escale
   horizontalmente.

> **RF**: EF-11, punto 4. **Rúbrica**: criterio 3 (Cache, Expiración & Rate Limiting).

### HU-03 — Inicio de sesión

**Como** postulante registrado **quiero** entrar con mi cédula y contraseña
**para** acceder a mi formulario.

**Criterios de aceptación**

1. El login usa **cédula** (usuario) y **contraseña**.
2. Con credenciales incorrectas se muestra
   *"sus credenciales de ingreso son incorrectas"*.
3. La contraseña nunca se almacena en claro: se guarda un hash **Argon2id**.
4. La sesión se transporta en una **cookie `HttpOnly`**; el token no es
   accesible desde JavaScript.
5. Al primer ingreso de un postulante registrado, el sistema lo dirige
   **obligatoriamente** al formulario de postulación.

> **RF**: EF-11, puntos 5 y 6.

### HU-04 — Recuperación de contraseña con OTP

**Como** postulante **quiero** recuperar mi contraseña mediante mi correo registrado
**para** volver a entrar sin depender del servicio de Registro Civil.

**Criterios de aceptación**

1. Ofrecer «¿Olvidaste tu contraseña?» desde login y solicitar la cédula.
2. `POST /api/v1/auth/password/forgot` envía OTP exclusivamente al correo de la
   cuenta registrada y responde un mensaje genérico sin revelar si la cuenta existe.
3. Introducir cédula, OTP y contraseña nueva en `POST /api/v1/auth/password/reset`.
4. Comprobar el desafío con propósito de recuperación, expiración y uso único,
   actualizar el hash, invalidar sesiones anteriores y permitir iniciar sesión de nuevo.
5. No solicitar fecha de expedición ni código dactilar, ni permitir cambiar el
   destinatario del OTP durante la recuperación.

> **RF de origen:** EF-11.6. **Adaptación aprobada:** ADR-008. Consolidación en T16/FT16.

### HU-05 — Formulario de postulación en 9 secciones

**Como** postulante **quiero** completar el formulario EF-12 por secciones
**para** revered mis datos y adjuntar los documentos habilitantes.

**Criterios de aceptación**

| # | Sección | Contenido mínimo |
|---|---|---|
| 1 | Información personal | Nombres, apellidos, cédula, género, fecha de nacimiento, nacionalidad, estado civil, datos de la pareja si casado/unión de hecho |
| 2 | Información domiciliaria y de contacto | País, provincia (solo Ecuador), cantón, teléfonos, 2 correos, dirección |
| 3 | Información adicional | Familiares en 2.º grado de consanguinidad / 4.º de afinidad que trabajen en IESS o BIESS |
| 4 | Selección del cargo | Sector (Afiliados / Jubilados / Empleadores) y documento de respaldo de la organización |
| 5 | Formación académica | Nivel, institución, tipo, título, área, registro SENESCYT, documento |
| 6 | Experiencia profesional | Tipo, cargo, institución, fechas, actividades, funciones (máx. 1000 caracteres), documento; **duración calculada en años, meses y días** |
| 7 | Publicaciones e investigaciones | Título, tipo, medio, fecha, relación con el campo, documento |
| 8 | Capacitación | Tipo (curso/seminario/taller), nombre, institución, fechas, horas, documento |
| 9 | Inhabilidades | 13 preguntas de Sí/No; si la 1.9.7 es "Sí" se piden institución, cargo, fecha de fin y motivo |

Además:

- Cada sección tiene **Guardar**, **Editar**, **Eliminar** y **Guardar
  información**.
- El botón **Guardar Postulación** muestra el texto de consentimiento y
  declaración jurada (5 puntos del RF, literal).
- El botón **Enviar Postulación** sella la postulación.

> **RF**: EF-12, puntos 1.1 a 1.10.

### HU-06 — Cascada geográfica y restricción de nacionalidad

**Como** postulante **quiero** que el formulario se adapte a mi situación
**para** no enviar datos que el concurso no admite.

**Criterios de aceptación**

1. La sección 1.2.1 ofrece la lista completa de países del RF.
2. Si el país es **Ecuador**, la provincia se elige de las 24 provincias y el
   cantón de la lista de la provincia.
3. Si el país es **otro**, provincia y cantón se ingresan manualmente.
4. Si la nacionalidad **no es ecuatoriana**, el sistema **no permite continuar**
   y cierra la postulación.

> **RF**: EF-12, puntos 1.1.6, 1.2.1, 1.2.2 y 1.2.3.

### HU-07 — Adjuntos PDF con límites por tipo

**Como** postulante **quiero** adjuntar mis documentos de respaldo **para**
acreditar mi información.

**Criterios de aceptación**

| Documento | Formato | Tamaño máximo |
|---|---|---|
| Respaldo de organización (sección 4) | PDF | 4 MB |
| Formación académica (5), experiencia (6), publicaciones (7), capacitación (8) | PDF | 2 MB |
| Total por postulante | PDF | 50 MB |

> **RF**: EF-1 (Restricciones) y EF-12, puntos 1.4.1.3.1.2, 1.5.2, 1.6.8, 1.7.1.6 y 1.8.7.

### HU-08 — Validación de elegibilidad por sector

**Como** postulante **quiero** que el sistema me informe si puedo postular por
mi sector **para** no perder tiempo en un proceso que se cerrará.

**Criterios de aceptación**

1. Al elegir **Afiliados**, el sistema valida la calidad de afiliado y muestra
   *"Postulante afiliado al IESS"* o *"Postulante No Afiliado al IESS"*.
2. Si no mantiene la calidad, se muestra el mensaje del RF y **el sistema se
   cierra automáticamente**.
3. El mismo criterio aplica a **Jubilados** (*"Postulante acredita jubilación"*)
   y **Empleadores** (*"Postulante es empleador"*).
4. Si el sector es válido, se pregunta si cuenta con respaldo de una
   organización legalmente constituida ante autoridad competente.

> **RF**: EF-12, puntos 1.4.1 a 1.4.3. En el parcial se **simula** el historial
> laboral.

### HU-09 — Guardado de progreso

**Como** postulante **quiero** guardar mi avance **para** no perder lo que
ingresé.

**Criterios de aceptación**

1. El botón *"Guardar Progreso"* existe en cada sección.
2. Al reingresar, el postulante ve lo que ya completó.
3. Una postulación en estado `draft` es **editable**; una enviada (`submitted`)
   **no**.

> **RF**: EF-12, notas al pie de la sección 1.10.

### HU-10 — Envío, sellado y notificación

**Como** postulante **quiero** enviar mi postulación **para** que sea
considerada y me llegue la confirmación.

**Criterios de aceptación**

1. *"Enviar Postulación"* sella la postulación: calcula el **hash SHA-256**
   del contenido canónico (claves ordenadas) y la fecha de envío.
2. El estado pasa a `submitted` y el contenido queda bloqueado.
3. El formulario completo se envía **automáticamente al correo** del
   postulante.
4. La postulación tiene un **código de numeración único**.

> **RF**: EF-12, puntos 1.10.1 a 1.10.3.

### HU-11 — Impresión y firma manuscrita

**Como** postulante **quiero** imprimir el formulario firmado **para**
entregarlo físicamente junto con los requisitos.

**Criterios de aceptación**

1. El formulario incluye un apartado de **firma manuscrita**, nombres,
   apellidos y número de cédula.
2. Existe una vista de impresión del formulario completo.

> **RF**: EF-12, punto 1.10.2 y literal del punto 5 de la declaración.

### HU-12 — Dashboard de administración

**Como** analista de Talento Humano **quiero** ver las postulaciones en tiempo
real **para** dimensionar el concurso.

**Criterios de aceptación**

1. El dashboard muestra el **número de postulantes por cargo**.
2. El administrador puede **habilitar** el apartado EF-12 (publicar la fase).
3. El historial permite abrir el detalle íntegro de una postulación cerrada.
4. La matriz comparativa se exporta a **.xlsx** y **PDF**.

> **RF**: EF-13.

---

## 6. Reglas de negocio transversales

| ID | Regla | Dónde se aplica |
|---|---|---|
| RN-01 | La cédula debe ser válida por el algoritmo módulo 10 | Backend (dominio) |
| RN-02 | La contraseña exige ≥ 8 caracteres, mayúscula, minúscula, número y símbolo | Backend (dominio) + frontend |
| RN-03 | Tres intentos fallidos bloquean la operación 10 minutos | Redis con TTL |
| RN-04 | El hash de la contraseña usa Argon2id con sal aleatoria | Backend |
| RN-05 | El hash de la postulación se calcula sobre el JSON **canónico** | Backend (dominio) |
| RN-06 | Una postulación enviada no se edita ni se elimina | Backend (dominio) |
| RN-07 | El tamaño máximo del cuerpo de una petición es 10 MiB; los PDF tienen límites propios por tipo | Backend + nginx |
| RN-08 | El token de sesión viaja en cookie `HttpOnly` y nunca en `localStorage` | Backend + frontend |
| RN-09 | Los catálogos geográficos se cachean para no consultar PostgreSQL en cada request | Redis |
| RN-10 | Los datos personales no se registran en logs | Backend (logger con redacción) |

---

## 7. Matriz de trazabilidad RF → HU → Implementación

Las rutas marcadas como *planificada* todavía no existen en el árbol: son el
destino del trabajo descrito en [`../tasks/`](../tasks/).

| RF | Punto | HU | Ubicación en el código | Estado |
|---|---|---|---|---|
| EF-11 adaptado | Registro y OTP | HU-01 | `apps/backend/internal/identity/` y `platform/cache/otp.go` | Base disponible; consolidación T16 (ADR-008) |
| EF-11 | 4 — 3 intentos y bloqueo | HU-02 | `apps/backend/internal/platform/cache/` | ⬜ Planificada |
| EF-11 | 5 — Login | HU-03 | `apps/backend/internal/identity/service.go` | ⬜ Planificada |
| EF-11 | 6 — Recuperación | HU-04 | `apps/backend/internal/identity/service.go` | ⬜ Planificada |
| EF-12 | 1.1 — Información personal | HU-05 | `apps/backend/internal/submission/sections.go` | ⬜ Planificada |
| EF-12 | 1.2 — Domiciliaria y contacto | HU-05 | `apps/backend/internal/submission/sections.go` | ⬜ Planificada |
| EF-12 | 1.3 — Información adicional | HU-05 | `apps/backend/internal/submission/sections.go` | ⬜ Planificada |
| EF-12 | 1.4 — Sector y elegibilidad | HU-08 | `apps/backend/internal/submission/rules.go` | ⬜ Planificada |
| EF-12 | 1.5 — Formación académica | HU-05 | `apps/backend/internal/submission/sections.go` | ⬜ Planificada |
| EF-12 | 1.6 — Experiencia y duración | HU-05 | `apps/backend/internal/submission/rules.go` | ⬜ Planificada |
| EF-12 | 1.7 — Publicaciones | HU-05 | `apps/backend/internal/submission/sections.go` | ⬜ Planificada |
| EF-12 | 1.8 — Capacitación | HU-05 | `apps/backend/internal/submission/sections.go` | ⬜ Planificada |
| EF-12 | 1.9 — Inhabilidades | HU-05 | `apps/backend/internal/submission/sections.go` | ⬜ Planificada |
| EF-12 | 1.2.1–1.2.3 — Cascada geográfica | HU-06 | `apps/backend/internal/catalog/` · `apps/frontend/src/data/ecuadorGeo.ts` | 🟡 Parcial (solo frontend) |
| EF-12 | 1.1.6 — Nacionalidad | HU-06 | `apps/backend/internal/submission/rules.go` | ⬜ Planificada |
| EF-12 | Adjuntos PDF y límites | HU-07 | `apps/backend/internal/document/` · `apps/backend/internal/platform/storage/local.go` | 🟡 Parcial (solo almacenamiento) |
| EF-12 | Guardar progreso | HU-09 | `apps/backend/internal/submission/service.go` | ✅ Implementado (estado `draft`) |
| EF-12 | 1.10 — Enviar y notificar | HU-10 | `apps/backend/internal/submission/submission.go` · `apps/backend/internal/platform/mail/mail.go` | 🟡 Parcial (hash sí; envío y correo no) |
| EF-12 | 1.10.2 — Firma e impresión | HU-11 | `apps/frontend/src/pages/imprimir-view.tsx` | ⬜ Planificada |
| EF-13 | Dashboard y matriz | HU-12 | `apps/backend/internal/review/` · `apps/frontend/src/pages/dashboard-view.tsx` | 🟡 Parcial (solo frontend, con datos mock) |

---

## 8. Requisitos no funcionales

| ID | Categoría | Requisito |
|---|---|---|
| RNF-01 | Rendimiento | La API responde en menos de 300 ms (p95) en listados paginados |
| RNF-02 | Rendimiento | El dashboard de 100 000 postulaciones no debe degradarse (paginación por cursor) |
| RNF-03 | Escalabilidad | El backend es stateless: la sesión vive en el almacén compartido |
| RNF-04 | Seguridad | Contraseñas con Argon2id; tokens de sesión opacos y aleatorios de 256 bits |
| RNF-05 | Seguridad | CORS con lista blanca explícita; en producción, mismo origen detrás de nginx |
| RNF-06 | Privacidad | Consentimiento explícito y verificación de veracidad (LOPDP, art. 7) |
| RNF-07 | Privacidad | Los documentos se almacenan fuera del directorio de la aplicación, con claves no adivinables |
| RNF-08 | Auditoría | Toda acción sensible queda registrada con actor, fecha y resultado |
| RNF-09 | Disponibilidad | Healthchecks `/healthz` y `/readyz` para orquestadores |
| RNF-10 | Calidad | CI con formato, lint y tests en frontend y backend; cobertura en el backend |

---

## 9. Riesgos y supuestos

| # | Riesgo | Mitigación |
|---|---|---|
| R-01 | No hay acceso a los sistemas del IESS (Registro Civil, Historial Laboral) | Registro Civil no se utiliza: módulo 10 + OTP por correo según ADR-008. Historial Laboral usa un puerto de elegibilidad y fuente simulada de servidor (T17); no confiar en flags del cliente |
| R-02 | La rúbrica menciona Fiber o Gin y el proyecto usa chi | ADR-001 con la justificación técnica; la arquitectura en capas es idéntica y es lo que se evalúa |
| R-03 | Volumen alto de postulaciones en el dashboard | Paginación por cursor (keyset) y caché Redis para los conteos |
| R-04 | Datos personales sensibles en el repositorio | `.gitignore` estricto, hashes de documentos, sin logs de contenido |
| R-05 | Plazo de 30 días de la Disposición Transitoria | Prioridad al flujo registro → formulario → envío; lo demás queda fuera del parcial |

---

## 10. Glosario

| Término | Significado |
|---|---|
| EF | Especificación Funcional del requerimiento del IESS |
| RF | Requerimiento Funcional (documento rector) |
| PRD | Product Requirements Document — requisitos de producto |
| TRD | Technical Requirements Document — requisitos técnicos |
| ADR | Architecture Decision Record — decisión de arquitectura registrada |
| Sector | Representación por la que postula: Afiliados, Jubilados o Empleadores |
| OTP | Código de un solo uso enviado por correo; el proyecto utiliza seis dígitos y TTL actual de diez minutos |
| Hash criptográfico | Huella SHA-256 del contenido canónico de una postulación |

---

**Ver también**: [`../trd/TRD.md`](../trd/TRD.md) ·
[`../api-contracts/API_CONTRACT.md`](../api-contracts/API_CONTRACT.md) ·
[`../decisions/`](../decisions/) · [`requerimiento.txt`](../requerimiento.txt) (fuente)
