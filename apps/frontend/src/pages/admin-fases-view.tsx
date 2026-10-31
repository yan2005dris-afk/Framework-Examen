import { InfoIcon, PlusIcon } from "lucide-react"
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
import {
  useConvocatoriaAdminDetalle,
  useConvocatoriasAdmin,
  useGuardarConfiguracionConvocatoria,
} from "@/features/convocatorias/queries"
import { RUTAS } from "@/lib/rutas"
import {
  NOMBRES_FASES_OFICIALES,
  type CodigoFase,
  type FasePublica,
} from "@/types/convocatoria"
import { formatearFechaEcuador } from "@/utils/fechas"

const CODIGOS_FASES_DISPONIBLES: CodigoFase[] = [
  "postulacion",
  "meritos",
  "oposicion_teorica",
  "oposicion_psicotecnica",
  "reconsideracion",
  "resultados",
]

/**
 * Panel de administración de fases y configuración T18 (FT15).
 * Controla versión, inicio aprobado, publicación oficial, sectores, fases y habilitación EF-12.
 * Aplica control optimista con versionado estricto sin datos simulados.
 */
export function AdminFasesView() {
  const [searchParams, setSearchParams] = useSearchParams()
  const { data: paginaConvocatorias, isLoading: cargandoLista } =
    useConvocatoriasAdmin()
  const mutacionConfig = useGuardarConfiguracionConvocatoria()

  const convocatorias = useMemo(
    () => paginaConvocatorias?.items ?? [],
    [paginaConvocatorias],
  )
  const convocatoriaIdQuery = searchParams.get("convocatoria")

  const convocatoriaIdActual = useMemo(() => {
    if (convocatoriaIdQuery) return convocatoriaIdQuery
    if (convocatorias.length > 0) return convocatorias[0].id
    return null
  }, [convocatorias, convocatoriaIdQuery])

  const {
    data: detalle,
    isLoading: cargandoDetalle,
    error: errorDetalle,
    refetch: recargarDetalle,
  } = useConvocatoriaAdminDetalle(convocatoriaIdActual)

  // Diálogo para agregar o modificar fase
  const [faseEditando, setFaseEditando] = useState<FasePublica | null>(null)
  const [modalFaseAbierto, setModalFaseAbierto] = useState(false)
  const [codigoFase, setCodigoFase] = useState<CodigoFase>("postulacion")
  const [tituloFase, setTituloFase] = useState("")
  const [aperturaFase, setAperturaFase] = useState("")
  const [cierreFase, setCierreFase] = useState("")
  const [publicadaFase, setPublicadaFase] = useState(true)
  const [habilitadaFase, setHabilitadaFase] = useState(false)

  function handleSeleccionarConvocatoria(id: string) {
    setSearchParams({ convocatoria: id })
  }

  async function handleToggleAprobacion(checked: boolean) {
    if (!detalle) return
    try {
      await mutacionConfig.mutateAsync({
        id: detalle.id,
        config: {
          version: detalle.version,
          inicioAprobado: checked,
          publicada: checked ? detalle.publicada : false,
          ef12Habilitado: checked ? detalle.ef12Habilitado : false,
          sectores: detalle.sectores,
          fases: detalle.fases,
        },
      })
      toast.success(
        checked
          ? "Inicio de convocatoria aprobado formalmente."
          : "Aprobación de convocatoria revocada.",
      )
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Error al actualizar aprobación.",
      )
    }
  }

  async function handleTogglePublicacion(checked: boolean) {
    if (!detalle) return
    if (checked && !detalle.inicioAprobado) {
      toast.error("Debe aprobar el inicio antes de publicar la convocatoria.")
      return
    }
    try {
      await mutacionConfig.mutateAsync({
        id: detalle.id,
        config: {
          version: detalle.version,
          inicioAprobado: detalle.inicioAprobado,
          publicada: checked,
          ef12Habilitado: checked ? detalle.ef12Habilitado : false,
          sectores: detalle.sectores,
          fases: detalle.fases,
        },
      })
      toast.success(
        checked
          ? "Convocatoria publicada oficialmente en el portal público."
          : "Convocatoria retirada del portal público.",
      )
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Error al actualizar publicación.",
      )
    }
  }

  async function handleToggleHabilitarEF12(checked: boolean) {
    if (!detalle) return
    if (checked && (!detalle.inicioAprobado || !detalle.publicada)) {
      toast.error(
        "Para habilitar el EF-12 la convocatoria debe estar aprobada y publicada.",
      )
      return
    }

    const tieneFasePostulacionHabilitada = detalle.fases.some(
      (f) => f.codigo === "postulacion" && f.publicada && f.habilitada,
    )

    if (checked && !tieneFasePostulacionHabilitada) {
      toast.error(
        "Se requiere una fase de 'postulacion' publicada y habilitada para activar el formulario EF-12.",
      )
      return
    }

    try {
      await mutacionConfig.mutateAsync({
        id: detalle.id,
        config: {
          version: detalle.version,
          inicioAprobado: detalle.inicioAprobado,
          publicada: detalle.publicada,
          ef12Habilitado: checked,
          sectores: detalle.sectores,
          fases: detalle.fases,
        },
      })
      toast.success(
        checked
          ? "Recepción de formularios EF-12 habilitada."
          : "Recepción de formularios EF-12 suspendida.",
      )
    } catch (err) {
      toast.error(
        err instanceof Error
          ? err.message
          : "Error al cambiar habilitación EF-12.",
      )
    }
  }

  function abrirModalNuevaFase() {
    setFaseEditando(null)
    setCodigoFase("postulacion")
    setTituloFase(NOMBRES_FASES_OFICIALES.postulacion)
    setAperturaFase(detalle?.apertura.slice(0, 16) ?? "")
    setCierreFase(detalle?.cierre.slice(0, 16) ?? "")
    setPublicadaFase(true)
    setHabilitadaFase(false)
    setModalFaseAbierto(true)
  }

  function abrirModalEditarFase(fase: FasePublica) {
    setFaseEditando(fase)
    setCodigoFase(fase.codigo)
    setTituloFase(fase.titulo)
    setAperturaFase(fase.apertura.slice(0, 16))
    setCierreFase(fase.cierre.slice(0, 16))
    setPublicadaFase(fase.publicada)
    setHabilitadaFase(fase.habilitada)
    setModalFaseAbierto(true)
  }

  async function handleGuardarFase(e: React.FormEvent) {
    e.preventDefault()
    if (!detalle) return

    const aperturaIso = new Date(aperturaFase).toISOString()
    const cierreIso = new Date(cierreFase).toISOString()

    if (new Date(cierreIso) <= new Date(aperturaIso)) {
      toast.error("La fecha de cierre debe ser posterior a la de apertura.")
      return
    }

    const nuevaFase: FasePublica = {
      codigo: codigoFase,
      titulo: tituloFase.trim() || NOMBRES_FASES_OFICIALES[codigoFase],
      apertura: aperturaIso,
      cierre: cierreIso,
      publicada: publicadaFase,
      habilitada: habilitadaFase,
    }

    const fasesActualizadas = faseEditando
      ? detalle.fases.map((f) =>
          f.codigo === faseEditando.codigo ? nuevaFase : f,
        )
      : [...detalle.fases.filter((f) => f.codigo !== codigoFase), nuevaFase]

    try {
      await mutacionConfig.mutateAsync({
        id: detalle.id,
        config: {
          version: detalle.version,
          inicioAprobado: detalle.inicioAprobado,
          publicada: detalle.publicada,
          ef12Habilitado: detalle.ef12Habilitado,
          sectores: detalle.sectores,
          fases: fasesActualizadas,
        },
      })
      toast.success("Fase guardada exitosamente en el servidor.")
      setModalFaseAbierto(false)
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Error al guardar la fase.",
      )
    }
  }

  if (cargandoLista || cargandoDetalle) {
    return (
      <div className="flex flex-1 items-center justify-center p-8">
        <EstadoCargando mensaje="Consultando configuración de fases del servidor…" />
      </div>
    )
  }

  if (convocatorias.length === 0) {
    return (
      <div className="flex flex-1 flex-col gap-4 p-6">
        <Alert className="border-amber-500/50 bg-amber-500/10">
          <InfoIcon className="size-5 text-amber-600" />
          <AlertTitle className="font-semibold text-amber-900 dark:text-amber-200">
            No existen convocatorias registradas
          </AlertTitle>
          <AlertDescription className="text-sm">
            Para configurar fases, primero cree una convocatoria desde el panel
            administrativo.
          </AlertDescription>
        </Alert>
        <div>
          <Button asChild>
            <Link to={RUTAS.adminConvocatorias}>Ir a Convocatorias</Link>
          </Button>
        </div>
      </div>
    )
  }

  if (errorDetalle || !detalle) {
    return (
      <div className="flex flex-1 flex-col p-6">
        <EstadoError
          error={errorDetalle}
          onReintentar={() => void recargarDetalle()}
          titulo="Error al consultar configuración"
          mensaje="No se pudo obtener la configuración oficial desde el servidor."
        />
      </div>
    )
  }

  return (
    <div className="flex flex-1 flex-col gap-6 p-6">
      {/* Cabecera y selector de convocatoria */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <div className="flex items-center gap-2">
            <h1 className="text-2xl font-bold tracking-tight text-foreground">
              Fases y Configuración (T18)
            </h1>
            <Badge variant="outline" className="border-primary/50 text-primary">
              Comisión de Selección
            </Badge>
          </div>
          <p className="text-sm text-muted-foreground">
            Control de versión optimista (v{detalle.version}), aprobación,
            publicación y habilitación de fases oficiales.
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
              value={detalle.id}
              onValueChange={handleSeleccionarConvocatoria}
            >
              <SelectTrigger id="selector-convocatoria" className="w-[240px]">
                <SelectValue placeholder="Seleccione convocatoria" />
              </SelectTrigger>
              <SelectContent>
                {convocatorias.map((c) => (
                  <SelectItem key={c.id} value={c.id}>
                    {c.codigo}
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

      {/* Controles de Estado Institucional (Aprobación, Publicación, EF-12) */}
      <div className="grid gap-4 sm:grid-cols-3">
        <Card className="border-l-4 border-l-blue-600">
          <CardHeader className="pb-2">
            <div className="flex items-center justify-between">
              <CardTitle className="text-sm font-semibold">
                Inicio Aprobado
              </CardTitle>
              <Switch
                checked={detalle.inicioAprobado}
                disabled={mutacionConfig.isPending}
                onCheckedChange={handleToggleAprobacion}
              />
            </div>
            <CardDescription className="text-xs">
              Resolución formal de la Comisión para iniciar el concurso
            </CardDescription>
          </CardHeader>
          <CardContent className="text-xs font-medium">
            {detalle.inicioAprobado ? (
              <span className="text-emerald-600 dark:text-emerald-400">
                Aprobado
              </span>
            ) : (
              <span className="text-muted-foreground">Pendiente</span>
            )}
          </CardContent>
        </Card>

        <Card className="border-l-4 border-l-emerald-600">
          <CardHeader className="pb-2">
            <div className="flex items-center justify-between">
              <CardTitle className="text-sm font-semibold">
                Publicada en Portal
              </CardTitle>
              <Switch
                checked={detalle.publicada}
                disabled={!detalle.inicioAprobado || mutacionConfig.isPending}
                onCheckedChange={handleTogglePublicacion}
              />
            </div>
            <CardDescription className="text-xs">
              Visibilidad de la convocatoria en el portal público
            </CardDescription>
          </CardHeader>
          <CardContent className="text-xs font-medium">
            {detalle.publicada ? (
              <span className="text-emerald-600 dark:text-emerald-400">
                Visible al público
              </span>
            ) : (
              <span className="text-muted-foreground">Oculta</span>
            )}
          </CardContent>
        </Card>

        <Card className="border-l-4 border-l-amber-600">
          <CardHeader className="pb-2">
            <div className="flex items-center justify-between">
              <CardTitle className="text-sm font-semibold">
                Formulario EF-12
              </CardTitle>
              <Switch
                checked={detalle.ef12Habilitado}
                disabled={!detalle.publicada || mutacionConfig.isPending}
                onCheckedChange={handleToggleHabilitarEF12}
              />
            </div>
            <CardDescription className="text-xs">
              Admisión activa de recepción y sellado de expedientes
            </CardDescription>
          </CardHeader>
          <CardContent className="text-xs font-medium">
            {detalle.ef12Habilitado ? (
              <span className="text-emerald-600 dark:text-emerald-400">
                Habilitado para postulantes
              </span>
            ) : (
              <span className="text-muted-foreground">Bloqueado</span>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Tabla de Fases Oficiales */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between pb-3">
          <div>
            <CardTitle className="text-lg">Fases Oficiales (T18)</CardTitle>
            <CardDescription>
              Capacidades reglamentarias publicadas y habilitadas en el
              servidor.
            </CardDescription>
          </div>
          <Button size="sm" onClick={abrirModalNuevaFase} className="gap-1.5">
            <PlusIcon className="size-4" />
            <span>Agregar fase</span>
          </Button>
        </CardHeader>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-[180px]">Código Técnico</TableHead>
                <TableHead>Título de la Fase</TableHead>
                <TableHead>Apertura (ECT)</TableHead>
                <TableHead>Cierre (ECT)</TableHead>
                <TableHead className="w-[120px]">Publicada</TableHead>
                <TableHead className="w-[120px]">Habilitada</TableHead>
                <TableHead className="text-right">Acción</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {detalle.fases.length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={7}
                    className="h-28 text-center text-muted-foreground text-xs"
                  >
                    No hay fases configuradas. Utilice "Agregar fase" para
                    definir las etapas del concurso.
                  </TableCell>
                </TableRow>
              ) : (
                detalle.fases.map((f) => (
                  <TableRow key={f.codigo}>
                    <TableCell className="font-mono text-xs font-semibold text-primary">
                      {f.codigo}
                    </TableCell>
                    <TableCell className="font-medium text-foreground">
                      {f.titulo}
                    </TableCell>
                    <TableCell className="text-xs">
                      {formatearFechaEcuador(f.apertura, { incluirHora: true })}
                    </TableCell>
                    <TableCell className="text-xs">
                      {formatearFechaEcuador(f.cierre, { incluirHora: true })}
                    </TableCell>
                    <TableCell>
                      {f.publicada ? (
                        <Badge variant="outline" className="text-emerald-600">
                          Sí
                        </Badge>
                      ) : (
                        <Badge variant="secondary">No</Badge>
                      )}
                    </TableCell>
                    <TableCell>
                      {f.habilitada ? (
                        <Badge className="bg-emerald-600">En curso</Badge>
                      ) : (
                        <Badge variant="outline">Inactiva</Badge>
                      )}
                    </TableCell>
                    <TableCell className="text-right">
                      <Button
                        variant="outline"
                        size="sm"
                        className="text-xs"
                        onClick={() => abrirModalEditarFase(f)}
                      >
                        Editar
                      </Button>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Nota institucional */}
      <Alert className="border-blue-500/30 bg-blue-500/5">
        <InfoIcon className="size-4 text-blue-600 dark:text-blue-400" />
        <AlertTitle className="text-sm font-semibold text-blue-900 dark:text-blue-300">
          Control de concurrencia optimista
        </AlertTitle>
        <AlertDescription className="mt-1 text-xs text-muted-foreground">
          Cada modificación actualiza la versión del registro en PostgreSQL (v
          {detalle.version}). Si otro miembro de la Comisión modifica la
          convocatoria concurrentemente, el servidor devolverá 409 y se
          recargarán los datos actualizados para evitar sobrescrituras.
        </AlertDescription>
      </Alert>

      {/* Modal Editar / Agregar Fase */}
      <Dialog open={modalFaseAbierto} onOpenChange={setModalFaseAbierto}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>
              {faseEditando ? "Modificar Fase" : "Agregar Fase Oficial"}
            </DialogTitle>
            <DialogDescription>
              Convocatoria {detalle.codigo} (v{detalle.version})
            </DialogDescription>
          </DialogHeader>

          <form onSubmit={handleGuardarFase} className="space-y-4">
            <div className="space-y-1.5">
              <Label htmlFor="codigo-fase">Código de capacidad *</Label>
              <Select
                value={codigoFase}
                onValueChange={(v) => {
                  const cod = v as CodigoFase
                  setCodigoFase(cod)
                  if (!faseEditando) {
                    setTituloFase(NOMBRES_FASES_OFICIALES[cod])
                  }
                }}
                disabled={Boolean(faseEditando)}
              >
                <SelectTrigger id="codigo-fase">
                  <SelectValue placeholder="Seleccione código" />
                </SelectTrigger>
                <SelectContent>
                  {CODIGOS_FASES_DISPONIBLES.map((cod) => (
                    <SelectItem key={cod} value={cod}>
                      {cod} ({NOMBRES_FASES_OFICIALES[cod]})
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="titulo-fase">Título oficial de la fase *</Label>
              <Input
                id="titulo-fase"
                value={tituloFase}
                onChange={(e) => setTituloFase(e.target.value)}
                required
              />
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-1.5">
                <Label htmlFor="apertura-fase">Apertura (ECT) *</Label>
                <Input
                  id="apertura-fase"
                  type="datetime-local"
                  value={aperturaFase}
                  onChange={(e) => setAperturaFase(e.target.value)}
                  required
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="cierre-fase">Cierre (ECT) *</Label>
                <Input
                  id="cierre-fase"
                  type="datetime-local"
                  value={cierreFase}
                  onChange={(e) => setCierreFase(e.target.value)}
                  required
                />
              </div>
            </div>

            <div className="flex items-center justify-between rounded-lg border p-3">
              <div className="space-y-0.5">
                <Label htmlFor="publicada-fase">Publicada</Label>
                <p className="text-2xs text-muted-foreground">
                  Visible en el cronograma público
                </p>
              </div>
              <Switch
                id="publicada-fase"
                checked={publicadaFase}
                onCheckedChange={setPublicadaFase}
              />
            </div>

            <div className="flex items-center justify-between rounded-lg border p-3">
              <div className="space-y-0.5">
                <Label htmlFor="habilitada-fase">Habilitada</Label>
                <p className="text-2xs text-muted-foreground">
                  Fase activa operacionalmente
                </p>
              </div>
              <Switch
                id="habilitada-fase"
                checked={habilitadaFase}
                disabled={!publicadaFase}
                onCheckedChange={setHabilitadaFase}
              />
            </div>

            <DialogFooter className="pt-2">
              <Button
                type="button"
                variant="outline"
                onClick={() => setModalFaseAbierto(false)}
              >
                Cancelar
              </Button>
              <Button type="submit" disabled={mutacionConfig.isPending}>
                {mutacionConfig.isPending ? "Guardando…" : "Guardar fase"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  )
}
