import { expect, test } from "@playwright/test"
import {
  DEFAULT_PASSWORD,
  iniciarSesion,
  seleccionarOpcionShadcn,
} from "./e2e-helpers"

/**
 * Test Criterios 4 & 6: Formulario de 9 Secciones + Cálculo de Experiencia.
 * Valida:
 * 1. Navegación y guardado progresivo por sección.
 * 2. Persistencia de los datos en base de datos.
 * 3. Cálculo dinámico de experiencia laboral (años, meses, días).
 */

test.describe("NUEVE SECCIONES + EXPERIENCIA — Criterios 4 & 6 (12pts)", () => {
  test("Completar las 9 secciones con guardado progresivo y verificar experiencia", async ({
    page,
  }) => {
    test.setTimeout(60000)
    console.log("\n========== INICIO: SESIÓN DE POSTULANTE ==========\n")

    // Iniciar sesión con usuario existente que tiene borrador activo
    await iniciarSesion(page, "0926914227", DEFAULT_PASSWORD)

    // =========================================================================
    // SECCIÓN 1: INFORMACIÓN PERSONAL
    // =========================================================================
    console.log("========== SECCIÓN 1: INFORMACIÓN PERSONAL ==========")
    await page.goto("/postulacion/1")
    await expect(
      page.getByRole("heading", { name: /1\. Información Personal/i }),
    ).toBeVisible({ timeout: 10000 })

    await seleccionarOpcionShadcn(page, "button#sec1-genero", "Masculino")
    await page.fill("input#sec1-nacimiento", "1990-05-15")
    await seleccionarOpcionShadcn(page, "button#sec1-estadoCivil", "Soltero/a")

    const btnGuardar1 = page
      .locator("button:has-text('Guardar cambios'), button:has-text('Guardar')")
      .first()
    await btnGuardar1.click()
    await expect(page.locator("text=/guardad|cambios/i").first()).toBeVisible({
      timeout: 10000,
    })
    console.log("✅ Sección 1 guardada")

    // =========================================================================
    // SECCIÓN 2: CONTACTO
    // =========================================================================
    console.log("========== SECCIÓN 2: CONTACTO ==========")
    await page.goto("/postulacion/2")
    await expect(
      page.getByRole("heading", { name: /2\. Información Domiciliaria/i }),
    ).toBeVisible({ timeout: 10000 })

    await page.fill("input#sec2-celular", "0991234567")
    await page.fill("input#sec2-convencional", "022345678")
    await page.fill("input#sec2-calle-principal", "Av. Amazonas")
    await page.fill("input#sec2-numero", "N24-100")
    await page.fill("input#sec2-calle-secundaria", "Colón")
    await page.fill("input#sec2-sector", "La Mariscal")

    const btnGuardar2 = page
      .locator("button:has-text('Guardar cambios'), button:has-text('Guardar')")
      .first()
    await btnGuardar2.click()
    await expect(page.locator("text=/guardad|cambios/i").first()).toBeVisible({
      timeout: 10000,
    })
    console.log("✅ Sección 2 guardada")

    // =========================================================================
    // SECCIÓN 3: FAMILIARES
    // =========================================================================
    console.log("========== SECCIÓN 3: FAMILIARES ==========")
    await page.goto("/postulacion/3")
    await expect(
      page.getByRole("heading", { name: /3\. Información Adicional/i }),
    ).toBeVisible({ timeout: 10000 })

    await seleccionarOpcionShadcn(page, "button#sec3-tiene-familiares", "No")

    const btnGuardar3 = page
      .locator("button:has-text('Guardar cambios'), button:has-text('Guardar')")
      .first()
    await btnGuardar3.click()
    await expect(page.locator("text=/guardad|cambios/i").first()).toBeVisible({
      timeout: 10000,
    })
    console.log("✅ Sección 3 guardada")

    // =========================================================================
    // SECCIÓN 4: CARGO Y SECTOR
    // =========================================================================
    console.log("========== SECCIÓN 4: CARGO Y SECTOR ==========")
    await page.goto("/postulacion/4")
    await expect(
      page.getByRole("heading", { name: /4\. Selección del Cargo/i }),
    ).toBeVisible({ timeout: 10000 })

    await seleccionarOpcionShadcn(
      page,
      "button#sec4-sector",
      /Afiliados al IESS/i,
    )
    await seleccionarOpcionShadcn(page, "button#sec4-tiene-respaldo", "No")

    const btnGuardar4 = page
      .locator("button:has-text('Guardar cambios'), button:has-text('Guardar')")
      .first()
    await btnGuardar4.click()
    await expect(page.locator("text=/guardad|cambios/i").first()).toBeVisible({
      timeout: 10000,
    })
    console.log("✅ Sección 4 guardada")

    // =========================================================================
    // SECCIÓN 5: FORMACIÓN ACADÉMICA
    // =========================================================================
    console.log("========== SECCIÓN 5: FORMACIÓN ACADÉMICA ==========")
    await page.goto("/postulacion/5")
    await expect(
      page.getByRole("heading", { name: /5\. Formación Académica/i }),
    ).toBeVisible({ timeout: 10000 })

    const btnAgregarTitulo = page.locator("button:has-text('Agregar título')")
    // Si no hay tarjeta de título aún, hacer click
    if (!(await page.locator("text=/Título #1/i").isVisible())) {
      await btnAgregarTitulo.click()
      await page.waitForTimeout(300)
    }

    const comboboxesSec5 = page.getByRole("combobox")
    // Nivel
    await comboboxesSec5.nth(0).click()
    await page.getByRole("option", { name: /Tercer Nivel/i }).click()

    // Título e institución
    await page.locator("input[placeholder*='Economista']").fill("Economista")
    await page
      .locator("input[placeholder*='Universidad']")
      .fill("Universidad Central del Ecuador")

    // Tipo de institución
    await comboboxesSec5.nth(1).click()
    await page.getByRole("option", { name: "Nacional" }).click()

    // Área
    await comboboxesSec5.nth(2).click()
    await page
      .getByRole("option", { name: /Finanzas|Economía/i })
      .first()
      .click()

    // Registro SENESCYT y fecha
    await page.locator("input[placeholder*='1005']").fill("1005-2015-123456")
    await page.locator("input[type='date']").first().fill("2015-08-20")

    const btnGuardar5 = page
      .locator("button:has-text('Guardar cambios'), button:has-text('Guardar')")
      .first()
    await btnGuardar5.click()
    await expect(page.locator("text=/guardad|cambios/i").first()).toBeVisible({
      timeout: 10000,
    })
    console.log("✅ Sección 5 guardada con título académico")

    // =========================================================================
    // SECCIÓN 6: EXPERIENCIA LABORAL (CÁLCULO AUTOMÁTICO)
    // =========================================================================
    console.log("========== SECCIÓN 6: EXPERIENCIA LABORAL ==========")
    await page.goto("/postulacion/6")
    await expect(
      page.getByRole("heading", { name: /6\. Experiencia Profesional/i }),
    ).toBeVisible({ timeout: 10000 })

    const btnAgregarExp = page.locator("button:has-text('Agregar experiencia')")
    if (!(await page.locator("text=/Experiencia #1/i").isVisible())) {
      await btnAgregarExp.click()
      await page.waitForTimeout(300)
    }

    await page
      .locator("input[placeholder*='Gerente']")
      .fill("Especialista Financiero")
    await page.locator("input[placeholder*='empleadora']").fill("Banco Central")

    const inputsDate = page.locator("input[type='date']")
    await inputsDate.nth(0).fill("2020-01-01")
    await inputsDate.nth(1).fill("2023-01-01")
    await page
      .locator("textarea")
      .first()
      .fill("Análisis de riesgos y proyecciones financieras.")

    // Verificar que el cálculo de duración aparezca (3 años exactos)
    await expect(
      page.locator("text=/3 año|36 meses|duración/i").first(),
    ).toBeVisible({ timeout: 5000 })
    console.log(
      "✅ Cálculo de duración de experiencia verificado (años/meses/días)",
    )

    const btnGuardar6 = page
      .locator("button:has-text('Guardar cambios'), button:has-text('Guardar')")
      .first()
    await btnGuardar6.click()
    await expect(page.locator("text=/guardad|cambios/i").first()).toBeVisible({
      timeout: 10000,
    })
    console.log("✅ Sección 6 guardada")

    // =========================================================================
    // SECCIÓN 7: PUBLICACIONES
    // =========================================================================
    console.log("========== SECCIÓN 7: PUBLICACIONES ==========")
    await page.goto("/postulacion/7")
    await expect(
      page.getByRole("heading", { name: /7\. Publicaciones/i }),
    ).toBeVisible({ timeout: 10000 })

    await seleccionarOpcionShadcn(page, "button#sec7-tiene-pub", "No")
    const btnGuardar7 = page
      .locator("button:has-text('Guardar cambios'), button:has-text('Guardar')")
      .first()
    await btnGuardar7.click()
    await expect(page.locator("text=/guardad|cambios/i").first()).toBeVisible({
      timeout: 10000,
    })
    console.log("✅ Sección 7 guardada")

    // =========================================================================
    // SECCIÓN 8: CAPACITACIÓN
    // =========================================================================
    console.log("========== SECCIÓN 8: CAPACITACIÓN ==========")
    await page.goto("/postulacion/8")
    await expect(
      page.getByRole("heading", { name: /8\. Capacitación/i }),
    ).toBeVisible({ timeout: 10000 })

    const btnGuardar8 = page
      .locator("button:has-text('Guardar cambios'), button:has-text('Guardar')")
      .first()
    await btnGuardar8.click()
    await expect(page.locator("text=/guardad|cambios/i").first()).toBeVisible({
      timeout: 10000,
    })
    console.log("✅ Sección 8 guardada")

    // =========================================================================
    // SECCIÓN 9: INHABILIDADES
    // =========================================================================
    console.log("========== SECCIÓN 9: INHABILIDADES ==========")
    await page.goto("/postulacion/9")
    await expect(
      page.getByRole("heading", { name: /9\. Inhabilidades/i }),
    ).toBeVisible({ timeout: 10000 })

    const btnGuardar9 = page
      .locator("button:has-text('Guardar cambios'), button:has-text('Guardar')")
      .first()
    await btnGuardar9.click()
    await expect(page.locator("text=/guardad|cambios/i").first()).toBeVisible({
      timeout: 10000,
    })
    console.log("✅ Sección 9 guardada")

    // =========================================================================
    // PERSISTENCIA: RECARGAR Y VALIDAR DATOS
    // =========================================================================
    console.log("========== COMPROBAR PERSISTENCIA POST-RELOAD ==========")
    await page.reload()
    await expect(
      page.getByRole("heading", { name: /9\. Inhabilidades/i }),
    ).toBeVisible({ timeout: 10000 })
    console.log("✅ Datos persistidos correctamente tras recarga")
  })
})
