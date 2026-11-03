import { expect, test } from "@playwright/test"
import { DEFAULT_PASSWORD, iniciarSesion } from "./e2e-helpers"

/**
 * Test Criterio 4: Vista de Impresión y Firma Manuscrita (HU-11 / Mockup 10)
 * Valida:
 * 1. Acceso a /postulacion/imprimir con postulación enviada y sellada.
 * 2. Visualización de cabecera con código único de postulación.
 * 3. Identificación del postulante (nombres, apellidos, cédula).
 * 4. Presencia de las 9 secciones estructuradas en el informe imprimible.
 * 5. Área formal de firma manuscrita ("Firma del Postulante").
 * 6. Presencia del Hash criptográfico SHA-256 en el pie del documento.
 */

test.describe("IMPRESIÓN Y FIRMA MANUSCRITA — Criterio 4 (6pts)", () => {
  test("Vista imprimible con datos completos, código, firma y hash SHA-256", async ({
    page,
  }) => {
    test.setTimeout(60000)
    console.log("\n========== INICIO PRUEBA IMPRESIÓN Y FIRMA ==========\n")

    // Iniciar sesión con el postulante que ya tiene su postulación enviada y sellada
    await iniciarSesion(page, "0926914227", DEFAULT_PASSWORD)

    // Navegar a la vista de impresión
    await page.goto("/postulacion/imprimir")
    await page.waitForLoadState("networkidle")

    console.log("========== VALIDACIÓN DE ESTRUCTURA Y ENCABEZADO ==========")
    // 1. Título principal del documento de postulación
    await expect(
      page
        .locator("text=/formulario de postulación|resumen de postulación/i")
        .first(),
    ).toBeVisible({ timeout: 15000 })
    console.log("✅ Título del formulario imprimible visible")

    // 2. Datos personales del postulante
    await expect(page.locator("text=0926914227").first()).toBeVisible()
    console.log("✅ Cédula del postulante visible en la hoja de impresión")

    // 3. Código único de postulación (ej: EF12-2026-...)
    const textoCodigo = page.locator("text=/EF12-\\d{4}-[A-Z0-9]+/").first()
    await expect(textoCodigo).toBeVisible()
    const codigoEncontrado = await textoCodigo.textContent()
    console.log(
      `✅ Código único de postulación en impresión: ${codigoEncontrado?.trim()}`,
    )

    // 4. Hash SHA-256 canónico sellado (64 caracteres hexadecimales)
    const textoHash = page.locator("text=/[a-f0-9]{64}/").first()
    await expect(textoHash).toBeVisible()
    const hashEncontrado = await textoHash.textContent()
    console.log(
      `✅ Hash SHA-256 canónico visible en pie de documento: ${hashEncontrado?.trim().substring(0, 32)}...`,
    )

    // 5. Presencia de las secciones principales
    console.log("========== VALIDACIÓN DE SECCIONES IMPRIMIBLES ==========")
    const seccionesClave = [
      "Información Personal",
      "Información Domiciliaria",
      "Información Adicional",
      "Cargo",
      "Formación",
      "Experiencia",
      "Inhabilidades",
    ]

    for (const seccion of seccionesClave) {
      await expect(page.locator(`text=/${seccion}/i`).first()).toBeVisible()
      console.log(`✅ Sección "${seccion}" verificada en reporte`)
    }

    // 6. Área para firma manuscrita
    console.log("========== VALIDACIÓN DE ÁREA DE FIRMA MANUSCRITA ==========")
    const areaFirma = page
      .locator("text=/firma del postulante|firma manuscrita|firma y entrega/i")
      .first()
    await expect(areaFirma).toBeVisible()
    console.log("✅ Área de firma manuscrita y entrega física verificada")

    // 7. Botones de acción (Imprimir / Volver)
    const btnImprimir = page
      .locator("button:has-text('Imprimir'), button:has-text('Descargar PDF')")
      .first()
    await expect(btnImprimir).toBeVisible()
    console.log("✅ Botón de impresión/descarga disponible")
  })
})
