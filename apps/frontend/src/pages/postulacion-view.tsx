import {
  ArrowLeftIcon,
  ArrowRightIcon,
  CheckCircleIcon,
  FileCheck2Icon,
  SaveIcon,
  ShieldCheckIcon,
} from "lucide-react"
import { useEffect, useMemo, useState } from "react"
import { Link, useNavigate, useParams } from "react-router"
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
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  BarraProgresoSecciones,
  PUNTOS_CONSENTIMIENTO_EF12,
  Seccion1Personal,
  Seccion2Contacto,
  Seccion3Adicional,
  Seccion4Cargo,
  Seccion5Academica,
  Seccion6Experiencia,
  Seccion7Publicaciones,
  Seccion8Capacitacion,
  Seccion9Inhabilidades,
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
import { RUTAS, rutaSeccion } from "@/lib/rutas"
import { validarSeccionEF12 } from "@/utils/validar-formulario"

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

  // Crear borrador automático si el postulante aún no tiene ninguno
  useEffect(() => {
    if (
      !cargandoPostulaciones &&
      postulacionesData &&
      postulacionesData.items.length === 0 &&
      !crearPostulacionMutation.isPending &&
      !crearPostulacionMutation.isSuccess
    ) {
      crearPostulacionMutation.mutate({})
    }
  }, [cargandoPostulaciones, postulacionesData, crearPostulacionMutation])

  // Contenido local del formulario sincronizado con la postulación
  const [formData, setFormData] = useState<FormularioPostulacionState>(
    FORMULARIO_BASE_DEFAULT,
  )
  const [bloqueadoNacionalidad, setBloqueadoNacionalidad] = useState(false)
  const [errorValidacion, setErrorValidacion] = useState<string | null>(null)
  const [dialogoGuardarAbierto, setDialogoGuardarAbierto] = useState(false)

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

  const yaEnviada = postulacionActiva && postulacionActiva.status !== "draft"

  async function guardarSeccionActual(irSiguiente = false): Promise<boolean> {
    if (!postulacionActiva?.id) return false

    setErrorValidacion(null)
    const claveSeccion = `seccion${numeroSeccion}` as keyof typeof formData
    const contenidoSeccion = formData[claveSeccion]

    // Validación pura
    const resultado = validarSeccionEF12(contenidoSeccion, numeroSeccion, false)
    if (!resultado.esValido && resultado.errores.length > 0) {
      const primerError = resultado.errores[0].mensaje
      setErrorValidacion(primerError)
      toast.warning(`Atención en sección ${numeroSeccion}: ${primerError}`)
    }

    try {
      await guardarSeccionMutation.mutateAsync({
        id: postulacionActiva.id,
        seccion: numeroSeccion,
        datos: contenidoSeccion,
      })

      if (irSiguiente) {
        if (numeroSeccion < 9) {
          navegar(rutaSeccion(numeroSeccion + 1))
        } else {
          navegar(RUTAS.postulacionRevision)
        }
      }
      return true
    } catch {
      return false
    }
  }

  return (
    <div className="mx-auto flex w-full max-w-5xl flex-1 flex-col gap-6 px-4 py-8">
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

      {/* Stepper y barra de progreso */}
      <BarraProgresoSecciones
        seccionActiva={numeroSeccion}
        seccionesCompletadas={seccionesCompletadas}
      />

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
              disabled={yaEnviada}
              onBloqueoNacionalidad={setBloqueadoNacionalidad}
            />
          )}

          {numeroSeccion === 2 && (
            <Seccion2Contacto
              datos={formData.seccion2 as Seccion2Data}
              onChange={(s2) =>
                setFormData((prev) => ({ ...prev, seccion2: s2 }))
              }
              disabled={yaEnviada}
            />
          )}

          {numeroSeccion === 3 && (
            <Seccion3Adicional
              datos={formData.seccion3 as Seccion3Data}
              onChange={(s3) =>
                setFormData((prev) => ({ ...prev, seccion3: s3 }))
              }
              disabled={yaEnviada}
            />
          )}

          {numeroSeccion === 4 && (
            <Seccion4Cargo
              datos={formData.seccion4 as Seccion4Data}
              onChange={(s4) =>
                setFormData((prev) => ({ ...prev, seccion4: s4 }))
              }
              disabled={yaEnviada}
            />
          )}

          {numeroSeccion === 5 && (
            <Seccion5Academica
              datos={formData.seccion5 as Seccion5Data}
              onChange={(s5) =>
                setFormData((prev) => ({ ...prev, seccion5: s5 }))
              }
              disabled={yaEnviada}
            />
          )}

          {numeroSeccion === 6 && (
            <Seccion6Experiencia
              datos={formData.seccion6 as Seccion6Data}
              onChange={(s6) =>
                setFormData((prev) => ({ ...prev, seccion6: s6 }))
              }
              disabled={yaEnviada}
            />
          )}

          {numeroSeccion === 7 && (
            <Seccion7Publicaciones
              datos={formData.seccion7 as Seccion7Data}
              onChange={(s7) =>
                setFormData((prev) => ({ ...prev, seccion7: s7 }))
              }
              disabled={yaEnviada}
            />
          )}

          {numeroSeccion === 8 && (
            <Seccion8Capacitacion
              datos={formData.seccion8 as Seccion8Data}
              onChange={(s8) =>
                setFormData((prev) => ({ ...prev, seccion8: s8 }))
              }
              disabled={yaEnviada}
            />
          )}

          {numeroSeccion === 9 && (
            <Seccion9Inhabilidades
              datos={formData.seccion9 as Seccion9Data}
              onChange={(s9) =>
                setFormData((prev) => ({ ...prev, seccion9: s9 }))
              }
              disabled={yaEnviada}
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
            {!yaEnviada && (
              <>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => guardarSeccionActual(false)}
                  disabled={
                    guardarSeccionMutation.isPending || bloqueadoNacionalidad
                  }
                  className="flex items-center gap-1.5"
                >
                  <SaveIcon className="size-4" aria-hidden="true" />
                  <span>
                    {guardarSeccionMutation.isPending
                      ? "Guardando…"
                      : "Guardar información"}
                  </span>
                </Button>

                <Button
                  type="button"
                  variant="secondary"
                  size="sm"
                  onClick={() => setDialogoGuardarAbierto(true)}
                  disabled={
                    guardarSeccionMutation.isPending || bloqueadoNacionalidad
                  }
                  className="flex items-center gap-1.5"
                >
                  <ShieldCheckIcon className="size-4" aria-hidden="true" />
                  <span>Guardar Postulación</span>
                </Button>
              </>
            )}

            {numeroSeccion < 9 ? (
              <Button
                type="button"
                size="sm"
                onClick={() => guardarSeccionActual(true)}
                disabled={
                  guardarSeccionMutation.isPending || bloqueadoNacionalidad
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
                  guardarSeccionMutation.isPending || bloqueadoNacionalidad
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

      {/* Diálogo de consentimiento obligatorio al Guardar Postulación (EF-12 1.10.1) */}
      <Dialog
        open={dialogoGuardarAbierto}
        onOpenChange={setDialogoGuardarAbierto}
      >
        <DialogContent className="max-w-2xl max-h-[85vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2 text-base font-bold">
              <ShieldCheckIcon className="size-5 text-primary" />
              <span>
                Guardar Postulación — Declaración Legal (EF-12 1.10.1)
              </span>
            </DialogTitle>
            <DialogDescription className="text-xs">
              Al guardar la postulación, se le informa el siguiente texto
              obligatorio y debe aceptar los términos de participación del
              concurso:
            </DialogDescription>
          </DialogHeader>

          <div className="flex flex-col gap-3 py-3 text-xs leading-relaxed text-foreground">
            {PUNTOS_CONSENTIMIENTO_EF12.map((p) => (
              <div key={p.id} className="rounded-md border bg-muted/20 p-3">
                <span className="font-semibold text-primary block mb-0.5">
                  Punto {p.numero}: {p.titulo}
                </span>
                <p className="text-muted-foreground">{p.texto}</p>
              </div>
            ))}
          </div>

          <DialogFooter className="flex-row items-center justify-end gap-2 pt-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => setDialogoGuardarAbierto(false)}
            >
              Cancelar
            </Button>
            <Button
              type="button"
              size="sm"
              onClick={async () => {
                setDialogoGuardarAbierto(false)
                await guardarSeccionActual(false)
              }}
              disabled={guardarSeccionMutation.isPending}
            >
              Aceptar y guardar postulación
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
