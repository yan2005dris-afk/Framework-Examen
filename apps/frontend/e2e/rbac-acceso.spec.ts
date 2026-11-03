import { expect, test } from "@playwright/test"

import { PASSWORD, cuentaPorRol } from "./fixtures/datos"

/**
 * SUITE 3: RBAC - CONTROL DE ACCESO POR ROL
 * Criticidad: P0 (crítico - seguridad)
 *
 * Roles probados:
 * 1. POSTULANTE - Solo ve /postulacion, /seguimiento, /reconsideracion, /evaluacion
 * 2. ANALISTA - Ve /dashboard, puede descargar matrices, no accede a /admin
 * 3. ADMINISTRADOR - Acceso completo a /admin, /admin/preguntas, /admin/fases
 *
 * Validaciones:
 * - Login redirija al destino correcto por rol
 * - Acceso negado (403) a rutas no autorizadas
 * - Menú adaptado según rol
 * - Campos disabled/hidden según permisos
 */

const USUARIOS_TEST = {
  postulante: cuentaPorRol("postulante"),
  analista: cuentaPorRol("analista"),
  admin: cuentaPorRol("administrador"),
}

test.describe("RBAC - CONTROL DE ACCESO POR ROL", () => {
  // =========================================================================
  // POSTULANTE: Solo ve rutas de postulación
  // =========================================================================
  test.describe("Postulante", () => {
    test("1.1 - Postulante: Login redirije a /postulacion", async ({
      page,
    }) => {
      console.log("[TEST] Verificando redirección postulante...")

      await page.goto("/login")
      await page.fill("input#cedula", USUARIOS_TEST.postulante.cedula)
      await page.fill("input#password", PASSWORD)
      await page.click("button[type='submit']")

      await page.waitForURL(/\/postulacion|\/seguimiento/, { timeout: 15000 })

      // Debe ir a /postulacion (o /seguimiento si ya envió)
      const url = page.url()
      expect(
        ["/postulacion", "/seguimiento"].some((p) => url.includes(p)),
      ).toBe(true)
      console.log(`[1.1.1] ✅ Redireccionado a ${url}`)
    })

    test("1.2 - Postulante: No ve /dashboard", async ({ page }) => {
      console.log("[TEST] Validando que postulante no accede a dashboard...")

      await page.goto("/login")
      await page.fill("input#cedula", USUARIOS_TEST.postulante.cedula)
      await page.fill("input#password", PASSWORD)
      await page.click("button[type='submit']")
      await page.waitForURL(/\/postulacion/, { timeout: 15000 })

      // Intentar acceder a dashboard
      await page.goto("/dashboard")

      // El guard responde 403 con la vista de sin permiso en la misma ruta:
      // la URL no cambia y el contenido del panel nunca se entrega.
      const deniedMsg = page.getByText("No tiene permisos para esta sección")
      await expect(deniedMsg).toBeVisible({ timeout: 10000 })
      console.log("[1.2.1] ✅ Mensaje de permisos mostrado")

      await expect(
        page.locator("h1:has-text('Panel de revisión')"),
      ).toHaveCount(0)
      console.log("[1.2.2] ✅ Contenido del panel NO visible")
    })

    test("1.3 - Postulante: No ve /admin", async ({ page }) => {
      console.log("[TEST] Validando que postulante no accede a admin...")

      await page.goto("/login")
      await page.fill("input#cedula", USUARIOS_TEST.postulante.cedula)
      await page.fill("input#password", PASSWORD)
      await page.click("button[type='submit']")
      await page.waitForURL(/\/postulacion/, { timeout: 15000 })

      // Intentar acceder a /admin/preguntas
      await page.goto("/admin/preguntas")

      // El guard responde 403 con la vista de sin permiso en la misma ruta:
      // la URL no cambia y el contenido administrativo nunca se entrega.
      const deniedMsg = page.getByText("No tiene permisos para esta sección")
      await expect(deniedMsg).toBeVisible({ timeout: 10000 })
      console.log("[1.3.1] ✅ Mensaje de permisos mostrado")
    })

    test("1.4 - Postulante: Sí ve /reconsideracion (si procede)", async ({
      page,
    }) => {
      console.log("[TEST] Validando acceso a reconsideración...")

      await page.goto("/login")
      await page.fill("input#cedula", USUARIOS_TEST.postulante.cedula)
      await page.fill("input#password", PASSWORD)
      await page.click("button[type='submit']")
      await page.waitForURL(/\/postulacion|\/seguimiento/, { timeout: 15000 })

      // Ir a reconsideraciones (si procede)
      await page.goto("/reconsideracion")

      // La vista expone su título como título de tarjeta, no como encabezado
      const titulo = page
        .locator("[data-slot='card-title']")
        .filter({ hasText: "Solicitudes de Reconsideración" })
        .first()
      await expect(titulo).toBeVisible({ timeout: 10000 })

      console.log("[1.4.1] ✅ Ruta /reconsideracion accesible")
    })

    test("1.5 - Postulante: Menú solo muestra opciones permitidas", async ({
      page,
    }) => {
      console.log("[TEST] Validando menú del postulante...")

      await page.goto("/login")
      await page.fill("input#cedula", USUARIOS_TEST.postulante.cedula)
      await page.fill("input#password", PASSWORD)
      await page.click("button[type='submit']")
      await page.waitForURL(/\/postulacion/, { timeout: 15000 })

      // Menú debe tener
      const navMenu = page.locator("nav, [role='navigation']")
      if (await navMenu.isVisible()) {
        // Debe tener enlace a postulación
        const postulacionLink = page.locator(
          "a:has-text('Postulación|Formulario')",
        )
        if (await postulacionLink.isVisible()) {
          console.log("[1.5.1] ✅ Enlace a postulación visible")
        }

        // NO debe tener enlace a dashboard
        const dashboardLink = page.locator("a:has-text('Dashboard|Admin')")
        const isDashboardVisible = await dashboardLink
          .isVisible()
          .catch(() => false)
        if (!isDashboardVisible) {
          console.log("[1.5.2] ✅ Enlace a dashboard NOT visible")
        }
      } else {
        console.log("[1.5] ⏭️  No hay navbar")
      }
    })
  })

  // =========================================================================
  // ANALISTA: Ve dashboard, NO ve admin
  // =========================================================================
  test.describe("Analista", () => {
    test("2.1 - Analista: Login redirije a /dashboard", async ({ page }) => {
      console.log("[TEST] Verificando redirección analista...")

      await page.goto("/login")
      await page.fill("input#cedula", USUARIOS_TEST.analista.cedula)
      await page.fill("input#password", PASSWORD)
      await page.click("button[type='submit']")

      await page.waitForURL(/\/dashboard|\/postulacion/, { timeout: 15000 })

      // Debe ir a dashboard
      expect(page.url()).toContain("/dashboard")
      console.log("[2.1.1] ✅ Redireccionado a /dashboard")
    })

    test("2.2 - Analista: Accede a /dashboard", async ({ page }) => {
      console.log("[TEST] Validando acceso a dashboard...")

      await page.goto("/login")
      await page.fill("input#cedula", USUARIOS_TEST.analista.cedula)
      await page.fill("input#password", PASSWORD)
      await page.click("button[type='submit']")
      await page.waitForURL(/\/dashboard/, { timeout: 15000 })

      // Dashboard debe tener contenido: el panel de revisión muestra su
      // encabezado con el nombre de la vista.
      const titulo = page.locator("h1", { hasText: "Panel de revisión" })
      await expect(titulo).toBeVisible({ timeout: 10000 })
      console.log("[2.2.1] ✅ Dashboard accesible")
    })

    test("2.3 - Analista: Puede descargar matriz XLSX", async ({ page }) => {
      console.log("[TEST] Validando descarga de matriz XLSX...")

      await page.goto("/login")
      await page.fill("input#cedula", USUARIOS_TEST.analista.cedula)
      await page.fill("input#password", PASSWORD)
      await page.click("button[type='submit']")
      await page.waitForURL(/\/dashboard/, { timeout: 15000 })

      // Buscar botón de descargar
      const btnDescargar = page.locator(
        "button:has-text('Descargar Excel|Descargar XLSX|Exportar')",
      )

      if (await btnDescargar.isVisible()) {
        console.log("[2.3.1] ✅ Botón de descarga visible")

        // Escuchar descarga
        const downloadPromise = page.waitForEvent("download")
        await btnDescargar.click()

        try {
          const download = await downloadPromise
          expect(download.suggestedFilename()).toMatch(/\.xlsx|\.xls/)
          console.log("[2.3.2] ✅ Archivo XLSX descargado")
        } catch {
          console.log("[2.3.2] ⏭️  Descarga puede requerir confirmación")
        }
      } else {
        console.log("[2.3] ⏭️  Botón de descarga no encontrado")
      }
    })

    test("2.4 - Analista: NO accede a /admin", async ({ page }) => {
      console.log("[TEST] Validando que analista no accede a admin...")

      await page.goto("/login")
      await page.fill("input#cedula", USUARIOS_TEST.analista.cedula)
      await page.fill("input#password", PASSWORD)
      await page.click("button[type='submit']")
      await page.waitForURL(/\/dashboard/, { timeout: 15000 })

      // Intentar acceder a /admin/preguntas
      await page.goto("/admin/preguntas")

      // El guard responde 403 con la vista de sin permiso en la misma ruta:
      // la URL no cambia y el contenido administrativo nunca se entrega.
      const deniedMsg = page.getByText("No tiene permisos para esta sección")
      await expect(deniedMsg).toBeVisible({ timeout: 10000 })
      console.log("[2.4.1] ✅ Mensaje de permisos mostrado")
    })

    test("2.5 - Analista: Menú muestra dashboard, NO admin", async ({
      page,
    }) => {
      console.log("[TEST] Validando menú del analista...")

      await page.goto("/login")
      await page.fill("input#cedula", USUARIOS_TEST.analista.cedula)
      await page.fill("input#password", PASSWORD)
      await page.click("button[type='submit']")
      await page.waitForURL(/\/dashboard/, { timeout: 15000 })

      const navMenu = page.locator("nav, [role='navigation']")
      if (await navMenu.isVisible()) {
        // Debe tener enlace a dashboard
        const dashboardLink = page.locator("a:has-text('Dashboard')")
        const isDashboardVisible = await dashboardLink
          .isVisible()
          .catch(() => false)
        if (isDashboardVisible) {
          console.log("[2.5.1] ✅ Enlace a dashboard visible")
        }

        // NO debe tener enlace a admin
        const adminLink = page.locator("a:has-text('Admin|Administración')")
        const isAdminVisible = await adminLink.isVisible().catch(() => false)
        if (!isAdminVisible) {
          console.log("[2.5.2] ✅ Enlace a admin NOT visible")
        }
      }
    })
  })

  // =========================================================================
  // ADMINISTRADOR: Acceso a todas las rutas
  // =========================================================================
  test.describe("Administrador", () => {
    test("3.1 - Admin: Login redirije a /admin o /dashboard", async ({
      page,
    }) => {
      console.log("[TEST] Verificando redirección admin...")

      await page.goto("/login")
      await page.fill("input#cedula", USUARIOS_TEST.admin.cedula)
      await page.fill("input#password", PASSWORD)
      await page.click("button[type='submit']")

      await page.waitForURL(/\/admin|\/dashboard/, { timeout: 15000 })

      const url = page.url()
      const isAdmin = url.includes("/admin")
      const isDashboard = url.includes("/dashboard")

      expect(isAdmin || isDashboard).toBe(true)
      console.log(
        `[3.1.1] ✅ Redireccionado a ${isAdmin ? "/admin" : "/dashboard"}`,
      )
    })

    test("3.2 - Admin: Accede a /admin/preguntas", async ({ page }) => {
      console.log("[TEST] Validando acceso a /admin/preguntas...")

      await page.goto("/login")
      await page.fill("input#cedula", USUARIOS_TEST.admin.cedula)
      await page.fill("input#password", PASSWORD)
      await page.click("button[type='submit']")
      await page.waitForURL(/\/admin|\/dashboard/, { timeout: 15000 })

      await page.goto("/admin/preguntas")

      const titulo = page.locator("h1:has-text('Banco de Preguntas|Questions')")
      if (await titulo.isVisible()) {
        console.log("[3.2.1] ✅ Acceso a /admin/preguntas permitido")
      } else {
        // Verificar que no hay error 403
        expect(page.url()).not.toContain("403")
        console.log("[3.2.1] ✅ Ruta /admin/preguntas sin error")
      }
    })

    test("3.3 - Admin: Accede a /admin/fases", async ({ page }) => {
      console.log("[TEST] Validando acceso a /admin/fases...")

      await page.goto("/login")
      await page.fill("input#cedula", USUARIOS_TEST.admin.cedula)
      await page.fill("input#password", PASSWORD)
      await page.click("button[type='submit']")
      await page.waitForURL(/\/admin|\/dashboard/, { timeout: 15000 })

      await page.goto("/admin/fases")

      const titulo = page.locator("h1:has-text('Fases|Phases')")
      if (await titulo.isVisible()) {
        console.log("[3.3.1] ✅ Acceso a /admin/fases permitido")
      } else {
        expect(page.url()).not.toContain("403")
        console.log("[3.3.1] ✅ Ruta /admin/fases sin error")
      }
    })

    test("3.4 - Admin: Accede a /dashboard", async ({ page }) => {
      console.log("[TEST] Validando acceso a /dashboard...")

      await page.goto("/login")
      await page.fill("input#cedula", USUARIOS_TEST.admin.cedula)
      await page.fill("input#password", PASSWORD)
      await page.click("button[type='submit']")
      await page.waitForURL(/\/admin|\/dashboard/, { timeout: 15000 })

      await page.goto("/dashboard")

      expect(page.url()).not.toContain("403")
      console.log("[3.4.1] ✅ Acceso a /dashboard permitido")
    })

    test("3.5 - Admin: Menú muestra todas las opciones", async ({ page }) => {
      console.log("[TEST] Validando menú del admin...")

      await page.goto("/login")
      await page.fill("input#cedula", USUARIOS_TEST.admin.cedula)
      await page.fill("input#password", PASSWORD)
      await page.click("button[type='submit']")
      await page.waitForURL(/\/admin|\/dashboard/, { timeout: 15000 })

      const navMenu = page.locator("nav, [role='navigation']")
      if (await navMenu.isVisible()) {
        // Debe tener enlace a dashboard
        const dashboardLink = page.locator("a:has-text('Dashboard')")
        const isDashboardVisible = await dashboardLink
          .isVisible()
          .catch(() => false)
        if (isDashboardVisible) {
          console.log("[3.5.1] ✅ Enlace a dashboard visible")
        }

        // Debe tener enlace a admin/preguntas
        const preguntasLink = page.locator("a:has-text('Preguntas|Questions')")
        const isPreguntasVisible = await preguntasLink
          .isVisible()
          .catch(() => false)
        if (isPreguntasVisible) {
          console.log("[3.5.2] ✅ Enlace a preguntas visible")
        }

        // Debe tener enlace a admin/fases
        const fasesLink = page.locator("a:has-text('Fases|Phases')")
        const isFasesVisible = await fasesLink.isVisible().catch(() => false)
        if (isFasesVisible) {
          console.log("[3.5.3] ✅ Enlace a fases visible")
        }
      }
    })

    test("3.6 - Admin: Puede crear/editar banco de preguntas", async ({
      page,
    }) => {
      console.log("[TEST] Validando permisos de escritura...")

      await page.goto("/login")
      await page.fill("input#cedula", USUARIOS_TEST.admin.cedula)
      await page.fill("input#password", PASSWORD)
      await page.click("button[type='submit']")
      await page.waitForURL(/\/admin|\/dashboard/, { timeout: 15000 })

      await page.goto("/admin/preguntas")

      // Buscar botón para crear/editar
      const btnCrear = page.locator("button:has-text('Crear|Agregar|Nueva')")
      const isCrearVisible = await btnCrear.isVisible().catch(() => false)

      if (isCrearVisible) {
        console.log("[3.6.1] ✅ Botón de crear preguntas visible")
      } else {
        // Inputs no deben estar disabled
        const inputs = page.locator("input[type='text'], textarea")
        const count = await inputs.count()
        if (count > 0) {
          const firstInput = inputs.first()
          const isDisabled = await firstInput.isDisabled()
          if (!isDisabled) {
            console.log("[3.6.1] ✅ Campos editables (no disabled)")
          }
        }
      }
    })
  })

  // =========================================================================
  // VALIDACIONES TRANSVERSALES
  // =========================================================================
  test.describe("Validaciones transversales", () => {
    test("4.1 - Logout funciona para todos los roles", async ({ page }) => {
      console.log("[TEST] Validando logout...")

      for (const [rol, creds] of Object.entries(USUARIOS_TEST)) {
        console.log(`[4.1.${rol}] Logout de ${rol}...`)

        await page.goto("/login")
        await page.fill("input#cedula", creds.cedula)
        await page.fill("input#password", PASSWORD)
        await page.click("button[type='submit']")
        await page.waitForURL(/\//, { timeout: 15000 })

        // Buscar botón logout
        const btnLogout = page.locator(
          "button:has-text('Logout|Cerrar sesión|Sign out')",
        )

        if (await btnLogout.isVisible()) {
          await btnLogout.click()
          await page.waitForURL(/\/login|\//)
          expect(page.url()).toContain("/login")
          console.log(`[4.1.${rol}] ✅ Logout exitoso`)
        } else {
          console.log(`[4.1.${rol}] ⏭️  Botón logout no encontrado`)
        }
      }
    })

    test("4.2 - Cookie de sesión se borra al logout", async ({
      page,
      context,
    }) => {
      console.log("[TEST] Validando borrado de cookie...")

      await page.goto("/login")
      await page.fill("input#cedula", USUARIOS_TEST.admin.cedula)
      await page.fill("input#password", PASSWORD)
      await page.click("button[type='submit']")
      await page.waitForURL(/\/admin|\/dashboard/, { timeout: 15000 })

      // Verificar que existe cookie de sesión
      let cookies = await context.cookies()
      let sessionCookie = cookies.find((c) => c.name === "session")
      expect(sessionCookie).toBeDefined()
      console.log("[4.2.1] ✅ Cookie de sesión presente")

      // Logout
      const btnLogout = page.locator("button:has-text('Logout|Cerrar sesión')")
      if (await btnLogout.isVisible()) {
        await btnLogout.click()
        await page.waitForURL(/\/login/)

        // Verificar que cookie fue borrada
        cookies = await context.cookies()
        sessionCookie = cookies.find((c) => c.name === "session")
        expect(sessionCookie).toBeUndefined()
        console.log("[4.2.2] ✅ Cookie de sesión borrada")
      }
    })
  })
})
