import { useState } from "react"
import { Link, useParams, useSearchParams } from "react-router"
import { ArrowLeftIcon, LockKeyholeIcon } from "lucide-react"
import { EstadoCargando, EstadoError } from "@/components/estados"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { ExpedienteSellado } from "@/features/revision/expediente-sellado"
import { DocumentosRevision } from "@/features/revision/documentos-revision"
import { EvaluacionMeritos } from "@/features/revision/evaluacion-meritos"
import { EvaluacionOposicion } from "@/features/evaluacion/evaluacion-oposicion"
import { PanelReconsideracionesAdmin } from "@/features/reconsideracion/panel-reconsideraciones-admin"
import {
  useCuentaRevision,
  useEventosRevision,
  useExpedienteRevision,
} from "@/features/revision/queries"
import {
  busquedaRevision,
  esObjetoRevision,
  fechaRevision,
  leerFiltrosRevision,
} from "@/features/revision/modelo"
import { esUuidRevision, type FiltrosRevision } from "@/lib/clientes/revision"
import { RUTAS } from "@/lib/rutas"
import { esErrorApi } from "@/lib/api-error"

function HistorialRevision({ id }: { id: string }) {
  const consulta = useEventosRevision(id)
  if (consulta.isPending)
    return <EstadoCargando mensaje="Consultando historial…" />
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
  const eventos = consulta.data?.pages.flatMap((p) => p.items) ?? []
  return (
    <div className="space-y-4">
      <ul className="space-y-2">
        {eventos.map((e) => (
          <li key={e.id} className="rounded-lg border p-3 text-sm">
            <p className="font-medium">
              {e.accion} · {e.resultado}
            </p>
            <p className="text-xs text-muted-foreground">
              {fechaRevision(e.createdAt)} · {e.rol} · actor{" "}
              {e.actorId || "no registrado"}
            </p>
            <p className="break-all text-xs text-muted-foreground">
              {e.tipoRecurso}: {e.recursoId}
            </p>
          </li>
        ))}
      </ul>
      {eventos.length === 0 && (
        <p className="text-sm text-muted-foreground">
          Sin eventos disponibles.
        </p>
      )}
      {consulta.isFetchNextPageError && (
        <EstadoError
          error={consulta.error}
          onReintentar={() => void consulta.fetchNextPage()}
        />
      )}
      {consulta.hasNextPage && (
        <Button
          variant="outline"
          disabled={consulta.isFetching}
          onClick={() => void consulta.fetchNextPage()}
        >
          Cargar más eventos
        </Button>
      )}
    </div>
  )
}

function DetalleRevision({
  id,
  filtros,
}: {
  id: string
  filtros: FiltrosRevision
}) {
  const consulta = useExpedienteRevision(id)
  const [historial, mostrarHistorial] = useState(false)
  if (consulta.isPending)
    return <EstadoCargando mensaje="Abriendo el expediente original…" />
  if (consulta.isError)
    return (
      <EstadoError
        error={consulta.error}
        titulo="No se puede abrir el expediente"
        onReintentar={() => void consulta.refetch()}
      />
    )
  const expediente = consulta.data
  if (
    expediente.id !== id.toLowerCase() ||
    expediente.status !== "submitted" ||
    !expediente.submittedAt ||
    !Number.isFinite(Date.parse(expediente.submittedAt)) ||
    !/^[a-f0-9]{64}$/i.test(expediente.hash ?? "") ||
    !esObjetoRevision(expediente.content)
  )
    return (
      <EstadoError
        titulo="Expediente no sellado"
        mensaje="La respuesta no contiene un envío sellado válido. No se mostrará como histórico confirmado."
      />
    )
  const coincidePeriodo =
    !filtros.periodo ||
    (filtros.periodo === "legado"
      ? !expediente.convocatoriaId
      : expediente.convocatoriaId === filtros.periodo)
  if (!coincidePeriodo)
    return (
      <EstadoError
        titulo="La convocatoria no coincide"
        mensaje="Este expediente no pertenece al período seleccionado. Regrese a la nómina para consultar el histórico correcto."
      />
    )
  return (
    <div className="space-y-7">
      <header className="space-y-4 rounded-xl border bg-card p-5">
        <div className="flex flex-wrap items-center gap-3">
          <LockKeyholeIcon className="size-5 text-primary" />
          <h1 className="text-xl font-semibold">
            Expediente {expediente.code}
          </h1>
          <Badge variant="secondary">Enviado y sellado · Solo lectura</Badge>
        </div>
        <p className="text-sm text-muted-foreground">
          Convocatoria de origen:{" "}
          {expediente.convocatoriaId || "Legado sin convocatoria"}
          <br />
          Enviado: {fechaRevision(expediente.submittedAt)} · Ecuador
        </p>
        <div className="rounded-lg bg-muted p-3">
          <p className="text-xs font-medium">
            SHA-256 del envío (informado por el servidor)
          </p>
          <p className="mt-1 break-all font-mono text-xs">{expediente.hash}</p>
        </div>
        <p className="text-xs text-muted-foreground">
          El sello preserva el contenido enviado; no es una firma electrónica
          certificada ni un resultado de fase. No se usan datos del perfil
          actual para completar este histórico.
        </p>
      </header>
      <ExpedienteSellado
        contenido={expediente.content}
        enviadoEn={expediente.submittedAt}
      />
      <DocumentosRevision id={id} />
      <EvaluacionMeritos
        submissionId={id}
        convocatoriaId={expediente.convocatoriaId}
      />
      <EvaluacionOposicion
        submissionId={id}
        convocatoriaId={expediente.convocatoriaId}
      />
      <PanelReconsideracionesAdmin submissionId={id} />
      <section className="space-y-4 border-t pt-6">
        <h2 className="text-lg font-semibold">
          Historial de acceso y acciones
        </h2>
        <p className="text-sm text-muted-foreground">
          Esta consulta también queda auditada. El historial no representa
          calificaciones de méritos.
        </p>
        <Button variant="outline" onClick={() => mostrarHistorial(!historial)}>
          {historial ? "Ocultar historial" : "Consultar historial"}
        </Button>
        {historial && <HistorialRevision id={id} />}
      </section>
    </div>
  )
}

export function DashboardPostulacionDetalleView() {
  const { id = "" } = useParams()
  const [params] = useSearchParams()
  const { filtros, error } = leerFiltrosRevision(params)
  const cuenta = useCuentaRevision()
  return (
    <main className="mx-auto w-full max-w-5xl space-y-6 p-4 md:p-8">
      <Button variant="ghost" asChild>
        <Link
          to={`${RUTAS.dashboardPostulaciones}${busquedaRevision(filtros)}`}
        >
          <ArrowLeftIcon />
          Volver a la nómina
        </Link>
      </Button>
      {!esUuidRevision(id) || error ? (
        <EstadoError
          titulo="Enlace de expediente no válido"
          mensaje={error ?? "El identificador de la postulación no es válido."}
        />
      ) : (
        <DetalleRevision
          key={`${cuenta}:${id}:${busquedaRevision(filtros)}`}
          id={id}
          filtros={filtros}
        />
      )}
    </main>
  )
}
