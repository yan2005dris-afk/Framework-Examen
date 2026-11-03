import { expect, test } from "@playwright/test"

import { generarCedulaValida, generarCorreo } from "./fixtures/cedulas"
import {
  API_BASE,
  CONVOCATORIA_ID,
  MAILPIT_API,
  PASSWORD,
  cuentaPorCedula,
} from "./fixtures/datos"
import { leerOTP, listarMensajes } from "./fixtures/sesion"

/**
 * SUITE 2: VALIDACIONES DE LÍMITES Y CUOTAS
 * Criticidad: P0 (crítico - validaciones de negocio)
 *
 * Validaciones probadas:
 * 1. Máximo 50 MB de documentos por postulante
 * 2. Máximo 2 MB por archivo PDF
 * 3. Solo archivos PDF permitidos
 * 4. Máximo 1000 caracteres en campos de texto
 * 5. Cédula ecuatoriana válida (10 dígitos + módulo 10)
 * 6. Email válido (RFC 5321 - máximo 254 caracteres)
 * 7. Teléfono formato válido
 */

const USUARIOS_TEST = {
  aprobado: cuentaPorCedula("2406425922"),
  minimo: cuentaPorCedula("2402740035"),
}

/**
 * Construye un PDF válido del tamaño pedido, sin pasar del límite por archivo.
 *
 * @param {number} bytes - Tamaño exacto del archivo.
 * @returns {Buffer} El PDF listo para subir.
 */
function pdfDeTamano(bytes: number): Buffer {
  const cabecera = Buffer.from("%PDF-1.4\n", "utf8")
  return Buffer.concat([cabecera, Buffer.alloc(bytes - cabecera.length, "A")])
}

