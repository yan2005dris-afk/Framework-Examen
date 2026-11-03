/**
 * Contrato de datos de las pruebas de extremo a extremo.
 *
 * El mismo conjunto que siembra `apps/backend/cmd/e2e-seed` se consume aquí
 * desde el navegador: las credenciales, la convocatoria y los expedientes son
 * los mismos, de modo que una prueba de interfaz y una de API hablan del
 * mismo postulante.
 *
 * Los identificadores se leen de `E2E_SEED_FILE` cuando el seed está
 * sembrado, y se usan los valores por defecto si el archivo no existe.
 */

/** Contraseña compartida por las cuentas sembradas. */
export const PASSWORD = process.env.E2E_PASSWORD || "TestPassword.2026*"

/** Rol de un postulante. */
export type Rol = "postulante" | "analista" | "administrador"

/** Cuenta de prueba de un rol. */
export interface Cuenta {
  /** Cedula es la clave de acceso de la cuenta. */
  cedula: string
  /** Correo es el correo declarado de la cuenta. */
  correo: string
  /** Rol es el rol que el servidor reconoce. */
  rol: Rol
}

/** Expediente de prueba de un escenario. */
export interface Expediente {
  /** Id es el identificador del expediente. */
  id: string
  /** Estado es el estado final del expediente. */
  estado: "draft" | "submitted" | "closed"
  /** Hash es el sello criptográfico del expediente enviado. */
  hash: string
}

/** Forma del archivo que publica el comando de siembra. */
interface ArchivoSeed {
  /** Convocatoria es el identificador de la convocatoria sembrada. */
  convocatoria: string
  /** ConvocatoriaCodigo es el código visible de la convocatoria. */
  convocatoriaCodigo: string
  /** Cuentas son las cuentas por cédula. */
  cuentas: Record<string, Cuenta>
  /** Expedientes son los expedientes por escenario. */
  expedientes: Record<string, Expediente>
}

/**
 * Base de la API. En desarrollo el navegador habla con el proxy de Vite, de
 * modo que un solo origen cubre la SPA y la API.
 */
export const API_BASE =
  process.env.E2E_API_BASE || "http://localhost:8080/api/v1"

/**
 * Base de Mailpit, donde el servidor deja los correos con los códigos.
 */
export const MAILPIT_API =
  process.env.E2E_MAILPIT_API || "http://localhost:8025/api/v1"

/**
 * Cuentas sembradas por defecto. Coinciden con las del backend para que la
 * suite funcione incluso antes de la primera siembra.
 */
const CUENTAS_POR_DEFECTO: Record<string, Cuenta> = {
  "2407612650": {
    cedula: "2407612650",
    correo: "e2e-administrador@biess.local",
    rol: "administrador",
  },
  "2402249987": {
    cedula: "2402249987",
    correo: "e2e-analista@biess.local",
    rol: "analista",
  },
  "2406425922": {
    cedula: "2406425922",
    correo: "e2e-aprobado@biess.local",
    rol: "postulante",
  },
  "2402740035": {
    cedula: "2402740035",
    correo: "e2e-minimo@biess.local",
    rol: "postulante",
  },
  "2403401355": {
    cedula: "2403401355",
    correo: "e2e-cerrado@biess.local",
    rol: "postulante",
  },
  "2408380695": {
    cedula: "2408380695",
    correo: "e2e-reconsideracion@biess.local",
    rol: "postulante",
  },
}

/**
 * Expedientes sembrados por defecto.
 */
const EXPEDIENTES_POR_DEFECTO: Record<string, Expediente> = {
  aprobado: { id: "", estado: "submitted", hash: "" },
  minimo: { id: "", estado: "draft", hash: "" },
  cerrado: { id: "", estado: "closed", hash: "" },
  reconsideracion: { id: "", estado: "submitted", hash: "" },
}

/**
 * Convocatoria sembrada por defecto.
 */
const CONVOCATORIA_POR_DEFECTO = "01a1954b-bf4e-75c0-a235-24242ce05127"

/**
 * Lee el archivo de siembra si existe.
 *
 * @returns {ArchivoSeed | null} El contenido del archivo o null si no existe.
 */
function leerArchivoSeed(): ArchivoSeed | null {
  const ruta = process.env.E2E_SEED_FILE
  if (!ruta) return null
  try {
    // eslint-disable-next-line @typescript-eslint/no-require-imports
    const { readFileSync } = require("node:fs") as typeof import("node:fs")
    return JSON.parse(readFileSync(ruta, "utf8")) as ArchivoSeed
  } catch {
    return null
  }
}

const semilla = leerArchivoSeed()

/**
 * Cuentas de prueba disponibles para la suite.
 */
export const CUENTAS: Record<string, Cuenta> =
  semilla?.cuentas ?? CUENTAS_POR_DEFECTO

/**
 * Expedientes de prueba disponibles para la suite.
 */
export const EXPEDIENTES: Record<string, Expediente> =
  semilla?.expedientes ?? EXPEDIENTES_POR_DEFECTO

/**
 * Identificador de la convocatoria sembrada.
 */
export const CONVOCATORIA_ID = semilla?.convocatoria ?? CONVOCATORIA_POR_DEFECTO

/**
 * Código visible de la convocatoria sembrada.
 */
export const CONVOCATORIA_CODIGO = semilla?.convocatoriaCodigo ?? "e2e-2026-01"

/**
 * Devuelve la cuenta del rol indicado.
 *
 * @param {Rol} rol - Rol buscado.
 * @returns {Cuenta} La cuenta del rol.
 */
export function cuentaPorRol(rol: Rol): Cuenta {
  for (const cuenta of Object.values(CUENTAS)) {
    if (cuenta.rol === rol) return cuenta
  }
  throw new Error(`no hay una cuenta sembrada con el rol ${rol}`)
}

/**
 * Devuelve la cuenta de la cédula indicada.
 *
 * @param {string} cedula - Cédula buscada.
 * @returns {Cuenta} La cuenta sembrada.
 */
export function cuentaPorCedula(cedula: string): Cuenta {
  const cuenta = CUENTAS[cedula]
  if (!cuenta)
    throw new Error(`no hay una cuenta sembrada con la cédula ${cedula}`)
  return cuenta
}

/**
 * Devuelve el expediente del escenario indicado.
 *
 * @param {string} escenario - Nombre del escenario.
 * @returns {Expediente} El expediente sembrado.
 */
export function expedientePorEscenario(escenario: string): Expediente {
  const expediente = EXPEDIENTES[escenario]
  if (!expediente)
    throw new Error(`no hay un expediente sembrado para ${escenario}`)
  return expediente
}
