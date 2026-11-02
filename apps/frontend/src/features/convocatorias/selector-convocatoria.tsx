import { ArrowRightIcon, CalendarClockIcon } from "lucide-react"

import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import type { ConvocatoriaPublica } from "@/types/convocatoria"
import { formatearFechaEcuador } from "@/utils/fechas"
import { SECTORES } from "./modelo"

export interface SelectorConvocatoriaProps {
  convocatorias: ConvocatoriaPublica[]
  onElegir: (convocatoriaId: string) => void
}

/**
 * Convocatorias con recepción abierta para que el postulante elija a cuál
 * postula. El borrador se crea sólo después de elegir: el sistema nunca
 * asigna una convocatoria cuando hay más de una opción.
 */
export function SelectorConvocatoria({
  convocatorias,
  onElegir,
}: SelectorConvocatoriaProps) {
  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-4 px-4 py-10">
      <div className="flex flex-col gap-1">
        <h1 className="text-xl font-semibold">
          Elija la convocatoria a la que postula
        </h1>
        <p className="text-sm text-muted-foreground">
          Puede tener una sola postulación en cada convocatoria. El borrador se
          crea cuando confirme su elección.
        </p>
      </div>
      {convocatorias.map((convocatoria) => {
        const recepcion = convocatoria.fases.find(
          (fase) => fase.codigo === "postulacion",
        )
        return (
          <Card key={convocatoria.id}>
            <CardHeader>
              <CardTitle className="text-base">{convocatoria.titulo}</CardTitle>
              <CardDescription>
                {convocatoria.sectores.map((s) => SECTORES[s]).join(" · ")}
              </CardDescription>
            </CardHeader>
            <CardContent className="flex items-center gap-2 text-sm text-muted-foreground">
              <CalendarClockIcon className="size-4" aria-hidden="true" />
              <span>
                Recepción hasta el{" "}
                {formatearFechaEcuador(
                  recepcion?.cierre ?? convocatoria.cierre,
                  { incluirHora: true },
                )}{" "}
                (hora de Ecuador)
              </span>
            </CardContent>
            <CardFooter>
              <Button type="button" onClick={() => onElegir(convocatoria.id)}>
                Postular a esta convocatoria
                <ArrowRightIcon data-icon="inline-end" />
              </Button>
            </CardFooter>
          </Card>
        )
      })}
    </div>
  )
}
