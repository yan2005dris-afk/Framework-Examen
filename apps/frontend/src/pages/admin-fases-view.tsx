import { CheckCircle2Icon, ClockIcon, InfoIcon, LockIcon } from "lucide-react"
import { useMemo, useState } from "react"
import { Link, useSearchParams } from "react-router"
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
import { Switch } from "@/components/ui/switch"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Textarea } from "@/components/ui/textarea"
import {
  useActualizarFase,
  useAlternarHabilitacionFormulario,
  useConvocatoriasAdmin,
} from "@/features/convocatorias/queries"
import { RUTAS } from "@/lib/rutas"
import type { EstadoFase, FaseConcurso } from "@/types/convocatoria"
import { formatearFechaEcuador } from "@/utils/fechas"

/**
 * Panel de administración de fases y cronograma oficial del concurso (FT15).
 * Permite gestionar las 6 fases de la convocatoria, habilitar/bloquear
 * el formulario EF-12 y programar prórrogas o aperturas con trazabilidad.
 */
export function AdminFasesView() {
  const [searchParams, setSearchParams] = useSearchParams()
  const {
    data: paginaConvocatorias,
    isLoading,
    error,
    refetch,
  } = useConvocatoriasAdmin()
  const mutacionHabilitacion = useAlternarHabilitacionFormulario()
  const mutacionFase = useActualizarFase()

  // Convocatoria seleccionada desde query string o la primera disponible
  const convocatorias = useMemo(
    () => paginaConvocatorias?.items ?? [],
    [paginaConvocatorias],
  )
  const convocatoriaIdQuery = searchParams.get("convocatoria")

  const convocatoriaActual = useMemo(() => {
    if (!convocatorias.length) return null
    if (convocatoriaIdQuery) {
      const encontrada = convocatorias.find((c) => c.id === convocatoriaIdQuery)
      if (encontrada) return encontrada
    }
    return convocatorias[0]
  }, [convocatorias, convocatoriaIdQuery])

  // Diálogo para alternar habilitación EF-12
  const [modalHabilitarAbierto, setModalHabilitarAbierto] = useState(false)
  const [nuevoEstadoHabilitacion, setNuevoEstadoHabilitacion] = useState(false)
  const [motivoHabilitacion, setMotivoHabilitacion] = useState("")

  // Diálogo para editar cronograma de fase
  const [faseParaEditar, setFaseParaEditar] = useState<FaseConcurso | null>(
    null,
  )
  const [nuevaFechaInicio, setNuevaFechaInicio] = useState("")
  const [nuevaFechaFin, setNuevaFechaFin] = useState("")
  const [nuevoEstadoFase, setNuevoEstadoFase] = useState<EstadoFase>("en_curso")
  const [motivoFase, setMotivoFase] = useState("")

  function handleSeleccionarConvocatoria(id: string) {
    setSearchParams({ convocatoria: id })
  }

  function abrirModalHabilitacion(nuevoValor: boolean) {
    setNuevoEstadoHabilitacion(nuevoValor)
    setMotivoHabilitacion("")
    setModalHabilitarAbierto(true)
  }

  async function handleConfirmarHabilitacion() {
    if (!convocatoriaActual) return
    if (!motivoHabilitacion.trim()) {
      toast.error(
        "Debe ingresar un motivo institucional para el registro de auditoría.",
      )
      return
    }

    try {
      await mutacionHabilitacion.mutateAsync({
        convocatoriaId: convocatoriaActual.id,
        habilitada: nuevoEstadoHabilitacion,
        motivo: motivoHabilitacion.trim(),
      })
      toast.success(
        nuevoEstadoHabilitacion
          ? "Formulario EF-12 habilitado para recepción de postulaciones."
          : "Formulario EF-12 bloqueado. No se admiten envíos.",
      )
      setModalHabilitarAbierto(false)
    } catch (err) {
      toast.error(
        err instanceof Error
          ? err.message
          : "El servidor rechazó el cambio de habilitación.",
      )
    }
  }

  function abrirModalEditarFase(fase: FaseConcurso) {
    setFaseParaEditar(fase)
    // Convertir ISO a formato datetime-local aproximado (YYYY-MM-DDTHH:MM)
    const inicioStr = fase.fechaInicio.slice(0, 16)
    const finStr = fase.fechaFin.slice(0, 16)
    setNuevaFechaInicio(inicioStr)
    setNuevaFechaFin(finStr)
    setNuevoEstadoFase(fase.estado)
    setMotivoFase("")
  }

  async function handleConfirmarEdicionFase() {
    if (!convocatoriaActual || !faseParaEditar) return
    if (!motivoFase.trim()) {
      toast.error(
        "Debe ingresar la resolución o motivo de la modificación del cronograma.",
      )
      return
    }

    const inicioIso = new Date(nuevaFechaInicio).toISOString()
    const finIso = new Date(nuevaFechaFin).toISOString()

    if (new Date(finIso) <= new Date(inicioIso)) {
      toast.error(
        "La fecha de fin de la fase debe ser posterior a la fecha de inicio.",
      )
      return
    }

    try {
      await mutacionFase.mutateAsync({
        convocatoriaId: convocatoriaActual.id,
        faseId: faseParaEditar.id,
        payload: {
          fechaInicio: inicioIso,
          fechaFin: finIso,
          estado: nuevoEstadoFase,
          motivo: motivoFase.trim(),
        },
      })
      toast.success(
        `Cronograma de ${faseParaEditar.nombre} actualizado con éxito.`,
      )
      setFaseParaEditar(null)
    } catch (err) {
      toast.error(
        err instanceof Error
          ? err.message
          : "El servidor rechazó la modificación de la fase.",
      )
    }
  }

  if (isLoading) {
    return (
      <div className="flex flex-1 items-center justify-center p-8">
        <EstadoCargando mensaje="Cargando cronograma y fases oficiales…" />
      </div>
    )
  }

  if (error || !convocatoriaActual) {
    return (
      <div className="flex flex-1 flex-col p-6">
        <EstadoError
          error={error}
          onReintentar={() => void refetch()}
          titulo="Error al cargar fases"
          mensaje="No se pudo consultar el cronograma de fases de la convocatoria."
        />
      </div>
    )
  }

  const fases = convocatoriaActual.fases || []
  const formularioHabilitado =
    convocatoriaActual.postulacionHabilitada && !convocatoriaActual.cerrada

  return (
    <div className="flex flex-1 flex-col gap-6 p-6">
      {/* Cabecera y selector de convocatoria */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <div className="flex items-center gap-2">
            <h1 className="text-2xl font-bold tracking-tight text-foreground">
              Fases y Cronograma Oficial
            </h1>
            <Badge variant="outline" className="border-primary/50 text-primary">
              Comisión de Selección
            </Badge>
          </div>
          <p className="text-sm text-muted-foreground">
            Control de etapas oficiales, aprobación de cronogramas y
            habilitación del formulario EF-12.
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-3">
          <div className="flex items-center gap-2">
            <Label
              htmlFor="selector-convocatoria"
              className="text-xs font-semibold"
            >
              Convocatoria:
            </Label>
            <Select
              value={convocatoriaActual.id}
              onValueChange={handleSeleccionarConvocatoria}
            >
              <SelectTrigger id="selector-convocatoria" className="w-[240px]">
                <SelectValue placeholder="Seleccione convocatoria" />
              </SelectTrigger>
              <SelectContent>
                {convocatorias.map((c) => (
                  <SelectItem key={c.id} value={c.id}>
                    {c.codigo} ({c.periodo})
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <Button asChild variant="outline" size="sm">
            <Link to={RUTAS.adminConvocatorias}>Volver a convocatorias</Link>
          </Button>
        </div>
      </div>

      {/* Banner de información de la convocatoria seleccionada */}
      <div className="flex flex-wrap items-center justify-between gap-4 rounded-xl border bg-card p-4 shadow-sm">
        <div className="space-y-1">
          <div className="flex items-center gap-2">
            <span className="font-mono text-sm font-bold text-primary">
              {convocatoriaActual.codigo}
            </span>
            <span className="text-muted-foreground">·</span>
            <span className="text-sm font-semibold text-foreground">
              {convocatoriaActual.titulo}
            </span>
          </div>
          <p className="text-xs text-muted-foreground">
            {convocatoriaActual.resolucion} · Período global:{" "}
            {formatearFechaEcuador(convocatoriaActual.apertura)} al{" "}
            {formatearFechaEcuador(convocatoriaActual.cierre)} (ECT)
          </p>
        </div>

        <div>
          {convocatoriaActual.cerrada ? (
            <Badge variant="destructive" className="gap-1">
              <LockIcon className="size-3" />
              <span>Convocatoria Cerrada</span>
            </Badge>
          ) : (
            <Badge className="bg-emerald-600 hover:bg-emerald-700 gap-1">
              <CheckCircle2Icon className="size-3" />
              <span>Convocatoria Activa</span>
            </Badge>
          )}
        </div>
      </div>

      {/* Tarjeta de Control Operativo del Formulario EF-12 */}
      <Card className="border-l-4 border-l-primary shadow-sm">
        <CardHeader className="pb-3">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div className="space-y-1">
              <div className="flex items-center gap-2">
                <CardTitle className="text-lg">
                  Admisión y Recepción del Formulario EF-12
                </CardTitle>
                {formularioHabilitado ? (
                  <Badge className="bg-emerald-600 hover:bg-emerald-700">
                    Habilitado
                  </Badge>
                ) : (
                  <Badge variant="destructive">Bloqueado</Badge>
                )}
              </div>
              <CardDescription>
                Habilita o suspende en tiempo real la capacidad de los
                postulantes para crear, editar o remitir expedientes en el
                portal público.
              </CardDescription>
            </div>

            <div className="flex items-center gap-3">
              <span className="text-xs font-semibold">
                {formularioHabilitado ? "Habilitado" : "Bloqueado"}
              </span>
              <Switch
                checked={formularioHabilitado}
                disabled={
                  convocatoriaActual.cerrada || mutacionHabilitacion.isPending
                }
                onCheckedChange={(checked) => abrirModalHabilitacion(checked)}
                aria-label="Alternar habilitación de recepción del formulario EF-12"
              />
            </div>
          </div>
        </CardHeader>
        <CardContent className="text-xs text-muted-foreground">
          <p>
            <strong>Regla de negocio:</strong> La habilitación operativa del
            formulario es independiente de las fechas del cronograma de méritos
            u oposición. El cierre formal de la convocatoria bloquea el
            formulario de manera permanente.
          </p>
        </CardContent>
      </Card>

      {/* Tabla de Cronograma Oficial de Fases */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">
            Cronograma Oficial de Fases (1 a 6)
          </CardTitle>
          <CardDescription>
            Secuencia reglamentaria del Concurso Público de Méritos y Oposición.
          </CardDescription>
        </CardHeader>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-[80px]">Orden</TableHead>
                <TableHead className="w-[280px]">
                  Nombre y Detalle de la Fase
                </TableHead>
                <TableHead>Fecha Inicio (ECT)</TableHead>
                <TableHead>Fecha Fin (ECT)</TableHead>
                <TableHead className="w-[120px]">Estado</TableHead>
                <TableHead className="w-[130px]">Formulario</TableHead>
                <TableHead className="text-right">Acciones</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {fases.map((fase) => (
                <TableRow key={fase.id}>
                  <TableCell className="font-mono text-sm font-semibold text-muted-foreground">
                    Fase {fase.orden}
                  </TableCell>
                  <TableCell>
                    <div className="space-y-1">
                      <p className="font-semibold text-foreground">
                        {fase.nombre}
                      </p>
                      <p className="text-xs text-muted-foreground line-clamp-2">
                        {fase.descripcion}
                      </p>
                    </div>
                  </TableCell>
                  <TableCell className="text-xs font-medium">
                    {formatearFechaEcuador(fase.fechaInicio, {
                      incluirHora: true,
                    })}
                  </TableCell>
                  <TableCell className="text-xs font-medium">
                    {formatearFechaEcuador(fase.fechaFin, {
                      incluirHora: true,
                    })}
                  </TableCell>
                  <TableCell>
                    {fase.estado === "en_curso" ? (
                      <Badge className="bg-primary text-primary-foreground gap-1">
                        <ClockIcon className="size-3" />
                        <span>En curso</span>
                      </Badge>
                    ) : fase.estado === "finalizada" ? (
                      <Badge variant="secondary" className="gap-1">
                        <CheckCircle2Icon className="size-3" />
                        <span>Finalizada</span>
                      </Badge>
                    ) : fase.estado === "cancelada" ? (
                      <Badge variant="destructive">Cancelada</Badge>
                    ) : (
                      <Badge
                        variant="outline"
                        className="text-muted-foreground"
                      >
                        Pendiente
                      </Badge>
                    )}
                  </TableCell>
                  <TableCell>
                    {fase.permitePostulacion ? (
                      <span className="inline-flex items-center gap-1 text-xs font-semibold text-emerald-600 dark:text-emerald-400">
                        <span className="size-1.5 rounded-full bg-emerald-600" />
                        Postulación
                      </span>
                    ) : (
                      <span className="text-xs text-muted-foreground">
                        No aplica
                      </span>
                    )}
                  </TableCell>
                  <TableCell className="text-right">
                    <Button
                      variant="outline"
                      size="sm"
                      className="text-xs"
                      onClick={() => abrirModalEditarFase(fase)}
                    >
                      Modificar
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Nota de separación de responsabilidades */}
      <Alert className="border-blue-500/30 bg-blue-500/5 text-foreground">
        <InfoIcon className="size-4 text-blue-600 dark:text-blue-400" />
        <AlertTitle className="text-sm font-semibold text-blue-900 dark:text-blue-300">
          Principio de separación institucional de etapas
        </AlertTitle>
        <AlertDescription className="mt-1 text-xs text-muted-foreground leading-relaxed">
          La administración de fases y habilitación del formulario EF-12 se
          mantiene estrictamente independiente de las funciones de calificación
          técnica y de la publicación de resultados de méritos u oposición
          (T18/T20). Esta vista no realiza asignaciones de puntajes ni altera
          expedientes previamente sellados.
        </AlertDescription>
      </Alert>

      {/* Modal Confirmar Habilitación de EF-12 */}
      <Dialog
        open={modalHabilitarAbierto}
        onOpenChange={setModalHabilitarAbierto}
      >
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>
              {nuevoEstadoHabilitacion
                ? "Habilitar Recepción del Formulario EF-12"
                : "Bloquear Recepción del Formulario EF-12"}
            </DialogTitle>
            <DialogDescription>
              Convocatoria {convocatoriaActual.codigo}
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-2">
            <div className="rounded-lg border bg-muted/30 p-3 text-xs leading-relaxed text-muted-foreground">
              {nuevoEstadoHabilitacion ? (
                <>
                  Al habilitar el formulario, cualquier ciudadano registrado en
                  el portal podrá completar las 9 secciones y remitir su
                  postulación digital con firma y sellado criptográfico.
                </>
              ) : (
                <>
                  Al bloquear el formulario, se suspenderá inmediatamente la
                  admisión de nuevas postulaciones. Los usuarios con formularios
                  en borrador no podrán remitir su expediente.
                </>
              )}
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="motivo-hab">
                Resolución o justificación de auditoría *
              </Label>
              <Textarea
                id="motivo-hab"
                placeholder="Indique la resolución del Consejo Directivo o el acta oficial que autoriza este cambio…"
                rows={3}
                value={motivoHabilitacion}
                onChange={(e) => setMotivoHabilitacion(e.target.value)}
                required
              />
              <p className="text-2xs text-muted-foreground">
                Este registro queda asentado en la bitácora inmutable de
                auditoría.
              </p>
            </div>
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => setModalHabilitarAbierto(false)}
            >
              Cancelar
            </Button>
            <Button
              type="button"
              variant={nuevoEstadoHabilitacion ? "default" : "destructive"}
              onClick={handleConfirmarHabilitacion}
              disabled={mutacionHabilitacion.isPending}
            >
              {mutacionHabilitacion.isPending
                ? "Guardando…"
                : nuevoEstadoHabilitacion
                  ? "Habilitar formulario"
                  : "Bloquear formulario"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Modal Modificar Cronograma de Fase */}
      <Dialog
        open={!!faseParaEditar}
        onOpenChange={(abierto) => {
          if (!abierto) setFaseParaEditar(null)
        }}
      >
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>Modificar Cronograma de Fase</DialogTitle>
            <DialogDescription>
              {faseParaEditar?.nombre} · Convocatoria{" "}
              {convocatoriaActual.codigo}
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-2">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-1.5">
                <Label htmlFor="fecha-inicio">Fecha de inicio (ECT) *</Label>
                <Input
                  id="fecha-inicio"
                  type="datetime-local"
                  value={nuevaFechaInicio}
                  onChange={(e) => setNuevaFechaInicio(e.target.value)}
                  required
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="fecha-fin">Fecha de fin (ECT) *</Label>
                <Input
                  id="fecha-fin"
                  type="datetime-local"
                  value={nuevaFechaFin}
                  onChange={(e) => setNuevaFechaFin(e.target.value)}
                  required
                />
              </div>
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="estado-fase">Estado de la fase</Label>
              <Select
                value={nuevoEstadoFase}
                onValueChange={(v) => setNuevoEstadoFase(v as EstadoFase)}
              >
                <SelectTrigger id="estado-fase">
                  <SelectValue placeholder="Seleccione estado" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="pendiente">Pendiente</SelectItem>
                  <SelectItem value="en_curso">En curso</SelectItem>
                  <SelectItem value="finalizada">Finalizada</SelectItem>
                  <SelectItem value="cancelada">Cancelada</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="motivo-fase">
                Acta o resolución de prórroga / modificación *
              </Label>
              <Textarea
                id="motivo-fase"
                placeholder="Indique el número de resolución que aprueba el cambio de fechas…"
                rows={3}
                value={motivoFase}
                onChange={(e) => setMotivoFase(e.target.value)}
                required
              />
              <p className="text-2xs text-muted-foreground">
                Toda modificación de plazos queda registrada con sello de tiempo
                y usuario administrador responsable.
              </p>
            </div>
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => setFaseParaEditar(null)}
            >
              Cancelar
            </Button>
            <Button
              type="button"
              onClick={handleConfirmarEdicionFase}
              disabled={mutacionFase.isPending}
            >
              {mutacionFase.isPending ? "Guardando…" : "Actualizar cronograma"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
