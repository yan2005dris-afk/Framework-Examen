# Examen Práctico del Primer Parcial: Sistema de Registro y Postulación al Directorio del BIESS
**Asignatura:** Frameworks de Programación Web

**Ponderación:** 40 Puntos (Equivalente al 100% de la evaluación práctica del parcial)

**Modalidad:** Proyecto práctico en grupo

## 1. Descripción del Proyecto
El presente examen práctico consiste en el diseño, arquitectura e implementación del módulo inicial de registro y postulación para el Concurso Público de Méritos y Oposición del Directorio del BIESS.

La solución debe construirse bajo una arquitectura desacoplada moderna:

- Frontend: Single Page Application (SPA) reactiva y modular.
- Backend: Microservicio transaccional de alto rendimiento en Go con separación estricta de responsabilidades (Inyección de Dependencias), persistencia relacional en PostgreSQL y optimización en memoria con Redis.

## 2. Alcance Técnico y Requerimientos a Evaluar
### Frontend (SPA - React / Vite + Tailwind CSS)
Estructura y Estilos: Configuración limpia de proyecto con Vite y diseño utilitario responsivo con Tailwind CSS.
- Componentes y Estado: Separación modular de componentes y gestión de estados locales/globales para feedback visual inmediato (mensajes de error, validaciones de formularios y loaders).
- Módulo de Autenticación y Registro (EF-11): Formulario de acceso con simulación de validación dactilar y fecha de expedición, además de feedback ante bloqueo de cuenta.
- Módulo de Postulación (EF-12): Formulario dinámico con cascada de selección geográfica (País → Provincia → Cantón) y restricción estricta de nacionalidad ecuatoriana.
- Consumo Asíncrono: Consumo de APIs REST mediante promesas (async/await, Fetch o Axios) y manejo de rutas protegidas.
### Backend (Go - Fiber/Gin + PostgreSQL + Redis)
- Arquitectura: Organización en capas limpias mediante Inyección de Dependencias (Handler/Controller → Service/Use Case → Repository).
- Persistencia Relacional (PostgreSQL): Modelado de datos para postulantes, postulaciones y catálogos geográficos, aplicando transacciones de base de datos (BEGIN, COMMIT, ROLLBACK) para garantizar atomicidad.
- Caché y Seguridad (Redis):
  - Control de intentos de autenticación: bloqueo temporal de 10 minutos al tercer intento fallido consecutivo.
  - Caching de catálogos de consulta frecuente (países, provincias, cantones).
- Validación de Reglas de Negocio: Validación en servidor de los requisitos de postulación del documento de especificación funcional.
## 3. Rúbrica de Evaluación Analítica (Base: 40 Puntos)
| Criterio / Dimensión | Excelente | Bueno | Regular | Insuficiente | Puntaje |
| --- | --- | --- | --- | --- | --- |
| 1. Arquitectura Backend & Inyección de Dependencias (Go - Fiber/Gin) | Arquitectura en capas perfectamente desacoplada (handlers, services, repositories) mediante inyección de dependencias. Manejo robusto de errores nativos y concurrencia. (8.0 pts) | Capas separadas pero con dependencias acopladas directamente o manejo de errores incompleto en rutas secundarias. (6.0 pts) | Lógica de negocio mezclada en los controladores/handlers, sin inyección de dependencias clara. (4.0 pts) | Código monolítico, sin estructura por capas o que no compila/ejecuta. (0.0 pts) | / 8 pts |
| 2. Persistencia Relacional & Transacciones (PostgreSQL) | Modelado normalizado y uso riguroso de transacciones de base de datos para inserciones multi-tabla en el guardado de la postulación. (8.0 pts) | Persistencia funcional pero sin manejo transaccional atómico ante fallos en inserciones múltiples. (6.0 pts) | Modelo relacional incompleto, datos huérfanos o inconsistencias al persistir información. (4.0 pts) | Base de datos no relacional, datos en memoria volátil o fallas críticas de persistencia. (0.0 pts) | / 8 pts |
| 3. Cache, Expiración & Rate Limiting (Redis) | Implementa con éxito el bloqueo temporal de 10 min tras 3 intentos fallidos (EF-11) y el cacheo de catálogos geográficos con claves y TTL óptimos. (6.0 pts) | Redis implementado de forma parcial (solo bloqueo de intentos o solo caché de lectura), funcionando correctamente. (4.5 pts) | Conexión a Redis configurada, pero con errores en el manejo de llaves, expiración (TTL) o lógica de control. (3.0 pts) | No integra Redis; maneja estados temporales en memoria volátil de la aplicación. (0.0 pts) | / 6 pts |
| 4. Frontend SPA: Reactividad, Componentes & Estado | SPA modular construida en Vite + Tailwind CSS; gestión de estado sólida, interfaz responsiva y feedback visual inmediato (validaciones, loaders). (6.0 pts) | Componentes funcionales y reactivos con Tailwind CSS, pero con renderizados innecesarios o acoplamiento de estado. (4.5 pts) | Interfaz poco estructurada, uso deficiente del framework de estilos o estado desincronizado. (3.0 pts) | Interfaz estática (HTML clásico), sin empaquetador moderno ni reactividad. (0.0 pts) |	/ 6 pts |
| 5. Consumo de API & Manejo Asíncrono | Enrutamiento dinámico protegido por sesión; consumo asíncrono óptimo (async/await) con manejo exhaustivo de estados (loading, success, error). (6.0 pts) | Consumo de endpoints y enrutamiento funcional, con manejo básico o parcial de excepciones de red. (4.5 pts) | Fallos ocasionales en llamadas asíncronas, bloqueo de UI o rutas no protegidas adecuadamente. (3.0 pts) | Sin integración cliente-servidor o consumo asíncrono roto. (0.0 pts) | / 6 pts |
| 6. Cumplimiento de Reglas del Negocio (RF IESS) | Cumple fielmente EF-11 y EF-12: bloqueo de intentos, validación dactilar simulada, restricción de nacionalidad ecuatoriana y cascada país/provincia/cantón. (6.0 pts) | Cumple con la mayor parte del flujo de registro y postulación, omitiendo como máximo una validación menor. (4.5 pts) | Flujo incompleto; omite validaciones clave como la restricción de nacionalidad o combos anidados. (3.0 pts) | No implementa las especificaciones funcionales solicitadas en el documento base. (0.0 pts) | / 6 pts |
| Total | | |	| | / 40 pts |
## 4. Instrucciones y Formato de Entrega (Para el Aula Virtual)
- Formato del Entregable:
  1. Enlace al repositorio de Git (GitHub / GitLab) con historial de commits y ramas ordenadas.
  2. Archivo README.md detallando:
      - Prerrequisitos de instalación (versión de Go, Node.js, PostgreSQL y Redis).
      - Instrucciones de compilación y ejecución local (o archivo docker-compose.yml para levantar la base de datos y Redis).
      - Variables de entorno de ejemplo (.env.example).
  3. Archivo .sql con el script DDL de creación de tablas y datos semilla de catálogos (división política).
- Demostración: Cada estudiante/equipo deberá realizar una sustentación funcional de 10 minutos explicando la arquitectura de software implementada y respondiendo preguntas técnicas sobre el código.
