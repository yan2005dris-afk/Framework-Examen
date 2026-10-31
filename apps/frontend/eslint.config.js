import js from "@eslint/js"
import globals from "globals"
import reactHooks from "eslint-plugin-react-hooks"
import reactRefresh from "eslint-plugin-react-refresh"
import tseslint from "typescript-eslint"
import { defineConfig, globalIgnores } from "eslint/config"

export default defineConfig([
  globalIgnores(["dist"]),
  {
    files: ["**/*.{ts,tsx}"],
    extends: [
      js.configs.recommended,
      tseslint.configs.recommended,
      reactHooks.configs.flat.recommended,
      reactRefresh.configs.vite,
    ],
    languageOptions: {
      globals: globals.browser,
    },
  },
  {
    // Código generado por el CLI de shadcn/ui. Estos archivos exportan las
    // variantes de estilo junto al componente (p. ej. `buttonVariants`) y
    // sincronizan estado dentro de efectos, que es el patrón de Radix/embla.
    // Las reglas estrictas se siguen aplicando a todo el código propio.
    files: ["src/components/ui/**/*.{ts,tsx}", "src/hooks/**/*.{ts,tsx}"],
    rules: {
      "react-refresh/only-export-components": "off",
      "react-hooks/set-state-in-effect": "off",
    },
  },
  {
    // Módulos de contexto (`*-context.tsx`): exportan el proveedor junto a su
    // hook de consumo para que el contexto sea un solo archivo cohesionado.
    // La regla sigue activa: solo se permiten los hooks declarados acá, y
    // cambiar este archivo hace que el HMR recargue el módulo completo (no hay
    // Fast Refresh del provider). Al crear otro contexto, agregá su hook a la
    // lista en lugar de apagar la regla.
    files: ["src/features/**/*-context.tsx"],
    rules: {
      "react-refresh/only-export-components": [
        "error",
        {
          allowConstantExport: true,
          allowCompoundComponents: true,
          allowExportNames: ["useSesion"],
        },
      ],
    },
  },
])
