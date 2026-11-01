import {
  AlertCircleIcon,
  ArrowLeftIcon,
  ArrowRightIcon,
  CheckCircleIcon,
  FileCheck2Icon,
  LockIcon,
  SaveIcon,
} from "lucide-react"
import { useEffect, useMemo, useState } from "react"
import { Link, useNavigate, useParams, useSearchParams } from "react-router"
import { toast } from "sonner"

import { EstadoCargando, EstadoError } from "@/components/estados"
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
  BarraProgresoSecciones,
  Seccion1Personal,
  Seccion2Contacto,
  Seccion3Adicional,
  Seccion4Cargo,
  Seccion5Academica,
  Seccion6Experiencia,
  Seccion7Publicaciones,
  Seccion8Capacitacion,
  Seccion9Inhabilidades,
  useDocumentosPostulacion,
  useSubirDocumento,
  useEliminarDocumento,
  useDescargarDocumento,
  type Seccion1Data,
  type Seccion2Data,
  type Seccion3Data,
  type Seccion4Data,
  type Seccion5Data,
  type Seccion6Data,
  type Seccion7Data,
  type Seccion8Data,
  type Seccion9Data,
} from "@/features/formulario"
import {
  useCrearPostulacion,
  useGuardarSeccion,
  useListarPostulaciones,
} from "@/features/postulaciones/queries"
import { useDisponibilidadEF12 } from "@/features/convocatorias/use-disponibilidad"
import { esErrorApi } from "@/lib/api-error"
import { formatearTamanoBytes } from "@/lib/clientes/documentos"
import { RUTAS, rutaSeccion } from "@/lib/rutas"
import { validarSeccionEF12 } from "@/features/formulario/validacion"

export interface FormularioPostulacionState {
  seccion1: Seccion1Data
  seccion2: Seccion2Data
  seccion3: Seccion3Data
  seccion4: Seccion4Data
  seccion5: Seccion5Data
  seccion6: Seccion6Data
  seccion7: Seccion7Data
  seccion8: Seccion8Data
  seccion9: Seccion9Data
}

const FORMULARIO_BASE_DEFAULT: FormularioPostulacionState = {
  seccion1: {
    nombres: "",
    apellidos: "",
    cedula: "",
    genero: "Femenino",
    fechaNacimiento: "1990-01-01",
    nacionalidad: "Ecuatoriana",
    estadoCivil: "Soltero/a",
  },
  seccion2: {
    pais: "Ecuador",
    provincia: "",
    canton: "",
    telefonoCelular: "",
    correo1: "",
    callePrincipal: "",
    numero: "",
    calleSecundaria: "",
    sector: "",
    referencia: "",
    telefonoDomiciliario: "",
  },
  seccion3: {
    tieneFamiliaresIessBiess: "No",
    familiares: [],
  },
  seccion4: {
    sector: "afiliados",
    verificacionIessCumplida: true,
    cuentaConRespaldoOrganizacion: "No",
  },
  seccion5: {
    titulos: [],
  },
  seccion6: {
    experiencias: [],
  },
  seccion7: {
    cuentaConPublicaciones: "No",
    publicaciones: [],
  },
  seccion8: {
    capacitaciones: [],
  },
  seccion9: {
    q1: "Sí",
    q2: "No",
    q3: "No",
    q4: "No",
    q5: "No",
    q6: "No",
    q7: "No",
    q8: "No",
    q9: "No",
    q10: "No",
    q11: "No",
    q12: "No",
    q13: "No",
  },
}

