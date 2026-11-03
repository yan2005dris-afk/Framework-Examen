import { SalaExamen } from "@/features/evaluacion/sala-examen"

/**
 * Aplicación de examen con reloj de cuenta regresiva y navegación de preguntas (EF-14 / FT26).
 */
export function EvaluacionExamenView() {
  return (
    <div className="mx-auto flex w-full max-w-6xl flex-1 flex-col gap-6 px-5 pt-8 pb-16">
      <SalaExamen />
    </div>
  )
}
