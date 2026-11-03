import { calcularDuracionExperiencia } from "@/helpers/experiencia"
import {
  formatearTamanoBytes,
  type DocumentoDto,
} from "@/lib/clientes/documentos"
import type { AdjuntoInfo } from "./adjunto-uploader"
import type { ConsentimientoEF12Data } from "./constantes"

/**
 * Referencia que el formulario guarda tras subir un respaldo PDF. Contiene
 * sólo los campos del contrato `DocumentReference` (`definition.json`): un
 * campo adicional hace que la validación local y el servidor rechacen la
 * sección y el postulante no pueda continuar.
 * @param documento - Documento confirmado por el servidor tras la subida
 * @returns Referencia lista para guardar en la sección
 */
export function referenciaDocumento(documento: DocumentoDto): AdjuntoInfo {
  return {
    id: documento.id,
    nombre: documento.name,
    tamano: formatearTamanoBytes(documento.sizeBytes),
    sha256: documento.sha256,
    fechaSubida: documento.createdAt,
  }
}

/**
 * Valores que el formulario ofrece por defecto. Se escriben en los datos (no
 * sólo en pantalla) para que lo que el postulante ve sea lo que se guarda.
 */
const VALORES_INICIALES: Record<string, Record<string, string>> = {
  seccion1: { nacionalidad: "Ecuatoriana" },
  seccion2: { pais: "Ecuador" },
}

/**
 * Completa los campos con valor por defecto que llegan vacíos al abrir un
 * borrador. El servidor crea la sección 2 sólo con el correo, así que su país
 * llega vacío. Nunca reemplaza un valor ya elegido.
 * @param contenido - Contenido de la postulación devuelto por el servidor
 * @returns Copia del contenido con los valores iniciales aplicados
 */
export function aplicarValoresIniciales(
  contenido: Record<string, unknown>,
): Record<string, unknown> {
  const resultado: Record<string, unknown> = { ...contenido }
  for (const [seccion, valores] of Object.entries(VALORES_INICIALES)) {
    const datos = resultado[seccion]
    if (datos === null || typeof datos !== "object" || Array.isArray(datos))
      continue
    const completa: Record<string, unknown> = { ...datos }
    for (const [campo, valor] of Object.entries(valores)) {
      if (
        completa[campo] === undefined ||
        completa[campo] === null ||
        completa[campo] === ""
      )
        completa[campo] = valor
    }
    resultado[seccion] = completa
  }
  return resultado
}

/** Consentimiento tal como lo admite el contrato `Consent` del servidor al enviar. */
export interface ConsentimientoEnvio {
  aceptacionTratamientoDatos: boolean
  declaracionVeracidad: boolean
  aceptacionNormasConcurso: boolean
  autorizacionPublicacionNombre: boolean
  constanciaEntregaFisica: boolean
  politicaPrivacidadId?: string
  hashPoliticaPrivacidad?: string
}

/**
 * Arma el consentimiento que se envía al sellar la postulación: los cinco
 * puntos y, si hay política vigente, su ID y hash. El servidor rechaza campos
 * fuera del contrato (400 `cuerpo_invalido`) y completa por sí mismo versión,
 * textos y fecha de aceptación, así que no se envían.
 * @param consentimiento - Estado de los cinco puntos en la pantalla
 * @param politica - Política de privacidad vigente de la convocatoria, si existe
 * @returns Consentimiento con sólo los campos del contrato
 */
export function consentimientoParaEnvio(
  consentimiento: ConsentimientoEF12Data,
  politica?: { id: string; hash: string },
): ConsentimientoEnvio {
  return {
    aceptacionTratamientoDatos: consentimiento.aceptacionTratamientoDatos,
    declaracionVeracidad: consentimiento.declaracionVeracidad,
    aceptacionNormasConcurso: consentimiento.aceptacionNormasConcurso,
    autorizacionPublicacionNombre: consentimiento.autorizacionPublicacionNombre,
    constanciaEntregaFisica: consentimiento.constanciaEntregaFisica,
    ...(politica
      ? {
          politicaPrivacidadId: politica.id,
          hashPoliticaPrivacidad: politica.hash,
        }
      : {}),
  }
}

/** Confirma un registro del modal sin mutar la lista ni pisar una edición concurrente. */
export function confirmarEdicionRegistro<T extends { id: string }>(
  lista: readonly T[],
  borrador: T,
  original: T | null,
): T[] {
  const actual = lista.find((item) => item.id === borrador.id)
  if (
    original
      ? !actual || JSON.stringify(actual) !== JSON.stringify(original)
      : Boolean(actual)
  ) {
    throw new Error(
      "El registro cambió. Cierre el diálogo y vuelva a abrirlo antes de guardar.",
    )
  }
  return original
    ? lista.map((item) =>
        item.id === borrador.id ? structuredClone(borrador) : item,
      )
    : [...lista, structuredClone(borrador)]
}

/** Conserva la duración a la fecha de envío; nunca usa el reloj actual para una experiencia vigente sellada. */
export function duracionExperienciaImprimible(
  registro: {
    fechaInicio: string
    fechaFin?: string
    actualmenteLaborando?: boolean
  },
  enviadaEn?: string,
): string {
  let fin = registro.fechaFin
  if (registro.actualmenteLaborando) {
    if (!enviadaEn || !Number.isFinite(Date.parse(enviadaEn)))
      return "Fecha de envío no disponible"
    const partes = new Intl.DateTimeFormat("es-EC", {
      timeZone: "America/Guayaquil",
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
    }).formatToParts(new Date(enviadaEn))
    const valor = (tipo: string) =>
      partes.find((parte) => parte.type === tipo)?.value
    fin = `${valor("year")}-${valor("month")}-${valor("day")}`
  }
  return registro.fechaInicio && fin
    ? calcularDuracionExperiencia(registro.fechaInicio, fin, false).texto
    : "Fechas pendientes"
}
