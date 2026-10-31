import { rutaInicial } from "@/lib/rutas"
import { useSesion } from "./sesion-context"

/**
 * Ruta a la que conviene entrar con la sesión activa.
 *
 * La decisión es puramente derivada del estado de sesión (ver {@link rutaInicial}),
 * así que no dispara consultas ni necesita estado propio: la portada puede
 * pintar el acceso correcto sin esperar a la red.
 *
 * @returns La ruta sugerida (`null` mientras la sesión se resuelve) y si la
 * decisión sigue en curso.
 */
export function useRutaInicial(): { ruta: string | null; calculando: boolean } {
  const { postulante, estado } = useSesion()

  if (estado === "cargando") {
    return { ruta: null, calculando: true }
  }
  if (estado === "anonimo" || !postulante) {
    return { ruta: null, calculando: false }
  }
  return { ruta: rutaInicial(postulante), calculando: false }
}
