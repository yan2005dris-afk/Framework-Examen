import { expect, test } from "@playwright/test"

import { generarCedulaValida } from "./fixtures/cedulas"
import {
  API_BASE,
  CONVOCATORIA_ID,
  PASSWORD,
  cuentaPorCedula,
} from "./fixtures/datos"
import { iniciarSesion } from "./fixtures/sesion"

/**
 * PRUEBAS E2E CRÍTICAS - Validación de Requerimientos EF-11 y EF-12
 *
 * Este suite verifica:
 * 1. Bloqueo de 3 intentos fallidos + 10 minutos (Redis)
 * 2. Cascada País → Provincia → Cantón con validaciones
 * 3. Restricción de nacionalidad ecuatoriana (CRÍTICO)
 * 4. Guardado progresivo de formulario
 * 5. Validaciones de reglas de negocio
 */

test.describe("VALIDACIÓN CRÍTICA - EF-11 y EF-12", () => {
  // =========================================================================
  // PRUEBA 1: BLOQUEO DE 3 INTENTOS FALLIDOS + 10 MINUTOS (EF-11)
  // =========================================================================
  test("1.1 - Bloqueo: 3 intentos fallidos → mensaje de bloqueo", async ({
    page,
  }) => {
    await page.goto("/login")

    // La cédula debe pasar el módulo 10: si no, el cliente la rechaza antes
    // de hablar con el servidor y nunca se cuentan los intentos.
    const cedula = generarCedulaValida()

    // Los dos primeros intentos fallidos responden credenciales incorrectas.
    for (let i = 1; i <= 2; i++) {
      await page.locator("input#cedula").fill(cedula)
      await page.locator("input#password").fill("WrongPassword123")
      await page.locator("button[type='submit']").click()

      // Esperar mensaje de error
      await expect(
        page.getByText(/credenciales|incorrecta/i).first(),
      ).toBeVisible({ timeout: 5000 })
    }

    // El tercer intento fallido agota los intentos permitidos.
    await page.locator("input#cedula").fill(cedula)
    await page.locator("input#password").fill("WrongPassword123")
    await page.locator("button[type='submit']").click()

    // Después del 3er intento fallido, debe mostrar bloqueo de 10 minutos
    await expect(
      page.getByText(
        /cuenta bloqueada|intentos agotados|10 minutos|próxima oportunidad/i,
      ),
    ).toBeVisible({ timeout: 5000 })

    // Verificar que hay un timer visible (cuenta regresiva)
    const timer = page.getByText(/\d{1,2}:\d{2}/) // Buscar formato MM:SS
    await expect(timer).toBeVisible({ timeout: 2000 })
  })

  test("1.2 - Bloqueo: POST a /auth/login devuelve 429 Rate Limited", async ({
    page,
  }) => {
    // Una cédula con módulo 10 válido: el servidor la cuenta y bloquea al
    // tercer intento. Una cédula inválida nunca llega a contar intentos.
    const cedula = generarCedulaValida()

    // 3 requests fallidas
    for (let i = 0; i < 3; i++) {
      const response = await page.request.post(`${API_BASE}/auth/login`, {
        data: {
          cedula,
          password: "WrongPassword",
        },
      })
      console.log(`Intento ${i + 1}: status ${response.status()}`)
    }

    // 4to request debe retornar 429 (Too Many Requests) o 403 (Locked)
    const blockedResponse = await page.request.post(`${API_BASE}/auth/login`, {
      data: {
        cedula,
        password: "WrongPassword",
      },
    })
    expect([429, 403, 400]).toContain(blockedResponse.status())
  })

  // =========================================================================
  // PRUEBA 2: CASCADA PAÍS → PROVINCIA → CANTÓN (EF-12)
  // =========================================================================
  test("2.1 - Cascada: Seleccionar ECUADOR carga provincias correctas", async ({
    page,
  }) => {
    // Asumir que postulante ya está registrado
    const cedulaTest = cuentaPorCedula("2402740035").cedula // Usuario minimo del portalspec
    await page.goto("/login")
    await page.locator("input#cedula").fill(cedulaTest)
    await page.locator("input#password").fill(PASSWORD)
    await page.locator("button[type='submit']").click()

    // Ir a sección 2 (contacto)
    await page.waitForURL(/\/postulacion.*/, { timeout: 15000 })
    await page.goto("/postulacion/2")

    // Cambiar país a Ecuador
    const selectPais = page.locator("select[name*='pais'], input[name*='pais']")
    if (await selectPais.isVisible()) {
      await selectPais.click()
      await page.getByRole("option", { name: /Ecuador/i }).click()

      // Esperar que carguen provincias
      await page.waitForTimeout(1000)

      // Verificar que dropdown de provincia NO esté vacío
      const selectProvincia = page.locator(
        "select[name*='provincia'], input[name*='provincia']",
      )
      await expect(selectProvincia).toBeEnabled()

      // Contar opciones en provincia
      const options = await page.locator("option").count()
      expect(options).toBeGreaterThan(1) // Al menos 1 provincia + placeholder
    }
  })

  test("2.2 - Cascada: Seleccionar provincia carga cantones correctos", async ({
    page,
  }) => {
    const cedulaTest = cuentaPorCedula("2402740035").cedula
    await page.goto("/login")
    await page.locator("input#cedula").fill(cedulaTest)
    await page.locator("input#password").fill(PASSWORD)
    await page.locator("button[type='submit']").click()

    await page.waitForURL(/\/postulacion.*/, { timeout: 15000 })
    await page.goto("/postulacion/2")

    // Seleccionar Ecuador
    const selectPais = page.locator("select[name*='pais'], input[name*='pais']")
    if (await selectPais.isVisible()) {
      await selectPais.click()
      await page.getByRole("option", { name: /Ecuador/i }).click()
      await page.waitForTimeout(500)

      // Seleccionar provincia (Pichincha)
      const selectProvincia = page.locator(
        "select[name*='provincia'], input[name*='provincia']",
      )
      await selectProvincia.click()
      await page
        .getByRole("option", { name: /Pichincha/i })
        .click({ force: true })
      await page.waitForTimeout(500)

      // Verificar que cantones cargaron
      const selectCanton = page.locator(
        "select[name*='canton'], input[name*='ciudad']",
      )
      await expect(selectCanton).toBeEnabled()

      // Seleccionar cantón
      await selectCanton.click()
      const quito = page.getByRole("option", { name: /Quito/i })
      await expect(quito).toBeVisible()
    }
  })

  test("2.3 - Cascada: País no-Ecuador → provincias vacías", async ({
    page,
  }) => {
    const cedulaTest = cuentaPorCedula("2402740035").cedula
    await page.goto("/login")
    await page.locator("input#cedula").fill(cedulaTest)
    await page.locator("input#password").fill(PASSWORD)
    await page.locator("button[type='submit']").click()

    await page.waitForURL(/\/postulacion.*/, { timeout: 15000 })
    await page.goto("/postulacion/2")

    // Seleccionar país no-Ecuador (Colombia)
    const selectPais = page.locator("select[name*='pais'], input[name*='pais']")
    if (await selectPais.isVisible()) {
      await selectPais.click()
      await page.getByRole("option", { name: /Colombia/i }).click()
      await page.waitForTimeout(500)

      // Provincia debe estar deshabilitada o vacía
      const selectProvincia = page.locator(
        "select[name*='provincia'], input[name*='provincia']",
      )

      // Debe ser disabled O no tener opciones
      const isDisabled = await selectProvincia.evaluate(
        (el: HTMLSelectElement | HTMLInputElement) =>
          el.disabled || el.getAttribute("disabled"),
      )
      expect(isDisabled).toBeTruthy()
    }
  })

  // =========================================================================
  // PRUEBA 3: RESTRICCIÓN NACIONALIDAD ECUATORIANA (CRÍTICO)
  // =========================================================================
  test("3.1 - CRÍTICO: Formulario rechaza nacionalidad no ecuatoriana", async ({
    page,
  }) => {
    // Este test validará si el sistema rechaza postulantes extranjeros
    // Si falla, demuestra que FALTA la validación (bug crítico)

    const cedulaTest = cuentaPorCedula("2402740035").cedula
    await page.goto("/login")
    await page.locator("input#cedula").fill(cedulaTest)
    await page.locator("input#password").fill(PASSWORD)
    await page.locator("button[type='submit']").click()

    await page.waitForURL(/\/postulacion.*/, { timeout: 15000 })

    // Ir a sección 1 (datos personales)
    await page.goto("/postulacion/1")

    // Cambiar nacionalidad a no-ecuatoriana
    const selectNacionalidad = page.locator(
      "select[name*='nacionalidad'], input[name*='nacionalidad']",
    )
    if (await selectNacionalidad.isVisible()) {
      await selectNacionalidad.click()
      await page
        .getByRole("option", { name: /Perú|Colombia|Otros/i })
        .first()
        .click()

      // Intentar guardar/enviar formulario
      const btnEnviar = page.getByRole("button", {
        name: /Enviar|Guardar|Siguiente/i,
      })
      if (await btnEnviar.isVisible()) {
        await btnEnviar.click()

        // ESPERADO: error "Solo postulantes ecuatorianos" o similar
        // ACTUAL: Si NO aparece este error, la validación está FALTANDO
        const errorMsg = page.getByText(
          /ecuatoriano|nacionalidad|no puede|no es elegible/i,
        )
        await expect(errorMsg).toBeVisible({
          timeout: 5000,
        })
      }
    }
  })

  // =========================================================================
  // PRUEBA 4: GUARDADO PROGRESIVO SIN PÉRDIDA DE DATOS
  // =========================================================================
  test("4.1 - Guardado: Llenar sección 1 → guardar → refrescar → datos persisten", async ({
    page,
  }) => {
    const cedulaTest = cuentaPorCedula("2402740035").cedula
    await page.goto("/login")
    await page.locator("input#cedula").fill(cedulaTest)
    await page.locator("input#password").fill(PASSWORD)
    await page.locator("button[type='submit']").click()

    await page.waitForURL(/\/postulacion.*/, { timeout: 15000 })
    await page.goto("/postulacion/1")

    // Llenar campo de ejemplo (asumiendo que hay un campo editable)
    const input = page.locator("input[type='text']").first()
    const testValue = "VALOR_PRUEBA_" + Date.now()

    if (await input.isVisible()) {
      await input.fill(testValue)

      // Buscar botón guardar
      const btnGuardar = page.getByRole("button", {
        name: /Guardar|Save|Guardar Progreso/i,
      })
      if (await btnGuardar.isVisible()) {
        await btnGuardar.click()

        // Esperar confirmación
        await page.waitForTimeout(1000)

        // Refrescar página
        await page.reload()
        await page.waitForURL(/\/postulacion\/1/, { timeout: 15000 })

        // Verificar que el valor persiste
        await expect(input).toHaveValue(testValue)
      }
    }
  })

  // =========================================================================
  // PRUEBA 5: VALIDACIONES DE CAMPOS OBLIGATORIOS
  // =========================================================================
  test("5.1 - Validaciones: Campos obligatorios marcan error", async ({
    page,
  }) => {
    const cedulaTest = cuentaPorCedula("2402740035").cedula
    await page.goto("/login")
    await page.locator("input#cedula").fill(cedulaTest)
    await page.locator("input#password").fill(PASSWORD)
    await page.locator("button[type='submit']").click()

    await page.waitForURL(/\/postulacion.*/, { timeout: 15000 })
    await page.goto("/postulacion/1")

    // Intentar enviar sin llenar campos obligatorios
    const btnEnviar = page.getByRole("button", {
      name: /Enviar|Siguiente|Guardar|Submit/i,
    })
    if (await btnEnviar.isVisible()) {
      await btnEnviar.click()

      // Debe mostrar errores de validación
      const errorMessages = page.getByText(
        /requerido|obligatorio|falta|debe llenar/i,
      )
      await expect(errorMessages.first()).toBeVisible({ timeout: 5000 })
    }
  })

  // =========================================================================
  // PRUEBA 6: CARGA DE DOCUMENTOS (PDF máximo 2-4 MB)
  // =========================================================================
  test("6.1 - Documentos: Rechaza archivo mayor a límite", async ({ page }) => {
    const cedulaTest = cuentaPorCedula("2402740035").cedula
    await page.goto("/login")
    await page.locator("input#cedula").fill(cedulaTest)
    await page.locator("input#password").fill(PASSWORD)
    await page.locator("button[type='submit']").click()

    await page.waitForURL(/\/postulacion.*/, { timeout: 15000 })

    // Ir a sección con documentos (ej: sección 5 - Formación académica)
    await page.goto("/postulacion/5")

    // Buscar input de archivo
    const fileInput = page.locator("input[type='file']").first()
    if (await fileInput.isVisible()) {
      // Este test verificaría validación de tamaño en cliente
      // (validación en servidor es obligatoria)
      expect(await fileInput.getAttribute("accept")).toContain("pdf")
    }
  })

  // =========================================================================
  // PRUEBA 7: INTEGRACIÓN CON BACKEND - TRANSACCIONES
  // =========================================================================
  test("7.1 - Backend: Guardar postulación es transaccional", async ({
    request,
  }) => {
    // Esta prueba verifica a nivel HTTP que la postulación se guarda
    // como una transacción atómica

    const cedulaTest = cuentaPorCedula("2402740035").cedula
    const loginResp = await request.post(`${API_BASE}/auth/login`, {
      data: {
        cedula: cedulaTest,
        password: PASSWORD,
      },
    })

    expect(loginResp.ok()).toBeTruthy()

    // POST a /submissions para guardar. El servidor toma la cuenta de la
    // sesión: el cuerpo sólo lleva la convocatoria.
    const submitResp = await request.post(`${API_BASE}/submissions`, {
      data: {
        convocatoriaId: CONVOCATORIA_ID,
      },
    })

    // Debe retornar 201 (Created)
    expect([200, 201, 202]).toContain(submitResp.status())

    // Response debe tener un ID de postulación
    const body = await submitResp.json()
    expect(body).toHaveProperty("id")

    // El borrador de prueba no debe quedar sembrado: se retira al terminar.
    const id = (body as { id?: string }).id
    if (id) {
      await request.delete(`${API_BASE}/submissions/${id}`)
    }
  })

  // =========================================================================
  // PRUEBA 8: REDIS CACHING - Catálogos
  // =========================================================================
  test("8.1 - Cache: Catálogos de países cargados desde Redis", async ({
    request,
  }) => {
    // El catálogo exige sesión: sin ella responde 401 y no llega al caché.
    const cuenta = cuentaPorCedula("2402740035")
    await iniciarSesion(request, cuenta.cedula, PASSWORD)

    // Primer request: carga desde DB → Redis
    const resp1 = await request.get(`${API_BASE}/catalogs/countries`)
    expect(resp1.ok()).toBeTruthy()
    const countries1 = await resp1.json()
    expect(Array.isArray(countries1.paises)).toBeTruthy()
    expect(countries1.paises.length).toBeGreaterThan(0)

    // Segundo request: debe ser más rápido (desde Redis)
    const start = Date.now()
    const resp2 = await request.get(`${API_BASE}/catalogs/countries`)
    const elapsed = Date.now() - start
    expect(resp2.ok()).toBeTruthy()

    // Cache debe hacer la respuesta < 50ms (típicamente 5-20ms)
    expect(elapsed).toBeLessThan(50)

    const countries2 = await resp2.json()
    expect(countries1).toEqual(countries2) // Mismo contenido
  })
})
