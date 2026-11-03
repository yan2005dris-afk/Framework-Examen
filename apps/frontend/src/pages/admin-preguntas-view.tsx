import { GestionBancoPreguntas } from "@/features/preguntas/gestion-banco-preguntas"

/**
 * Gestión del banco de preguntas y configuración de evaluación técnica (EF-14 / FT24-FT25).
 */
export function AdminPreguntasView() {
  return (
    <div className="mx-auto flex w-full max-w-6xl flex-1 flex-col gap-6 px-5 pt-8 pb-16">
      <GestionBancoPreguntas />
    </div>
  )
}
