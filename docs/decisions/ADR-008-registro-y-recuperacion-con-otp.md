# ADR-008 — Registro y recuperación mediante OTP por correo

| Campo | Valor |
|---|---|
| Estado | Aceptada |
| Fecha | 2026-10-31 |
| Ámbito | Identidad, interfaz de autenticación y documentación |
| Fuente | Decisión del responsable del proyecto; adaptación de EF-11 del RF v1.4 |

## Contexto

El RF original exige validar la fecha de expedición o el código dactilar mediante
Registro Civil. El proyecto no dispone de acceso a ese servicio. Se adopta OTP por
correo como mecanismo de registro y recuperación. `docs/requerimiento.txt` conserva
el texto original; el PRD, el contrato y las tareas describen esta adaptación.

## Decisión

- Validar la cédula mediante módulo 10 y comprobar el control del correo mediante
  un OTP numérico de seis dígitos. Su vigencia actual es de diez minutos, una
  decisión técnica del proyecto distinta del bloqueo de diez minutos del RF.
- Registro: solicitar cédula y correo, enviar OTP y verificarlo al completar los
  datos de cuenta y la contraseña. Persistir `metodo_validacion = otp_correo`.
- Recuperación: solicitar cédula y enviar el OTP exclusivamente al correo ya
  registrado. Comprobarlo antes de actualizar la contraseña e invalidar sesiones.
- No solicitar ni simular la validación de fecha de expedición o código dactilar.
- Precargar en EF-12 únicamente los datos de la cuenta disponibles. Los demás
  datos personales y de pareja se declaran en el formulario; no se presentan como
  certificados ni autocompletados por Registro Civil.
- El registro público solo puede crear el rol `postulante`.

## Consecuencias y pendientes

El OTP acredita acceso al correo, y módulo 10 comprueba la estructura de la cédula;
estos mecanismos no certifican la identidad civil, nacionalidad ni estado civil.
La integración con Registro Civil queda fuera del alcance actual, sin bloquear
la entrega del flujo OTP. La elegibilidad laboral por sector es una capacidad
separada; no se deriva del OTP.

La base OTP existe en `internal/identity/` y `internal/platform/cache/otp.go`.
La tarea T16 debe completar la vinculación del desafío a cédula, correo y finalidad,
el consumo atómico, la separación entre registro y recuperación y los controles de
reenvío. FT3 y FT16 cubren la interfaz. Esta decisión no afirma que esos pendientes
ya estén implementados.
