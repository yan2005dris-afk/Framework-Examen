export interface SeccionEstadoInfo {
  numero: number
  titulo: string
  tituloCorto: string
}

export const SECCIONES_EF12_INFO: SeccionEstadoInfo[] = [
  { numero: 1, titulo: "Información personal", tituloCorto: "Personal" },
  {
    numero: 2,
    titulo: "Información domiciliaria y contacto",
    tituloCorto: "Contacto",
  },
  { numero: 3, titulo: "Información adicional", tituloCorto: "Adicional" },
  {
    numero: 4,
    titulo: "Selección del cargo al que postula",
    tituloCorto: "Cargo",
  },
  { numero: 5, titulo: "Formación académica", tituloCorto: "Formación" },
  { numero: 6, titulo: "Experiencia profesional", tituloCorto: "Experiencia" },
  {
    numero: 7,
    titulo: "Publicaciones e investigaciones",
    tituloCorto: "Publicaciones",
  },
  { numero: 8, titulo: "Capacitación", tituloCorto: "Capacitación" },
  { numero: 9, titulo: "Inhabilidades", tituloCorto: "Inhabilidades" },
]

export const PREGUNTAS_INHABILIDADES = [
  {
    id: "q1",
    numero: "1.9.1",
    texto:
      "¿Es Ecuatoriano/a y está en pleno goce de los derechos de participación política?",
    requiereSi: true,
  },
  {
    id: "q2",
    numero: "1.9.2",
    texto: "¿Se encuentra inhabilitado para ejercer el comercio?",
  },
  {
    id: "q3",
    numero: "1.9.3",
    texto:
      "¿Está en mora, directa o indirectamente, en el pago de sus obligaciones en cualquiera de las instituciones del Estado o de las instituciones bajo control de la Superintendencia de Bancos, incluidas las off-shore?",
  },
  {
    id: "q4",
    numero: "1.9.4",
    texto:
      "¿Mantiene algún vínculo, por propiedad o administración, con las instituciones del sistema financiero y del sistema de seguros privados, con las bolsas de valores del país, Casas de Valores y Administradoras de Fondos y Fideicomisos, de acuerdo con las normas que la Junta de Política y Regulación Monetaria y Financiera emita para el efecto?",
  },
  {
    id: "q5",
    numero: "1.9.5",
    texto: "¿Es funcionario o empleado del IESS - BIESS?",
  },
  {
    id: "q6",
    numero: "1.9.6",
    texto:
      "¿Mantiene interés propio o representa a terceros en la propiedad, la dirección o la gestión de las compañías aseguradoras u otras personas que integran el sistema nacional de seguridad social?",
  },
  {
    id: "q7",
    numero: "1.9.7",
    texto:
      "¿En los últimos 5 años, ha sido removido por algún Organismo de Control?",
    tieneDetalle: true,
  },
  {
    id: "q8",
    numero: "1.9.8",
    texto: "¿Mantiene obligaciones pendientes con el SRI?",
  },
  {
    id: "q9",
    numero: "1.9.9",
    texto:
      "¿Ha incurrido en los últimos 5 años en castigo de sus obligaciones por parte de cualquier institución financiera?",
  },
  {
    id: "q10",
    numero: "1.9.10",
    texto: "¿Actualmente se encuentra litigando en contra del IESS- BIESS?",
  },
  {
    id: "q11",
    numero: "1.9.11",
    texto:
      "¿Se encuentra procesado, enjuiciado o condenado por algún delito relacionado con corrupción o crimen organizado?",
  },
  {
    id: "q12",
    numero: "1.9.12",
    texto:
      "¿Mantiene en firme algún tipo de responsabilidades por la Contraloría General del Estado?",
  },
  {
    id: "q13",
    numero: "1.9.13",
    texto:
      "¿Se encuentra registrado en la base de datos de la Unidad de Análisis Financiero y Económico?",
  },
] as const
