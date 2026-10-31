# Corrección de seguridad de identidad y correo

**2026-10-31 — revisión de `INFORME-BACKEND-T1-T9.md`.** Se corrigen los riesgos
actuales de OTP/Redis/notificaciones sin contabilizar como terminadas las fases,
elegibilidad o instrumentación de módulos futuros. No se cambia el frontend.

## Activación necesaria

1. Aplicar la migración nueva `00011_mail_outbox.sql` mediante el procedimiento
   normal de migraciones. Agregarla al repositorio no la aplica a un despliegue.
2. Configurar Redis para identidad y un servidor SMTP para entregar OTP.
3. Generar **fuera del repositorio** una clave aleatoria estable de 32 bytes y
   asignar su representación base64 a `MAIL_OUTBOX_KEY`:

   ```powershell
   python -c "import secrets,base64; print(base64.b64encode(secrets.token_bytes(32)).decode())"
   ```

   No copiar el resultado en documentación, commits o logs. Inyectarlo en el
   entorno de la API; Compose de desarrollo usa el entorno de la terminal o
   `.env` de la raíz, producción usa `deploy/.env`. Los ejemplos dejan el valor
   vacío. Respaldar la clave junto con la base, bajo controles independientes.
4. No cambiar la clave con trabajos pendientes sin un procedimiento de migración
   de claves: la carga cifrada no puede recuperarse si se pierde su clave.

Sin Redis, login/OTP responden **503 `seguridad_no_disponible`**, no continúan
sin bloqueo. Las sesiones ya persistidas se pueden consultar/revocar en
PostgreSQL y los catálogos conservan su fallback a PostgreSQL. Una caída inicial
de Redis exige reiniciar la API después de restablecerlo; si el cliente ya estaba
conectado, sus reconexiones siguen la política de go-redis.

Sin clave válida no se aceptan nuevos OTP ni sellos que requieran registrar la
confirmación. Sin SMTP no se emiten desafíos OTP nuevos; una confirmación ya
encolada conserva `pending` para una futura configuración. El worker retira
cargas vencidas incluso si no tiene SMTP o una clave de descifrado utilizable.

## Contrato OTP compatible

Las rutas existentes no cambian:

| Operación | Ruta bajo `/api/v1/auth` |
|---|---|
| Solicitar registro | `POST /register/request` |
| Verificar registro | `POST /register/verify` |
| Solicitar recuperación | `POST /password/forgot` |
| Confirmar recuperación | `POST /password/reset` |

Las solicitudes devuelven, además del mensaje, `desafio_id` y `expira_en`.
Las verificaciones admiten `desafio_id` adicional. Durante la transición, un
cliente anterior puede omitirlo: el servidor resuelve su desafío activo y
**sigue comprobando finalidad, cédula y correo vinculado**. No se reutiliza la
antigua clave OTP sin finalidad ni se acepta el correo de recuperación del cliente.
FT16 debe conservar explícitamente el ID de desafío entre pasos.

- Finalidades separadas: `registro` y `recuperacion`.
- Expiración de 10 minutos; tres intentos por desafío.
- Espera mínima de 60 s para reenvío; máximo tres emisiones en una ventana de
  10 minutos por finalidad/cédula. Reenviar no elude el bloqueo del desafío.
- Cambiar el correo de registro exige un desafío nuevo. Verificar con otro
  correo no crea una cuenta ni consume correctamente el código anterior.
- El código se guarda como hash ligado al ID aleatorio y al vínculo de identidad;
  las claves Redis llevan un hash del identificador, no cédula/correo legibles.
- Comparación, consumo, intentos y preservación del TTL ocurren en scripts Lua.
  Una lectura previa sólo calcula el hash candidato: la decisión/consumo vuelve
  a comparar el ID dentro del script. Dos instancias no pueden consumir el mismo
  desafío correctamente. No se usa `GETDEL` incondicional que quemaría un código
  válido al recibir un intento incorrecto.
- Contador y activación del bloqueo de login también son atómicos. `INCR` fija
  TTL sólo al crear la ventana; no usa un pipeline ni renueva el TTL en cada fallo.

| HTTP | Código | Significado |
|---|---|---|
| 401 | `otp_invalido` | Código, vínculo o desafío inválido/consumido/vencido |
| 429 | `reenvio_limitado` | Cooldown/cuota; `Retry-After: 60` como guía conservadora |
| 429 | `intentos_excedidos` | Cuenta/desafío bloqueado; `Retry-After: 600` |
| 503 | `seguridad_no_disponible` | No se pudo aplicar el control Redis; no hay acceso degradado |
| 503 | `notificacion_no_disponible` | Registro no pudo aceptar durablemente su correo |

