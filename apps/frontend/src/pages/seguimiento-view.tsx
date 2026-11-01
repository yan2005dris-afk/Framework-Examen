import { Link } from "react-router"
import {
  CheckCircle2Icon,
  ClockIcon,
  FileCheckIcon,
  XCircleIcon,
} from "lucide-react"

import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { RUTAS } from "@/lib/rutas"
import { useListarPostulaciones } from "@/features/postulaciones/queries"
import { useResultadoMeritosPostulante } from "@/features/revision/queries"
import { centipuntosAPuntos } from "@/lib/clientes/meritos"

/**
 * Seguimiento de fases, resultados y notificaciones del postulante (FT19/FT22).
 */
export function SeguimientoView() {
  const { data: postulacionesData } = useListarPostulaciones({ limit: 1 })
  const postulacion = postulacionesData?.items?.[0]

  const { data: meritosData } = useResultadoMeritosPostulante(
    postulacion?.id ?? "",
  )

  const resultado = meritosData?.resultado
  const publicado = meritosData?.publicado && Boolean(resultado)

  return (
    <div className="mx-auto flex w-full max-w-5xl flex-1 flex-col gap-6 p-4 sm:p-6">
      <Card>
        <CardHeader>
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div className="flex items-center gap-2">
              <FileCheckIcon className="size-5 text-primary" />
              <CardTitle>Seguimiento de Fases y Resultados (FT19)</CardTitle>
            </div>
            {postulacion && (
              <Badge variant="outline" className="font-mono text-xs">
                {postulacion.code}
              </Badge>
            )}
          </div>
          <CardDescription>
            Consulte las resoluciones oficiales y el avance de su expediente en
            cada etapa del concurso.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          {/* Fase 1: Postulación y Sellado */}
          <div className="rounded-lg border p-4 space-y-2">
            <div className="flex items-center justify-between">
              <span className="font-semibold text-sm">
                1. Postulación y Entrega EF-12
              </span>
              {postulacion?.status === "submitted" ? (
                <Badge variant="default">Sellada y Enviada</Badge>
              ) : postulacion?.status === "closed" ? (
                <Badge variant="destructive">Cerrada por Elegibilidad</Badge>
              ) : (
                <Badge variant="secondary">Borrador en Edición</Badge>
              )}
            </div>
            <p className="text-xs text-muted-foreground">
              {postulacion?.status === "submitted"
                ? `Expediente sellado el ${new Date(postulacion.submittedAt || postulacion.createdAt).toLocaleDateString("es-EC")} con firma criptográfica SHA-256.`
                : "Complete y selle su postulación para acceder a la calificación de méritos."}
            </p>
          </div>

          {/* Fase 2: Calificación de Méritos (FT19) */}
          <div className="rounded-lg border p-4 space-y-4">
            <div className="flex items-center justify-between">
              <span className="font-semibold text-sm">
                2. Calificación de Méritos
              </span>
              {publicado ? (
                <Badge
                  variant={
                    resultado?.decision === "avanza" ? "default" : "destructive"
                  }
                  className="capitalize"
                >
                  Resolución: {resultado?.decision}
                </Badge>
              ) : (
                <Badge variant="outline" className="flex items-center gap-1">
                  <ClockIcon className="size-3" />
                  <span>En Calificación</span>
                </Badge>
              )}
            </div>

            {publicado ? (
              <Alert
                className={
                  resultado?.decision === "avanza"
                    ? "border-emerald-500/40 bg-emerald-50/50 dark:bg-emerald-950/20"
                    : "border-destructive/40 bg-destructive/10"
                }
              >
                {resultado?.decision === "avanza" ? (
                  <CheckCircle2Icon className="size-4 text-emerald-600" />
                ) : (
                  <XCircleIcon className="size-4 text-destructive" />
                )}
                <AlertTitle className="text-sm font-semibold">
                  {resultado?.decision === "avanza"
                    ? "Postulación Habilitada para Oposición Técnica"
                    : "Postulación No Habilitada"}
                </AlertTitle>
                <AlertDescription className="text-xs space-y-2 mt-1">
                  <p>
                    <strong>Puntaje oficial de méritos:</strong>{" "}
                    <span className="font-semibold">
                      {centipuntosAPuntos(
                        resultado?.evaluacion.totalCentipuntos ?? 0,
                      )}{" "}
                      puntos
                    </span>{" "}
                    (de un máximo de{" "}
                    {centipuntosAPuntos(
                      resultado?.evaluacion.maximoCentipuntos ?? 0,
                    )}{" "}
                    pts).
                  </p>
                  <p>
                    <strong>Motivo de la resolución:</strong>{" "}
                    {resultado?.motivo}
                  </p>
                  <p className="text-[11px] text-muted-foreground">
                    {resultado?.decision === "avanza"
                      ? "Conforme a las bases del concurso, al superar el umbral de méritos se encuentra facultado para rendir las pruebas psicotécnicas y teóricas."
                      : "La postulación no alcanzó el umbral mínimo exigido o presentó causales de exclusión documental."}
                  </p>
                </AlertDescription>
              </Alert>
            ) : (
              <p className="text-xs text-muted-foreground">
                La Comisión Especial se encuentra verificando los documentos de
                respaldo y requisitos habilitantes. Una vez publicada la
                resolución oficial, podrá consultar aquí su puntaje desglosado.
              </p>
            )}
          </div>

          <div className="flex gap-2">
            <Button asChild variant="outline" size="sm">
              <Link to={RUTAS.postulacion}>Volver a mi postulación</Link>
            </Button>
            {postulacion?.status === "submitted" && (
              <Button asChild size="sm">
                <Link to={RUTAS.postulacionImprimir}>
                  Ver comprobante imprimible
                </Link>
              </Button>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
