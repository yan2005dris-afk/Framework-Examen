import { expect, test } from "@playwright/test"

const DEFAULT_PASSWORD = process.env.E2E_PASSWORD || "TestPassword.2026*"

const USUARIOS = {
  admin: {
    cedula: "2407612650",
    password: DEFAULT_PASSWORD,
  },
  analista: {
    cedula: "2402249987",
    password: DEFAULT_PASSWORD,
  },
  aprobado: {
    cedula: "2406425922",
    password: DEFAULT_PASSWORD,
  },
  minimo: {
    cedula: "2402740035",
    password: DEFAULT_PASSWORD,
  },
  cerrado: {
    cedula: "2403401355",
    password: DEFAULT_PASSWORD,
  },
  reconsideracion: {
    cedula: "2408380695",
    password: DEFAULT_PASSWORD,
  },
}

const CONVOCATORIA_ID = "01a1954b-bf4e-75c0-a235-24242ce05127"

test.describe("Portal BIESS - Pruebas E2E Sintéticas Completas", () => {
  test("1. Portada pública y navegación inicial", async ({ page }) => {
    await page.goto("/")
    await expect(page).toHaveTitle(/BIESS/i)
    await expect(page.getByRole("heading", { level: 1 })).toBeVisible()
    await expect(
      page.getByRole("link", { name: /Iniciar sesión/i }),
    ).toBeVisible()
  })

  test("2. Login y navegación de postulante con expediente enviado (aprobado)", async ({
    page,
  }) => {
    await page.goto("/login")
    await page.locator("input#cedula").fill(USUARIOS.aprobado.cedula)
    await page.locator("input#password").fill(USUARIOS.aprobado.password)
    await page.locator("button[type='submit']").click()

    await expect(page).toHaveURL(/\/postulacion/, { timeout: 15000 })
    await expect(page.getByText(/Postulación enviada y sellada/i)).toBeVisible({
      timeout: 15000,
    })
  })

  test("3. Formulario EF-12 y documentos: navegación y cuota de respaldos (minimo)", async ({
    page,
  }) => {
    await page.goto("/login")
    await page.locator("input#cedula").fill(USUARIOS.minimo.cedula)
    await page.locator("input#password").fill(USUARIOS.minimo.password)
    await page.locator("button[type='submit']").click()

    await expect(page).toHaveURL(/\/postulacion/, { timeout: 15000 })
    await expect(page.getByText(/Progreso del Formulario EF-12/i)).toBeVisible({
      timeout: 15000,
    })
    await expect(page.getByText(/Cuota de respaldos PDF/i)).toBeVisible({
      timeout: 15000,
    })

    // Navegación entre secciones
    await page.getByRole("link", { name: /Sec\. 2 Contacto/i }).click()
    await expect(page).toHaveURL(/\/postulacion\/2/, { timeout: 15000 })
    await expect(
      page.getByRole("heading", {
        name: /Información Domiciliaria y de Contacto/i,
      }),
    ).toBeVisible({ timeout: 15000 })
  })

  test("4. Envío e impresión: vista imprimible con hash criptográfico (aprobado)", async ({
    page,
  }) => {
    await page.goto("/login")
    await page.locator("input#cedula").fill(USUARIOS.aprobado.cedula)
    await page.locator("input#password").fill(USUARIOS.aprobado.password)
    await page.locator("button[type='submit']").click()

    await expect(page).toHaveURL(/\/postulacion/, { timeout: 15000 })
    await page.goto("/postulacion/imprimir")
    await expect(page).toHaveURL(/\/postulacion\/imprimir/, { timeout: 15000 })
    await expect(
      page.getByText(/Formulario sellado con Hash Criptográfico/i),
    ).toBeVisible({ timeout: 15000 })
    await expect(
      page.getByRole("button", {
        name: /Imprimir Formulario \(o Guardar PDF\)/i,
      }),
    ).toBeVisible({ timeout: 15000 })
  })

  test("5. Postulación cerrada por no elegibilidad (cerrado)", async ({
    page,
  }) => {
    await page.goto("/login")
    await page.locator("input#cedula").fill(USUARIOS.cerrado.cedula)
    await page.locator("input#password").fill(USUARIOS.cerrado.password)
    await page.locator("button[type='submit']").click()

    await expect(page).toHaveURL(/\/postulacion/, { timeout: 15000 })
    await expect(
      page.getByRole("heading", { name: /La postulación no puede continuar/i }),
    ).toBeVisible({ timeout: 15000 })
  })

  test("6. Solicitudes de reconsideración del postulante", async ({ page }) => {
    await page.goto("/login")
    await page.locator("input#cedula").fill(USUARIOS.reconsideracion.cedula)
    await page.locator("input#password").fill(USUARIOS.reconsideracion.password)
    await page.locator("button[type='submit']").click()

    await expect(page).toHaveURL(/\/postulacion|\/seguimiento/, {
      timeout: 15000,
    })
    await page.goto("/reconsideracion")
    await expect(page).toHaveURL(/\/reconsideracion/, { timeout: 15000 })
    await expect(
      page.locator(
        "div[data-slot='card-title']:has-text('Solicitudes de Reconsideración')",
      ),
    ).toBeVisible({ timeout: 15000 })
  })

  test("7. Banco de preguntas y evaluación técnica (admin)", async ({
    page,
  }) => {
    await page.goto("/login")
    await page.locator("input#cedula").fill(USUARIOS.admin.cedula)
    await page.locator("input#password").fill(USUARIOS.admin.password)
    await page.locator("button[type='submit']").click()

    await expect(page).toHaveURL(/\/admin|\/dashboard/, { timeout: 15000 })
    await page.goto("/admin/preguntas")
    await expect(page).toHaveURL(/\/admin\/preguntas/, { timeout: 15000 })
    await expect(
      page.locator(
        "div[data-slot='card-title']:has-text('Banco de Preguntas')",
      ),
    ).toBeVisible({ timeout: 15000 })
    await expect(
      page.getByText(/Inmutabilidad y Auditoría de Reactivos/i),
    ).toBeVisible({ timeout: 15000 })
  })

  test("8. Examen completo y resultado calificado (aprobado)", async ({
    page,
  }) => {
    await page.goto("/login")
    await page.locator("input#cedula").fill(USUARIOS.aprobado.cedula)
    await page.locator("input#password").fill(USUARIOS.aprobado.password)
    await page.locator("button[type='submit']").click()

    await expect(page).toHaveURL(/\/postulacion/, { timeout: 15000 })
    await page.goto("/evaluacion")
    await expect(page).toHaveURL(/\/evaluacion/, { timeout: 15000 })
    await expect(page.getByText(/Evaluación Calificada/i)).toBeVisible({
      timeout: 15000,
    })
    await expect(
      page.getByRole("button", { name: /Descargar reporte PDF/i }),
    ).toBeVisible({ timeout: 15000 })
  })

  test("9. Privacidad institucional y nómina pública", async ({ page }) => {
    // 9a. Vista administrativa de fases y privacidad
    await page.goto("/login")
    await page.locator("input#cedula").fill(USUARIOS.admin.cedula)
    await page.locator("input#password").fill(USUARIOS.admin.password)
    await page.locator("button[type='submit']").click()

    await expect(page).toHaveURL(/\/admin|\/dashboard/, { timeout: 15000 })
    await page.goto("/admin/fases")
    await expect(page).toHaveURL(/\/admin\/fases/, { timeout: 15000 })
    await expect(
      page.getByRole("heading", { name: /Fases y publicación/i }),
    ).toBeVisible({ timeout: 15000 })

    // 9b. Consulta pública de resultados con consentimiento
    await page.goto("/resultados")
    await expect(page).toHaveURL(/\/resultados/, { timeout: 15000 })
    await expect(
      page.getByRole("heading", { name: /Nómina Oficial de Elegibles/i }),
    ).toBeVisible({ timeout: 15000 })
  })

  test("10. Exportaciones XLSX y PDF en revisión administrativa (analista)", async ({
    page,
  }) => {
    await page.goto("/login")
    await page.locator("input#cedula").fill(USUARIOS.analista.cedula)
    await page.locator("input#password").fill(USUARIOS.analista.password)
    await page.locator("button[type='submit']").click()

    await expect(page).toHaveURL(/\/dashboard/, { timeout: 15000 })
    await page.goto(`/dashboard/postulaciones?convocatoria=${CONVOCATORIA_ID}`)
    await expect(page).toHaveURL(new RegExp(CONVOCATORIA_ID), {
      timeout: 15000,
    })

    await expect(
      page.getByRole("button", { name: /Descargar Excel/i }),
    ).toBeVisible({ timeout: 15000 })
    await expect(
      page.getByRole("button", { name: /Descargar PDF/i }),
    ).toBeVisible({ timeout: 15000 })
  })

  test("11. Control de acceso 403: postulante intenta acceder a /dashboard", async ({
    page,
  }) => {
    await page.goto("/login")
    await page.locator("input#cedula").fill(USUARIOS.aprobado.cedula)
    await page.locator("input#password").fill(USUARIOS.aprobado.password)
    await page.locator("button[type='submit']").click()

    await expect(page).toHaveURL(/\/postulacion/, { timeout: 15000 })
    await page.goto("/dashboard")
    await expect(
      page.getByText(/No tiene permisos para esta sección/i),
    ).toBeVisible({ timeout: 15000 })
  })
})
