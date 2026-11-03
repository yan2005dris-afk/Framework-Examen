import { ResultadoExamen } from "@/features/evaluacion/resultado-examen"

/**
 * Nota final, desglose de respuestas y descarga de acta oficial de examen (EF-14 / FT27).
 */
export function EvaluacionResultadoView() {
  return (
    <div className="mx-auto flex w-full max-w-6xl flex-1 flex-col gap-6 px-5 pt-8 pb-16">
      <ResultadoExamen />
    </div>
  )
}
