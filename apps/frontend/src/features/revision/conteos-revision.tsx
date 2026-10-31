import { useSyncExternalStore } from "react"
import { UsersIcon, RefreshCwIcon } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { EstadoCargando, EstadoError } from "@/components/estados"
import type { FiltrosRevision } from "@/lib/clientes/revision"
import { fechaRevision, SECTORES_REVISION } from "./modelo"
import { useConteosRevision } from "./queries"

function suscribirRed(actualizar: () => void) {
  window.addEventListener("online", actualizar)
  window.addEventListener("offline", actualizar)
  return () => {
    window.removeEventListener("online", actualizar)
    window.removeEventListener("offline", actualizar)
  }
}

export function ConteosRevisionPanel({
  filtros,
}: {
  filtros: FiltrosRevision
}) {
  const consulta = useConteosRevision(filtros)
  const conectado = useSyncExternalStore(
    suscribirRed,
    () => navigator.onLine,
    () => true,
  )
  if (consulta.isPending)
    return (
      <EstadoCargando
        mensaje={
          conectado
            ? "Consultando conteos reales…"
            : "Sin conexión. Los conteos se consultarán al reconectar."
        }
      />
    )
  if (consulta.isError)
    return (
      <EstadoError
        error={consulta.error}
        titulo="Conteos no disponibles"
        onReintentar={() => void consulta.refetch()}
      />
    )
  const conteos = consulta.data.conteos ?? []
  const sectores = filtros.sector
    ? SECTORES_REVISION.filter((s) => s.valor === filtros.sector)
    : SECTORES_REVISION
  return (
    <section className="space-y-4" aria-label="Conteos de la selección">
      <div className="flex flex-wrap items-center justify-between gap-3 text-sm text-muted-foreground">
        <p>
          {conectado
            ? "Consulta automática cada 30 s, con el panel visible."
            : "Sin conexión: se muestra la última consulta; se actualizará al reconectar."}{" "}
          <span className="block">
            Última consulta: {fechaRevision(consulta.data.consultadoEn)} ·
            Ecuador
          </span>
        </p>
        <Button
          variant="outline"
          size="sm"
          disabled={consulta.isFetching || !conectado}
          onClick={() => void consulta.refetch()}
        >
          <RefreshCwIcon
            className={consulta.isFetching ? "animate-spin" : ""}
          />
          Actualizar
        </Button>
      </div>
      <div className="grid gap-4 md:grid-cols-3">
        {sectores.map((s) => {
          const cuentas = conteos.filter((c) => c.sector === s.valor)
          return (
            <Card key={s.valor}>
              <CardHeader className="flex-row items-center justify-between">
                <CardTitle className="text-sm font-medium">
                  {s.etiqueta}
                </CardTitle>
                <UsersIcon className="size-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <p className="text-3xl font-semibold tabular-nums">
                  {cuentas.reduce((n, c) => n + c.postulantes, 0)}
                </p>
                <p className="text-sm text-muted-foreground">
                  postulantes ·{" "}
                  {cuentas.reduce((n, c) => n + c.postulaciones, 0)} expedientes
                  sellados
                </p>
              </CardContent>
            </Card>
          )
        })}
      </div>
      {conteos.some(
        (c) => !SECTORES_REVISION.some((s) => s.valor === c.sector),
      ) && (
        <p className="text-sm text-muted-foreground">
          Hay registros legados sin sector reconocido. Permanecen visibles en la
          nómina sin filtro de sector.
        </p>
      )}
      <p className="text-xs text-muted-foreground">
        Cada tarjeta cuenta personas distintas dentro de su sector. No se suman
        como un total de personas únicas entre sectores.
      </p>
    </section>
  )
}
