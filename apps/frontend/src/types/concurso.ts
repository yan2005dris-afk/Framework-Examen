export type SectorPostulacion = "afiliados" | "jubilados" | "empleadores"

export type EstadoCivil =
  "Soltero/a" | "Casado/a" | "Unión de hecho" | "Divorciado/a" | "Viudo/a"

export type Genero = "Masculino" | "Femenino"

export type Nacionalidad = "Ecuatoriana" | "Extranjera"

export type NivelInstruccion =
  | "Tercer Nivel de Grado"
  | "Cuarto Nivel - Especialidad Académica"
  | "Cuarto Nivel - Maestría Académica"
  | "Cuarto Nivel PhD - Doctorado"

export type AreaEstudio =
  | "Finanzas"
  | "Mercado de Capitales"
  | "Administración de Empresas"
  | "Economía"
  | "Derecho o materias afines"

export type TipoExperiencia = "General" | "Dirección"

export type TipoCapacitacion = "Curso" | "Seminario" | "Taller"

export type TipoPublicacion = "Publicación" | "Trabajo de investigación"

export interface Familiar {
  id: string
  nombres: string
  parentesco: string
  institucion: "IESS" | "BIESS"
  area: string
  cargo: string
}

export interface TituloAcademico {
  id: string
  nivel: NivelInstruccion
  institucion: string
  tipo: "Nacional" | "Extranjera"
  titulo: string
  area: AreaEstudio
  registroSenescyt: string
  fechaRegistroSenescyt: string
  documentoPdf?: {
    nombre: string
    tamano: string
    fechaSubida?: string
  }
}

export interface ExperienciaLaboral {
  id: string
  tipo: TipoExperiencia
  cargo: string
  institucion: string
  fechaInicio: string
  fechaFin: string
  actualmenteLaborando?: boolean
  actividades: string
  /** RF 1.6.6: actividades relacionadas a (catálogo cerrado). */
  actividadRelacionada: AreaEstudio | ""
  descripcionFunciones: string
  duracionCalculada?: {
    anios: number
    meses: number
    dias: number
    texto: string
  }
  documentoPdf?: {
    nombre: string
    tamano: string
    fechaSubida?: string
  }
}

export interface PublicacionInvestigacion {
  id: string
  titulo: string
  tipo: TipoPublicacion
  medio: string
  fechaPublicacion: string
  relacionCampo: "Sí" | "No"
  documentoPdf?: {
    nombre: string
    tamano: string
    fechaSubida?: string
  }
}

export interface EventoCapacitacion {
  id: string
  tipo: TipoCapacitacion
  nombre: string
  institucion: string
  fechaInicio: string
  fechaFin: string
  horas: number
  documentoPdf?: {
    nombre: string
    tamano: string
    fechaSubida?: string
  }
}

export interface InhabilidadesRespuestas {
  q1: "Sí" | "No" | "" // Ecuatoriano y pleno goce de derechos
  q2: "Sí" | "No" | "" // Inhabilitado para ejercer comercio
  q3: "Sí" | "No" | "" // En mora instituciones del Estado o SB
  q4: "Sí" | "No" | "" // Vínculo con sist financiero/seguros/bolsa
  q5: "Sí" | "No" | "" // Funcionario/empleado IESS-BIESS
  q6: "Sí" | "No" | "" // Interés en aseguradoras o sist seg social
  q7: "Sí" | "No" | "" // Removido en últimos 5 años por Org Control
  q7Detalles?: {
    institucion: string
    cargo: string
    fechaFin: string
    motivo: string
  }
  q8: "Sí" | "No" | "" // Obligaciones pendientes SRI
  q9: "Sí" | "No" | "" // Castigo de obligaciones en 5 años
  q10: "Sí" | "No" | "" // Litigando contra IESS-BIESS
  q11: "Sí" | "No" | "" // Procesado/enjuiciado/condenado corrupción/crimen
  q12: "Sí" | "No" | "" // Responsabilidad en firme CGE
  q13: "Sí" | "No" | "" // Registrado base de datos UAFE
}

export interface PostulanteUser {
  cedula: string
  nombres: string
  apellidos: string
  correo: string
  fechaRegistro: string
  metodoValidacion: "fecha_expedicion" | "codigo_dactilar"
}

export interface FormularioPostulacionData {
  codigoPostulacion: string
  estado: "borrador" | "en_revision" | "enviado"
  fechaUltimaModificacion: string
  fechaEnvio?: string
  hashCriptografico?: string

  // Sección 1: Información personal
  seccion1: {
    nombres: string
    apellidos: string
    cedula: string
    genero: Genero | ""
    fechaNacimiento: string
    nacionalidad: Nacionalidad
    estadoCivil: EstadoCivil | ""
    nombreConyuge?: string
    cedulaConyuge?: string
    verificadoRegistroCivil: boolean
  }

  // Sección 2: Información domiciliaria y de contacto
  seccion2: {
    pais: string
    provincia: string
    canton: string
    telefonoCelular: string
    correo1: string
    correo2: string
    callePrincipal: string
    numero: string
    calleSecundaria: string
    sector: string
    referencia: string
    telefonoDomiciliario: string
  }

  // Sección 3: Información adicional
  seccion3: {
    tieneFamiliaresIessBiess: "Sí" | "No" | ""
    familiares: Familiar[]
  }

  // Sección 4: Selección del cargo al que postula
  seccion4: {
    sector: SectorPostulacion | ""
    verificacionIessCumplida: boolean
    detalleVerificacionIess: string
    cuentaConRespaldoOrganizacion: "Sí" | "No" | ""
    nombreOrganizacion: string
    documentoRespaldoPdf?: {
      nombre: string
      tamano: string
      fechaSubida: string
    }
  }

  // Sección 5: Formación académica
  seccion5: {
    titulos: TituloAcademico[]
  }

  // Sección 6: Experiencia profesional
  seccion6: {
    experiencias: ExperienciaLaboral[]
  }

  // Sección 7: Publicaciones e investigaciones
  seccion7: {
    cuentaConPublicaciones: "Sí" | "No" | ""
    publicaciones: PublicacionInvestigacion[]
  }

  // Sección 8: Capacitación
  seccion8: {
    capacitaciones: EventoCapacitacion[]
  }

  // Sección 9: Inhabilidades
  seccion9: InhabilidadesRespuestas

  // Declaraciones y consentimiento final
  consentimiento: {
    aceptacionTratamientoDatos: boolean
    declaracionVeracidad: boolean
    aceptacionNormasConcurso: boolean
    autorizacionPublicacionNombre: boolean
    constanciaEntregaFisica: boolean
    firmaManuscritaDataUrl: string
    fechaFirma: string
  }
}

export type SeccionFormularioId = 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9

export interface SeccionMeta {
  id: SeccionFormularioId
  numero: string
  titulo: string
  descripcionCorta: string
}
