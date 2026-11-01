/**
 * Tipos de dominio para convocatorias y fases según el contrato compartido T18 y COMPETITION_LIFECYCLE.md.
 * No define datos ficticios ni cronogramas precargados.
 */

export type CodigoFase =
  | "postulacion"
  | "meritos"
  | "oposicion_teorica"
  | "oposicion_psicotecnica"
  | "reconsideracion"
  | "resultados"

export type SectorConvocatoria = "afiliados" | "jubilados" | "empleadores"

export interface FasePublica {
  codigo: CodigoFase
  titulo: string
  apertura: string
  cierre: string
  publicada: boolean
  habilitada: boolean
}

/** Modelo público devuelto por GET /api/v1/competitions y GET /api/v1/competitions/{id} */
export interface ConvocatoriaPublica {
  id: string
  codigo: string
  titulo: string
  apertura: string
  cierre: string
  sectores: SectorConvocatoria[]
  fases: FasePublica[]
  ef12Disponible: boolean
  zonaHoraria: string
  limites: string
}

/** Configuración administrativa completa con control optimista de versión */
export interface ConfiguracionConvocatoria {
  version: number
  inicioAprobado: boolean
  publicada: boolean
  ef12Habilitado: boolean
  sectores: SectorConvocatoria[]
  fases: FasePublica[]
}

/** Modelo de administración devuelto por GET /api/v1/admin/competitions/{id} */
export interface ConvocatoriaAdmin extends ConfiguracionConvocatoria {
  id: string
  codigo: string
  titulo: string
  apertura: string
  cierre: string
  aprobadoEn?: string
  aprobadoPor?: string
}

/** Resumen devuelto en listados administrativos (T9/T18) */
export interface ResumenConvocatoriaAdmin {
  id: string
  codigo: string
  titulo: string
  apertura: string
  cierre: string
  cerrada: boolean
}

export interface CrearConvocatoriaPayload {
  codigo: string
  titulo: string
  apertura: string
  cierre: string
  sectores: SectorConvocatoria[]
  fases?: FasePublica[]
}

/** Catálogo de nombres institucionales para los códigos de fase del servidor */
export const NOMBRES_FASES_OFICIALES: Record<CodigoFase, string> = {
  postulacion: "Recepción de postulaciones",
  meritos: "Valoración de méritos",
  oposicion_teorica: "Prueba técnica de conocimientos",
  oposicion_psicotecnica: "Evaluación psicométrica",
  reconsideracion: "Impugnación y reconsideración",
  resultados: "Proclamación de resultados y designación",
}
