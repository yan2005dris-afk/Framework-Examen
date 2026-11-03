import { ErrorApi } from "./api-error"
import { API_BASE_URL } from "./config"

/**
 * Parámetros de consulta serializables en la URL.
 */
export type ParametrosConsulta = Record<
  string,
  string | number | boolean | undefined | null
>

/**
 * Opciones para las peticiones HTTP al backend.
 */
export interface OpcionesPeticion extends Omit<RequestInit, "body"> {
  params?: ParametrosConsulta
  body?: unknown
  baseUrl?: string
  formatoRespuesta?: "blob"
}

/**
 * Callback invocado cuando la API devuelve un código 401 (no autorizado).
 */
export type ManejadorNoAutorizado = () => void

let manejadorNoAutorizadoGlobal: ManejadorNoAutorizado | null = null

/**
 * Registra un manejador global para respuestas 401 (útil para redirigir a login o limpiar sesión).
 */
export function configurarManejadorNoAutorizado(
  manejador: ManejadorNoAutorizado | null,
): void {
  manejadorNoAutorizadoGlobal = manejador
}

/**
 * Construye una URL completa combinando la base, la ruta relativa y los parámetros de consulta.
 * Omite claves con valores nulos o indefinidos.
 */
export function construirUrl(
  ruta: string,
  params?: ParametrosConsulta,
  baseUrl = API_BASE_URL,
): string {
  const urlBase =
    ruta.startsWith("http://") || ruta.startsWith("https://")
      ? ruta
      : `${baseUrl.replace(/\/+$/, "")}/${ruta.replace(/^\/+/, "")}`

  if (!params) {
    return urlBase
  }

  const queryParams = new URLSearchParams()
  for (const [clave, valor] of Object.entries(params)) {
    if (valor !== undefined && valor !== null && valor !== "") {
      queryParams.append(clave, String(valor))
    }
  }

  const queryString = queryParams.toString()
  if (!queryString) {
    return urlBase
  }

  return urlBase.includes("?")
    ? `${urlBase}&${queryString}`
    : `${urlBase}?${queryString}`
}

/**
 * Extrae la información de error desde la respuesta HTTP del backend.
 * También parsea las cabeceras Retry-After y X-Retry-At para errores 429.
 */
async function procesarErrorRespuesta(respuesta: Response): Promise<never> {
  let codigo = "error_desconocido"
  let mensaje =
    respuesta.statusText || `Error en la petición (${respuesta.status})`
  let detalles: string[] | undefined

  try {
    const tipoContenido = respuesta.headers.get("content-type") ?? ""
    if (tipoContenido.includes("application/json")) {
      const cuerpo = await respuesta.json()
      if (cuerpo && typeof cuerpo === "object") {
        const errorInterno =
          "error" in cuerpo ? (cuerpo as { error: unknown }).error : null
        if (errorInterno && typeof errorInterno === "object") {
          const errObj = errorInterno as Record<string, unknown>
          if (typeof errObj.codigo === "string") codigo = errObj.codigo
          if (typeof errObj.mensaje === "string") mensaje = errObj.mensaje
          if (Array.isArray(errObj.detalle))
            detalles = errObj.detalle.map(String)
        } else {
          const cuerpoObj = cuerpo as Record<string, unknown>
          if (typeof cuerpoObj.codigo === "string") codigo = cuerpoObj.codigo
          if (typeof cuerpoObj.mensaje === "string") mensaje = cuerpoObj.mensaje
          if (Array.isArray(cuerpoObj.detalle))
            detalles = cuerpoObj.detalle.map(String)
        }
      }
    } else {
      const texto = await respuesta.text()
      if (texto.trim()) {
        mensaje = texto.trim()
      }
    }
  } catch {
    // Si no se puede parsear el cuerpo, conservamos el mensaje por defecto con el statusText.
  }

  if (respuesta.status === 401 && manejadorNoAutorizadoGlobal) {
    try {
      manejadorNoAutorizadoGlobal()
    } catch {
      // Ignorar errores del callback global para no enmascarar la excepción original.
    }
  }

  // Parsear cabeceras de control de velocidad (Retry-After y X-Retry-At) del servidor.
  let retryAfterSeg: number | undefined
  let retryAt: Date | undefined
  const retryAfterHeader = respuesta.headers.get("Retry-After")
  const retryAtHeader = respuesta.headers.get("X-Retry-At")
  if (retryAfterHeader) {
    const parsed = parseInt(retryAfterHeader, 10)
    if (!isNaN(parsed) && parsed > 0) retryAfterSeg = parsed
  }
  if (retryAtHeader) {
    const parsed = new Date(retryAtHeader)
    if (!isNaN(parsed.getTime())) retryAt = parsed
  }

  throw new ErrorApi(mensaje, {
    codigo,
    estado: respuesta.status,
    detalles,
    retryAfterSeg,
    retryAt,
  })
}

