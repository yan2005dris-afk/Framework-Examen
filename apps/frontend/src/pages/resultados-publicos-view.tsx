import { useState } from "react"
import { Link, useSearchParams } from "react-router"
import {
  ArrowLeftIcon,
  CheckCircle2Icon,
  GlobeIcon,
  SearchIcon,
  ShieldCheckIcon,
  UsersIcon,
} from "lucide-react"

import { EstadoCargando, EstadoError, EstadoVacio } from "@/components/estados"
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
import { SECTORES } from "@/features/convocatorias/modelo"
import { useConvocatoriasPublicas } from "@/features/convocatorias/queries"
import { useResultadosPublicosVigentes } from "@/features/resultados/queries"
import { RUTAS } from "@/lib/rutas"
import type { SectorConvocatoria } from "@/types/convocatoria"
import { formatearFechaEcuador } from "@/utils/fechas"

export function ResultadosPublicosView() {
  const [parametros, setParametros] = useSearchParams()
  const listadoConvocatorias = useConvocatoriasPublicas()
  const convocatorias =
    listadoConvocatorias.data?.pages.flatMap((p) => p.items) ?? []

  const periodSeleccionado =
    parametros.get("period") || convocatorias[0]?.id || ""
  const sectorSeleccionado = (parametros.get("sector") ||
    "afiliados") as SectorConvocatoria

  const [terminoBusqueda, setTerminoBusqueda] = useState("")

  const consulta = useResultadosPublicosVigentes(
    periodSeleccionado,
    sectorSeleccionado,
  )

  const actualizarParametro = (clave: string, valor: string) => {
    const nuevos = new URLSearchParams(parametros)
    nuevos.set(clave, valor)
    setParametros(nuevos)
  }

  const proyeccion = consulta.data
  const nombres = proyeccion?.nombres?.items ?? []
  const nombresFiltrados = terminoBusqueda.trim()
    ? nombres.filter((n) =>
        n.toLowerCase().includes(terminoBusqueda.trim().toLowerCase()),
      )
    : nombres

  return (
    <div className="mx-auto max-w-6xl space-y-8 px-4 py-8 sm:px-6 md:py-12">
      {/* Encabezado y navegación */}
      <div className="space-y-4">
        <Button asChild variant="ghost" size="sm" className="-ml-2">
          <Link to={RUTAS.inicio}>
            <ArrowLeftIcon className="mr-1 size-4" />
            Volver al inicio
          </Link>
        </Button>

        <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
          <div>
            <div className="flex items-center gap-2 text-primary">
              <GlobeIcon className="size-5" />
              <span className="text-xs font-semibold tracking-wider uppercase">
                Consulta Ciudadana Oficial
              </span>
            </div>
            <h1 className="text-3xl font-bold tracking-tight text-foreground sm:text-4xl">
              Nómina Oficial de Elegibles
            </h1>
            <p className="mt-1 max-w-3xl text-sm text-muted-foreground sm:text-base">
              Publicación oficial de postulantes elegibles conforme a la Ley
              Orgánica de Protección de Datos Personales (LOPDP) y resoluciones
              institucionales.
            </p>
          </div>
        </div>
      </div>

      {/* Selectores de Convocatoria y Sector */}
      <Card>
        <CardHeader className="pb-4">
          <CardTitle className="text-base font-semibold">
            Selección de convocatoria y sector
          </CardTitle>
          <CardDescription>
            Elija el período y el sector para visualizar los resultados
            oficiales publicados.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-3">
            <div className="space-y-1.5">
              <Label htmlFor="selector-convocatoria">
                Convocatoria / Período
              </Label>
              {convocatorias.length > 0 ? (
                <Select
                  value={periodSeleccionado}
                  onValueChange={(val) => actualizarParametro("period", val)}
                >
                  <SelectTrigger id="selector-convocatoria">
                    <SelectValue placeholder="Seleccione período" />
                  </SelectTrigger>
                  <SelectContent>
                    {convocatorias.map((c) => (
                      <SelectItem key={c.id} value={c.id}>
                        {c.codigo} · {c.titulo}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              ) : (
                <Input
                  id="selector-convocatoria"
                  placeholder="ID o código de período"
                  value={periodSeleccionado}
                  onChange={(e) =>
                    actualizarParametro("period", e.target.value)
                  }
                />
              )}
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="selector-sector">Sector de representación</Label>
              <Select
                value={sectorSeleccionado}
                onValueChange={(val) => actualizarParametro("sector", val)}
              >
                <SelectTrigger id="selector-sector">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="afiliados">
                    {SECTORES.afiliados}
                  </SelectItem>
                  <SelectItem value="jubilados">
                    {SECTORES.jubilados}
                  </SelectItem>
                  <SelectItem value="empleadores">
                    {SECTORES.empleadores}
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-1.5 sm:col-span-2 md:col-span-1">
              <Label htmlFor="busqueda-nombre">Buscar por nombre</Label>
              <div className="relative">
                <SearchIcon className="absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  id="busqueda-nombre"
                  placeholder="Filtrar por apellidos o nombres…"
                  value={terminoBusqueda}
                  onChange={(e) => setTerminoBusqueda(e.target.value)}
                  className="pl-9"
                  disabled={nombres.length === 0}
                />
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Aviso de Privacidad y LOPDP */}
      <Alert className="border-primary/20 bg-primary/5">
        <ShieldCheckIcon className="size-5 text-primary" />
        <AlertTitle className="text-foreground">
          Garantía de Privacidad y Protección de Datos
        </AlertTitle>
        <AlertDescription className="text-muted-foreground">
          En cumplimiento del principio de minimización y la Ley Orgánica de
          Protección de Datos Personales, esta nómina ciudadana expone
          únicamente la proyección de nombres y apellidos de los postulantes que
          han alcanzado la condición de elegibles y otorgaron su consentimiento
          expreso. No se publican números de cédula, correos de contacto ni
          calificaciones desagregadas.
        </AlertDescription>
      </Alert>

      {/* Estado y contenido */}
      {consulta.isPending ? (
        <EstadoCargando mensaje="Cargando nómina pública de resultados…" />
      ) : consulta.isError ? (
        <EstadoError
          error={consulta.error}
          titulo="No fue posible consultar la lista pública"
          onReintentar={() => void consulta.refetch()}
        />
      ) : !proyeccion || nombres.length === 0 ? (
        <EstadoVacio
          titulo="No hay resultados públicos vigentes"
          descripcion={`No se ha publicado una lista oficial de elegibles para el sector de ${SECTORES[sectorSeleccionado]} en este período, o la publicación aún se encuentra en etapa de aprobación institucional.`}
        />
      ) : (
        <div className="space-y-6">
          {/* Metadatos de la publicación oficial */}
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-4">
            <Card>
              <CardContent className="pt-6">
                <div className="text-xs font-medium text-muted-foreground">
                  Sector
                </div>
                <div className="mt-1 text-lg font-bold text-foreground">
                  {SECTORES[sectorSeleccionado] ?? proyeccion.sector}
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardContent className="pt-6">
                <div className="text-xs font-medium text-muted-foreground">
                  Versión publicada
                </div>
                <div className="mt-1 text-lg font-bold text-foreground">
                  Versión {proyeccion.version}
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardContent className="pt-6">
                <div className="text-xs font-medium text-muted-foreground">
                  Fecha de publicación
                </div>
                <div className="mt-1 text-sm font-semibold text-foreground">
                  {formatearFechaEcuador(proyeccion.publicadaEn, {
                    incluirHora: true,
                  })}
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardContent className="pt-6">
                <div className="text-xs font-medium text-muted-foreground">
                  Total postulantes elegibles
                </div>
                <div className="mt-1 text-lg font-bold text-foreground">
                  {nombres.length}
                </div>
              </CardContent>
            </Card>
          </div>

          {/* Tabla de nombres */}
          <Card>
            <CardHeader className="flex flex-row items-center justify-between pb-4">
              <div>
                <CardTitle className="flex items-center gap-2 text-base">
                  <UsersIcon className="size-4 text-primary" />
                  Postulantes Elegibles (Orden Alfabético)
                </CardTitle>
                <CardDescription>
                  Listado oficial de postulantes aptos para conformar la terna
                  de representación.
                </CardDescription>
              </div>
              <Badge variant="outline" className="text-xs">
                {nombresFiltrados.length} de {nombres.length} registros
              </Badge>
            </CardHeader>
            <CardContent>
              {nombresFiltrados.length === 0 ? (
                <div className="py-8 text-center text-sm text-muted-foreground">
                  No se encontraron postulantes que coincidan con &quot;
                  {terminoBusqueda}&quot;.
                </div>
              ) : (
                <div className="overflow-x-auto rounded-md border">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead className="w-16 text-center">#</TableHead>
                        <TableHead>Apellidos y Nombres</TableHead>
                        <TableHead className="w-40 text-center">
                          Condición
                        </TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {nombresFiltrados.map((nombre, idx) => (
                        <TableRow key={`${nombre}-${idx}`}>
                          <TableCell className="text-center font-mono text-xs text-muted-foreground">
                            {idx + 1}
                          </TableCell>
                          <TableCell className="font-medium text-foreground">
                            {nombre}
                          </TableCell>
                          <TableCell className="text-center">
                            <Badge
                              variant="secondary"
                              className="bg-emerald-500/10 text-emerald-700 dark:text-emerald-300"
                            >
                              <CheckCircle2Icon className="mr-1 size-3" />
                              Elegible
                            </Badge>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  )
}