La respuesta de recuperación es deliberadamente genérica: no confirma existencia
de cuenta ni entrega. Una cuenta ausente recibe un desafío y un trabajo suprimido,
sin destinatario y que nunca se envía por SMTP. Las incidencias concretas de
consulta/aceptación de recuperación se registran sin datos personales y mantienen
esa respuesta genérica; los límites y caídas globales de seguridad se evalúan
antes de consultar la cuenta. Esto homogeneiza el contrato, no promete ausencia
absoluta de canales laterales de tiempo/red.

La contraseña débil se rechaza **antes** de consumir el OTP. El cambio de clave y
la revocación de sesiones se confirman juntos. La emisión de una sesión vuelve a
comprobar, bajo el mismo bloqueo de cuenta, que el hash validado no fue sustituido
por un reset concurrente. Los errores de contador, revocación y persistencia ya
no se descartan silenciosamente.

## Correo durable, no entrega ficticia

`mail_outbox` guarda sólo metadatos y una carga cifrada AES-GCM. Destinatario,
OTP/formulario y cuerpo no se persisten en texto claro. La identidad del trabajo
se autentica como AAD; mover o alterar una carga impide descifrarla.

- OTP: se confirma la cola antes de responder la aceptación. Un fallo invalida
  únicamente ese desafío, sin borrar uno posterior. El worker comprueba que aún
  esté activo antes de enviarlo; desafíos consumidos/reemplazados se suprimen.
- EF-12: en la composición real, `QueuedConfirmation.Enqueue` se ejecuta con el
  contexto de la transacción del sello. No llama SMTP dentro de la transacción.
  Si no puede persistir la cola, se revierte el sello y responde 503. La entrega
  física ocurre exclusivamente después del commit.
- Cada worker reclama como máximo 10 trabajos con `SKIP LOCKED` y un lease de
  cinco minutos. Un worker antiguo no puede confirmar el lease de otro.
- Hasta cinco intentos, retroceso exponencial y vencimiento. OTP no supera el
  vencimiento de su desafío; confirmaciones tienen una ventana técnica inicial de
  siete días. No es un plazo legal ni una política institucional de retención.
- Al terminar/expirar se elimina la carga cifrada y se conservan sólo metadatos.
  Los errores guardados son códigos cerrados (`smtp_error`, etc.), no errores del
  proveedor que puedan incluir cuentas, credenciales o contenido.
- Un proceso que cae tras aceptación SMTP pero antes de confirmar el estado puede
  reenviar el mensaje: semántica **al menos una vez**, no exactamente una vez.

El EF-12 puede devolver `notificacion` como metadato, separado de `content`/hash:
`estado`, `intentos`, `codigoError`, `actualizadoEn`. Estados:
`pending`, `processing`, `smtp_accepted`, `failed`, `expired`, `superseded`.
Para sellos anteriores sin registro o una consulta de estado fallida se usa
`unavailable`; nunca se inventa una entrega previa.
`smtp_accepted` acredita aceptación del relay, no lectura ni llegada al buzón.
La vista de revisión debe conservar esa diferencia al integrar FT6/FT22.

El notificador síncrono previo se conserva como adaptador de compatibilidad para
pruebas/composiciones internas; `cmd/api` utiliza siempre la cola transaccional.
T23 todavía debe integrar las demás etapas y su administración/reintento por rol.

## Verificación y límites

- Tests locales de scripts reales mediante miniredis: finalidades/correo,
  consumo concurrente único, reenvío, bloqueo, expiración y fallo cerrado.
- Tests HTTP/servicio: cambio de correo, cruce de códigos, revocación, errores y
  rollback de sello si falla la cola. Tests de cifrado/integridad de la carga.
- Integración optativa: `AUTHORIZATION_TEST_DATABASE_URL`,
  `MAIL_TEST_DATABASE_URL`, `REDIS_TEST_URL`, `STORAGE_TEST_ENDPOINT` y credenciales
  `STORAGE_TEST_ACCESS_KEY`/`STORAGE_TEST_SECRET_KEY`, además de las variables PG
  ya existentes. Usar exclusivamente bases/buckets de pruebas.
- `.github/workflows/backend-integration.yml` configura PostgreSQL 18, Redis 7,
  RustFS y SMTP local, ejecuta tests con `-race` y conserva resultados. Tener ese
  workflow escrito no prueba que ya pasó en GitHub.
- No se implementan T17/T18/T20–T27 ni se aprueba retención/columnas por esta
  corrección. La prueba operativa de recuperación de objetos y los casos de
  almacenamiento con commit ambiguo conservan su seguimiento en T4/T12.
