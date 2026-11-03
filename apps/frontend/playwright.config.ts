import { defineConfig, devices } from "@playwright/test"

/**
 * Configuración de las pruebas de navegador E2E con datos sintéticos.
 *
 * La suite se apoya en el mismo seed que usa la suite de API del backend
 * (`apps/backend/cmd/e2e-seed`): las dos hablan del mismo conjunto de
 * cuentas, convocatoria y expedientes.
 */
export default defineConfig({
  testDir: "./e2e",
  globalSetup: "./e2e/global-setup.ts",
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: 0,
  workers: 1,
  reporter: [["list"]],
  use: {
    baseURL: process.env.PLAYWRIGHT_BASE_URL || "http://localhost:5173",
    trace: "on-first-retry",
    screenshot: "only-on-failure",
    video: "retain-on-failure",
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
  ],
})
