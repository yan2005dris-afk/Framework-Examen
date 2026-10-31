import { CheckIcon, XIcon } from "lucide-react"

import { cn } from "@/lib/utils"
import { evaluarFortalezaPassword } from "@/utils/validators"

type MedidorPasswordProps = {
  password: string
  /** Muestra la lista detallada de requisitos cumplidos. */
  mostrarRequisitos?: boolean
  className?: string
}

/**
 * Componente visual de fortaleza de contraseña según el estándar del RF:
 * - Mínimo 8 caracteres
 * - Letra mayúscula
 * - Letra minúscula
 * - Número
 * - Carácter especial
 */
export function MedidorPassword({
  password,
  mostrarRequisitos = true,
  className,
}: MedidorPasswordProps) {
  if (!password) {
    return null
  }

  const evaluacion = evaluarFortalezaPassword(password)

  const requisitos = [
    { etiqueta: "Al menos 8 caracteres", cumple: evaluacion.cumpleLongitud },
    { etiqueta: "Una letra mayúscula", cumple: evaluacion.cumpleMayuscula },
    { etiqueta: "Una letra minúscula", cumple: evaluacion.cumpleMinuscula },
    { etiqueta: "Al menos un número", cumple: evaluacion.cumpleNumero },
    {
      etiqueta: "Un carácter especial (!@#$%...)",
      cumple: evaluacion.cumpleEspecial,
    },
  ]

  return (
    <div className={cn("space-y-2", className)} aria-live="polite">
      <div className="flex items-center justify-between text-xs">
        <span className="text-muted-foreground">Fortaleza de contraseña:</span>
        <span
          className={cn(
            "font-medium",
            evaluacion.nivel === "muy_debil" && "text-red-500",
            evaluacion.nivel === "debil" && "text-amber-500",
            evaluacion.nivel === "media" && "text-blue-600",
            evaluacion.nivel === "fuerte" && "text-emerald-600",
          )}
        >
          {evaluacion.etiqueta}
        </span>
      </div>

      <div className="relative h-1.5 w-full overflow-hidden rounded-full bg-muted">
        <div
          className={cn("h-full transition-all duration-300", evaluacion.color)}
          style={{ width: `${evaluacion.porcentaje}%` }}
          role="progressbar"
          aria-valuenow={evaluacion.porcentaje}
          aria-valuemin={0}
          aria-valuemax={100}
          aria-label="Fortaleza de la contraseña"
        />
      </div>

      {mostrarRequisitos && (
        <ul className="grid grid-cols-1 gap-1 text-xs text-muted-foreground sm:grid-cols-2">
          {requisitos.map((req) => (
            <li
              key={req.etiqueta}
              className={cn(
                "flex items-center gap-1.5 transition-colors",
                req.cumple
                  ? "font-medium text-emerald-600 dark:text-emerald-400"
                  : "text-muted-foreground",
              )}
            >
              {req.cumple ? (
                <CheckIcon
                  className="size-3.5 shrink-0 text-emerald-600"
                  aria-hidden="true"
                />
              ) : (
                <XIcon
                  className="size-3.5 shrink-0 text-muted-foreground/60"
                  aria-hidden="true"
                />
              )}
              <span>{req.etiqueta}</span>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
