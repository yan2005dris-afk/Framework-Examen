import { CalendarDaysIcon, PlusIcon } from "lucide-react"
import { useState, type FormEvent } from "react"
import { Link, useNavigate } from "react-router"
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  useConvocatoriasAdmin,
  useCrearConvocatoria,
} from "@/features/convocatorias/queries"
import {
  SECTORES,
  detalleErrorConvocatoria,
} from "@/features/convocatorias/modelo"
import { useAhora } from "@/features/convocatorias/use-ahora"
import { RUTAS } from "@/lib/rutas"
import type {
  CrearConvocatoriaPayload,
  SectorConvocatoria,
} from "@/types/convocatoria"
import { fechaControlEcuadorAISO, formatearFechaEcuador } from "@/utils/fechas"

type Filtro = "todas" | "programadas" | "en_plazo" | "cerradas"
const VACIO = {
  codigo: "",
  titulo: "",
  apertura: "",
  cierre: "",
  sectores: [] as SectorConvocatoria[],
}

/** FT15: listado histórico y alta sin fechas precargadas; no confunde ventana temporal con publicación o resultados. */
export function AdminConvocatoriasView() {
  const lista = useConvocatoriasAdmin()
  const crear = useCrearConvocatoria()
  const navegar = useNavigate()
  const ahora = useAhora()
  const [filtro, setFiltro] = useState<Filtro>("todas")
  const [abierto, setAbierto] = useState(false)
  const [form, setForm] = useState(VACIO)
  const [propuesta, setPropuesta] = useState<CrearConvocatoriaPayload | null>(
    null,
  )
  const [error, setError] = useState("")
  const items = lista.data?.pages.flatMap((p) => p.items) ?? []
  const estado = (c: { apertura: string; cierre: string }) =>
    ahora < Date.parse(c.apertura)
      ? "programadas"
      : ahora >= Date.parse(c.cierre)
        ? "cerradas"
        : "en_plazo"
  const visibles = items.filter(
    (c) => filtro === "todas" || estado(c) === filtro,
  )

  function revisar(e: FormEvent) {
    e.preventDefault()
    const apertura = fechaControlEcuadorAISO(form.apertura),
      cierre = fechaControlEcuadorAISO(form.cierre)
    if (!apertura || !cierre || apertura >= cierre) {
      setError(
        "Indique fechas válidas: el cierre debe ser posterior a la apertura, en hora de Ecuador.",
      )
      return
    }
    if (!form.codigo.trim() || !form.titulo.trim() || !form.sectores.length) {
      setError(
        "Complete código, título y al menos un sector de representación.",
      )
      return
    }
    setError("")
    setPropuesta({
      codigo: form.codigo.trim(),
      titulo: form.titulo.trim(),
      apertura,
      cierre,
      sectores: [...form.sectores],
      fases: [],
    })
  }
  async function confirmar() {
    if (!propuesta || crear.isPending) return
    try {
      const creada = await crear.mutateAsync(propuesta)
      setAbierto(false)
      setPropuesta(null)
      setForm(VACIO)
      navegar(
        `${RUTAS.adminFases}?convocatoria=${encodeURIComponent(creada.id)}`,
      )
    } catch (err) {
      setError(detalleErrorConvocatoria(err))
      setPropuesta(null)
    }
  }

  return (
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-6 p-4 sm:p-6">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <p className="mb-2 font-mono text-xs text-muted-foreground">
            ADMINISTRACIÓN
          </p>
          <h1 className="text-2xl font-semibold tracking-tight">
            Convocatorias del concurso
          </h1>
          <p className="mt-2 max-w-2xl text-sm text-muted-foreground">
            Consulte períodos y configure su publicación. Los expedientes y
            resultados pertenecen siempre a su convocatoria de origen.
          </p>
        </div>
        <Button
          onClick={() => {
            setForm(VACIO)
            setPropuesta(null)
            setError("")
            setAbierto(true)
          }}
        >
          <PlusIcon aria-hidden="true" />
          Nueva convocatoria
        </Button>
      </div>
      <div className="grid gap-4 sm:grid-cols-3">
        {[
          ["Períodos cargados", items.length],
          [
            "Dentro de fechas",
            items.filter((c) => estado(c) === "en_plazo").length,
          ],
          [
            "Períodos cerrados",
            items.filter((c) => estado(c) === "cerradas").length,
          ],
        ].map(([nombre, valor]) => (
          <Card key={nombre}>
            <CardHeader>
              <CardDescription>{nombre}</CardDescription>
              <CardTitle className="text-3xl tabular-nums">{valor}</CardTitle>
            </CardHeader>
          </Card>
        ))}
      </div>
      <p className="text-xs text-muted-foreground">
        Resumen de los períodos cargados, no un total global. Estar dentro de
        fechas no implica aprobación ni habilitación del formulario de
        postulación.
      </p>
      <Card>
        <CardHeader className="gap-4">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <CardTitle>Histórico de convocatorias</CardTitle>
              <CardDescription className="mt-1">
                Apertura y cierre en America/Guayaquil (UTC−5).
              </CardDescription>
            </div>
            <div>
              <Label htmlFor="filtro-periodos">Mostrar períodos</Label>
              <select
                id="filtro-periodos"
                value={filtro}
                onChange={(e) => setFiltro(e.target.value as Filtro)}
                className="mt-1 h-10 rounded-md border bg-background px-3 text-sm"
              >
                <option value="todas">Todos los cargados</option>
                <option value="programadas">Programados</option>
                <option value="en_plazo">Dentro de fechas</option>
                <option value="cerradas">Cerrados</option>
              </select>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {lista.isPending ? (
            <EstadoCargando mensaje="Cargando convocatorias…" />
          ) : lista.error ? (
            <EstadoError
              error={lista.error}
              onReintentar={() => void lista.refetch()}
            />
          ) : (
            <>
              {visibles.length ? (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead scope="col">Convocatoria</TableHead>
                      <TableHead scope="col">Apertura</TableHead>
                      <TableHead scope="col">Cierre</TableHead>
                      <TableHead scope="col">Calendario</TableHead>
                      <TableHead scope="col" className="text-right">
                        Acciones
                      </TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {visibles.map((c) => (
                      <TableRow key={c.id}>
                        <TableCell className="max-w-sm whitespace-normal">
                          <p className="font-mono text-xs">{c.codigo}</p>
                          <p className="mt-1 text-sm font-medium">{c.titulo}</p>
                        </TableCell>
                        <TableCell className="whitespace-normal text-xs">
                          {formatearFechaEcuador(c.apertura, {
                            incluirHora: true,
                          })}
                        </TableCell>
                        <TableCell className="whitespace-normal text-xs">
                          {formatearFechaEcuador(c.cierre, {
                            incluirHora: true,
                          })}
                        </TableCell>
                        <TableCell>
                          <Badge variant="outline">
                            {estado(c) === "programadas"
                              ? "Programada"
                              : estado(c) === "cerradas"
                                ? "Cerrada"
                                : "Dentro de fechas"}
                          </Badge>
                        </TableCell>
                        <TableCell>
                          <div className="flex flex-wrap justify-end gap-2">
                            <Button asChild variant="outline" size="sm">
                              <Link
                                to={`${RUTAS.adminFases}?convocatoria=${encodeURIComponent(c.id)}`}
                              >
                                Configurar fases
                              </Link>
                            </Button>
                            <Button asChild variant="ghost" size="sm">
                              <Link
                                to={`${RUTAS.dashboardPostulaciones}?convocatoria=${encodeURIComponent(c.id)}`}
                              >
                                Expedientes
                              </Link>
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              ) : (
                <div className="space-y-3 py-10 text-center">
                  <CalendarDaysIcon
                    className="mx-auto size-8 text-muted-foreground"
                    aria-hidden="true"
                  />
                  <h3 className="font-medium">
                    {items.length
                      ? "No hay períodos con este filtro"
                      : "Aún no hay convocatorias registradas"}
                  </h3>
                  <p className="text-sm text-muted-foreground">
                    {items.length
                      ? "Cambie el filtro o cargue más períodos del histórico."
                      : "Cree el período y después revise su configuración antes de publicarlo."}
                  </p>
                </div>
              )}
            </>
          )}
          {lista.hasNextPage && (
            <Button
              className="mt-5"
              variant="outline"
              disabled={lista.isFetchingNextPage}
              onClick={() => void lista.fetchNextPage()}
            >
              {lista.isFetchingNextPage ? "Cargando…" : "Cargar más períodos"}
            </Button>
          )}
        </CardContent>
      </Card>
      <Alert>
        <CalendarDaysIcon aria-hidden="true" />
        <AlertTitle>
          Publicación, recepción y evaluación son estados distintos
        </AlertTitle>
        <AlertDescription>
          Desde «Configurar fases» puede revisar la aprobación, la visibilidad y
          la recepción. Aquí no se presentan puntajes ni resultados de
          evaluación.
        </AlertDescription>
      </Alert>
      <Dialog
        open={abierto}
        onOpenChange={(v) => {
          if (!crear.isPending) setAbierto(v)
        }}
      >
        <DialogContent className="max-h-[90svh] overflow-y-auto sm:max-w-xl">
          <DialogHeader>
            <DialogTitle>
              {propuesta
                ? "Confirmar nueva convocatoria"
                : "Nueva convocatoria"}
            </DialogTitle>
            <DialogDescription>
              El período se crea sin aprobar, sin publicar y con el formulario
              de postulación deshabilitado. No se publica automáticamente en
              portales externos.
            </DialogDescription>
          </DialogHeader>
          {error && (
            <Alert variant="destructive">
              <AlertTitle>No se pudo registrar</AlertTitle>
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}
          {propuesta ? (
            <div className="space-y-5">
              <dl className="space-y-3 text-sm">
                <div>
                  <dt className="text-muted-foreground">Código y título</dt>
                  <dd className="font-medium">
                    {propuesta.codigo} · {propuesta.titulo}
                  </dd>
                </div>
                <div>
                  <dt className="text-muted-foreground">Apertura</dt>
                  <dd>
                    {formatearFechaEcuador(propuesta.apertura, {
                      incluirHora: true,
                    })}
                  </dd>
                </div>
                <div>
                  <dt className="text-muted-foreground">Cierre</dt>
                  <dd>
                    {formatearFechaEcuador(propuesta.cierre, {
                      incluirHora: true,
                    })}
                  </dd>
                </div>
                <div>
                  <dt className="text-muted-foreground">Sectores</dt>
                  <dd>
                    {propuesta.sectores.map((s) => SECTORES[s]).join(", ")}
                  </dd>
                </div>
              </dl>
              <DialogFooter>
                <Button
                  variant="outline"
                  disabled={crear.isPending}
                  onClick={() => setPropuesta(null)}
                >
                  Volver a editar
                </Button>
                <Button
                  disabled={crear.isPending}
                  onClick={() => void confirmar()}
                >
                  {crear.isPending ? "Registrando…" : "Confirmar creación"}
                </Button>
              </DialogFooter>
            </div>
          ) : (
            <form className="space-y-5" onSubmit={revisar}>
              <div className="space-y-2">
                <Label htmlFor="codigo">Código del período</Label>
                <Input
                  id="codigo"
                  required
                  maxLength={64}
                  value={form.codigo}
                  onChange={(e) => setForm({ ...form, codigo: e.target.value })}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="titulo">Título de la convocatoria</Label>
                <Input
                  id="titulo"
                  required
                  maxLength={200}
                  value={form.titulo}
                  onChange={(e) => setForm({ ...form, titulo: e.target.value })}
                />
              </div>
              <div className="grid gap-4 sm:grid-cols-2">
                {(["apertura", "cierre"] as const).map((campo) => (
                  <div key={campo} className="space-y-2">
                    <Label htmlFor={campo}>
                      {campo === "apertura" ? "Apertura" : "Cierre"} (Ecuador
                      UTC−5)
                    </Label>
                    <Input
                      id={campo}
                      type="datetime-local"
                      required
                      value={form[campo]}
                      onChange={(e) =>
                        setForm({ ...form, [campo]: e.target.value })
                      }
                    />
                  </div>
                ))}
              </div>
              <fieldset className="space-y-3">
                <legend className="mb-2 text-sm font-medium">
                  Sectores convocados
                </legend>
                {(Object.keys(SECTORES) as SectorConvocatoria[]).map((s) => (
                  <label key={s} className="flex items-center gap-3 text-sm">
                    <input
                      type="checkbox"
                      className="size-5 accent-primary"
                      checked={form.sectores.includes(s)}
                      onChange={(e) =>
                        setForm({
                          ...form,
                          sectores: e.target.checked
                            ? [...form.sectores, s]
                            : form.sectores.filter((x) => x !== s),
                        })
                      }
                    />
                    {SECTORES[s]}
                  </label>
                ))}
              </fieldset>
              <DialogFooter>
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => setAbierto(false)}
                >
                  Cancelar
                </Button>
                <Button type="submit">Revisar creación</Button>
              </DialogFooter>
            </form>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}
