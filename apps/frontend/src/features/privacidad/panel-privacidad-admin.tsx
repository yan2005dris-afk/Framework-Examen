import { useState, type FormEvent } from "react"
import {
  AlertTriangleIcon,
  ClockIcon,
  EyeIcon,
  HistoryIcon,
  LockIcon,
  PlusIcon,
  ScaleIcon,
  ShieldAlertIcon,
  ShieldCheckIcon,
  UnlockIcon,
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
import { ModalPoliticaPrivacidad } from "./modal-politica-privacidad"
import {
  useHistorialPoliticas,
  usePoliticaVigente,
  usePublicarPolitica,
  useRestringirPolitica,
  useRevisionesRetencion,
} from "./queries"
import type {
  PoliticaPrivacidadDto,
  PublicarPoliticaDto,
} from "@/lib/clientes/privacidad"

export interface PanelPrivacidadAdminProps {
  convocatoriaId: string
  convocatoriaTitulo?: string
}

/**
 * Panel de Administración de Privacidad y Tratamiento de Datos (FT28 / T28).
 * Permite a la Comisión/Administrador consultar la política vigente, publicar nuevas versiones,
 * aplicar restricciones cautelares y supervisar las revisiones de retención de expedientes.
 */
export function PanelPrivacidadAdmin({
  convocatoriaId,
  convocatoriaTitulo,
}: PanelPrivacidadAdminProps) {
  const consultaVigente = usePoliticaVigente(convocatoriaId)
  const consultaHistorial = useHistorialPoliticas(convocatoriaId)
  const consultaRetencion = useRevisionesRetencion(convocatoriaId)

  const mutacionPublicar = usePublicarPolitica(convocatoriaId)
  const mutacionRestringir = useRestringirPolitica(convocatoriaId)

  const politica = consultaVigente.data
  const [modalDetallePolitica, setModalDetallePolitica] =
    useState<PoliticaPrivacidadDto | null>(null)

  // Estado para modal de publicación
  const [modalPublicarAbierto, setModalPublicarAbierto] = useState(false)
  const [formPublicar, setFormPublicar] = useState({
    referenciaAprobacion: "",
    texto: "",
    informacionTratamiento: "",
    diasAcceso: 365,
    diasDocumentos: 180,
    diasReportes: 365,
    diasPublicacionNombres: 90,
    diasCorreo: 30,
    diasRevisionConservacion: 730,
    permiteExportacion: true,
    permiteCorreo: true,
    permiteNombresPublicos: true,
  })

  // Estado para modal de restricción cautelar
  const [modalRestriccionAbierto, setModalRestriccionAbierto] = useState(false)
  const [motivoRestriccion, setMotivoRestriccion] = useState("")

  const handleAbrirPublicar = () => {
    if (politica) {
      setFormPublicar({
        referenciaAprobacion: politica.definicion.referenciaAprobacion,
        texto: politica.definicion.texto,
        informacionTratamiento: politica.definicion.informacionTratamiento,
        diasAcceso: politica.definicion.diasAcceso,
        diasDocumentos: politica.definicion.diasDocumentos,
        diasReportes: politica.definicion.diasReportes,
        diasPublicacionNombres: politica.definicion.diasPublicacionNombres,
        diasCorreo: politica.definicion.diasCorreo,
        diasRevisionConservacion: politica.definicion.diasRevisionConservacion,
        permiteExportacion: politica.definicion.permiteExportacion,
        permiteCorreo: politica.definicion.permiteCorreo,
        permiteNombresPublicos: politica.definicion.permiteNombresPublicos,
      })
    }
    setModalPublicarAbierto(true)
  }

  const handleSubmitPublicar = (e: FormEvent) => {
    e.preventDefault()
    if (
      !formPublicar.referenciaAprobacion.trim() ||
      !formPublicar.texto.trim() ||
      !formPublicar.informacionTratamiento.trim()
    ) {
      return
    }

    const payload: PublicarPoliticaDto = {
      versionAnterior: politica?.versionControl ?? 0,
      definicion: {
        ...formPublicar,
        disposicionHistoricos: "revision_humana_sin_borrado_automatico",
      },
    }

    mutacionPublicar.mutate(payload, {
      onSuccess: () => {
        setModalPublicarAbierto(false)
      },
    })
  }

  const handleSubmitRestriccion = (e: FormEvent) => {
    e.preventDefault()
    if (!motivoRestriccion.trim() || !politica) return

    const nuevoEstado = !politica.restringido
    mutacionRestringir.mutate(
      {
        versionAnterior: politica.versionControl,
        restringido: nuevoEstado,
        motivo: motivoRestriccion.trim(),
      },
      {
        onSuccess: () => {
          setModalRestriccionAbierto(false)
          setMotivoRestriccion("")
        },
      },
    )
  }

  if (consultaVigente.isPending) {
    return (
      <EstadoCargando mensaje="Consultando políticas de privacidad de la convocatoria…" />
    )
  }

  return (
    <div className="space-y-6">
      {/* Cabecera / Estado de la Política Vigente */}
      <Card className="border-primary/20">
        <CardHeader>
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex items-center gap-2">
              <ShieldCheckIcon className="size-5 text-primary" />
              <CardTitle className="text-lg">
                Política Institucional Vigente (LOPDP)
              </CardTitle>
            </div>
            {politica ? (
              <div className="flex items-center gap-2">
                <Badge variant="outline" className="font-mono text-xs">
                  Versión {politica.version}
                </Badge>
                {politica.restringido ? (
                  <Badge variant="destructive" className="text-xs">
                    Restricción Cautelar
                  </Badge>
                ) : (
                  <Badge
                    variant="secondary"
                    className="bg-emerald-500/10 text-emerald-700 dark:text-emerald-300 text-xs"
                  >
                    Vigente
                  </Badge>
                )}
              </div>
            ) : (
              <Badge
                variant="outline"
                className="text-xs text-muted-foreground"
              >
                Sin versión publicada
              </Badge>
            )}
          </div>
          <CardDescription>
            {convocatoriaTitulo
              ? `Convocatoria: ${convocatoriaTitulo}`
              : "Reglas de tratamiento, plazos de acceso y garantías legales del concurso."}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {politica ? (
            <>
              {politica.restringido && (
                <Alert variant="destructive">
                  <AlertTriangleIcon className="size-4" />
                  <AlertTitle>Medida Cautelar Activa</AlertTitle>
                  <AlertDescription className="text-xs">
                    Las salidas de datos, exportaciones, envíos y sellos están
                    suspendidos para esta convocatoria conforme a la restricción
                    vigente.
                  </AlertDescription>
                </Alert>
              )}

              <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-xs">
                <div className="rounded-lg border p-3.5 bg-muted/20 space-y-1.5">
                  <span className="font-semibold text-foreground flex items-center gap-1.5">
                    <ScaleIcon className="size-4 text-primary" />
                    Aprobación Institucional
                  </span>
                  <p className="text-muted-foreground">
                    {politica.definicion.referenciaAprobacion}
                  </p>
                  <p className="text-[11px] text-muted-foreground pt-1">
                    Publicada:{" "}
                    {new Date(politica.publicadaEn).toLocaleDateString(
                      "es-EC",
                      {
                        year: "numeric",
                        month: "long",
                        day: "numeric",
                        hour: "2-digit",
                        minute: "2-digit",
                      },
                    )}
                  </p>
                </div>

                <div className="rounded-lg border p-3.5 bg-muted/20 space-y-1.5">
                  <span className="font-semibold text-foreground flex items-center gap-1.5">
                    <LockIcon className="size-4 text-primary" />
                    Huella de Integridad Criptográfica
                  </span>
                  <p className="font-mono text-[11px] text-foreground break-all">
                    {politica.hash}
                  </p>
                  <p className="text-[11px] text-muted-foreground pt-1">
                    Control de concurrencia: versión {politica.versionControl}
                  </p>
                </div>
              </div>

              {/* Botones de acción */}
              <div className="flex flex-wrap items-center gap-2 pt-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setModalDetallePolitica(politica)}
                >
                  <EyeIcon className="size-3.5 mr-1.5" />
                  Ver detalle completo
                </Button>
                <Button
                  variant={politica.restringido ? "default" : "destructive"}
                  size="sm"
                  onClick={() => setModalRestriccionAbierto(true)}
                >
                  {politica.restringido ? (
                    <>
                      <UnlockIcon className="size-3.5 mr-1.5" />
                      Levantar Restricción
                    </>
                  ) : (
                    <>
                      <ShieldAlertIcon className="size-3.5 mr-1.5" />
                      Aplicar Restricción Cautelar
                    </>
                  )}
                </Button>
                <Button size="sm" onClick={handleAbrirPublicar}>
                  <PlusIcon className="size-3.5 mr-1.5" />
                  Publicar Nueva Versión
                </Button>
              </div>
            </>
          ) : (
            <div className="space-y-3">
              <Alert className="border-amber-500/30 bg-amber-500/5 text-amber-900 dark:text-amber-200">
                <AlertTriangleIcon className="size-4 text-amber-600 dark:text-amber-400" />
                <AlertTitle>Convocatoria sin política publicada</AlertTitle>
                <AlertDescription className="text-xs">
                  Esta convocatoria no cuenta con una versión de política de
                  privacidad publicada. Los postulantes no podrán sellar sus
                  envíos hasta que la Comisión apruebe y publique la política
                  institucional.
                </AlertDescription>
              </Alert>

              <Button size="sm" onClick={() => setModalPublicarAbierto(true)}>
                <PlusIcon className="size-3.5 mr-1.5" />
                Publicar Primera Versión de Política
              </Button>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Historial de Versiones Documentales */}
      <Card>
        <CardHeader>
          <div className="flex items-center gap-2">
            <HistoryIcon className="size-4 text-primary" />
            <CardTitle className="text-base">
              Historial de Versiones Publicadas
            </CardTitle>
          </div>
          <CardDescription>
            Trazabilidad documental inmutable de las políticas emitidas para
            este período.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {consultaHistorial.isPending ? (
            <EstadoCargando mensaje="Cargando historial de políticas…" />
          ) : consultaHistorial.isError ? (
            <EstadoError
              error={consultaHistorial.error}
              titulo="Error al cargar historial"
              onReintentar={() => void consultaHistorial.refetch()}
            />
          ) : (consultaHistorial.data?.items?.length ?? 0) === 0 ? (
            <p className="text-xs text-muted-foreground text-center py-4">
              No hay versiones históricas registradas.
            </p>
          ) : (
            <div className="overflow-x-auto rounded-lg border">
              <Table className="text-xs">
                <TableHeader>
                  <TableRow>
                    <TableHead>Versión</TableHead>
                    <TableHead>Fecha de Emisión</TableHead>
                    <TableHead>Referencia de Aprobación</TableHead>
                    <TableHead>Hash Criptográfico</TableHead>
                    <TableHead className="text-right">Acción</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {consultaHistorial.data?.items?.map((item) => (
                    <TableRow key={item.id}>
                      <TableCell className="font-bold">
                        v{item.version}
                        {item.id === politica?.id && (
                          <Badge
                            variant="secondary"
                            className="ml-2 text-[10px]"
                          >
                            Actual
                          </Badge>
                        )}
                      </TableCell>
                      <TableCell>
                        {new Date(item.publicadaEn).toLocaleDateString("es-EC")}
                      </TableCell>
                      <TableCell>
                        {item.definicion.referenciaAprobacion}
                      </TableCell>
                      <TableCell className="font-mono text-[10px] truncate max-w-[140px]">
                        {item.hash}
                      </TableCell>
                      <TableCell className="text-right">
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-7 text-xs"
                          onClick={() => setModalDetallePolitica(item)}
                        >
                          <EyeIcon className="size-3 mr-1" />
                          Consultar
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Bandeja de Supervisión de Retención de Expedientes */}
      <Card>
        <CardHeader>
          <div className="flex items-center gap-2">
            <ClockIcon className="size-4 text-primary" />
            <CardTitle className="text-base">
              Supervisión de Retención y Conservación de Expedientes
            </CardTitle>
          </div>
          <CardDescription>
            Expedientes con plazo de revisión vencido según la política
            institucional vigente.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="rounded-lg border bg-muted/20 p-3 text-xs leading-relaxed text-muted-foreground">
            <strong>Régimen normativo:</strong> Revisión humana sin borrado
            automático. La llegada a la fecha límite genera una alerta para
            auditoría de conservación; el sistema protege la evidencia sellada y
            no ejecuta purgas de base de datos sin instrucción formal.
          </div>

          {consultaRetencion.isPending ? (
            <EstadoCargando mensaje="Consultando expedientes en retención…" />
          ) : (consultaRetencion.data?.items?.length ?? 0) === 0 ? (
            <p className="text-xs text-muted-foreground text-center py-4">
              No existen expedientes que requieran revisión de conservación en
              esta convocatoria.
            </p>
          ) : (
            <div className="overflow-x-auto rounded-lg border">
              <Table className="text-xs">
                <TableHeader>
                  <TableRow>
                    <TableHead>ID de Postulación</TableHead>
                    <TableHead>Política Vinculada</TableHead>
                    <TableHead>Fecha Límite de Revisión</TableHead>
                    <TableHead className="text-right">Estado</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {consultaRetencion.data?.items?.map((rev) => (
                    <TableRow key={rev.postulacionId}>
                      <TableCell className="font-mono text-[11px]">
                        {rev.postulacionId}
                      </TableCell>
                      <TableCell className="font-mono text-[11px]">
                        {rev.politicaId}
                      </TableCell>
                      <TableCell>
                        {new Date(rev.revisarDesde).toLocaleDateString("es-EC")}
                      </TableCell>
                      <TableCell className="text-right">
                        <Badge variant="outline" className="text-amber-600">
                          Revisión Pendiente
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

      {/* Modal para Publicar Nueva Versión */}
      <Dialog
        open={modalPublicarAbierto}
        onOpenChange={setModalPublicarAbierto}
      >
        <DialogContent className="max-w-xl max-h-[90vh] overflow-y-auto">
          <form onSubmit={handleSubmitPublicar}>
            <DialogHeader>
              <DialogTitle>
                Publicar Versión de Política de Privacidad
              </DialogTitle>
              <DialogDescription className="text-xs">
                Ingrese el texto legal, la información de tratamiento conforme a
                la LOPDP y los plazos aplicables a esta convocatoria.
              </DialogDescription>
            </DialogHeader>

            <div className="space-y-4 py-4 text-xs">
              <div className="space-y-1.5">
                <Label htmlFor="refAprobacion">Referencia de Aprobación</Label>
                <Input
                  id="refAprobacion"
                  placeholder="Ej. Resolución CD-BIESS-2026-004"
                  value={formPublicar.referenciaAprobacion}
                  onChange={(e) =>
                    setFormPublicar((prev) => ({
                      ...prev,
                      referenciaAprobacion: e.target.value,
                    }))
                  }
                  required
                />
              </div>

              <div className="space-y-1.5">
                <Label htmlFor="textoPolitica">
                  Texto Institucional de la Política
                </Label>
                <textarea
                  id="textoPolitica"
                  className="w-full min-h-[100px] rounded-md border bg-transparent p-2 text-xs focus:ring-1 focus:ring-primary"
                  placeholder="Texto completo de la política aprobado por el órgano competente…"
                  value={formPublicar.texto}
                  onChange={(e) =>
                    setFormPublicar((prev) => ({
                      ...prev,
                      texto: e.target.value,
                    }))
                  }
                  required
                />
              </div>

              <div className="space-y-1.5">
                <Label htmlFor="infoTratamiento">
                  Información de Tratamiento de Datos (Art. 7 LOPDP)
                </Label>
                <textarea
                  id="infoTratamiento"
                  className="w-full min-h-[80px] rounded-md border bg-transparent p-2 text-xs focus:ring-1 focus:ring-primary"
                  placeholder="Finalidades, destinatarios y derechos del titular…"
                  value={formPublicar.informacionTratamiento}
                  onChange={(e) =>
                    setFormPublicar((prev) => ({
                      ...prev,
                      informacionTratamiento: e.target.value,
                    }))
                  }
                  required
                />
              </div>

              <div className="grid grid-cols-2 gap-3 pt-2">
                <div className="space-y-1">
                  <Label htmlFor="diasAcceso">Días Acceso a Expediente</Label>
                  <Input
                    id="diasAcceso"
                    type="number"
                    min={1}
                    max={36500}
                    value={formPublicar.diasAcceso}
                    onChange={(e) =>
                      setFormPublicar((prev) => ({
                        ...prev,
                        diasAcceso: Number(e.target.value),
                      }))
                    }
                    required
                  />
                </div>
                <div className="space-y-1">
                  <Label htmlFor="diasDocumentos">Días Documentos PDF</Label>
                  <Input
                    id="diasDocumentos"
                    type="number"
                    min={1}
                    max={36500}
                    value={formPublicar.diasDocumentos}
                    onChange={(e) =>
                      setFormPublicar((prev) => ({
                        ...prev,
                        diasDocumentos: Number(e.target.value),
                      }))
                    }
                    required
                  />
                </div>
                <div className="space-y-1">
                  <Label htmlFor="diasPublicacion">
                    Días Publicación Nombres
                  </Label>
                  <Input
                    id="diasPublicacion"
                    type="number"
                    min={1}
                    max={36500}
                    value={formPublicar.diasPublicacionNombres}
                    onChange={(e) =>
                      setFormPublicar((prev) => ({
                        ...prev,
                        diasPublicacionNombres: Number(e.target.value),
                      }))
                    }
                    required
                  />
                </div>
                <div className="space-y-1">
                  <Label htmlFor="diasRevision">Días Revisión Retención</Label>
                  <Input
                    id="diasRevision"
                    type="number"
                    min={1}
                    max={36500}
                    value={formPublicar.diasRevisionConservacion}
                    onChange={(e) =>
                      setFormPublicar((prev) => ({
                        ...prev,
                        diasRevisionConservacion: Number(e.target.value),
                      }))
                    }
                    required
                  />
                </div>
              </div>

              <div className="space-y-2 pt-2 border-t">
                <div className="flex items-center justify-between">
                  <Label
                    htmlFor="permiteExportacion"
                    className="cursor-pointer"
                  >
                    Permitir Exportación de Matrices (Excel/PDF)
                  </Label>
                  <Switch
                    id="permiteExportacion"
                    checked={formPublicar.permiteExportacion}
                    onCheckedChange={(checked) =>
                      setFormPublicar((prev) => ({
                        ...prev,
                        permiteExportacion: checked,
                      }))
                    }
                  />
                </div>
                <div className="flex items-center justify-between">
                  <Label htmlFor="permiteCorreo" className="cursor-pointer">
                    Permitir Notificaciones por Correo Electrónico
                  </Label>
                  <Switch
                    id="permiteCorreo"
                    checked={formPublicar.permiteCorreo}
                    onCheckedChange={(checked) =>
                      setFormPublicar((prev) => ({
                        ...prev,
                        permiteCorreo: checked,
                      }))
                    }
                  />
                </div>
                <div className="flex items-center justify-between">
                  <Label htmlFor="permiteNombres" className="cursor-pointer">
                    Permitir Difusión Pública Ciudadana de Elegibles
                  </Label>
                  <Switch
                    id="permiteNombres"
                    checked={formPublicar.permiteNombresPublicos}
                    onCheckedChange={(checked) =>
                      setFormPublicar((prev) => ({
                        ...prev,
                        permiteNombresPublicos: checked,
                      }))
                    }
                  />
                </div>
              </div>
            </div>

            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => setModalPublicarAbierto(false)}
              >
                Cancelar
              </Button>
              <Button type="submit" disabled={mutacionPublicar.isPending}>
                {mutacionPublicar.isPending
                  ? "Publicando…"
                  : "Publicar Política Oficial"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Modal para Restricción Cautelar */}
      <Dialog
        open={modalRestriccionAbierto}
        onOpenChange={setModalRestriccionAbierto}
      >
        <DialogContent className="max-w-md">
          <form onSubmit={handleSubmitRestriccion}>
            <DialogHeader>
              <DialogTitle className="flex items-center gap-2">
                <AlertTriangleIcon className="size-5 text-destructive" />
                <span>
                  {politica?.restringido
                    ? "Levantar Restricción Cautelar"
                    : "Aplicar Restricción Cautelar de Privacidad"}
                </span>
              </DialogTitle>
              <DialogDescription className="text-xs">
                {politica?.restringido
                  ? "Esta acción restablecerá el sello y las salidas de expedientes de esta convocatoria."
                  : "Esta medida bloquea de inmediato todo envío y salida de expedientes hasta su resolución."}
              </DialogDescription>
            </DialogHeader>

            <div className="space-y-3 py-4 text-xs">
              <div className="space-y-1.5">
                <Label htmlFor="motivoRestriccion">
                  Motivo de la Disposición Administrativa
                </Label>
                <textarea
                  id="motivoRestriccion"
                  className="w-full min-h-[80px] rounded-md border bg-transparent p-2 text-xs focus:ring-1 focus:ring-primary"
                  placeholder="Indique el motivo técnico o administrativo de la medida…"
                  value={motivoRestriccion}
                  onChange={(e) => setMotivoRestriccion(e.target.value)}
                  required
                />
              </div>
            </div>

            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => setModalRestriccionAbierto(false)}
              >
                Cancelar
              </Button>
              <Button
                type="submit"
                variant={politica?.restringido ? "default" : "destructive"}
                disabled={mutacionRestringir.isPending}
              >
                {mutacionRestringir.isPending
                  ? "Procesando…"
                  : politica?.restringido
                    ? "Confirmar Restitución"
                    : "Confirmar Restricción"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Modal de Detalle Documental de la Política */}
      <ModalPoliticaPrivacidad
        politica={modalDetallePolitica}
        abierto={Boolean(modalDetallePolitica)}
        onCerrar={() => setModalDetallePolitica(null)}
      />
    </div>
  )
}
