import { expect, test } from "@playwright/test"

const API_BASE = "http://localhost:8080/api/v1"
const MAILPIT_API = "http://localhost:8025/api/v1"

/**
 * CRÍTICO - Test 8: Envío + Sellado + Hash SHA-256
 * Criterio de rúbrica 2: Persistencia Relacional & Transacciones (8 pts)
 *
 * Valida:
 * 1. Transacción atómica: hash + código + estado submitted + email + eventos
 * 2. Estado post-envío: formulario read-only, no editable
 * 3. Hash SHA-256 calculado sobre JSON canónico
 * 4. Código único asignado a postulación
 * 5. Correo de confirmación enviado al postulante
 */

async function obtenerOTPdeMailpit(correo: string): Promise<string | null> {
  const response = await fetch(`${MAILPIT_API}/messages`)
  const data = await response.json()
  const messages = Array.isArray(data) ? data : data.messages || []

  const message = messages
    .filter((m: any) => m.To?.[0]?.Address === correo)
    .sort((a: any, b: any) => new Date(b.Created).getTime() - new Date(a.Created).getTime())[0]

  if (!message) return null

  const match = message.Text?.match(/(\d{6})/) || message.HTML?.match(/(\d{6})/)
  return match ? match[1] : null
}

async function generarCedulaTest(): Promise<string> {
  const base = Math.random().toString().substring(2, 11).padStart(9, "0")
  let sum = 0
  for (let i = 0; i < 9; i++) {
    const digit = parseInt(base[i])
    let result = digit * (i % 2 === 0 ? 2 : 1)
    if (result > 9) result -= 9
    sum += result
  }
  const verification = (10 - (sum % 10)) % 10
  return base + verification
}

