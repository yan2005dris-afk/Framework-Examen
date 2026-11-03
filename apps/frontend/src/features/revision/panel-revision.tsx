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

function NominaRevision({
  filtros,
  matriz = false,
}: {
  filtros: FiltrosRevision
  matriz?: boolean
}) {
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
          {matriz ? "Vista previa de la matriz" : "Nómina de expedientes"}
        </h2>
        <p className="text-sm text-muted-foreground">
          {filas.length} cargados · orden del servidor
        </p>
      </div>
      {matriz && (
        <p className="text-sm text-muted-foreground">
          Vista previa paginada de los expedientes cargados. Las descargas
          incluyen toda la selección; no se asignan puntajes ni dictámenes en
          esta tabla.
        </p>
      )}
      <div className="relative overflow-x-auto rounded-lg border">
        <table
          className={`w-full text-left text-sm ${matriz ? "min-w-[1100px]" : "min-w-[680px]"}`}
        >
          <caption className="sr-only">
            {matriz
              ? "Matriz técnica de datos declarados en expedientes sellados"
              : "Postulaciones enviadas de la convocatoria y sector seleccionados"}
          </caption>
          <thead className="border-b bg-muted/50 text-xs text-muted-foreground">
            <tr>
              {[
                "Postulante / código",
                "Cargo / sector",
                "Envío (Ecuador)",
                "Estado",
                ...(matriz
                  ? [
                      "Formación declarada",
                      "Experiencia al envío",
                      "Publicaciones",
                      "Capacitación (horas)",
                    ]
                  : []),
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
                {matriz && (
                  <>
                    <td className="p-4">
                      <ul className="space-y-2">
                        {fila.titulos?.length ? (
                          fila.titulos.map((titulo, n) => (
                            <li key={n}>
                              <span className="font-medium">
                                {titulo.titulo}
                              </span>
                              <span className="block text-xs text-muted-foreground">
                                {titulo.nivel}
                              </span>
                            </li>
                          ))
                        ) : (
                          <li>Sin registros declarados</li>
                        )}
                      </ul>
                    </td>
                    <td className="p-4">
                      <ul className="space-y-2">
                        {fila.experiencias?.length ? (
                          fila.experiencias.map((experiencia) => (
                            <li key={experiencia.id}>
                              <span className="font-medium">
                                {experiencia.cargo}
                              </span>
                              <span className="block text-xs text-muted-foreground">
                                {experiencia.duracion?.texto ??
                                  "Duración no disponible"}
                              </span>
                            </li>
                          ))
                        ) : (
                          <li>Sin registros declarados</li>
                        )}
                      </ul>
                    </td>
                    <td className="p-4 font-mono tabular-nums">
                      {fila.publicaciones}
                    </td>
                    <td className="p-4 font-mono tabular-nums">
                      {fila.horasCapacitacion}
                    </td>
                  </>
                )}
                <td className="p-4">
                  <Button variant="outline" size="sm" asChild>
                    <Link
                      to={`${rutaDetallePostulacion(fila.id)}${busquedaRevision(filtros)}`}
                      aria-label={`Ver expediente ${fila.codigo}`}
                    >
                      Ver formulario
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
  integrado = false,
}: {
  vista: "inicio" | "nomina" | "reportes"
  integrado?: boolean
}) {
  const [params] = useSearchParams()
  const { filtros, error } = leerFiltrosRevision(params)
  const cuenta = useCuentaRevision()
  const clave = `${cuenta}:${busquedaRevision(filtros)}`
  const titulos = {
    inicio: "Dashboard del concurso",
    nomina: "Postulaciones e históricos",
    reportes: "Matriz comparativa",
  }
  const Contenedor = integrado ? "section" : "main"
  return (
    <Contenedor
      className={
        integrado
          ? "space-y-6"
          : "mx-auto w-full max-w-6xl space-y-6 px-5 pt-8 pb-16"
      }
    >
      {!integrado && (
        <header className="flex flex-wrap items-start justify-between gap-4">
          <div className="space-y-2">
            <p className="text-xs font-semibold uppercase tracking-widest text-muted-foreground">
              Talento Humano · Uso interno
            </p>
            <h1 className="text-2xl font-semibold tracking-tight md:text-3xl">
              {titulos[vista]}
            </h1>
            <p className="max-w-3xl text-sm text-muted-foreground">
              Consulte los envíos y sus respaldos originales por convocatoria.
              La información declarada no constituye verificación de identidad
              civil ni calificación de méritos.
            </p>
          </div>
          {vista === "inicio" && (
            <div className="flex flex-wrap gap-2">
              <Button asChild variant="outline">
                <Link
                  to={`${RUTAS.dashboardReportes}${busquedaRevision(filtros)}`}
                >
                  Generar matriz
                </Link>
              </Button>
              <Button asChild>
                <Link
                  to={`${RUTAS.dashboardPostulaciones}${busquedaRevision(filtros)}`}
                >
                  Ver postulaciones
                </Link>
              </Button>
            </div>
          )}
        </header>
      )}
      <FiltrosRevisionPanel />
      {!filtros.periodo || error ? (
        <EstadoVacio
          titulo="Seleccione una convocatoria"
          descripcion="Elija un período, incluso cerrado, o el legado sin convocatoria. No se mezclarán expedientes de distintos períodos."
        />
      ) : (
        <div key={clave} className="space-y-6">
          {vista === "inicio" && <ConteosRevisionPanel filtros={filtros} />}
          {vista === "inicio" && (
            <div className="grid gap-4 md:grid-cols-2">
              {[
                {
                  ruta: RUTAS.dashboardPostulaciones,
                  titulo: "Revisar la nómina",
                  texto:
                    "Formulario íntegro, certificados y duración de las experiencias al momento del envío.",
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
          {vista === "reportes" && <NominaRevision filtros={filtros} matriz />}
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
    </Contenedor>
  )
}
