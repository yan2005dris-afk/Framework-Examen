import { execFileSync } from "node:child_process"
import { existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs"
import { dirname, resolve } from "node:path"

/**
 * Estado global de la corrida: el archivo con el conjunto sembrado.
 */
const RUTA_ESTADO = resolve(process.cwd(), "e2e", ".estado", "seed.json")

/**
 * Siembra el conjunto de datos de extremo a extremo antes de la corrida.
 *
 * La siembra es idempotente, así que puede ejecutarse en cada corrida sin
 * duplicar cuentas ni expedientes. El comando vive en el backend porque
 * el seed debe hablar el mismo idioma que el dominio: si el backend no
 * acepta un dato, la suite de interfaz tampoco debe poder sembrarlo.
 *
 * @returns {Promise<void>} Se resuelve cuando el conjunto está sembrado.
 */
export default async function globalSetup(): Promise<void> {
  const url =
    process.env.DATABASE_URL ||
    "postgres://postgres@localhost:5432/biess?sslmode=disable"
  const backend = resolve(process.cwd(), "..", "backend")

  try {
    const salida = execFileSync("go", ["run", "./cmd/e2e-seed"], {
      cwd: backend,
      encoding: "utf8",
      env: { ...process.env, DATABASE_URL: url },
      stdio: ["ignore", "pipe", "pipe"],
    })
    const estado = JSON.parse(salida) as Record<string, unknown>
    mkdirSync(dirname(RUTA_ESTADO), { recursive: true })
    writeFileSync(RUTA_ESTADO, JSON.stringify(estado, null, 2), "utf8")
    process.env.E2E_SEED_FILE = RUTA_ESTADO
    await reiniciarAPI(backend)
  } catch (error) {
    // Si el seed no puede aplicarse, la suite no debe fingir que los datos
    // existen: se avisa y se continúa con los valores por defecto, que la
    // propia suite de API verifica.
    const detalle = error instanceof Error ? error.message : String(error)
    console.warn(`[e2e] no se pudo ejecutar el seed: ${detalle}`)
    console.warn("[e2e] se continúa con el conjunto de cuentas por defecto")
  }
}

/**
 * Espera async sin busy-wait.
 */
function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

/**
 * Reinicia la API para que cargue la simulación de elegibilidad.
 *
 * El servidor lee esa simulación al arrancar, no en cada petición: si el
 * seed la publica con cuentas nuevas, la API tiene que volver a leerla o
 * seguirá respondiendo `503 elegibilidad_no_disponible`.
 *
 * @param {string} backend - Ruta del módulo backend.
 */
async function reiniciarAPI(backend: string): Promise<void> {
  if (process.env.E2E_RESTART_API === "0") return
  try {
    execFileSync("docker", ["compose", "restart", "api"], {
      cwd: resolve(backend, ".."),
      stdio: "ignore",
    })
    await esperarAPI()
  } catch (error) {
    const detalle = error instanceof Error ? error.message : String(error)
    console.warn(`[e2e] no se pudo reiniciar la API: ${detalle}`)
  }
}

/**
 * Espera a que la API vuelva a responder.
 */
async function esperarAPI(): Promise<void> {
  const url = process.env.E2E_API_URL || "http://localhost:8080/readyz"
  for (let intento = 0; intento < 60; intento += 1) {
    try {
      const respuesta = await fetch(url)
      if (respuesta.status === 200) return
    } catch {
      // La API todavía no escucha: se reintenta.
    }
    await sleep(500)
  }
  console.warn("[e2e] la API no respondió a tiempo tras el reinicio")
}

/**
 * Lee el conjunto sembrado, si existe.
 *
 * @returns {Record<string, unknown> | null} El conjunto o null.
 */
export function leerEstado(): Record<string, unknown> | null {
  if (!existsSync(RUTA_ESTADO)) return null
  return JSON.parse(readFileSync(RUTA_ESTADO, "utf8")) as Record<
    string,
    unknown
  >
}
