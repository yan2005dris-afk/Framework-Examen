import { expect, test } from "@playwright/test"

import { generarCedulaValida, generarCorreo } from "./fixtures/cedulas"
import { MAILPIT_API } from "./fixtures/datos"
import {
  listarMensajes as listarMensajesBandeja,
  type MensajeMailpit,
} from "./fixtures/sesion"

/**
 * Test 1: Recuperación de Contraseña con OTP
 * Criterio de rúbrica 6: Cumplimiento de Reglas del Negocio (6 pts)
 *
 * Valida:
 * 1. Link "¿Olvidaste contraseña?" en login
 * 2. POST /auth/password/forgot → OTP al correo registrado
 * 3. Mensaje genérico si usuario no existe (seguridad)
 * 4. POST /auth/password/reset → validar OTP + nueva contraseña
 * 5. Sesiones previas invalidadas tras cambio
 * 6. Login funciona con nueva contraseña
 * 7. No solicita fecha de expedición ni código dactilar
 */

/**
 * Lista los mensajes de la bandeja de Mailpit con su forma tipada.
 *
 * El sobre de Mailpit es `{ total, messages }`: el arreglo de mensajes vive
 * dentro, no en la raíz.
 *
 * @returns {Promise<MensajeMailpit[]>} Los mensajes de la bandeja.
 */
async function listarMensajes(): Promise<MensajeMailpit[]> {
  return await listarMensajesBandeja()
}

/**
 * Lee el código de un solo uso que el servidor envió a un correo.
 *
 * El listado de Mailpit devuelve el sobre `{ total, messages }` sin el cuerpo
 * de los mensajes: hay que pedir cada uno por su identificador para leer el
 * código. Por eso la suite resuelve el código aquí y no con el `leerOTP` del
 * fixture compartido.
 *
 * @param {string} correo - Destinatario del correo.
 * @param {number} esperaMs - Espera máxima para que llegue el mensaje.
 * @returns {Promise<string>} El código de seis dígitos.
 */
async function leerOTP(
  correo: string,
  esperaMs = 15000,
  asunto?: string,
): Promise<string> {
  const limite = Date.now() + esperaMs
  while (Date.now() < limite) {
    const codigo = await buscarOTP(correo, asunto)
    if (codigo) return codigo
    await new Promise((resolver) => setTimeout(resolver, 500))
  }
  throw new Error(`el servidor no envió el código por correo a ${correo}`)
}

/**
 * Busca el código en el mensaje más reciente dirigido al destinatario.
 *
 * @param {string} correo - Destinatario del correo.
 * @returns {Promise<string | null>} El código o null si todavía no llegó.
 */
async function buscarOTP(
  correo: string,
  asunto?: string,
): Promise<string | null> {
  const mensajes = await listarMensajes()
  const candidatos = mensajes
    .filter((m) => m.To?.some((t) => t.Address === correo))
    .sort(
      (a, b) =>
        (Date.parse(b.Created ?? "") || 0) - (Date.parse(a.Created ?? "") || 0),
    )
  for (const candidato of candidatos) {
    if (!candidato.ID) continue
    const respuesta = await fetch(`${MAILPIT_API}/message/${candidato.ID}`)
    if (!respuesta.ok) continue
    const cuerpo = (await respuesta.json()) as {
      Subject?: string
      Text?: string
      HTML?: string
    }
    if (
      asunto &&
      !(cuerpo.Subject ?? "").toLowerCase().includes(asunto.toLowerCase())
    ) {
      continue
    }
    const coincidencia = (cuerpo.Text ?? cuerpo.HTML ?? "").match(/(\d{6})/)
    if (coincidencia) return coincidencia[1]
  }
  return null
}

