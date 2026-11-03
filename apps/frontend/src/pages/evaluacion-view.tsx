import { PortalEvaluacionPostulante } from "@/features/evaluacion/portal-evaluacion-postulante"

/**
 * Portal de acceso a la evaluación técnica EF-14 (FT25).
 * Presenta elegibilidad, verificación de méritos, sala de espera y accesos oficiales.
 */
export function EvaluacionView() {
  return (
    <div className="mx-auto flex w-full max-w-6xl flex-1 flex-col gap-6 px-5 pt-8 pb-16">
      <PortalEvaluacionPostulante />
    </div>
  )
}
