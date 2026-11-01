import { useEffect, useState } from "react"

/** Actualiza los límites visibles también en pestañas que permanecieron abiertas. No autoriza operaciones. */
export function useAhora() {
  const [ahora, setAhora] = useState(Date.now)
  useEffect(() => {
    const actualizar = () => setAhora(Date.now())
    const timer = window.setInterval(actualizar, 1000)
    window.addEventListener("focus", actualizar)
    document.addEventListener("visibilitychange", actualizar)
    return () => {
      window.clearInterval(timer)
      window.removeEventListener("focus", actualizar)
      document.removeEventListener("visibilitychange", actualizar)
    }
  }, [])
  return ahora
}
