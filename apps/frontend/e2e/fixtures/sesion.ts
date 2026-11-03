import { expect, type APIRequestContext, type Page } from "@playwright/test"

import { API_BASE, MAILPIT_API, PASSWORD } from "./datos"

/**
 * Inicia sesión en la API y devuelve las cookies de sesión.
 *
 * La sesión viaja en una cookie HttpOnly: la suite la reusa tal como lo
 * hace el navegador, en lugar de inventar un token que el servidor nunca
 * devuelve.
 *
 * @param {APIRequestContext} request - Contexto de peticiones de Playwright.
 * @param {string} cedula - Cédula de la cuenta.
 * @param {string} password - Contraseña de la cuenta.
 * @returns {Promise<void>} Se resuelve cuando la sesión está abierta.
 */
export async function iniciarSesion(
  request: APIRequestContext,
  cedula: string,
  password: string = PASSWORD,
): Promise<void> {
  const respuesta = await request.post(`${API_BASE}/auth/login`, {
    data: { cedula, password },
    headers: { Origin: "http://localhost:5173" },
  })
  expect(respuesta.status(), await cuerpoDe(respuesta)).toBe(200)
}

/**
 * Inicia sesión en el navegador y espera a que la aplicación reaccione.
 *
 * @param {Page} page - Página del navegador.
 * @param {string} cedula - Cédula de la cuenta.
 * @param {string} password - Contraseña de la cuenta.
 * @returns {Promise<void>} Se resuelve cuando el formulario se envía.
 */
export async function entrarComo(
  page: Page,
  cedula: string,
  password: string = PASSWORD,
): Promise<void> {
  await page.goto("/login")
  await page.locator("input#cedula").fill(cedula)
  await page.locator("input#password").fill(password)
  await page.locator("button[type='submit']").click()
}

/**
 * Lee el código de un solo uso que el servidor envió a un correo.
 *
 * @param {string} correo - Destinatario del correo.
 * @param {number} esperaMs - Espera máxima para que llegue el mensaje.
 * @returns {Promise<string>} El código de seis dígitos.
 */
export async function leerOTP(
  correo: string,
  esperaMs = 15000,
): Promise<string> {
  const limite = Date.now() + esperaMs
  while (Date.now() < limite) {
    const codigo = await buscarOTP(correo)
    if (codigo) return codigo
    await new Promise((resolver) => setTimeout(resolver, 500))
  }
  throw new Error(`el servidor no envió el código por correo a ${correo}`)
}

/**
 * Busca el código en un mensaje de Mailpit.
 *
 * El listado de Mailpit no trae el cuerpo del correo, sólo un resumen: hay
 * que abrir cada mensaje candidato para leer su texto.
 *
 * @param {string} correo - Destinatario buscado.
 * @returns {Promise<string | null>} El código o null si todavía no llegó.
 */
async function buscarOTP(correo: string): Promise<string | null> {
  const mensajes = await listarMensajes()
  const candidatos = mensajes
    .filter((m) => m.To?.some((destinatario) => destinatario.Address === correo))
    .sort((a, b) => instante(b.Created) - instante(a.Created))

  for (const candidato of candidatos) {
    if (!candidato.ID) continue
    const cuerpo = await leerCuerpo(candidato.ID)
    if (!cuerpo) continue
    const coincidencia = cuerpo.match(/\b\d{6}\b/)
    if (coincidencia) return coincidencia[0]
  }
  return null
}

/**
 * Lee el cuerpo de un mensaje de Mailpit.
 *
 * @param {string} id - Identificador del mensaje.
 * @returns {Promise<string | null>} El texto del mensaje o null si no se pudo leer.
 */
async function leerCuerpo(id: string): Promise<string | null> {
  try {
    const respuesta = await fetch(`${MAILPIT_API}/message/${id}`)
    if (!respuesta.ok) return null
    const cuerpo = (await respuesta.json()) as MensajeMailpit
    return cuerpo.Text || cuerpo.HTML || null
  } catch {
    return null
  }
}

/**
 * ListadoMailpit es la forma del listado de mensajes que expone Mailpit:
 * un objeto con los contadores y el arreglo de mensajes.
 */
export interface ListadoMailpit {
  /** Total es el número de mensajes del buzón. */
  total?: number
  /** Messages son los mensajes capturados. */
  messages?: MensajeMailpit[]
}

/**
 * Lista los mensajes capturados por Mailpit.
 *
 * Mailpit devuelve un objeto con los contadores y el arreglo de mensajes; el
 * arreglo es el que interesan a las pruebas.
 *
 * @returns {Promise<MensajeMailpit[]>} Los mensajes de la bandeja.
 */
export async function listarMensajes(): Promise<MensajeMailpit[]> {
  const respuesta = await fetch(`${MAILPIT_API}/messages`)
  if (!respuesta.ok) return []
  const cuerpo = (await respuesta.json()) as ListadoMailpit
  return cuerpo.messages ?? []
}

/**
 * Convierte una fecha de Mailpit en milisegundos. Un valor ausente o
 * ilegible ordena al final, sin romper la búsqueda del código más reciente.
 *
 * @param {string | undefined} valor - Fecha en texto.
 * @returns {number} Milisegundos desde epoch.
 */
function instante(valor: string | undefined): number {
  if (!valor) return 0
  const fecha = new Date(valor)
  const milisegundos = fecha.getTime()
  return Number.isNaN(milisegundos) ? 0 : milisegundos
}

/**
 * Mensaje de Mailpit tal como lo expone su API.
 */
export interface MensajeMailpit {
  /** ID es el identificador del mensaje. */
  ID?: string
  /** Created es la fecha de creación. */
  Created?: string
  /** To son los destinatarios. */
  To?: { Address?: string }[]
  /** Text es el cuerpo en texto plano. */
  Text?: string
  /** HTML es el cuerpo en HTML. */
  HTML?: string
}

/**
 * Lee el cuerpo de una respuesta como texto para poder explicar un fallo.
 *
 * @param {import('@playwright/test').APIResponse} respuesta - Respuesta leída.
 * @returns {Promise<string>} El cuerpo o una descripción del estado.
 */
async function cuerpoDe(respuesta: {
  status(): number
  text(): Promise<string>
}): Promise<string> {
  try {
    return await respuesta.text()
  } catch {
    return `estado ${respuesta.status()}`
  }
}

/**
 * Construye un PDF mínimo válido para las pruebas de adjuntos.
 *
 * @param {string} contenido - Texto que se imprime en el PDF.
 * @returns {Buffer} El PDF listo para subir.
 */
export function crearPDF(contenido: string): Buffer {
  const cabecera = "%PDF-1.4\n"
  const cuerpo = `
1 0 obj
<< /Type /Catalog /Pages 2 0 R >>
endobj
2 0 obj
<< /Type /Pages /Kids [3 0 R ] /Count 1 >>
endobj
3 0 obj
<< /Type /Page /Parent 2 0 R /Resources << /Font << /F1 4 0 R >> >> /MediaBox [0 0 612 792] /Contents 5 0 R >>
endobj
4 0 obj
<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>
endobj
5 0 obj
<< >>
stream
BT /F1 12 Tf 100 750 Td (${contenido}) Tj ET
endstream
endobj
trailer
<< /Size 6 /Root 1 0 R >>
%%EOF
`
  return Buffer.from(cabecera + cuerpo, "utf8")
}
