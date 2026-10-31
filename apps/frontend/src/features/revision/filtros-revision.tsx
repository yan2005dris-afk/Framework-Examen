import { useSearchParams } from "react-router"
import { Button } from "@/components/ui/button"
import { EstadoError } from "@/components/estados"
import { useConvocatoriasRevision } from "./queries"
import {
  busquedaRevision,
  fechaRevision,
  leerFiltrosRevision,
  SECTORES_REVISION,
} from "./modelo"
import type { FiltrosRevision } from "@/lib/clientes/revision"

export function FiltrosRevisionPanel() {
  const [params, cambiarParams] = useSearchParams()
  const { filtros, error } = leerFiltrosRevision(params)
  const convocatorias = useConvocatoriasRevision()
  const periodos = convocatorias.data?.pages.flatMap((p) => p.items) ?? []
  function cambiar(cambios: Partial<FiltrosRevision>) {
    cambiarParams(busquedaRevision({ ...filtros, ...cambios }))
  }
  const clase =
    "h-10 w-full min-w-0 rounded-md border bg-background px-3 text-sm focus-visible:outline-2 focus-visible:outline-ring"
  return (
    <section
      aria-label="Filtros de revisión"
      className="space-y-4 rounded-xl border bg-card p-5"
    >
      <div className="grid gap-4 lg:grid-cols-[2fr_1fr_1fr]">
        <label className="space-y-2 text-sm font-medium">
          <span>Convocatoria o histórico</span>
          <select
            className={clase}
            value={filtros.periodo}
            onChange={(e) => cambiar({ periodo: e.target.value })}
          >
            <option value="">Seleccione un período</option>
            <option value="legado">Legado sin convocatoria</option>
            {filtros.periodo &&
              filtros.periodo !== "legado" &&
              !periodos.some((p) => p.id === filtros.periodo) && (
                <option value={filtros.periodo}>
                  Período seleccionado · {filtros.periodo}
                </option>
              )}
            {periodos.map((p) => (
              <option key={p.id} value={p.id}>
                {p.codigo} · {p.titulo} {p.cerrada ? "(cerrada)" : ""}
              </option>
            ))}
          </select>
        </label>
        <label className="space-y-2 text-sm font-medium">
          <span>Cargo / sector</span>
          <select
            className={clase}
            value={filtros.sector}
            onChange={(e) =>
              cambiar({ sector: e.target.value as FiltrosRevision["sector"] })
            }
          >
            <option value="">Todos los sectores</option>
            {SECTORES_REVISION.map((s) => (
              <option key={s.valor} value={s.valor}>
                {s.etiqueta}
              </option>
            ))}
          </select>
        </label>
        <label className="space-y-2 text-sm font-medium">
          <span>Estado del expediente</span>
          <select
            className={clase}
            value={filtros.estado}
            onChange={(e) =>
              cambiar({ estado: e.target.value as FiltrosRevision["estado"] })
            }
          >
            <option value="">Todos los disponibles (sellados)</option>
            <option value="submitted">Enviado y sellado</option>
          </select>
        </label>
      </div>
      {error && (
        <p role="alert" className="text-sm text-destructive">
          {error}{" "}
          <Button variant="link" onClick={() => cambiarParams({})}>
            Restablecer filtros
          </Button>
        </p>
      )}
      {convocatorias.isPending && (
        <p role="status" className="text-sm text-muted-foreground">
          Consultando convocatorias…
        </p>
      )}
      {convocatorias.isError && (
        <EstadoError
          error={convocatorias.error}
          titulo="No se pudieron consultar las convocatorias"
          onReintentar={() => void convocatorias.refetch()}
        />
      )}
      {convocatorias.isSuccess && periodos.length === 0 && (
        <p className="text-sm text-muted-foreground">
          No hay convocatorias registradas. El legado se consulta por separado;
          no se asigna a una convocatoria actual.
        </p>
      )}
      {convocatorias.hasNextPage && (
        <Button
          variant="outline"
          disabled={convocatorias.isFetchingNextPage}
          onClick={() => void convocatorias.fetchNextPage()}
        >
          Cargar más convocatorias
        </Button>
      )}
      {periodos.find((p) => p.id === filtros.periodo) && (
        <p className="text-xs text-muted-foreground">
          Período:{" "}
          {fechaRevision(
            periodos.find((p) => p.id === filtros.periodo)?.apertura,
          )}{" "}
          —{" "}
          {fechaRevision(
            periodos.find((p) => p.id === filtros.periodo)?.cierre,
          )}{" "}
          (hora de Ecuador)
        </p>
      )}
      <p className="text-xs text-muted-foreground">
        Solo expedientes enviados. Los borradores no forman parte de esta
        nómina. El envío no equivale a aprobación de méritos ni a resultado de
        fase.
      </p>
    </section>
  )
}
