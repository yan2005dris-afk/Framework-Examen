import {
  AlertTriangleIcon,
  CalendarDaysIcon,
  CheckCircle2Icon,
  ClockIcon,
  LockIcon,
  PlusIcon,
} from "lucide-react"
import { useMemo, useState } from "react"
import { Link } from "react-router"
import { toast } from "sonner"

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
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Textarea } from "@/components/ui/textarea"
import {
  useActualizarEstadoConvocatoria,
  useConvocatoriasAdmin,
  useCrearConvocatoria,
} from "@/features/convocatorias/queries"
import { RUTAS } from "@/lib/rutas"
import type {
  ConvocatoriaDetalle,
  EstadoConvocatoria,
} from "@/types/convocatoria"
import { formatearFechaEcuador, obtenerEstadoTemporal } from "@/utils/fechas"

type TabFiltro = "todas" | "activas" | "cerradas"

/**
 * Panel de administración y programación de convocatorias (FT15).
 * Permite a la Comisión Especial de Selección publicar convocatorias,
 * programar calendarios, aprobar aperturas y declarar cierres oficiales.
 */
export function AdminConvocatoriasView() {
  const {
    data: paginaConvocatorias,
    isLoading,
    error,
    refetch,
  } = useConvocatoriasAdmin()
  const mutacionCrear = useCrearConvocatoria()
  const mutacionActualizar = useActualizarEstadoConvocatoria()

  const [tabActual, setTabActual] = useState<TabFiltro>("todas")
  const [modalCrearAbierto, setModalCrearAbierto] = useState(false)
  const [convocatoriaParaEditar, setConvocatoriaParaEditar] =
    useState<ConvocatoriaDetalle | null>(null)

  // Estado del formulario de nueva convocatoria
  const [nuevoCodigo, setNuevoCodigo] = useState("")
  const [nuevoTitulo, setNuevoTitulo] = useState("")
  const [nuevaResolucion, setNuevaResolucion] = useState("Resolución No. C.D. ")
  const [nuevoPeriodo, setNuevoPeriodo] = useState("2026 – 2030")
  const [nuevaApertura, setNuevaApertura] = useState("2026-11-01T08:00")
  const [nuevoCierre, setNuevoCierre] = useState("2026-11-30T23:59")
  const [nuevaDescripcion, setNuevaDescripcion] = useState("")

  // Estado del formulario de cambio de estado / cierre
  const [nuevoEstado, setNuevoEstado] = useState<EstadoConvocatoria>("en_curso")
  const [motivoCambio, setMotivoCambio] = useState("")

  const items = useMemo(
    () => paginaConvocatorias?.items ?? [],
    [paginaConvocatorias],
  )

  const itemsFiltrados = useMemo(() => {
    if (tabActual === "activas") {
      return items.filter((c) => !c.cerrada && c.estado !== "archivada")
    }
    if (tabActual === "cerradas") {
      return items.filter(
        (c) => c.cerrada || c.estado === "cerrada" || c.estado === "archivada",
      )
    }
    return items
  }, [items, tabActual])

  // Métricas rápidas
  const totalConvocatorias = items.length
  const activasCount = items.filter(
    (c) => !c.cerrada && c.estado !== "archivada",
  ).length
  const cerradasCount = items.filter(
    (c) => c.cerrada || c.estado === "cerrada",
  ).length

  function abrirModalCambioEstado(c: ConvocatoriaDetalle) {
    setConvocatoriaParaEditar(c)
    setNuevoEstado(c.cerrada ? "cerrada" : c.estado)
    setMotivoCambio("")
  }

  async function handleGuardarNuevaConvocatoria(e: React.FormEvent) {
    e.preventDefault()
    if (!nuevoCodigo.trim() || !nuevoTitulo.trim()) {
      toast.error("El código y el título oficial son obligatorios.")
      return
    }

    const aperturaIso = new Date(nuevaApertura).toISOString()
    const cierreIso = new Date(nuevoCierre).toISOString()

    if (new Date(cierreIso) <= new Date(aperturaIso)) {
      toast.error(
        "La fecha de cierre debe ser posterior a la fecha de apertura.",
      )
      return
    }

    try {
      await mutacionCrear.mutateAsync({
        codigo: nuevoCodigo.trim().toUpperCase(),
        titulo: nuevoTitulo.trim(),
        resolucion: nuevaResolucion.trim(),
        periodo: nuevoPeriodo.trim(),
        apertura: aperturaIso,
        cierre: cierreIso,
        descripcion: nuevaDescripcion.trim(),
      })
      toast.success("Convocatoria creada y programada con éxito.")
      setModalCrearAbierto(false)
      // Limpiar campos
      setNuevoCodigo("")
      setNuevoTitulo("")
      setNuevaDescripcion("")
    } catch (err) {
      toast.error(
        err instanceof Error
          ? err.message
          : "Error al registrar la convocatoria en el servidor.",
      )
    }
  }

  async function handleConfirmarCambioEstado() {
    if (!convocatoriaParaEditar) return
    if (!motivoCambio.trim()) {
      toast.error(
        "Debe ingresar un motivo institucional para el registro de auditoría.",
      )
      return
    }

    const esCierre = nuevoEstado === "cerrada"

    try {
      await mutacionActualizar.mutateAsync({
        id: convocatoriaParaEditar.id,
        payload: {
          estado: nuevoEstado,
          cerrada: esCierre,
          postulacionHabilitada: !esCierre,
          motivo: motivoCambio.trim(),
        },
      })
      toast.success(
        esCierre
          ? "Convocatoria cerrada oficialmente. Postulaciones bloqueadas."
          : `Estado de la convocatoria actualizado a ${nuevoEstado}.`,
      )
      setConvocatoriaParaEditar(null)
    } catch (err) {
      toast.error(
        err instanceof Error
          ? err.message
          : "El servidor rechazó la operación de cambio de estado.",
      )
    }
  }

  if (isLoading) {
    return (
      <div className="flex flex-1 items-center justify-center p-8">
        <EstadoCargando mensaje="Cargando convocatorias institucionales…" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex flex-1 flex-col p-6">
        <EstadoError
          error={error}
          onReintentar={() => void refetch()}
          titulo="Error al cargar convocatorias"
          mensaje="No se pudo obtener el listado oficial de convocatorias desde el servidor."
        />
      </div>
    )
  }

  return (
    <div className="flex flex-1 flex-col gap-6 p-6">
      {/* Cabecera y acciones */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <div className="flex items-center gap-2">
            <h1 className="text-2xl font-bold tracking-tight text-foreground">
              Gestión de Convocatorias
            </h1>
            <Badge variant="outline" className="border-primary/50 text-primary">
              Comisión de Selección
            </Badge>
          </div>
          <p className="text-sm text-muted-foreground">
            Configuración oficial, programación de cronogramas y control de
            apertura del Directorio del BIESS.
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <Button
            onClick={() => setModalCrearAbierto(true)}
            className="flex items-center gap-1.5"
          >
            <PlusIcon className="size-4" />
            <span>Nueva convocatoria</span>
          </Button>
          <Button
            asChild
            variant="outline"
            className="flex items-center gap-1.5"
          >
            <Link to={RUTAS.adminFases}>
              <ClockIcon className="size-4" />
              <span>Ver cronograma y fases</span>
            </Link>
          </Button>
        </div>
      </div>

      {/* KPI Stats */}
      <div className="grid gap-4 sm:grid-cols-3">
        <Card>
          <CardHeader className="pb-2">
            <CardDescription className="text-xs uppercase font-semibold">
              Total Convocatorias
            </CardDescription>
            <CardTitle className="text-2xl font-bold">
              {totalConvocatorias}
            </CardTitle>
          </CardHeader>
          <CardContent className="text-xs text-muted-foreground">
            Registradas en el sistema institucional
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardDescription className="text-xs uppercase font-semibold">
              Convocatorias Activas
            </CardDescription>
            <CardTitle className="text-2xl font-bold text-emerald-600 dark:text-emerald-400">
              {activasCount}
            </CardTitle>
          </CardHeader>
          <CardContent className="text-xs text-muted-foreground">
            En período vigente de postulación o evaluación
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardDescription className="text-xs uppercase font-semibold">
              Cerradas / Históricas
            </CardDescription>
            <CardTitle className="text-2xl font-bold text-muted-foreground">
              {cerradasCount}
            </CardTitle>
          </CardHeader>
          <CardContent className="text-xs text-muted-foreground">
            Concluidas con expedientes sellados
          </CardContent>
        </Card>
      </div>

      {/* Filtros de pestaña */}
      <div className="flex items-center justify-between border-b pb-2">
        <Tabs
          value={tabActual}
          onValueChange={(v) => setTabActual(v as TabFiltro)}
          className="w-full"
        >
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <TabsList>
              <TabsTrigger value="todas">
                Todas ({totalConvocatorias})
              </TabsTrigger>
              <TabsTrigger value="activas">
                Activas ({activasCount})
              </TabsTrigger>
              <TabsTrigger value="cerradas">
                Cerradas / Históricas ({cerradasCount})
              </TabsTrigger>
            </TabsList>
          </div>
        </Tabs>
      </div>

      {/* Tabla de convocatorias */}
      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-[180px]">Código y Período</TableHead>
                <TableHead>Título Oficial y Resolución</TableHead>
                <TableHead>Apertura y Cierre (ECT)</TableHead>
                <TableHead className="w-[130px]">Estado Concurso</TableHead>
                <TableHead className="w-[140px]">Formulario EF-12</TableHead>
                <TableHead className="text-right">Acciones</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {itemsFiltrados.length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={6}
                    className="h-32 text-center text-muted-foreground"
                  >
                    No se encontraron convocatorias en esta categoría.
                  </TableCell>
                </TableRow>
              ) : (
                itemsFiltrados.map((c) => {
                  const temporal = obtenerEstadoTemporal(
                    c.apertura,
                    c.cierre,
                    c.cerrada,
                  )
                  return (
                    <TableRow key={c.id}>
                      <TableCell className="font-medium">
                        <div className="space-y-1">
                          <span className="font-mono text-sm font-bold text-primary">
                            {c.codigo}
                          </span>
                          <p className="text-xs text-muted-foreground">
                            {c.periodo}
                          </p>
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="space-y-1">
                          <p className="font-semibold text-foreground line-clamp-2">
                            {c.titulo}
                          </p>
                          <span className="inline-block rounded bg-muted px-1.5 py-0.5 text-2xs text-muted-foreground">
                            {c.resolucion}
                          </span>
                        </div>
                      </TableCell>
                      <TableCell className="text-xs">
                        <div className="space-y-1">
                          <p>
                            <span className="text-muted-foreground">
                              Desde:
                            </span>{" "}
                            {formatearFechaEcuador(c.apertura, {
                              incluirHora: true,
                            })}
                          </p>
                          <p>
                            <span className="text-muted-foreground">
                              Hasta:
                            </span>{" "}
                            {formatearFechaEcuador(c.cierre, {
                              incluirHora: true,
                            })}
                          </p>
                        </div>
                      </TableCell>
                      <TableCell>
                        {c.cerrada ? (
                          <Badge variant="destructive" className="gap-1">
                            <LockIcon className="size-3" />
                            <span>Cerrada</span>
                          </Badge>
                        ) : temporal === "programada" ? (
                          <Badge
                            variant="outline"
                            className="border-amber-500 text-amber-700 dark:text-amber-400 gap-1"
                          >
                            <CalendarDaysIcon className="size-3" />
                            <span>Programada</span>
                          </Badge>
                        ) : (
                          <Badge className="bg-emerald-600 hover:bg-emerald-700 gap-1">
                            <CheckCircle2Icon className="size-3" />
                            <span>En curso</span>
                          </Badge>
                        )}
                      </TableCell>
                      <TableCell>
                        {c.postulacionHabilitada && !c.cerrada ? (
                          <span className="inline-flex items-center gap-1.5 text-xs font-semibold text-emerald-600 dark:text-emerald-400">
                            <span className="size-2 rounded-full bg-emerald-600 animate-pulse" />
                            Habilitado
                          </span>
                        ) : (
                          <span className="inline-flex items-center gap-1.5 text-xs text-muted-foreground">
                            <span className="size-2 rounded-full bg-muted-foreground" />
                            Bloqueado
                          </span>
                        )}
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex items-center justify-end gap-1.5">
                          <Button
                            asChild
                            variant="ghost"
                            size="sm"
                            className="text-xs"
                          >
                            <Link
                              to={`${RUTAS.adminFases}?convocatoria=${c.id}`}
                            >
                              Fases
                            </Link>
                          </Button>
                          <Button
                            asChild
                            variant="ghost"
                            size="sm"
                            className="text-xs"
                          >
                            <Link
                              to={`${RUTAS.dashboardPostulaciones}?convocatoriaId=${c.id}`}
                            >
                              Postulaciones
                            </Link>
                          </Button>
                          <Button
                            variant="outline"
                            size="sm"
                            className="text-xs"
                            onClick={() => abrirModalCambioEstado(c)}
                          >
                            Estado
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  )
                })
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Modal Crear Convocatoria */}
      <Dialog open={modalCrearAbierto} onOpenChange={setModalCrearAbierto}>
        <DialogContent className="max-w-xl">
          <DialogHeader>
            <DialogTitle>Nueva Convocatoria Oficial</DialogTitle>
            <DialogDescription>
              Configure el período y los datos de aprobación del Concurso de
              Méritos y Oposición.
            </DialogDescription>
          </DialogHeader>

          <form onSubmit={handleGuardarNuevaConvocatoria} className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-1.5">
                <Label htmlFor="codigo">Código oficial *</Label>
                <Input
                  id="codigo"
                  placeholder="ej. BIESS-DIR-2027"
                  value={nuevoCodigo}
                  onChange={(e) => setNuevoCodigo(e.target.value)}
                  required
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="periodo">Período *</Label>
                <Input
                  id="periodo"
                  placeholder="ej. 2026 – 2030"
                  value={nuevoPeriodo}
                  onChange={(e) => setNuevoPeriodo(e.target.value)}
                  required
                />
              </div>
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="titulo">Título oficial del concurso *</Label>
              <Input
                id="titulo"
                placeholder="Concurso Público de Méritos y Oposición…"
                value={nuevoTitulo}
                onChange={(e) => setNuevoTitulo(e.target.value)}
                required
              />
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="resolucion">Resolución de aprobación</Label>
              <Input
                id="resolucion"
                placeholder="Resolución No. C.D. 701"
                value={nuevaResolucion}
                onChange={(e) => setNuevaResolucion(e.target.value)}
              />
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-1.5">
                <Label htmlFor="apertura">Fecha de apertura (ECT) *</Label>
                <Input
                  id="apertura"
                  type="datetime-local"
                  value={nuevaApertura}
                  onChange={(e) => setNuevaApertura(e.target.value)}
                  required
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="cierre">Fecha de cierre (ECT) *</Label>
                <Input
                  id="cierre"
                  type="datetime-local"
                  value={nuevoCierre}
                  onChange={(e) => setNuevoCierre(e.target.value)}
                  required
                />
              </div>
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="descripcion">Descripción o antecedentes</Label>
              <Textarea
                id="descripcion"
                placeholder="Detalle o normativa aplicable…"
                rows={3}
                value={nuevaDescripcion}
                onChange={(e) => setNuevaDescripcion(e.target.value)}
              />
            </div>

            <DialogFooter className="pt-2">
              <Button
                type="button"
                variant="outline"
                onClick={() => setModalCrearAbierto(false)}
              >
                Cancelar
              </Button>
              <Button type="submit" disabled={mutacionCrear.isPending}>
                {mutacionCrear.isPending
                  ? "Guardando…"
                  : "Publicar y programar"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Modal Cambiar Estado / Cierre */}
      <Dialog
        open={!!convocatoriaParaEditar}
        onOpenChange={(abierto) => {
          if (!abierto) setConvocatoriaParaEditar(null)
        }}
      >
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Cambiar Estado de la Convocatoria</DialogTitle>
            <DialogDescription>
              Convocatoria{" "}
              <strong className="text-foreground">
                {convocatoriaParaEditar?.codigo}
              </strong>
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-2">
            <div className="space-y-1.5">
              <Label htmlFor="estado">Nuevo estado</Label>
              <Select
                value={nuevoEstado}
                onValueChange={(v) => setNuevoEstado(v as EstadoConvocatoria)}
              >
                <SelectTrigger id="estado">
                  <SelectValue placeholder="Seleccione estado" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="en_curso">En curso (Abierta)</SelectItem>
                  <SelectItem value="publicada">
                    Publicada (Programada)
                  </SelectItem>
                  <SelectItem value="cerrada">Cerrada oficialmente</SelectItem>
                  <SelectItem value="archivada">Archivada</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {nuevoEstado === "cerrada" && (
              <Alert variant="destructive" className="text-xs">
                <AlertTriangleIcon className="size-4" />
                <AlertTitle className="text-xs font-semibold">
                  Advertencia legal de cierre
                </AlertTitle>
                <AlertDescription className="text-2xs leading-relaxed">
                  El cierre oficial impedirá permanentemente el envío de nuevos
                  formularios EF-12 y bloqueará cualquier modificación a
                  borradores pendientes. Esta acción queda auditada.
                </AlertDescription>
              </Alert>
            )}

            <div className="space-y-1.5">
              <Label htmlFor="motivo">
                Motivo institucional o resolución de cambio *
              </Label>
              <Textarea
                id="motivo"
                placeholder="Indique la resolución del Consejo Directivo o acta de la Comisión…"
                rows={3}
                value={motivoCambio}
                onChange={(e) => setMotivoCambio(e.target.value)}
                required
              />
              <p className="text-2xs text-muted-foreground">
                Este motivo se registra en la bitácora de auditoría inmutable
                (T8/T15).
              </p>
            </div>
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => setConvocatoriaParaEditar(null)}
            >
              Cancelar
            </Button>
            <Button
              type="button"
              variant={nuevoEstado === "cerrada" ? "destructive" : "default"}
              onClick={handleConfirmarCambioEstado}
              disabled={mutacionActualizar.isPending}
            >
              {mutacionActualizar.isPending ? "Guardando…" : "Confirmar cambio"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
