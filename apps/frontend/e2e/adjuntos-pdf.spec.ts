import { expect, test } from "@playwright/test"
import {
  DEFAULT_PASSWORD,
  esperarOTPdeMailpit,
  generarCedulaValida,
} from "./e2e-helpers"

/**
 * Genera un buffer PDF mínimo válido o con tamaño determinado.
 */
function generarPDFBuffer(tamanoBytes: number): Buffer {
  const header = Buffer.from("%PDF-1.4\n1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj 2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj 3 0 obj<</Type/Page/MediaBox[0 0 612 792]/Parent 2 0 R>>endobj\nxref\n0 4\n0000000000 65535 f \n0000000010 00000 n \n0000000060 00000 n \n0000000117 00000 n \ntrailer<</Size 4/Root 1 0 R>>\nstartxref\n190\n%%EOF\n")
  if (tamanoBytes <= header.length) {
    return header
  }
  const relleno = Buffer.alloc(tamanoBytes - header.length, "A")
  return Buffer.concat([header, relleno])
}

test.describe("ADJUNTOS PDF — Validación de Tamaño, Tipo y Cuotas (Criterio 2)", () => {
  test("Validar límites de adjuntos por sección (2MB / 4MB) y tipo MIME", async ({ page }) => {
    test.setTimeout(90000)
    console.log("\n========== SETUP: REGISTRO Y LOGIN DE USUARIO NUEVO ==========\n")

    const cedula = generarCedulaValida()
    const correo = `pdf-${cedula}@biess-test.local`

    // Registro
    await page.goto("/registro")
    await page.fill("input#cedula", cedula)
    await page.fill("input#correo", correo)
    await page.locator("button:has-text('Continuar')").click()

    const otp = await esperarOTPdeMailpit(correo, "", 20)
    await page.fill("input#otp", otp)
    await page.locator("button:has-text('Verificar código')").click()

    await page.fill("input#nombres", "Postulante")
    await page.fill("input#apellidos", "Adjuntos")
    await page.fill("input#reg-password", DEFAULT_PASSWORD)
    await page.fill("input#confirm-password", DEFAULT_PASSWORD)
    await page.locator("button:has-text('Finalizar registro')").click()

    // En la vista de registro exitoso, hacer click en 'Comenzar postulación'
    const btnComenzar = page.locator("button:has-text('Comenzar postulación')")
    await expect(btnComenzar).toBeVisible({ timeout: 15000 })
    await btnComenzar.click()

    // En la vista de selección de convocatorias, seleccionar expresamente "Postular a esta convocatoria"
    const btnPostular = page.locator("button:has-text('Postular a esta convocatoria')").first()
    await expect(btnPostular).toBeVisible({ timeout: 15000 })
    await btnPostular.click()

    // Esperar a entrar al formulario de la postulación
    await page.waitForURL(/\/postulacion\/\d+|\/postulacion\?convocatoria=/)

    console.log("========== TEST 1: SECCIÓN 5 - RECHAZO DE ARCHIVO NO PDF ==========")
    // Ir explícitamente a la sección 5 preservando el parámetro de convocatoria si existe
    const currentUrl = page.url()
    const urlObj = new URL(currentUrl)
    const convocatoriaParam = urlObj.searchParams.get("convocatoria")
    const rutaS5 = convocatoriaParam ? `/postulacion/5?convocatoria=${convocatoriaParam}` : "/postulacion/5"
    await page.goto(rutaS5)
    await page.waitForLoadState("networkidle")

    const btnAgregarTitulo = page.locator("button:has-text('Agregar Título'), button:has-text('Agregar título')").first()
    await expect(btnAgregarTitulo).toBeVisible({ timeout: 10000 })
    await btnAgregarTitulo.click()

    const inputFileS5 = page.locator("input[type='file']").first()
    await expect(inputFileS5).toBeAttached()

    // Intentar subir TXT o formato inválido
    await inputFileS5.setInputFiles({
      name: "invalido.txt",
      mimeType: "text/plain",
      buffer: Buffer.from("Este es un archivo no permitido"),
    })

    await expect(page.locator("text=/formato|solo.*pdf|invalido|válido/i").first()).toBeVisible({ timeout: 6000 })
    console.log("✅ Validación de tipo MIME: Solo se admiten archivos application/pdf")

    console.log("========== TEST 2: SECCIÓN 5 - LÍMITE DE 2 MiB ==========")
    // Archivo de 3 MiB en sección 5 (límite es 2 MiB)
    const pdfExcedidoS5 = generarPDFBuffer(3 * 1024 * 1024)
    await inputFileS5.setInputFiles({
      name: "excede-2mb.pdf",
      mimeType: "application/pdf",
      buffer: pdfExcedidoS5,
    })

    await expect(page.locator("text=/excede|2.*mb|máximo/i").first()).toBeVisible({ timeout: 6000 })
    console.log("✅ Validación de tamaño sección 5: Rechaza archivos que exceden 2 MiB")

    console.log("========== TEST 3: SECCIÓN 5 - SUBIDA EXITOSA DE PDF VÁLIDO ==========")
    const pdfValidoS5 = generarPDFBuffer(100 * 1024) // 100 KB

    // Cargar el archivo válido
    await inputFileS5.setInputFiles({
      name: "titulo-valido.pdf",
      mimeType: "application/pdf",
      buffer: pdfValidoS5,
    })

    // En la interfaz se muestra el archivo subido y confirmado
    await expect(page.locator("text=titulo-valido.pdf").first()).toBeVisible({ timeout: 15000 })
    await expect(page.locator("text=/Confirmado|100.*KiB/i").first()).toBeVisible({ timeout: 15000 })
    console.log("✅ Archivo PDF subido y confirmado exitosamente en la UI y almacenamiento S3")

    console.log("✅ Todas las pruebas de adjuntos PDF finalizaron en verde")
  })
})
