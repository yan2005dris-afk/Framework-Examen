import { esErrorApi } from "@/lib/api-error"
import type {
  ConfiguracionConvocatoria,
  ConvocatoriaPublica,
  FasePublica,
  SectorConvocatoria,
} from "@/types/convocatoria"

export const SECTORES: Record<SectorConvocatoria, string> = {
  afiliados: "Afiliados",
  jubilados: "Jubilados",
  empleadores: "Empleadores",
}
export type EstadoCalendario =
  "programada" | "abierta" | "cerrada" | "suspendida" | "sin_confirmar"

/** Sólo restringe la disponibilidad informada por el servidor; el reloj del navegador nunca autoriza escrituras. */
export function estadoRecepcion(
  c: ConvocatoriaPublica | null | undefined,
  ahora: number,
  consultaValida = true,
): EstadoCalendario {
  if (!c || !consultaValida) return "sin_confirmar"
  const inicio = Date.parse(c.apertura),
    fin = Date.parse(c.cierre)
  if (!Number.isFinite(inicio) || !Number.isFinite(fin) || fin <= inicio)
    return "sin_confirmar"
  if (ahora < inicio) return "programada"
  if (ahora >= fin) return "cerrada"
  const fase = c.fases.find((f) => f.codigo === "postulacion" && f.publicada)
  if (!fase) return "suspendida"
  const desde = Date.parse(fase.apertura),
    hasta = Date.parse(fase.cierre)
  if (!Number.isFinite(desde) || !Number.isFinite(hasta) || hasta <= desde)
    return "sin_confirmar"
  if (ahora < desde) return "programada"
  if (ahora >= hasta) return "cerrada"
  return c.ef12Disponible && fase.habilitada ? "abierta" : "suspendida"
}

export const ETIQUETAS_RECEPCION: Record<EstadoCalendario, string> = {
  programada: "Apertura programada",
  abierta: "Postulaciones abiertas",
  cerrada: "Recepción cerrada",
  suspendida: "Recepción deshabilitada",
  sin_confirmar: "Disponibilidad por confirmar",
}

/** La habilitación operativa no implica que una fase esté ocurriendo ahora ni que existan notas. */
export function estadoFase(fase: FasePublica, ahora: number): string {
  if (!fase.publicada) return "No publicada"
  if (ahora >= Date.parse(fase.cierre)) return "Finalizada"
  if (ahora < Date.parse(fase.apertura)) return "Programada"
  return fase.habilitada ? "Habilitada en plazo" : "Deshabilitada"
}

/** Espejo preventivo de T18. La API conserva la decisión final y valida la versión. */
export function validarConfiguracion(
  config: ConfiguracionConvocatoria,
  apertura: string,
  cierre: string,
): string | null {
  if (
    !config.sectores.length ||
    new Set(config.sectores).size !== config.sectores.length
  )
    return "Seleccione al menos un sector, sin duplicados."
  if (config.publicada && !config.inicioAprobado)
    return "Debe aprobar el inicio antes de publicar."
  if (
    config.ef12Habilitado &&
    (!config.inicioAprobado ||
      !config.publicada ||
      !config.fases.some(
        (f) => f.codigo === "postulacion" && f.publicada && f.habilitada,
      ))
  )
    return "El formulario de postulación requiere una convocatoria aprobada y publicada, y una fase de postulación publicada y habilitada."
  const codigos = new Set<string>()
  for (const fase of config.fases) {
    const inicio = Date.parse(fase.apertura),
      fin = Date.parse(fase.cierre)
    if (codigos.has(fase.codigo))
      return "No puede repetirse el código de una fase."
    codigos.add(fase.codigo)
    if (!fase.titulo.trim() || fase.titulo.length > 200)
      return "Cada fase requiere un título de hasta 200 caracteres."
    if (!Number.isFinite(inicio) || !Number.isFinite(fin) || inicio >= fin)
      return "El cierre de cada fase debe ser posterior a su apertura."
    if (inicio < Date.parse(apertura) || fin > Date.parse(cierre))
      return "Las fases deben estar dentro de las fechas de la convocatoria."
    if (fase.habilitada && !fase.publicada)
      return "Una fase habilitada debe estar publicada."
  }
  return null
}

/** Conserva estado HTTP y código para que un rechazo no quede oculto en una notificación efímera. */
export function detalleErrorConvocatoria(error: unknown): string {
  if (esErrorApi(error))
    return `${error.mensaje} (HTTP ${error.estado} · ${error.codigo})`
  return error instanceof Error
    ? error.message
    : "No fue posible completar la operación."
}

/** Un enlace institucional identifica una convocatoria, nunca una sesión ni una ruta administrativa. */
export function enlacePublicoConvocatoria(origen: string, id: string): string {
  const url = new URL("/", origen)
  url.searchParams.set("convocatoria", id)
  return url.toString()
}
