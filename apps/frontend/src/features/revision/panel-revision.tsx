import { Link, useSearchParams } from "react-router"
import {
  ArrowRightIcon,
  FileSpreadsheetIcon,
  FolderSearchIcon,
} from "lucide-react"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { EstadoCargando, EstadoError, EstadoVacio } from "@/components/estados"
import { RUTAS, rutaDetallePostulacion } from "@/lib/rutas"
import { esErrorApi } from "@/lib/api-error"
import type { FiltrosRevision } from "@/lib/clientes/revision"
import { FiltrosRevisionPanel } from "./filtros-revision"
import { ConteosRevisionPanel } from "./conteos-revision"
import { ExportarMatriz } from "./exportar-matriz"
import {
  busquedaRevision,
  fechaRevision,
  leerFiltrosRevision,
  SECTORES_REVISION,
} from "./modelo"
import { useCuentaRevision, useNominaRevision } from "./queries"

function NominaRevision({ filtros }: { filtros: FiltrosRevision }) {
  const consulta = useNominaRevision(filtros)
  if (consulta.isPending)
    return <EstadoCargando mensaje="Consultando expedientes sellados…" />
  if (
    consulta.isError &&
    (!consulta.isFetchNextPageError ||
      (esErrorApi(consulta.error) &&
        [401, 403].includes(consulta.error.estado)))
  )
    return (
      <EstadoError
        error={consulta.error}
        onReintentar={() => void consulta.refetch()}
      />
    )
  const filas = consulta.data?.pages.flatMap((p) => p.items) ?? []
  if (filas.length === 0)
    return (
      <EstadoVacio
        titulo="No hay expedientes en esta selección"
        descripcion="No se encontraron envíos sellados con estos filtros. Esto no incluye borradores ni resultados de fases."
      />
    )
  return (
    <section className="space-y-4" aria-labelledby="titulo-nomina">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 id="titulo-nomina" className="text-lg font-semibold">
          Nómina de expedientes
        </h2>
        <p className="text-sm text-muted-foreground">
          {filas.length} cargados · orden del servidor
        </p>
      </div>
      <div className="overflow-x-auto rounded-xl border">
        <table className="w-full min-w-[680px] text-left text-sm">
          <caption className="sr-only">
            Postulaciones enviadas de la convocatoria y sector seleccionados
          </caption>
          <thead className="border-b bg-muted/50 text-xs text-muted-foreground">
            <tr>
              {[
                "Postulante / código",
                "Cargo / sector",
                "Envío (Ecuador)",
                "Estado",
                "Expediente",
              ].map((c) => (
                <th key={c} scope="col" className="p-4 font-medium">
                  {c}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {filas.map((fila) => (
              <tr
                key={fila.id}
                className="border-b last:border-0 hover:bg-muted/30"
              >
                <td className="p-4">
                  <p className="font-medium">
                    {fila.apellidos} {fila.nombres}
                  </p>
                  <p className="font-mono text-xs text-muted-foreground">
                    {fila.codigo}
                  </p>
                </td>
                <td className="p-4">
                  {SECTORES_REVISION.find((s) => s.valor === fila.sector)
                    ?.etiqueta ??
                    (fila.sector || "No consta (legado)")}
                </td>
                <td className="p-4">{fechaRevision(fila.submittedAt)}</td>
                <td className="p-4">
                  <Badge variant="secondary">
                    {fila.status === "submitted"
                      ? "Enviado y sellado"
                      : "Estado no reconocido"}
                  </Badge>
                </td>
                <td className="p-4">
                  <Button variant="outline" size="sm" asChild>
                    <Link
                      to={`${rutaDetallePostulacion(fila.id)}${busquedaRevision(filtros)}`}
                      aria-label={`Ver expediente ${fila.codigo}`}
                    >
                      Ver EF-12
                      <ArrowRightIcon />
                    </Link>
                  </Button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {consulta.isFetchNextPageError && (
        <EstadoError
          error={consulta.error}
          titulo="No se pudo cargar la siguiente página"
          onReintentar={() => void consulta.fetchNextPage()}
        />
      )}
      {consulta.hasNextPage ? (
        <Button
          variant="outline"
          disabled={consulta.isFetching}
          onClick={() => void consulta.fetchNextPage()}
        >
          {consulta.isFetchingNextPage ? "Cargando…" : "Cargar más expedientes"}
        </Button>
      ) : (
        <p className="text-xs text-muted-foreground">
          Fin de la nómina para estos filtros.
        </p>
      )}
    </section>
  )
}

/** Las tres vistas comparten filtros en la URL; cambiarla desmonta descargas y cursores anteriores. */
export function PanelRevision({
  vista,
}: {
  vista: "inicio" | "nomina" | "reportes"
}) {
  const [params] = useSearchParams()
  const { filtros, error } = leerFiltrosRevision(params)
  const cuenta = useCuentaRevision()
  const clave = `${cuenta}:${busquedaRevision(filtros)}`
  const titulos = {
    inicio: "Panel de revisión",
    nomina: "Postulaciones e históricos",
    reportes: "Matriz comparativa",
  }
  return (
    <main className="mx-auto w-full max-w-7xl space-y-6 p-4 md:p-8">
      <header className="space-y-2">
        <p className="text-xs font-semibold uppercase tracking-widest text-muted-foreground">
          Talento Humano · Uso interno
        </p>
        <h1 className="text-2xl font-semibold tracking-tight md:text-3xl">
          {titulos[vista]}
        </h1>
        <p className="max-w-3xl text-sm text-muted-foreground">
          Consulte los envíos y sus respaldos originales por convocatoria. La
          información declarada no constituye verificación de identidad civil ni
          calificación de méritos.
        </p>
      </header>
      <FiltrosRevisionPanel />
      {!filtros.periodo || error ? (
        <EstadoVacio
          titulo="Seleccione una convocatoria"
          descripcion="Elija un período, incluso cerrado, o el legado sin convocatoria. No se mezclarán expedientes de distintos períodos."
        />
      ) : (
        <div key={clave} className="space-y-6">
          <ConteosRevisionPanel filtros={filtros} />
          {vista === "inicio" && (
            <div className="grid gap-4 md:grid-cols-2">
              {[
                {
                  ruta: RUTAS.dashboardPostulaciones,
                  titulo: "Revisar la nómina",
                  texto:
                    "EF-12 íntegro, certificados y duración de las experiencias al momento del envío.",
                  icono: FolderSearchIcon,
                },
                {
                  ruta: RUTAS.dashboardReportes,
                  titulo: "Exportar la matriz",
                  texto:
                    "Excel o PDF de toda la selección. Sin puntajes ni lista pública de elegibles.",
                  icono: FileSpreadsheetIcon,
                },
              ].map((enlace) => (
                <Link
                  key={enlace.ruta}
                  className="group space-y-3 rounded-xl border bg-card p-6 transition-colors hover:bg-muted/40 focus-visible:outline-2 focus-visible:outline-ring"
                  to={`${enlace.ruta}${busquedaRevision(filtros)}`}
                >
                  <enlace.icono className="size-5 text-primary" />
                  <h2 className="font-semibold">{enlace.titulo} →</h2>
                  <p className="text-sm text-muted-foreground">
                    {enlace.texto}
                  </p>
                </Link>
              ))}
            </div>
          )}
          {vista === "nomina" && <NominaRevision filtros={filtros} />}
          {vista !== "inicio" && <ExportarMatriz filtros={filtros} />}
          {vista === "reportes" && (
            <p className="text-sm text-muted-foreground">
              Columnas técnicas: código, postulante, cargo / sector, envío en
              UTC, formación declarada, experiencia por registro, publicaciones
              y horas de capacitación. La convocatoria, el estado enviado y la
              fecha de consulta constan en la cabecera del archivo.{" "}
              <Link
                className="text-primary underline"
                to={`${RUTAS.dashboardPostulaciones}${busquedaRevision(filtros)}`}
              >
                Ver la nómina con estos filtros
              </Link>
              .
            </p>
          )}
        </div>
      )}
    </main>
  )
}
