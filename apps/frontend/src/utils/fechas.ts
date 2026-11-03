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
    const ecuador = new Date(fecha.getTime() - 5 * 60 * 60 * 1000)
    const d = ecuador.getUTCDate()
    const m = MESES[ecuador.getUTCMonth()] ?? ""
    const y = ecuador.getUTCFullYear()
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
  const ahora = Date.now()
  const apertura = new Date(aperturaIso).getTime()
  const cierre = new Date(cierreIso).getTime()

  if (
    !Number.isFinite(apertura) ||
    !Number.isFinite(cierre) ||
    cierre <= apertura
  )
    return "cerrada"

  if (ahora < apertura) return "programada"
  if (ahora >= cierre || cerradaFlag) return "cerrada"
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

/** Convierte un instante del servidor al control datetime-local en Ecuador, no a la zona del equipo. */
export function fechaParaControlEcuador(iso: string): string {
  const fecha = new Date(iso)
  if (!Number.isFinite(fecha.getTime())) return ""
  return new Date(fecha.getTime() - 5 * 60 * 60 * 1000)
    .toISOString()
    .slice(0, 16)
}

/** Interpreta un control en UTC-5 explícito y rechaza fechas inexistentes o incompletas. */
export function fechaControlEcuadorAISO(valor: string): string | null {
  if (!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/.test(valor)) return null
  const fecha = new Date(`${valor}:00-05:00`)
  if (
    !Number.isFinite(fecha.getTime()) ||
    fechaParaControlEcuador(fecha.toISOString()) !== valor
  )
    return null
  return fecha.toISOString()
}
