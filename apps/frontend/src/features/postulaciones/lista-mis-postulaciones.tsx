import { ArrowRightIcon } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import type {
  EstadoPostulacion,
  PostulacionDto,
} from "@/lib/clientes/postulaciones"

const ETIQUETAS_ESTADO: Record<EstadoPostulacion, string> = {
  draft: "Borrador",
  submitted: "Enviada",
  closed: "Cerrada",
  under_review: "En revisión",
  approved: "Aprobada",
  rejected: "No aprobada",
}

export interface ListaMisPostulacionesProps {
  postulaciones: PostulacionDto[]
  /** Título de cada convocatoria por id, si el listado público lo trae. */
  titulos: Record<string, string>
  onAbrir: (convocatoriaId: string) => void
}

/**
 * Postulaciones del postulante cuando tiene más de una y la URL no indica
 * cuál abrir. Cada una pertenece a una convocatoria distinta.
 */
export function ListaMisPostulaciones({
  postulaciones,
  titulos,
  onAbrir,
}: ListaMisPostulacionesProps) {
  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-4 px-4 py-10">
      <div className="flex flex-col gap-1">
        <h1 className="text-xl font-semibold">Mis postulaciones</h1>
        <p className="text-sm text-muted-foreground">
          Tiene postulaciones en varias convocatorias. Elija cuál desea abrir.
        </p>
      </div>
      {postulaciones.map((postulacion) => {
        const convocatoriaId = postulacion.convocatoriaId
        return (
          <Card key={postulacion.id}>
            <CardHeader>
              <CardTitle className="flex flex-wrap items-center gap-2 text-base">
                {(convocatoriaId && titulos[convocatoriaId]) ||
                  "Expediente sin convocatoria asignada"}
                <Badge variant="outline">
                  {ETIQUETAS_ESTADO[postulacion.status] ?? postulacion.status}
                </Badge>
              </CardTitle>
              <CardDescription>
                Código {postulacion.code} · Avance{" "}
                {postulacion.progress.percentage}%
              </CardDescription>
            </CardHeader>
            <CardFooter>
              <Button
                type="button"
                variant="outline"
                disabled={!convocatoriaId}
                onClick={() => convocatoriaId && onAbrir(convocatoriaId)}
              >
                Abrir postulación
                <ArrowRightIcon data-icon="inline-end" />
              </Button>
            </CardFooter>
          </Card>
        )
      })}
    </div>
  )
}
