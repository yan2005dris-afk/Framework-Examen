import type { PostulacionDto } from "@/lib/clientes/postulaciones"

/**
 * Qué postulación corresponde a una pantalla del postulante.
 *
 * - `encontrada`: hay una postulación que mostrar.
 * - `sin_postulacion`: el postulante aún no tiene postulación (en la
 *   convocatoria indicada, si la hay).
 * - `elegir`: tiene varias y la URL no dice cuál; debe elegir.
 */
export type PostulacionActual =
  | {
      estado: "encontrada"
      postulacion: PostulacionDto
      convocatoriaId?: string
    }
  | { estado: "sin_postulacion"; convocatoriaId?: string }
  | { estado: "elegir"; postulaciones: PostulacionDto[] }

/**
 * Resuelve la postulación de la pantalla a partir de `?convocatoria=` y de
 * las postulaciones del servidor. Nunca toma «la primera de la lista» cuando
 * hay varias: el postulante tiene como máximo una por convocatoria y, sin
 * parámetro, elige él.
 * @param convocatoriaId - Valor de `?convocatoria=`; vacío si no llegó
 * @param postulaciones - Postulaciones del servidor (filtradas por convocatoria cuando hay parámetro)
 * @returns El estado de la resolución
 */
export function resolverPostulacionActual(
  convocatoriaId: string | undefined,
  postulaciones: PostulacionDto[],
): PostulacionActual {
  if (convocatoriaId) {
    const postulacion = postulaciones.find(
      (p) => p.convocatoriaId === convocatoriaId,
    )
    return postulacion
      ? { estado: "encontrada", postulacion, convocatoriaId }
      : { estado: "sin_postulacion", convocatoriaId }
  }
  if (postulaciones.length === 1) {
    const [postulacion] = postulaciones
    return {
      estado: "encontrada",
      postulacion,
      convocatoriaId: postulacion.convocatoriaId,
    }
  }
  if (postulaciones.length > 1) {
    return { estado: "elegir", postulaciones }
  }
  return { estado: "sin_postulacion" }
}

/**
 * Decide si la pantalla crea el borrador de la convocatoria de la URL. Tras
 * descartar un borrador no se crea otro solo: hace falta que el postulante
 * vuelva a elegir la convocatoria.
 * @param condiciones.convocatoriaId - Convocatoria de la URL que aún no tiene postulación
 * @param condiciones.abierta - La convocatoria admite postulaciones
 * @param condiciones.creacionLibre - No hay otra creación en curso ni terminada en la pantalla
 * @param condiciones.borradorDescartado - El postulante acaba de descartar su borrador
 * @returns true si corresponde crear el borrador
 */
export function debeCrearBorrador(condiciones: {
  convocatoriaId: string | undefined
  abierta: boolean
  creacionLibre: boolean
  borradorDescartado: boolean
}): boolean {
  const { convocatoriaId, abierta, creacionLibre, borradorDescartado } =
    condiciones
  return (
    Boolean(convocatoriaId) && abierta && creacionLibre && !borradorDescartado
  )
}

/**
 * Quita una postulación de una página en caché. Una copia vieja que todavía la
 * incluya haría que la pantalla vuelva a seleccionarla después de eliminarla.
 * @param datos - Datos de una consulta de postulaciones (página, cursor u otro)
 * @param id - Postulación eliminada
 * @returns Los mismos datos sin esa postulación; sin cambios si no son una página
 */
export function quitarPostulacionDePagina<T>(datos: T, id: string): T {
  if (typeof datos !== "object" || datos === null || !("items" in datos)) {
    return datos
  }
  const { items } = datos as { items: unknown }
  if (!Array.isArray(items)) return datos
  const restantes = items.filter(
    (item: unknown) =>
      typeof item !== "object" ||
      item === null ||
      (item as { id?: unknown }).id !== id,
  )
  return restantes.length === items.length
    ? datos
    : { ...datos, items: restantes }
}
