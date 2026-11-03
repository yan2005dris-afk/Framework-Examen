/**
 * Genera un identificador único UUIDv4 en formato canónico.
 * Utiliza la API estándar Web Crypto del navegador o entorno Node.
 */
export function generarUuid(): string {
  if (
    typeof crypto !== "undefined" &&
    typeof crypto.randomUUID === "function"
  ) {
    return crypto.randomUUID()
  }

  // Fallback seguro en caso de entornos sin crypto.randomUUID
  return "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx".replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0
    const v = c === "x" ? r : (r & 0x3) | 0x8
    return v.toString(16)
  })
}

/**
 * Valida que un valor tenga forma de UUID canónico.
 *
 * Es una regla de formato, no de negocio: vive en `lib/` para que los
 * adaptadores HTTP y los modelos puros compartan la misma comprobación sin
 * que ninguno dependa del otro.
 *
 * @param valor - Cadena a verificar.
 * @returns `true` si el valor es un UUID con la forma `8-4-4-4-12`.
 */
export function esUuid(valor: string): boolean {
  return /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(
    valor,
  )
}
