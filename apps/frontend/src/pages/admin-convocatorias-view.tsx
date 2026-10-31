import {
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
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  useConvocatoriasAdmin,
  useCrearConvocatoria,
} from "@/features/convocatorias/queries"
import { RUTAS } from "@/lib/rutas"
import type { ResumenConvocatoriaAdmin } from "@/types/convocatoria"
import { formatearFechaEcuador, obtenerEstadoTemporal } from "@/utils/fechas"

type TabFiltro = "todas" | "activas" | "cerradas"

/**
 * Panel de administración y programación de convocatorias (FT15).
 * Permite a la Comisión Especial de Selección registrar y supervisar períodos oficiales.
 * Respeta el contrato T18 sin fabricar fechas ni estados ficticios.
 */
export function AdminConvocatoriasView() {
  const {
    data: paginaConvocatorias,
    isLoading,
    error,
    refetch,
  } = useConvocatoriasAdmin()
  const mutacionCrear = useCrearConvocatoria()

  const [tabActual, setTabActual] = useState<TabFiltro>("todas")
  const [modalCrearAbierto, setModalCrearAbierto] = useState(false)

  // Estado del formulario de nueva convocatoria
  const [nuevoCodigo, setNuevoCodigo] = useState("")
  const [nuevoTitulo, setNuevoTitulo] = useState("")
  const [nuevaApertura, setNuevaApertura] = useState("2026-11-01T08:00")
  const [nuevoCierre, setNuevoCierre] = useState("2026-11-30T23:59")

  const items = useMemo(
    () => paginaConvocatorias?.items ?? [],
    [paginaConvocatorias],
  )

  const itemsFiltrados = useMemo(() => {
    if (tabActual === "activas") {
      return items.filter((c) => !c.cerrada)
    }
    if (tabActual === "cerradas") {
      return items.filter((c) => c.cerrada)
    }
    return items
  }, [items, tabActual])

  const totalConvocatorias = items.length
  const activasCount = items.filter((c) => !c.cerrada).length
  const cerradasCount = items.filter((c) => c.cerrada).length

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
        apertura: aperturaIso,
        cierre: cierreIso,
        sectores: ["afiliados", "jubilados", "empleadores"],
        fases: [],
      })
      toast.success(
        "Convocatoria creada en estado deshabilitado según contrato T18.",
      )
      setModalCrearAbierto(false)
      setNuevoCodigo("")
      setNuevoTitulo("")
    } catch (err) {
      toast.error(
        err instanceof Error
          ? err.message
          : "Error al registrar la convocatoria en el servidor.",
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
            <CardDescription className="text-xs font-semibold uppercase">
              Total Convocatorias
            </CardDescription>
            <CardTitle className="text-2xl font-bold">
              {totalConvocatorias}
            </CardTitle>
          </CardHeader>
          <CardContent className="text-xs text-muted-foreground">
            Registradas en el servidor
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardDescription className="text-xs font-semibold uppercase">
              Convocatorias Activas
            </CardDescription>
            <CardTitle className="text-2xl font-bold text-emerald-600 dark:text-emerald-400">
              {activasCount}
            </CardTitle>
          </CardHeader>
          <CardContent className="text-xs text-muted-foreground">
            En período vigente según [apertura, cierre)
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardDescription className="text-xs font-semibold uppercase">
              Cerradas / Históricas
            </CardDescription>
            <CardTitle className="text-2xl font-bold text-muted-foreground">
              {cerradasCount}
            </CardTitle>
          </CardHeader>
          <CardContent className="text-xs text-muted-foreground">
            Concluidas o no admiten postulaciones
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
          <TabsList>
            <TabsTrigger value="todas">
              Todas ({totalConvocatorias})
            </TabsTrigger>
            <TabsTrigger value="activas">Activas ({activasCount})</TabsTrigger>
            <TabsTrigger value="cerradas">
              Cerradas / Históricas ({cerradasCount})
            </TabsTrigger>
          </TabsList>
        </Tabs>
      </div>

      {/* Tabla de convocatorias */}
      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-[180px]">Código</TableHead>
                <TableHead>Título Oficial</TableHead>
                <TableHead>Apertura y Cierre (ECT)</TableHead>
                <TableHead className="w-[130px]">Estado</TableHead>
                <TableHead className="text-right">Acciones</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {itemsFiltrados.length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={5}
                    className="h-32 text-center text-muted-foreground"
                  >
                    No se encontraron convocatorias en esta categoría.
                  </TableCell>
                </TableRow>
              ) : (
                itemsFiltrados.map((c: ResumenConvocatoriaAdmin) => {
                  const temporal = obtenerEstadoTemporal(
                    c.apertura,
                    c.cierre,
                    c.cerrada,
                  )
                  return (
                    <TableRow key={c.id}>
                      <TableCell className="font-medium">
                        <span className="font-mono text-sm font-bold text-primary">
                          {c.codigo}
                        </span>
                      </TableCell>
                      <TableCell>
                        <p className="font-semibold text-foreground line-clamp-2">
                          {c.titulo}
                        </p>
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
                              Fases y configuración
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
              Crea un período inicial deshabilitado (T18). La aprobación y
              publicación posterior se configuran en el panel de fases.
            </DialogDescription>
          </DialogHeader>

          <form onSubmit={handleGuardarNuevaConvocatoria} className="space-y-4">
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
              <Label htmlFor="titulo">Título oficial del concurso *</Label>
              <Input
                id="titulo"
                placeholder="Concurso Público de Méritos y Oposición…"
                value={nuevoTitulo}
                onChange={(e) => setNuevoTitulo(e.target.value)}
                required
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

            <DialogFooter className="pt-2">
              <Button
                type="button"
                variant="outline"
                onClick={() => setModalCrearAbierto(false)}
              >
                Cancelar
              </Button>
              <Button type="submit" disabled={mutacionCrear.isPending}>
                {mutacionCrear.isPending ? "Guardando…" : "Crear convocatoria"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  )
}
