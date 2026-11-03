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
