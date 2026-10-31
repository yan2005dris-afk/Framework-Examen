import type { PostulanteDto } from "@/lib/clientes/auth"

/**
 * Nivel de acceso que exige una ruta del portal.
 *
 * - `publico`: visible sin sesión.
 * - `con-sesion`: exige un postulante autenticado.
 * - `con-rol-analista`: exige sesión y `rol === "analista"` o `"administrador"`.
 * - `con-rol-admin`: exige sesión y `rol === "administrador"` (Comisión de Selección).
 */
export type TipoAcceso =
  "publico" | "con-sesion" | "con-rol-analista" | "con-rol-admin"

/**
 * Rutas estáticas del portal. Las rutas con parámetros se arman con
 * {@link rutaSeccion} y {@link rutaDetallePostulacion}.
 */
export const RUTAS = {
  inicio: "/",
  registro: "/registro",
  login: "/login",
  recuperarContrasena: "/recuperar-contrasena",
  postulacion: "/postulacion",
  postulacionRevision: "/postulacion/revision",
  postulacionImprimir: "/postulacion/imprimir",
  dashboard: "/dashboard",
  dashboardPostulaciones: "/dashboard/postulaciones",
  // Seguimiento, reconsideraciones y evaluación EF-14 (FT14-FT27)
  seguimiento: "/seguimiento",
  reconsideracion: "/reconsideracion",
  evaluacion: "/evaluacion",
  evaluacionExamen: "/evaluacion/examen",
  evaluacionResultado: "/evaluacion/resultado",
  adminConvocatorias: "/admin/convocatorias",
  adminFases: "/admin/fases",
  adminPreguntas: "/admin/preguntas",
} as const

/**
 * Acceso exigido por cada prefijo de ruta. Las rutas hijas
 * (`/postulacion/:seccion`, `/dashboard/postulaciones/:id`) heredan el acceso
 * de su prefijo, por eso el guard se monta sobre el layout y no sobre la hoja.
 */
export const ACCESO_POR_RUTA: Record<string, TipoAcceso> = {
  [RUTAS.inicio]: "publico",
  [RUTAS.registro]: "publico",
  [RUTAS.login]: "publico",
  [RUTAS.recuperarContrasena]: "publico",
  [RUTAS.postulacion]: "con-sesion",
  [RUTAS.postulacionRevision]: "con-sesion",
  [RUTAS.postulacionImprimir]: "con-sesion",
  [RUTAS.seguimiento]: "con-sesion",
  [RUTAS.reconsideracion]: "con-sesion",
  [RUTAS.evaluacion]: "con-sesion",
  [RUTAS.evaluacionExamen]: "con-sesion",
  [RUTAS.evaluacionResultado]: "con-sesion",
  [RUTAS.dashboard]: "con-rol-analista",
  [RUTAS.dashboardPostulaciones]: "con-rol-analista",
  [RUTAS.adminConvocatorias]: "con-rol-admin",
  [RUTAS.adminFases]: "con-rol-admin",
  [RUTAS.adminPreguntas]: "con-rol-admin",
}

/** Ruta del formulario para una sección concreta del EF-12. */
export function rutaSeccion(seccion: number | string): string {
  return `${RUTAS.postulacion}/${seccion}`
}

/** Ruta del detalle de una postulación en el panel del analista. */
export function rutaDetallePostulacion(id: string): string {
  return `${RUTAS.dashboardPostulaciones}/${id}`
}

/**
 * Indica si la cuenta tiene rol de analista (o superior).
 */
export function esAnalista(
  postulante: PostulanteDto | null | undefined,
): boolean {
  return postulante?.rol === "analista" || postulante?.rol === "administrador"
}

/**
 * Indica si la cuenta tiene rol de administrador (Comisión de Selección).
 */
export function esAdministrador(
  postulante: PostulanteDto | null | undefined,
): boolean {
  return postulante?.rol === "administrador"
}

/**
 * Decide si una sesión puede abrir una ruta con el acceso indicado.
 */
export function puedeAcceder(
  acceso: TipoAcceso,
  sesion: PostulanteDto | null,
): boolean {
  if (acceso === "publico") {
    return true
  }
  if (!sesion) {
    return false
  }
  if (acceso === "con-rol-admin") {
    return esAdministrador(sesion)
  }
  if (acceso === "con-rol-analista") {
    return esAnalista(sesion)
  }
  return true
}

/**
 * Ruta inicial sugerida después de autenticarse.
 *
 * - Administrador -> Panel de convocatorias y fases.
 * - Analista -> Dashboard de postulaciones.
 * - Postulante -> Formulario EF-12.
 */
export function rutaInicial(postulante: PostulanteDto | null): string {
  if (!postulante) {
    return RUTAS.login
  }
  if (esAdministrador(postulante)) {
    return RUTAS.adminConvocatorias
  }
  if (esAnalista(postulante)) {
    return RUTAS.dashboard
  }
  return RUTAS.postulacion
}

/**
 * Construye la URL de login conservando el destino al que se quería llegar.
 * Los destinos vacíos o que apuntan al propio login no se arrastran.
 */
export function construirRutaLogin(destino?: string): string {
  const destinoLimpio = destino?.trim() ?? ""
  if (
    destinoLimpio === "" ||
    destinoLimpio === RUTAS.login ||
    destinoLimpio === RUTAS.inicio
  ) {
    return RUTAS.login
  }

  const parametros = new URLSearchParams({ destino: destinoLimpio })
  return `${RUTAS.login}?${parametros.toString()}`
}

/**
 * Lee el destino guardado en la query string del login.
 * Si no existe (o viene vacío) devuelve la ruta indicada como respaldo.
 */
export function leerDestinoLogin(
  busqueda: string,
  porDefecto: string = RUTAS.inicio,
): string {
  const parametros = new URLSearchParams(busqueda)
  const destino = parametros.get("destino")?.trim() ?? ""
  return destino === "" ? porDefecto : destino
}