/**
 * Cliente HTTP base para la comunicación con la API.
 * Aplica credentials: 'include', cabeceras JSON por defecto y serialización de envelopes.
 */
export async function apiFetch<T>(
  ruta: string,
  opciones: OpcionesPeticion = {},
): Promise<T> {
  const { params, body, baseUrl, headers, formatoRespuesta, ...restoOpciones } =
    opciones
  const urlFinal = construirUrl(ruta, params, baseUrl)

  const cabeceras = new Headers(headers)
  if (!cabeceras.has("Accept")) cabeceras.set("Accept", "application/json")

  let cuerpoPeticion: BodyInit | null | undefined

  if (body !== undefined && body !== null) {
    if (
      typeof body === "string" ||
      body instanceof FormData ||
      body instanceof Blob ||
      body instanceof URLSearchParams
    ) {
      cuerpoPeticion = body as BodyInit
    } else {
      if (!cabeceras.has("Content-Type")) {
        cabeceras.set("Content-Type", "application/json; charset=utf-8")
      }
      cuerpoPeticion = JSON.stringify(body)
    }
  }

  let respuesta: Response
  try {
    respuesta = await fetch(urlFinal, {
      ...restoOpciones,
      credentials: restoOpciones.credentials ?? "include",
      headers: cabeceras,
      body: cuerpoPeticion,
    })
  } catch (error) {
    if (error instanceof Error && error.name === "AbortError") {
      throw error
    }
    throw new ErrorApi(
      "No se pudo conectar con el servidor. Verifique su conexión de red.",
      {
        codigo: "error_red",
        estado: 0,
        causa: error,
      },
    )
  }

  if (!respuesta.ok) {
    return await procesarErrorRespuesta(respuesta)
  }

  if (respuesta.status === 204) {
    return undefined as unknown as T
  }

  if (formatoRespuesta === "blob") {
    return (await respuesta.blob()) as T
  }

  const tipoContenido = respuesta.headers.get("content-type") ?? ""
  if (tipoContenido.includes("application/json")) {
    return (await respuesta.json()) as T
  }

  return (await respuesta.text()) as unknown as T
}

/** Realiza una petición GET. */
export async function apiGet<T>(
  ruta: string,
  opciones?: OpcionesPeticion,
): Promise<T> {
  return apiFetch<T>(ruta, { ...opciones, method: "GET" })
}

/** Descarga binaria con las mismas reglas de sesión, cancelación y errores que JSON. */
export function apiGetBlob(
  ruta: string,
  opciones?: OpcionesPeticion,
): Promise<Blob> {
  return apiFetch<Blob>(ruta, {
    ...opciones,
    method: "GET",
    formatoRespuesta: "blob",
  })
}

/** Realiza una petición POST. */
export async function apiPost<T>(
  ruta: string,
  body?: unknown,
  opciones?: OpcionesPeticion,
): Promise<T> {
  return apiFetch<T>(ruta, { ...opciones, method: "POST", body })
}

/** Realiza una petición PUT. */
export async function apiPut<T>(
  ruta: string,
  body?: unknown,
  opciones?: OpcionesPeticion,
): Promise<T> {
  return apiFetch<T>(ruta, { ...opciones, method: "PUT", body })
}

/** Realiza una petición PATCH. */
export async function apiPatch<T>(
  ruta: string,
  body?: unknown,
  opciones?: OpcionesPeticion,
): Promise<T> {
  return apiFetch<T>(ruta, { ...opciones, method: "PATCH", body })
}

/** Realiza una petición DELETE. */
export async function apiDelete<T>(
  ruta: string,
  opciones?: OpcionesPeticion,
): Promise<T> {
  return apiFetch<T>(ruta, { ...opciones, method: "DELETE" })
}
