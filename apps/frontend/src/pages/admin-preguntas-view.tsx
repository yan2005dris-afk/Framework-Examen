import { GestionBancoPreguntas } from "@/features/preguntas/gestion-banco-preguntas"

/**
 * Gestión del banco de preguntas y configuración de evaluación técnica (EF-14 / FT24-FT25).
 */
export function AdminPreguntasView() {
  return (
    <div className="flex flex-1 flex-col gap-6 p-4 md:p-6">
      <GestionBancoPreguntas />
    </div>
  )
}
