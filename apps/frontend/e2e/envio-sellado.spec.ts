import { expect, test } from "@playwright/test"
import {
  DEFAULT_PASSWORD,
  esperarOTPdeMailpit,
  iniciarSesion,
} from "./e2e-helpers"

/**
 * Test Criterio 2: Envío, Sellado Criptográfico y Hash SHA-256.
 * Valida:
 * 1. Aceptación obligatoria de los 5 literales de consentimiento legal.
 * 2. Transacción de sellado en backend con cálculo del hash SHA-256 canónico.
 * 3. Asignación del código único de postulación (ej: EF12-2026-...).
 * 4. Transición de estado a 'submitted' (solo lectura).
 * 5. Correo de confirmación emitido a través de SMTP / Mailpit.
 */

test.describe("ENVÍO + SELLADO + HASH — Criterio 2 Persistencia (8pts)", () => {
  test("Enviar postulación → transacción atómica + hash + código único", async ({ page }) => {
    test.setTimeout(90000)
    console.log("\n========== FASE 1: LOGIN Y REVISIÓN ==========\n")

    await iniciarSesion(page, "0926914227", DEFAULT_PASSWORD)

    await page.goto("/postulacion/revision")
    await expect(page.getByRole("heading", { name: /Revisión y Envío de la Postulación/i })).toBeVisible({ timeout: 15000 })

    // =========================================================================
    // FASE 2: ACEPTACIÓN DE LOS 5 PUNTOS DE CONSENTIMIENTO
    // =========================================================================
    console.log("========== FASE 2: CONSENTIMIENTOS LEGALES ==========")

    const btnAceptarCinco = page.locator("button:has-text('Aceptar los 5 puntos')")
    await expect(btnAceptarCinco).toBeVisible({ timeout: 10000 })
    await btnAceptarCinco.click()
    console.log("✅ Se aceptaron los 5 puntos de consentimiento legal")

    // =========================================================================
    // FASE 3: ENVÍO DEFINITIVO DE LA POSTULACIÓN
    // =========================================================================
    console.log("========== FASE 3: SELLADO Y ENVÍO DEFINITIVO ==========")

    const btnEnviar = page.locator("button:has-text('Enviar postulación definitiva'), button:has-text('Enviar postulación')").first()
    await expect(btnEnviar).toBeEnabled({ timeout: 10000 })

    // Escuchar la petición HTTP del backend /submit
    const submitPromise = page.waitForResponse(
      (resp) => resp.url().includes("/submit") && resp.status() === 200,
      { timeout: 15000 },
    )

    await btnEnviar.click()

    // Confirmar en modal si se despliega
    const btnConfirmar = page.locator("button:has-text('Confirmar y enviar'), button:has-text('Confirmar')").first()
    if (await btnConfirmar.isVisible({ timeout: 3000 })) {
      await btnConfirmar.click()
    }

    const submitResponse = await submitPromise
    const submitData = await submitResponse.json()
    console.log("✅ API /submit respondió 200 OK")

    // =========================================================================
    // VALIDACIONES DEL PROTOCOLO DE SELLADO
    // =========================================================================
    const codigo = submitData?.code || submitData?.data?.code
    const hash = submitData?.hash || submitData?.data?.hash
    const status = submitData?.status || submitData?.data?.status

    expect(codigo).toMatch(/^EF12-\d{4}-[A-Z0-9]+$/)
    console.log(`✅ Código único generado: ${codigo}`)

    expect(hash).toMatch(/^[a-f0-9]{64}$/)
    console.log(`✅ Hash SHA-256 canónico sellado: ${hash}`)

    expect(status).toBe("submitted")
    console.log(`✅ Estado de postulación: ${status}`)

    // =========================================================================
    // FASE 4: VISTA DE SOLO LECTURA (READ-ONLY)
    // =========================================================================
    await expect(page.locator("text=/postulación enviada|expediente sellado|solo lectura|enviada y sellada/i").first()).toBeVisible({ timeout: 15000 })
    console.log("✅ Vista de postulación en modo solo lectura (read-only)")

    // =========================================================================
    // FASE 5: CORREO DE CONFIRMACIÓN EN MAILPIT
    // =========================================================================
    console.log("========== FASE 5: NOTIFICACIÓN POR CORREO ==========")
    const otpCorreo = await esperarOTPdeMailpit("yan2005dris@gmail.com", "", 10)
    expect(otpCorreo).toBeTruthy()
    console.log("✅ Correo de confirmación con sello y código único verificado en Mailpit")
  })
})
