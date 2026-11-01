import { Link } from "react-router"
import {
  BrainIcon,
  CheckCircle2Icon,
  ClockIcon,
  FileCheckIcon,
  MailIcon,
  ScaleIcon,
  XCircleIcon,
} from "lucide-react"

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
import { useEvaluacionPostulante } from "@/features/evaluacion/queries"
import { useNotificacionesPostulante } from "@/features/notificaciones/queries"
import { useListarPostulaciones } from "@/features/postulaciones/queries"
import { useReconsideracionesPostulante } from "@/features/reconsideracion/queries"
import { useResultadoMeritosPostulante } from "@/features/revision/queries"
import { centipuntosAPuntos } from "@/lib/clientes/meritos"
import { RUTAS } from "@/lib/rutas"

/**
 * Seguimiento de fases, resultados y notificaciones del postulante (FT19/FT20/FT21/FT22).
 */
export function SeguimientoView() {
  const { data: postulacionesData } = useListarPostulaciones({ limit: 1 })
  const postulacion = postulacionesData?.items?.[0]
  const submissionId = postulacion?.id ?? ""

  // Consultas de fases y calificaciones oficiales
  const { data: meritosData } = useResultadoMeritosPostulante(submissionId)
  const { data: evaluacionData } = useEvaluacionPostulante(submissionId)
  const { data: reconsideracionesData } =
    useReconsideracionesPostulante(submissionId)
  const { data: notificacionesData } = useNotificacionesPostulante(submissionId)

  const resultadoMeritos = meritosData?.resultado
  const meritosPublicado = meritosData?.publicado && Boolean(resultadoMeritos)

  const resultadosPublicados = evaluacionData?.expediente.resultados ?? {}
  const psicotecnicaPublicada = resultadosPublicados["oposicion_psicotecnica"]
  const teoricaPublicada = resultadosPublicados["oposicion_teorica"]
  const consolidadoPublicado = resultadosPublicados["resultados"]

  const listaReconsideraciones =
    reconsideracionesData?.pages.flatMap((p) => p.items) ?? []
  const impugnacionesPorFase = new Map(
    listaReconsideraciones.map((r) => [r.resultadoOriginal.fase, r]),
  )

  return (
    <div className="mx-auto flex w-full max-w-5xl flex-1 flex-col gap-6 p-4 sm:p-6">
      <Card>
        <CardHeader>
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div className="flex items-center gap-2">
              <FileCheckIcon className="size-5 text-primary" />
              <CardTitle>
                Seguimiento de Fases, Oposición y Resultados
              </CardTitle>
            </div>
            {postulacion && (
              <Badge variant="outline" className="font-mono text-xs">
                Expediente: {postulacion.code}
              </Badge>
            )}
          </div>
          <CardDescription>
            Consulte las resoluciones oficiales, sus calificaciones en cada
            etapa y acceda a la presentación de reconsideraciones.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          {/* Fase 1: Postulación y Sellado */}
          <div className="rounded-lg border p-4 space-y-2">
            <div className="flex items-center justify-between">
              <span className="font-semibold text-sm">
                1. Postulación y Entrega
              </span>
              {postulacion?.status === "submitted" ? (
                <Badge variant="default">Sellada y Enviada</Badge>
              ) : postulacion?.status === "closed" ? (
                <Badge variant="destructive">Cerrada por Elegibilidad</Badge>
              ) : (
                <Badge variant="secondary">Borrador en Edición</Badge>
              )}
            </div>
            <p className="text-xs text-muted-foreground">
              {postulacion?.status === "submitted"
                ? `Expediente sellado el ${new Date(postulacion.submittedAt || postulacion.createdAt).toLocaleDateString("es-EC")} con firma criptográfica SHA-256.`
                : "Complete y selle su postulación para acceder a la calificación de méritos."}
            </p>
          </div>

          {/* Fase 2: Calificación de Méritos (FT19) */}
          <div className="rounded-lg border p-4 space-y-4">
            <div className="flex items-center justify-between">
              <span className="font-semibold text-sm">
                2. Calificación de Méritos
              </span>
              {meritosPublicado ? (
                <Badge
                  variant={
                    resultadoMeritos?.decision === "avanza"
                      ? "default"
                      : "destructive"
                  }
                  className="capitalize"
                >
                  Resolución: {resultadoMeritos?.decision}
                </Badge>
              ) : (
                <Badge variant="outline" className="flex items-center gap-1">
                  <ClockIcon className="size-3" />
                  <span>En Calificación</span>
                </Badge>
              )}
            </div>

            {meritosPublicado ? (
              <div className="space-y-3">
                <Alert
                  className={
                    resultadoMeritos?.decision === "avanza"
                      ? "border-emerald-500/40 bg-emerald-50/50 dark:bg-emerald-950/20"
                      : "border-destructive/40 bg-destructive/10"
                  }
                >
                  {resultadoMeritos?.decision === "avanza" ? (
                    <CheckCircle2Icon className="size-4 text-emerald-600" />
                  ) : (
                    <XCircleIcon className="size-4 text-destructive" />
                  )}
                  <AlertTitle className="text-sm font-semibold">
                    {resultadoMeritos?.decision === "avanza"
                      ? "Postulación Habilitada para Oposición Técnica"
                      : "Postulación No Habilitada"}
                  </AlertTitle>
                  <AlertDescription className="text-xs space-y-2 mt-1">
                    <p>
                      <strong>Puntaje oficial de méritos:</strong>{" "}
                      <span className="font-semibold">
                        {centipuntosAPuntos(
                          resultadoMeritos?.evaluacion.totalCentipuntos ?? 0,
                        )}{" "}
                        puntos
                      </span>{" "}
                      (de un máximo de{" "}
                      {centipuntosAPuntos(
                        resultadoMeritos?.evaluacion.maximoCentipuntos ?? 0,
                      )}{" "}
                      pts).
                    </p>
                    <p>
                      <strong>Motivo de la resolución:</strong>{" "}
                      {resultadoMeritos?.motivo}
                    </p>
                  </AlertDescription>
                </Alert>

                <div className="flex items-center justify-between pt-1 text-xs">
                  {impugnacionesPorFase.has("meritos") ? (
                    <span className="text-muted-foreground flex items-center gap-1">
                      <ScaleIcon className="size-3 text-primary" />
                      Reconsideración interpuesta (Estado:{" "}
                      <strong>
                        {impugnacionesPorFase.get("meritos")?.estado}
                      </strong>
                      )
                    </span>
                  ) : (
                    <Button asChild variant="outline" size="sm">
                      <Link to={`${RUTAS.reconsideracion}?fase=meritos`}>
                        <ScaleIcon className="size-3 mr-1" />
                        Presentar Reconsideración sobre Méritos
                      </Link>
                    </Button>
                  )}
                </div>
              </div>
            ) : (
              <p className="text-xs text-muted-foreground">
                La Comisión Especial se encuentra verificando los documentos de
                respaldo y requisitos habilitantes. Una vez publicada la
                resolución oficial, podrá consultar aquí su puntaje desglosado.
              </p>
            )}
          </div>

          {/* Fase 3: Oposición Psicotécnica y Teórica (FT20) */}
          <div className="rounded-lg border p-4 space-y-4">
            <div className="flex items-center justify-between">
              <span className="font-semibold text-sm">
                3. Oposición Psicotécnica y Pruebas Teóricas
              </span>
              {psicotecnicaPublicada ? (
                <Badge
                  variant={
                    psicotecnicaPublicada.habilitaAvance
                      ? "default"
                      : "destructive"
                  }
                >
                  Psicotécnica:{" "}
                  {psicotecnicaPublicada.habilitaAvance
                    ? "Habilitada"
                    : "No Habilitada"}
                </Badge>
              ) : (
                <Badge variant="outline" className="flex items-center gap-1">
                  <ClockIcon className="size-3" />
                  <span>En Desarrollo</span>
                </Badge>
              )}
            </div>

            <div className="grid gap-3 sm:grid-cols-2">
              {/* Bloque Psicotécnico */}
              <div className="rounded border bg-muted/20 p-3 space-y-2 text-xs">
                <div className="flex items-center justify-between">
                  <span className="font-semibold flex items-center gap-1">
                    <BrainIcon className="size-3.5 text-primary" />
                    Evaluación Psicotécnica
                  </span>
                  {psicotecnicaPublicada ? (
                    <Badge variant="default" className="text-[10px]">
                      Oficial v{psicotecnicaPublicada.version}
                    </Badge>
                  ) : (
                    <Badge variant="secondary" className="text-[10px]">
                      Pendiente
                    </Badge>
                  )}
                </div>
                {psicotecnicaPublicada ? (
                  <div className="space-y-1">
                    <p>
                      <strong>Calificación oficial:</strong>{" "}
                      {psicotecnicaPublicada.puntaje !== null
                        ? `${psicotecnicaPublicada.puntaje} / ${psicotecnicaPublicada.maximo ?? "—"} pts`
                        : "Diagnóstico cualitativo acreditado"}
                    </p>
                    <p className="text-muted-foreground">
                      {psicotecnicaPublicada.motivo}
                    </p>
                    <div className="pt-2">
                      {impugnacionesPorFase.has("oposicion_psicotecnica") ? (
                        <span className="text-[11px] text-muted-foreground flex items-center gap-1">
                          <ScaleIcon className="size-3 text-primary" />
                          Reconsideración interpuesta (Estado:{" "}
                          <strong>
                            {
                              impugnacionesPorFase.get("oposicion_psicotecnica")
                                ?.estado
                            }
                          </strong>
                          )
                        </span>
                      ) : (
                        <Button asChild variant="outline" size="sm">
                          <Link
                            to={`${RUTAS.reconsideracion}?fase=oposicion_psicotecnica`}
                          >
                            <ScaleIcon className="size-3 mr-1" />
                            Reconsiderar Psicotécnica
                          </Link>
                        </Button>
                      )}
                    </div>
                  </div>
                ) : (
                  <p className="text-muted-foreground">
                    Los resultados de su informe psicométrico se publicarán una
                    vez aprobados por la Comisión.
                  </p>
                )}
              </div>

              {/* Bloque Teórico */}
              <div className="rounded border bg-muted/20 p-3 space-y-2 text-xs">
                <div className="flex items-center justify-between">
                  <span className="font-semibold flex items-center gap-1">
                    <FileCheckIcon className="size-3.5 text-primary" />
                    Examen Teórico de Conocimientos
                  </span>
                  {teoricaPublicada ? (
                    <Badge variant="default" className="text-[10px]">
                      Oficial
                    </Badge>
                  ) : (
                    <Badge variant="outline" className="text-[10px]">
                      Pendiente
                    </Badge>
                  )}
                </div>
                {teoricaPublicada ? (
                  <div className="space-y-1">
                    <p>
                      <strong>Nota obtenida:</strong> {teoricaPublicada.puntaje}{" "}
                      / {teoricaPublicada.maximo} pts
                    </p>
                    <p className="text-muted-foreground">
                      {teoricaPublicada.motivo}
                    </p>
                  </div>
                ) : (
                  <p className="text-muted-foreground">
                    La rendición del examen de conocimientos se habilitará
                    conforme al calendario institucional.
                  </p>
                )}
              </div>
            </div>
          </div>

          {/* Fase 4: Consolidación Definitiva (FT20) */}
          <div className="rounded-lg border p-4 space-y-4">
            <div className="flex items-center justify-between">
              <span className="font-semibold text-sm">
                4. Resultados Consolidados Definitivos
              </span>
              {consolidadoPublicado ? (
                <Badge
                  variant={
                    consolidadoPublicado.habilitaAvance
                      ? "default"
                      : "destructive"
                  }
                >
                  {consolidadoPublicado.habilitaAvance
                    ? "Elegible / Aprobado"
                    : "No Elegible"}
                </Badge>
              ) : (
                <Badge variant="outline" className="flex items-center gap-1">
                  <ClockIcon className="size-3" />
                  <span>Pendiente de Consolidación</span>
                </Badge>
              )}
            </div>

            {consolidadoPublicado ? (
              <div className="space-y-3 text-xs">
                <div className="flex items-center justify-between rounded bg-muted/40 p-3">
                  <div>
                    <span className="text-muted-foreground">
                      Puntaje total ponderado:
                    </span>
                    <p className="text-xl font-black text-primary">
                      {consolidadoPublicado.puntaje} /{" "}
                      {consolidadoPublicado.maximo ?? 10000} pts
                    </p>
                  </div>
                  <Badge variant="outline">
                    Motor: {consolidadoPublicado.versionMotor}
                  </Badge>
                </div>

                {consolidadoPublicado.desglose && (
                  <div className="grid gap-2 sm:grid-cols-3">
                    {consolidadoPublicado.desglose.map((item) => (
                      <div
                        key={item.fase}
                        className="rounded border p-2 text-center"
                      >
                        <span className="font-semibold capitalize">
                          {item.fase.replace("_", " ")}
                        </span>
                        <p className="text-muted-foreground">
                          Aporte: {item.aporte} pts
                        </p>
                      </div>
                    ))}
                  </div>
                )}

                <div className="pt-2">
                  {impugnacionesPorFase.has("resultados") ? (
                    <span className="text-muted-foreground flex items-center gap-1">
                      <ScaleIcon className="size-3 text-primary" />
                      Reconsideración sobre resultados definitivos interpuesta
                    </span>
                  ) : (
                    <Button asChild variant="outline" size="sm">
                      <Link to={`${RUTAS.reconsideracion}?fase=resultados`}>
                        <ScaleIcon className="size-3 mr-1" />
                        Reconsiderar Resultado Consolidado
                      </Link>
                    </Button>
                  )}
                </div>
              </div>
            ) : (
              <p className="text-xs text-muted-foreground">
                Una vez concluidas las fases de méritos, psicotécnica y teórica,
                la Comisión publicará el acta consolidada con los aportes
                ponderados.
              </p>
            )}
          </div>

          {/* Fase 5: Notificaciones y Comunicaciones Oficiales (FT22) */}
          <div className="rounded-lg border p-4 space-y-3">
            <div className="flex items-center justify-between">
              <span className="font-semibold text-sm flex items-center gap-1.5">
                <MailIcon className="size-4 text-primary" />
                Comunicaciones y Notificaciones de Etapa
              </span>
              <Badge variant="outline">
                {notificacionesData?.pages.flatMap((p) => p.items).length ?? 0}{" "}
                Avisos
              </Badge>
            </div>
            <p className="text-xs text-muted-foreground">
              Avisos generados automáticamente para su expediente por cada hito
              del concurso. La confirmación SMTP acredita la recepción por el
              servidor de correo, no la apertura por el destinatario.
            </p>

            <div className="space-y-2 pt-1">
              {(notificacionesData?.pages.flatMap((p) => p.items) ?? [])
                .length === 0 ? (
                <p className="text-xs text-muted-foreground italic">
                  Aún no se han emitido comunicaciones oficiales por correo para
                  este expediente.
                </p>
              ) : (
                notificacionesData?.pages
                  .flatMap((p) => p.items)
                  .map((notif) => (
                    <div
                      key={notif.id}
                      className="flex flex-wrap items-center justify-between gap-2 rounded border bg-muted/20 p-2.5 text-xs"
                    >
                      <div>
                        <p className="font-semibold">{notif.accion}</p>
                        <p className="text-[11px] text-muted-foreground">
                          {new Date(notif.creadaEn).toLocaleDateString("es-EC")}{" "}
                          · Plantilla: {notif.plantilla}
                        </p>
                      </div>
                      <div>
                        {notif.estado === "smtp_accepted" ? (
                          <Badge
                            variant="default"
                            className="flex items-center gap-1 bg-emerald-600 text-[10px]"
                          >
                            <CheckCircle2Icon className="size-3" />
                            <span>Aceptado por SMTP</span>
                          </Badge>
                        ) : notif.estado === "failed" ? (
                          <Badge
                            variant="destructive"
                            className="flex items-center gap-1 text-[10px]"
                          >
                            <XCircleIcon className="size-3" />
                            <span>Fallo en entrega</span>
                          </Badge>
                        ) : (
                          <Badge
                            variant="secondary"
                            className="flex items-center gap-1 text-[10px]"
                          >
                            <ClockIcon className="size-3" />
                            <span>En cola</span>
                          </Badge>
                        )}
                      </div>
                    </div>
                  ))
              )}
            </div>
          </div>

          {/* Botones de navegación inferior */}
          <div className="flex flex-wrap gap-2 border-t pt-4">
            <Button asChild variant="outline" size="sm">
              <Link to={RUTAS.postulacion}>Volver a mi postulación</Link>
            </Button>
            <Button asChild variant="secondary" size="sm">
              <Link to={RUTAS.reconsideracion}>
                <ScaleIcon className="size-3.5 mr-1.5" />
                Bandeja de Reconsideraciones
              </Link>
            </Button>
            {postulacion?.status === "submitted" && (
              <Button asChild size="sm">
                <Link to={RUTAS.postulacionImprimir}>
                  Ver comprobante imprimible
                </Link>
              </Button>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