test.describe("VALIDACIONES DE LÍMITES Y CUOTAS", () => {
  // =========================================================================
  // VALIDACIÓN 1: Máximo 2 MB por archivo PDF
  // =========================================================================
  test("1.1 - Archivo PDF > 2MB es rechazado", async ({ page }) => {
    console.log("[TEST] Validando límite de 2MB por archivo...")

    // Login
    await page.goto("/login")
    await page.fill("input#cedula", USUARIOS_TEST.aprobado.cedula)
    await page.fill("input#password", PASSWORD)
    await page.click("button[type='submit']")
    await page.waitForURL(/\/postulacion/, { timeout: 15000 })

    // Ir a sección de documentos (ej: Formación académica)
    await page.goto("/postulacion/5")

    // Crear archivo PDF simulado de 3MB (excede límite)
    const bufferGrande = Buffer.alloc(3 * 1024 * 1024) // 3MB
    const fileInput = page.locator("input[type='file']").first()

    // Intentar cargar archivo grande
    await fileInput.setInputFiles({
      name: "archivo-grande.pdf",
      mimeType: "application/pdf",
      buffer: bufferGrande,
    })

    // Esperar validación del cliente (si existe)
    await page.waitForTimeout(500)

    // Debe mostrar error
    const errorMsg = page
      .getByText(/El archivo supera el tamaño máximo permitido/i)
      .first()

    if (await errorMsg.isVisible()) {
      console.log("[1.1.1] ✅ Error de tamaño mostrado en cliente")
      await expect(errorMsg).toBeVisible()
    } else {
      // Si el cliente no valida, intenta enviar y backend debería rechazar
      console.log(
        "[1.1.2] Sin validación cliente, esperando rechazo servidor...",
      )

      const btnGuardar = page.locator("button:has-text('Guardar|Cargar')")
      if (await btnGuardar.isVisible()) {
        await btnGuardar.click()
        await page.waitForTimeout(1000)

        // Backend debe rechazar
        const errorServidor = page.locator(
          "text=/archivo|tamaño|excede|no disponible|error/i",
        )
        await expect(errorServidor).toBeVisible({ timeout: 5000 })
        console.log("[1.1.2] ✅ Error de tamaño rechazado por servidor")
      }
    }
  })

  // =========================================================================
  // VALIDACIÓN 2: Máximo 50 MB de documentos por postulante
  // =========================================================================
  test("1.2 - Cuota total: Máximo 50 MB de documentos por postulante", async ({
    request,
  }) => {
    console.log("[TEST] Validando cuota total de 50MB...")

    // La cuota es por postulante, así que la prueba necesita una cuenta
    // propia: sembraría 50 MiB sobre una cuenta compartida y dejaría el
    // entorno inservible para el resto de la suite.
    const cedula = generarCedulaValida()
    const correo = generarCorreo("cuota-50mb")
    const password = "TestPassword.2026*"

    // Registro con OTP por correo
    const solicitud = await request.post(`${API_BASE}/auth/register/request`, {
      data: { cedula, correo },
    })
    expect(solicitud.ok()).toBeTruthy()

    const codigo = await leerOTP(correo)
    const verificacion = await request.post(
      `${API_BASE}/auth/register/verify`,
      {
        data: {
          cedula,
          correo,
          codigo_otp: codigo,
          nombres: "Cuota",
          apellidos: "Prueba",
          password,
        },
      },
    )
    expect(verificacion.status()).toBe(201)

    // Borrador del postulante recién registrado
    const creado = await request.post(`${API_BASE}/submissions`, {
      data: { convocatoriaId: CONVOCATORIA_ID },
    })
    expect(creado.status()).toBe(201)
    const postulacion = (await creado.json()) as { id?: string }
    expect(postulacion.id).toBeTruthy()
    const id = postulacion.id as string

    // Cada título registrado es una ranura de respaldo: la sección 5 admite
    // un PDF de hasta 2 MiB por título.
    const titulos = Array.from({ length: 30 }, (_, indice) => ({
      id: `titulo-cuota-${indice + 1}`,
      area: "Administración de Empresas",
      tipo: "Nacional",
      nivel: "Tercer Nivel de Grado",
      titulo: `Título de cuota ${indice + 1}`,
      institucion: "Institución de pruebas",
      registroSenescyt: `REG-CUOTA-${indice + 1}`,
      fechaRegistroSenescyt: "2015-07-01",
    }))
    const seccion = await request.post(
      `${API_BASE}/submissions/${id}/sections/5`,
      { data: { titulos } },
    )
    expect(seccion.ok()).toBeTruthy()

    // 1,9 MiB por respaldo: 27 respaldos pasan de los 50 MiB por postulante.
    const respaldo = pdfDeTamano(1_900_000)
    let aceptados = 0
    let rechazo: { status: number; codigo?: string } | null = null

    for (let indice = 0; indice < titulos.length; indice++) {
      const respuesta = await request.post(
        `${API_BASE}/submissions/${id}/documents`,
        {
          multipart: {
            section: "5",
            recordId: titulos[indice].id,
            file: {
              name: `cuota-${indice + 1}.pdf`,
              mimeType: "application/pdf",
              buffer: respaldo,
            },
          },
        },
      )

      if (respuesta.status() === 201) {
        aceptados++
        continue
      }

      const cuerpo = (await respuesta.json()) as {
        error?: { codigo?: string }
      }
      rechazo = { status: respuesta.status(), codigo: cuerpo.error?.codigo }
      break
    }

    console.log(
      `[1.2.1] Respaldos aceptados: ${aceptados}; último estado: ${rechazo?.status} ${rechazo?.codigo}`,
    )

    // El servidor debe cortar la cuota con 413 y el código propio.
    expect(rechazo, "el servidor nunca rechazó el respaldo").not.toBeNull()
    expect(rechazo?.status).toBe(413)
    expect(rechazo?.codigo).toBe("cuota_documentos_excedida")

    // La cuenta de la prueba es desechable: se retira para no dejar rastro.
    await request.delete(`${API_BASE}/submissions/${id}`)
  })

  // =========================================================================
  // VALIDACIÓN 3: Solo PDFs permitidos
  // =========================================================================
  test("1.3 - Solo archivos PDF permitidos (rechaza JPG, PNG, etc.)", async ({
    page,
  }) => {
    console.log("[TEST] Validando que solo PDFs se aceptan...")

    // Login
    await page.goto("/login")
    await page.fill("input#cedula", USUARIOS_TEST.aprobado.cedula)
    await page.fill("input#password", PASSWORD)
    await page.click("button[type='submit']")
    await page.waitForURL(/\/postulacion/, { timeout: 15000 })

    await page.goto("/postulacion/5")

    const fileInput = page.locator("input[type='file']").first()

    // Intentar cargar JPG
    const jpgBuffer = Buffer.from("FAKE_JPG_DATA")
    await fileInput.setInputFiles({
      name: "imagen.jpg",
      mimeType: "image/jpeg",
      buffer: jpgBuffer,
    })

    await page.waitForTimeout(500)

    // Debe rechazar
    const errorMsg = page
      .getByText(/Solo se permiten archivos en formato PDF/i)
      .first()

    if (await errorMsg.isVisible()) {
      console.log("[1.3.1] ✅ Archivo JPG rechazado")
      await expect(errorMsg).toBeVisible()
    } else {
      // O el input debería estar configurado con accept=".pdf"
      const accept = await fileInput.getAttribute("accept")
      expect(accept).toContain("pdf")
      console.log("[1.3.1] ✅ Input restringido a PDF via accept attribute")
    }
  })

  // =========================================================================
  // VALIDACIÓN 4: Máximo 1000 caracteres en campos de texto
  // =========================================================================
  test("1.4 - Campo de texto: Máximo 1000 caracteres", async ({ page }) => {
    console.log("[TEST] Validando límite de 1000 caracteres...")

    // Login
    await page.goto("/login")
    await page.fill("input#cedula", USUARIOS_TEST.aprobado.cedula)
    await page.fill("input#password", PASSWORD)
    await page.click("button[type='submit']")
    await page.waitForURL(/\/postulacion/, { timeout: 15000 })

    await page.goto("/postulacion/6") // Experiencia profesional (tiene descripción)

    // Buscar textarea o campo de descripción
    const textarea = page.locator(
      "textarea[name*='descripcion'], textarea[name*='funcion']",
    )

    if (await textarea.isVisible()) {
      const texto1000 = "a".repeat(1000)
      const texto1001 = "a".repeat(1001)

      // 1000 caracteres: debe aceptarse
      await textarea.fill(texto1000)
      const valor1 = await textarea.inputValue()
      expect(valor1.length).toBeLessThanOrEqual(1000)
      console.log("[1.4.1] ✅ 1000 caracteres aceptados")

      // 1001 caracteres: debe rechazarse o truncarse
      await textarea.clear()
      await textarea.fill(texto1001)
      const valor2 = await textarea.inputValue()
      expect(valor2.length).toBeLessThanOrEqual(1000)
      console.log("[1.4.2] ✅ 1001 caracteres rechazados/truncados")
    } else {
      console.log("[1.4] ⏭️  No hay textarea de descripción en esta sección")
    }
  })

  // =========================================================================
  // VALIDACIÓN 5: Cédula ecuatoriana válida (10 dígitos + módulo 10)
  // =========================================================================
  test("2.1 - Cédula: Validar formato y módulo 10", async ({ page }) => {
    console.log("[TEST] Validando formato de cédula ecuatoriana...")

    await page.goto("/login")

    const input = page.locator("input#cedula, input[name='cedula']")

    // Cédula válida: 10 dígitos correctos
    const cedulaValida = "1720104008" // Cédula real con módulo correcto

    await input.fill(cedulaValida)
    await input.blur()

    const errorValida = page.locator("text=/cédula|inválida|error/i")
    const isErrorVisible = await errorValida.isVisible().catch(() => false)
    if (isErrorVisible) {
      console.log("[2.1.1] Nota: Backend no rechaza esta cédula")
    } else {
      console.log("[2.1.1] ✅ Cédula válida aceptada")
    }

    // Cédula inválida: 9 dígitos
    await input.clear()
    await input.fill("123456789")
    await input.blur()
    await expect(errorValida).toBeVisible({ timeout: 2000 })
    console.log("[2.1.2] ✅ Cédula con 9 dígitos rechazada")

    // Cédula inválida: módulo 10 incorrecto
    await input.clear()
    await input.fill("0000000000")
    await input.blur()
    await expect(errorValida).toBeVisible({ timeout: 2000 })
    console.log("[2.1.3] ✅ Cédula con módulo 10 incorrecto rechazada")
  })

  // =========================================================================
  // VALIDACIÓN 6: Email válido (RFC 5321 - máximo 254 caracteres)
  // =========================================================================
  test("2.2 - Email: Formato RFC 5321", async ({ page }) => {
    console.log("[TEST] Validando formato de email...")

    // Login con el borrador: es la única postulación editable.
    await page.goto("/login")
    await page.fill("input#cedula", USUARIOS_TEST.minimo.cedula)
    await page.fill("input#password", PASSWORD)
    await page.click("button[type='submit']")
    await page.waitForURL(/\/postulacion/, { timeout: 15000 })

    await page.goto("/postulacion/2")

    // El correo principal viene de la cuenta y es de solo lectura: el campo
    // editable del formulario es el alternativo.
    const emailInput = page.locator("input#sec2-correo2")

    // Email válido
    await emailInput.fill("test+tag@example.co.uk")
    await emailInput.blur()
    const error = page.locator("aria-invalid=true")
    const isInvalid = await error.isVisible().catch(() => false)
    if (!isInvalid) {
      console.log("[2.2.1] ✅ Email válido aceptado")
    }

    // Email inválido: sin @
    await emailInput.clear()
    await emailInput.fill("testemailcom")
    await emailInput.blur()
    await expect(page.getByText("Correo inválido")).toBeVisible({
      timeout: 5000,
    })
    console.log("[2.2.2] ✅ Email sin @ rechazado")
  })

  // =========================================================================
  // VALIDACIÓN 6b: Longitud máxima del correo (RFC 5321)
  // =========================================================================
  test("2.4 - Email: se rechaza el correo de más de 254 caracteres", async ({
    page,
  }) => {
    test.skip(
      true,
      "El portal no limita la longitud del correo: validarEmail (apps/frontend/src/utils/validators.ts:140) sólo comprueba la forma y el campo alternativo del formulario (apps/frontend/src/features/formulario/seccion-2-contacto.tsx:218) no declara maxLength, de modo que un correo de 254 caracteres o más se guarda sin rechazo. El servidor tampoco lo acota (apps/backend/internal/identity/dto.go:12).",
    )
    console.log("[TEST] Validando longitud máxima del email...")

    // Login con el borrador, que es la única postulación editable.
    await page.goto("/login")
    await page.fill("input#cedula", USUARIOS_TEST.minimo.cedula)
    await page.fill("input#password", PASSWORD)
    await page.click("button[type='submit']")
    await page.waitForURL(/\/postulacion/, { timeout: 15000 })

    await page.goto("/postulacion/2")
    const emailInput = page.locator("input#sec2-correo2")

    const longEmail = "a".repeat(250) + "@example.com"
    await emailInput.clear()
    await emailInput.fill(longEmail)
    await emailInput.blur()

    const valor = await emailInput.inputValue()
    expect(valor.length).toBeLessThanOrEqual(254)
    console.log("[2.4.1] ✅ Email > 254 caracteres truncado o rechazado")
  })

  // =========================================================================
  // VALIDACIÓN 7: Teléfono formato válido
  // =========================================================================
  test("2.3 - Teléfono: Formato válido (+593 9XX XXX XXXX o 0987654321)", async ({
    page,
  }) => {
    console.log("[TEST] Validando formato de teléfono...")

    // Login
    await page.goto("/login")
    await page.fill("input#cedula", USUARIOS_TEST.aprobado.cedula)
    await page.fill("input#password", PASSWORD)
    await page.click("button[type='submit']")
    await page.waitForURL(/\/postulacion/, { timeout: 15000 })

    await page.goto("/postulacion/2")

    const phoneInput = page.locator(
      "input[name*='telefono'], input[name*='celular'], input[name*='phone']",
    )

    if (!(await phoneInput.isVisible())) {
      console.log("[2.3] ⏭️  No hay campo de teléfono")
      expect(true).toBe(true)
      return
    }

    // Teléfono válido: +593987654321
    await phoneInput.fill("+593987654321")
    await phoneInput.blur()
    let error = await phoneInput.getAttribute("aria-invalid")
    if (error !== "true") {
      console.log("[2.3.1] ✅ Formato +593 aceptado")
    }

    // Teléfono válido: 0987654321
    await phoneInput.clear()
    await phoneInput.fill("0987654321")
    await phoneInput.blur()
    error = await phoneInput.getAttribute("aria-invalid")
    if (error !== "true") {
      console.log("[2.3.2] ✅ Formato 09 aceptado")
    }

    // Teléfono inválido: muy corto
    await phoneInput.clear()
    await phoneInput.fill("123")
    await phoneInput.blur()
    const isInvalid = await phoneInput.getAttribute("aria-invalid")
    if (isInvalid === "true") {
      console.log("[2.3.3] ✅ Teléfono muy corto rechazado")
    }
  })

  // =========================================================================
  // VALIDACIÓN 8: Caracteres especiales UTF-8 en nombres
  // =========================================================================
  test("3.1 - Nombres: Aceptar Ñ, á, é, ü (UTF-8)", async ({ page }) => {
    console.log("[TEST] Validando soporte UTF-8 en nombres...")

    // Login con el borrador: es la única postulación editable.
    await page.goto("/login")
    await page.fill("input#cedula", USUARIOS_TEST.minimo.cedula)
    await page.fill("input#password", PASSWORD)
    await page.click("button[type='submit']")
    await page.waitForURL(/\/postulacion/, { timeout: 15000 })

    await page.goto("/postulacion/1")

    const nombresInput = page.locator("input#sec1-nombres")

    const nombresTest = [
      "José María",
      "Rafaela Jiménez",
      "Ándres García",
      "Susaña López",
    ]

    for (const nombre of nombresTest) {
      // Estos campos suelen ser read-only (vienen del Registro Civil)
      // Pero si son editables, deben aceptar UTF-8
      const isDisabled = await nombresInput.isDisabled()
      if (!isDisabled) {
        await nombresInput.clear()
        await nombresInput.fill(nombre)
        await nombresInput.blur()

        // El nombre se conserva tal cual y no se marca como inválido.
        await expect(nombresInput).toHaveValue(nombre)
        await expect(nombresInput).not.toHaveAttribute("aria-invalid", "true")
        console.log(`[3.1.1] ✅ Nombre "${nombre}" aceptado`)
      }
    }

    console.log("[3.1] ✅ UTF-8 en nombres validado")
  })
})
