import {
  SECCIONES_EF12_INFO,
  PREGUNTAS_INHABILIDADES,
} from "@/features/formulario/constantes"
import { duracionHistorica, esObjetoRevision } from "./modelo"

const ETIQUETAS: Record<string, string> = {
  cedula: "Cédula",
  genero: "Género",
  fechaNacimiento: "Fecha de nacimiento",
  estadoCivil: "Estado civil",
  nombreConyuge: "Nombre del cónyuge",
  cedulaConyuge: "Cédula del cónyuge",
  pais: "País",
  canton: "Cantón",
  telefonoCelular: "Teléfono celular",
  correo1: "Correo principal",
  correo2: "Correo adicional",
  telefonoDomiciliario: "Teléfono domiciliario",
  direccion: "Dirección",
  numero: "Número",
  tieneFamiliaresIessBiess: "¿Tiene familiares en IESS / BIESS?",
  area: "Área",
  institucion: "Institución",
  verificadoRegistroCivil:
    "Indicador legado de Registro Civil (declarado, no certifica verificación)",
  verificacionIessCumplida: "Indicador de verificación IESS declarado",
  detalleVerificacionIess: "Detalle declarado sobre IESS",
  documentoRespaldoPdf: "Respaldo de organización",
  documentoPdf: "Respaldo PDF",
  documentoId: "ID del documento",
  registroSenescyt: "Registro SENESCYT",
  fechaRegistroSenescyt: "Fecha de registro SENESCYT",
  titulos: "Títulos",
  titulo: "Título",
  duracionCalculada: "Duración almacenada al sello",
  anios: "Años",
  dias: "Días",
  descripcionFunciones: "Descripción de funciones",
  aceptacionTratamientoDatos: "Aceptación del tratamiento de datos",
  declaracionVeracidad: "Declaración de veracidad",
  aceptacionNormasConcurso: "Aceptación de normas",
  autorizacionPublicacionNombre: "Autorización de publicación del nombre",
  constanciaEntregaFisica: "Constancia de entrega física",
  firmaManuscritaDataUrl: "Firma manuscrita",
  versionTextos: "Versión de los textos consentidos",
  version: "Versión del consentimiento",
  puntos: "Literales aceptados al enviar",
  aceptadoEn: "Aceptado en",
  textosAceptados: "Textos aceptados al envío",
  ...Object.fromEntries(
    PREGUNTAS_INHABILIDADES.map((p) => [p.id, `${p.numero} · ${p.texto}`]),
  ),
}

function etiqueta(clave: string) {
  return (
    ETIQUETAS[clave] ??
    clave
      .replace(/([a-záéíóúñ])([A-ZÁÉÍÓÚÑ])/g, "$1 $2")
      .replace(/^./, (c) => c.toUpperCase())
  )
}

function ValorSellado({
  valor,
  campo = "",
  enviadoEn,
}: {
  valor: unknown
  campo?: string
  enviadoEn: string
}) {
  if (
    campo === "firmaManuscritaDataUrl" &&
    typeof valor === "string" &&
    valor
  ) {
    // Solo imágenes raster embebidas; nunca se cargan URLs arbitrarias del postulante.
    return /^data:image\/(png|jpeg);base64,[a-z0-9+/=\r\n]+$/i.test(valor) &&
      valor.length < 3_000_000 ? (
      <img
        src={valor}
        alt="Firma manuscrita conservada en el envío"
        className="max-h-40 max-w-full rounded border bg-white"
      />
    ) : (
      <span>Firma registrada en formato legado no visualizable.</span>
    )
  }
  if (valor === null || valor === undefined || valor === "")
    return <span className="text-muted-foreground">No consta en el envío</span>
  if (typeof valor === "boolean") return <span>{valor ? "Sí" : "No"}</span>
  if (Array.isArray(valor))
    return valor.length === 0 ? (
      <span className="text-muted-foreground">Sin registros</span>
    ) : (
      <ol className="space-y-4">
        {valor.map((item, indice) => (
          <li
            key={
              esObjetoRevision(item) && typeof item.id === "string"
                ? item.id
                : indice
            }
            className="space-y-3 rounded-lg border p-4"
          >
            <p className="text-xs font-medium text-muted-foreground">
              Registro {indice + 1}
            </p>
            {campo === "experiencias" && esObjetoRevision(item) && (
              <p className="rounded-md bg-muted p-3 text-sm font-medium">
                Duración: {duracionHistorica(item, enviadoEn)}
              </p>
            )}
            <ValorSellado valor={item} enviadoEn={enviadoEn} />
          </li>
        ))}
      </ol>
    )
  if (esObjetoRevision(valor))
    return (
      <dl className="grid min-w-0 gap-4 sm:grid-cols-2">
        {Object.entries(valor).map(([clave, dato]) => (
          <div
            key={clave}
            className={
              typeof dato === "object" && dato !== null
                ? "min-w-0 space-y-2 sm:col-span-2"
                : "min-w-0 space-y-1"
            }
          >
            <dt className="text-xs font-medium text-muted-foreground">
              {etiqueta(clave)}
            </dt>
            <dd className="whitespace-pre-wrap break-words text-sm">
              <ValorSellado valor={dato} campo={clave} enviadoEn={enviadoEn} />
            </dd>
          </div>
        ))}
      </dl>
    )
  return <span>{String(valor)}</span>
}

/** Representa exclusivamente el JSON sellado; no mezcla perfil, catálogos ni valores del formulario vigente. */
export function ExpedienteSellado({
  contenido,
  enviadoEn,
}: {
  contenido: Record<string, unknown>
  enviadoEn: string
}) {
  const adicionales = Object.fromEntries(
    Object.entries(contenido).filter(
      ([k]) => !/^seccion[1-9]$/.test(k) && k !== "consentimiento",
    ),
  )
  return (
    <section className="space-y-3" aria-label="EF-12 íntegro de solo lectura">
      {SECCIONES_EF12_INFO.map((s) => (
        <details
          key={s.numero}
          className="group rounded-xl border bg-card p-5"
          open={s.numero === 1}
        >
          <summary className="cursor-pointer font-semibold">
            {s.numero}. {s.titulo}
          </summary>
          <div className="mt-5">
            <ValorSellado
              valor={contenido[`seccion${s.numero}`]}
              enviadoEn={enviadoEn}
            />
          </div>
        </details>
      ))}
      <details className="rounded-xl border bg-card p-5">
        <summary className="cursor-pointer font-semibold">
          Declaraciones, consentimiento y firma
        </summary>
        <div className="mt-5 space-y-4">
          <p className="text-xs text-muted-foreground">
            Se muestran los valores y textos guardados en este envío. No se
            sustituyen por la versión actual del consentimiento.
          </p>
          <ValorSellado
            valor={contenido.consentimiento}
            enviadoEn={enviadoEn}
          />
        </div>
      </details>
      {Object.keys(adicionales).length > 0 && (
        <details className="rounded-xl border bg-card p-5">
          <summary className="cursor-pointer font-semibold">
            Otros datos conservados en el envío
          </summary>
          <div className="mt-5">
            <ValorSellado valor={adicionales} enviadoEn={enviadoEn} />
          </div>
        </details>
      )}
    </section>
  )
}
