import {
  formatearTamanoBytes,
  type DocumentoDto,
} from "@/lib/clientes/documentos"
import type { AdjuntoInfo } from "./adjunto-uploader"

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
