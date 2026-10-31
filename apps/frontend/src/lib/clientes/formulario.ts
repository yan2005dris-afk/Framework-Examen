import { apiGet } from "../api"

export interface SeccionDefinicionEF12 {
  numero: number
  titulo: string
  obligatorios?: string[]
  opcionales?: string[]
  lista?: string
  registroObligatorios?: string[]
  condiciones?: string[]
}

export interface DefinicionFormularioEF12 {
  version: string
  opciones: Record<string, string[]>
  secciones: SeccionDefinicionEF12[]
  preguntasInhabilidades: { id: string; texto: string }[]
  provincias: Record<string, string[]>
  limites: Record<string, number>
  identidad: {
    metodo: string
    precargados: string[]
    verificacionRegistroCivil: boolean
  }
  consentimiento: {
    version: string
    puntos: string[]
  }
}

/** Obtiene catálogos, campos y literales del EF-12 con la sesión actual. */
export async function obtenerDefinicionFormulario(
  signal?: AbortSignal,
): Promise<DefinicionFormularioEF12> {
  return apiGet<DefinicionFormularioEF12>("/submissions/form-definition", {
    signal,
  })
}
