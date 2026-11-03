import { Link } from "react-router"
import { Button } from "@/components/ui/button"
import { EstadoCargando, EstadoError } from "@/components/estados"
import { HojaPostulacion } from "@/features/formulario/hoja-postulacion"
import { usePostulacionActual } from "@/features/postulaciones/queries"
import { RUTAS, rutaPostulacion } from "@/lib/rutas"

/** Presentación A4 del formulario consultado, sin alterar contenido ni hash. */
export function ImprimirView() {
  const { postulacion, isLoading, error, refetch } = usePostulacionActual()
  if (isLoading)
    return (
      <div className="mx-auto max-w-6xl p-8">
        <EstadoCargando />
      </div>
    )
  if (error || !postulacion)
    return (
      <div className="mx-auto max-w-6xl p-8">
        <EstadoError
          error={error}
          titulo="No se pudo cargar el formulario"
          onReintentar={() => void refetch()}
        />
      </div>
    )
  const sellada =
    postulacion.status !== "draft" &&
    Boolean(postulacion.hash && postulacion.submittedAt)
  return (
    <div className="vista-impresion mx-auto w-full max-w-6xl space-y-6 px-5 pt-8 pb-16">
      <div className="no-print flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">
            Formulario para impresión y firma
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Hoja A4 íntegra con espacio para firma manuscrita.
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button asChild variant="outline">
            <Link
              to={rutaPostulacion(
                RUTAS.postulacionRevision,
                postulacion.convocatoriaId,
              )}
            >
              Volver a la revisión
            </Link>
          </Button>
          <Button
            type="button"
            variant="outline"
            onClick={() => window.print()}
          >
            Guardar como PDF
          </Button>
          <Button type="button" onClick={() => window.print()}>
            Imprimir
          </Button>
        </div>
      </div>
      <aside
        className="no-print rounded-lg border bg-brand-100 p-4 text-sm"
        role="status"
      >
        <strong>
          {sellada
            ? "Formulario enviado y sellado"
            : postulacion.status === "draft"
              ? "Borrador preliminar sin sello confirmado"
              : "Formulario sin sello confirmado"}
        </strong>
        <p className="mt-1">
          Al imprimir se ocultan cabecera, navegación, pie y botones.{" "}
          {sellada
            ? "Se conservan el código y el hash del envío. Imprima íntegramente y firme de forma manuscrita según la convocatoria."
            : "Esta vista previa no formaliza la postulación ni sustituye su envío."}
        </p>
      </aside>
      <HojaPostulacion postulacion={postulacion} />
    </div>
  )
}
