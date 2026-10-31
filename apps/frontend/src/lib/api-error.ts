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
 *
 * Para errores 429 con cabeceras Retry-After / X-Retry-At, los campos
 * `retryAfterSeg` y `retryAt` contienen los valores del servidor.
 */
export class ErrorApi extends Error {
  readonly codigo: string
  readonly estado: number
  readonly detalles?: string[]
  /** Segundos de espera indicados por el servidor en la cabecera Retry-After. */
  readonly retryAfterSeg?: number
  /** Instante absoluto hasta el que debe esperarse, según X-Retry-At del servidor. */
  readonly retryAt?: Date

  constructor(
    mensaje: string,
    opciones: {
      codigo?: string
      estado: number
      detalles?: string[]
      causa?: unknown
      retryAfterSeg?: number
      retryAt?: Date
    },
  ) {
    super(mensaje, { cause: opciones.causa })
    this.name = "ErrorApi"
    this.codigo = opciones.codigo ?? "error_desconocido"
    this.estado = opciones.estado
    this.detalles = opciones.detalles
    this.retryAfterSeg = opciones.retryAfterSeg
    this.retryAt = opciones.retryAt
  }

  /**
   * Mensaje legible del error en español. Alias de `message`.
   */
  get mensaje(): string {
    return this.message
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
   * Indica si el reenvío del OTP fue limitado por el servidor (429 reenvio_limitado).
   */
  esReenvioLimitado(): boolean {
    return this.estado === 429 && this.codigo === "reenvio_limitado"
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

  /**
   * Indica si el servicio no está disponible temporalmente (503).
   */
  esServicioNoDisponible(): boolean {
    return this.estado === 503
  }
}

/**
 * Type guard para determinar si un valor desconocido es una instancia de ErrorApi.
 */
export function esErrorApi(error: unknown): error is ErrorApi {
  return error instanceof ErrorApi
}
