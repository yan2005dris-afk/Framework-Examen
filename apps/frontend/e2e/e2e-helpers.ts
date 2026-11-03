import { expect, type Page } from "@playwright/test"

export const MAILPIT_API = "http://localhost:8025/api/v1"
export const API_BASE = "http://localhost:8080/api/v1"
export const DEFAULT_PASSWORD = process.env.E2E_PASSWORD || "TestPassword.2026*"

/**
 * Genera una cédula ecuatoriana válida por módulo 10 (persona natural, provincia 17).
 */
export function generarCedulaValida(): string {
  const prov = "17"
  const d3 = Math.floor(Math.random() * 6).toString()
  const mid = Math.floor(Math.random() * 1000000).toString().padStart(6, "0")
  const base = prov + d3 + mid
  const coef = [2, 1, 2, 1, 2, 1, 2, 1, 2]
  let sum = 0
  for (let i = 0; i < 9; i++) {
    let val = parseInt(base[i], 10) * coef[i]
    if (val >= 10) val -= 9
    sum += val
  }
  const ver = sum % 10 === 0 ? 0 : 10 - (sum % 10)
  return base + ver
}

/**
 * Obtiene el código OTP de 6 dígitos enviado a un correo en Mailpit.
 */
export async function esperarOTPdeMailpit(
  correo: string,
  subjectFilter?: string,
  maxIntentos = 10,
): Promise<string> {
  for (let i = 0; i < maxIntentos; i++) {
    try {
      const response = await fetch(`${MAILPIT_API}/messages`)
      const data = await response.json()
      const messages = Array.isArray(data) ? data : data.messages || []

      const message = messages
        .filter((m: any) => {
          const matchesEmail = m.To?.[0]?.Address === correo
          const matchesSubject = subjectFilter
            ? (m.Subject || "").toLowerCase().includes(subjectFilter.toLowerCase())
            : true
          return matchesEmail && matchesSubject
        })
        .sort(
          (a: any, b: any) =>
            new Date(b.Created).getTime() - new Date(a.Created).getTime(),
        )[0]

      if (message) {
        const textToSearch = message.Snippet || message.Text || message.HTML || ""
        const match = textToSearch.match(/(\d{6})/)
        if (match) return match[1]

        const detailResp = await fetch(`${MAILPIT_API}/message/${message.ID}`)
        const detail = await detailResp.json()
        const detailText = detail.Text || detail.HTML || ""
        const detailMatch = detailText.match(/(\d{6})/)
        if (detailMatch) return detailMatch[1]
      }
    } catch {
      // Ignorar error transitorio de red
    }
    await new Promise((r) => setTimeout(r, 1000))
  }
  throw new Error(`No se encontró OTP en Mailpit para ${correo} (filtro: ${subjectFilter || "ninguno"})`)
}

/**
 * Registra un nuevo postulante y verifica su correo mediante OTP.
 */
export async function registrarNuevoPostulante(
  page: Page,
  datos?: { cedula?: string; correo?: string; password?: string; nombres?: string; apellidos?: string },
): Promise<{ cedula: string; correo: string; password: string; nombres: string; apellidos: string }> {
  const cedula = datos?.cedula || generarCedulaValida()
  const correo = datos?.correo || `postulante-${cedula}@biess-test.local`
  const password = datos?.password || DEFAULT_PASSWORD
  const nombres = datos?.nombres || "Juan Carlos"
  const apellidos = datos?.apellidos || "Pérez Gómez"

  await page.goto("/registro")
  await page.fill("input#cedula", cedula)
  await page.fill("input#correo", correo)
  await page.locator("button:has-text('Continuar con la verificación')").click()

  const otp = await esperarOTPdeMailpit(correo, "código")
  await page.fill("input#otp", otp)
  await page.locator("button:has-text('Verificar código')").click()

  await page.fill("input#nombres", nombres)
  await page.fill("input#apellidos", apellidos)
  await page.fill("input#reg-password", password)
  await page.fill("input#confirm-password", password)
  await page.locator("button:has-text('Finalizar registro')").click()

  await expect(page).toHaveURL(/\/login/, { timeout: 10000 })
  return { cedula, correo, password, nombres, apellidos }
}

/**
 * Inicia sesión con credenciales y asegura redirección al área solicitada.
 */
export async function iniciarSesion(
  page: Page,
  cedula: string,
  password = DEFAULT_PASSWORD,
): Promise<void> {
  await page.goto("/login")
  await page.fill("input#cedula", cedula)
  await page.fill("input#password", password)
  await page.locator("button[type='submit']").click()
  await page.waitForURL((url) => !url.pathname.includes("/login"), { timeout: 15000 })
}

/**
 * Selecciona una opción en un Radix/shadcn SelectTrigger.
 */
export async function seleccionarOpcionShadcn(
  page: Page,
  triggerSelector: string,
  opcionTexto: string | RegExp,
): Promise<void> {
  const trigger = page.locator(triggerSelector)
  await trigger.click()
  const opcion = page.getByRole("option", { name: opcionTexto })
  await opcion.waitFor({ state: "visible", timeout: 5000 })
  await opcion.click()
}
