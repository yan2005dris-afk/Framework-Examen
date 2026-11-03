import { expect, test } from "@playwright/test"

import { generarCedulaValida, generarCorreo } from "./fixtures/cedulas"
import { MAILPIT_API } from "./fixtures/datos"
import { leerOTP, listarMensajes } from "./fixtures/sesion"

/**
 * DEBUG: Prueba que crea usuario mediante registro y verifica login redirect
 */

test.describe("DEBUG - Login Redirect", () => {
  test("Registrar usuario y verificar que login redirige", async ({ page }) => {
    // Abrir navegador con logs de consola capturados
    const consoleLogs: string[] = []
    page.on("console", (msg) => {
      consoleLogs.push(`[${msg.type()}] ${msg.text()}`)
    })

    // Generar datos únicos
    const cedula = generarCedulaValida()
    const correo = generarCorreo("e2e-test")
    const password = "TestPassword.2026*"
    const nombres = "Test User"
    const apellidos = "E2E"

    console.log("\n========== FASE 1: REGISTRO ==========")
    console.log("🔵 Cedula:", cedula)
    console.log("🔵 Email:", correo)

    // Paso 1: Ir a registro
    await page.goto("/registro")
    expect(page).toHaveURL(/\/registro/)

    // Paso 2: Llenar cédula y email
    await page.fill("input#cedula", cedula)
    await page.fill("input#correo", correo)

    // Click en "Continuar con la verificación"
    const btnContinuar = page.locator(
      "button:has-text('Continuar con la verificación')",
    )
    await expect(btnContinuar).toBeVisible()
    await btnContinuar.click()
    console.log("🔵 Esperando OTP en Mailpit...")

    // Esperar a que llegue el email
    const otp = await leerOTP(correo)
    console.log("🟢 OTP recibido:", otp)

    // Paso 3: Ingresar OTP
    await page.fill("input#otp", otp)
    const btnVerificar = page.locator("button:has-text('Verificar código')")
    await expect(btnVerificar).toBeVisible()
    await btnVerificar.click()
    await page.waitForTimeout(500)

    // Paso 4: Ingresar credenciales
    console.log("🔵 Llenando datos de credenciales...")
    await page.fill("input#nombres", nombres)
    await page.fill("input#apellidos", apellidos)
    await page.fill("input#reg-password", password)
    await page.fill("input#confirm-password", password)

    // Submit registro
    const btnRegistrar = page.locator("button:has-text('Finalizar registro')")
    await expect(btnRegistrar).toBeVisible()
    await btnRegistrar.click()

    console.log("🔵 Esperando confirmación de registro...")
    // Esperar a que se complete
    await page.waitForTimeout(3000)

    console.log("\n========== FASE 2: LOGIN ==========")

    // Ir a login
    await page.goto("/login", { waitUntil: "domcontentloaded" })
    console.log("🔵 Navegué a /login")
    await page.waitForTimeout(500)

    // Llenar formulario de login
    await page.fill("input#cedula", cedula)
    await page.fill("input#password", password)
    console.log("🔵 Login credentials ingresadas:", cedula, "***")

    // Click en submit
    const btnSubmit = page.locator("button[type='submit']")
    expect(btnSubmit).toBeVisible()

    // Escuchar respuesta del API
    const loginPromise = page.waitForResponse(
      (resp) =>
        resp.url().includes("/auth/login") &&
        (resp.status() === 200 ||
          resp.status() === 401 ||
          resp.status() === 429),
    )

    await btnSubmit.click()
    console.log("🔵 Botón submit clickeado, esperando respuesta API...")

    const loginResponse = await loginPromise
    const status = loginResponse.status()
    const body: unknown = await loginResponse.json()

    console.log(`🔵 Respuesta API: ${status}`, body)

    if (status !== 200) {
      console.log("❌ Login falló con status:", status)
      console.log("Logs de consola del navegador:")
      consoleLogs.forEach((log) => console.log("  " + log))
      throw new Error(
        `Login falló con status ${status}: ${JSON.stringify(body)}`,
      )
    }

    console.log("🟢 Login fue exitoso (status 200)")

    // Esperar corto para ver si navega automáticamente
    await page.waitForTimeout(1000)

    // Verificar URL actual
    const urlActual = page.url()
    console.log("🔵 URL actual después de login:", urlActual)

    // Logs de consola del navegador
    console.log("\n=== LOGS DE CONSOLA DEL NAVEGADOR ===")
    consoleLogs.forEach((log) => console.log(log))
    console.log("====================================\n")

    // Intentar navegar manualmente a /postulacion para ver si el guard la bloquea
    console.log("🔵 Intentando navegar manualmente a /postulacion...")
    await page.goto("/postulacion")

    const urlFinal = page.url()
    console.log("🔵 URL final después de navegar a /postulacion:", urlFinal)

    if (urlFinal.includes("/postulacion")) {
      console.log("✅ Navegación a /postulacion funcionó!")
    } else if (urlFinal.includes("/login")) {
      console.log("❌ Guard redirigió al login - sesión no está activa")
    } else {
      console.log("❓ Ruta inesperada:", urlFinal)
    }
  })
})