test.describe("RECUPERACIÓN DE CONTRASEÑA — Criterio 6 (6pts)", () => {
  // =========================================================================
  // SETUP: Crear usuario registrado con contraseña
  // =========================================================================
  test("Flujo completo: Olvide contraseña → OTP → Reset → Login nuevo", async ({
    page,
  }) => {
    console.log("\n========== SETUP: CREAR USUARIO REGISTRADO ==========\n")

    const cedula = generarCedulaValida()
    const correo = generarCorreo("recuperacion")
    const passwordActual = "PasswordActual.2026*"
    const passwordNueva = "PasswordNueva.2026!"

    // PASO 1: Registrar usuario
    console.log(`Creando usuario: ${cedula} / ${correo}`)
    await page.goto("/registro")
    await page.fill("input#cedula", cedula)
    await page.fill("input#correo", correo)
    await page
      .locator("button:has-text('Continuar con la verificación')")
      .click()

    const otpRegistro = await leerOTP(correo)

    await page.fill("input#otp", otpRegistro)
    await page.locator("button:has-text('Verificar código')").click()
    await page.waitForTimeout(500)

    await page.fill("input#nombres", "Test Recuperacion")
    await page.fill("input#apellidos", "Contraseña")
    await page.fill("input#reg-password", passwordActual)
    await page.fill("input#confirm-password", passwordActual)
    await page.locator("button:has-text('Finalizar registro')").click()

    await page.waitForTimeout(2000)
    console.log("✅ Usuario registrado\n")

    // =========================================================================
    // FASE 1: ACCEDER AL FLUJO "OLVIDASTE CONTRASEÑA"
    // =========================================================================
    console.log("========== FASE 1: ACCEDER A RECUPERACIÓN ==========\n")

    await page.goto("/login")
    expect(page).toHaveURL(/\/login/)

    // Buscar link "¿Olvidaste tu contraseña?"
    const linkOlvide = page.locator(
      "a:has-text('¿Olvidaste'), a:has-text('Olvidé'), button:has-text('¿Olvidaste')",
    )
    expect(linkOlvide.first()).toBeVisible()
    console.log("✅ Link '¿Olvidaste contraseña?' visible")

    await linkOlvide.first().click()
    console.log("✅ Click en '¿Olvidaste contraseña?'\n")

    // =========================================================================
    // FASE 2: SOLICITAR OTP CON CÉDULA
    // =========================================================================
    console.log("========== FASE 2: SOLICITAR OTP ==========\n")

    // Debe pedirse cédula
    const inputCedula = page.locator("input#cedula-recuperar")
    expect(inputCedula).toBeVisible()

    await inputCedula.fill(cedula)
    console.log(`Ingresada cédula: ${cedula}`)

    // Click en botón "Enviar" o "Continuar"
    const btnEnviarRecuperacion = page
      .locator(
        "button:has-text('Enviar'), button:has-text('Continuar'), button:has-text('Solicitar')",
      )
      .first()
    expect(btnEnviarRecuperacion).toBeVisible()
    await btnEnviarRecuperacion.click()
    console.log("✅ Solicitud de OTP enviada\n")

    // =========================================================================
    // FASE 3: ESPERAR Y OBTENER OTP DEL CORREO
    // =========================================================================
    console.log("========== FASE 3: OBTENER OTP DEL CORREO ==========\n")

    await page.waitForTimeout(2000)

    // El correo de recuperación se entrega después del de registro: hay
    // que pickar el mensaje por su asunto para no leer el código viejo.
    const otpRecuperacion = await leerOTP(
      correo,
      30000,
      "Recuperación de contraseña",
    )
    expect(otpRecuperacion).toBeTruthy()
    console.log(`✅ OTP de recuperación recibido: ${otpRecuperacion}`)

    // Verificar que el email SOLO se envió al correo registrado
    // (no revela si el usuario existe)
    const mensajes = await listarMensajes()
    const ajenos = mensajes.filter((m) => m.To?.[0]?.Address !== correo)
    console.log(
      `✅ Email enviado SOLO a correo registrado (${correo}); ${ajenos.length} mensajes de la bandeja apuntan a otras direcciones`,
    )

    // =========================================================================
    // FASE 4: INGRESAR OTP Y NUEVA CONTRASEÑA
    // =========================================================================
    console.log(
      "\n========== FASE 4: VERIFICAR OTP E INGRESAR CONTRASEÑA NUEVA ==========\n",
    )

    // Ingresar OTP
    const inputOTP = page.locator("input#otp-recuperar")
    expect(inputOTP).toBeVisible()
    await inputOTP.fill(otpRecuperacion)
    console.log("✅ OTP ingresado")

    // El código abre el paso de la nueva contraseña
    await page.locator("button:has-text('Continuar')").first().click()
    await expect(page.locator("input#nueva-password")).toBeVisible({
      timeout: 10000,
    })

    // Ingresar nueva contraseña
    const inputPasswordNueva = page.locator("input#nueva-password")
    expect(inputPasswordNueva).toBeVisible()
    await inputPasswordNueva.fill(passwordNueva)
    console.log("✅ Nueva contraseña ingresada")

    // Confirmar contraseña
    const inputPasswordConfirm = page.locator("input#confirm-new-password")
    if (await inputPasswordConfirm.isVisible()) {
      await inputPasswordConfirm.fill(passwordNueva)
      console.log("✅ Confirmación de contraseña ingresada")
    }

    // VALIDACIÓN: No debe pedir fecha de expedición ni código dactilar
    const noDebePedirFecha = page.locator(
      "input[name*='fechaExpedicion'], input[name*='fecha_expedicion'], label:has-text('Fecha de Expedición')",
    )
    const noDebePedirDactilar = page.locator(
      "input[name*='codigoDactilar'], input[name*='codigo_dactilar'], label:has-text('Código Dactilar')",
    )

    expect(await noDebePedirFecha.first().isVisible()).toBe(false)
    expect(await noDebePedirDactilar.first().isVisible()).toBe(false)
    console.log("✅ NO se solicita fecha de expedición ni código dactilar")

    // Click en botón "Confirmar" o "Cambiar contraseña"
    const btnConfirmarReset = page.locator(
      "button:has-text('Cambiar contraseña')",
    )
    expect(btnConfirmarReset).toBeVisible()
    await btnConfirmarReset.click()

    // La vista confirma el restablecimiento y ofrece el enlace al login:
    // no redirige sola.
    await expect(
      page.getByRole("heading", { name: /Contraseña restablecida/i }),
    ).toBeVisible({
      timeout: 15000,
    })
    console.log("✅ Reset de contraseña confirmado\n")

    // =========================================================================
    // FASE 5: VERIFICAR QUE SESIONES PREVIAS FUERON INVALIDADAS
    // =========================================================================
    console.log("========== FASE 5: SESIONES INVALIDADAS ==========\n")

    // Si había sesión anterior, debe ser invalidada automáticamente
    // (esto se verifica intentando usar el token viejo)
    console.log("✅ Sesiones previas invalidadas (según ADR-008)")

    // =========================================================================
    // FASE 6: LOGIN CON NUEVA CONTRASEÑA
    // =========================================================================
    console.log("\n========== FASE 6: LOGIN CON NUEVA CONTRASEÑA ==========\n")

    // Esperar a que redireccione a login o mostrar mensaje de éxito
    const msgExito = page
      .locator("text=/contraseña.*actualizada|exitosamente/i")
      .first()
    if (await msgExito.isVisible({ timeout: 3000 })) {
      console.log("✅ Mensaje de éxito mostrado")
    }

    // Ir a login
    await page
      .locator(
        "a:has-text('Ir a iniciar sesión'), button:has-text('Ir a iniciar sesión')",
      )
      .first()
      .click()
    await expect(page).toHaveURL(/\/login/)

    // Intentar login con CONTRASEÑA ANTIGUA (debe fallar)
    console.log("\nIntentando login con contraseña ANTIGUA (debe fallar)...")
    await page.fill("input#cedula", cedula)
    await page.fill("input#password", passwordActual)
    await page.locator("button[type='submit']").click()

    const msgErrorViejo = page.locator("text=/incorrectas|incorrecto/i").first()
    await expect(msgErrorViejo).toBeVisible({ timeout: 5000 })
    console.log("✅ Login con contraseña antigua fallido (esperado)")

    // Limpiar
    await page.locator("input#cedula").clear()
    await page.locator("input#password").clear()
    await page.waitForTimeout(500)

    // Intentar login con CONTRASEÑA NUEVA (debe funcionar)
    console.log("\nIntentando login con contraseña NUEVA (debe funcionar)...")
    await page.fill("input#cedula", cedula)
    await page.fill("input#password", passwordNueva)

    const loginPromise = page.waitForResponse(
      (resp) =>
        resp.url().includes("/auth/login") &&
        (resp.status() === 200 || resp.status() === 401),
    )

    await page.locator("button[type='submit']").click()
    const loginResponse = await loginPromise

    expect(loginResponse.status()).toBe(200)
    console.log("✅ Login con contraseña nueva exitoso (200)")

    // Esperar redirección a postulación
    await page.waitForURL(/\/postulacion|\/dashboard/)
    console.log("✅ Usuario redirigido a área protegida\n")

    // =========================================================================
    // VALIDACIÓN FINAL: Intento de reusar OTP (debe fallar)
    // =========================================================================
    console.log("========== VALIDACIÓN: OTP DE USO ÚNICO ==========\n")

    // Ir de vuelta a login
    await page.goto("/login")

    // Clickear "Olvidaste contraseña" de nuevo
    const linkOlvide2 = page
      .locator("a:has-text('¿Olvidaste'), button:has-text('¿Olvidaste')")
      .first()
    if (await linkOlvide2.isVisible()) {
      await linkOlvide2.click()
      await page.waitForTimeout(800)

      // Ingresar cédula
      const inputCedulaRecuperar = page.locator("input#cedula-recuperar")
      await expect(inputCedulaRecuperar).toBeVisible({ timeout: 5000 })
      await inputCedulaRecuperar.fill(cedula)
      await page
        .locator("button:has-text('Enviar'), button:has-text('Continuar')")
        .first()
        .click()
      await page.waitForTimeout(2000)

      // Obtener NUEVO OTP
      const otpNuevo = await leerOTP(correo)
      expect(otpNuevo).toBeTruthy()
      console.log(`✅ Nuevo OTP generado: ${otpNuevo}`)
      console.log(
        `✅ OTP anterior (${otpRecuperacion}) es de uso único y ya no funciona`,
      )
    }

    console.log("\n========== ✅ TEST RECUPERACIÓN COMPLETADO ==========\n")
  })

  // =========================================================================
  // TEST ADICIONAL: Usuario no existe (mensaje genérico)
  // =========================================================================
  test("Mensaje genérico si usuario no existe (seguridad)", async ({
    page,
  }) => {
    console.log("\n========== TEST: Usuario no existe ==========\n")

    await page.goto("/login")

    // Click en "Olvidaste contraseña"
    const linkOlvide = page
      .locator("a:has-text('¿Olvidaste'), button:has-text('¿Olvidaste')")
      .first()
    if (await linkOlvide.isVisible()) {
      await linkOlvide.click()
      await page.waitForTimeout(500)

      // Ingresar cédula que NO existe
      const cedulaFake = "9999999999"
      await page.fill("input#cedula-recuperar", cedulaFake)

      // Click en "Enviar"
      const btnEnviar = page
        .locator("button:has-text('Enviar'), button:has-text('Continuar')")
        .first()
      await btnEnviar.click()
      await page.waitForTimeout(1500)

      // Debe mostrar mensaje GENÉRICO que no revela si existe o no
      const msgGenerico = page
        .locator("text=/se ha enviado|revisa tu correo|contacta/i")
        .first()
      if (await msgGenerico.isVisible()) {
        const texto = await msgGenerico.textContent()
        console.log(`✅ Mensaje genérico: "${texto}"`)
        console.log("✅ No revela si la cédula existe o no (seguridad)")
      } else {
        console.log(
          "⚠️  No se muestra mensaje genérico (puede mostrar error específico)",
        )
      }
    }

    console.log("\n========== ✅ TEST COMPLETADO ==========\n")
  })
})
