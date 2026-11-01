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
  {
    // Frontera entre features. Cada feature es una unidad autónoma: la lógica
    // pura vive en `features/<dominio>/modelo.ts` o `validacion.ts`, y una
    // feature no debe alcanzar los internos de otra.
    //
    // `auth` es un primitivo transversal (la sesión es global y un 401 debe
    // poder limpiarla desde cualquier lado). El resto de la lista son módulos
    // compartidos por dos o más features; si aparece un destino nuevo, la
    // regla lo bloquea y hay que subirlo a la capa compartida en vez de
    // ampliar la lista.
    //
    // Los tests quedan fuera: los tests de vista renderizan páginas a propósito.
    files: ["src/features/**/*.{ts,tsx}"],
    ignores: ["**/*.test.ts", "**/*.test.tsx"],
    rules: {
      "no-restricted-imports": [
        "error",
        {
          patterns: [
            {
              regex:
                "^@/features/(?!auth(?:/|$)|formulario/constantes$|postulaciones/queries$|revision/queries$|preguntas/queries$|privacidad/modal-politica-privacidad$)",
              message:
                "Una feature no importa los internos de otra. Subí el código compartido a la capa común o dejalo en la feature que lo usa.",
            },
            {
              regex: "^@/(pages|layouts)/",
              message:
                "Las features no dependen de pages ni de layouts: la composicion vive en la capa de aplicación.",
            },
          ],
        },
      ],
    },
  },
  {
    // Dirección unidireccional: las capas compartidas (`lib`, `utils`,
    // `helpers`, `types`, `data`) no dependen de las features ni de las vistas.
    // `lib` es infraestructura de transporte, `utils` son utilidades genéricas
    // y `helpers` son reglas de este portal: nada de eso conoce React.
    files: ["src/{lib,utils,helpers,types,data}/**/*.{ts,tsx}"],
    rules: {
      "no-restricted-imports": [
        "error",
        {
          patterns: [
            {
              regex: "^@/(features|pages|layouts|components)/",
              message:
                "Las capas compartidas no dependen de features ni de vistas: la dependencia va en la dirección contraria.",
            },
          ],
        },
      ],
    },
  },
  {
    // El design system es una hoja: `components/ui` no conoce features ni
    // vistas, porque el shadcn/ui se regenera con el CLI y no admite lógica
    // del dominio.
    files: ["src/components/ui/**/*.{ts,tsx}"],
    rules: {
      "no-restricted-imports": [
        "error",
        {
          patterns: [
            {
              regex: "^@/(features|pages|layouts)/",
              message:
                "components/ui es una hoja: no importa features ni vistas.",
            },
          ],
        },
      ],
    },
  },
])
