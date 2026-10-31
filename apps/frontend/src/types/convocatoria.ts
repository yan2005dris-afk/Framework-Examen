/**
 * Tipos de dominio para convocatorias, fases y cronograma oficial (FT14 y FT15).
 * Conforme a EF-1, EF-11, EF-13 y Resolución No. C.D. 701.
 */

export type EstadoConvocatoria =
  "borrador" | "publicada" | "en_curso" | "cerrada" | "archivada"

export type EstadoFase = "pendiente" | "en_curso" | "finalizada" | "cancelada"

export type TipoFase =
  | "convocatoria_postulacion"
  | "verificacion_requisitos"
  | "calificacion_meritos"
  | "oposicion_examen"
  | "impugnacion_ciudadana"
  | "designacion_final"

export interface FaseConcurso {
  id: string
  convocatoriaId: string
  tipo: TipoFase
  nombre: string
  descripcion: string
  fechaInicio: string
  fechaFin: string
  estado: EstadoFase
  permitePostulacion: boolean
  orden: number
}

export interface ConvocatoriaDetalle {
  id: string
  codigo: string
  titulo: string
  descripcion: string
  resolucion: string
  periodo: string
  apertura: string
  cierre: string
  estado: EstadoConvocatoria
  cerrada: boolean
  postulacionHabilitada: boolean
  fases: FaseConcurso[]
  faseActual?: TipoFase
  totalPostulantes?: number
  createdAt: string
}

export interface CrearConvocatoriaPayload {
  codigo: string
  titulo: string
  descripcion?: string
  resolucion?: string
  periodo?: string
  apertura: string
  cierre: string
}

export interface ActualizarConvocatoriaPayload {
  titulo?: string
  descripcion?: string
  resolucion?: string
  apertura?: string
  cierre?: string
  estado?: EstadoConvocatoria
  cerrada?: boolean
  postulacionHabilitada?: boolean
  motivo?: string
}

export interface ActualizarFasePayload {
  nombre?: string
  descripcion?: string
  fechaInicio?: string
  fechaFin?: string
  estado?: EstadoFase
  permitePostulacion?: boolean
  motivo?: string
}

/**
 * Plantilla oficial de las 6 fases del Concurso de Méritos y Oposición
 * para el Directorio del BIESS (normativa EF-1, EF-11, EF-13).
 */
export const FASES_OFICIALES_DEFECTO: Omit<
  FaseConcurso,
  "id" | "convocatoriaId"
>[] = [
  {
    tipo: "convocatoria_postulacion",
    nombre: "Fase 1: Convocatoria pública y postulación",
    descripcion:
      "Publicación oficial del concurso, registro de postulantes y recepción de formularios EF-12 con expedientes digitales.",
    fechaInicio: "2026-10-15T00:00:00-05:00",
    fechaFin: "2026-11-15T23:59:59-05:00",
    estado: "en_curso",
    permitePostulacion: true,
    orden: 1,
  },
  {
    tipo: "verificacion_requisitos",
    nombre: "Fase 2: Verificación de requisitos mínimos e inhabilidades",
    descripcion:
      "Revisión de expedientes digitales, validación de certificados y comprobación de no inhabilidades legales para el cargo.",
    fechaInicio: "2026-11-16T00:00:00-05:00",
    fechaFin: "2026-11-25T23:59:59-05:00",
    estado: "pendiente",
    permitePostulacion: false,
    orden: 2,
  },
  {
    tipo: "calificacion_meritos",
    nombre: "Fase 3: Valoración y calificación de méritos",
    descripcion:
      "Evaluación técnica y asignación de puntajes por títulos académicos, experiencia profesional, publicaciones y capacitación.",
    fechaInicio: "2026-11-26T00:00:00-05:00",
    fechaFin: "2026-12-05T23:59:59-05:00",
    estado: "pendiente",
    permitePostulacion: false,
    orden: 3,
  },
  {
    tipo: "oposicion_examen",
    nombre: "Fase 4: Examen técnico de conocimientos (Oposición)",
    descripcion:
      "Rendición de la prueba técnica objetiva de conocimientos sobre banca, seguridad social, finanzas y gobernanza corporativa.",
    fechaInicio: "2026-12-06T00:00:00-05:00",
    fechaFin: "2026-12-15T23:59:59-05:00",
    estado: "pendiente",
    permitePostulacion: false,
    orden: 4,
  },
  {
    tipo: "impugnacion_ciudadana",
    nombre: "Fase 5: Impugnación ciudadana y reconsideración",
    descripcion:
      "Período de escrutinio público para presentación y resolución de impugnaciones motivadas de la ciudadanía contra postulantes.",
    fechaInicio: "2026-12-16T00:00:00-05:00",
    fechaFin: "2026-12-24T23:59:59-05:00",
    estado: "pendiente",
    permitePostulacion: false,
    orden: 5,
  },
  {
    tipo: "designacion_final",
    nombre: "Fase 6: Designación oficial y posesión",
    descripcion:
      "Proclamación de resultados oficiales, resolución de designación de vocales principales y alternos y notificación formal.",
    fechaInicio: "2026-12-26T00:00:00-05:00",
    fechaFin: "2026-12-30T23:59:59-05:00",
    estado: "pendiente",
    permitePostulacion: false,
    orden: 6,
  },
]
