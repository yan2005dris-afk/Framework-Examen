import {
  AlertCircleIcon,
  ArrowLeftIcon,
  CheckCircle2Icon,
  CheckIcon,
  ClockIcon,
  CopyIcon,
  Edit3Icon,
  PrinterIcon,
  SendIcon,
  ShieldCheckIcon,
} from "lucide-react"
import { useMemo, useState } from "react"
import { Link } from "react-router"
import { toast } from "sonner"

import { EstadoCargando, EstadoError } from "@/components/estados"
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion"
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
import {
  CONSENTIMIENTO_DEFAULT,
  PREGUNTAS_INHABILIDADES,
  type ConsentimientoEF12Data,
} from "@/features/formulario/constantes"
import { ConsentimientoEF12 } from "@/features/postulacion/consentimiento"
import {
  useListarPostulaciones,
  useSellarPostulacion,
} from "@/features/postulaciones/queries"
import { useDisponibilidadEF12 } from "@/features/convocatorias/use-disponibilidad"
import { usePoliticaVigente } from "@/features/privacidad/queries"
import { esErrorApi } from "@/lib/api-error"
import { RUTAS, rutaSeccion } from "@/lib/rutas"
import { calcularDuracionExperiencia } from "@/utils/validators"
import type {
  Seccion1Data,
  Seccion2Data,
  Seccion3Data,
  Seccion4Data,
  Seccion5Data,
  Seccion6Data,
  Seccion7Data,
  Seccion8Data,
  Seccion9Data,
} from "@/features/formulario"

/**
 * Vista de revisión y envío de la postulación EF-12 (HU-10).
 * Presenta el resumen de las 9 secciones, los 5 puntos de consentimiento
 * y realiza el sellado criptográfico inmutable.
 */
