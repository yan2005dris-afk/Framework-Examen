import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react"

import { configurarManejadorNoAutorizado } from "@/lib/api"
import { esErrorApi } from "@/lib/api-error"
import {
  cerrarSesion,
  obtenerSesion,
  type PostulanteDto,
} from "@/lib/clientes/auth"

/**
 * Estado de la sesión mientras el portal consulta a la API.
 * `cargando` evita que el guard muestre el login antes de saber si hay sesión.
 */
export type EstadoSesion = "cargando" | "autenticado" | "anonimo"

/**
 * Valor que comparten el proveedor de sesión y sus consumidores.
 */
export type ValorSesion = {
  /** Postulante autenticado; `null` cuando no hay sesión. */
  postulante: PostulanteDto | null
  /** Estado del ciclo de vida de la consulta de sesión. */
  estado: EstadoSesion
  /** Vuelve a consultar `/auth/session` y actualiza el estado. */
  refrescar: () => Promise<void>
  /** Cierra la sesión en el servidor y limpia el estado local. */
  cerrar: () => Promise<void>
  /** Registra la sesión que acaba de iniciar el formulario de login (FT3). */
  iniciarSesionCon: (postulante: PostulanteDto) => void
}

/**
 * Contexto de sesión. El hook de consumo vive en este mismo módulo porque
 * `react-refresh/only-export-components` lo permite explícitamente en los
 * archivos `*-context.tsx` (ver `eslint.config.js`).
 */
const ContextoSesion = createContext<ValorSesion | null>(null)

/**
 * Devuelve la sesión activa. Solo puede usarse dentro de `<ProveedorSesion>`.
 */
export function useSesion(): ValorSesion {
  const valor = useContext(ContextoSesion)
  if (!valor) {
    throw new Error("useSesion debe usarse dentro de <ProveedorSesion>.")
  }
  return valor
}

type ProveedorSesionProps = {
  children: ReactNode
}

/**
 * Proveedor del estado de sesión del portal.
 *
 * Consulta `/auth/session` una sola vez al montar, con un `AbortSignal` que se
 * cancela al desmontar. Un `401` (tanto en la consulta inicial como en
 * cualquier petición posterior) deja el estado en `anonimo`; el guard que
 * envuelve la ruta es quien redirige al login conservando el destino, así que
 * no hay recarga de página ni doble navegación.
 */
export function ProveedorSesion({ children }: ProveedorSesionProps) {
  const [postulante, setPostulante] = useState<PostulanteDto | null>(null)
  const [estado, setEstado] = useState<EstadoSesion>("cargando")

  const aplicarSesion = useCallback((nuevoPostulante: PostulanteDto) => {
    setPostulante(nuevoPostulante)
    setEstado("autenticado")
  }, [])

  const aplicarAnonimo = useCallback((error?: unknown) => {
    if (error && !(esErrorApi(error) && error.esNoAutorizado())) {
      // Una caída de red no debe romper la pantalla: se trata como sesión
      // ausente para que el guard ofrezca el login.
      console.warn("No se pudo recuperar la sesión del postulante", error)
    }
    setPostulante(null)
    setEstado("anonimo")
  }, [])

  useEffect(() => {
    const controlador = new AbortController()

    async function inicializar() {
      try {
        const respuesta = await obtenerSesion(controlador.signal)
        if (!controlador.signal.aborted) {
          aplicarSesion(respuesta.postulante)
        }
      } catch (error: unknown) {
        if (!controlador.signal.aborted) {
          aplicarAnonimo(error)
        }
      }
    }

    void inicializar()

    return () => controlador.abort()
  }, [aplicarSesion, aplicarAnonimo])

  const limpiarSesion = useCallback(() => {
    setPostulante(null)
    setEstado("anonimo")
  }, [])

  useEffect(() => {
    configurarManejadorNoAutorizado(limpiarSesion)
    return () => configurarManejadorNoAutorizado(null)
  }, [limpiarSesion])

  const refrescar = useCallback(async () => {
    setEstado("cargando")
    try {
      const respuesta = await obtenerSesion()
      aplicarSesion(respuesta.postulante)
    } catch (error) {
      aplicarAnonimo(error)
    }
  }, [aplicarSesion, aplicarAnonimo])

  const cerrar = useCallback(async () => {
    try {
      await cerrarSesion()
    } catch (error) {
      // El cierre local manda: aunque el servidor no responda, la sesión del
      // navegador debe quedar limpia.
      console.warn("No se pudo revocar la sesión en el servidor", error)
    } finally {
      limpiarSesion()
    }
  }, [limpiarSesion])

  const iniciarSesionCon = useCallback((nuevoPostulante: PostulanteDto) => {
    setPostulante(nuevoPostulante)
    setEstado("autenticado")
  }, [])

  const valor = useMemo<ValorSesion>(
    () => ({
      postulante,
      estado,
      refrescar,
      cerrar,
      iniciarSesionCon,
    }),
    [postulante, estado, refrescar, cerrar, iniciarSesionCon],
  )

  return (
    <ContextoSesion.Provider value={valor}>{children}</ContextoSesion.Provider>
  )
}
