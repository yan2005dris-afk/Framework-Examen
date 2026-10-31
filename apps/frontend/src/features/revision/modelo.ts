import { esUuidRevision, type FiltrosRevision } from "@/lib/clientes/revision"
import { calcularDuracionExperiencia } from "@/utils/validators"

export const SECTORES_REVISION = [
  { valor: "afiliados", etiqueta: "Afiliados" },
  { valor: "jubilados", etiqueta: "Jubilados" },
  { valor: "empleadores", etiqueta: "Empleadores" },
] as const

/** Lee los filtros compartidos por panel, nómina y matriz sin degradar un UUID inválido a legado. */
export function leerFiltrosRevision(params: URLSearchParams): {
  filtros: FiltrosRevision
  error?: string
} {
  const periodo = (params.get("convocatoria") ?? "").toLowerCase()
  const sector = params.get("sector") ?? ""
  const estado = params.get("estado") ?? ""
  const filtros: FiltrosRevision = {
    periodo,
    sector: sector as FiltrosRevision["sector"],
    estado: estado as FiltrosRevision["estado"],
  }
  const repetidos = ["convocatoria", "sector", "estado"].some(
    (clave) => params.getAll(clave).length > 1,
  )
  if (
    repetidos ||
    (periodo !== "" && periodo !== "legado" && !esUuidRevision(periodo)) ||
    !["", ...SECTORES_REVISION.map((s) => s.valor)].includes(sector) ||
    !["", "submitted"].includes(estado)
  ) {
    return {
      filtros,
      error:
        "La selección de la URL no es válida. Corrija los filtros para consultar.",
    }
  }
  return { filtros }
}

/** Serializa solo filtros de la vista; no conserva cursores de otra selección. */
export function busquedaRevision(filtros: FiltrosRevision): string {
  const params = new URLSearchParams()
  if (filtros.periodo) params.set("convocatoria", filtros.periodo)
  if (filtros.sector) params.set("sector", filtros.sector)
  if (filtros.estado) params.set("estado", filtros.estado)
  return params.size ? `?${params}` : ""
}

/** Formatea instantes en la zona del concurso, sin modificar las fechas del snapshot. */
export function fechaRevision(valor?: string): string {
  if (!valor || !Number.isFinite(Date.parse(valor))) return "No disponible"
  return new Intl.DateTimeFormat("es-EC", {
    dateStyle: "medium",
    timeStyle: "short",
    timeZone: "America/Guayaquil",
  }).format(new Date(valor))
}

export function esObjetoRevision(
  valor: unknown,
): valor is Record<string, unknown> {
  return typeof valor === "object" && valor !== null && !Array.isArray(valor)
}

/** El cálculo legado se corta en el envío, nunca en la fecha actual de consulta. */
export function duracionHistorica(
  experiencia: Record<string, unknown>,
  enviadoEn: string,
): string {
  const almacenada = experiencia.duracionCalculada
  if (
    esObjetoRevision(almacenada) &&
    ["anios", "meses", "dias"].every(
      (k) =>
        typeof almacenada[k] === "number" &&
        Number.isFinite(almacenada[k]) &&
        Number(almacenada[k]) >= 0,
    )
  ) {
    return `${almacenada.anios} años, ${almacenada.meses} meses, ${almacenada.dias} días (al sello)`
  }
  if (
    typeof experiencia.fechaInicio !== "string" ||
    !/^\d{4}-\d{2}-\d{2}$/.test(experiencia.fechaInicio) ||
    !Number.isFinite(Date.parse(enviadoEn))
  )
    return "Duración no disponible"
  const corte = new Intl.DateTimeFormat("en-CA", {
    timeZone: "America/Guayaquil",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).format(new Date(enviadoEn))
  const fin =
    experiencia.actualmenteLaborando === true ? corte : experiencia.fechaFin
  if (
    typeof fin !== "string" ||
    !/^\d{4}-\d{2}-\d{2}$/.test(fin) ||
    fin > corte
  )
    return "Duración no disponible"
  const duracion = calcularDuracionExperiencia(
    experiencia.fechaInicio,
    fin,
    false,
  )
  if (duracion.texto === "Fecha no válida") return "Duración no disponible"
  return `${duracion.anios} años, ${duracion.meses} meses, ${duracion.dias} días (calculada con fechas del envío; legado)`
}

/** Solo admite capacidades HTTP(S) vigentes; no interpreta URLs del contenido declarado. */
export function enlaceTemporalValido(
  enlace: { url: string; expiresAt: string },
  ahora = Date.now(),
): boolean {
  try {
    const url = new URL(enlace.url)
    return (
      ["https:", "http:"].includes(url.protocol) &&
      !url.username &&
      !url.password &&
      Date.parse(enlace.expiresAt) > ahora
    )
  } catch {
    return false
  }
}
