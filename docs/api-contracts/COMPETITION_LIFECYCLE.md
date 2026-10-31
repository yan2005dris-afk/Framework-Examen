# Contrato compartido: backend T15–T18 y frontend FT14–FT17

Esta es la integración técnica en desarrollo. No usar fechas, resoluciones,
convocatorias o fases de ejemplo como información oficial cuando falle la API.
Sin respuesta del servidor, mostrar indisponibilidad y deshabilitar acciones.
El registro OTP no certifica identidad civil ni elegibilidad IESS.

## T18: convocatorias y calendario

- `GET /api/v1/competitions?limit=20&cursor=…`: público, página por cursor; solo
  convocatorias aprobadas y publicadas. Una página vacía es un estado válido.
- `GET /api/v1/competitions/{id}`: detalle público; solo fases publicadas. Devuelve
  `id`, `codigo`, `titulo`, `apertura`, `cierre`, `sectores`, `fases`,
  `ef12Disponible`, `zonaHoraria: "America/Guayaquil"` y `limites: "[apertura,cierre)"`.
- `GET /api/v1/admin/competitions`: conserva el listado de revisión T9 (analista o
  administrador); no confundirlo con el endpoint público ni con el detalle de gestión.
- `POST /api/v1/admin/competitions`: solo administrador. Crea un período **deshabilitado**.
- `GET /api/v1/admin/competitions/{id}`: administrador, configuración y versión.
- `PUT /api/v1/admin/competitions/{id}/configuration`: administrador; reemplazo de
  configuración con `version` obligatoria (control optimista; 409 si cambió).

Creación (fechas ficticias, deben ser definidas por el responsable):

```json
{
  "codigo": "CONCURSO-PRUEBA",
  "titulo": "Período de prueba",
  "apertura": "2027-01-01T00:00:00-05:00",
  "cierre": "2027-02-01T00:00:00-05:00",
  "sectores": ["afiliados", "jubilados", "empleadores"],
  "fases": []
}
```

Configuración para habilitar el EF-12 (tras la aprobación institucional correspondiente):

```json
{
  "version": 1,
  "inicioAprobado": true,
  "publicada": true,
  "ef12Habilitado": true,
  "sectores": ["afiliados"],
  "fases": [{
    "codigo": "postulacion",
    "titulo": "Recepción de postulaciones",
    "apertura": "2027-01-01T00:00:00-05:00",
    "cierre": "2027-02-01T00:00:00-05:00",
    "publicada": true,
    "habilitada": true
  }]
}
```

Códigos de fase disponibles: `postulacion`, `meritos`, `oposicion_teorica`,
`oposicion_psicotecnica`, `reconsideracion`, `resultados`. Son capacidades, **no un
cronograma oficial precargado**. La fase publicada debe estar dentro del período.
Estas fases no calculan resultados ni habilitan por sí solas un examen.

La creación de un EF-12 exige `convocatoriaId`. Altas, guardados, bajas de borrador,
adjuntos y envíos comprueban aprobación, habilitación, sector y ventana en PostgreSQL,
incluidas las pestañas que quedaron abiertas. Apertura inclusiva, cierre exclusivo:
propuesta técnica pendiente de validación institucional. El legado sin convocatoria
se conserva para lectura; no se reasigna ni se habilita automáticamente.

## T17: verificación y cierres

El servidor ignora `verificacionIessCumplida` y `detalleVerificacionIess` del cliente.
La respuesta de la postulación añade `elegibilidad` con `id`, `cuentaId`,
`convocatoriaId`, `sector`, `elegible`, `origen`, `verificadoEn`, `codigo`, `mensaje`.
El adaptador local requiere `ELIGIBILITY_SIMULATION_FILE` con resultados explícitos
por UUID de cuenta y sector. Un resultado ausente es indisponibilidad, no inelegibilidad.
El origen `simulacion_controlada_servidor` **no es una consulta real al IESS**.

Un resultado negativo confirma `status: "closed"`, conserva evidencia y devuelve
409 `postulacion_cerrada_elegibilidad`. Recargar la postulación permite consultar el
cierre persistido. No se reabre ni elimina automáticamente; la revisión administrativa
queda para una política institucional posterior. 503 `elegibilidad_no_disponible`
permite reintentar y no cierra el borrador. Nacionalidad extranjera produce cierre
declarativo con su causa, sin afirmar verificación civil. Los respaldos declarados
siguen requiriendo nombre y PDF real al completar/enviar.

## T15/T16: correo y OTP

La cola confirma persistencia, no entrega en buzón. `pending`, `processing`,
`smtp_accepted`, `failed`, `expired` y `superseded` son estados operativos.
El `Message-ID` se conserva entre reintentos. SMTP es al menos una vez: un corte
después de aceptar DATA y antes de confirmar la cola puede producir duplicados.

OTP conserva finalidad/cuenta/correo/desafío y consumo único atómico. Configuración:
`OTP_RESEND_DELAY` (por defecto `1m`, mínimo `1s`, menor a `10m`) y `OTP_MAX_SENDS`
(por defecto 3, máximo 10 por diez minutos). No cambia tres fallos ni bloqueo de
diez minutos. Los rechazos 429 informan `Retry-After` y, cuando hay TTL efectivo,
`X-Retry-At`; no reiniciar contadores desde el navegador. Reenvío y bloqueo no son
estados exitosos de correo.
