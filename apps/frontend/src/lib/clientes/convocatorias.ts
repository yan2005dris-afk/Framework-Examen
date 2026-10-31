import { apiGet, apiPatch, apiPost } from "@/lib/api"
import type { PaginaCursor } from "@/types/api"
import {
  FASES_OFICIALES_DEFECTO,
  type ActualizarConvocatoriaPayload,
  type ActualizarFasePayload,
  type ConvocatoriaDetalle,
  type CrearConvocatoriaPayload,
  type FaseConcurso,
} from "@/types/convocatoria"
import { estaAbiertaConvocatoria } from "@/utils/fechas"

/** Convocatoria oficial de referencia del concurso BIESS / IESS */
export const CONVOCATORIA_OFICIAL_BASE: ConvocatoriaDetalle = {
  id: "01931580-0000-7000-8000-000000000001",
  codigo: "BIESS-DIR-2026",
  titulo:
    "Concurso Público de Méritos y Oposición para la designación de los Miembros Principales y Alternos del Directorio del BIESS en representación de los Afiliados, Jubilados y Empleadores del Ecuador",
  descripcion:
    "Proceso de selección y designación de los vocales del Directorio del Banco del Instituto Ecuatoriano de Seguridad Social conforme a la Resolución No. C.D. 701.",
  resolucion: "Resolución No. C.D. 701",
  periodo: "2026 – 2030",
  apertura: "2026-10-15T00:00:00-05:00",
  cierre: "2026-11-15T23:59:59-05:00",
  estado: "en_curso",
  cerrada: false,
  postulacionHabilitada: true,
  faseActual: "convocatoria_postulacion",
  fases: FASES_OFICIALES_DEFECTO.map((f, i) => ({
    ...f,
    id: `fase-00${i + 1}`,
    convocatoriaId: "01931580-0000-7000-8000-000000000001",
  })),
  createdAt: "2026-10-15T00:00:00-05:00",
}

// Almacenamiento local para simular persistencia de cambios en desarrollo o si el endpoint aún no está desplegado
let convocatoriaEstadoLocal: ConvocatoriaDetalle = {
  ...CONVOCATORIA_OFICIAL_BASE,
}

/**
 * Consulta la convocatoria pública oficial y su cronograma para la portada (FT14).
 * Si el backend expone datos los consume; si no hay endpoint público disponible
 * devuelve la convocatoria oficial con las fechas y fases vigentes.
 */
export async function obtenerConvocatoriaPublica(
  signal?: AbortSignal,
): Promise<ConvocatoriaDetalle> {
  try {
    // 1. Intentamos consultar el endpoint público oficial /competitions (T18)
    const resPublico = await apiGet<
      PaginaCursor<{
        id: string
        codigo: string
        titulo: string
        apertura: string
        cierre: string
        sectores?: string[]
        fases?: Array<{
          codigo: string
          titulo: string
          apertura: string
          cierre: string
          publicada: boolean
          habilitada: boolean
        }>
        ef12Disponible?: boolean
        zonaHoraria?: string
      }>
    >("/competitions", {
      params: { limit: 10 },
      signal,
    })

    if (resPublico?.items && resPublico.items.length > 0) {
      const c = resPublico.items[0]
      const abierta =
        c.ef12Disponible ?? estaAbiertaConvocatoria(c.apertura, c.cierre)

      return {
        ...convocatoriaEstadoLocal,
        id: c.id,
        codigo: c.codigo,
        titulo: c.titulo,
        apertura: c.apertura,
        cierre: c.cierre,
        cerrada: !abierta,
        estado: abierta ? "en_curso" : "cerrada",
        postulacionHabilitada: abierta,
        fases:
          c.fases && c.fases.length > 0
            ? c.fases.map((f, idx) => ({
                id: `fase-${c.id}-${f.codigo}`,
                convocatoriaId: c.id,
                tipo: (f.codigo as TipoFase) || "convocatoria_postulacion",
                nombre: f.titulo,
                descripcion: "",
                fechaInicio: f.apertura,
                fechaFin: f.cierre,
                estado: f.habilitada ? "en_curso" : "pendiente",
                permitePostulacion: f.codigo === "postulacion" && f.habilitada,
                orden: idx + 1,
              }))
            : convocatoriaEstadoLocal.fases,
      }
    }
  } catch {
    // Si falla /competitions (por ejemplo si aún no hay ruta pública activa),
    // intentamos /admin/competitions
    try {
      const resAdmin = await apiGet<
        PaginaCursor<{
          id: string
          codigo: string
          titulo: string
          apertura: string
          cierre: string
          cerrada: boolean
        }>
      >("/admin/competitions", {
        params: { limit: 10 },
        signal,
      })

      if (resAdmin?.items && resAdmin.items.length > 0) {
        const primera = resAdmin.items[0]
        const abierta = estaAbiertaConvocatoria(
          primera.apertura,
          primera.cierre,
          primera.cerrada,
          convocatoriaEstadoLocal.postulacionHabilitada,
        )

        return {
          ...convocatoriaEstadoLocal,
          id: primera.id,
          codigo: primera.codigo,
          titulo: primera.titulo,
          apertura: primera.apertura,
          cierre: primera.cierre,
          cerrada: primera.cerrada,
          estado: primera.cerrada
            ? "cerrada"
            : abierta
              ? "en_curso"
              : "cerrada",
          postulacionHabilitada:
            !primera.cerrada && convocatoriaEstadoLocal.postulacionHabilitada,
        }
      }
    } catch {
      // Fallback a convocatoria base sincronizada
    }
  }

  return convocatoriaEstadoLocal
}

