import { useState } from "react"
import {
  AlertCircleIcon,
  CheckCircle2Icon,
  ClockIcon,
  EyeIcon,
  FileCodeIcon,
  HelpCircleIcon,
  LockIcon,
  PlusIcon,
  SettingsIcon,
  ShieldAlertIcon,
  SlidersIcon,
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
import { Textarea } from "@/components/ui/textarea"
import { useSesion } from "@/features/auth/sesion-context"
import { useConvocatoriasRevision } from "@/features/revision/queries"
import {
  useConfigurarBanco,
  useCrearVersionBanco,
  useVersionBancoDetalle,
  useVersionesBanco,
} from "./queries"
import type {
  EntradaBancoPreguntasDto,
  PreguntaBancoDto,
  ReglasConfiguracionExamenDto,
} from "@/lib/clientes/preguntas"
import { formatearFechaEcuador } from "@/utils/fechas"

const EJEMPLO_PREGUNTAS_JSON = JSON.stringify(
  [
    {
      id: "p1",
      enunciado:
        "¿Cuál es el marco regulatorio principal para la gestión de fondos previsionales del BIESS?",
      opciones: [
        {
          id: "a",
          texto: "Ley del Banco del Instituto Ecuatoriano de Seguridad Social",
        },
        { id: "b", texto: "Código de Comercio" },
        { id: "c", texto: "Ley de Compañías" },
      ],
      correcta: "a",
    },
    {
      id: "p2",
      enunciado:
        "¿Qué quorum de instalación requiere el Directorio del BIESS para sesionar válidamente?",
      opciones: [
        { id: "a", texto: "Mayoría simple de sus integrantes" },
        {
          id: "b",
          texto:
            "La totalidad de los representantes y el Presidente del Directorio",
        },
        { id: "c", texto: "Dos tercios de los miembros con derecho a voto" },
      ],
      correcta: "a",
    },
  ],
  null,
  2,
)

export function GestionBancoPreguntas() {
  const { postulante } = useSesion()
  const esAdmin = postulante?.rol === "administrador"

  const { data: convocatoriasData } = useConvocatoriasRevision()
  const convocatorias = convocatoriasData?.pages.flatMap((p) => p.items) ?? []
  const [periodoSeleccionado, setPeriodoSeleccionado] = useState<string>("")
  const [sectorSeleccionado, setSectorSeleccionado] =
    useState<string>("afiliados")

  const periodo = periodoSeleccionado || convocatorias[0]?.id || ""

  // Consultas y mutaciones
  const listadoVersiones = useVersionesBanco(periodo, sectorSeleccionado)
  const versiones = listadoVersiones.data?.pages.flatMap((p) => p.items) ?? []
  const mutacionCrear = useCrearVersionBanco(periodo, sectorSeleccionado)

  // Estado modales
  const [modalNuevaVersion, setModalNuevaVersion] = useState(false)
  const [referenciaNueva, setReferenciaNueva] = useState("")
  const [jsonPreguntas, setJsonPreguntas] = useState(EJEMPLO_PREGUNTAS_JSON)
  const [errorValidacionJson, setErrorValidacionJson] = useState<string | null>(
    null,
  )

  const [bancoIdDetalle, setBancoIdDetalle] = useState<string | null>(null)
  const consultaDetalle = useVersionBancoDetalle(bancoIdDetalle ?? "")

  const [bancoIdConfigurar, setBancoIdConfigurar] = useState<string | null>(
    null,
  )
  const mutacionConfigurar = useConfigurarBanco(bancoIdConfigurar ?? "")

  const [configForm, setConfigForm] = useState<{
    cantidadPreguntas: number
    duracionMinutos: number
    maximo: number
    umbral: number
    redondeo: "inferior" | "mas_cercano"
    referenciaAprobacion: string
    abreEn: string
    cierraEn: string
  }>({
    cantidadPreguntas: 20,
    duracionMinutos: 60,
    maximo: 100,
    umbral: 70,
    redondeo: "mas_cercano",
    referenciaAprobacion: "",
    abreEn: "",
    cierraEn: "",
  })

  // Validar y crear nueva versión
  const handleGuardarNuevaVersion = () => {
    setErrorValidacionJson(null)
    if (!referenciaNueva.trim()) {
      setErrorValidacionJson(
        "Debe ingresar el documento institucional o acta de aprobación del banco.",
      )
      return
    }

    let parsed: unknown
    try {
      parsed = JSON.parse(jsonPreguntas)
    } catch {
      setErrorValidacionJson(
        "El contenido no es un JSON válido. Verifique la sintaxis.",
      )
      return
    }

    if (!Array.isArray(parsed) || parsed.length === 0) {
      setErrorValidacionJson(
        "El JSON debe contener un arreglo con al menos una pregunta.",
      )
      return
    }

    const preguntasValidadas: PreguntaBancoDto[] = []
    const idsVistos = new Set<string>()

    for (let i = 0; i < parsed.length; i++) {
      const p = parsed[i] as Record<string, unknown>
      if (!p.id || typeof p.id !== "string") {
        setErrorValidacionJson(`La pregunta #${i + 1} no tiene un 'id' válido.`)
        return
      }
      if (idsVistos.has(p.id)) {
        setErrorValidacionJson(
          `El ID de pregunta '${p.id}' está duplicado en la lista.`,
        )
        return
      }
      idsVistos.add(p.id)

      if (!p.enunciado || typeof p.enunciado !== "string") {
        setErrorValidacionJson(
          `La pregunta '${p.id}' no tiene un enunciado válido.`,
        )
        return
      }

      if (
        !Array.isArray(p.opciones) ||
        p.opciones.length < 2 ||
        p.opciones.length > 8
      ) {
        setErrorValidacionJson(
          `La pregunta '${p.id}' debe tener entre 2 y 8 opciones.`,
        )
        return
      }

      const opcionIds = new Set<string>()
      for (const op of p.opciones as Record<string, unknown>[]) {
        if (!op.id || typeof op.id !== "string" || !op.texto) {
          setErrorValidacionJson(
            `Una opción en la pregunta '${p.id}' no tiene id o texto válido.`,
          )
          return
        }
        if (opcionIds.has(op.id)) {
          setErrorValidacionJson(
            `La opción '${op.id}' está duplicada en la pregunta '${p.id}'.`,
          )
          return
        }
        opcionIds.add(op.id)
      }

      if (
        !p.correcta ||
        typeof p.correcta !== "string" ||
        !opcionIds.has(p.correcta)
      ) {
        setErrorValidacionJson(
          `La respuesta correcta '${String(p.correcta)}' de la pregunta '${p.id}' no coincide con ninguna de sus opciones.`,
        )
        return
      }

      preguntasValidadas.push({
        id: p.id,
        enunciado: p.enunciado,
        opciones: p.opciones as { id: string; texto: string }[],
        correcta: p.correcta,
      })
    }

    const versionAnterior =
      versiones.length > 0 ? Math.max(...versiones.map((v) => v.version)) : 0

    const entrada: EntradaBancoPreguntasDto = {
      versionAnterior,
      referencia: referenciaNueva.trim(),
      preguntas: preguntasValidadas,
    }

    mutacionCrear.mutate(entrada, {
      onSuccess: () => {
        setModalNuevaVersion(false)
        setReferenciaNueva("")
      },
    })
  }

  // Guardar configuración de evaluación (FT25)
  const handleGuardarConfiguracion = () => {
    if (!bancoIdConfigurar) return
    if (!configForm.referenciaAprobacion.trim()) {
      alert("Debe ingresar la referencia o acta de aprobación.")
      return
    }
    if (!configForm.abreEn || !configForm.cierraEn) {
      alert("Debe especificar la fecha y hora de apertura y cierre.")
      return
    }

    const reglas: ReglasConfiguracionExamenDto = {
      cantidadPreguntas: Number(configForm.cantidadPreguntas),
      duracionSegundos: Number(configForm.duracionMinutos) * 60,
      maximo: Number(configForm.maximo),
      umbral: Number(configForm.umbral),
      redondeo: configForm.redondeo,
      politicaIntentos: "un_intento",
      politicaUnicidad: "orden_preguntas_unico_rechazar_agotamiento",
      referenciaAprobacion: configForm.referenciaAprobacion.trim(),
      abreEn: new Date(configForm.abreEn).toISOString(),
      cierraEn: new Date(configForm.cierraEn).toISOString(),
    }

    mutacionConfigurar.mutate(reglas, {
      onSuccess: () => {
        setBancoIdConfigurar(null)
      },
    })
  }

  return (
    <div className="space-y-6">
      {/* Selector de Período y Sector */}
      <Card>
        <CardHeader className="pb-4">
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <CardTitle className="flex items-center gap-2 text-base font-semibold">
                <HelpCircleIcon className="size-5 text-primary" />
                Banco de Preguntas y Evaluación Técnica (EF-14 / FT24)
              </CardTitle>
              <CardDescription>
                Administración oficial de reactivos, opciones y matrices
                auditadas para el examen teórico.
              </CardDescription>
            </div>
            {esAdmin && (
              <Button
                onClick={() => setModalNuevaVersion(true)}
                size="sm"
                className="self-start sm:self-auto"
              >
                <PlusIcon className="mr-1.5 size-4" />
                Cargar nueva versión
              </Button>
            )}
          </div>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-3">
            <div className="space-y-1">
              <Label htmlFor="periodo-banco" className="text-xs font-semibold">
                Convocatoria / Período
              </Label>
              <Select
                value={periodo}
                onValueChange={(val) => setPeriodoSeleccionado(val)}
              >
                <SelectTrigger id="periodo-banco">
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
              <Label htmlFor="sector-banco" className="text-xs font-semibold">
                Sector convocado
              </Label>
              <Select
                value={sectorSeleccionado}
                onValueChange={setSectorSeleccionado}
              >
                <SelectTrigger id="sector-banco">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="afiliados">Afiliados</SelectItem>
                  <SelectItem value="jubilados">Jubilados</SelectItem>
                  <SelectItem value="empleadores">Empleadores</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Alerta de Garantía de Inmutabilidad y Auditoría */}
      <Alert className="border-amber-500/20 bg-amber-500/5 text-amber-900 dark:text-amber-200">
        <ShieldAlertIcon className="size-5 text-amber-600 dark:text-amber-400" />
        <AlertTitle>Inmutabilidad y Auditoría de Reactivos</AlertTitle>
        <AlertDescription className="text-xs leading-relaxed text-amber-800/90 dark:text-amber-300/90">
          Toda carga de banco genera una nueva versión inmutable con cálculo de
          hash criptográfico SHA-256. Los exámenes ya iniciados o completados
          conservan su versión original congelada. La visualización de la matriz
          de respuestas correctas queda estrictamente registrada en la bitácora
          de auditoría.
        </AlertDescription>
      </Alert>

      {/* Tabla de versiones del banco */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between pb-3">
          <div>
            <CardTitle className="text-sm font-semibold">
              Versiones Históricas del Banco
            </CardTitle>
            <CardDescription className="text-xs">
              Historial de reactivos aprobados para este sector.
            </CardDescription>
          </div>
          <Badge variant="outline" className="text-xs font-mono">
            {versiones.length} versiones registradas
          </Badge>
        </CardHeader>
        <CardContent>
          {listadoVersiones.isPending ? (
            <EstadoCargando mensaje="Consultando versiones del banco de preguntas…" />
          ) : listadoVersiones.isError ? (
            <EstadoError
              error={listadoVersiones.error}
              titulo="No se pudo cargar el historial del banco"
              onReintentar={() => void listadoVersiones.refetch()}
            />
          ) : versiones.length === 0 ? (
            <div className="py-8 text-center text-sm text-muted-foreground">
              No se han registrado versiones del banco de preguntas para este
              sector.
            </div>
          ) : (
            <div className="overflow-x-auto rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-20 text-center">Versión</TableHead>
                    <TableHead>Referencia Institucional</TableHead>
                    <TableHead className="w-28 text-center">
                      Reactivos
                    </TableHead>
                    <TableHead>Hash SHA-256</TableHead>
                    <TableHead className="w-40">Fecha Creación</TableHead>
                    <TableHead className="w-44 text-right">Acciones</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {versiones.map((v) => (
                    <TableRow key={v.id}>
                      <TableCell className="text-center font-bold">
                        v{v.version}
                      </TableCell>
                      <TableCell className="font-medium text-foreground">
                        {v.definicion?.referencia ?? "Aprobación oficial"}
                      </TableCell>
                      <TableCell className="text-center">
                        <Badge variant="secondary">
                          {v.definicion?.preguntas?.length ?? 0} preguntas
                        </Badge>
                      </TableCell>
                      <TableCell className="font-mono text-xs text-muted-foreground">
                        {v.hash.substring(0, 16)}…
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground">
                        {formatearFechaEcuador(v.creadaEn, {
                          incluirHora: true,
                        })}
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex items-center justify-end gap-1.5">
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => setBancoIdDetalle(v.id)}
                            title="Ver reactivos y matriz correcta (auditado)"
                          >
                            <EyeIcon className="mr-1 size-3.5" />
                            Ver
                          </Button>
                          {esAdmin && (
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => {
                                setBancoIdConfigurar(v.id)
                                setConfigForm((prev) => ({
                                  ...prev,
                                  cantidadPreguntas: Math.min(
                                    20,
                                    v.definicion?.preguntas?.length || 20,
                                  ),
                                }))
                              }}
                              title="Configurar y aprobar parámetros de evaluación (FT25)"
                            >
                              <SettingsIcon className="mr-1 size-3.5" />
                              Configurar
                            </Button>
                          )}
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Modal: Crear / Cargar nueva versión del banco */}
      <Dialog open={modalNuevaVersion} onOpenChange={setModalNuevaVersion}>
        <DialogContent className="max-w-2xl sm:max-h-[90vh] sm:overflow-y-auto">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <FileCodeIcon className="size-5 text-primary" />
              Cargar Nueva Versión del Banco de Preguntas
            </DialogTitle>
            <DialogDescription>
              Cargue una definición completa e inmutable. El formato debe ser un
              arreglo JSON con enunciados, opciones y clave de respuesta
              correcta.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-2">
            <div className="space-y-1.5">
              <Label htmlFor="referencia-banco">
                Referencia institucional / Acta de aprobación *
              </Label>
              <Input
                id="referencia-banco"
                placeholder="Ej. Resolución No. C.D. 701 - Banco Aprobado"
                value={referenciaNueva}
                onChange={(e) => setReferenciaNueva(e.target.value)}
              />
            </div>

            <div className="space-y-1.5">
              <div className="flex items-center justify-between">
                <Label htmlFor="json-preguntas">
                  Estructura JSON de reactivos *
                </Label>
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-6 text-xs text-primary"
                  onClick={() => setJsonPreguntas(EJEMPLO_PREGUNTAS_JSON)}
                >
                  Restaurar formato de ejemplo
                </Button>
              </div>
              <Textarea
                id="json-preguntas"
                rows={12}
                className="font-mono text-xs"
                value={jsonPreguntas}
                onChange={(e) => setJsonPreguntas(e.target.value)}
              />
            </div>

            {errorValidacionJson && (
              <Alert variant="destructive">
                <AlertCircleIcon className="size-4" />
                <AlertTitle>Error en la estructura del banco</AlertTitle>
                <AlertDescription className="text-xs">
                  {errorValidacionJson}
                </AlertDescription>
              </Alert>
            )}
          </div>

          <DialogFooter className="gap-2 sm:gap-0">
            <Button
              variant="outline"
              onClick={() => setModalNuevaVersion(false)}
            >
              Cancelar
            </Button>
            <Button
              onClick={handleGuardarNuevaVersion}
              disabled={mutacionCrear.isPending}
            >
              {mutacionCrear.isPending ? "Registrando…" : "Registrar versión"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Modal: Ver versión y matriz correcta (Auditado) */}
      <Dialog
        open={Boolean(bancoIdDetalle)}
        onOpenChange={(open) => !open && setBancoIdDetalle(null)}
      >
        <DialogContent className="max-w-3xl sm:max-h-[85vh] sm:overflow-y-auto">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <LockIcon className="size-5 text-amber-500" />
              Matriz de Preguntas y Claves de Calificación
            </DialogTitle>
            <DialogDescription>
              Versión {consultaDetalle.data?.version} · Referencia:{" "}
              {consultaDetalle.data?.definicion?.referencia}
            </DialogDescription>
          </DialogHeader>

          {consultaDetalle.isPending ? (
            <EstadoCargando mensaje="Recuperando matriz auditada del banco…" />
          ) : consultaDetalle.isError ? (
            <EstadoError
              error={consultaDetalle.error}
              titulo="No se pudo consultar el banco"
            />
          ) : (
            <div className="space-y-4 py-2">
              <Alert className="border-amber-500/20 bg-amber-500/10 text-amber-900 dark:text-amber-200">
                <AlertCircleIcon className="size-4 text-amber-600 dark:text-amber-400" />
                <AlertTitle>Acceso Restringido y Auditado</AlertTitle>
                <AlertDescription className="text-xs">
                  Esta consulta ha sido registrada en la bitácora
                  administrativa. Las respuestas correctas son de uso reservado
                  para la Comisión de Selección.
                </AlertDescription>
              </Alert>

              <div className="space-y-4">
                {consultaDetalle.data?.definicion?.preguntas?.map((p, idx) => (
                  <Card key={p.id} className="p-4">
                    <div className="flex items-start justify-between gap-2">
                      <span className="font-mono text-xs font-bold text-muted-foreground">
                        Pregunta {idx + 1} ({p.id})
                      </span>
                      <Badge
                        variant="secondary"
                        className="font-mono text-[11px]"
                      >
                        Clave: {p.correcta}
                      </Badge>
                    </div>
                    <p className="mt-2 text-sm font-medium text-foreground">
                      {p.enunciado}
                    </p>
                    <div className="mt-3 space-y-1.5 pl-2">
                      {p.opciones.map((op) => {
                        const esCorrecta = op.id === p.correcta
                        return (
                          <div
                            key={op.id}
                            className={`flex items-center gap-2 rounded px-2.5 py-1 text-xs ${
                              esCorrecta
                                ? "border border-emerald-500/30 bg-emerald-500/10 font-semibold text-emerald-800 dark:text-emerald-200"
                                : "text-muted-foreground"
                            }`}
                          >
                            <span className="font-mono font-bold">
                              {op.id})
                            </span>
                            <span>{op.texto}</span>
                            {esCorrecta && (
                              <CheckCircle2Icon className="ml-auto size-3.5 text-emerald-600 dark:text-emerald-400" />
                            )}
                          </div>
                        )
                      })}
                    </div>
                  </Card>
                ))}
              </div>
            </div>
          )}

          <DialogFooter>
            <Button variant="outline" onClick={() => setBancoIdDetalle(null)}>
              Cerrar
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Modal: Configurar Parámetros de Evaluación Técnica (FT25) */}
      <Dialog
        open={Boolean(bancoIdConfigurar)}
        onOpenChange={(open) => !open && setBancoIdConfigurar(null)}
      >
        <DialogContent className="max-w-xl sm:max-h-[85vh] sm:overflow-y-auto">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <SlidersIcon className="size-5 text-primary" />
              Configurar Evaluación Técnica (FT25)
            </DialogTitle>
            <DialogDescription>
              Apruebe la cantidad de preguntas, duración, umbral mínimo y
              ventana de aplicación para esta versión del banco.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-2">
            <div className="space-y-1.5">
              <Label htmlFor="config-ref">
                Referencia institucional / Acta de Consejo *
              </Label>
              <Input
                id="config-ref"
                placeholder="Ej. Acta No. 04-2026 de la Comisión de Selección"
                value={configForm.referenciaAprobacion}
                onChange={(e) =>
                  setConfigForm((prev) => ({
                    ...prev,
                    referenciaAprobacion: e.target.value,
                  }))
                }
              />
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-1.5">
                <Label htmlFor="config-cant">Cantidad de reactivos *</Label>
                <Input
                  id="config-cant"
                  type="number"
                  min={1}
                  max={100}
                  value={configForm.cantidadPreguntas}
                  onChange={(e) =>
                    setConfigForm((prev) => ({
                      ...prev,
                      cantidadPreguntas: Number(e.target.value),
                    }))
                  }
                />
              </div>

              <div className="space-y-1.5">
                <Label htmlFor="config-dur">Duración (minutos) *</Label>
                <Input
                  id="config-dur"
                  type="number"
                  min={5}
                  max={240}
                  value={configForm.duracionMinutos}
                  onChange={(e) =>
                    setConfigForm((prev) => ({
                      ...prev,
                      duracionMinutos: Number(e.target.value),
                    }))
                  }
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-1.5">
                <Label htmlFor="config-max">Puntaje máximo *</Label>
                <Input
                  id="config-max"
                  type="number"
                  min={1}
                  value={configForm.maximo}
                  onChange={(e) =>
                    setConfigForm((prev) => ({
                      ...prev,
                      maximo: Number(e.target.value),
                    }))
                  }
                />
              </div>

              <div className="space-y-1.5">
                <Label htmlFor="config-umbral">Umbral de aprobación *</Label>
                <Input
                  id="config-umbral"
                  type="number"
                  min={0}
                  max={configForm.maximo}
                  value={configForm.umbral}
                  onChange={(e) =>
                    setConfigForm((prev) => ({
                      ...prev,
                      umbral: Number(e.target.value),
                    }))
                  }
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-1.5">
                <Label htmlFor="config-abre">
                  Apertura de sala (Ecuador) *
                </Label>
                <Input
                  id="config-abre"
                  type="datetime-local"
                  value={configForm.abreEn}
                  onChange={(e) =>
                    setConfigForm((prev) => ({
                      ...prev,
                      abreEn: e.target.value,
                    }))
                  }
                />
              </div>

              <div className="space-y-1.5">
                <Label htmlFor="config-cierra">
                  Cierre de sala (Ecuador) *
                </Label>
                <Input
                  id="config-cierra"
                  type="datetime-local"
                  value={configForm.cierraEn}
                  onChange={(e) =>
                    setConfigForm((prev) => ({
                      ...prev,
                      cierraEn: e.target.value,
                    }))
                  }
                />
              </div>
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="config-redondeo">Método de redondeo</Label>
              <Select
                value={configForm.redondeo}
                onValueChange={(val: "inferior" | "mas_cercano") =>
                  setConfigForm((prev) => ({ ...prev, redondeo: val }))
                }
              >
                <SelectTrigger id="config-redondeo">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="mas_cercano">
                    Más cercano (0.5 redondea hacia arriba)
                  </SelectItem>
                  <SelectItem value="inferior">Inferior (Truncado)</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <Alert className="border-primary/20 bg-primary/5 text-xs">
              <ClockIcon className="size-4 text-primary" />
              <AlertDescription>
                Se aplica la política de <strong>un intento oficial</strong> y
                permutación criptográfica única de reactivos. Una vez aprobada,
                la configuración es inmutable.
              </AlertDescription>
            </Alert>
          </div>

          <DialogFooter className="gap-2 sm:gap-0">
            <Button
              variant="outline"
              onClick={() => setBancoIdConfigurar(null)}
            >
              Cancelar
            </Button>
            <Button
              onClick={handleGuardarConfiguracion}
              disabled={mutacionConfigurar.isPending}
            >
              {mutacionConfigurar.isPending
                ? "Aprobando…"
                : "Aprobar y congelar"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
