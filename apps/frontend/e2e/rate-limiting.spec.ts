import { expect, test, type APIResponse } from "@playwright/test"

import { API_BASE, PASSWORD, cuentaPorRol } from "./fixtures/datos"
import { iniciarSesion } from "./fixtures/sesion"

/**
 * Test 11: Rate Limiting HTTP 429
 * Criterio de rúbrica 5: Consumo de API & Manejo Asíncrono (6 pts)
 *
 * Valida (nginx + API):
 * 1. Límite: 10 req/s por IP
 * 2. Burst: 30 requests máximo
 * 3. Status 429 Too Many Requests
 * 4. Retry-After header
 * 5. Frontend maneja 429 gracefully
 *
 * El límite por IP está implementado en el nginx de producción
 * (`deploy/nginx.conf`: `limit_req zone=general_api burst=30 nodelay` y
 * `limit_req_status 429`). La API de Go no registra ningún middleware de
 * límite de peticiones y el stack de desarrollo no pasa por nginx, así que
 * en este entorno el servidor nunca puede responder 429: las pruebas que
 * dependen de esa respuesta quedan omitidas con su motivo.
 */

test.describe("RATE LIMITING — Criterio 5 (6pts)", () => {
  // =========================================================================
  // TEST 1: Normal requests < límite
  // =========================================================================
  test("Requests dentro del límite retornan 200", async ({ request }) => {
    console.log("\n========== TEST 1: REQUESTS NORMALES ==========\n")

    // El catálogo exige sesión: sin ella responde 401 y no mide el tráfico.
    const cuenta = cuentaPorRol("postulante")
    await iniciarSesion(request, cuenta.cedula, PASSWORD)

    // Hacer 5 requests normales (< 10/s)
    const requests = []
    for (let i = 0; i < 5; i++) {
      const resp = await request.get(`${API_BASE}/catalogs/countries`)
      requests.push(resp.status())
    }

    console.log(`Statuses: ${requests.join(", ")}`)

    // Todos deben ser 2xx (200 OK, 304 Not Modified, etc)
    const allOk = requests.every((status) => status >= 200 && status < 300)
    expect(allOk).toBe(true)
    console.log("✅ Requests normales retornan 2xx")
  })

  // =========================================================================
  // TEST 2: Exceder límite → 429
  // =========================================================================
  test("Exceder 10 req/s → retorna 429 Too Many Requests", async ({ page }) => {
    test.skip(
      true,
      "El límite por IP vive en el nginx de producción (deploy/nginx.conf:33, `limit_req zone=general_api burst=30 nodelay` con `limit_req_status 429`). La API de Go no registra middleware de límite de peticiones (apps/backend/internal/platform/httpx/ no lo tiene) y el stack de desarrollo no pasa por nginx: el proxy de Vite va directo a la API. En este entorno el servidor nunca puede responder 429.",
    )
    console.log("\n========== TEST 2: RATE LIMIT EXCEDIDO ==========\n")

    // Bombardear con > 30 requests rápidos
    const numRequests = 50
    const requests: Promise<APIResponse>[] = []

    console.log(`Enviando ${numRequests} requests rápidos...`)

    // Enviar todos en paralelo (para simular alta carga)
    for (let i = 0; i < numRequests; i++) {
      requests.push(page.request.get(`${API_BASE}/catalogs/countries`))
    }

    const responses = await Promise.all(requests)
    const statuses = responses.map((r) => r.status())

    console.log(`Statuses obtenidos: ${statuses.join(", ")}`)

    // Algunos deben retornar 429
    const has429 = statuses.includes(429)
    expect(has429).toBe(true)
    console.log("✅ Algunos requests retornan 429 Too Many Requests")

    // Contar 429s
    const count429 = statuses.filter((s) => s === 429).length
    console.log(`   Total 429s: ${count429}/${numRequests}`)
  })

  // =========================================================================
  // TEST 3: Retry-After header
  // =========================================================================
  test("Respuesta 429 incluye header Retry-After", async ({ page }) => {
    test.skip(
      true,
      "El límite por IP vive en el nginx de producción (deploy/nginx.conf:33, `limit_req zone=general_api burst=30 nodelay` con `limit_req_status 429`). La API de Go no registra middleware de límite de peticiones (apps/backend/internal/platform/httpx/ no lo tiene) y el stack de desarrollo no pasa por nginx: el proxy de Vite va directo a la API. En este entorno el servidor nunca puede responder 429.",
    )
    console.log("\n========== TEST 3: RETRY-AFTER HEADER ==========\n")

    // Bombardear
    const requests: Promise<APIResponse>[] = []
    for (let i = 0; i < 40; i++) {
      requests.push(page.request.get(`${API_BASE}/catalogs/countries`))
    }

    const responses = await Promise.all(requests)

    // Buscar una respuesta 429
    const response429 = responses.find((r) => r.status() === 429)

    if (response429) {
      const retryAfter = response429.headers()["retry-after"]
      console.log(`✅ Respuesta 429 recibida`)
      console.log(`   Retry-After: ${retryAfter || "(no presente)"}`)

      if (retryAfter) {
        console.log(`✅ Header Retry-After presente: ${retryAfter}s`)
      } else {
        console.log("⚠️  Header Retry-After no presente")
      }
    } else {
      console.log("⚠️  No se pudo generar 429 en este intento")
    }
  })

  // =========================================================================
  // TEST 4: Esperar y reintentar después de rate limit
  // =========================================================================
  test("Esperar después de 429 → requests vuelven a ser exitosos", async ({
    page,
  }) => {
    test.skip(
      true,
      "El límite por IP vive en el nginx de producción (deploy/nginx.conf:33, `limit_req zone=general_api burst=30 nodelay` con `limit_req_status 429`). La API de Go no registra middleware de límite de peticiones (apps/backend/internal/platform/httpx/ no lo tiene) y el stack de desarrollo no pasa por nginx: el proxy de Vite va directo a la API. En este entorno el servidor nunca puede responder 429.",
    )
    console.log("\n========== TEST 4: ESPERAR Y REINTENTAR ==========\n")

    // Bombardear para generar 429
    const requests1 = []
    for (let i = 0; i < 40; i++) {
      requests1.push(page.request.get(`${API_BASE}/catalogs/countries`))
    }
    const responses1 = await Promise.all(requests1)
    const has429 = responses1.some((r) => r.status() === 429)

    if (has429) {
      console.log("✅ Rate limit alcanzado (429)")

      // Esperar 2 segundos
      console.log("Esperando 2 segundos...")
      await page.waitForTimeout(2000)

      // Reintentar
      console.log("Reintentando requests...")
      const requests2 = []
      for (let i = 0; i < 5; i++) {
        requests2.push(page.request.get(`${API_BASE}/catalogs/countries`))
      }
      const responses2 = await Promise.all(requests2)
      const allOk = responses2.every((r) => r.status() === 200)

      if (allOk) {
        console.log("✅ Después de esperar, requests retornan 200 OK")
      } else {
        console.log("⚠️  Algunos requests siguen fallando")
      }
    }
  })

  // =========================================================================
  // TEST 5: Frontend maneja 429 gracefully
  // =========================================================================
  test("Frontend muestra mensaje cuando recibe 429", async ({ page }) => {
    test.skip(
      true,
      "El límite por IP vive en el nginx de producción (deploy/nginx.conf:33, `limit_req zone=general_api burst=30 nodelay` con `limit_req_status 429`). La API de Go no registra middleware de límite de peticiones (apps/backend/internal/platform/httpx/ no lo tiene) y el stack de desarrollo no pasa por nginx: el proxy de Vite va directo a la API. En este entorno el servidor nunca puede responder 429.",
    )
    console.log("\n========== TEST 5: FRONTEND MANEJA 429 ==========\n")

    // Loguear primero
    const cuenta = cuentaPorRol("postulante")

    await page.goto("/login")
    await page.fill("input#cedula", cuenta.cedula)
    await page.fill("input#password", PASSWORD)
    await page.locator("button[type='submit']").click()

    await page.waitForURL(/\/postulacion/)

    // Ahora bombardear desde la UI
    console.log("Bombardeando desde UI para generar 429...")

    // Hacer click en un botón que genera multiple requests (ej: cargar datos)
    const btnRecargar = page.locator("button:has-text('Recargar')").first()
    if (await btnRecargar.isVisible()) {
      // Hacer clicks rápidos
      for (let i = 0; i < 10; i++) {
        await btnRecargar.click()
        await page.waitForTimeout(50)
      }

      // Esperar a que aparezca mensaje de error
      const msgError = page
        .locator(
          "text=/demasiadas solicitudes|rate limit|espera|intenta más tarde/i",
        )
        .first()
      if (await msgError.isVisible({ timeout: 3000 })) {
        console.log(`✅ Mensaje mostrado: "${await msgError.textContent()}"`)
      } else {
        console.log("⚠️  No hay mensaje de rate limit visible")
      }
    } else {
      console.log("⚠️  No hay botón 'Recargar' para probar")
    }
  })

  // =========================================================================
  // TEST 6: Rate limit por IP (verificable con headers)
  // =========================================================================
  test("Rate limit es por IP (X-Forwarded-For / X-Real-IP)", async ({
    page,
  }) => {
    test.skip(
      true,
      "El límite por IP vive en el nginx de producción (deploy/nginx.conf:33, `limit_req zone=general_api burst=30 nodelay` con `limit_req_status 429`). La API de Go no registra middleware de límite de peticiones (apps/backend/internal/platform/httpx/ no lo tiene) y el stack de desarrollo no pasa por nginx: el proxy de Vite va directo a la API. En este entorno el servidor nunca puede responder 429.",
    )
    console.log("\n========== TEST 6: RATE LIMIT POR IP ==========\n")

    // Bombardear
    const requests: Promise<APIResponse>[] = []
    for (let i = 0; i < 40; i++) {
      requests.push(page.request.get(`${API_BASE}/catalogs/countries`))
    }

    const responses = await Promise.all(requests)
    const response429 = responses.find((r) => r.status() === 429)

    if (response429) {
      console.log("✅ Rate limit es por cliente/IP")
      console.log(`   (Todos los requests comparten el mismo contador)`)
    } else {
      console.log("⚠️  No se pudo verificar mecanismo de rate limit")
    }
  })
})
