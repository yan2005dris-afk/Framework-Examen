import {
  CalendarDaysIcon,
  CopyIcon,
  ExternalLinkIcon,
  InfoIcon,
  Settings2Icon,
  Trash2Icon,
} from "lucide-react"
import { useState, type FormEvent } from "react"
import { Link, useSearchParams } from "react-router"
import { EstadoCargando, EstadoError } from "@/components/estados"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  useConvocatoriaAdminDetalle,
  useConvocatoriasAdmin,
  useGuardarConfiguracionConvocatoria,
} from "@/features/convocatorias/queries"
import {
  SECTORES,
  detalleErrorConvocatoria,
  enlacePublicoConvocatoria,
  estadoFase,
  validarConfiguracion,
} from "@/features/convocatorias/modelo"
import { useAhora } from "@/features/convocatorias/use-ahora"
import { RUTAS } from "@/lib/rutas"
import {
  NOMBRES_FASES_OFICIALES,
  type CodigoFase,
  type ConfiguracionConvocatoria,
  type ConvocatoriaAdmin,
  type FasePublica,
  type SectorConvocatoria,
} from "@/types/convocatoria"
import {
  fechaControlEcuadorAISO,
  fechaParaControlEcuador,
  formatearFechaEcuador,
} from "@/utils/fechas"

interface Edicion {
  origen: ConvocatoriaAdmin
  config: ConfiguracionConvocatoria
  revisar: boolean
}