export function PostulacionView() {
  const { seccion } = useParams<{ seccion: string }>()
  const navegar = useNavigate()
  const [parametros] = useSearchParams()

  const numeroSeccion = seccion ? Number(seccion) : 1
  const esSeccionValida =
    Number.isInteger(numeroSeccion) && numeroSeccion >= 1 && numeroSeccion <= 9

  // Consultar postulación existente o crear borrador
  const {
    data: postulacionesData,
    isLoading: cargandoPostulaciones,
    error: errorPostulaciones,
    refetch: recargarPostulaciones,
  } = useListarPostulaciones({ limit: 1 })

  const crearPostulacionMutation = useCrearPostulacion()
  const guardarSeccionMutation = useGuardarSeccion()

  const postulacionActiva = postulacionesData?.items?.[0]
  const periodo =
    postulacionActiva?.convocatoriaId ?? parametros.get("convocatoria")
  const disponibilidad = useDisponibilidadEF12(periodo)
  const abiertaConvocatoria = disponibilidad.abierta

  // Crear borrador automático si el postulante aún no tiene ninguno
  useEffect(() => {
    if (
      !cargandoPostulaciones &&
      postulacionesData &&
      postulacionesData.items.length === 0 &&
      !crearPostulacionMutation.isPending &&
      !crearPostulacionMutation.isSuccess &&
      !crearPostulacionMutation.isError &&
      Boolean(periodo) &&
      abiertaConvocatoria
    ) {
      crearPostulacionMutation.mutate({ convocatoriaId: periodo! })
    }
  }, [
    cargandoPostulaciones,
    postulacionesData,
    crearPostulacionMutation,
    periodo,
    abiertaConvocatoria,
  ])

  // Contenido local del formulario sincronizado con la postulación
  const [formData, setFormData] = useState<FormularioPostulacionState>(
    FORMULARIO_BASE_DEFAULT,
  )
  const [bloqueadoNacionalidad, setBloqueadoNacionalidad] = useState(false)
  const [errorValidacion, setErrorValidacion] = useState<string | null>(null)

  // Ajuste seguro de estado en fase de render para evitar efecto en cascada
  const [prevId, setPrevId] = useState<string | null>(null)
  if (postulacionActiva && postulacionActiva.id !== prevId) {
    setPrevId(postulacionActiva.id)
    if (postulacionActiva.content) {
      setFormData((prev) => ({
        ...prev,
        ...(postulacionActiva.content as unknown as Partial<FormularioPostulacionState>),
      }))
    }
  }

  // Mapa de secciones completadas según el progreso devuelto por la API
  const seccionesCompletadas = useMemo(() => {
    const mapa: Record<number, boolean> = {}
    for (let i = 1; i <= 9; i++) mapa[i] = false

    if (postulacionActiva?.progress?.sections) {
      for (const s of postulacionActiva.progress.sections) {
        mapa[s.number] = s.complete
      }
    }
    return mapa
  }, [postulacionActiva])

  // Documentos reales y cuota total (FT18)
  const { cuota } = useDocumentosPostulacion(postulacionActiva?.id)
  const subirDocMutation = useSubirDocumento(postulacionActiva?.id)
  const eliminarDocMutation = useEliminarDocumento(postulacionActiva?.id)
  const descargarDocMutation = useDescargarDocumento()
  const [cargandoDocId, setCargandoDocId] = useState<string | null>(null)

  async function subirRespaldoSeccion4(archivo: File) {
    if (!postulacionActiva?.id) return
    try {
      setCargandoDocId("sec4-org")
      await guardarSeccionMutation.mutateAsync({
        id: postulacionActiva.id,
        seccion: 4,
        datos: formData.seccion4,
      })
      const doc = await subirDocMutation.mutateAsync({
        seccion: 4,
        archivo,
        recordId: "",
      })
      setFormData((prev) => ({
        ...prev,
        seccion4: {
          ...prev.seccion4,
          documentoRespaldoPdf: {
            id: doc.id,
            nombre: doc.name,
            tamano: formatearTamanoBytes(doc.sizeBytes),
            sizeBytes: doc.sizeBytes,
            referenciaId: doc.id,
            sha256: doc.sha256,
          },
        },
      }))
    } finally {
      setCargandoDocId(null)
    }
  }

  async function eliminarRespaldoSeccion4() {
    const docId = formData.seccion4.documentoRespaldoPdf?.id
    if (docId) {
      await eliminarDocMutation.mutateAsync(docId)
    }
    setFormData((prev) => ({
      ...prev,
      seccion4: {
        ...prev.seccion4,
        documentoRespaldoPdf: null,
      },
    }))
  }

  async function subirRespaldoTitulo(tituloId: string, archivo: File) {
    if (!postulacionActiva?.id) return
    try {
      setCargandoDocId(tituloId)
      await guardarSeccionMutation.mutateAsync({
        id: postulacionActiva.id,
        seccion: 5,
        datos: formData.seccion5,
      })
      const doc = await subirDocMutation.mutateAsync({
        seccion: 5,
        archivo,
        recordId: tituloId,
      })
      setFormData((prev) => ({
        ...prev,
        seccion5: {
          ...prev.seccion5,
          titulos: prev.seccion5.titulos.map((t) =>
            t.id === tituloId
              ? {
                  ...t,
                  documentoPdf: {
                    id: doc.id,
                    nombre: doc.name,
                    tamano: formatearTamanoBytes(doc.sizeBytes),
                    sizeBytes: doc.sizeBytes,
                    referenciaId: doc.id,
                    sha256: doc.sha256,
                  },
                }
              : t,
          ),
        },
      }))
    } finally {
      setCargandoDocId(null)
    }
  }

  async function eliminarRespaldoTitulo(tituloId: string) {
    const t = formData.seccion5.titulos.find((x) => x.id === tituloId)
    if (t?.documentoPdf?.id) {
      await eliminarDocMutation.mutateAsync(t.documentoPdf.id)
    }
    setFormData((prev) => ({
      ...prev,
      seccion5: {
        ...prev.seccion5,
        titulos: prev.seccion5.titulos.map((x) =>
          x.id === tituloId ? { ...x, documentoPdf: null } : x,
        ),
      },
    }))
  }

  async function subirRespaldoExperiencia(expId: string, archivo: File) {
    if (!postulacionActiva?.id) return
    try {
      setCargandoDocId(expId)
      await guardarSeccionMutation.mutateAsync({
        id: postulacionActiva.id,
        seccion: 6,
        datos: formData.seccion6,
      })
      const doc = await subirDocMutation.mutateAsync({
        seccion: 6,
        archivo,
        recordId: expId,
      })
      setFormData((prev) => ({
        ...prev,
        seccion6: {
          ...prev.seccion6,
          experiencias: prev.seccion6.experiencias.map((e) =>
            e.id === expId
              ? {
                  ...e,
                  documentoPdf: {
                    id: doc.id,
                    nombre: doc.name,
                    tamano: formatearTamanoBytes(doc.sizeBytes),
                    sizeBytes: doc.sizeBytes,
                    referenciaId: doc.id,
                    sha256: doc.sha256,
                  },
                }
              : e,
          ),
        },
      }))
    } finally {
      setCargandoDocId(null)
    }
  }

  async function eliminarRespaldoExperiencia(expId: string) {
    const e = formData.seccion6.experiencias.find((x) => x.id === expId)
    if (e?.documentoPdf?.id) {
      await eliminarDocMutation.mutateAsync(e.documentoPdf.id)
    }
    setFormData((prev) => ({
      ...prev,
      seccion6: {
        ...prev.seccion6,
        experiencias: prev.seccion6.experiencias.map((x) =>
          x.id === expId ? { ...x, documentoPdf: null } : x,
        ),
      },
    }))
  }

  async function subirRespaldoPublicacion(pubId: string, archivo: File) {
    if (!postulacionActiva?.id) return
    try {
      setCargandoDocId(pubId)
      await guardarSeccionMutation.mutateAsync({
        id: postulacionActiva.id,
        seccion: 7,
        datos: formData.seccion7,
      })
      const doc = await subirDocMutation.mutateAsync({
        seccion: 7,
        archivo,
        recordId: pubId,
      })
      setFormData((prev) => ({
        ...prev,
        seccion7: {
          ...prev.seccion7,
          publicaciones: prev.seccion7.publicaciones.map((p) =>
            p.id === pubId
              ? {
                  ...p,
                  documentoPdf: {
                    id: doc.id,
                    nombre: doc.name,
                    tamano: formatearTamanoBytes(doc.sizeBytes),
                    sizeBytes: doc.sizeBytes,
                    referenciaId: doc.id,
                    sha256: doc.sha256,
                  },
                }
              : p,
          ),
        },
      }))
    } finally {
      setCargandoDocId(null)
    }
  }

  async function eliminarRespaldoPublicacion(pubId: string) {
    const p = formData.seccion7.publicaciones.find((x) => x.id === pubId)
    if (p?.documentoPdf?.id) {
      await eliminarDocMutation.mutateAsync(p.documentoPdf.id)
    }
    setFormData((prev) => ({
      ...prev,
      seccion7: {
        ...prev.seccion7,
        publicaciones: prev.seccion7.publicaciones.map((x) =>
          x.id === pubId ? { ...x, documentoPdf: null } : x,
        ),
      },
    }))
  }

  async function subirRespaldoCapacitacion(capId: string, archivo: File) {
    if (!postulacionActiva?.id) return
    try {
      setCargandoDocId(capId)
      await guardarSeccionMutation.mutateAsync({
        id: postulacionActiva.id,
        seccion: 8,
        datos: formData.seccion8,
      })
      const doc = await subirDocMutation.mutateAsync({
        seccion: 8,
        archivo,
        recordId: capId,
      })
      setFormData((prev) => ({
        ...prev,
        seccion8: {
          ...prev.seccion8,
          capacitaciones: prev.seccion8.capacitaciones.map((c) =>
            c.id === capId
              ? {
                  ...c,
                  documentoPdf: {
                    id: doc.id,
                    nombre: doc.name,
                    tamano: formatearTamanoBytes(doc.sizeBytes),
                    sizeBytes: doc.sizeBytes,
                    referenciaId: doc.id,
                    sha256: doc.sha256,
                  },
                }
              : c,
          ),
        },
      }))
    } finally {
      setCargandoDocId(null)
    }
  }

  async function eliminarRespaldoCapacitacion(capId: string) {
    const c = formData.seccion8.capacitaciones.find((x) => x.id === capId)
    if (c?.documentoPdf?.id) {
      await eliminarDocMutation.mutateAsync(c.documentoPdf.id)
    }
    setFormData((prev) => ({
      ...prev,
      seccion8: {
        ...prev.seccion8,
        capacitaciones: prev.seccion8.capacitaciones.map((x) =>
          x.id === capId ? { ...x, documentoPdf: null } : x,
        ),
      },
    }))
  }

  if (
    !cargandoPostulaciones &&
    !postulacionActiva &&
    crearPostulacionMutation.isError
  ) {
    return (
      <EstadoError
        error={crearPostulacionMutation.error}
        titulo="No se pudo abrir el borrador"
        onReintentar={() => {
          crearPostulacionMutation.reset()
          void disponibilidad.refetch()
        }}
      />
    )
  }
  if (
    !cargandoPostulaciones &&
    !postulacionActiva &&
    !abiertaConvocatoria &&
    !crearPostulacionMutation.isPending
  ) {
    return (
      <div className="mx-auto w-full max-w-3xl px-4 py-10">
        <Card>
          <CardHeader>
            <CardTitle>
              {disponibilidad.isPending && periodo
                ? "Confirmando disponibilidad…"
                : "La recepción no está disponible"}
            </CardTitle>
            <CardDescription>
              {periodo
                ? "No se ha confirmado una convocatoria abierta y habilitada. No se crea un borrador ni se envían cambios mientras no se confirme el período."
                : "Seleccione una convocatoria publicada antes de comenzar su postulación."}
            </CardDescription>
          </CardHeader>
          <CardFooter>
            <Button asChild variant="outline">
              <Link to={RUTAS.inicio}>Consultar convocatorias</Link>
            </Button>
          </CardFooter>
        </Card>
      </div>
    )
  }

  if (seccion && !esSeccionValida) {
    return (
      <div className="mx-auto w-full max-w-3xl px-4 py-12">
        <Card>
          <CardHeader>
            <CardTitle>Sección no encontrada</CardTitle>
            <CardDescription>
              El formulario EF-12 tiene 9 secciones numeradas del 1 al 9. La
              sección «{seccion}» no existe.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button asChild variant="outline">
              <Link to={RUTAS.postulacion}>Abrir el formulario</Link>
            </Button>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (cargandoPostulaciones || crearPostulacionMutation.isPending) {
    return (
      <div className="mx-auto w-full max-w-4xl px-4 py-12">
        <EstadoCargando
          mensaje="Cargando su postulación…"
          variante="pantalla-completa"
        />
      </div>
    )
  }

  if (errorPostulaciones) {
    return (
      <div className="mx-auto w-full max-w-4xl px-4 py-12">
        <EstadoError
          error={errorPostulaciones}
          titulo="Error al cargar la postulación"
          onReintentar={() => recargarPostulaciones()}
        />
      </div>
    )
  }

  const yaEnviada =
    postulacionActiva && postulacionActiva.status === "submitted"
  const estaCerrada = postulacionActiva && postulacionActiva.status === "closed"
  const noModificable =
    yaEnviada || Boolean(estaCerrada) || !abiertaConvocatoria

  // La disponibilidad real del formulario EF-12 la define el servidor mediante ef12Disponible (T18)

  async function guardarSeccionActual(irSiguiente = false): Promise<boolean> {
    if (!postulacionActiva?.id) return false

    if (estaCerrada || yaEnviada) {
      toast.error(
        "La postulación se encuentra cerrada y no admite modificaciones.",
      )
      return false
    }

    if (!abiertaConvocatoria && !yaEnviada) {
      toast.error(
        "La convocatoria no admite modificaciones porque el plazo concluyó o la postulación está deshabilitada.",
      )
      return false
    }

    setErrorValidacion(null)
    const claveSeccion = `seccion${numeroSeccion}` as keyof typeof formData
    const contenidoSeccion = formData[claveSeccion]

    // Validación pura
    const resultado = validarSeccionEF12(contenidoSeccion, numeroSeccion, false)
    if (!resultado.esValido && resultado.errores.length > 0) {
      const primerError = resultado.errores[0].mensaje
      setErrorValidacion(primerError)
      toast.warning(`Atención en sección ${numeroSeccion}: ${primerError}`)
      return false
    }

    try {
      await guardarSeccionMutation.mutateAsync({
        id: postulacionActiva.id,
        seccion: numeroSeccion,
        datos: contenidoSeccion,
      })

      if (irSiguiente) {
        if (numeroSeccion < 9) {
          navegar(
            `${rutaSeccion(numeroSeccion + 1)}${periodo ? `?convocatoria=${encodeURIComponent(periodo)}` : ""}`,
          )
        } else {
          navegar(RUTAS.postulacionRevision)
        }
      }
      return true
    } catch (err: unknown) {
      if (esErrorApi(err)) {
        if (
          err.codigo === "postulacion_cerrada_elegibilidad" ||
          (err.esConflicto() &&
            err.mensaje.toLowerCase().includes("elegibilidad"))
        ) {
          toast.error(
            "La postulación no puede continuar por no cumplir los criterios de elegibilidad del sector.",
          )
          void recargarPostulaciones()
          return false
        }
        if (
          err.codigo === "elegibilidad_no_disponible" ||
          err.esServicioNoDisponible()
        ) {
          toast.warning(
            "La verificación de elegibilidad no está disponible temporalmente. Puede continuar editando y volver a intentar más tarde.",
          )
          return false
        }
        if (err.codigo === "convocatoria_no_disponible")
          void disponibilidad.refetch()
        toast.error(err.mensaje || "Error al guardar la sección.")
      }
      return false
    }
  }

  return (
    <div className="mx-auto flex w-full max-w-5xl flex-1 flex-col gap-6 px-4 py-8">
      {/* Banner de POSTULACIÓN SELLADA */}
      {yaEnviada && (
        <div
          role="alert"
          className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-primary/30 bg-primary/10 p-4 text-sm text-primary"
        >
          <div className="flex items-center gap-3">
            <FileCheck2Icon className="size-5 shrink-0" aria-hidden="true" />
            <div>
              <h4 className="font-semibold">Postulación enviada y sellada</h4>
              <p className="mt-0.5 text-xs text-muted-foreground">
                Código:{" "}
                <strong className="text-foreground">
                  {postulacionActiva.code}
                </strong>
                {postulacionActiva.hash && (
                  <>
                    {" "}
                    · Hash:{" "}
                    <code className="font-mono text-2xs text-foreground">
                      {postulacionActiva.hash.slice(0, 16)}…
                    </code>
                  </>
                )}{" "}
                · Este formulario ya no admite modificaciones adicionales.
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Button asChild variant="outline" size="sm">
              <Link to={RUTAS.postulacionRevision}>Ver revisión</Link>
            </Button>
            <Button asChild size="sm">
              <Link to={RUTAS.postulacionImprimir}>Imprimir (HU-11)</Link>
            </Button>
          </div>
        </div>
      )}

      {/* Banner de POSTULACIÓN CERRADA POR ELEGIBILIDAD (Mockup 06-04b) */}
      {estaCerrada && (
        <div
          role="alert"
          aria-live="assertive"
          className="rounded-lg border border-destructive/50 bg-destructive/10 p-4 space-y-3"
        >
          <div className="flex items-start gap-3">
            <AlertCircleIcon className="size-5 text-destructive shrink-0 mt-0.5" />
            <div className="flex-1">
              <h4 className="font-semibold text-destructive">
                La postulación no puede continuar
              </h4>
              <p className="mt-0.5 text-xs text-destructive/90">
                {postulacionActiva.elegibilidad?.mensaje ||
                  "Validación de elegibilidad no cumplida. La postulación se encuentra cerrada automáticamente."}
              </p>
            </div>
          </div>

          <div className="rounded border bg-background/80 p-3 text-xs space-y-1">
            <div className="flex justify-between">
              <span className="text-muted-foreground">Sector:</span>
              <span className="font-medium capitalize">
                {postulacionActiva.elegibilidad?.sector ||
                  formData.seccion4.sector}
              </span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Resultado:</span>
              <span className="font-medium">
                {postulacionActiva.elegibilidad?.mensaje ||
                  "No cumple condiciones de elegibilidad"}
              </span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">
                Origen de verificación:
              </span>
              <span className="font-mono text-muted-foreground">
                {postulacionActiva.elegibilidad?.origen ===
                "simulacion_controlada_servidor"
                  ? "simulación controlada del servidor"
                  : postulacionActiva.elegibilidad?.origen || "servidor"}
              </span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">
                Estado del expediente:
              </span>
              <span className="font-semibold text-destructive">
                Cerrada automáticamente
              </span>
            </div>
          </div>

          <div className="flex items-center justify-between gap-3 pt-1">
            <p className="text-xs text-muted-foreground">
              Si considera que el resultado es incorrecto, puede presentar un
              reclamo ante la Comisión Especial dentro del plazo de la
              convocatoria.
            </p>
            <Button asChild variant="secondary" size="sm">
              <Link to={RUTAS.inicio}>Volver a la convocatoria</Link>
            </Button>
          </div>
        </div>
      )}

      {!abiertaConvocatoria && !yaEnviada && !estaCerrada && (
        <div
          role="alert"
          className="flex items-center gap-3 rounded-lg border border-destructive/40 bg-destructive/10 p-4 text-sm text-foreground dark:bg-destructive/20"
        >
          <LockIcon
            className="size-5 shrink-0 text-destructive"
            aria-hidden="true"
          />
          <div>
            <h4 className="font-semibold text-destructive">
              Recepción no disponible o sin confirmar
            </h4>
            <p className="mt-0.5 text-xs text-muted-foreground">
              No se ha confirmado una recepción abierta para el período de
              origen. El plazo puede haber concluido, estar deshabilitado o no
              haberse podido consultar el servidor. Su borrador se conserva; por
              ahora no se envían modificaciones ni nuevos expedientes.
            </p>
          </div>
        </div>
      )}

      {/* Stepper y barra de progreso */}
      <BarraProgresoSecciones
        seccionActiva={numeroSeccion}
        seccionesCompletadas={seccionesCompletadas}
      />

      {/* Indicador de cuota total de documentos (FT18) */}
      <div className="flex flex-wrap items-center justify-between gap-2 rounded-md border bg-muted/20 px-3.5 py-2 text-xs">
        <div className="flex items-center gap-2">
          <span className="font-medium text-foreground">
            Cuota de respaldos PDF:
          </span>
          <span className="text-muted-foreground">{cuota.formateado}</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="h-2 w-28 overflow-hidden rounded-full bg-muted">
            <div
              className={`h-full transition-all ${cuota.excedida ? "bg-destructive" : cuota.porcentaje > 80 ? "bg-amber-500" : "bg-primary"}`}
              style={{ width: `${cuota.porcentaje}%` }}
            />
          </div>
          <span
            className={`font-mono text-2xs ${cuota.excedida ? "font-bold text-destructive" : "text-muted-foreground"}`}
          >
            {cuota.porcentaje}%
          </span>
        </div>
      </div>

      {errorValidacion && (
        <div
          role="alert"
          className="flex items-center gap-2 rounded-md border border-amber-500/40 bg-amber-500/10 p-3 text-xs text-amber-800 dark:text-amber-400"
        >
          <span>{errorValidacion}</span>
        </div>
      )}

      {/* Tarjeta contenedora de la sección activa */}
      <Card className="shadow-xs">
        <CardContent className="pt-6">
          {numeroSeccion === 1 && (
            <Seccion1Personal
              datos={formData.seccion1}
              onChange={(s1) =>
                setFormData((prev) => ({ ...prev, seccion1: s1 }))
              }
              disabled={noModificable}
              onBloqueoNacionalidad={setBloqueadoNacionalidad}
            />
          )}

          {numeroSeccion === 2 && (
            <Seccion2Contacto
              datos={formData.seccion2 as Seccion2Data}
              onChange={(s2) =>
                setFormData((prev) => ({ ...prev, seccion2: s2 }))
              }
              disabled={noModificable}
            />
          )}

          {numeroSeccion === 3 && (
            <Seccion3Adicional
              datos={formData.seccion3 as Seccion3Data}
              onChange={(s3) =>
                setFormData((prev) => ({ ...prev, seccion3: s3 }))
              }
              disabled={noModificable}
            />
          )}

          {numeroSeccion === 4 && (
            <Seccion4Cargo
              datos={formData.seccion4 as Seccion4Data}
              onChange={(s4) =>
                setFormData((prev) => ({ ...prev, seccion4: s4 }))
              }
              disabled={noModificable}
              elegibilidad={postulacionActiva?.elegibilidad}
              onSubirRespaldo={subirRespaldoSeccion4}
              onEliminarRespaldo={eliminarRespaldoSeccion4}
              onDescargarRespaldo={
                formData.seccion4.documentoRespaldoPdf?.id
                  ? () =>
                      descargarDocMutation.mutate(
                        formData.seccion4.documentoRespaldoPdf!.id!,
                      )
                  : undefined
              }
              cargandoRespaldo={cargandoDocId === "sec4-org"}
            />
          )}

          {numeroSeccion === 5 && (
            <Seccion5Academica
              datos={formData.seccion5 as Seccion5Data}
              onChange={(s5) =>
                setFormData((prev) => ({ ...prev, seccion5: s5 }))
              }
              disabled={noModificable}
              onSubirRespaldo={subirRespaldoTitulo}
              onEliminarRespaldo={eliminarRespaldoTitulo}
              onDescargarRespaldo={(id) => descargarDocMutation.mutate(id)}
              cargandoRespaldoId={cargandoDocId}
            />
          )}

          {numeroSeccion === 6 && (
            <Seccion6Experiencia
              datos={formData.seccion6 as Seccion6Data}
              onChange={(s6) =>
                setFormData((prev) => ({ ...prev, seccion6: s6 }))
              }
              disabled={noModificable}
              onSubirRespaldo={subirRespaldoExperiencia}
              onEliminarRespaldo={eliminarRespaldoExperiencia}
              onDescargarRespaldo={(id) => descargarDocMutation.mutate(id)}
              cargandoRespaldoId={cargandoDocId}
            />
          )}

          {numeroSeccion === 7 && (
            <Seccion7Publicaciones
              datos={formData.seccion7 as Seccion7Data}
              onChange={(s7) =>
                setFormData((prev) => ({ ...prev, seccion7: s7 }))
              }
              disabled={noModificable}
              onSubirRespaldo={subirRespaldoPublicacion}
              onEliminarRespaldo={eliminarRespaldoPublicacion}
              onDescargarRespaldo={(id) => descargarDocMutation.mutate(id)}
              cargandoRespaldoId={cargandoDocId}
            />
          )}

          {numeroSeccion === 8 && (
            <Seccion8Capacitacion
              datos={formData.seccion8 as Seccion8Data}
              onChange={(s8) =>
                setFormData((prev) => ({ ...prev, seccion8: s8 }))
              }
              disabled={noModificable}
              onSubirRespaldo={subirRespaldoCapacitacion}
              onEliminarRespaldo={eliminarRespaldoCapacitacion}
              onDescargarRespaldo={(id) => descargarDocMutation.mutate(id)}
              cargandoRespaldoId={cargandoDocId}
            />
          )}

          {numeroSeccion === 9 && (
            <Seccion9Inhabilidades
              datos={formData.seccion9 as Seccion9Data}
              onChange={(s9) =>
                setFormData((prev) => ({ ...prev, seccion9: s9 }))
              }
              disabled={noModificable}
            />
          )}
        </CardContent>

        <CardFooter className="flex flex-wrap items-center justify-between gap-3 border-t bg-muted/10 p-4">
          <div className="flex items-center gap-2">
            {numeroSeccion > 1 && (
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => navegar(rutaSeccion(numeroSeccion - 1))}
                disabled={guardarSeccionMutation.isPending}
                className="flex items-center gap-1.5"
              >
                <ArrowLeftIcon className="size-4" aria-hidden="true" />
                <span>Anterior</span>
              </Button>
            )}
          </div>

          <div className="flex items-center gap-2">
            {!noModificable && (
              <>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => guardarSeccionActual(false)}
                  disabled={
                    guardarSeccionMutation.isPending ||
                    bloqueadoNacionalidad ||
                    !abiertaConvocatoria
                  }
                  className="flex items-center gap-1.5"
                >
                  <SaveIcon className="size-4" aria-hidden="true" />
                  <span>
                    {guardarSeccionMutation.isPending
                      ? "Guardando…"
                      : "Guardar borrador"}
                  </span>
                </Button>
              </>
            )}

            {numeroSeccion < 9 ? (
              <Button
                type="button"
                size="sm"
                onClick={() => guardarSeccionActual(true)}
                disabled={
                  guardarSeccionMutation.isPending ||
                  bloqueadoNacionalidad ||
                  !abiertaConvocatoria
                }
                className="flex items-center gap-1.5"
              >
                <span>Siguiente</span>
                <ArrowRightIcon className="size-4" aria-hidden="true" />
              </Button>
            ) : (
              <Button
                type="button"
                size="sm"
                onClick={() => guardarSeccionActual(true)}
                disabled={
                  guardarSeccionMutation.isPending ||
                  bloqueadoNacionalidad ||
                  !abiertaConvocatoria
                }
                className="flex items-center gap-1.5 bg-emerald-700 text-white hover:bg-emerald-800"
              >
                <CheckCircleIcon className="size-4" aria-hidden="true" />
                <span>Ir a Revisión y Envío</span>
              </Button>
            )}
          </div>
        </CardFooter>
      </Card>
    </div>
  )
}
