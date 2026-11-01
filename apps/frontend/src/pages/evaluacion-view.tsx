import { PortalEvaluacionPostulante } from "@/features/evaluacion/portal-evaluacion-postulante"

/**
 * Portal de acceso a la evaluación técnica EF-14 (FT25).
 * Presenta elegibilidad, verificación de méritos, sala de espera y accesos oficiales.
 */
export function EvaluacionView() {
  return (
    <div className="mx-auto flex w-full max-w-5xl flex-1 flex-col gap-6 p-4 sm:p-6">
      <PortalEvaluacionPostulante />
    </div>
  )
}