/** FT15: una edición conserva la versión de origen aunque el refresco encuentre otra configuración. */
export function AdminFasesView() {
  const [params, setParams] = useSearchParams()
  const id = params.get("convocatoria")
  const lista = useConvocatoriasAdmin()
  const consulta = useConvocatoriaAdminDetalle(id)
  const guardar = useGuardarConfiguracionConvocatoria()
  const detalle = consulta.data
  const items = lista.data?.pages.flatMap((p) => p.items) ?? []
  const ahora = useAhora()
  const [edicion, setEdicion] = useState<Edicion | null>(null)
  const [error, setError] = useState("")
  const [aviso, setAviso] = useState("")
  const [copiado, setCopiado] = useState(false)
  const enlace =
    id && typeof window !== "undefined"
      ? enlacePublicoConvocatoria(window.location.origin, id)
      : ""
  const obsoleta = Boolean(
    edicion &&
    detalle &&
    (edicion.origen.version !== detalle.version || edicion.origen.id !== id),
  )

  function editar() {
    if (!detalle) return
    const {
      version,
      inicioAprobado,
      publicada,
      ef12Habilitado,
      sectores,
      fases,
    } = detalle
    setEdicion({
      origen: structuredClone(detalle),
      config: structuredClone({
        version,
        inicioAprobado,
        publicada,
        ef12Habilitado,
        sectores,
        fases,
      }),
      revisar: false,
    })
    setError("")
    setAviso("")
  }
  function cambiar(patch: Partial<ConfiguracionConvocatoria>) {
    if (edicion)
      setEdicion({
        ...edicion,
        config: { ...edicion.config, ...patch },
        revisar: false,
      })
  }
  function cambiarFase(indice: number, patch: Partial<FasePublica>) {
    if (edicion)
      cambiar({
        fases: edicion.config.fases.map((f, i) =>
          i === indice ? { ...f, ...patch } : f,
        ),
      })
  }
  function revisar(e: FormEvent) {
    e.preventDefault()
    if (!edicion) return
    const problema = validarConfiguracion(
      edicion.config,
      edicion.origen.apertura,
      edicion.origen.cierre,
    )
    if (problema) {
      setError(problema)
      return
    }
    setError("")
    setEdicion({ ...edicion, revisar: true })
  }
  async function confirmar() {
    if (!edicion || guardar.isPending || obsoleta) return
    try {
      const resultado = await guardar.mutateAsync({
        id: edicion.origen.id,
        config: edicion.config,
      })
      setEdicion(null)
      setError("")
      setAviso(
        `Configuración confirmada por el servidor. Versión ${resultado.version}.`,
      )
    } catch (err) {
      setError(detalleErrorConvocatoria(err))
      setEdicion({ ...edicion, revisar: false })
    }
  }
  async function copiar() {
    try {
      await navigator.clipboard.writeText(enlace)
      setCopiado(true)
    } catch {
      setAviso(
        "No se pudo copiar automáticamente. Seleccione y copie el enlace mostrado.",
      )
    }
  }

  return (
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-6 p-4 sm:p-6">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <p className="mb-2 font-mono text-xs text-muted-foreground">
            ADMINISTRACIÓN · EF-13
          </p>
          <h1 className="text-2xl font-semibold tracking-tight">
            Fases y publicación
          </h1>
          <p className="mt-2 text-sm text-muted-foreground">
            Revise el calendario, la visibilidad y la recepción de cada
            convocatoria por separado.
          </p>
        </div>
        <Button asChild variant="outline">
          <Link to={RUTAS.adminConvocatorias}>Volver a convocatorias</Link>
        </Button>
      </div>
      <div className="flex flex-wrap items-end gap-3">
        <div className="min-w-0 flex-1 space-y-2">
          <Label htmlFor="periodo-fases">Convocatoria de trabajo</Label>
          <select
            id="periodo-fases"
            className="h-10 w-full rounded-md border bg-background px-3 text-sm"
            value={id ?? ""}
            onChange={(e) => {
              setParams(e.target.value ? { convocatoria: e.target.value } : {})
              setEdicion(null)
              setError("")
              setAviso("")
              setCopiado(false)
            }}
          >
            <option value="">Seleccione una convocatoria</option>
            {id && !items.some((c) => c.id === id) && (
              <option value={id}>
                {detalle?.codigo ?? "Convocatoria del enlace"}
              </option>
            )}
            {items.map((c) => (
              <option key={c.id} value={c.id}>
                {c.codigo} · {c.titulo}
              </option>
            ))}
          </select>
        </div>
        {lista.hasNextPage && (
          <Button
            variant="outline"
            disabled={lista.isFetchingNextPage}
            onClick={() => void lista.fetchNextPage()}
          >
            Cargar más períodos
          </Button>
        )}
      </div>
      {lista.error && (
        <EstadoError
          error={lista.error}
          titulo="No se pudo actualizar el selector de períodos"
          onReintentar={() => void lista.refetch()}
        />
      )}
      {!id && (
        <Card>
          <CardContent className="space-y-3 py-10 text-center">
            <CalendarDaysIcon
              className="mx-auto size-8 text-muted-foreground"
              aria-hidden="true"
            />
            <h2 className="font-semibold">
              Seleccione el período que desea administrar
            </h2>
            <p className="text-sm text-muted-foreground">
              No se elige automáticamente una convocatoria activa. También puede
              consultar períodos cerrados.
            </p>
            {lista.isPending && <EstadoCargando mensaje="Cargando períodos…" />}
            {lista.isSuccess && !items.length && (
              <p className="text-sm">
                Aún no hay convocatorias. Registre una desde el listado.
              </p>
            )}
          </CardContent>
        </Card>
      )}
      {id && consulta.isPending && (
        <EstadoCargando mensaje="Consultando la configuración del período…" />
      )}
      {id && consulta.error && (
        <EstadoError
          error={consulta.error}
          titulo="No se pudo confirmar la configuración"
          onReintentar={() => void consulta.refetch()}
        />
      )}
      {aviso && (
        <p role="status" className="rounded-lg border bg-muted/40 p-4 text-sm">
          {aviso}
        </p>
      )}
      {detalle && (
        <>
          <Card>
            <CardHeader>
              <div className="flex flex-wrap items-start justify-between gap-4">
                <div>
                  <p className="mb-2 font-mono text-xs text-muted-foreground">
                    {detalle.codigo} · Versión {detalle.version}
                  </p>
                  <CardTitle>{detalle.titulo}</CardTitle>
                  <CardDescription className="mt-2">
                    {formatearFechaEcuador(detalle.apertura, {
                      incluirHora: true,
                    })}{" "}
                    —{" "}
                    {formatearFechaEcuador(detalle.cierre, {
                      incluirHora: true,
                    })}
                  </CardDescription>
                </div>
                <Button
                  onClick={editar}
                  disabled={
                    consulta.isError || consulta.isFetching || guardar.isPending
                  }
                >
                  <Settings2Icon aria-hidden="true" />
                  Editar configuración
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground">
                Sectores: {detalle.sectores.map((s) => SECTORES[s]).join(", ")}.
                Zona: America/Guayaquil (UTC−5).
              </p>
            </CardContent>
          </Card>
          <div className="grid gap-4 sm:grid-cols-3">
            {[
              [
                "Inicio del concurso",
                detalle.inicioAprobado ? "Aprobado" : "Pendiente de aprobación",
                "La aprobación se confirma expresamente por el administrador.",
              ],
              [
                "Portal público",
                detalle.publicada ? "Publicado" : "No publicado",
                "Visibilidad de esta convocatoria y sus fases publicadas.",
              ],
              [
                "Formulario EF-12",
                detalle.ef12Habilitado
                  ? "Recepción habilitada"
                  : "Recepción deshabilitada",
                "Además requiere fase de postulación habilitada y estar dentro del plazo.",
              ],
            ].map(([titulo, estado, texto]) => (
              <Card key={titulo}>
                <CardHeader>
                  <CardDescription>{titulo}</CardDescription>
                  <CardTitle className="text-base">{estado}</CardTitle>
                </CardHeader>
                <CardContent className="text-xs leading-relaxed text-muted-foreground">
                  {texto}
                </CardContent>
              </Card>
            ))}
          </div>
          <Card>
            <CardHeader>
              <CardTitle>Cronograma de fases</CardTitle>
              <CardDescription>
                No representa el estado de un formulario ni los resultados de
                evaluación.
              </CardDescription>
            </CardHeader>
            <CardContent>
              {detalle.fases.length ? (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead scope="col">Fase</TableHead>
                      <TableHead scope="col">Apertura · Ecuador</TableHead>
                      <TableHead scope="col">Cierre · Ecuador</TableHead>
                      <TableHead scope="col">Publicación</TableHead>
                      <TableHead scope="col">Estado temporal</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {detalle.fases.map((f) => (
                      <TableRow key={f.codigo}>
                        <TableCell className="whitespace-normal">
                          <p className="font-medium">{f.titulo}</p>
                          <p className="mt-1 font-mono text-xs text-muted-foreground">
                            {f.codigo}
                          </p>
                        </TableCell>
                        <TableCell className="whitespace-normal text-xs">
                          {formatearFechaEcuador(f.apertura, {
                            incluirHora: true,
                          })}
                        </TableCell>
                        <TableCell className="whitespace-normal text-xs">
                          {formatearFechaEcuador(f.cierre, {
                            incluirHora: true,
                          })}
                        </TableCell>
                        <TableCell>
                          <Badge variant="outline">
                            {f.publicada ? "Publicada" : "No publicada"}
                          </Badge>
                        </TableCell>
                        <TableCell className="text-xs">
                          {estadoFase(f, ahora)}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              ) : (
                <p className="py-6 text-sm text-muted-foreground">
                  Sin fases configuradas. Agregue sólo las etapas y fechas
                  aprobadas por la institución.
                </p>
              )}
            </CardContent>
          </Card>
          <Alert>
            <InfoIcon aria-hidden="true" />
            <AlertTitle>
              Los cambios requieren revisión y confirmación
            </AlertTitle>
            <AlertDescription>
              Se guardan con la versión {detalle.version} y quedan registrados
              por el servidor. Si otro responsable modifica el período, deberá
              recargar y revisar de nuevo; no se sobrescribe su configuración.
            </AlertDescription>
          </Alert>
          <Card>
            <CardHeader>
              <CardTitle>Enlace para portales IESS / BIESS</CardTitle>
              <CardDescription>
                El destino es la convocatoria pública exacta. No contiene una
                sesión ni da permisos administrativos.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              <Label htmlFor="enlace-publico">
                Enlace de integración (origen de este entorno)
              </Label>
              <Input id="enlace-publico" value={enlace} readOnly />
              <div className="flex flex-wrap gap-2">
                <Button variant="outline" onClick={() => void copiar()}>
                  <CopyIcon aria-hidden="true" />
                  {copiado ? "Enlace copiado" : "Copiar enlace"}
                </Button>
                <Button asChild variant="outline">
                  <a href={enlace} target="_blank" rel="noopener noreferrer">
                    Abrir en nueva pestaña{" "}
                    <ExternalLinkIcon aria-hidden="true" />
                  </a>
                </Button>
              </div>
              <p className="text-xs text-muted-foreground">
                Antes de integrarlo, use el dominio público de producción. El
                enlace sólo mostrará información si el período está aprobado y
                publicado. Su incorporación a sitios externos sigue pendiente de
                coordinación institucional.
              </p>
            </CardContent>
          </Card>
        </>
      )}
      <Dialog
        open={Boolean(edicion)}
        onOpenChange={(v) => {
          if (!v && !guardar.isPending) setEdicion(null)
        }}
      >
        <DialogContent className="max-h-[90svh] overflow-y-auto sm:max-w-3xl">
          <DialogHeader>
            <DialogTitle>
              {edicion?.revisar
                ? "Confirmar publicación y configuración"
                : "Editar configuración del período"}
            </DialogTitle>
            <DialogDescription>
              {edicion?.origen.codigo} · Versión de origen{" "}
              {edicion?.config.version}. Fechas en Ecuador (UTC−5).
            </DialogDescription>
          </DialogHeader>
          {error && (
            <Alert variant="destructive">
              <AlertTitle>No se guardaron los cambios</AlertTitle>
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}
          {obsoleta && (
            <Alert variant="destructive">
              <AlertTitle>La convocatoria cambió durante la edición</AlertTitle>
              <AlertDescription>
                Su borrador no se enviará sobre otra versión. Cierre este
                diálogo y abra nuevamente la configuración para revisar los
                cambios actuales.
              </AlertDescription>
            </Alert>
          )}
          {edicion &&
            (edicion.revisar ? (
              <div className="space-y-5">
                <Alert>
                  <InfoIcon aria-hidden="true" />
                  <AlertTitle>
                    Revise el estado que quedará publicado
                  </AlertTitle>
                  <AlertDescription>
                    Esta confirmación afecta sólo a {edicion.origen.codigo}. La
                    apertura/cierre del período se conserva; las fases no
                    publican resultados de evaluación.
                  </AlertDescription>
                </Alert>
                <dl className="grid gap-3 rounded-lg border p-4 text-sm sm:grid-cols-2">
                  {[
                    [
                      "Inicio aprobado",
                      edicion.config.inicioAprobado ? "Sí" : "No",
                    ],
                    [
                      "Convocatoria pública",
                      edicion.config.publicada ? "Sí" : "No",
                    ],
                    [
                      "EF-12 habilitado",
                      edicion.config.ef12Habilitado ? "Sí" : "No",
                    ],
                    [
                      "Sectores",
                      edicion.config.sectores
                        .map((s) => SECTORES[s])
                        .join(", "),
                    ],
                  ].map(([k, v]) => (
                    <div key={k}>
                      <dt className="text-muted-foreground">{k}</dt>
                      <dd className="font-medium">{v}</dd>
                    </div>
                  ))}
                </dl>
                <h3 className="text-sm font-semibold">
                  Fases que se conservarán ({edicion.config.fases.length})
                </h3>
                {edicion.config.fases.map((f) => (
                  <div
                    key={f.codigo}
                    className="space-y-1 rounded-lg border p-3 text-sm"
                  >
                    <p className="font-medium">{f.titulo}</p>
                    <p>
                      {formatearFechaEcuador(f.apertura, { incluirHora: true })}{" "}
                      — {formatearFechaEcuador(f.cierre, { incluirHora: true })}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      {f.publicada ? "Publicada" : "No publicada"} ·{" "}
                      {f.habilitada ? "Habilitada" : "Deshabilitada"}
                    </p>
                  </div>
                ))}
                {edicion.origen.fases.some(
                  (f) =>
                    !edicion.config.fases.some((n) => n.codigo === f.codigo),
                ) && (
                  <p className="text-sm text-destructive">
                    Se retirarán estas fases:{" "}
                    {edicion.origen.fases
                      .filter(
                        (f) =>
                          !edicion.config.fases.some(
                            (n) => n.codigo === f.codigo,
                          ),
                      )
                      .map((f) => f.titulo)
                      .join(", ")}
                    . El historial permanece en el servidor.
                  </p>
                )}
                <DialogFooter>
                  <Button
                    variant="outline"
                    disabled={guardar.isPending}
                    onClick={() => setEdicion({ ...edicion, revisar: false })}
                  >
                    Volver a editar
                  </Button>
                  <Button
                    disabled={guardar.isPending || obsoleta || consulta.isError}
                    onClick={() => void confirmar()}
                  >
                    {guardar.isPending ? "Confirmando…" : "Confirmar cambios"}
                  </Button>
                </DialogFooter>
              </div>
            ) : (
              <form className="space-y-6" onSubmit={revisar}>
                <fieldset
                  disabled={guardar.isPending || obsoleta || consulta.isError}
                  className="space-y-6"
                >
                  <legend className="sr-only">Configuración y fases</legend>
                  <div className="space-y-3">
                    {(
                      [
                        [
                          "inicioAprobado",
                          "Aprobar inicio",
                          "Confirma que existe aprobación institucional para este período.",
                        ],
                        [
                          "publicada",
                          "Publicar convocatoria",
                          "Hace visible el período en el portal público, tras aprobar su inicio.",
                        ],
                        [
                          "ef12Habilitado",
                          "Habilitar recepción EF-12",
                          "Requiere fase de postulación publicada y habilitada; el plazo se verifica en cada operación.",
                        ],
                      ] as const
                    ).map(([campo, titulo, texto]) => (
                      <div
                        key={campo}
                        className="flex items-start justify-between gap-4 rounded-lg border p-4"
                      >
                        <div>
                          <Label htmlFor={campo}>{titulo}</Label>
                          <p className="mt-1 text-xs leading-relaxed text-muted-foreground">
                            {texto}
                          </p>
                        </div>
                        <Switch
                          id={campo}
                          checked={edicion.config[campo]}
                          onCheckedChange={(v) => cambiar({ [campo]: v })}
                        />
                      </div>
                    ))}
                  </div>
                  <fieldset className="space-y-3">
                    <legend className="mb-2 text-sm font-semibold">
                      Sectores convocados
                    </legend>
                    {(Object.keys(SECTORES) as SectorConvocatoria[]).map(
                      (s) => (
                        <label
                          key={s}
                          className="flex items-center gap-3 text-sm"
                        >
                          <input
                            type="checkbox"
                            className="size-5 accent-primary"
                            checked={edicion.config.sectores.includes(s)}
                            onChange={(e) =>
                              cambiar({
                                sectores: e.target.checked
                                  ? [...edicion.config.sectores, s]
                                  : edicion.config.sectores.filter(
                                      (x) => x !== s,
                                    ),
                              })
                            }
                          />
                          {SECTORES[s]}
                        </label>
                      ),
                    )}
                  </fieldset>
                  <div className="space-y-4">
                    <h3 className="font-semibold">Fases del concurso</h3>
                    {edicion.config.fases.map((f, i) => (
                      <fieldset
                        key={f.codigo}
                        className="space-y-4 rounded-lg border p-4"
                      >
                        <legend className="px-1 text-sm font-medium">
                          {NOMBRES_FASES_OFICIALES[f.codigo]}
                        </legend>
                        <div className="flex items-center justify-between gap-3">
                          <span className="font-mono text-xs text-muted-foreground">
                            {f.codigo}
                          </span>
                          <Button
                            type="button"
                            variant="ghost"
                            size="sm"
                            onClick={() =>
                              cambiar({
                                fases: edicion.config.fases.filter(
                                  (_, n) => n !== i,
                                ),
                              })
                            }
                          >
                            <Trash2Icon aria-hidden="true" />
                            Retirar fase
                          </Button>
                        </div>
                        <div className="space-y-2">
                          <Label htmlFor={`titulo-${i}`}>
                            Título publicado
                          </Label>
                          <Input
                            id={`titulo-${i}`}
                            required
                            maxLength={200}
                            value={f.titulo}
                            onChange={(e) =>
                              cambiarFase(i, { titulo: e.target.value })
                            }
                          />
                        </div>
                        <div className="grid gap-3 sm:grid-cols-2">
                          {(["apertura", "cierre"] as const).map((campo) => (
                            <div key={campo} className="space-y-2">
                              <Label htmlFor={`${campo}-${i}`}>
                                {campo === "apertura" ? "Apertura" : "Cierre"} ·
                                Ecuador
                              </Label>
                              <Input
                                id={`${campo}-${i}`}
                                type="datetime-local"
                                required
                                min={fechaParaControlEcuador(
                                  edicion.origen.apertura,
                                )}
                                max={fechaParaControlEcuador(
                                  edicion.origen.cierre,
                                )}
                                value={fechaParaControlEcuador(f[campo])}
                                onChange={(e) =>
                                  cambiarFase(i, {
                                    [campo]:
                                      fechaControlEcuadorAISO(e.target.value) ??
                                      e.target.value,
                                  })
                                }
                              />
                            </div>
                          ))}
                        </div>
                        <div className="flex flex-wrap gap-6">
                          {(["publicada", "habilitada"] as const).map(
                            (campo) => (
                              <div
                                key={campo}
                                className="flex items-center gap-3"
                              >
                                <Switch
                                  id={`${campo}-${i}`}
                                  checked={f[campo]}
                                  onCheckedChange={(v) =>
                                    cambiarFase(i, { [campo]: v })
                                  }
                                />
                                <Label htmlFor={`${campo}-${i}`}>
                                  {campo === "publicada"
                                    ? "Publicada"
                                    : "Habilitada"}
                                </Label>
                              </div>
                            ),
                          )}
                        </div>
                      </fieldset>
                    ))}
                    <div className="space-y-2">
                      <Label htmlFor="agregar-fase">
                        Agregar etapa aprobada
                      </Label>
                      <select
                        id="agregar-fase"
                        value=""
                        className="h-10 w-full rounded-md border bg-background px-3 text-sm"
                        onChange={(e) => {
                          if (!e.target.value) return
                          const codigo = e.target.value as CodigoFase
                          cambiar({
                            fases: [
                              ...edicion.config.fases,
                              {
                                codigo,
                                titulo: NOMBRES_FASES_OFICIALES[codigo],
                                apertura: "",
                                cierre: "",
                                publicada: false,
                                habilitada: false,
                              },
                            ],
                          })
                        }}
                      >
                        <option value="">
                          Seleccione una fase para agregar
                        </option>
                        {(Object.keys(NOMBRES_FASES_OFICIALES) as CodigoFase[])
                          .filter(
                            (c) =>
                              !edicion.config.fases.some((f) => f.codigo === c),
                          )
                          .map((c) => (
                            <option key={c} value={c}>
                              {NOMBRES_FASES_OFICIALES[c]}
                            </option>
                          ))}
                      </select>
                      <p className="text-xs text-muted-foreground">
                        No se precargan cronogramas. Cada etapa nueva comienza
                        sin publicar ni habilitar.
                      </p>
                    </div>
                  </div>
                </fieldset>
                <DialogFooter>
                  <Button
                    type="button"
                    variant="outline"
                    disabled={guardar.isPending}
                    onClick={() => setEdicion(null)}
                  >
                    Cancelar
                  </Button>
                  <Button
                    type="submit"
                    disabled={guardar.isPending || obsoleta || consulta.isError}
                  >
                    Revisar cambios
                  </Button>
                </DialogFooter>
              </form>
            ))}
        </DialogContent>
      </Dialog>
    </div>
  )
}
