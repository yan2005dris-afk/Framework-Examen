import {
  ArrowRightIcon,
  CalendarDaysIcon,
  ExternalLinkIcon,
  FileCheck2Icon,
  InfoIcon,
  LockKeyholeIcon,
  ShieldCheckIcon,
  UsersIcon,
} from "lucide-react"
import { Link, useSearchParams } from "react-router"
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
import { useSesion } from "@/features/auth/sesion-context"
import { useRutaInicial } from "@/features/auth/use-ruta-inicial"
import { useConvocatoriasPublicas } from "@/features/convocatorias/queries"
import {
  ETIQUETAS_RECEPCION,
  SECTORES,
  estadoFase,
  estadoRecepcion,
} from "@/features/convocatorias/modelo"
import { useDisponibilidadEF12 } from "@/features/convocatorias/use-disponibilidad"
import { useAhora } from "@/features/convocatorias/use-ahora"
import { RUTAS, construirRutaLogin } from "@/lib/rutas"
import { formatearFechaEcuador } from "@/utils/fechas"

/** Portal FT14: composición del mockup 01, con cronograma y disponibilidad reales, nunca fechas ilustrativas. */
export function PortadaView() {
  const [parametros, setParametros] = useSearchParams()
  const listado = useConvocatoriasPublicas()
  const items = listado.data?.pages.flatMap((p) => p.items) ?? []
  const id =
    parametros.get("convocatoria") ||
    items.find((c) => c.ef12Disponible)?.id ||
    items[0]?.id
  const consulta = useDisponibilidadEF12(id)
  const convocatoria = consulta.data
  const ahora = useAhora()
  const { estado: sesion } = useSesion()
  const { ruta } = useRutaInicial()
  const estado = estadoRecepcion(
    convocatoria,
    ahora,
    !consulta.isError &&
      !listado.isError &&
      consulta.estadoRecepcion !== "sin_confirmar",
  )
  const abierta = estado === "abierta"
  const destino = `${RUTAS.postulacion}${id ? `?convocatoria=${encodeURIComponent(id)}` : ""}`
  const rutaCuenta = ruta === RUTAS.postulacion ? destino : ruta
  const sectores = convocatoria?.sectores ?? []
  const cargando = listado.isPending || (Boolean(id) && consulta.isPending)
  const error = consulta.error || listado.error

  return (
    <div>
      <section
        className="border-b bg-brand-900 text-background"
        aria-labelledby="titulo-concurso"
      >
        <div className="mx-auto max-w-6xl space-y-6 px-4 py-10 sm:px-6 md:py-16">
          <div className="flex flex-wrap items-center gap-3">
            <span className="text-xs font-semibold tracking-widest uppercase">
              Directorio del BIESS · Concurso público
            </span>
            {convocatoria && (
              <span className="rounded-full border border-current/40 px-3 py-1 text-xs font-medium">
                {ETIQUETAS_RECEPCION[estado]}
              </span>
            )}
          </div>
          <h1
            id="titulo-concurso"
            className="max-w-4xl text-3xl font-semibold tracking-tight sm:text-4xl md:text-5xl"
          >
            {convocatoria?.titulo ?? "Concurso Público de Méritos y Oposición"}
          </h1>
          <p className="max-w-3xl text-base leading-relaxed opacity-90 sm:text-lg">
            Selección de los Miembros Principales y Alternos del Directorio del
            BIESS, en representación de Afiliados, Jubilados y Empleadores.
          </p>
          <div className="flex flex-wrap gap-3">
            {sesion === "autenticado" && rutaCuenta ? (
              <Button asChild size="lg" variant="secondary">
                <Link to={rutaCuenta}>
                  Ir a mi espacio <ArrowRightIcon aria-hidden="true" />
                </Link>
              </Button>
            ) : (
              <>
                {abierta ? (
                  <Button asChild size="lg" variant="secondary">
                    <Link
                      to={`${RUTAS.registro}?convocatoria=${encodeURIComponent(id!)}`}
                    >
                      Iniciar postulación <ArrowRightIcon aria-hidden="true" />
                    </Link>
                  </Button>
                ) : (
                  <Button size="lg" variant="secondary" disabled>
                    Postulaciones no disponibles
                  </Button>
                )}
                <Button
                  asChild
                  size="lg"
                  variant="outline"
                  className="border-current/50 bg-transparent text-inherit hover:bg-background/10 hover:text-inherit"
                >
                  <Link to={construirRutaLogin(destino)}>Ya tengo cuenta</Link>
                </Button>
              </>
            )}
          </div>
          <p className="text-sm opacity-85">
            Registro con cédula y verificación de correo mediante OTP. Los demás
            datos personales son declarados.
          </p>
        </div>
      </section>

      <div className="mx-auto max-w-6xl space-y-8 px-4 py-8 sm:px-6 md:space-y-10 md:py-10">
        <section aria-labelledby="convocatoria-publicada" className="space-y-4">
          <div className="flex flex-wrap items-end justify-between gap-4">
            <div>
              <h2 id="convocatoria-publicada" className="text-xl font-semibold">
                Convocatoria y cronograma
              </h2>
              <p className="mt-1 text-sm text-muted-foreground">
                Fechas de Ecuador · America/Guayaquil (UTC−5). Consulta
                actualizada cada 30 segundos.
              </p>
            </div>
            {items.length > 0 && (
              <div className="w-full space-y-1.5 sm:w-auto sm:max-w-sm">
                <label
                  htmlFor="convocatoria-publica"
                  className="text-sm font-medium"
                >
                  Período publicado
                </label>
                <select
                  id="convocatoria-publica"
                  className="h-10 w-full rounded-md border bg-background px-3 text-sm"
                  value={id ?? ""}
                  onChange={(e) =>
                    setParametros({ convocatoria: e.target.value })
                  }
                >
                  {id && !items.some((c) => c.id === id) && (
                    <option value={id}>
                      {convocatoria?.codigo ?? "Convocatoria del enlace"}
                    </option>
                  )}
                  {items.map((c) => (
                    <option key={c.id} value={c.id}>
                      {c.codigo} · {c.titulo}
                    </option>
                  ))}
                </select>
              </div>
            )}
          </div>
          {cargando && (
            <EstadoCargando mensaje="Consultando la convocatoria publicada…" />
          )}
          {error && (
            <EstadoError
              error={error}
              titulo="No podemos confirmar la disponibilidad"
              onReintentar={() => {
                void listado.refetch()
                void consulta.refetch()
              }}
            />
          )}
          {!cargando && !error && !id && (
            <Alert>
              <InfoIcon aria-hidden="true" />
              <AlertTitle>Sin convocatorias publicadas</AlertTitle>
              <AlertDescription>
                Cuando exista una convocatoria aprobada y publicada, su
                calendario aparecerá aquí. Puede iniciar sesión para consultar
                un expediente anterior.
              </AlertDescription>
            </Alert>
          )}
          {convocatoria && (
            <>
              {estado !== "abierta" && (
                <Alert>
                  <LockKeyholeIcon aria-hidden="true" />
                  <AlertTitle>{ETIQUETAS_RECEPCION[estado]}</AlertTitle>
                  <AlertDescription>
                    {estado === "programada"
                      ? "La recepción aún no ha iniciado. Revise las fechas de la fase de postulación."
                      : estado === "cerrada"
                        ? "La recepción de esta convocatoria finalizó. Sus expedientes enviados conservan su código y sello."
                        : estado === "sin_confirmar"
                          ? "No se habilitan nuevas postulaciones mientras no se confirme la disponibilidad con el servidor."
                          : "El servidor no tiene habilitada la recepción de EF-12 para esta convocatoria. Puede consultar la información publicada y sus expedientes anteriores."}
                  </AlertDescription>
                </Alert>
              )}
              <Card>
                <CardContent className="grid gap-5 pt-6 sm:grid-cols-3">
                  <div>
                    <p className="text-xs text-muted-foreground">
                      Código del período
                    </p>
                    <p className="mt-1 font-mono text-sm font-semibold">
                      {convocatoria.codigo}
                    </p>
                  </div>
                  <div>
                    <p className="text-xs text-muted-foreground">
                      Apertura del período
                    </p>
                    <p className="mt-1 text-sm font-medium">
                      {formatearFechaEcuador(convocatoria.apertura, {
                        incluirHora: true,
                      })}
                    </p>
                  </div>
                  <div>
                    <p className="text-xs text-muted-foreground">
                      Cierre del período
                    </p>
                    <p className="mt-1 text-sm font-medium">
                      {formatearFechaEcuador(convocatoria.cierre, {
                        incluirHora: true,
                      })}
                    </p>
                  </div>
                </CardContent>
              </Card>
              <ol
                className="grid gap-3 md:grid-cols-2"
                aria-label="Fases publicadas"
              >
                {convocatoria.fases
                  .filter((f) => f.publicada)
                  .map((f) => (
                    <li
                      key={f.codigo}
                      className="rounded-lg border bg-card p-5"
                    >
                      <div className="mb-3 flex items-start justify-between gap-3">
                        <h3 className="text-sm font-semibold">{f.titulo}</h3>
                        <Badge variant="outline" className="shrink-0">
                          {estadoFase(f, ahora)}
                        </Badge>
                      </div>
                      <p className="text-sm text-muted-foreground">
                        {formatearFechaEcuador(f.apertura, {
                          incluirHora: true,
                        })}
                        <br />
                        <span className="text-xs">hasta</span>{" "}
                        {formatearFechaEcuador(f.cierre, { incluirHora: true })}
                      </p>
                    </li>
                  ))}
              </ol>
              {!convocatoria.fases.some((f) => f.publicada) && (
                <p className="text-sm text-muted-foreground">
                  Aún no se han publicado fases para este período.
                </p>
              )}
              <p className="text-xs text-muted-foreground">
                El cierre es exclusivo: al llegar la hora indicada ya no se
                admiten cambios. El servidor verifica cada operación. Una fase
                habilitada no equivale a una calificación ni a un resultado
                publicado.
              </p>
            </>
          )}
          {listado.hasNextPage && (
            <Button
              variant="outline"
              disabled={listado.isFetchingNextPage}
              onClick={() => void listado.fetchNextPage()}
            >
              {listado.isFetchingNextPage
                ? "Cargando…"
                : "Ver más períodos publicados"}
            </Button>
          )}
        </section>

        <section aria-labelledby="sectores" className="space-y-4">
          <div>
            <h2 id="sectores" className="text-xl font-semibold">
              Sectores de representación
            </h2>
            <p className="mt-1 text-sm text-muted-foreground">
              Se muestran únicamente los sectores de la convocatoria
              seleccionada.
            </p>
          </div>
          <div className="grid gap-4 md:grid-cols-3">
            {sectores.map((sector) => (
              <Card key={sector}>
                <CardHeader>
                  <UsersIcon
                    className="mb-1 size-5 text-primary"
                    aria-hidden="true"
                  />
                  <Badge variant="outline" className="w-fit">
                    {SECTORES[sector]}
                  </Badge>
                  <CardTitle className="mt-2">
                    1 principal y 1 alterno
                  </CardTitle>
                </CardHeader>
                <CardContent className="text-sm leading-relaxed text-muted-foreground">
                  {sector === "afiliados"
                    ? "Representación de los afiliados al IESS."
                    : sector === "jubilados"
                      ? "Representación de las personas jubiladas."
                      : "Representación del sector empleador."}{" "}
                  La condición y los respaldos se revisan conforme a las bases
                  del concurso.
                </CardContent>
              </Card>
            ))}
          </div>
        </section>

        <section aria-labelledby="ruta-postulacion" className="space-y-4">
          <h2 id="ruta-postulacion" className="text-xl font-semibold">
            Su ruta de postulación
          </h2>
          <ol className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {[
              [
                "Cree su cuenta",
                "Valide su cédula y confirme el acceso a su correo con un código OTP.",
              ],
              [
                "Complete el EF-12",
                "Guarde su avance en nueve secciones y adjunte sus respaldos PDF.",
              ],
              [
                "Revise y envíe",
                "Revise sus datos y acepte las cinco declaraciones antes de sellar el envío.",
              ],
              [
                "Imprima y entregue",
                "Conserve su código y hash. Entregue la copia con firma manuscrita según las bases.",
              ],
            ].map(([titulo, texto], i) => (
              <li key={titulo} className="rounded-lg border bg-card p-5">
                <span className="font-mono text-xs text-muted-foreground">
                  PASO 0{i + 1}
                </span>
                <h3 className="mt-3 text-sm font-semibold">{titulo}</h3>
                <p className="mt-2 text-sm leading-relaxed text-muted-foreground">
                  {texto}
                </p>
              </li>
            ))}
          </ol>
        </section>

        <div className="grid gap-5 md:grid-cols-2">
          <Card>
            <CardHeader>
              <FileCheck2Icon
                className="size-5 text-primary"
                aria-hidden="true"
              />
              <CardTitle>Antes de comenzar</CardTitle>
              <CardDescription>
                Prepare la información de su expediente.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <ul className="list-disc space-y-2 pl-5 text-sm text-muted-foreground">
                <li>Un correo personal al que pueda acceder.</li>
                <li>
                  Datos de formación y experiencia, con sus respaldos PDF.
                </li>
                <li>
                  Las bases, requisitos y fechas publicados por la institución.
                </li>
              </ul>
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <ShieldCheckIcon
                className="size-5 text-primary"
                aria-hidden="true"
              />
              <CardTitle>Canales institucionales</CardTitle>
              <CardDescription>
                Consulte las publicaciones oficiales en IESS y BIESS.
              </CardDescription>
            </CardHeader>
            <CardContent className="flex flex-wrap gap-3">
              <Button asChild variant="outline">
                <a
                  href="https://www.iess.gob.ec"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  Portal IESS <ExternalLinkIcon aria-hidden="true" />
                  <span className="sr-only">(nueva pestaña)</span>
                </a>
              </Button>
              <Button asChild variant="outline">
                <a
                  href="https://www.biess.fin.ec"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  Portal BIESS <ExternalLinkIcon aria-hidden="true" />
                  <span className="sr-only">(nueva pestaña)</span>
                </a>
              </Button>
            </CardContent>
          </Card>
        </div>
        <p className="flex items-start gap-2 border-t pt-5 text-xs leading-relaxed text-muted-foreground">
          <CalendarDaysIcon
            className="mt-0.5 size-4 shrink-0"
            aria-hidden="true"
          />
          La publicación de un enlace en portales externos requiere coordinación
          institucional. Este portal no afirma haber realizado esa publicación.
        </p>
      </div>
    </div>
  )
}
