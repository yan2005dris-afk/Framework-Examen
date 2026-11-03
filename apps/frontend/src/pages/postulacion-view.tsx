import { ExcepcionPostulacion } from "@/features/formulario/excepcion-postulacion"
import {
  ArrowLeftIcon,
  ArrowRightIcon,
  CheckCircleIcon,
  FileCheck2Icon,
  LockIcon,
  SaveIcon,
} from "lucide-react"
import { useEffect, useMemo, useState } from "react"
import {
  Link,
  useLocation,
  useNavigate,
  useParams,
  useSearchParams,
} from "react-router"
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
  useEliminarPostulacion,
  useGuardarSeccion,
  usePostulacionActual,
} from "@/features/postulaciones/queries"
import { DescartarBorrador } from "@/features/postulaciones/descartar-borrador"
import { ListaMisPostulaciones } from "@/features/postulaciones/lista-mis-postulaciones"
import { convocatoriasAbiertas } from "@/features/convocatorias/modelo"
import { useConvocatoriasPublicas } from "@/features/convocatorias/queries"
import { SelectorConvocatoria } from "@/features/convocatorias/selector-convocatoria"
import { useAhora } from "@/features/convocatorias/use-ahora"
import { useDisponibilidadEF12 } from "@/features/convocatorias/use-disponibilidad"
import { esErrorApi } from "@/lib/api-error"
import { RUTAS, rutaPostulacion, rutaSeccion } from "@/lib/rutas"
import {
  aplicarValoresIniciales,
  referenciaDocumento,
} from "@/features/formulario/modelo"
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
  const { state: estadoNavegacion } = useLocation()
  const [parametros, setParametros] = useSearchParams()

  const numeroSeccion = seccion ? Number(seccion) : 1
  const esSeccionValida =
    Number.isInteger(numeroSeccion) && numeroSeccion >= 1 && numeroSeccion <= 9

  // Postulación de la convocatoria de la URL, o la única que tenga el postulante
  const {
    resolucion,
    postulacion: postulacionActiva,
    convocatoriaId: periodo,
    isLoading: cargandoPostulaciones,
    error: errorPostulaciones,
    refetch: recargarPostulaciones,
  } = usePostulacionActual()

  const crearPostulacionMutation = useCrearPostulacion()
  const guardarSeccionMutation = useGuardarSeccion()
  const eliminarPostulacionMutation = useEliminarPostulacion()

  const disponibilidad = useDisponibilidadEF12(periodo)
  const abiertaConvocatoria = disponibilidad.abierta

  // Sin postulación ni `?convocatoria=`: hay que elegir convocatoria. Con una
  // sola abierta se elige sola; con varias decide el postulante. Tras
  // descartar un borrador tampoco se elige sola, para no recrearlo.
  const listadoConvocatorias = useConvocatoriasPublicas()
  const ahora = useAhora()
  const convocatoriasPublicadas =
    listadoConvocatorias.data?.pages.flatMap((p) => p.items) ?? []
  const abiertas = convocatoriasAbiertas(convocatoriasPublicadas, ahora)
  const necesitaConvocatoria =
    resolucion?.estado === "sin_postulacion" && !parametros.get("convocatoria")
  const recienDescartado = Boolean(
    (estadoNavegacion as { borradorDescartado?: boolean } | null)
      ?.borradorDescartado,
  )
  const convocatoriaAutomatica =
    necesitaConvocatoria && !recienDescartado && abiertas.length === 1
      ? abiertas[0].id
      : undefined

  function elegirConvocatoria(convocatoriaId: string) {
    setParametros(
      (previos) => {
        const siguientes = new URLSearchParams(previos)
        siguientes.set("convocatoria", convocatoriaId)
        return siguientes
      },
      { replace: true },
    )
  }

  useEffect(() => {
    if (!convocatoriaAutomatica) return
    setParametros(
      (previos) => {
        const siguientes = new URLSearchParams(previos)
        siguientes.set("convocatoria", convocatoriaAutomatica)
        return siguientes
      },
      { replace: true },
    )
  }, [convocatoriaAutomatica, setParametros])

  // Crear el borrador cuando la URL indica una convocatoria abierta sin postulación.
  // El servidor garantiza una sola por convocatoria: si otra pestaña ya la creó
  // responde 409 y se abre la existente.
  const convocatoriaSinPostulacion =
    resolucion?.estado === "sin_postulacion"
      ? resolucion.convocatoriaId
      : undefined
  useEffect(() => {
    if (
      convocatoriaSinPostulacion &&
      abiertaConvocatoria &&
      crearPostulacionMutation.isIdle
    ) {
      crearPostulacionMutation.mutate({
        convocatoriaId: convocatoriaSinPostulacion,
      })
    }
  }, [
    convocatoriaSinPostulacion,
    abiertaConvocatoria,
    crearPostulacionMutation,
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
        ...(aplicarValoresIniciales(
          postulacionActiva.content,
        ) as unknown as Partial<FormularioPostulacionState>),
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

  /**
   * Antes de guardar la sección para subir un respaldo se aplica la misma
   * validación que «Guardar»: un dato inválido se informa aquí, con un mensaje
   * concreto, en vez de llegar al servidor y bloquear la subida.
   */
  function validarAntesDeSubir(numero: number, contenido: unknown): boolean {
    const resultado = validarSeccionEF12(contenido, numero, false)
    if (resultado.esValido) return true
    const mensaje =
      resultado.errores[0]?.mensaje ?? "Revise los datos de la sección."
    setErrorValidacion(mensaje)
    toast.warning(`Atención en sección ${numero}: ${mensaje}`)
    return false
  }

  async function subirRespaldoSeccion4(archivo: File) {
    if (!postulacionActiva?.id) return
    if (!validarAntesDeSubir(4, formData.seccion4)) return
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
          documentoRespaldoPdf: referenciaDocumento(doc),
        },
      }))
    } catch {
      // La mutación ya mostró el error en un toast; aquí sólo se evita la promesa rechazada.
    } finally {
      setCargandoDocId(null)
    }
  }

  async function eliminarRespaldoSeccion4() {
    const docId = formData.seccion4.documentoRespaldoPdf?.id
    if (docId) {
      try {
        await eliminarDocMutation.mutateAsync(docId)
      } catch {
        // El servidor rechazó la baja (ya informada en un toast): el adjunto se conserva.
        return
      }
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
    if (!validarAntesDeSubir(5, formData.seccion5)) return
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
                  documentoPdf: referenciaDocumento(doc),
                }
              : t,
          ),
        },
      }))
    } catch {
      // La mutación ya mostró el error en un toast; aquí sólo se evita la promesa rechazada.
    } finally {
      setCargandoDocId(null)
    }
  }

  async function eliminarRespaldoTitulo(tituloId: string) {
    const t = formData.seccion5.titulos.find((x) => x.id === tituloId)
    if (t?.documentoPdf?.id) {
      try {
        await eliminarDocMutation.mutateAsync(t.documentoPdf.id)
      } catch {
        // El servidor rechazó la baja (ya informada en un toast): el adjunto se conserva.
        return
      }
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
    if (!validarAntesDeSubir(6, formData.seccion6)) return
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
                  documentoPdf: referenciaDocumento(doc),
                }
              : e,
          ),
        },
      }))
    } catch {
      // La mutación ya mostró el error en un toast; aquí sólo se evita la promesa rechazada.
    } finally {
      setCargandoDocId(null)
    }
  }

  async function eliminarRespaldoExperiencia(expId: string) {
    const e = formData.seccion6.experiencias.find((x) => x.id === expId)
    if (e?.documentoPdf?.id) {
      try {
        await eliminarDocMutation.mutateAsync(e.documentoPdf.id)
      } catch {
        // El servidor rechazó la baja (ya informada en un toast): el adjunto se conserva.
        return
      }
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
    if (!validarAntesDeSubir(7, formData.seccion7)) return
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
                  documentoPdf: referenciaDocumento(doc),
                }
              : p,
          ),
        },
      }))
    } catch {
      // La mutación ya mostró el error en un toast; aquí sólo se evita la promesa rechazada.
    } finally {
      setCargandoDocId(null)
    }
  }

  async function eliminarRespaldoPublicacion(pubId: string) {
    const p = formData.seccion7.publicaciones.find((x) => x.id === pubId)
    if (p?.documentoPdf?.id) {
      try {
        await eliminarDocMutation.mutateAsync(p.documentoPdf.id)
      } catch {
        // El servidor rechazó la baja (ya informada en un toast): el adjunto se conserva.
        return
      }
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
    if (!validarAntesDeSubir(8, formData.seccion8)) return
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
                  documentoPdf: referenciaDocumento(doc),
                }
              : c,
          ),
        },
      }))
    } catch {
      // La mutación ya mostró el error en un toast; aquí sólo se evita la promesa rechazada.
    } finally {
      setCargandoDocId(null)
    }
  }

  async function eliminarRespaldoCapacitacion(capId: string) {
    const c = formData.seccion8.capacitaciones.find((x) => x.id === capId)
    if (c?.documentoPdf?.id) {
      try {
        await eliminarDocMutation.mutateAsync(c.documentoPdf.id)
      } catch {
        // El servidor rechazó la baja (ya informada en un toast): el adjunto se conserva.
        return
      }
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

  /** Descarta el borrador y vuelve a la elección de convocatoria sin recrearlo. */
  async function descartarBorrador() {
    if (!postulacionActiva) return
    await eliminarPostulacionMutation.mutateAsync(postulacionActiva.id)
    crearPostulacionMutation.reset()
    setPrevId(null)
    setFormData(FORMULARIO_BASE_DEFAULT)
    navegar(RUTAS.postulacion, {
      replace: true,
      state: { borradorDescartado: true },
    })
  }

  // Un 409 «postulacion_existente» no es un fallo: se abre la postulación existente.
  const errorCreacion =
    crearPostulacionMutation.isError &&
    !(
      esErrorApi(crearPostulacionMutation.error) &&
      crearPostulacionMutation.error.codigo === "postulacion_existente"
    )
  if (!cargandoPostulaciones && !postulacionActiva && errorCreacion) {
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

  if (resolucion?.estado === "elegir") {
    return (
      <ListaMisPostulaciones
        postulaciones={resolucion.postulaciones}
        titulos={Object.fromEntries(
          convocatoriasPublicadas.map((c) => [c.id, c.titulo]),
        )}
        onAbrir={elegirConvocatoria}
      />
    )
  }

  if (necesitaConvocatoria) {
    if (listadoConvocatorias.isPending || convocatoriaAutomatica) {
      return (
        <div className="mx-auto w-full max-w-4xl px-4 py-12">
          <EstadoCargando
            mensaje="Consultando las convocatorias abiertas…"
            variante="pantalla-completa"
          />
        </div>
      )
    }
    if (listadoConvocatorias.isError) {
      return (
        <div className="mx-auto w-full max-w-4xl px-4 py-12">
          <EstadoError
            error={listadoConvocatorias.error}
            titulo="No se pudieron consultar las convocatorias"
            onReintentar={() => void listadoConvocatorias.refetch()}
          />
        </div>
      )
    }
    if (abiertas.length > 0) {
      return (
        <SelectorConvocatoria
          convocatorias={abiertas}
          onElegir={elegirConvocatoria}
        />
      )
    }
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
                ? "La convocatoria indicada no tiene la recepción abierta. No se crea un borrador ni se envían cambios mientras no se confirme el período."
                : "No hay convocatorias con la recepción de postulaciones abierta en este momento."}
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
              El formulario de postulación tiene 9 secciones numeradas del 1 al
              9. La sección «{seccion}» no existe.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button asChild variant="outline">
              <Link to={rutaPostulacion(RUTAS.postulacion, periodo)}>
                Abrir el formulario
              </Link>
            </Button>
          </CardContent>
        </Card>
      </div>
    )
  }

  // Mientras se crea el borrador o se vuelve a consultar tras crearlo (o tras un 409)
  const esperandoBorrador =
    !postulacionActiva &&
    Boolean(convocatoriaSinPostulacion) &&
    abiertaConvocatoria &&
    !errorCreacion
  if (
    cargandoPostulaciones ||
    crearPostulacionMutation.isPending ||
    esperandoBorrador
  ) {
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
          navegar(rutaPostulacion(rutaSeccion(numeroSeccion + 1), periodo))
        } else {
          navegar(rutaPostulacion(RUTAS.postulacionRevision, periodo))
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

  if (estaCerrada) {
    const evidencia = postulacionActiva.elegibilidad
    const nacionalidad =
      evidencia?.nacionalidadDeclarada ?? formData.seccion1.nacionalidad
    const porNacionalidad =
      Boolean(evidencia?.codigo?.toLowerCase().includes("nacionalidad")) ||
      (Boolean(nacionalidad) && nacionalidad !== "Ecuatoriana")
    return (
      <div className="mx-auto w-full max-w-[46rem] px-5 pt-8 pb-16">
        <ExcepcionPostulacion
          tipo={porNacionalidad ? "nacionalidad" : "sector"}
          cerrada
          valor={
            porNacionalidad
              ? nacionalidad
              : (evidencia?.sector ?? formData.seccion4.sector)
          }
          mensaje={
            evidencia?.mensaje ??
            "El servidor confirmó el cierre de esta postulación."
          }
          origen={
            evidencia?.origen === "simulacion_controlada_servidor"
              ? "Simulación controlada del servidor"
              : "Verificación del servidor"
          }
        />
      </div>
    )
  }

  return (
    <div className="mx-auto flex w-full max-w-6xl flex-1 flex-col gap-6 px-5 pt-8 pb-16">
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
              <Link to={rutaPostulacion(RUTAS.postulacionRevision, periodo)}>
                Ver revisión
              </Link>
            </Button>
            <Button asChild size="sm">
              <Link to={rutaPostulacion(RUTAS.postulacionImprimir, periodo)}>
                Imprimir
              </Link>
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

      {/* Convocatoria de la postulación y descarte del borrador */}
      {postulacionActiva && (
        <div className="flex flex-wrap items-center justify-between gap-3 rounded-md border bg-muted/20 px-3.5 py-2 text-xs">
          <div className="flex flex-col gap-0.5">
            <span className="font-medium text-foreground">
              {disponibilidad.data?.titulo ?? "Convocatoria"}
            </span>
            <span className="text-muted-foreground">
              Código {postulacionActiva.code}
            </span>
          </div>
          {postulacionActiva.status === "draft" && (
            <DescartarBorrador
              convocatoria={disponibilidad.data?.titulo ?? "esta convocatoria"}
              pendiente={eliminarPostulacionMutation.isPending}
              onConfirmar={descartarBorrador}
            />
          )}
        </div>
      )}

      <h1 className="sr-only">Formulario de postulación</h1>
      <div className="grid items-start gap-8 min-[60rem]:grid-cols-[17rem_minmax(0,1fr)]">
        {/* Stepper lateral; en móvil se desplaza horizontalmente sin tapar el formulario. */}
        <BarraProgresoSecciones
          seccionActiva={numeroSeccion}
          seccionesCompletadas={seccionesCompletadas}
          convocatoriaId={periodo}
        />

        <div className="min-w-0 space-y-5">
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
          <Card>
            <CardHeader>
              <div className="flex flex-wrap items-center justify-between gap-2 text-xs font-semibold">
                <span className="rounded-md bg-brand-100 px-2.5 py-1 text-primary">
                  Sección {numeroSeccion} de 9
                </span>
                <span className="rounded-md bg-muted px-2.5 py-1 text-muted-foreground">
                  {yaEnviada
                    ? "Enviada · Sólo lectura"
                    : estaCerrada
                      ? "Cerrada"
                      : "Borrador"}
                </span>
              </div>
            </CardHeader>
            <CardContent className="pt-0">
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
                  fechaEnvio={postulacionActiva?.submittedAt}
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
                    onClick={() =>
                      navegar(
                        rutaPostulacion(
                          rutaSeccion(numeroSeccion - 1),
                          periodo,
                        ),
                      )
                    }
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
      </div>
    </div>
  )
}