export function RevisionPostulacionView() {
  const {
    data: postulacionesData,
    isLoading,
    error,
    refetch,
  } = useListarPostulaciones({ limit: 1 })

  const sellarMutation = useSellarPostulacion()

  const postulacion = postulacionesData?.items?.[0]
  const yaEnviada = postulacion?.status !== "draft"
  const disponibilidad = useDisponibilidadEF12(postulacion?.convocatoriaId)
  const { data: politicaVigente } = usePoliticaVigente(
    postulacion?.convocatoriaId,
  )

  // Estado local para los cinco consentimientos si no están guardados previamente
  const [consentimiento, setConsentimiento] = useState<ConsentimientoEF12Data>(
    () => {
      const c = postulacion?.content?.consentimiento as
        ConsentimientoEF12Data | undefined
      return c || CONSENTIMIENTO_DEFAULT
    },
  )

  const [copiado, setCopiado] = useState(false)

  // Desglose de contenido por sección
  const content = postulacion?.content || {}
  const s1 = useMemo(
    () => (content.seccion1 || {}) as Partial<Seccion1Data>,
    [content.seccion1],
  )
  const s2 = useMemo(
    () => (content.seccion2 || {}) as Partial<Seccion2Data>,
    [content.seccion2],
  )
  const s3 = useMemo(
    () => (content.seccion3 || {}) as Partial<Seccion3Data>,
    [content.seccion3],
  )
  const s4 = useMemo(
    () => (content.seccion4 || {}) as Partial<Seccion4Data>,
    [content.seccion4],
  )
  const s5 = useMemo(
    () => (content.seccion5 || {}) as Partial<Seccion5Data>,
    [content.seccion5],
  )
  const s6 = useMemo(
    () => (content.seccion6 || {}) as Partial<Seccion6Data>,
    [content.seccion6],
  )
  const s7 = useMemo(
    () => (content.seccion7 || {}) as Partial<Seccion7Data>,
    [content.seccion7],
  )
  const s8 = useMemo(
    () => (content.seccion8 || {}) as Partial<Seccion8Data>,
    [content.seccion8],
  )
  const s9 = useMemo(
    () => (content.seccion9 || {}) as Partial<Seccion9Data>,
    [content.seccion9],
  )

  // Mapa de secciones completadas
  const mapaSecciones = useMemo(() => {
    const mapa: Record<number, boolean> = {}
    for (let i = 1; i <= 9; i++) mapa[i] = false
    if (postulacion?.progress?.sections) {
      for (const s of postulacion.progress.sections) {
        mapa[s.number] = s.complete
      }
    }
    return mapa
  }, [postulacion])

  const totalCompletadas = useMemo(() => {
    return Object.values(mapaSecciones).filter(Boolean).length
  }, [mapaSecciones])

  const formularioCompleto = totalCompletadas === 9

  // Comprobación de respaldos PDF pendientes (FT18 / Mockup 07b)
  const documentosPendientes = useMemo(() => {
    const pendientes: string[] = []
    if (
      s4.cuentaConRespaldoOrganizacion === "Sí" &&
      !s4.documentoRespaldoPdf?.id &&
      !s4.documentoRespaldoPdf?.referenciaId
    ) {
      pendientes.push("Respaldo de Organización o Gremio (Sección 4)")
    }
    if (s5.titulos) {
      for (const t of s5.titulos) {
        if (!t.documentoPdf?.id && !t.documentoPdf?.referenciaId) {
          pendientes.push(
            `Certificado de título «${t.titulo || t.nivel}» (Sección 5)`,
          )
        }
      }
    }
    if (s6.experiencias) {
      for (const e of s6.experiencias) {
        if (!e.documentoPdf?.id && !e.documentoPdf?.referenciaId) {
          pendientes.push(
            `Certificado laboral en «${e.institucion || e.cargo}» (Sección 6)`,
          )
        }
      }
    }
    if (s7.publicaciones && s7.cuentaConPublicaciones === "Sí") {
      for (const p of s7.publicaciones) {
        if (!p.documentoPdf?.id && !p.documentoPdf?.referenciaId) {
          pendientes.push(
            `Respaldo de publicación «${p.titulo || "sin título"}» (Sección 7)`,
          )
        }
      }
    }
    if (s8.capacitaciones) {
      for (const c of s8.capacitaciones) {
        if (!c.documentoPdf?.id && !c.documentoPdf?.referenciaId) {
          pendientes.push(
            `Certificado de capacitación «${c.nombre || "sin nombre"}» (Sección 8)`,
          )
        }
      }
    }
    return pendientes
  }, [s4, s5, s6, s7, s8])

  const todosConsentimientosAceptados =
    Boolean(consentimiento.aceptacionTratamientoDatos) &&
    Boolean(consentimiento.declaracionVeracidad) &&
    Boolean(consentimiento.aceptacionNormasConcurso) &&
    Boolean(consentimiento.autorizacionPublicacionNombre) &&
    Boolean(consentimiento.constanciaEntregaFisica)

  async function handleCopiarHash(hash: string) {
    try {
      await navigator.clipboard.writeText(hash)
      setCopiado(true)
      toast.success("Hash copiado al portapapeles")
      setTimeout(() => setCopiado(false), 2500)
    } catch {
      toast.error("No se pudo copiar el hash al portapapeles")
    }
  }

  function handleEnviarPostulacion() {
    if (!disponibilidad.abierta) {
      toast.error(
        "La recepción no está confirmada o el período cerró. Consulte la convocatoria.",
      )
      return
    }
    if (!postulacion) return
    if (!formularioCompleto) {
      toast.error(
        "Debe completar las 9 secciones del formulario antes de enviar.",
      )
      return
    }
    if (documentosPendientes.length > 0) {
      toast.error(
        `Faltan ${documentosPendientes.length} documentos obligatorios. Cargue todos los respaldos PDF antes de enviar.`,
      )
      return
    }
    if (!todosConsentimientosAceptados) {
      toast.error(
        "Debe aceptar los 5 puntos de la declaración legal para continuar.",
      )
      return
    }
    if (politicaVigente?.restringido) {
      toast.error(
        "La política de privacidad está temporalmente restringida. No es posible enviar postulaciones.",
      )
      return
    }

    const payloadConsentimiento: ConsentimientoEF12Data = {
      ...consentimiento,
      ...(politicaVigente
        ? {
            politicaPrivacidadId: politicaVigente.id,
            hashPoliticaPrivacidad: politicaVigente.hash,
            versionPolitica: politicaVigente.version,
            referenciaAprobacion:
              politicaVigente.definicion.referenciaAprobacion,
          }
        : {}),
    }

    sellarMutation.mutate(
      {
        id: postulacion.id,
        consentimiento: payloadConsentimiento,
      },
      {
        onSuccess: (postulacionSellada) => {
          toast.success(
            `¡Postulación sellada y enviada exitosamente! Código: ${postulacionSellada.code || "registrado"}.`,
          )
        },
        onError: (err: unknown) => {
          if (esErrorApi(err)) {
            if (err.codigo === "version_privacidad_conflicto") {
              toast.error(
                "La política de privacidad ha sido actualizada. Por favor revise la nueva versión y confirme su aceptación.",
              )
            } else if (err.codigo === "privacidad_pendiente") {
              toast.error(
                "No se pudo confirmar la política de privacidad activa. Verifique los términos institucionales.",
              )
            } else {
              toast.error(`Error al enviar la postulación: ${err.mensaje}`)
            }
          } else {
            toast.error("Error al enviar la postulación definitiva.")
          }
        },
      },
    )
  }

  if (isLoading) {
    return (
      <div className="mx-auto w-full max-w-4xl px-4 py-12">
        <EstadoCargando />
      </div>
    )
  }

  if (error || !postulacion) {
    return (
      <div className="mx-auto w-full max-w-4xl px-4 py-12">
        <EstadoError
          error={error}
          titulo="Error al cargar la postulación"
          mensaje="No se pudo obtener el expediente para revisión."
          onReintentar={() => {
            void refetch()
          }}
        />
      </div>
    )
  }

  return (
    <div className="mx-auto w-full max-w-4xl px-4 py-8">
      {/* Encabezado */}
      <div className="mb-6 flex flex-col gap-2 border-b pb-4">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-foreground">
              Revisión y Envío de la Postulación
            </h1>
            <p className="text-sm text-muted-foreground">
              Formulario EF-12 · Concurso de Méritos y Oposición BIESS / IESS
            </p>
          </div>

          <div className="flex items-center gap-2">
            <Button
              asChild
              variant="outline"
              size="sm"
              className="flex items-center gap-1.5"
            >
              <Link to={RUTAS.postulacion}>
                <ArrowLeftIcon className="size-4" aria-hidden="true" />
                <span>Volver al formulario</span>
              </Link>
            </Button>
            <Button
              asChild
              variant="outline"
              size="sm"
              className="flex items-center gap-1.5"
            >
              <Link to={RUTAS.postulacionImprimir}>
                <PrinterIcon className="size-4" aria-hidden="true" />
                <span>Imprimir (HU-11)</span>
              </Link>
            </Button>
          </div>
        </div>
      </div>

      {/* Banner de estado: Enviada vs Borrador */}
      {yaEnviada ? (
        <Card className="mb-8 border-emerald-500/40 bg-emerald-50/50 dark:bg-emerald-950/20">
          <CardHeader className="pb-3">
            <div className="flex items-start gap-3">
              <ShieldCheckIcon
                className="mt-0.5 size-6 shrink-0 text-emerald-600 dark:text-emerald-400"
                aria-hidden="true"
              />
              <div className="flex-1">
                <div className="flex flex-wrap items-center gap-2">
                  <CardTitle className="text-lg text-emerald-900 dark:text-emerald-100">
                    Postulación Sellada y Enviada
                  </CardTitle>
                  <Badge
                    variant="outline"
                    className="border-emerald-600 text-emerald-700 dark:text-emerald-300"
                  >
                    {postulacion.code || "REGISTRADA"}
                  </Badge>
                </div>
                <CardDescription className="mt-1 text-emerald-800 dark:text-emerald-200">
                  El formulario ha sido sellado con hash criptográfico
                  inmutable. El contenido se encuentra bloqueado para
                  modificaciones.
                </CardDescription>
              </div>
            </div>
          </CardHeader>
          <CardContent className="flex flex-col gap-3 text-xs">
            {postulacion.hash && (
              <div className="flex flex-col gap-1 rounded-md border border-emerald-500/30 bg-white p-3 dark:bg-card">
                <span className="font-semibold text-muted-foreground">
                  Firma digital / Hash SHA-256 de la postulación:
                </span>
                <div className="flex items-center justify-between gap-2 overflow-hidden">
                  <code className="font-mono text-xs text-foreground break-all">
                    {postulacion.hash}
                  </code>
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={() => {
                      void handleCopiarHash(postulacion.hash || "")
                    }}
                    className="h-7 shrink-0 px-2"
                  >
                    {copiado ? (
                      <CheckIcon className="size-3.5 text-emerald-600" />
                    ) : (
                      <CopyIcon className="size-3.5" />
                    )}
                    <span className="sr-only">Copiar hash</span>
                  </Button>
                </div>
              </div>
            )}

            <div className="flex flex-wrap items-center justify-between gap-2 pt-1 text-muted-foreground">
              <span>
                Fecha de envío:{" "}
                <strong className="text-foreground">
                  {postulacion.submittedAt
                    ? new Date(postulacion.submittedAt).toLocaleString("es-EC")
                    : "Confirmado"}
                </strong>
              </span>
              <span>
                Notificación por correo:{" "}
                <strong className="text-foreground">
                  Enviada al correo registrado
                </strong>
              </span>
            </div>
          </CardContent>
          <CardFooter className="pt-0">
            <Button
              asChild
              className="flex items-center gap-2 bg-emerald-700 text-white hover:bg-emerald-800"
            >
              <Link to={RUTAS.postulacionImprimir}>
                <PrinterIcon className="size-4" aria-hidden="true" />
                <span>Imprimir Formulario con Firma Manuscrita (HU-11)</span>
              </Link>
            </Button>
          </CardFooter>
        </Card>
      ) : (
        <div className="mb-8">
          {formularioCompleto ? (
            <div
              role="alert"
              className="flex items-start gap-3 rounded-lg border border-emerald-500/40 bg-emerald-50/50 p-4 text-emerald-900 dark:bg-emerald-950/20 dark:text-emerald-100"
            >
              <CheckCircle2Icon
                className="mt-0.5 size-5 shrink-0 text-emerald-600 dark:text-emerald-400"
                aria-hidden="true"
              />
              <div>
                <p className="font-semibold text-sm">
                  Formulario completo (9 de 9 secciones verificadas)
                </p>
                <p className="mt-0.5 text-xs text-emerald-800 dark:text-emerald-200">
                  Revise los datos a continuación, acepte los cinco puntos de
                  consentimiento legal al final de la página y proceda con el
                  envío definitivo.
                </p>
              </div>
            </div>
          ) : (
            <div
              role="alert"
              className="flex items-start gap-3 rounded-lg border border-amber-500/40 bg-amber-50/50 p-4 text-amber-900 dark:bg-amber-950/20 dark:text-amber-100"
            >
              <ClockIcon
                className="mt-0.5 size-5 shrink-0 text-amber-600 dark:text-amber-400"
                aria-hidden="true"
              />
              <div className="flex-1">
                <p className="font-semibold text-sm">
                  Formulario incompleto: {totalCompletadas} de 9 secciones
                  completadas
                </p>
                <p className="mt-0.5 text-xs text-amber-800 dark:text-amber-200">
                  Debe completar todas las secciones antes de que se habilite el
                  botón de envío definitivo. Las secciones pendientes están
                  marcadas abajo.
                </p>
              </div>
            </div>
          )}

          {/* Bloqueo de envío por documentos pendientes (Mockup 07b) */}
          {documentosPendientes.length > 0 && (
            <div
              role="alert"
              className="mt-4 flex items-start gap-3 rounded-lg border border-destructive/40 bg-destructive/10 p-4 text-foreground dark:bg-destructive/20"
            >
              <AlertCircleIcon
                className="mt-0.5 size-5 shrink-0 text-destructive"
                aria-hidden="true"
              />
              <div className="flex-1">
                <div className="flex flex-wrap items-center gap-2">
                  <p className="font-semibold text-sm text-destructive">
                    Envío bloqueado por documentos pendientes (
                    {documentosPendientes.length})
                  </p>
                  <Badge variant="destructive" className="text-[10px]">
                    Requerido
                  </Badge>
                </div>
                <p className="mt-1 text-xs text-muted-foreground">
                  Faltan documentos obligatorios. Cargue los respaldos PDF
                  pendientes para poder enviar la postulación:
                </p>
                <ul className="mt-2 list-inside list-disc space-y-1 text-xs text-muted-foreground">
                  {documentosPendientes.map((d, i) => (
                    <li key={i} className="font-medium text-foreground">
                      {d}
                    </li>
                  ))}
                </ul>
              </div>
            </div>
          )}
        </div>
      )}

      {/* Resumen Secciones en Accordion */}
      <div className="mb-8">
        <h2 className="mb-3 font-semibold text-lg">
          Resumen detallado de las 9 secciones
        </h2>

        <Accordion
          type="multiple"
          defaultValue={["sec-1", "sec-4", "sec-9"]}
          className="flex flex-col gap-3"
        >
          {/* SECCIÓN 1: Personal */}
          <AccordionItem
            value="sec-1"
            className="rounded-lg border bg-card px-4"
          >
            <AccordionTrigger className="hover:no-underline">
              <div className="flex items-center gap-3">
                <span className="font-mono font-bold text-xs text-primary">
                  1
                </span>
                <span className="font-medium text-sm">
                  Información Personal
                </span>
                {mapaSecciones[1] ? (
                  <Badge
                    variant="outline"
                    className="border-emerald-500 text-2xs text-emerald-700"
                  >
                    Completa
                  </Badge>
                ) : (
                  <Badge variant="secondary" className="text-2xs">
                    Pendiente
                  </Badge>
                )}
              </div>
            </AccordionTrigger>
            <AccordionContent className="pt-2 pb-4 text-sm">
              <div className="grid grid-cols-1 gap-3 text-xs sm:grid-cols-2">
                <div>
                  <span className="text-muted-foreground">
                    Nombres completos:
                  </span>
                  <p className="font-medium text-foreground">
                    {s1.nombres || "—"} {s1.apellidos || "—"}
                  </p>
                </div>
                <div>
                  <span className="text-muted-foreground">Cédula:</span>
                  <p className="font-medium text-foreground">
                    {s1.cedula || "—"}
                  </p>
                </div>
                <div>
                  <span className="text-muted-foreground">Género:</span>
                  <p className="font-medium text-foreground">
                    {s1.genero || "—"}
                  </p>
                </div>
                <div>
                  <span className="text-muted-foreground">
                    Fecha de nacimiento:
                  </span>
                  <p className="font-medium text-foreground">
                    {s1.fechaNacimiento || "—"}
                  </p>
                </div>
                <div>
                  <span className="text-muted-foreground">Nacionalidad:</span>
                  <p className="font-medium text-foreground">
                    {s1.nacionalidad || "—"}
                  </p>
                </div>
                <div>
                  <span className="text-muted-foreground">Estado civil:</span>
                  <p className="font-medium text-foreground">
                    {s1.estadoCivil || "—"}
                  </p>
                </div>
                {(s1.estadoCivil === "Casado/a" ||
                  s1.estadoCivil === "Unión de hecho") && (
                  <div className="rounded border bg-muted/20 p-2 sm:col-span-2">
                    <span className="font-semibold text-muted-foreground">
                      Datos del cónyuge / pareja:
                    </span>
                    <p className="font-medium text-foreground">
                      {s1.nombreConyuge || "—"} (Cédula:{" "}
                      {s1.cedulaConyuge || "—"})
                    </p>
                  </div>
                )}
              </div>
              {!yaEnviada && (
                <div className="mt-4 flex justify-end">
                  <Button
                    asChild
                    variant="ghost"
                    size="sm"
                    className="h-7 text-xs"
                  >
                    <Link to={rutaSeccion(1)}>
                      <Edit3Icon className="mr-1 size-3" /> Editar sección
                    </Link>
                  </Button>
                </div>
              )}
            </AccordionContent>
          </AccordionItem>

          {/* SECCIÓN 2: Contacto */}
          <AccordionItem
            value="sec-2"
            className="rounded-lg border bg-card px-4"
          >
            <AccordionTrigger className="hover:no-underline">
              <div className="flex items-center gap-3">
                <span className="font-mono font-bold text-xs text-primary">
                  2
                </span>
                <span className="font-medium text-sm">
                  Residencia y Contacto
                </span>
                {mapaSecciones[2] ? (
                  <Badge
                    variant="outline"
                    className="border-emerald-500 text-2xs text-emerald-700"
                  >
                    Completa
                  </Badge>
                ) : (
                  <Badge variant="secondary" className="text-2xs">
                    Pendiente
                  </Badge>
                )}
              </div>
            </AccordionTrigger>
            <AccordionContent className="pt-2 pb-4 text-sm">
              <div className="grid grid-cols-1 gap-3 text-xs sm:grid-cols-2">
                <div>
                  <span className="text-muted-foreground">
                    País / Provincia / Cantón:
                  </span>
                  <p className="font-medium text-foreground">
                    {s2.pais || "—"} / {s2.provincia || "—"} /{" "}
                    {s2.canton || "—"}
                  </p>
                </div>
                <div>
                  <span className="text-muted-foreground">
                    Teléfono celular:
                  </span>
                  <p className="font-medium text-foreground">
                    {s2.telefonoCelular || "—"}
                  </p>
                </div>
                <div>
                  <span className="text-muted-foreground">
                    Correo electrónico principal:
                  </span>
                  <p className="font-medium text-foreground">
                    {s2.correo1 || "—"}
                  </p>
                </div>
                <div>
                  <span className="text-muted-foreground">
                    Correo secundario:
                  </span>
                  <p className="font-medium text-foreground">
                    {s2.correo2 || "Ninguno declarado"}
                  </p>
                </div>
                <div className="sm:col-span-2">
                  <span className="text-muted-foreground">
                    Dirección domiciliaria:
                  </span>
                  <p className="font-medium text-foreground">
                    {s2.callePrincipal || ""} {s2.numero || ""}{" "}
                    {s2.calleSecundaria ? `y ${s2.calleSecundaria}` : ""}{" "}
                    {s2.sector ? `(Sector: ${s2.sector})` : ""}{" "}
                    {s2.referencia ? `— Ref: ${s2.referencia}` : ""}
                  </p>
                </div>
              </div>
              {!yaEnviada && (
                <div className="mt-4 flex justify-end">
                  <Button
                    asChild
                    variant="ghost"
                    size="sm"
                    className="h-7 text-xs"
                  >
                    <Link to={rutaSeccion(2)}>
                      <Edit3Icon className="mr-1 size-3" /> Editar sección
                    </Link>
                  </Button>
                </div>
              )}
            </AccordionContent>
          </AccordionItem>

          {/* SECCIÓN 3: Parentesco */}
          <AccordionItem
            value="sec-3"
            className="rounded-lg border bg-card px-4"
          >
            <AccordionTrigger className="hover:no-underline">
              <div className="flex items-center gap-3">
                <span className="font-mono font-bold text-xs text-primary">
                  3
                </span>
                <span className="font-medium text-sm">
                  Parentesco IESS / BIESS
                </span>
                {mapaSecciones[3] ? (
                  <Badge
                    variant="outline"
                    className="border-emerald-500 text-2xs text-emerald-700"
                  >
                    Completa
                  </Badge>
                ) : (
                  <Badge variant="secondary" className="text-2xs">
                    Pendiente
                  </Badge>
                )}
              </div>
            </AccordionTrigger>
            <AccordionContent className="pt-2 pb-4 text-sm">
              <div className="flex flex-col gap-2 text-xs">
                <p>
                  <span className="text-muted-foreground">
                    ¿Tiene familiares hasta 4to consanguinidad / 2do afinidad en
                    IESS/BIESS?:
                  </span>{" "}
                  <strong>{s3.tieneFamiliaresIessBiess || "No"}</strong>
                </p>
                {s3.familiares && s3.familiares.length > 0 && (
                  <div className="mt-2 divide-y rounded border">
                    {s3.familiares.map((f, i) => (
                      <div key={i} className="p-2">
                        <p className="font-semibold">{f.nombres}</p>
                        <p className="text-muted-foreground">
                          Parentesco: {f.parentesco} · {f.institucion} ({f.area}{" "}
                          — {f.cargo})
                        </p>
                      </div>
                    ))}
                  </div>
                )}
              </div>
              {!yaEnviada && (
                <div className="mt-4 flex justify-end">
                  <Button
                    asChild
                    variant="ghost"
                    size="sm"
                    className="h-7 text-xs"
                  >
                    <Link to={rutaSeccion(3)}>
                      <Edit3Icon className="mr-1 size-3" /> Editar sección
                    </Link>
                  </Button>
                </div>
              )}
            </AccordionContent>
          </AccordionItem>

          {/* SECCIÓN 4: Cargo */}
          <AccordionItem
            value="sec-4"
            className="rounded-lg border bg-card px-4"
          >
            <AccordionTrigger className="hover:no-underline">
              <div className="flex items-center gap-3">
                <span className="font-mono font-bold text-xs text-primary">
                  4
                </span>
                <span className="font-medium text-sm">
                  Cargo y Sector Postulado
                </span>
                {mapaSecciones[4] ? (
                  <Badge
                    variant="outline"
                    className="border-emerald-500 text-2xs text-emerald-700"
                  >
                    Completa
                  </Badge>
                ) : (
                  <Badge variant="secondary" className="text-2xs">
                    Pendiente
                  </Badge>
                )}
              </div>
            </AccordionTrigger>
            <AccordionContent className="pt-2 pb-4 text-sm">
              <div className="grid grid-cols-1 gap-3 text-xs sm:grid-cols-2">
                <div>
                  <span className="text-muted-foreground">
                    Sector de representación:
                  </span>
                  <p className="font-medium text-foreground capitalize">
                    {s4.sector || "—"}
                  </p>
                </div>
                <div>
                  <span className="text-muted-foreground">
                    Verificación IESS cumplida:
                  </span>
                  <p className="font-medium text-foreground">
                    {s4.verificacionIessCumplida ? "Sí" : "No"}
                  </p>
                </div>
                <div className="sm:col-span-2">
                  <span className="text-muted-foreground">
                    Respaldo de organización:
                  </span>
                  <p className="font-medium text-foreground">
                    {s4.cuentaConRespaldoOrganizacion === "Sí"
                      ? `Sí — Organización: ${s4.nombreOrganizacion || "No especificada"}`
                      : "No cuenta con respaldo"}
                  </p>
                </div>
              </div>
              {!yaEnviada && (
                <div className="mt-4 flex justify-end">
                  <Button
                    asChild
                    variant="ghost"
                    size="sm"
                    className="h-7 text-xs"
                  >
                    <Link to={rutaSeccion(4)}>
                      <Edit3Icon className="mr-1 size-3" /> Editar sección
                    </Link>
                  </Button>
                </div>
              )}
            </AccordionContent>
          </AccordionItem>

          {/* SECCIÓN 5: Formación */}
          <AccordionItem
            value="sec-5"
            className="rounded-lg border bg-card px-4"
          >
            <AccordionTrigger className="hover:no-underline">
              <div className="flex items-center gap-3">
                <span className="font-mono font-bold text-xs text-primary">
                  5
                </span>
                <span className="font-medium text-sm">
                  Formación Académica ({s5.titulos?.length || 0})
                </span>
                {mapaSecciones[5] ? (
                  <Badge
                    variant="outline"
                    className="border-emerald-500 text-2xs text-emerald-700"
                  >
                    Completa
                  </Badge>
                ) : (
                  <Badge variant="secondary" className="text-2xs">
                    Pendiente
                  </Badge>
                )}
              </div>
            </AccordionTrigger>
            <AccordionContent className="pt-2 pb-4 text-sm">
              <div className="flex flex-col gap-2 text-xs">
                {s5.titulos && s5.titulos.length > 0 ? (
                  <div className="divide-y rounded border">
                    {s5.titulos.map((t, idx) => (
                      <div key={idx} className="p-2.5">
                        <p className="font-semibold text-foreground">
                          {t.titulo}
                        </p>
                        <p className="text-muted-foreground">
                          Nivel: {t.nivel} · Institución: {t.institucion} (
                          {t.tipo})
                        </p>
                        <p className="text-muted-foreground">
                          SENESCYT: {t.registroSenescyt} (Reg:{" "}
                          {t.fechaRegistroSenescyt})
                        </p>
                      </div>
                    ))}
                  </div>
                ) : (
                  <p className="italic text-muted-foreground">
                    No se han registrado títulos de educación superior.
                  </p>
                )}
              </div>
              {!yaEnviada && (
                <div className="mt-4 flex justify-end">
                  <Button
                    asChild
                    variant="ghost"
                    size="sm"
                    className="h-7 text-xs"
                  >
                    <Link to={rutaSeccion(5)}>
                      <Edit3Icon className="mr-1 size-3" /> Editar sección
                    </Link>
                  </Button>
                </div>
              )}
            </AccordionContent>
          </AccordionItem>

          {/* SECCIÓN 6: Experiencia */}
          <AccordionItem
            value="sec-6"
            className="rounded-lg border bg-card px-4"
          >
            <AccordionTrigger className="hover:no-underline">
              <div className="flex items-center gap-3">
                <span className="font-mono font-bold text-xs text-primary">
                  6
                </span>
                <span className="font-medium text-sm">
                  Experiencia Laboral ({s6.experiencias?.length || 0})
                </span>
                {mapaSecciones[6] ? (
                  <Badge
                    variant="outline"
                    className="border-emerald-500 text-2xs text-emerald-700"
                  >
                    Completa
                  </Badge>
                ) : (
                  <Badge variant="secondary" className="text-2xs">
                    Pendiente
                  </Badge>
                )}
              </div>
            </AccordionTrigger>
            <AccordionContent className="pt-2 pb-4 text-sm">
              <div className="flex flex-col gap-2 text-xs">
                {s6.experiencias && s6.experiencias.length > 0 ? (
                  <div className="divide-y rounded border">
                    {s6.experiencias.map((exp, idx) => {
                      const fin = exp.actualmenteLaborando
                        ? new Date().toISOString().slice(0, 10)
                        : exp.fechaFin
                      const dur = calcularDuracionExperiencia(
                        exp.fechaInicio,
                        fin,
                      )
                      return (
                        <div key={idx} className="p-2.5">
                          <div className="flex items-start justify-between gap-2">
                            <p className="font-semibold text-foreground">
                              {exp.cargo}
                            </p>
                            <Badge variant="outline" className="text-2xs">
                              {exp.tipo}
                            </Badge>
                          </div>
                          <p className="text-muted-foreground">
                            {exp.institucion}
                          </p>
                          <p className="mt-0.5 text-muted-foreground">
                            Periodo: {exp.fechaInicio} a{" "}
                            {exp.actualmenteLaborando
                              ? "Presente"
                              : exp.fechaFin}
                            {dur.texto ? ` · Duración: ${dur.texto}` : ""}
                          </p>
                          <p className="mt-1 font-medium text-primary">
                            Funciones:{" "}
                            <span className="font-normal text-foreground">
                              {exp.descripcionFunciones || exp.actividades}
                            </span>
                          </p>
                        </div>
                      )
                    })}
                  </div>
                ) : (
                  <p className="italic text-muted-foreground">
                    No se han registrado experiencias laborales.
                  </p>
                )}
              </div>
              {!yaEnviada && (
                <div className="mt-4 flex justify-end">
                  <Button
                    asChild
                    variant="ghost"
                    size="sm"
                    className="h-7 text-xs"
                  >
                    <Link to={rutaSeccion(6)}>
                      <Edit3Icon className="mr-1 size-3" /> Editar sección
                    </Link>
                  </Button>
                </div>
              )}
            </AccordionContent>
          </AccordionItem>

          {/* SECCIÓN 7: Publicaciones */}
          <AccordionItem
            value="sec-7"
            className="rounded-lg border bg-card px-4"
          >
            <AccordionTrigger className="hover:no-underline">
              <div className="flex items-center gap-3">
                <span className="font-mono font-bold text-xs text-primary">
                  7
                </span>
                <span className="font-medium text-sm">
                  Publicaciones e Investigaciones
                </span>
                {mapaSecciones[7] ? (
                  <Badge
                    variant="outline"
                    className="border-emerald-500 text-2xs text-emerald-700"
                  >
                    Completa
                  </Badge>
                ) : (
                  <Badge variant="secondary" className="text-2xs">
                    Pendiente
                  </Badge>
                )}
              </div>
            </AccordionTrigger>
            <AccordionContent className="pt-2 pb-4 text-sm">
              <div className="flex flex-col gap-2 text-xs">
                <p>
                  <span className="text-muted-foreground">
                    ¿Cuenta con publicaciones?:
                  </span>{" "}
                  <strong>{s7.cuentaConPublicaciones || "No"}</strong>
                </p>
                {s7.publicaciones && s7.publicaciones.length > 0 && (
                  <div className="divide-y rounded border">
                    {s7.publicaciones.map((pub, idx) => (
                      <div key={idx} className="p-2">
                        <p className="font-semibold text-foreground">
                          {pub.titulo}
                        </p>
                        <p className="text-muted-foreground">
                          {pub.tipo} · {pub.medio} ({pub.fechaPublicacion})
                        </p>
                      </div>
                    ))}
                  </div>
                )}
              </div>
              {!yaEnviada && (
                <div className="mt-4 flex justify-end">
                  <Button
                    asChild
                    variant="ghost"
                    size="sm"
                    className="h-7 text-xs"
                  >
                    <Link to={rutaSeccion(7)}>
                      <Edit3Icon className="mr-1 size-3" /> Editar sección
                    </Link>
                  </Button>
                </div>
              )}
            </AccordionContent>
          </AccordionItem>

          {/* SECCIÓN 8: Capacitación */}
          <AccordionItem
            value="sec-8"
            className="rounded-lg border bg-card px-4"
          >
            <AccordionTrigger className="hover:no-underline">
              <div className="flex items-center gap-3">
                <span className="font-mono font-bold text-xs text-primary">
                  8
                </span>
                <span className="font-medium text-sm">
                  Capacitación ({s8.capacitaciones?.length || 0})
                </span>
                {mapaSecciones[8] ? (
                  <Badge
                    variant="outline"
                    className="border-emerald-500 text-2xs text-emerald-700"
                  >
                    Completa
                  </Badge>
                ) : (
                  <Badge variant="secondary" className="text-2xs">
                    Pendiente
                  </Badge>
                )}
              </div>
            </AccordionTrigger>
            <AccordionContent className="pt-2 pb-4 text-sm">
              <div className="flex flex-col gap-2 text-xs">
                {s8.capacitaciones && s8.capacitaciones.length > 0 ? (
                  <div className="divide-y rounded border">
                    {s8.capacitaciones.map((c, idx) => (
                      <div key={idx} className="p-2">
                        <p className="font-semibold text-foreground">
                          {c.nombre} ({c.horas} horas)
                        </p>
                        <p className="text-muted-foreground">
                          {c.tipo} · {c.institucion} ({c.fechaInicio} a{" "}
                          {c.fechaFin})
                        </p>
                      </div>
                    ))}
                  </div>
                ) : (
                  <p className="italic text-muted-foreground">
                    No se han registrado cursos o talleres adicionales.
                  </p>
                )}
              </div>
              {!yaEnviada && (
                <div className="mt-4 flex justify-end">
                  <Button
                    asChild
                    variant="ghost"
                    size="sm"
                    className="h-7 text-xs"
                  >
                    <Link to={rutaSeccion(8)}>
                      <Edit3Icon className="mr-1 size-3" /> Editar sección
                    </Link>
                  </Button>
                </div>
              )}
            </AccordionContent>
          </AccordionItem>

          {/* SECCIÓN 9: Inhabilidades */}
          <AccordionItem
            value="sec-9"
            className="rounded-lg border bg-card px-4"
          >
            <AccordionTrigger className="hover:no-underline">
              <div className="flex items-center gap-3">
                <span className="font-mono font-bold text-xs text-primary">
                  9
                </span>
                <span className="font-medium text-sm">
                  Inhabilidades y Prohibiciones (13 preguntas)
                </span>
                {mapaSecciones[9] ? (
                  <Badge
                    variant="outline"
                    className="border-emerald-500 text-2xs text-emerald-700"
                  >
                    Completa
                  </Badge>
                ) : (
                  <Badge variant="secondary" className="text-2xs">
                    Pendiente
                  </Badge>
                )}
              </div>
            </AccordionTrigger>
            <AccordionContent className="pt-2 pb-4 text-sm">
              <div className="flex flex-col gap-2 text-xs">
                {PREGUNTAS_INHABILIDADES.map((p) => {
                  const val =
                    (s9 as Record<string, string | undefined>)[p.id] || "—"
                  return (
                    <div
                      key={p.id}
                      className="flex flex-col border-b py-1 last:border-b-0"
                    >
                      <div className="flex items-center justify-between">
                        <span className="pr-4 text-muted-foreground">
                          <strong className="text-foreground">
                            {p.numero}
                          </strong>{" "}
                          {p.texto}
                        </span>
                        <Badge
                          variant={val === "Sí" ? "default" : "outline"}
                          className="shrink-0 text-2xs"
                        >
                          {val}
                        </Badge>
                      </div>
                      {p.id === "q7" && val === "Sí" && s9.q7Detalles && (
                        <div className="mt-1.5 rounded bg-muted/20 p-2 text-2xs text-muted-foreground">
                          <p>
                            <strong>Institución:</strong>{" "}
                            {s9.q7Detalles.institucion}
                          </p>
                          <p>
                            <strong>Cargo:</strong> {s9.q7Detalles.cargo} ·{" "}
                            <strong>Fecha:</strong> {s9.q7Detalles.fechaFin}
                          </p>
                          <p>
                            <strong>Motivo:</strong> {s9.q7Detalles.motivo}
                          </p>
                        </div>
                      )}
                    </div>
                  )
                })}
              </div>
              {!yaEnviada && (
                <div className="mt-4 flex justify-end">
                  <Button
                    asChild
                    variant="ghost"
                    size="sm"
                    className="h-7 text-xs"
                  >
                    <Link to={rutaSeccion(9)}>
                      <Edit3Icon className="mr-1 size-3" /> Editar sección
                    </Link>
                  </Button>
                </div>
              )}
            </AccordionContent>
          </AccordionItem>
        </Accordion>
      </div>

      {/* Bloque de Consentimiento y Declaración Legal */}
      <div className="mb-8">
        <ConsentimientoEF12
          datos={consentimiento}
          onChange={setConsentimiento}
          politica={politicaVigente}
          disabled={
            yaEnviada ||
            !disponibilidad.abierta ||
            Boolean(politicaVigente?.restringido)
          }
        />
      </div>

      {!yaEnviada && !disponibilidad.abierta && (
        <p
          role="alert"
          className="rounded-lg border border-destructive/40 p-4 text-sm"
        >
          La recepción del período no está disponible o no pudo confirmarse. El
          borrador se conserva; no se admiten nuevos envíos.{" "}
          <Link className="underline" to={RUTAS.inicio}>
            Consultar convocatoria
          </Link>
        </p>
      )}
      {/* Acciones Finales de Envío */}
      <div className="flex flex-wrap items-center justify-between gap-4 border-t pt-6">
        <Button
          asChild
          variant="outline"
          size="sm"
          className="flex items-center gap-1.5"
        >
          <Link to={RUTAS.postulacion}>
            <ArrowLeftIcon className="size-4" aria-hidden="true" />
            <span>Volver al formulario</span>
          </Link>
        </Button>

        {!yaEnviada ? (
          <div className="flex items-center gap-3">
            {documentosPendientes.length > 0 ? (
              <span className="hidden text-xs font-medium text-destructive sm:inline">
                {documentosPendientes.length} documentos obligatorios pendientes
              </span>
            ) : !todosConsentimientosAceptados ? (
              <span className="hidden text-xs font-medium text-amber-700 dark:text-amber-300 sm:inline">
                Debe marcar los 5 puntos para habilitar el envío
              </span>
            ) : null}
            <Button
              type="button"
              onClick={handleEnviarPostulacion}
              disabled={
                !todosConsentimientosAceptados ||
                !formularioCompleto ||
                documentosPendientes.length > 0 ||
                sellarMutation.isPending ||
                !disponibilidad.abierta ||
                Boolean(politicaVigente?.restringido)
              }
              className="flex items-center gap-2 bg-emerald-700 text-white hover:bg-emerald-800"
            >
              <SendIcon className="size-4" aria-hidden="true" />
              <span>
                {sellarMutation.isPending
                  ? "Sellando y enviando…"
                  : "Enviar Postulación Definitiva"}
              </span>
            </Button>
          </div>
        ) : (
          <Button
            asChild
            className="flex items-center gap-2 bg-emerald-700 text-white hover:bg-emerald-800"
          >
            <Link to={RUTAS.postulacionImprimir}>
              <PrinterIcon className="size-4" aria-hidden="true" />
              <span>Imprimir Formulario Firmado (HU-11)</span>
            </Link>
          </Button>
        )}
      </div>
    </div>
  )
}
