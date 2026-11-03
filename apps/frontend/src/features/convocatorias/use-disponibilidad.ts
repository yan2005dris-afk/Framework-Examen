import { useEffect, useState } from "react"
import { useConvocatoriaPublica } from "./queries"
import { estadoRecepcion } from "./modelo"
import { useAhora } from "./use-ahora"

/** Comprueba el período de origen. Una consulta fallida o antigua no mantiene permisos de edición. */
export function useDisponibilidadEF12(id?: string | null) {
  const consulta = useConvocatoriaPublica(id, Boolean(id))
  const ahora = useAhora()
  const [enLinea, setEnLinea] = useState(
    () => typeof navigator === "undefined" || navigator.onLine !== false,
  )
  useEffect(() => {
    const actualizar = () => setEnLinea(navigator.onLine)
    window.addEventListener("online", actualizar)
    window.addEventListener("offline", actualizar)
    return () => {
      window.removeEventListener("online", actualizar)
      window.removeEventListener("offline", actualizar)
    }
  }, [])
  const confirmada =
    Boolean(id) &&
    !consulta.isError &&
    enLinea &&
    ahora - consulta.dataUpdatedAt < 45_000
  const estado = estadoRecepcion(consulta.data, ahora, confirmada)
  return { ...consulta, estadoRecepcion: estado, abierta: estado === "abierta" }
}
