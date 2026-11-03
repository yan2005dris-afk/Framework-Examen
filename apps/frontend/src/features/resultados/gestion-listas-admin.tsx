import { useState } from "react"
import {
  AlertCircleIcon,
  CheckCircle2Icon,
  DownloadIcon,
  FileSpreadsheetIcon,
  GlobeIcon,
  ListOrderedIcon,
  LockIcon,
  PlusIcon,
  SendIcon,
  UsersIcon,
} from "lucide-react"

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
import { Checkbox } from "@/components/ui/checkbox"
import {
  Dialog,
  DialogContent,
  DialogDescription,
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
import { useSesion } from "@/features/auth/sesion-context"
import { useConvocatoriasRevision } from "@/features/revision/queries"
import {
  descargarCsvListaAdmin,
  type EntradaGenerarListaDto,
  type EntradaPublicarListaDto,
  type ListaResultadosMetadataDto,
} from "@/lib/clientes/resultados"
import {
  useEntradasListaAdmin,
  useGenerarListaAdmin,
  useListasAdmin,
  usePublicacionesAdmin,
  usePublicarListaAdmin,
} from "./queries"

const SECTORES = [
  { valor: "afiliados", etiqueta: "Afiliados al IESS" },
  { valor: "jubilados", etiqueta: "Jubilados del IESS" },
  { valor: "empleadores", etiqueta: "Empleadores" },
]

export function GestionListasAdmin() {
  const { postulante } = useSesion()
  const esAdmin = postulante?.rol === "administrador"

  const { data: convocatoriasData } = useConvocatoriasRevision()
  const convocatorias = convocatoriasData?.pages.flatMap((p) => p.items) ?? []

  const [periodoSeleccionado, setPeriodoSeleccionado] = useState(
    () => convocatorias[0]?.id ?? "",
  )
  const [sectorSeleccionado, setSectorSeleccionado] = useState("afiliados")

  // Consultas de listas y publicaciones
  const {
    data: listasData,
    isLoading: cargandoListas,
    error: errorListas,
    refetch: refetchListas,
  } = useListasAdmin(periodoSeleccionado, sectorSeleccionado)

  const { data: publicacionesData } = usePublicacionesAdmin(
    periodoSeleccionado,
    sectorSeleccionado,
  )

  const generarMutation = useGenerarListaAdmin(
    periodoSeleccionado,
    sectorSeleccionado,
  )
  const publicarMutation = usePublicarListaAdmin(
    listasData?.pages?.[0]?.items?.[0]?.id ?? "",
  )

  // Estados locales para generación de lista
  const [mostrarGenerar, setMostrarGenerar] = useState(false)
  const [referenciaOrden, setReferenciaOrden] = useState("")

  // Estados locales para publicación
  const [listaAPublicar, setListaAPublicar] =
    useState<ListaResultadosMetadataDto | null>(null)
  const [referenciaAprobacion, setReferenciaAprobacion] = useState("")
  const [esPublica, setEsPublica] = useState(false)
  const [referenciaPrivacidad, setReferenciaPrivacidad] = useState("")

  // Estados para modal de entradas
  const [listaVerEntradas, setListaVerEntradas] =
    useState<ListaResultadosMetadataDto | null>(null)
  const { data: entradasData, isLoading: cargandoEntradas } =
    useEntradasListaAdmin(listaVerEntradas?.id ?? "")

  const [descargandoCsv, setDescargandoCsv] = useState(false)

  const listas = listasData?.pages.flatMap((p) => p.items) ?? []
  const publicaciones = publicacionesData?.pages.flatMap((p) => p.items) ?? []
  const publicacionVigente = publicaciones[0]

  const manejarGenerar = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!referenciaOrden.trim()) return

    const entrada: EntradaGenerarListaDto = {
      reglas: {
        orden: ["resultados", "meritos"],
        referenciaOrden: referenciaOrden.trim(),
      },
    }

    await generarMutation.mutateAsync(entrada)
    setReferenciaOrden("")
    setMostrarGenerar(false)
  }

  const manejarPublicar = async () => {
    if (!listaAPublicar || !referenciaAprobacion.trim()) return
    if (esPublica && !referenciaPrivacidad.trim()) return

    const payload: EntradaPublicarListaDto = {
      versionPublicadaAnterior: publicacionVigente?.version ?? 0,
      referenciaAprobacion: referenciaAprobacion.trim(),
      publica: esPublica,
      referenciaPrivacidad: esPublica ? referenciaPrivacidad.trim() : undefined,
      sustituyeId: publicacionVigente?.id,
    }

    await publicarMutation.mutateAsync(payload)
    setListaAPublicar(null)
    setReferenciaAprobacion("")
    setReferenciaPrivacidad("")
    setEsPublica(false)
  }

  const descargarCsv = async (listId: string) => {
    try {
      setDescargandoCsv(true)
      const blob = await descargarCsvListaAdmin(listId)
      const url = URL.createObjectURL(blob)
      const link = document.createElement("a")
      link.href = url
      link.download = `lista-elegibles-${sectorSeleccionado}.csv`
      document.body.appendChild(link)
      link.click()
      link.remove()
      setTimeout(() => URL.revokeObjectURL(url), 1000)
    } catch {
      // Error handled by api
    } finally {
      setDescargandoCsv(false)
    }
  }

  return (
    <div className="space-y-6">
      {/* Controles de Selección de Convocatoria y Sector */}
      <Card>
        <CardHeader>
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div className="flex items-center gap-2">
              <ListOrderedIcon className="size-5 text-primary" />
              <CardTitle>Listas de Elegibles y Publicación Oficial</CardTitle>
            </div>
            {publicacionVigente && (
              <Badge
                variant={publicacionVigente.publica ? "default" : "secondary"}
                className="flex items-center gap-1"
              >
                {publicacionVigente.publica ? (
                  <>
                    <GlobeIcon className="size-3" />
                    <span>
                      Publicación Ciudadana Activa (v
                      {publicacionVigente.version})
                    </span>
                  </>
                ) : (
                  <>
                    <LockIcon className="size-3" />
                    <span>Versión Interna (v{publicacionVigente.version})</span>
                  </>
                )}
              </Badge>
            )}
          </div>
          <CardDescription>
            Generación de listas internas reproducibles con orden aprobado,
            exportación CSV y publicación de elegibles con proyección de
            privacidad.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-1">
              <Label htmlFor="periodo" className="text-xs font-semibold">
                Convocatoria
              </Label>
              <Select
                value={periodoSeleccionado}
                onValueChange={(val) => setPeriodoSeleccionado(val)}
              >
                <SelectTrigger id="periodo">
                  <SelectValue placeholder="Seleccione período" />
                </SelectTrigger>
                <SelectContent>
                  {convocatorias.map((c) => (
                    <SelectItem key={c.id} value={c.id}>
                      {c.titulo} ({c.codigo})
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-1">
              <Label htmlFor="sector" className="text-xs font-semibold">
                Sector / Cargo convocado
              </Label>
              <Select
                value={sectorSeleccionado}
                onValueChange={(val) => setSectorSeleccionado(val)}
              >
                <SelectTrigger id="sector">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {SECTORES.map((s) => (
                    <SelectItem key={s.valor} value={s.valor}>
                      {s.etiqueta}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          <div className="flex gap-2 pt-2">
            <Button
              size="sm"
              disabled={!periodoSeleccionado}
              onClick={() => setMostrarGenerar(!mostrarGenerar)}
            >
              <PlusIcon className="size-4 mr-1.5" />
              {mostrarGenerar ? "Cancelar" : "Generar Lista Interna"}
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Formulario de Generación de Lista */}
      {mostrarGenerar && (
        <Card className="border-primary/40">
          <CardHeader>
            <div className="flex items-center gap-2">
              <PlusIcon className="size-5 text-primary" />
              <CardTitle className="text-base">
                Generar Snapshot Interno de Resultados
              </CardTitle>
            </div>
            <CardDescription className="text-xs">
              Se validará el estado de las postulaciones, publicaciones
              definitivas de méritos, oposición y resolución de todas las
              reconsideraciones.
            </CardDescription>
          </CardHeader>
          <form onSubmit={manejarGenerar}>
            <CardContent className="space-y-4 text-xs">
              <Alert className="border-primary/20 bg-primary/5">
                <AlertCircleIcon className="size-4 text-primary" />
                <AlertTitle className="font-semibold text-xs">
                  Criterio de Ordenamiento Institucional
                </AlertTitle>
                <AlertDescription className="text-[11px] text-muted-foreground mt-1">
                  El motor aplica ordenamiento jerárquico descendente exacto:
                  primero por <strong>Resultados Consolidados</strong> y luego
                  por <strong>Méritos</strong>. Quienes empatan comparten puesto
                  sin adjudicar cargos automáticamente.
                </AlertDescription>
              </Alert>

              <div className="space-y-1">
                <Label
                  htmlFor="referenciaOrden"
                  className="text-xs font-semibold"
                >
                  Acta o resolución de aprobación del orden
                </Label>
                <Input
                  id="referenciaOrden"
                  value={referenciaOrden}
                  onChange={(e) => setReferenciaOrden(e.target.value)}
                  placeholder="Ej. Acta de Sesión Extraordinaria CESD-2026-004"
                />
              </div>
            </CardContent>
            <div className="flex justify-end gap-2 border-t p-4">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => setMostrarGenerar(false)}
              >
                Cancelar
              </Button>
              <Button
                type="submit"
                size="sm"
                disabled={generarMutation.isPending || !referenciaOrden.trim()}
              >
                <CheckCircle2Icon className="size-4 mr-1.5" />
                {generarMutation.isPending
                  ? "Generando Lista..."
                  : "Generar Lista Oficial"}
              </Button>
            </div>
          </form>
        </Card>
      )}

      {/* Listado de Versiones Generadas */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <FileSpreadsheetIcon className="size-5 text-primary" />
              <CardTitle>Versiones Internas de Listas</CardTitle>
            </div>
            <Badge variant="outline">
              {listas.length === 1 ? "1 Versión" : `${listas.length} Versiones`}
            </Badge>
          </div>
          <CardDescription>
            Snapshots reproducibles generados a partir de calificaciones
            selladas.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {cargandoListas ? (
            <EstadoCargando mensaje="Cargando listas de resultados…" />
          ) : errorListas ? (
            <EstadoError
              error={errorListas}
              titulo="Error al cargar las listas"
              onReintentar={() => void refetchListas()}
            />
          ) : listas.length === 0 ? (
            <div className="rounded-lg border border-dashed p-6 text-center text-xs text-muted-foreground">
              No se han generado listas de resultados para esta convocatoria y
              sector.
            </div>
          ) : (
            <div className="overflow-x-auto rounded-lg border">
              <table className="w-full text-left text-xs">
                <thead className="border-b bg-muted/50 font-semibold text-muted-foreground">
                  <tr>
                    <th className="p-3">Versión</th>
                    <th className="p-3">Generada</th>
                    <th className="p-3">Reglas de Orden</th>
                    <th className="p-3">Hash Fuentes</th>
                    <th className="p-3 text-right">Acciones</th>
                  </tr>
                </thead>
                <tbody>
                  {listas.map((lista) => (
                    <tr
                      key={lista.id}
                      className="border-b last:border-0 hover:bg-muted/20"
                    >
                      <td className="p-3 font-bold">v{lista.version}</td>
                      <td className="p-3">
                        {new Date(lista.creadaEn).toLocaleDateString("es-EC")}{" "}
                        {new Date(lista.creadaEn).toLocaleTimeString("es-EC", {
                          hour: "2-digit",
                          minute: "2-digit",
                        })}
                      </td>
                      <td className="p-3">
                        <span className="font-semibold">
                          {lista.reglas.orden.join(" → ")}
                        </span>
                        <p className="text-[10px] text-muted-foreground">
                          {lista.reglas.referenciaOrden}
                        </p>
                      </td>
                      <td className="p-3 font-mono text-[10px] truncate max-w-[120px]">
                        {lista.hashFuentes}
                      </td>
                      <td className="p-3 text-right space-x-1">
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-7 text-xs"
                          onClick={() => setListaVerEntradas(lista)}
                        >
                          <UsersIcon className="size-3 mr-1" />
                          Entradas
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-7 text-xs"
                          disabled={descargandoCsv}
                          onClick={() => descargarCsv(lista.id)}
                        >
                          <DownloadIcon className="size-3 mr-1" />
                          CSV
                        </Button>
                        {esAdmin && (
                          <Button
                            variant="default"
                            size="sm"
                            className="h-7 text-xs"
                            onClick={() => setListaAPublicar(lista)}
                          >
                            <SendIcon className="size-3 mr-1" />
                            Publicar
                          </Button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Modal para ver Entradas de la Lista */}
      <Dialog
        open={Boolean(listaVerEntradas)}
        onOpenChange={(open) => !open && setListaVerEntradas(null)}
      >
        <DialogContent className="max-w-4xl max-h-[85vh] flex flex-col">
          <DialogHeader>
            <DialogTitle>
              Entradas de la Lista de Elegibles v{listaVerEntradas?.version}
            </DialogTitle>
            <DialogDescription className="text-xs">
              Snapshot de postulantes, orden por méritos/resultados y
              consentimientos sellados.
            </DialogDescription>
          </DialogHeader>

          <div className="flex-1 overflow-y-auto space-y-3 py-2 text-xs">
            {cargandoEntradas ? (
              <EstadoCargando mensaje="Cargando entradas del snapshot…" />
            ) : (
              <div className="overflow-x-auto rounded-lg border">
                <table className="w-full text-left text-xs">
                  <thead className="border-b bg-muted/50 font-semibold text-muted-foreground">
                    <tr>
                      <th className="p-2.5">Puesto</th>
                      <th className="p-2.5">Postulante</th>
                      <th className="p-2.5">Código</th>
                      <th className="p-2.5">Estado</th>
                      <th className="p-2.5">Consentimiento</th>
                    </tr>
                  </thead>
                  <tbody>
                    {(entradasData?.pages.flatMap((p) => p.items) ?? []).map(
                      (entrada) => (
                        <tr
                          key={entrada.postulacionId}
                          className="border-b last:border-0 hover:bg-muted/20"
                        >
                          <td className="p-2.5 font-bold">
                            {entrada.puesto ?? "—"}
                            {entrada.empate && (
                              <Badge
                                variant="outline"
                                className="ml-1 text-[10px] text-amber-600"
                              >
                                Empate
                              </Badge>
                            )}
                          </td>
                          <td className="p-2.5 font-medium">
                            {entrada.apellidos} {entrada.nombres}
                          </td>
                          <td className="p-2.5 font-mono text-[11px] text-muted-foreground">
                            {entrada.codigo}
                          </td>
                          <td className="p-2.5">
                            <Badge
                              variant={
                                entrada.elegible ? "default" : "destructive"
                              }
                            >
                              {entrada.elegible ? "Elegible" : "No Elegible"}
                            </Badge>
                          </td>
                          <td className="p-2.5">
                            {entrada.consentimientoNombre ? (
                              <Badge
                                variant="outline"
                                className="text-emerald-600 border-emerald-500/40 text-[10px]"
                              >
                                Acreditado RF
                              </Badge>
                            ) : (
                              <Badge
                                variant="destructive"
                                className="text-[10px]"
                              >
                                Sin Consentimiento
                              </Badge>
                            )}
                          </td>
                        </tr>
                      ),
                    )}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </DialogContent>
      </Dialog>

      {/* Modal para Publicar Oficialmente la Lista */}
      <Dialog
        open={Boolean(listaAPublicar)}
        onOpenChange={(open) => !open && setListaAPublicar(null)}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              Publicar Lista Oficial de Elegibles (v{listaAPublicar?.version})
            </DialogTitle>
            <DialogDescription className="text-xs">
              Como Administrador, declare la resolución formal de publicación y
              defina el nivel de visibilidad institucional o ciudadano.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-2 text-xs">
            <div className="space-y-1">
              <Label htmlFor="refAprobacion" className="text-xs font-semibold">
                Referencia de aprobación institucional (obligatorio)
              </Label>
              <Input
                id="refAprobacion"
                value={referenciaAprobacion}
                onChange={(e) => setReferenciaAprobacion(e.target.value)}
                placeholder="Ej. Resolución CESD-2026-008"
              />
            </div>

            <div className="flex items-start space-x-2 rounded-lg border p-3 bg-muted/20">
              <Checkbox
                id="esPublica"
                checked={esPublica}
                onCheckedChange={(c) => setEsPublica(Boolean(c))}
              />
              <div className="space-y-1">
                <label
                  htmlFor="esPublica"
                  className="font-semibold text-xs cursor-pointer"
                >
                  Publicación Abierta a la Ciudadanía
                </label>
                <p className="text-[11px] text-muted-foreground">
                  Habilita la consulta pública sin sesión. Requiere que todos
                  los elegibles tengan su consentimiento sellado y una
                  referencia de privacidad.
                </p>
              </div>
            </div>

            {esPublica && (
              <div className="space-y-1 border-t pt-2">
                <Label
                  htmlFor="refPrivacidad"
                  className="text-xs font-semibold"
                >
                  Referencia de aprobación de privacidad (obligatorio para lista
                  pública)
                </Label>
                <Input
                  id="refPrivacidad"
                  value={referenciaPrivacidad}
                  onChange={(e) => setReferenciaPrivacidad(e.target.value)}
                  placeholder="Ej. Dictamen Jurídico de Protección de Datos DPD-2026-012"
                />
              </div>
            )}
          </div>

          <div className="flex justify-end gap-2 border-t pt-4">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setListaAPublicar(null)}
            >
              Cancelar
            </Button>
            <Button
              size="sm"
              disabled={
                publicarMutation.isPending ||
                !referenciaAprobacion.trim() ||
                (esPublica && !referenciaPrivacidad.trim())
              }
              onClick={manejarPublicar}
            >
              <SendIcon className="size-4 mr-1.5" />
              {publicarMutation.isPending
                ? "Publicando..."
                : "Confirmar Publicación Oficial"}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}
