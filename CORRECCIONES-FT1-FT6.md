# Correcciones pendientes del frontend: FT1–FT6

Este documento concentra las correcciones necesarias antes de considerar
cerradas las tareas FT1 a FT6. El alcance termina en revisión, envío e
impresión del formulario EF-12; FT7 en adelante queda fuera.

## Prioridad de corrección

1. Corregir los errores de compilación y lint.
2. Impedir que se guarden secciones inválidas.
3. Unificar el contrato de la sección 9.
4. Corregir el flujo de consentimientos y la impresión.
5. Completar los controles de OTP y las pruebas de componentes.

## FT1 — Cliente HTTP

- [x] Reemplazar el uso de `.then()` en `features/auth/sesion-context.tsx`
      por `async/await`, para cumplir la convención del cliente.
- [x] Agregar pruebas para peticiones `FormData`, especialmente antes de
      integrar documentos reales.
- [x] Probar el flujo completo de sesión expirada: respuesta `401`, limpieza
      de sesión y redirección al login desde una ruta protegida.
- [x] Revisar el manejo silencioso de errores de parseo y del callback global
      `401`, conservando siempre el error original y siguiendo el mecanismo de
      logging del proyecto.

## FT2 — Router y rutas protegidas

- [x] Agregar pruebas de acceso para las combinaciones postulante, analista y
      administrador.
- [x] Verificar que un postulante no pueda abrir `/dashboard` y que un analista
      no pueda abrir `/admin/*`.
- [x] Verificar la redirección posterior al login usando el parámetro
      `destino`.
- [x] Verificar el comportamiento cuando la sesión expira mientras se usa una
      ruta protegida.
- [x] Confirmar la regla de ruta inicial para postulantes que ya tienen una
      postulación enviada, si el backend distingue ese caso.

## FT3 — Registro, login y recuperación

- [x] Eliminar los contadores de intentos calculados localmente en
      `login-view.tsx`. Los intentos restantes solo deben mostrarse si los
      informa el servidor.
- [x] No crear un bloqueo local fijo de 600 segundos. El vencimiento debe
      provenir del servidor y persistir aunque se recargue la página.
- [x] Implementar el reenvío de OTP según el contrato de la API.
- [x] Mostrar estados de OTP inválido, expirado, usado o bloqueado según los
      datos recibidos del servidor.
- [x] Agregar pruebas de los componentes y flujos completos de registro,
      login y recuperación, no solo de validadores y clases de error.
- [x] Mantener la confirmación genérica de recuperación sin revelar si una
      cédula está registrada.

## FT4 — Estado remoto con TanStack Query

- [x] Mantener estados reales de carga, éxito, vacío y error en cada vista
      conectada a la API.
- [x] No marcar como completadas las vistas que todavía son placeholders.
- [x] Revisar que cada mutación invalide únicamente las consultas necesarias,
      evitando invalidaciones excesivamente amplias.
- [x] Verificar con pruebas que los errores de red se reintentan y los errores
      `4xx` no.
- [x] Verificar que las acciones importantes muestren feedback visual de
      éxito y error mediante `sonner`.

## FT5 — Formulario EF-12

### Bloqueantes

- [x] En `postulacion-view.tsx`, detener `guardarSeccionActual()` cuando
      `validarSeccionEF12()` devuelva errores. Actualmente se muestra el error
      pero se continúa guardando la sección inválida.
- [x] Unificar la estructura de la sección 9. La interfaz usa:
      `q7Institucion`, `q7Cargo`, `q7FechaFin` y `q7Motivo`, mientras el
      validador espera `q7Detalles.institucion`, `q7Detalles.cargo`,
      `q7Detalles.fechaFin` y `q7Detalles.motivo`.
- [x] Agregar una prueba que demuestre que una sección inválida no se envía a
      la API.

### Integración funcional

- [x] Integrar la precarga de nombres, apellidos, cédula y correo desde la
      sesión sin sobrescribir datos ya guardados.
- [x] Confirmar que la precarga no provoque ciclos de render ni cambios
      repetidos.
- [x] Verificar que la cascada país → provincia → cantón conserve el contrato
      de catálogos del servidor.
- [x] Confirmar que la nacionalidad no admitida bloquee el avance y el envío,
      sin permitir guardar datos contradictorios.
- [x] Mantener la estrategia pesimista de guardado de forma uniforme, con
      feedback claro cuando la API falla.

### Adjuntos

- [x] Conservar el objeto `File` seleccionado o enviarlo inmediatamente al
      cliente de documentos.
- [x] Implementar carga `multipart/form-data`, listado y eliminación mediante
      la API cuando corresponda.
- [x] Guardar la referencia devuelta por el servidor, no solo nombre, tamaño y
      fecha locales.
- [x] Mantener la validación previa de formato PDF y límites de 2 MB / 4 MB.

## FT6 — Revisión, envío e impresión

### Compilación y consistencia de tipos

- [x] Corregir los errores de TypeScript en `revision-postulacion-view.tsx` e
      `imprimir-view.tsx` causados por nombres de campos desactualizados:
      `nombreTitulo`, `pais`, `numeroRegistroSenescyt`,
      `institucionEmpresa`, `actualmenteTrabajando`,
      `funcionesPrincipales`, `editorialMedio`, `fecha`, `nombreEvento` e
      `institucionOrganizadora`.
- [x] Cambiar `alReintentar` por `onReintentar` al usar `EstadoError`.
- [x] Eliminar imports y variables sin utilizar en las vistas de revisión,
      impresión y postulación.
- [x] Lograr que `pnpm lint` y `pnpm build` terminen correctamente.

### Consentimientos

- [x] Definir claramente el propósito de “Guardar Postulación”. Si debe
      persistir los consentimientos, incluir los cinco controles y enviarlos a
      la API; si solo guarda un borrador, cambiar el texto para no sugerir
      aceptación legal.
- [x] Mantener los cinco puntos visibles en revisión y en el formulario final.
- [x] Deshabilitar el envío mientras falte cualquiera de los cinco puntos.
- [x] Persistir el consentimiento junto con el sellado y comprobar la respuesta
      actualizada del servidor.

### Sellado y bloqueo

- [x] Confirmar que después del sellado la caché se actualice con el nuevo
      estado, código, fecha y hash.
- [x] Mantener todas las secciones y acciones de edición bloqueadas cuando el
      estado ya no sea `draft`.
- [x] Mostrar el hash completo en revisión e impresión, sin truncarlo en el
      documento oficial.

### Impresión

- [x] No imprimir los cinco consentimientos como `[X]` automáticamente. Debe
      reflejarse el estado real de cada punto.
- [x] Mantener las nueve secciones, código único, nombres, apellidos, cédula,
      hash y espacio para firma manuscrita.
- [x] Conservar los estilos de impresión que ocultan la navegación y los
      controles de pantalla.
- [x] Verificar la impresión con una postulación sellada y otra en borrador.

### Notificación

- [x] No mostrar “Se ha notificado al correo registrado” sin confirmación de la
      API.
- [x] Consumir y presentar el estado real de notificación: enviada, pendiente,
      fallida u otro estado definido por el contrato.

## Verificación de cierre

- [x] `pnpm --dir apps/frontend test`
- [x] `pnpm --dir apps/frontend lint`
- [x] `pnpm --dir apps/frontend build`
- [x] Probar manualmente registro, login, recuperación, ruta protegida,
      formulario, revisión, sellado e impresión.
- [x] Confirmar que FT1–FT6 no dependan de datos mock para el flujo principal.

