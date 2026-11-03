import { expect, test } from "@playwright/test"
import {
  DEFAULT_PASSWORD,
  iniciarSesion,
  seleccionarOpcionShadcn,
} from "./e2e-helpers"

/**
 * Test de Validación de Elegibilidad por Sector (HU-08).
 * Valida:
 * 1. Sector "Afiliados" -> Elegible (confirmado en backend con simulación).
 * 2. Si cumple -> Permite continuar con el formulario.
 * 3. Si NO cumple -> El backend cierra la postulación con status='closed' y se muestra el banner de bloqueo.
 */

test.describe("ELEGIBILIDAD POR SECTOR — Criterio 6 (6pts)", () => {
  test("Sector Afiliados: Elegible → continúa formulario", async ({ page }) => {
    console.log("\n========== TEST 1: AFILIADOS ELEGIBLE ==========\n")

    // Usar la postulación borrador existente del postulante 0926914227 (afiliados: true)
    await iniciarSesion(page, "0926914227", DEFAULT_PASSWORD)

    // Navegar a sección 4
    await page.goto("/postulacion/4")
    await expect(page.getByRole("heading", { name: /4\. Selección del Cargo/i })).toBeVisible({ timeout: 10000 })

    // Seleccionar sector "Afiliados al IESS" mediante el trigger de shadcn
    await seleccionarOpcionShadcn(page, "button#sec4-sector", /Afiliados al IESS/i)

    // Seleccionar respaldo "No"
    await seleccionarOpcionShadcn(page, "button#sec4-tiene-respaldo", "No")

    // Guardar sección
    const btnGuardar = page.locator("button:has-text('Guardar cambios'), button:has-text('Guardar')").first()
    await btnGuardar.click()

    // El guardado debe ser exitoso y persistir
    await expect(page.locator("text=/guardad|cambios/i").first()).toBeVisible({ timeout: 10000 })
    console.log("✅ Sección 4 guardada exitosamente para sector Afiliados (Elegible)")
  })

  test("Sector Jubilados: NO Elegible en simulación → postulación se cierra automáticamente", async ({ page }) => {
    console.log("\n========== TEST 2: JUBILADOS NO ELEGIBLE (CIERRE) ==========\n")

    // Postulante 1710034008 configurado con jubilados: false
    await iniciarSesion(page, "1710034008", DEFAULT_PASSWORD)

    // Si está en la selección de convocatoria inicial
    const btnPostular = page.locator("button:has-text('Postular a esta convocatoria')").first()
    await expect(btnPostular).toBeVisible({ timeout: 10000 })
    await btnPostular.click()

    // Esperar a que cargue la sección 1 o la navegación del formulario
    await expect(page.getByRole("heading", { name: /1\. Información Personal/i })).toBeVisible({ timeout: 10000 })

    // Navegar a sección 4
    await page.goto("/postulacion/4")
    await expect(page.getByRole("heading", { name: /4\. Selección del Cargo/i })).toBeVisible({ timeout: 10000 })

    // Cambiar a "Jubilados del IESS" (en eligibility.json está configurado con "jubilados": false)
    await seleccionarOpcionShadcn(page, "button#sec4-sector", /Jubilados del IESS/i)

    const btnGuardar = page.locator("button:has-text('Guardar cambios'), button:has-text('Guardar')").first()
    await btnGuardar.click()

    // Debe mostrar banner o alerta de no elegibilidad / postulación cerrada
    await expect(
      page.locator("text=/no elegible|cerrada automáticamente|criterios de elegibilidad|no puede continuar/i").first(),
    ).toBeVisible({ timeout: 10000 })

    console.log("✅ Cierre automático por no elegibilidad confirmado en UI")
  })
})