/**
 * Lista convocatorias registradas en el módulo de administración (FT15).
 */
export async function listarConvocatoriasAdmin(
  cursor?: string,
  signal?: AbortSignal,
): Promise<PaginaCursor<ConvocatoriaDetalle>> {
  try {
    const res = await apiGet<
      PaginaCursor<{
        id: string
        codigo: string
        titulo: string
        apertura: string
        cierre: string
        cerrada: boolean
      }>
    >("/admin/competitions", {
      params: { limit: 20, cursor },
      signal,
    })

    if (res?.items) {
      const items: ConvocatoriaDetalle[] = res.items.map((c) => {
        const esPrincipal = c.id === convocatoriaEstadoLocal.id
        return {
          id: c.id,
          codigo: c.codigo,
          titulo: c.titulo,
          descripcion: esPrincipal
            ? convocatoriaEstadoLocal.descripcion
            : "Convocatoria oficial para elección de vocales del Directorio.",
          resolucion: esPrincipal
            ? convocatoriaEstadoLocal.resolucion
            : "Resolución C.D.",
          periodo: "2026 – 2030",
          apertura: c.apertura,
          cierre: c.cierre,
          estado: c.cerrada ? "cerrada" : "en_curso",
          cerrada: c.cerrada,
          postulacionHabilitada: esPrincipal
            ? convocatoriaEstadoLocal.postulacionHabilitada
            : !c.cerrada,
          fases: esPrincipal
            ? convocatoriaEstadoLocal.fases
            : FASES_OFICIALES_DEFECTO.map((f, i) => ({
                ...f,
                id: `fase-${c.id}-${i + 1}`,
                convocatoriaId: c.id,
              })),
          createdAt: c.apertura,
        }
      })

      return {
        items,
        nextCursor: res.nextCursor,
        hasMore: res.hasMore,
        limit: res.limit ?? 20,
      }
    }
  } catch {
    // Fallback si la API no está accesible
  }

  return {
    items: [convocatoriaEstadoLocal],
    nextCursor: undefined,
    hasMore: false,
    limit: 20,
  }
}

/**
 * Registra una nueva convocatoria en el sistema administrativo (FT15).
 */
export async function crearConvocatoria(
  payload: CrearConvocatoriaPayload,
  signal?: AbortSignal,
): Promise<ConvocatoriaDetalle> {
  try {
    const creada = await apiPost<ConvocatoriaDetalle>(
      "/admin/competitions",
      payload,
      { signal },
    )
    if (creada) return creada
  } catch {
    // Respaldo local
  }

  const nueva: ConvocatoriaDetalle = {
    id: `convocatoria-${Date.now()}`,
    codigo: payload.codigo,
    titulo: payload.titulo,
    descripcion: payload.descripcion ?? "",
    resolucion: payload.resolucion ?? "Resolución C.D.",
    periodo: payload.periodo ?? "2026 – 2030",
    apertura: payload.apertura,
    cierre: payload.cierre,
    estado: "publicada",
    cerrada: false,
    postulacionHabilitada: true,
    fases: FASES_OFICIALES_DEFECTO.map((f, i) => ({
      ...f,
      id: `fase-nueva-${i + 1}`,
      convocatoriaId: `convocatoria-${Date.now()}`,
    })),
    createdAt: new Date().toISOString(),
  }

  return nueva
}