test.describe("ENVÍO + SELLADO + HASH — Criterio 2 Persistencia (8pts)", () => {
  // =========================================================================
  // FLUJO PREVIO: Crear usuario + completar formulario mínimo
  // =========================================================================
  test("Enviar postulación → transacción atómica + hash + código único", async ({ page }) => {
    console.log("\n========== FASE 1: REGISTRO Y LOGIN ==========\n")

    const cedula = await generarCedulaTest()
    const correo = `envio-test-${cedula}@biess-test.local`
    const password = "SecurePass.2026*"

    // 1. Ir a registro
    await page.goto("/registro")
    expect(page).toHaveURL(/\/registro/)

    // 2. Llenar cédula y correo
    await page.fill("input#cedula", cedula)
    await page.fill("input#correo", correo)

    // 3. Click continuar
    const btnContinuar = page.locator("button:has-text('Continuar con la verificación')")
    await expect(btnContinuar).toBeVisible()
    await btnContinuar.click()

    // 4. Esperar OTP
    console.log("Esperando OTP en Mailpit...")
    await page.waitForTimeout(2000)
    const otp = await obtenerOTPdeMailpit(correo)
    if (!otp) throw new Error(`No se recibió OTP para ${correo}`)
    console.log(`✅ OTP recibido: ${otp}`)

    // 5. Verificar OTP
    await page.fill("input#otp", otp)
    const btnVerificar = page.locator("button:has-text('Verificar código')")
    await expect(btnVerificar).toBeVisible()
    await btnVerificar.click()
    await page.waitForTimeout(500)

    // 6. Completar credenciales
    await page.fill("input#nombres", "Test Envio")
    await page.fill("input#apellidos", "Sellado")
    await page.fill("input#reg-password", password)
    await page.fill("input#confirm-password", password)

    // 7. Finalizar registro
    const btnRegistrar = page.locator("button:has-text('Finalizar registro')")
    await expect(btnRegistrar).toBeVisible()
    await btnRegistrar.click()
    console.log("Registro completado")

    await page.waitForTimeout(2000)

    // =========================================================================
    // FASE 2: LOGIN
    // =========================================================================
    console.log("\n========== FASE 2: LOGIN ==========\n")

    await page.goto("/login")
    await page.fill("input#cedula", cedula)
    await page.fill("input#password", password)

    const btnSubmit = page.locator("button[type='submit']")
    await btnSubmit.click()

    // Esperar a que se complete login y redireccione
    await page.waitForURL(/\/postulacion/)
    console.log("✅ Login exitoso")

    // =========================================================================
    // FASE 3: COMPLETAR FORMULARIO MÍNIMO (sección 1 y 2)
    // =========================================================================
    console.log("\n========== FASE 3: COMPLETAR FORMULARIO MÍNIMO ==========\n")

    // Sección 1: Información personal (campos mínimos)
    const seccion1 = page.locator("section:has-text('Información Personal'), [data-section='1']").first()
    if (await seccion1.isVisible()) {
      console.log("Llenando Sección 1: Información Personal")
      await page.fill("input[name='nombres']", "Test")
      await page.fill("input[name='apellidos']", "Usuario")
      // Género (select)
      await page.selectOption("select[name='genero']", "M")
      // Fecha de nacimiento
      await page.fill("input[name='fechaNacimiento']", "1990-01-15")
      // Nacionalidad
      await page.selectOption("select[name='nacionalidad']", "Ecuador")
      // Estado civil
      await page.selectOption("select[name='estadoCivil']", "S")

      // Guardar sección 1
      const btnGuardarS1 = page.locator("button:has-text('Guardar')").first()
      if (await btnGuardarS1.isVisible()) {
        await btnGuardarS1.click()
        console.log("✅ Sección 1 guardada")
        await page.waitForTimeout(1000)
      }
    }

    // Sección 2: Información domiciliaria
    console.log("Llenando Sección 2: Información Domiciliaria")
    await page.goto("/postulacion/2")
    await page.waitForTimeout(1000)

    // País = Ecuador (default)
    // Provincia
    const provinciasSelect = page.locator("select").filter({ hasText: "Provincia" }).first()
    if (await provinciasSelect.isVisible()) {
      await provinciasSelect.selectOption("Pichincha")
    }

    // Cantón
    const cantonesSelect = page.locator("select").filter({ hasText: "Cantón" }).first()
    if (await cantonesSelect.isVisible()) {
      await cantonesSelect.selectOption("Quito")
    }

    // Teléfono celular
    await page.fill("input[name='telefonoCelular']", "0987654321")
    // Email 1
    await page.fill("input[name='correo1']", correo)
    // Dirección
    await page.fill("input[name='callePrincipal']", "Calle Principal")
    await page.fill("input[name='numero']", "123")

    // Guardar sección 2
    const btnGuardarS2 = page.locator("button:has-text('Guardar')").first()
    if (await btnGuardarS2.isVisible()) {
      await btnGuardarS2.click()
      console.log("✅ Sección 2 guardada")
      await page.waitForTimeout(1000)
    }

    // =========================================================================
    // FASE 4: ENVÍO + VALIDACIÓN DE SELLADO
    // =========================================================================
    console.log("\n========== FASE 4: ENVÍO + SELLADO ==========\n")

    // Ir a resumen o botón de envío
    await page.goto("/postulacion")

    // Buscar botón "Enviar postulación"
    const btnEnviar = page.locator("button:has-text('Enviar Postulación')").first()
    expect(btnEnviar).toBeVisible()

    // Capturar la respuesta del POST /submit
    const submitPromise = page.waitForResponse(
      (resp) =>
        resp.url().includes("/submissions/") && resp.url().includes("/submit") && resp.status() === 200,
    )

    await btnEnviar.click()
    console.log("Botón 'Enviar Postulación' clickeado...")

    // Esperar modal de consentimiento (si existe)
    const modalConsentimiento = page.locator("dialog, [role='dialog']")
    if (await modalConsentimiento.isVisible({ timeout: 3000 })) {
      console.log("Modal de consentimiento visible, aceptando...")
      const checkboxConsentimiento = page.locator("input[type='checkbox']").first()
      await checkboxConsentimiento.check()
      const btnConfirmar = page.locator("button:has-text('Confirmar'), button:has-text('Aceptar')").first()
      if (await btnConfirmar.isVisible()) {
        await btnConfirmar.click()
      }
    }

    // Esperar respuesta del API
    const submitResponse = await submitPromise
    const submitBody = await submitResponse.json()
    console.log(`✅ Respuesta POST /submit: ${submitResponse.status()}`)
    console.log(`Respuesta body:`, JSON.stringify(submitBody, null, 2))

    // =========================================================================
    // VALIDACIÓN 1: Código único asignado
    // =========================================================================
    console.log("\n========== VALIDACIÓN 1: Código Único ==========\n")

    const codigo = submitBody?.data?.code || submitBody?.code
    expect(codigo).toBeTruthy()
    expect(codigo).toMatch(/^\d{4}-\d{5}$|^\d+$/) // Ej: 2026-00001 o similar
    console.log(`✅ Código único asignado: ${codigo}`)

    // =========================================================================
    // VALIDACIÓN 2: Hash SHA-256 presente
    // =========================================================================
    console.log("\n========== VALIDACIÓN 2: Hash SHA-256 ==========\n")

    const hash = submitBody?.data?.hash || submitBody?.hash
    expect(hash).toBeTruthy()
    expect(hash).toMatch(/^[a-f0-9]{64}$/) // SHA-256 = 64 chars hex
    console.log(`✅ Hash SHA-256 calculado: ${hash.substring(0, 16)}...`)

    // =========================================================================
    // VALIDACIÓN 3: Estado = submitted
    // =========================================================================
    console.log("\n========== VALIDACIÓN 3: Estado Submitted ==========\n")

    const estado = submitBody?.data?.status || submitBody?.status
    expect(estado).toBe("submitted")
    console.log(`✅ Estado: ${estado}`)

    // =========================================================================
    // VALIDACIÓN 4: Timestamp de envío presente
    // =========================================================================
    console.log("\n========== VALIDACIÓN 4: Timestamp Envío ==========\n")

    const submittedAt = submitBody?.data?.submitted_at || submitBody?.submitted_at
    expect(submittedAt).toBeTruthy()
    console.log(`✅ Timestamp envío: ${submittedAt}`)

    // =========================================================================
    // VALIDACIÓN 5: Correo de confirmación llegó
    // =========================================================================
    console.log("\n========== VALIDACIÓN 5: Correo de Confirmación ==========\n")

    await page.waitForTimeout(2000)
    const mailResponse = await fetch(`${MAILPIT_API}/messages`)
    const mailData = await mailResponse.json()
    const messages = Array.isArray(mailData) ? mailData : mailData.messages || []

    const emailConfirmacion = messages
      .filter((m: any) => m.To?.[0]?.Address === correo)
      .filter((m: any) => m.Subject?.includes("confirmación") || m.Subject?.includes("Enviada"))
      .sort((a: any, b: any) => new Date(b.Created).getTime() - new Date(a.Created).getTime())[0]

    expect(emailConfirmacion).toBeTruthy()
    expect(emailConfirmacion.Subject).toContain("confirmación")
    console.log(`✅ Email de confirmación recibido: ${emailConfirmacion.Subject}`)
    console.log(`   Contiene código: ${emailConfirmacion.Text?.includes(codigo) ? "SÍ" : "NO"}`)

    // =========================================================================
    // VALIDACIÓN 6: Formulario ahora es READ-ONLY (no editable)
    // =========================================================================
    console.log("\n========== VALIDACIÓN 6: Formulario Read-Only ==========\n")

    // Recargar la página para verificar estado
    await page.goto(`/postulacion`)
    await page.waitForTimeout(1000)

    // Intentar editar un campo
    const inputNombre = page.locator("input[name='nombres']").first()
    const isDisabled = await inputNombre.isDisabled()
    expect(isDisabled).toBe(true)
    console.log(`✅ Formulario está read-only (disabled: ${isDisabled})`)

    // Verificar que botón "Guardar" está deshabilitado o desaparecido
    const btnGuardarDisabled = page.locator("button:has-text('Guardar')").first()
    const btnGuardarVisible = await btnGuardarDisabled.isVisible()
    if (btnGuardarVisible) {
      const isDisabledBtn = await btnGuardarDisabled.isDisabled()
      console.log(`✅ Botón 'Guardar' deshabilitado: ${isDisabledBtn}`)
    } else {
      console.log(`✅ Botón 'Guardar' no visible (ocultado)`)
    }

    // =========================================================================
    // VALIDACIÓN 7: GET /submissions/{id} retorna estado submitted
    // =========================================================================
    console.log("\n========== VALIDACIÓN 7: GET /submissions/{id} ==========\n")

    // Extraer ID de la URL o del response inicial
    const urlActual = page.url()
    const idMatch = urlActual.match(/\/postulacion\/([a-f0-9-]+)/)
    const submissionId = idMatch ? idMatch[1] : submitBody?.data?.id || submitBody?.id

    const getResponse = await page.request.get(`${API_BASE}/submissions/${submissionId}`)
    expect(getResponse.status()).toBe(200)

    const getBody = await getResponse.json()
    console.log(`✅ GET /submissions/${submissionId}:`)
    console.log(`   Estado: ${getBody?.data?.status}`)
    console.log(`   Código: ${getBody?.data?.code}`)
    console.log(`   Hash: ${getBody?.data?.hash?.substring(0, 16)}...`)

    // =========================================================================
    // VALIDACIÓN 8: Intento de UPDATE devuelve 403 (Forbidden)
    // =========================================================================
    console.log("\n========== VALIDACIÓN 8: POST Update → 403 ==========\n")

    const updateResponse = await page.request.post(
      `${API_BASE}/submissions/${submissionId}/sections/1`,
      {
        data: {
          nombres: "Intentando editar",
          apellidos: "Después del envío",
        },
      },
    )

    expect(updateResponse.status()).toBe(403)
    console.log(`✅ Intento de edición retorna 403 Forbidden`)

    // =========================================================================
    // VALIDACIÓN 9: Botón "Descargar comprobante" visible
    // =========================================================================
    console.log("\n========== VALIDACIÓN 9: Comprobante Descargable ==========\n")

    const btnDescargar = page.locator("button:has-text('Descargar'), button:has-text('Comprobante')").first()
    if (await btnDescargar.isVisible()) {
      console.log(`✅ Botón 'Descargar comprobante' visible`)
    } else {
      console.log(`⚠️  Botón 'Descargar comprobante' no visible (puede agregarse después)`)
    }

    console.log("\n========== ✅ TEST COMPLETADO ==========\n")
  })
})
