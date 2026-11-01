import { useState } from "react"
import {
  AlertCircleIcon,
  CalculatorIcon,
  CheckCircle2Icon,
  FileCheckIcon,
  SendIcon,
  ShieldAlertIcon,
  ShieldCheckIcon,
  XCircleIcon,
} from "lucide-react"

import { EstadoCargando, EstadoError } from "@/components/estados"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
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
import { Textarea } from "@/components/ui/textarea"
import { PREGUNTAS_INHABILIDADES } from "@/features/formulario/constantes"
import {
  useCalcularMeritos,
  useCrearRevisionMeritos,
  useExpedienteMeritosAdmin,
  usePublicarMeritos,
} from "./queries"
import {
  centipuntosAPuntos,
  type RevisionDocumento,
  type RevisionInhabilidad,
  type RevisionRequisito,
} from "@/lib/clientes/meritos"
import { useSesion } from "@/features/auth/sesion-context"

export interface EvaluacionMeritosProps {
  submissionId: string
  convocatoriaId?: string
}

export function EvaluacionMeritos({
  submissionId,
  convocatoriaId,
}: EvaluacionMeritosProps) {
  const { postulante } = useSesion()
  const esAdmin = postulante?.rol === "administrador"

  const {
    data: expedienteMeritos,
    isLoading,
    error,
    refetch,
  } = useExpedienteMeritosAdmin(submissionId)

  const crearRevisionMutation = useCrearRevisionMeritos(submissionId)
  const calcularMutation = useCalcularMeritos(submissionId)
  const publicarMutation = usePublicarMeritos(submissionId)

  // Estados locales para la revisión
  const [motivoRevision, setMotivoRevision] = useState("")
  const [inhabilidadesEstado, setInhabilidadesEstado] = useState<
    Record<string, { verificada: boolean; observacion: string }>
  >({})
  const [requisitosEstado, setRequisitosEstado] = useState<
    Record<string, { cumple: boolean; observacion: string }>
  >({})

  // Estados locales para cálculo y publicación
  const [motivoCalculo, setMotivoCalculo] = useState(
    "Cálculo formal de méritos según reglas vigentes",
  )
  const [decisionPublicacion, setDecisionPublicacion] = useState<
    "avanza" | "excluido"
  >("avanza")
  const [motivoPublicacion, setMotivoPublicacion] = useState("")

  if (isLoading) {
    return <EstadoCargando mensaje="Cargando expediente de méritos…" />
  }

  if (error) {
    return (
      <EstadoError
        error={error}
        titulo="Error al cargar méritos"
        onReintentar={() => void refetch()}
      />
    )
  }

  const reglas = expedienteMeritos?.reglasActivas
  const ultimaRevision = expedienteMeritos?.revision
  const ultimoCalculo = expedienteMeritos?.calculo
  const ultimaPublicacion = expedienteMeritos?.publicacion
  const evaluacion = ultimoCalculo?.evaluacion || ultimaPublicacion?.evaluacion

  if (!reglas) {
    return (
      <Card className="border-amber-500/40 bg-amber-50/20 dark:bg-amber-950/10">
        <CardHeader>
          <div className="flex items-center gap-2">
            <ShieldAlertIcon className="size-5 text-amber-600" />
            <CardTitle className="text-base text-amber-900 dark:text-amber-100">
              Matriz de reglas de méritos pendiente
            </CardTitle>
          </div>
          <CardDescription>
            No existe una matriz de méritos aprobada para esta convocatoria y
            sector. Un administrador debe aprobar la matriz antes de registrar
            revisiones o cálculos.
          </CardDescription>
        </CardHeader>
      </Card>
    )
  }

  // Inicializar o leer inhabilidades
  const preguntasInhabilidades = PREGUNTAS_INHABILIDADES

  function handleGuardarRevision() {
    if (!reglas) return

    const documentosPayload: RevisionDocumento[] = []

    const requisitosPayload: RevisionRequisito[] = (
      reglas.definicion.requisitos || []
    ).map((req) => {
      const estado = requisitosEstado[req.codigo] || {
        cumple: true,
        observacion: "",
      }
      return {
        codigo: req.codigo,
        cumple: estado.cumple,
        observacion: estado.observacion,
      }
    })

    const inhabilidadesPayload: RevisionInhabilidad[] =
      preguntasInhabilidades.map((p) => {
        const estado = inhabilidadesEstado[p.id] || {
          verificada: true,
          observacion: "",
        }
        return {
          pregunta: p.id,
          verificada: estado.verificada,
          observacion: estado.observacion,
        }
      })

    crearRevisionMutation.mutate({
      versionAnterior: ultimaRevision?.version ?? 0,
      reglasId: reglas.id,
      documentos: documentosPayload,
      requisitos: requisitosPayload,
      inhabilidades: inhabilidadesPayload,
      motivo:
        motivoRevision || "Revisión técnica de requisitos e inhabilidades",
    })
  }

  function handleCalcular() {
    if (!ultimaRevision) return
    calcularMutation.mutate({
      revisionId: ultimaRevision.id,
      versionAnterior: ultimoCalculo?.version ?? 0,
      motivo: motivoCalculo || "Cálculo oficial de méritos",
    })
  }

  function handlePublicar() {
    if (!ultimoCalculo) return
    publicarMutation.mutate({
      calculoId: ultimoCalculo.id,
      versionAnterior: ultimaPublicacion?.version ?? 0,
      decision: decisionPublicacion,
      motivo: motivoPublicacion,
    })
  }

  return (
    <div className="space-y-6">
      {/* 1. Header de Estado de Méritos */}
      <Card>
        <CardHeader className="pb-3">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div className="flex items-center gap-2">
              <FileCheckIcon className="size-5 text-primary" />
              <CardTitle className="text-lg">
                Revisión y Calificación de Méritos (FT19)
              </CardTitle>
            </div>
            <div className="flex items-center gap-2">
              <Badge variant="outline" className="font-mono text-xs">
                Reglas v{reglas.version}
              </Badge>
              {ultimaPublicacion ? (
                <Badge
                  variant={
                    ultimaPublicacion.decision === "avanza"
                      ? "default"
                      : "destructive"
                  }
                  className="capitalize"
                >
                  Resolución: {ultimaPublicacion.decision}
                </Badge>
              ) : ultimoCalculo ? (
                <Badge variant="secondary">
                  Calculado · Pendiente publicación
                </Badge>
              ) : ultimaRevision ? (
                <Badge variant="secondary">Revisado · Pendiente cálculo</Badge>
              ) : (
                <Badge variant="outline">Pendiente de revisión</Badge>
              )}
            </div>
          </div>
          <CardDescription>
            {convocatoriaId && `Convocatoria: ${convocatoriaId} · `}Sector:{" "}
            <strong className="capitalize">{reglas.sector}</strong> · Matriz
            oficial ref. {reglas.referenciaAprobacion || "Aprobada"} · Umbral
            mínimo:{" "}
            <strong>
              {reglas.definicion.umbralCentipuntos
                ? centipuntosAPuntos(reglas.definicion.umbralCentipuntos)
                : "—"}{" "}
              pts
            </strong>
          </CardDescription>
        </CardHeader>

        {ultimaPublicacion && (
          <CardContent className="pt-0">
            <Alert
              className={
                ultimaPublicacion.decision === "avanza"
                  ? "border-emerald-500/40 bg-emerald-50/50 dark:bg-emerald-950/20"
                  : "border-destructive/40 bg-destructive/10"
              }
            >
              {ultimaPublicacion.decision === "avanza" ? (
                <CheckCircle2Icon className="size-4 text-emerald-600" />
              ) : (
                <XCircleIcon className="size-4 text-destructive" />
              )}
              <AlertTitle className="font-semibold text-sm">
                Resolución publicada: Postulante{" "}
                {ultimaPublicacion.decision === "avanza"
                  ? "AVANZA a fase técnica"
                  : "EXCLUIDO del concurso"}
              </AlertTitle>
              <AlertDescription className="text-xs space-y-1 mt-1">
                <p>
                  <strong>Motivo:</strong> {ultimaPublicacion.motivo}
                </p>
                <p className="text-muted-foreground text-[11px]">
                  Publicado el{" "}
                  {new Date(ultimaPublicacion.publicadaEn).toLocaleString(
                    "es-EC",
                  )}{" "}
                  · Versión {ultimaPublicacion.version}
                </p>
              </AlertDescription>
            </Alert>
          </CardContent>
        )}
      </Card>

      {/* 2. Sección de Revisión Técnica: Inhabilidades y Requisitos */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="text-base">
                1. Verificación de Inhabilidades y Requisitos
              </CardTitle>
              <CardDescription>
                Verifique las 13 declaraciones de inhabilidades y los requisitos
                institucionales.
              </CardDescription>
            </div>
            {ultimaRevision && (
              <Badge variant="outline" className="text-xs">
                Última revisión v{ultimaRevision.version}
              </Badge>
            )}
          </div>
        </CardHeader>
        <CardContent className="space-y-6">
          {/* Checklist de las 13 inhabilidades */}
          <div className="space-y-3">
            <h4 className="text-sm font-semibold text-foreground">
              Declaración de Inhabilidades (13 declaraciones)
            </h4>
            <div className="rounded-lg border divide-y text-xs">
              {preguntasInhabilidades.map((p) => {
                const estado = inhabilidadesEstado[p.id] ?? {
                  verificada: true,
                  observacion: "",
                }
                return (
                  <div key={p.id} className="p-3 space-y-2">
                    <div className="flex items-start justify-between gap-4">
                      <div className="space-y-0.5">
                        <span className="font-semibold text-primary">
                          {p.id.toUpperCase()}:
                        </span>{" "}
                        <span className="text-foreground">{p.texto}</span>
                        {"requiereSi" in p && p.requiereSi && (
                          <Badge
                            variant="secondary"
                            className="ml-2 text-[10px]"
                          >
                            Sentido afirmativo (requiere Sí)
                          </Badge>
                        )}
                      </div>
                      <div className="flex items-center gap-2 shrink-0">
                        <Label
                          htmlFor={`verif-${p.id}`}
                          className="text-xs text-muted-foreground"
                        >
                          {estado.verificada
                            ? "Cumple / Verificada"
                            : "Observada / Invalida"}
                        </Label>
                        <Select
                          value={estado.verificada ? "si" : "no"}
                          onValueChange={(val) =>
                            setInhabilidadesEstado((prev) => ({
                              ...prev,
                              [p.id]: {
                                verificada: val === "si",
                                observacion: prev[p.id]?.observacion ?? "",
                              },
                            }))
                          }
                        >
                          <SelectTrigger className="w-28 h-8 text-xs">
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="si">Cumple (Sí)</SelectItem>
                            <SelectItem value="no">Observar (No)</SelectItem>
                          </SelectContent>
                        </Select>
                      </div>
                    </div>
                    {!estado.verificada && (
                      <Input
                        placeholder="Ingrese la observación o impedimento detectado..."
                        value={estado.observacion}
                        onChange={(e) =>
                          setInhabilidadesEstado((prev) => ({
                            ...prev,
                            [p.id]: {
                              verificada: false,
                              observacion: e.target.value,
                            },
                          }))
                        }
                        className="text-xs h-8"
                      />
                    )}
                  </div>
                )
              })}
            </div>
          </div>

          {/* Requisitos aprobados */}
          {reglas.definicion.requisitos &&
            reglas.definicion.requisitos.length > 0 && (
              <div className="space-y-3">
                <h4 className="text-sm font-semibold text-foreground">
                  Requisitos Habilitantes Aprobados
                </h4>
                <div className="rounded-lg border divide-y text-xs">
                  {reglas.definicion.requisitos.map((req) => {
                    const estado = requisitosEstado[req.codigo] ?? {
                      cumple: true,
                      observacion: "",
                    }
                    return (
                      <div
                        key={req.codigo}
                        className="p-3 flex items-center justify-between gap-4"
                      >
                        <span className="font-medium text-foreground">
                          {req.nombre}
                        </span>
                        <div className="flex items-center gap-2 shrink-0">
                          <Select
                            value={estado.cumple ? "si" : "no"}
                            onValueChange={(val) =>
                              setRequisitosEstado((prev) => ({
                                ...prev,
                                [req.codigo]: {
                                  cumple: val === "si",
                                  observacion:
                                    prev[req.codigo]?.observacion ?? "",
                                },
                              }))
                            }
                          >
                            <SelectTrigger className="w-28 h-8 text-xs">
                              <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectItem value="si">Cumple</SelectItem>
                              <SelectItem value="no">No cumple</SelectItem>
                            </SelectContent>
                          </Select>
                        </div>
                      </div>
                    )
                  })}
                </div>
              </div>
            )}

          {/* Motivo de la revisión */}
          <div className="space-y-1.5">
            <Label htmlFor="motivo-rev" className="text-xs">
              Motivo o fundamento de la revisión técnica *
            </Label>
            <Textarea
              id="motivo-rev"
              placeholder="Fundamento técnico de la verificación de requisitos y documentos..."
              value={motivoRevision}
              onChange={(e) => setMotivoRevision(e.target.value)}
              rows={2}
              className="text-xs"
            />
          </div>
        </CardContent>
        <CardFooter className="border-t bg-muted/20 px-6 py-3 flex justify-end">
          <Button
            type="button"
            size="sm"
            onClick={handleGuardarRevision}
            disabled={crearRevisionMutation.isPending}
            className="flex items-center gap-1.5"
          >
            <ShieldCheckIcon className="size-4" />
            <span>
              {crearRevisionMutation.isPending
                ? "Guardando revisión…"
                : "Guardar revisión técnica"}
            </span>
          </Button>
        </CardFooter>
      </Card>

      {/* 3. Cálculo de Méritos del Motor */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div className="space-y-0.5">
              <CardTitle className="text-base flex items-center gap-2">
                <CalculatorIcon className="size-4 text-primary" />
                <span>2. Cálculo Oficial de Méritos (Motor meritos-v1)</span>
              </CardTitle>
              <CardDescription>
                Aplica la fórmula matemática exacta del servidor sobre los
                respaldos validados.
              </CardDescription>
            </div>
            {ultimoCalculo && (
              <Badge variant="outline" className="text-xs font-mono">
                {ultimoCalculo.versionMotor} · v{ultimoCalculo.version}
              </Badge>
            )}
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {evaluacion ? (
            <div className="space-y-4">
              {/* Resumen numérico */}
              <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 text-center">
                <div className="rounded-lg border bg-card p-3">
                  <span className="text-xs text-muted-foreground">
                    Puntaje Obtenido
                  </span>
                  <p className="text-xl font-bold text-foreground">
                    {centipuntosAPuntos(evaluacion.totalCentipuntos)}
                  </p>
                  <span className="text-[10px] text-muted-foreground">
                    puntos
                  </span>
                </div>
                <div className="rounded-lg border bg-card p-3">
                  <span className="text-xs text-muted-foreground">
                    Umbral Exigido
                  </span>
                  <p className="text-xl font-bold text-foreground">
                    {centipuntosAPuntos(evaluacion.umbralCentipuntos)}
                  </p>
                  <span className="text-[10px] text-muted-foreground">
                    mínimo
                  </span>
                </div>
                <div className="rounded-lg border bg-card p-3">
                  <span className="text-xs text-muted-foreground">
                    Puntaje Máximo
                  </span>
                  <p className="text-xl font-bold text-foreground">
                    {centipuntosAPuntos(evaluacion.maximoCentipuntos)}
                  </p>
                  <span className="text-[10px] text-muted-foreground">
                    posible
                  </span>
                </div>
                <div className="rounded-lg border bg-card p-3">
                  <span className="text-xs text-muted-foreground">
                    Aptitud de Avance
                  </span>
                  <p className="mt-1">
                    <Badge
                      variant={
                        evaluacion.puedeAvanzar ? "default" : "destructive"
                      }
                      className="text-xs"
                    >
                      {evaluacion.puedeAvanzar ? "Habilitado" : "No Habilitado"}
                    </Badge>
                  </p>
                  <span className="text-[10px] text-muted-foreground">
                    para fase técnica
                  </span>
                </div>
              </div>

              {/* Impedimentos */}
              {evaluacion.impedimentos &&
                evaluacion.impedimentos.length > 0 && (
                  <Alert variant="destructive">
                    <AlertCircleIcon className="size-4" />
                    <AlertTitle className="text-xs font-semibold">
                      Impedimentos detectados por el motor
                    </AlertTitle>
                    <AlertDescription className="text-xs mt-1">
                      <ul className="list-inside list-disc space-y-0.5">
                        {evaluacion.impedimentos.map((imp, idx) => (
                          <li key={idx}>{imp}</li>
                        ))}
                      </ul>
                    </AlertDescription>
                  </Alert>
                )}

              {/* Desglose por criterio */}
              {evaluacion.desglose && evaluacion.desglose.length > 0 && (
                <div className="space-y-2">
                  <h5 className="text-xs font-semibold text-muted-foreground">
                    Desglose por Criterio Ponderado:
                  </h5>
                  <div className="rounded-lg border overflow-hidden text-xs">
                    <table className="w-full text-left">
                      <thead className="bg-muted/50 border-b">
                        <tr>
                          <th className="p-2.5 font-medium">Criterio</th>
                          <th className="p-2.5 font-medium text-center">
                            Unidades Acreditadas
                          </th>
                          <th className="p-2.5 font-medium text-right">
                            Puntos Ponderados
                          </th>
                        </tr>
                      </thead>
                      <tbody className="divide-y">
                        {evaluacion.desglose.map((d, i) => (
                          <tr key={i} className="hover:bg-muted/10">
                            <td className="p-2.5 font-medium">{d.criterio}</td>
                            <td className="p-2.5 text-center font-mono">
                              {d.unidadesVerificadas}
                            </td>
                            <td className="p-2.5 text-right font-mono font-semibold">
                              {centipuntosAPuntos(d.centipuntos)} pts
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </div>
              )}
            </div>
          ) : (
            <p className="text-xs text-muted-foreground">
              Aún no se ha ejecutado el cálculo oficial. Guarde la revisión
              técnica y pulse «Calcular méritos».
            </p>
          )}

          <div className="space-y-1.5 pt-2">
            <Label htmlFor="motivo-calc" className="text-xs">
              Motivo o justificación del cálculo
            </Label>
            <Input
              id="motivo-calc"
              value={motivoCalculo}
              onChange={(e) => setMotivoCalculo(e.target.value)}
              className="text-xs h-8"
              placeholder="Cálculo formal de méritos según reglas vigentes"
            />
          </div>
        </CardContent>
        <CardFooter className="border-t bg-muted/20 px-6 py-3 flex justify-between items-center">
          <span className="text-xs text-muted-foreground">
            {ultimaRevision
              ? "Revisión lista para cálculo"
              : "Requiere guardar revisión previa"}
          </span>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={handleCalcular}
            disabled={!ultimaRevision || calcularMutation.isPending}
            className="flex items-center gap-1.5"
          >
            <CalculatorIcon className="size-4" />
            <span>
              {calcularMutation.isPending ? "Calculando…" : "Calcular méritos"}
            </span>
          </Button>
        </CardFooter>
      </Card>

      {/* 4. Publicación y Resolución Oficial (Solo Administrador) */}
      {esAdmin && (
        <Card className="border-primary/30">
          <CardHeader>
            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <CardTitle className="text-base flex items-center gap-2">
                  <SendIcon className="size-4 text-primary" />
                  <span>
                    3. Publicación de Resolución Oficial (Administrador)
                  </span>
                </CardTitle>
                <CardDescription>
                  Emite la resolución definitiva del postulante en la fase de
                  méritos.
                </CardDescription>
              </div>
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div className="space-y-1.5">
                <Label htmlFor="decision-pub" className="text-xs">
                  Decisión de la Comisión *
                </Label>
                <Select
                  value={decisionPublicacion}
                  onValueChange={(val) =>
                    setDecisionPublicacion(val as "avanza" | "excluido")
                  }
                >
                  <SelectTrigger id="decision-pub" className="text-xs">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem
                      value="avanza"
                      disabled={evaluacion && !evaluacion.puedeAvanzar}
                    >
                      Avanza a fase técnica{" "}
                      {evaluacion && !evaluacion.puedeAvanzar
                        ? "(Bloqueado por impedimentos)"
                        : ""}
                    </SelectItem>
                    <SelectItem value="excluido">
                      Excluido del concurso
                    </SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-1.5 sm:col-span-2">
                <Label htmlFor="motivo-pub" className="text-xs">
                  Motivación de la resolución * (requerida)
                </Label>
                <Textarea
                  id="motivo-pub"
                  placeholder="Fundamente la resolución adoptada por la Comisión Especial..."
                  value={motivoPublicacion}
                  onChange={(e) => setMotivoPublicacion(e.target.value)}
                  rows={3}
                  className="text-xs"
                />
              </div>
            </div>

            <p className="text-xs text-muted-foreground">
              <strong>Nota legal:</strong> La resolución publicada no se
              sobreescribe automáticamente. Sólo una resolución «Avanza»
              habilita al postulante para rendir el examen psicotécnico y
              teórico en la fase de oposición técnica.
            </p>
          </CardContent>
          <CardFooter className="border-t bg-muted/20 px-6 py-3 flex justify-end">
            <Button
              type="button"
              size="sm"
              onClick={handlePublicar}
              disabled={
                !ultimoCalculo ||
                !motivoPublicacion ||
                publicarMutation.isPending
              }
              className="flex items-center gap-1.5 bg-primary text-primary-foreground"
            >
              <FileCheckIcon className="size-4" />
              <span>
                {publicarMutation.isPending
                  ? "Publicando…"
                  : "Publicar resolución definitiva"}
              </span>
            </Button>
          </CardFooter>
        </Card>
      )}
    </div>
  )
}
