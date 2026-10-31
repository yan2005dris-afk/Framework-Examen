import path from "path"
import tailwindcss from "@tailwindcss/vite"
import react from "@vitejs/plugin-react"
import { defineConfig } from "vitest/config"

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(import.meta.dirname, "./src"),
    },
  },
  server: {
    // En Docker los archivos montados por bind mount no generan eventos de
    // inotify, así que el HMR solo funciona con polling.
    watch: { usePolling: true },
    // El servidor de desarrollo se expone en la red local del contenedor.
    allowedHosts: true,
    // Un solo origen para el navegador: /api se reenvía a la API Go, así que
    // no hay CORS ni cookies cross-origin durante el desarrollo.
    proxy: {
      "/api": {
        target: process.env.VITE_DEV_API_TARGET ?? "http://localhost:8080",
        changeOrigin: true,
      },
    },
  },
  preview: {
    proxy: {
      "/api": {
        target: process.env.VITE_DEV_API_TARGET ?? "http://localhost:8080",
        changeOrigin: true,
      },
    },
  },
  test: {
    environment: "node",
    include: ["src/**/*.test.ts"],
    coverage: {
      reporter: ["text", "lcov"],
      include: ["src/utils/**", "src/data/**"],
    },
  },
})
