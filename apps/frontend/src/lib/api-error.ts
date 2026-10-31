/**
 * Detalle opcional devuelto por la API en respuestas de error.
 */
export interface DetalleErrorApi {
  codigo: string
  mensaje: string
  detalles?: string[]
}

/**
 * Error unificado para todas las respuestas no exitosas o fallos de red de la API.
 * Extrae el código del envelope estándar del backend ({ error: { codigo, mensaje, detalle } })
 * permitiendo a las vistas y guards tomar decisiones sin parsear texto.
 */
export class ErrorApi extends Error {
  readonly codigo: string
  readonly estado: number
  readonly detalles?: string[]

  constructor(
    mensaje: string,
    opciones: {
      codigo?: string
      estado: number
      detalles?: string[]
      causa?: unknown
    },
  ) {
    super(mensaje, { cause: opciones.causa })
    this.name = "ErrorApi"
    this.codigo = opciones.codigo ?? "error_desconocido"
    this.estado = opciones.estado
    this.detalles = opciones.detalles
  }

  /**
   * Indica si la petición fue rechazada por falta o expiración de sesión (401).
   */
  esNoAutorizado(): boolean {
    return this.estado === 401
  }

  /**
   * Indica si el acceso fue denegado por permisos insuficientes (403).
   */
  esProhibido(): boolean {
    return this.estado === 403
  }

  /**
   * Indica si el recurso solicitado no existe (404).
   */
  esNoEncontrado(): boolean {
    return this.estado === 404
  }

  /**
   * Indica si la cuenta o IP fue bloqueada por superar los intentos de ingreso (429).
   */
  esBloqueoIntentos(): boolean {
    return this.estado === 429 || this.codigo === "intentos_excedidos"
  }

  /**
   * Indica si ocurrió un conflicto de concurrencia o postulación ya enviada (409).
   */
  esConflicto(): boolean {
    return this.estado === 409
  }

  /**
   * Indica si falló la validación de reglas de negocio o campos (422 o 400).
   */
  esErrorValidacion(): boolean {
    return this.estado === 422 || this.estado === 400
  }

  /**
   * Indica si se trata de un error de conectividad o red (estado 0).
   */
  esErrorRed(): boolean {
    return this.estado === 0 || this.codigo === "error_red"
  }
}

/**
 * Type guard para determinar si un valor desconocido es una instancia de ErrorApi.
 */
export function esErrorApi(error: unknown): error is ErrorApi {
  return error instanceof ErrorApi
}
