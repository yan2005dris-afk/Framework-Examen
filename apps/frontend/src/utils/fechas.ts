/**
 * Utilidades para manejo de fechas y zona horaria de Ecuador (America/Guayaquil, UTC-5).
 * Requerido por FT14 y FT15 para sincronización temporal con el servidor y T18.
 */

export const ZONA_HORARIA_ECUADOR = "America/Guayaquil"

const MESES = [
  "enero",
  "febrero",
  "marzo",
  "abril",
  "mayo",
  "junio",
  "julio",
  "agosto",
  "septiembre",
  "octubre",
  "noviembre",
  "diciembre",
]

/**
 * Formatea una fecha ISO en la zona horaria de Ecuador (America/Guayaquil).
 * Ejemplo: "15 de noviembre de 2026 · 23:59 (ECT)" o "15 de noviembre de 2026".
 */
export function formatearFechaEcuador(
  fechaIso: string,
  opciones?: { incluirHora?: boolean },
): string {
  if (!fechaIso) return ""
  const fecha = new Date(fechaIso)
  if (Number.isNaN(fecha.getTime())) return fechaIso

  try {
    const formatter = new Intl.DateTimeFormat("es-EC", {
      timeZone: ZONA_HORARIA_ECUADOR,
      day: "numeric",
      month: "long",
      year: "numeric",
      hour: opciones?.incluirHora ? "2-digit" : undefined,
      minute: opciones?.incluirHora ? "2-digit" : undefined,
      hour12: false,
    })

    const partes = formatter.formatToParts(fecha)
    const dia = partes.find((p) => p.type === "day")?.value ?? ""
    const mes = partes.find((p) => p.type === "month")?.value ?? ""
    const anio = partes.find((p) => p.type === "year")?.value ?? ""

    if (opciones?.incluirHora) {
      const hora = partes.find((p) => p.type === "hour")?.value ?? "00"
      const minuto = partes.find((p) => p.type === "minute")?.value ?? "00"
      return `${dia} de ${mes} de ${anio} · ${hora}:${minuto} (ECT)`
    }

    return `${dia} de ${mes} de ${anio}`
  } catch {
    // Respaldo en caso de que Intl con timeZone falle en navegadores antiguos
    const d = fecha.getUTCDate()
    const m = MESES[fecha.getUTCMonth()] ?? ""
    const y = fecha.getUTCFullYear()
    return `${d} de ${m} de ${y}`
  }
}

/**
 * Formatea un rango de fechas.
 * Ejemplo: "15 de octubre al 15 de noviembre de 2026"
 */
export function formatearRangoFechas(
  inicioIso: string,
  finIso: string,
): string {
  if (!inicioIso || !finIso) return ""
  const fInicio = new Date(inicioIso)
  const fFin = new Date(finIso)
  if (Number.isNaN(fInicio.getTime()) || Number.isNaN(fFin.getTime())) return ""

  const inicioTexto = formatearFechaEcuador(inicioIso)
  const finTexto = formatearFechaEcuador(finIso)

  return `${inicioTexto} al ${finTexto}`
}

/**
 * Determina el estado temporal de una convocatoria o fase.
 */
export function obtenerEstadoTemporal(
  aperturaIso: string,
  cierreIso: string,
  cerradaFlag?: boolean,
): "programada" | "abierta" | "cerrada" {
  if (cerradaFlag) return "cerrada"

  const ahora = Date.now()
  const apertura = new Date(aperturaIso).getTime()
  const cierre = new Date(cierreIso).getTime()

  if (Number.isNaN(apertura) || Number.isNaN(cierre)) return "abierta"

  if (ahora < apertura) return "programada"
  if (ahora >= cierre) return "cerrada"
  return "abierta"
}

/**
 * Indica si una convocatoria admite nuevas postulaciones en este momento.
 */
export function estaAbiertaConvocatoria(
  aperturaIso: string,
  cierreIso: string,
  cerradaFlag?: boolean,
  postulacionHabilitada?: boolean,
): boolean {
  if (cerradaFlag === true) return false
  if (postulacionHabilitada === false) return false

  return (
    obtenerEstadoTemporal(aperturaIso, cierreIso, cerradaFlag) === "abierta"
  )
}

/**
 * Calcula los días completos restantes hasta la fecha de cierre.
 * Si ya cerró, devuelve 0.
 */
export function diasHastaCierre(cierreIso: string): number {
  const cierre = new Date(cierreIso).getTime()
  if (Number.isNaN(cierre)) return 0

  const ahora = Date.now()
  const diferenciaMs = cierre - ahora
  if (diferenciaMs <= 0) return 0

  return Math.ceil(diferenciaMs / (1000 * 60 * 60 * 24))
}
