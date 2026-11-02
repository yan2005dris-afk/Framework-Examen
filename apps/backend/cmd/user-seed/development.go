package main

/**
 * developmentPhases son las fases de la convocatoria de desarrollo. Copian el
 * cronograma del RF para que el portal pueda probarse de punta a punta.
 */
const developmentPhases = `[
  {"codigo": "postulacion", "titulo": "Postulación y Recepción de Documentos", "apertura": "2026-01-05T05:00:00Z", "cierre": "2027-01-01T04:59:59.999999Z", "publicada": true, "habilitada": true},
  {"codigo": "meritos", "titulo": "Revisión y Calificación de Méritos", "apertura": "2026-01-05T05:00:00Z", "cierre": "2027-01-01T04:59:59.999999Z", "publicada": true, "habilitada": true},
  {"codigo": "oposicion_psicotecnica", "titulo": "Pruebas Psicotécnicas", "apertura": "2026-01-05T05:00:00Z", "cierre": "2027-01-01T04:59:59.999999Z", "publicada": true, "habilitada": true},
  {"codigo": "oposicion_teorica", "titulo": "Pruebas Teóricas y de Conocimiento", "apertura": "2026-01-05T05:00:00Z", "cierre": "2027-01-01T04:59:59.999999Z", "publicada": true, "habilitada": true},
  {"codigo": "reconsideracion", "titulo": "Impugnaciones y Reconsideraciones", "apertura": "2026-01-05T05:00:00Z", "cierre": "2027-01-01T04:59:59.999999Z", "publicada": true, "habilitada": true},
  {"codigo": "resultados", "titulo": "Publicación de Resultados Finales", "apertura": "2026-01-05T05:00:00Z", "cierre": "2027-01-01T04:59:59.999999Z", "publicada": true, "habilitada": true}
]`

/**
 * developmentPolicy es la política de privacidad del entorno de desarrollo.
 * El texto es el del RF: la publicación de nombres exige consentimiento
 * explícito y los documentos, resultados y notificaciones se conservan durante
 * los períodos definidos.
 */
const developmentPolicy = `{
  "texto": "Los datos de este expediente se usan exclusivamente para el concurso de méritos y oposición. La publicación de nombres requiere consentimiento explícito. Los documentos, resultados y notificaciones se conservan durante los períodos definidos en esta política.",
  "diasAcceso": 365,
  "diasCorreo": 180,
  "diasReportes": 365,
  "permiteCorreo": true,
  "diasDocumentos": 365,
  "permiteExportacion": true,
  "referenciaAprobacion": "politica-biess-2026-v1",
  "disposicionHistoricos": "revision_humana_sin_borrado_automatico",
  "diasPublicacionNombres": 365,
  "informacionTratamiento": "Concurso BIESS 2026; auditoría al menos 730 días; sesiones expiradas 30 días.",
  "permiteNombresPublicos": true,
  "diasRevisionConservacion": 365
}`

/**
 * developmentPolicyHash es el sello de la política de desarrollo. Cambia
 * junto con developmentPolicy: si se edita el texto, hay que recalcularlo.
 */
const developmentPolicyHash = "dc8e190c515bbb5bbd8a1593068c17a928bbba17cdc5d74ce17380c2ea70a076"
