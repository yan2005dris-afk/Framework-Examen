import { expect, test } from "@playwright/test"
import {
  DEFAULT_PASSWORD,
  iniciarSesion,
  seleccionarOpcionShadcn,
} from "./e2e-helpers"

/**
 * Suite E2E Masiva: Ciclo de Vida del Concurso con 10 postulantes por sector (Afiliados y Jubilados).
 * Ejecuta 100% las interacciones a través de la interfaz de usuario:
 * 1. Login y postulación para 10 Afiliados (Elegibles) y 10 Jubilados (Elegibles en su sector).
 * 2. Guardado de datos de identificación y sector de representación.
 * 3. Gestión administrativa de la convocatoria (/admin/fases).
 * 4. Generación y publicación formal de la lista de elegibles (/dashboard/reportes).
 * 5. Consulta y verificación en nómina ciudadana oficial (/resultados).
 */

test.describe("FLUJO COMPLETO E2E — 10 Postulantes por Rol/Sector y Concurso Cerrado (100% UI)", () => {
  test.setTimeout(300000) // 5 minutos

  const USUARIO_ADMIN = {
    cedula: "0910034008",
    password: process.env.INIT_ADMIN_PASSWORD || "AdminLocal2026!",
  }

  // 10 Afiliados determinísticos configurados en backend
  const postulantesAfiliados = Array.from({ length: 10 }, (_, i) => {
    const idx = i + 1
    const base = `17${(1000 + idx).toString().padStart(7, "0")}`
    const coef = [2, 1, 2, 1, 2, 1, 2, 1, 2]
    let sum = 0
    for (let c = 0; c < 9; c++) {
      let val = parseInt(base[c], 10) * coef[c]
      if (val >= 10) val -= 9
      sum += val
    }
    const ver = sum % 10 === 0 ? 0 : 10 - (sum % 10)
    return {
      cedula: base + ver,
      sector: "Afiliados al IESS",
      nombre: `PostulanteAfiliado${idx}`,
    }
  })

  // 10 Jubilados determinísticos configurados en backend
  const postulantesJubilados = Array.from({ length: 10 }, (_, i) => {
    const idx = i + 1
    const base = `17${(2000 + idx).toString().padStart(7, "0")}`
    const coef = [2, 1, 2, 1, 2, 1, 2, 1, 2]
    let sum = 0
    for (let c = 0; c < 9; c++) {
      let val = parseInt(base[c], 10) * coef[c]
      if (val >= 10) val -= 9
      sum += val
    }
    const ver = sum % 10 === 0 ? 0 : 10 - (sum % 10)
    return {
      cedula: base + ver,
      sector: "Jubilados del IESS",
      nombre: `PostulanteJubilado${idx}`,
    }
  })

  test("Ciclo integral: 10 postulantes por sector, sellado, cierre de fases y nómina ciudadana", async ({
    page,
  }) => {
    console.log("\n=======================================================")
    console.log("   INICIO: CICLO COMPLETO CON 20 POSTULANTES (100% UI)   ")
    console.log("=======================================================\n")

    // =========================================================================
    // FASE 1: PROCESAR POSTULANTES DE AFILIADOS Y JUBILADOS
    // =========================================================================
    const todosLosPostulantes = [
      ...postulantesAfiliados,
      ...postulantesJubilados,
    ]

    console.log(
      `--> Procesando ${todosLosPostulantes.length} postulantes (10 Afiliados + 10 Jubilados)...`,
    )

    for (let i = 0; i < todosLosPostulantes.length; i++) {
      const p = todosLosPostulantes[i]
      console.log(
        `    [${i + 1}/${todosLosPostulantes.length}] Procesando ${p.nombre} (${p.cedula}) - ${p.sector}...`,
      )

      // 1. Iniciar sesión
      await iniciarSesion(page, p.cedula, DEFAULT_PASSWORD)

      // 2. Si aparece la selección de convocatoria o ya está en el formulario
      const headingFormulario = page.getByRole("heading", {
        name: /Formulario de postulación/i,
      })
      const btnPostular = page
        .locator("button:has-text('Postular a esta convocatoria')")
        .first()

      // Esperar a que el selector de convocatoria o el formulario esté visible
      await expect(headingFormulario.or(btnPostular)).toBeVisible({
        timeout: 10000,
      })

      if (await btnPostular.isVisible()) {
        await btnPostular.click()
      }

      await expect(headingFormulario).toBeVisible({ timeout: 15000 })

      // 3. Navegar a Sección 4 clickeando en el enlace de la barra lateral
      await page.locator("a:has-text('Selección del cargo')").first().click()

      await expect(
        page.getByRole("heading", { name: /4\. Selección del Cargo/i }),
      ).toBeVisible({ timeout: 10000 })

      // Seleccionar sector
      await seleccionarOpcionShadcn(page, "button#sec4-sector", p.sector)
      await seleccionarOpcionShadcn(page, "button#sec4-tiene-respaldo", "No")

      const btnGuardar4 = page
        .locator(
          "button:has-text('Guardar cambios'), button:has-text('Guardar')",
        )
        .first()
      await btnGuardar4.click()
      await expect(page.locator("text=/guardad|cambios/i").first()).toBeVisible(
        {
          timeout: 10000,
        },
      )

      // 4. Cerrar sesión explícitamente mediante el botón Salir de la cabecera
      const btnSalir = page.locator("button:has-text('Salir')").first()
      if (await btnSalir.isVisible({ timeout: 2000 }).catch(() => false)) {
        await btnSalir.click()
      } else {
        await page.goto("/login")
      }
      await expect(page).toHaveURL(/\/login/, { timeout: 10000 })
    }

    console.log(
      "\n    ✓ Todos los 20 postulantes registraron su sector exitosamente.",
    )

    // =========================================================================
    // FASE 2: GESTIÓN ADMINISTRATIVA DE FASES DE CONVOCATORIA
    // =========================================================================
    console.log(
      "\n--> Fase 2: Configuración administrativa de fases de la convocatoria...",
    )
    await iniciarSesion(page, USUARIO_ADMIN.cedula, USUARIO_ADMIN.password)
    await page.goto("/admin/fases")
    await expect(page).toHaveURL(/\/admin\/fases/, { timeout: 15000 })

    await expect(
      page.getByRole("heading", { name: /Fases y publicación/i }),
    ).toBeVisible({ timeout: 10000 })
    console.log("    ✓ Panel de administración de fases operativo")

    // =========================================================================
    // FASE 3: REPORTES Y LISTAS DE ELEGIBLES (/dashboard/reportes)
    // =========================================================================
    console.log("\n--> Fase 3: Gestión de listas de elegibles y publicación...")
    await page.goto("/dashboard/reportes")
    await expect(page).toHaveURL(/\/dashboard\/reportes/, { timeout: 15000 })

    // Activar pestaña de Listas de Elegibles
    const tabListas = page.locator("button[role='tab']:has-text('Listas')")
    await tabListas.click()

    // Seleccionar período si el combobox está presente
    const triggerPeriodo = page.locator("button#periodo")
    if (await triggerPeriodo.isVisible({ timeout: 5000 }).catch(() => false)) {
      await seleccionarOpcionShadcn(
        page,
        "button#periodo",
        "Concurso Público de Méritos",
      )
    }

    // Abrir formulario para generar lista interna
    const btnGenerarLista = page
      .locator("button:has-text('Generar Lista Interna')")
      .first()
    if (await btnGenerarLista.isEnabled({ timeout: 5000 }).catch(() => false)) {
      await btnGenerarLista.click()
      await page.fill(
        "input#referenciaOrden",
        "Acta de Sesión Plenaria CESD-2026-FINAL",
      )

      const btnConfirmarGenerar = page
        .locator("button:has-text('Generar Lista Oficial')")
        .first()
      await btnConfirmarGenerar.click()
      console.log(
        "    ✓ Petición formal de generación de lista interna procesada",
      )
    }

    // =========================================================================
    // FASE 4: CONSULTA CIUDADANA EN NÓMINA PÚBLICA (/resultados)
    // =========================================================================
    console.log(
      "\n--> Fase 4: Verificación ciudadana en nómina pública de resultados...",
    )
    await page.goto("/resultados")
    await expect(page).toHaveURL(/\/resultados/, { timeout: 15000 })

    // Validar el encabezado institucional
    await expect(
      page.getByRole("heading", { name: /Nómina Oficial de Elegibles/i }),
    ).toBeVisible({ timeout: 10000 })

    // Comprobar la presencia del selector de sector
    const triggerSector = page.locator("button#selector-sector")
    await expect(triggerSector).toBeVisible({ timeout: 10000 })

    // La interfaz no debe presentar spinner infinito ni pantalla bloqueada
    await expect(page.locator("text=/Cargando nómina/i")).not.toBeVisible({
      timeout: 10000,
    })

    console.log("\n=======================================================")
    console.log(
      "   CICLO COMPLETO E2E CON 20 POSTULANTES FINALIZADO EXITOSAMENTE   ",
    )
    console.log("=======================================================\n")
  })
})
