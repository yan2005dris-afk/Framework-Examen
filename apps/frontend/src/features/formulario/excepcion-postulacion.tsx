import { Link } from "react-router"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from "@/components/ui/card"
import { RUTAS } from "@/lib/rutas"

interface ExcepcionPostulacionProps {
  tipo: "nacionalidad" | "sector"
  cerrada: boolean
  valor: string
  mensaje?: string
  origen?: string
  onCorregir?: () => void
}

/** Distingue una declaración pendiente de corregir de un cierre confirmado por el servidor. */
export function ExcepcionPostulacion({
  tipo,
  cerrada,
  valor,
  mensaje,
  origen,
  onCorregir,
}: ExcepcionPostulacionProps) {
  const nacionalidad = tipo === "nacionalidad"
  return (
    <Card className="w-full">
      <CardHeader>
        <span className="mb-2 w-fit rounded-md bg-destructive-soft px-2.5 py-1 text-xs font-semibold text-destructive">
          {cerrada ? "Postulación cerrada" : "Requiere corrección"}
        </span>
        <CardTitle>
          <h1 className="text-lg font-bold">
            {cerrada
              ? "La postulación no puede continuar"
              : "Nacionalidad distinta a la ecuatoriana"}
          </h1>
        </CardTitle>
        <CardDescription>
          {nacionalidad
            ? "Requisito de nacionalidad para la postulación."
            : "Validación de elegibilidad del sector seleccionado."}
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-5">
        <div
          role="alert"
          className="rounded-lg border border-destructive/30 bg-destructive-soft p-4 text-sm text-destructive"
        >
          <p className="font-semibold">
            {nacionalidad
              ? "Se requiere nacionalidad ecuatoriana"
              : "No cumple la elegibilidad del sector"}
          </p>
          <p className="mt-1">
            {mensaje ??
              "La nacionalidad declarada impide continuar. Si seleccionó un dato incorrecto, puede volver a editarlo antes de guardar."}
          </p>
        </div>
        <dl className="divide-y rounded-lg border px-4 text-sm">
          {[
            [
              nacionalidad ? "Nacionalidad declarada" : "Sector seleccionado",
              valor,
            ],
            [
              "Origen del dato",
              origen ??
                "Declaración del postulante; no certificada por Registro Civil",
            ],
            [
              "Estado de la postulación",
              cerrada
                ? "Cerrada por el servidor · No editable"
                : "Borrador · Pendiente de corrección",
            ],
          ].map(([etiqueta, contenido]) => (
            <div key={etiqueta} className="grid gap-1 py-3 sm:grid-cols-2">
              <dt className="font-semibold">{etiqueta}</dt>
              <dd>{contenido}</dd>
            </div>
          ))}
        </dl>
        <div className="rounded-lg border bg-muted/30 p-4 text-sm">
          <h2 className="font-semibold">Qué puede hacer</h2>
          <p className="mt-1 text-muted-foreground">
            {cerrada
              ? "Si considera que el resultado es incorrecto, consulte el canal de atención de la Comisión indicado en la convocatoria. Esta pantalla no reabre una postulación cerrada ni fija plazos de reclamo."
              : "Revise el dato declarado. La corrección no acredita una verificación de identidad civil."}
          </p>
        </div>
        {!cerrada && onCorregir && (
          <Button type="button" onClick={onCorregir}>
            Volver a editar mis datos
          </Button>
        )}
        <Button asChild variant="outline" className="w-full">
          <Link to={RUTAS.inicio}>Volver a la convocatoria</Link>
        </Button>
      </CardContent>
    </Card>
  )
}