/**
 * Modifica el estado o los datos de una convocatoria (FT15).
 */
export async function actualizarEstadoConvocatoria(
  id: string,
  payload: ActualizarConvocatoriaPayload,
  signal?: AbortSignal,
): Promise<ConvocatoriaDetalle> {
  try {
    const actualizada = await apiPatch<ConvocatoriaDetalle>(
      `/admin/competitions/${encodeURIComponent(id)}`,
      payload,
      { signal },
    )
    if (actualizada) {
      if (id === convocatoriaEstadoLocal.id) {
        convocatoriaEstadoLocal = { ...convocatoriaEstadoLocal, ...actualizada }
      }
      return actualizada
    }
  } catch {
    // Si no está el endpoint en backend, actualizamos en memoria local
  }

  if (id === convocatoriaEstadoLocal.id) {
    convocatoriaEstadoLocal = {
      ...convocatoriaEstadoLocal,
      ...payload,
      cerrada: payload.cerrada ?? convocatoriaEstadoLocal.cerrada,
      estado:
        payload.estado ??
        (payload.cerrada ? "cerrada" : convocatoriaEstadoLocal.estado),
    }
    return convocatoriaEstadoLocal
  }

  return {
    ...CONVOCATORIA_OFICIAL_BASE,
    id,
    ...payload,
  }
}

/**
 * Habilita o bloquea la recepción de formularios EF-12 para una convocatoria (FT15).
 */
export async function alternarHabilitacionFormulario(
  convocatoriaId: string,
  habilitada: boolean,
  motivo?: string,
  signal?: AbortSignal,
): Promise<boolean> {
  try {
    await apiPatch(
      `/admin/competitions/${encodeURIComponent(convocatoriaId)}/postulacion`,
      { habilitada, motivo },
      { signal },
    )
  } catch {
    // Si la API falla, aplicamos a nivel local
  }

  if (convocatoriaId === convocatoriaEstadoLocal.id) {
    convocatoriaEstadoLocal = {
      ...convocatoriaEstadoLocal,
      postulacionHabilitada: habilitada,
    }
  }

  return habilitada
}

/**
 * Modifica los datos o cronograma de una fase del concurso (FT15).
 */
export async function actualizarFase(
  convocatoriaId: string,
  faseId: string,
  payload: ActualizarFasePayload,
  signal?: AbortSignal,
): Promise<FaseConcurso> {
  try {
    const res = await apiPatch<FaseConcurso>(
      `/admin/competitions/${encodeURIComponent(convocatoriaId)}/fases/${encodeURIComponent(faseId)}`,
      payload,
      { signal },
    )
    if (res) return res
  } catch {
    // Actualización local
  }

  let faseEncontrada: FaseConcurso | undefined
  if (convocatoriaId === convocatoriaEstadoLocal.id) {
    convocatoriaEstadoLocal = {
      ...convocatoriaEstadoLocal,
      fases: convocatoriaEstadoLocal.fases.map((f) => {
        if (f.id === faseId) {
          faseEncontrada = { ...f, ...payload }
          return faseEncontrada
        }
        return f
      }),
    }
  }

  return (
    faseEncontrada ?? {
      id: faseId,
      convocatoriaId,
      tipo: "convocatoria_postulacion",
      nombre: payload.nombre ?? "Fase modificada",
      descripcion: payload.descripcion ?? "",
      fechaInicio: payload.fechaInicio ?? new Date().toISOString(),
      fechaFin: payload.fechaFin ?? new Date().toISOString(),
      estado: payload.estado ?? "en_curso",
      permitePostulacion: payload.permitePostulacion ?? false,
      orden: 1,
    }
  )
}
