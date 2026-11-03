/**
 * Respuesta genérica de paginación basada en offset.
 */
export interface Pagina<T> {
  items: T[]
  page: number
  pageSize: number
  total: number
}

/**
 * Respuesta genérica de paginación basada en cursor.
 */
export interface PaginaCursor<T> {
  items: T[]
  nextCursor?: string
  hasMore: boolean
  limit: number
}

/**
 * Respuesta estándar con mensaje informativo del backend.
 */
export interface RespuestaMensaje {
  mensaje: string
}
