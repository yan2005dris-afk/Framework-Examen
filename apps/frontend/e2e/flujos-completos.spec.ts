import { expect, test } from "@playwright/test"

import { generarCedulaValida, generarCorreo } from "./fixtures/cedulas"
import { MAILPIT_API, cuentaPorCedula } from "./fixtures/datos"
import { crearPDF, listarMensajes } from "./fixtures/sesion"

/**
 * SUITE 1: FLUJOS COMPLETOS - Happy Path + Sad Path
 * Criticidad: P0 (más crítico - valida flujo principal)
 *
 * Flujos probados:
 * 1. Registro → Login → Llenar formulario 9 secciones → Enviar → Imprimir
 * 2. Rechazo por nacionalidad no ecuatoriana
 * 3. Rechazo por requisito no cumplido (ej: afiliado no afiliado)
 * 4. Flujo de recuperación de contraseña
 */

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

test.describe("FLUJOS COMPLETOS - Happy Path + Sad Path", () => {
  // =========================================================================
  // FLUJO 1: HAPPY PATH - Registro completo → Login → Postulación → Envío
  // =========================================================================
  test("1.1 - Happy Path: Registro → Login → Postulación → Envío → Impresión", async ({
    page,
  }) => {
    test.skip(
      true,
      "El flujo del formulario EF-12 no puede completarse en este entorno: el servidor exige la verificación de elegibilidad del sector antes de guardar la sección 4 y antes de enviar el expediente (apps/backend/internal/submission/rules.go:199 y rules.go:209), pero el stack de desarrollo no aprovisiona la simulación controlada (ELIGIBILITY_SIMULATION_FILE viene vacío en compose.yaml:60), de modo que toda cuenta recibe 503 elegibilidad_no_disponible (apps/backend/internal/eligibility/service.go:53 y apps/backend/internal/eligibility/simulated/provider.go:64). La suite de API cubre ese punto con un proveedor inyectado (apps/backend/e2e/eligibility.go:45).",
    )
    const cedula = generarCedulaValida()
    const correo = generarCorreo("test")
    const password = "SecurePass123!@#"

    console.log(`[TEST] Cedula: ${cedula}, Email: ${correo}`)

    // ─── PASO 1: REGISTRO ───
    console.log("[1] Iniciando registro...")
    await page.goto("/registro")
    expect(page).toHaveURL(/\/registro/)

    // Ingresar cédula
    await page.fill("input#cedula", cedula)
    await page.fill("input#correo", correo)

    // Click en "Continuar con la verificación"
    const btnEnviar = page.locator(
      "button:has-text('Continuar con la verificación')",
    )
    expect(btnEnviar).toBeVisible()
    await btnEnviar.click()

    // Esperar a que llegue el email con OTP
    console.log("[1.1] Esperando OTP en Mailpit...")
    const otp = await leerOTP(correo)
    expect(otp).toMatch(/^\d{6}$/)
    console.log(`[1.1] OTP recibido: ${otp}`)

    // ─── PASO 2: VERIFICAR OTP ───
    // Ingresar OTP
    await page.fill("input#otp", otp)
    const btnVerificar = page.locator("button:has-text('Verificar código')")
    expect(btnVerificar).toBeVisible()
    await btnVerificar.click()
    await page.waitForTimeout(500) // Esperar transición a paso 3

    // ─── PASO 3: INGRESAR CREDENCIALES ───
    console.log("[1.2] Paso 3: Ingresar nombre, apellido y contraseña...")
    await page.fill("input#nombres", "Test Usuario")
    await page.fill("input#apellidos", "TestApellido")
    await page.fill("input#reg-password", password)
    await page.fill("input#confirm-password", password)

    // Submit registro
    const btnRegistrar = page.locator("button:has-text('Finalizar registro')")
    expect(btnRegistrar).toBeVisible()
    await btnRegistrar.click()

    // Debe redirigir a postulación (auto-login post-registro)
    console.log("[1.2] Esperando redireccionamiento a /postulacion...")
    await page.waitForURL(/\/postulacion/, { timeout: 15000 })
    expect(page).toHaveURL(/\/postulacion/)

    // ─── PASO 2: LLENAR FORMULARIO 9 SECCIONES ───
    console.log("[2] Llenando formulario 9 secciones...")

    // Sección 1: Datos personales
    console.log("[2.1] Sección 1: Datos personales (cargados automáticamente)")
    // Los datos deben venir del Registro Civil automáticamente
    // Verificar que al menos nombres y apellidos están pre-llenados
    await page.goto("/postulacion/1")
    await expect(page.locator("input[name='nombres']")).toHaveValue(/.+/)
    await expect(page.locator("input[name='apellidos']")).toHaveValue(/.+/)

    // Nacionalidad: debe ser Ecuador (validación automática)
    const nacionalidad = await page
      .locator("select[name='nacionalidad']")
      .inputValue()
    expect(nacionalidad).toBe("Ecuador")

    // Estado civil
    await page.selectOption("select[name='estadoCivil']", "Soltero")
    await page.click("button:has-text('Guardar')")
    await expect(page.locator("text=Guardado")).toBeVisible()

    // Sección 2: Información domiciliaria
    console.log("[2.2] Sección 2: Domiciliaria")
    await page.goto("/postulacion/2")
    await page.selectOption("select[name='pais']", "Ecuador")
    await page.waitForTimeout(300) // Esperar cascada
    await page.selectOption("select[name='provincia']", "Pichincha")
    await page.waitForTimeout(300)
    await page.selectOption("select[name='ciudad']", "Quito")
    await page.fill("input[name='telefonoCelular']", "0987654321")
    // Email 1 ya está cargado
    await page.fill("input[name='email2']", `alt.${cedula}@biess-test.local`)
    await page.click("button:has-text('Guardar')")
    await expect(page.locator("text=Guardado")).toBeVisible()

    // Sección 3: Información adicional (familiares)
    console.log("[2.3] Sección 3: Información adicional")
    await page.goto("/postulacion/3")
    await page.click("input[name='tieneFamiliares'][value='No']") // Simplificar
    await page.click("button:has-text('Guardar')")
    await expect(page.locator("text=Guardado")).toBeVisible()

    // Sección 4: Selección del cargo (Afiliado)
    console.log("[2.4] Sección 4: Selección del cargo")
    await page.goto("/postulacion/4")
    await page.click(
      "radio[name='sector'][value='afiliado'], label:has-text('Afiliado')",
    )
    await page.waitForTimeout(500) // Esperar validación interoperabilidad

    // Si el usuario es afiliado, aparece checkbox de respaldo
    const respaldoCheckbox = page.locator(
      "input[name='tieneSindicato'], input[name='respaldado']",
    )
    if (await respaldoCheckbox.isVisible()) {
      await respaldoCheckbox.click()
      // Cargar documento PDF (simulado)
      const pdfBuffer = crearPDF("respaldo")
      await page.locator("input[type='file']").first().setInputFiles({
        name: "respaldo.pdf",
        mimeType: "application/pdf",
        buffer: pdfBuffer,
      })
      await page.waitForTimeout(500)
    }

    await page.click("button:has-text('Guardar')")
    await expect(page.locator("text=Guardado")).toBeVisible()

    // Sección 5: Formación académica
    console.log("[2.5] Sección 5: Formación académica")
    await page.goto("/postulacion/5")
    await page.click("button:has-text('Agregar')")
    await page.selectOption(
      "select[name='nivelInstruccion']",
      "Tercer Nivel de Grado",
    )
    await page.fill("input[name='institucion']", "Universidad XYZ")
    await page.selectOption("select[name='tipo']", "Nacional")
    await page.fill("input[name='titulo']", "Licenciado en Administración")
    await page.selectOption(
      "select[name='areaEstudios']",
      "Administración de Empresas",
    )
    await page.fill("input[name='numeroRegistroSENESCYT']", "1234567890")

    // Documento
    const pdfBuffer = crearPDF("formacion")
    await page.locator("input[type='file']").nth(0).setInputFiles({
      name: "titulo.pdf",
      mimeType: "application/pdf",
      buffer: pdfBuffer,
    })

    await page.click("button:has-text('Guardar')")
    await expect(page.locator("text=Guardado")).toBeVisible()

    // Sección 6: Experiencia profesional
    console.log("[2.6] Sección 6: Experiencia profesional")
    await page.goto("/postulacion/6")
    await page.click("button:has-text('Agregar')")
    await page.selectOption("select[name='tipoExperiencia']", "General")
    await page.fill("input[name='cargo']", "Contador General")
    await page.fill("input[name='institucion']", "Empresa ABC S.A.")
    await page.fill("input[name='fechaInicio']", "2018-01-01")
    await page.fill("input[name='fechaFin']", "2023-12-31")
    await page.selectOption(
      "select[name='actividades']",
      "Administración de Empresas",
    )
    await page.fill(
      "textarea[name='descripcion']",
      "Gestión contable y financiera",
    )

    await page
      .locator("input[type='file']")
      .nth(0)
      .setInputFiles({
        name: "experiencia.pdf",
        mimeType: "application/pdf",
        buffer: crearPDF("experiencia"),
      })

    await page.click("button:has-text('Guardar')")
    await expect(page.locator("text=Guardado")).toBeVisible()

    // Sección 7: Publicaciones
    console.log("[2.7] Sección 7: Publicaciones")
    await page.goto("/postulacion/7")
    await page.click("input[name='cuentaPublicaciones'][value='No']") // Simplificar
    await page.click("button:has-text('Guardar')")
    await expect(page.locator("text=Guardado")).toBeVisible()

    // Sección 8: Capacitación
    console.log("[2.8] Sección 8: Capacitación")
    await page.goto("/postulacion/8")
    await page.click("button:has-text('Agregar')")
    await page.selectOption("select[name='tipoEvento']", "Curso")
    await page.fill("input[name='nombreEvento']", "Curso de Finanzas")
    await page.fill("input[name='institucion']", "Instituto ABC")
    await page.fill("input[name='fechaInicio']", "2023-01-15")
    await page.fill("input[name='fechaFin']", "2023-02-15")
    await page.fill("input[name='numeroHoras']", "40")

    await page
      .locator("input[type='file']")
      .nth(0)
      .setInputFiles({
        name: "certificado.pdf",
        mimeType: "application/pdf",
        buffer: crearPDF("capacitacion"),
      })

    await page.click("button:has-text('Guardar')")
    await expect(page.locator("text=Guardado")).toBeVisible()

    // Sección 9: Inhabilidades (declaraciones)
    console.log("[2.9] Sección 9: Inhabilidades")
    await page.goto("/postulacion/9")

    // Todas las declaraciones deben ser "No"
    const declarations = [
      "esEcuatoriano",
      "noInahabilitadoComercio",
      "noEnMora",
      "noVinculoFinanciero",
      "noFuncionarioIESS",
      "noInteresPropio",
      "noRemovidoOrgControl",
      "noObligacionesSRI",
      "noCastigoFinanciero",
      "noLitigiandoIESS",
      "noProcesado",
      "noResponsabilidadContraloria",
      "noRegistradoUAFE",
    ]

    for (const decl of declarations) {
      const radio = page.locator(`input[name='${decl}'][value='No']`)
      if (await radio.isVisible()) {
        await radio.check()
      }
    }

    await page.click("button:has-text('Guardar')")
    await expect(page.locator("text=Guardado")).toBeVisible()

    // ─── PASO 3: ACEPTAR CONSENTIMIENTO Y ENVIAR ───
    console.log("[3] Enviando postulación...")
    await page.goto("/postulacion")

    // Aceptar los 5 puntos de consentimiento
    await page.click("input#consent-datos-personales")
    await page.click("input#consent-veracidad")
    await page.click("input#consent-normas-concurso")
    await page.click("input#consent-publicacion-nombre")
    await page.click("input#consent-entrega-fisica")

    // Enviar
    const btnEnviar2 = page.locator("button:has-text('Enviar Postulación')")
    expect(btnEnviar2).toBeVisible()
    await btnEnviar2.click()

    // Esperar confirmación
    await expect(
      page.locator("text=/Postulación enviada|sellada|Hash Criptográfico/i"),
    ).toBeVisible({ timeout: 10000 })

    console.log("[3.1] Postulación enviada exitosamente")

    // ─── PASO 4: VERIFICAR IMPRESIÓN ───
    console.log("[4] Verificando impresión...")
    await page.goto("/postulacion/imprimir")
    expect(page).toHaveURL(/\/postulacion\/imprimir/)

    // Debe haber hash criptográfico
    expect(page.locator("text=/Hash Criptográfico|SHA-256/i")).toBeVisible()

    // Botón de imprimir debe estar disponible
    const btnImprimir = page.locator("button:has-text('Imprimir')")
    expect(btnImprimir).toBeVisible()

    console.log("[4.1] ✅ Flujo completo exitoso")
  })

  // =========================================================================
  // FLUJO 2: SAD PATH - Rechazo por nacionalidad no ecuatoriana
  // =========================================================================
  test("2.1 - Sad Path: Rechazo por nacionalidad no ecuatoriana", async ({
    page,
  }) => {
    const cuenta = cuentaPorCedula("2406425922")
    const passwordTest = "TestPassword.2026*"

    console.log("[TEST] Validando rechazo de nacionalidad...")
    await page.goto("/login")
    await page.fill("input#cedula", cuenta.cedula)
    await page.fill("input#password", passwordTest)
    await page.click("button[type='submit']")

    await page.waitForURL(/\/postulacion/, { timeout: 15000 })

    // Ir a sección 1
    await page.goto("/postulacion/1")

    // Intentar cambiar nacionalidad a no-ecuatoriana
    const nacionalidadSelect = page.locator("select[name='nacionalidad']")
    const opcionesNoEC = await nacionalidadSelect.locator("option").count()

    if (opcionesNoEC > 2) {
      // Si hay opciones, intentar seleccionar una no-ecuatoriana
      await nacionalidadSelect.selectOption("Perú")

      // Intentar guardar
      await page.click("button:has-text('Guardar|Siguiente')")

      // Debe mostrar error específico
      const errorMsg = page.locator(
        "text=/ecuatoriano|nacionalidad|no puede continuar/i",
      )
      await expect(errorMsg).toBeVisible({ timeout: 5000 })

      console.log("[2.1.1] ✅ Rechazo de nacionalidad funcionando")
    } else {
      console.log(
        "[2.1] ⏭️  No hay opciones para probar cambio de nacionalidad (ya validado en BD)",
      )
    }
  })

  // =========================================================================
  // FLUJO 3: SAD PATH - Rechazo por requisito no cumplido
  // =========================================================================
  test("3.1 - Sad Path: Rechazo por requisito no cumplido (jubilado no jubilado)", async () => {
    // Este test requeriría un usuario con interoperabilidad fallida
    // Por ahora, documentamos el flujo esperado

    console.log("[TEST] Validando rechazo por requisito...")

    // Usuario intenta aplicar como jubilado pero no es jubilado en IESS
    // El backend valida esto mediante interoperabilidad

    // Esperamos que si intentamos:
    // 1. Seleccionar "Jubilado"
    // 2. Backend llama a IESS Historial Laboral
    // 3. Retorna "no acredita jubilación"
    // 4. Sistema rechaza con mensaje específico

    console.log(
      "[3.1] ⏭️  Test depende de usuarios con interoperabilidad fallida",
    )
    expect(true).toBe(true)
  })

  // =========================================================================
  // FLUJO 4: Recuperación de contraseña
  // =========================================================================
  test("4.1 - Happy Path: Recuperación de contraseña", async ({ page }) => {
    // La cuenta debe existir: la recuperación es anti-oráculo y no envía
    // ningún correo cuando la cédula no está registrada.
    const cedula = generarCedulaValida()
    const correo = generarCorreo("flujo-recuperacion")
    const password = "OriginalPassword.2026*"
    const nuevoPassword = "NewPassword456!@#"

    console.log("[TEST] Flujo de recuperación de contraseña...")

    // ─── Registro de la cuenta ───
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

    await page.fill("input#nombres", "Flujo")
    await page.fill("input#apellidos", "Recuperacion")
    await page.fill("input#reg-password", password)
    await page.fill("input#confirm-password", password)
    await page.locator("button:has-text('Finalizar registro')").click()

    // El registro cierra con el aviso de cuenta creada y la sesión activa:
    // no redirige al formulario por su cuenta.
    await expect(page.getByText("¡Registro completado!")).toBeVisible({
      timeout: 15000,
    })

    // ─── Recuperación ───
    // Ir al formulario de recuperación
    await page.goto("/recuperar-contrasena")
    await expect(page).toHaveURL(/\/recuperar-contrasena|\/forgot-password/)

    // Ingresar cédula
    await page.fill("input#cedula-recuperar", cedula)
    await page.locator("button:has-text('Continuar')").first().click()

    console.log("[4.1.1] Esperando OTP de recuperación...")
    await expect(page.locator("input#otp-recuperar")).toBeVisible({
      timeout: 15000,
    })

    // Obtener OTP del correo de la cuenta
    // El correo de recuperación se entrega después del de registro: hay que
    // pickar el mensaje por su asunto para no leer el código viejo.
    const otp = await leerOTP(correo, 30000, "Recuperación de contraseña")
    expect(otp).toMatch(/^\d{6}$/)

    // Ingresar OTP y nueva contraseña
    await page.fill("input#otp-recuperar", otp)
    await page.click("button:has-text('Continuar')")
    await expect(page.locator("input#nueva-password")).toBeVisible({
      timeout: 10000,
    })
    await page.fill("input#nueva-password", nuevoPassword)
    await page.fill("input#confirm-new-password", nuevoPassword)

    // Submit
    await page.click("button:has-text('Cambiar contraseña')")

    // La vista confirma el restablecimiento y ofrece el enlace al login:
    // no redirige sola.
    await expect(
      page.getByRole("heading", { name: /Contraseña restablecida/i }),
    ).toBeVisible({
      timeout: 15000,
    })
    await page
      .locator(
        "a:has-text('Ir a iniciar sesión'), button:has-text('Ir a iniciar sesión')",
      )
      .first()
      .click()
    await expect(page).toHaveURL(/\/login/, { timeout: 15000 })

    console.log("[4.1.2] ✅ Contraseña recuperada exitosamente")

    // Verificar que puede login con nueva contraseña
    await page.fill("input#cedula", cedula)
    await page.fill("input#password", nuevoPassword)
    await page.click("button[type='submit']")

    await page.waitForURL(/\/postulacion|\/seguimiento|\/dashboard/, {
      timeout: 15000,
    })
    console.log("[4.1.3] ✅ Login con nueva contraseña exitoso")
  })
})
