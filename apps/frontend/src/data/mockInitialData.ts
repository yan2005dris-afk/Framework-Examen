import type {
  FormularioPostulacionData,
  PostulanteUser,
} from "../types/concurso"

export const POSTULANTE_DEMO: PostulanteUser = {
  cedula: "1713849204",
  nombres: "CARLOS ALBERTO",
  apellidos: "MENDOZA ANDRADE",
  correo: "carlos.mendoza@postulante-biess.ec",
  fechaRegistro: "2026-10-18T09:30:00Z",
  metodoValidacion: "codigo_dactilar",
}

export const FORMULARIO_INICIAL_VACIO: FormularioPostulacionData = {
  codigoPostulacion: "EF12-2026-004821",
  estado: "borrador",
  fechaUltimaModificacion: new Date().toISOString(),
  seccion1: {
    nombres: "",
    apellidos: "",
    cedula: "",
    genero: "",
    fechaNacimiento: "",
    nacionalidad: "Ecuatoriana",
    estadoCivil: "",
    verificadoRegistroCivil: false,
  },
  seccion2: {
    pais: "Ecuador",
    provincia: "",
    canton: "",
    telefonoCelular: "",
    correo1: "",
    correo2: "",
    callePrincipal: "",
    numero: "",
    calleSecundaria: "",
    sector: "",
    referencia: "",
    telefonoDomiciliario: "",
  },
  seccion3: {
    tieneFamiliaresIessBiess: "",
    familiares: [],
  },
  seccion4: {
    sector: "",
    verificacionIessCumplida: false,
    detalleVerificacionIess: "",
    cuentaConRespaldoOrganizacion: "",
    nombreOrganizacion: "",
  },
  seccion5: {
    titulos: [],
  },
  seccion6: {
    experiencias: [],
  },
  seccion7: {
    cuentaConPublicaciones: "",
    publicaciones: [],
  },
  seccion8: {
    capacitaciones: [],
  },
  seccion9: {
    q1: "",
    q2: "",
    q3: "",
    q4: "",
    q5: "",
    q6: "",
    q7: "",
    q8: "",
    q9: "",
    q10: "",
    q11: "",
    q12: "",
    q13: "",
  },
  consentimiento: {
    aceptacionTratamientoDatos: false,
    declaracionVeracidad: false,
    aceptacionNormasConcurso: false,
    autorizacionPublicacionNombre: false,
    constanciaEntregaFisica: false,
    firmaManuscritaDataUrl: "",
    fechaFirma: "",
  },
}

export const FORMULARIO_DEMO_COMPLETO: FormularioPostulacionData = {
  codigoPostulacion: "EF12-2026-004821",
  estado: "borrador",
  fechaUltimaModificacion: "2026-10-30T16:45:00Z",
  seccion1: {
    nombres: "CARLOS ALBERTO",
    apellidos: "MENDOZA ANDRADE",
    cedula: "1713849204",
    genero: "Masculino",
    fechaNacimiento: "1985-04-17",
    nacionalidad: "Ecuatoriana",
    estadoCivil: "Casado/a",
    nombreConyuge: "MARÍA ELENA VARGAS",
    cedulaConyuge: "1712345675",
    verificadoRegistroCivil: true,
  },
  seccion2: {
    pais: "Ecuador",
    provincia: "Pichincha",
    canton: "Quito",
    telefonoCelular: "0991234567",
    correo1: "carlos.mendoza@postulante-biess.ec",
    correo2: "c.mendoza@correo.com",
    callePrincipal: "Av. Amazonas",
    numero: "N34-120",
    calleSecundaria: "Av. Naciones Unidas",
    sector: "La Carolina",
    referencia: "Junto al parque La Carolina",
    telefonoDomiciliario: "022345678",
  },
  seccion3: {
    tieneFamiliaresIessBiess: "Sí",
    familiares: [
      {
        id: "fam-1",
        nombres: "PATRICIA ELENA MENDOZA ANDRADE",
        parentesco: "Hermana (Segundo Grado Consanguinidad)",
        institucion: "IESS",
        area: "Coordinación Provincial de Afiliación Pichincha",
        cargo: "Analista de Cartera e Imposiciones",
      },
    ],
  },
  seccion4: {
    sector: "afiliados",
    verificacionIessCumplida: true,
    detalleVerificacionIess:
      "Acreditado: Afiliado activo bajo relación de dependencia con 184 imposiciones continuas no interrumpidas en el IESS.",
    cuentaConRespaldoOrganizacion: "Sí",
    nombreOrganizacion:
      "Confederación Ecuatoriana de Organizaciones Clasistas Unitarias de Trabajadores (CEDOCUT)",
    documentoRespaldoPdf: {
      nombre: "aval_gremial_cedocut_resolucion_2026.pdf",
      tamano: "2.8 MB",
      fechaSubida: "2026-10-28 11:20",
    },
  },
  seccion5: {
    titulos: [
      {
        id: "tit-1",
        nivel: "Tercer Nivel de Grado",
        institucion: "Pontificia Universidad Católica del Ecuador",
        tipo: "Nacional",
        titulo: "Economista con mención en Finanzas Públicas",
        area: "Economía",
        registroSenescyt: "1005-02-348912",
        fechaRegistroSenescyt: "2002-08-15",
        documentoPdf: {
          nombre: "titulo_economista_puce_registrado.pdf",
          tamano: "1.4 MB",
        },
      },
      {
        id: "tit-2",
        nivel: "Cuarto Nivel - Maestría Académica",
        institucion: "Universidad Andina Simón Bolívar - Sede Ecuador",
        tipo: "Nacional",
        titulo: "Magíster en Dirección Financiera y Mercado de Capitales",
        area: "Finanzas",
        registroSenescyt: "1008-09-781204",
        fechaRegistroSenescyt: "2009-11-20",
        documentoPdf: {
          nombre: "maestria_direccion_financiera_uasb.pdf",
          tamano: "1.8 MB",
        },
      },
    ],
  },
  seccion6: {
    experiencias: [
      {
        id: "exp-1",
        tipo: "Dirección",
        cargo: "Gerente Nacional de Inversiones y Riesgos",
        institucion: "Banco de Desarrollo del Ecuador B.P.",
        fechaInicio: "2018-02-01",
        fechaFin: "2023-04-30",
        actualmenteLaborando: false,
        actividadRelacionada: "Finanzas",
        actividades:
          "Gestión y estructuración de portafolios de crédito, administración de inversiones financieras de liquidez institucional, y mitigación de riesgos de mercado.",
        descripcionFunciones:
          "Lideré el comité técnico de inversiones administrando un portafolio de colocación superior a 850 millones de USD. Supervisé el cumplimiento de límites normativos ante la Superintendencia de Bancos y la ejecución de operaciones de tesorería y mercado de valores.",
        duracionCalculada: {
          anios: 5,
          meses: 2,
          dias: 29,
          texto: "5 años, 2 meses, 29 días",
        },
        documentoPdf: {
          nombre: "certificacion_laboral_gerencia_bde.pdf",
          tamano: "1.6 MB",
        },
      },
      {
        id: "exp-2",
        tipo: "Dirección",
        cargo: "Director de Planificación Financiera y Control de Gestión",
        institucion: "Fideicomiso Mercantil de Titularización Hipotecaria",
        fechaInicio: "2023-05-15",
        fechaFin: "",
        actualmenteLaborando: true,
        actividadRelacionada: "Economía",
        actividades:
          "Supervisión de patrimonios autónomos, estructuración de títulos valores hipotecarios y análisis de flujos de pago para inversionistas institucionales.",
        descripcionFunciones:
          "Dirección integral de las operaciones fiduciarias y colocaciones en el Catastro Público de Mercado de Valores. Monitoreo diario del rendimiento del fondo fiduciario y calificación de riesgo crediticio AAA.",
        duracionCalculada: {
          anios: 3,
          meses: 5,
          dias: 15,
          texto: "3 años, 5 meses, 15 días",
        },
        documentoPdf: {
          nombre: "acta_nombramiento_director_fideicomiso.pdf",
          tamano: "1.2 MB",
        },
      },
    ],
  },
  seccion7: {
    cuentaConPublicaciones: "Sí",
    publicaciones: [
      {
        id: "pub-1",
        titulo:
          "Optimización de Portafolios de Seguridad Social en Mercados Emergentes",
        tipo: "Publicación",
        medio:
          "Revista Latinoamericana de Economía y Finanzas Públicas (Indexada Scopus)",
        fechaPublicacion: "2021-06-18",
        relacionCampo: "Sí",
        documentoPdf: {
          nombre: "paper_optimizacion_portafolios_seguridad_social.pdf",
          tamano: "1.9 MB",
        },
      },
    ],
  },
  seccion8: {
    capacitaciones: [
      {
        id: "cap-1",
        tipo: "Curso",
        nombre:
          "Gobernanza Corporativa y Gestión de Riesgos en Entidades Financieras Públicas",
        institucion:
          "Banco Interamericano de Desarrollo (BID) - Washington D.C.",
        fechaInicio: "2022-03-01",
        fechaFin: "2022-05-15",
        horas: 120,
        documentoPdf: {
          nombre: "diploma_gobernanza_riesgos_bid.pdf",
          tamano: "0.9 MB",
        },
      },
      {
        id: "cap-2",
        tipo: "Seminario",
        nombre:
          "Marco Regulatorio y Supervisión Prudencial de la Seguridad Social",
        institucion: "Superintendencia de Bancos del Ecuador",
        fechaInicio: "2023-08-10",
        fechaFin: "2023-08-25",
        horas: 60,
        documentoPdf: {
          nombre: "certificado_superintendencia_bancos.pdf",
          tamano: "0.7 MB",
        },
      },
    ],
  },
  seccion9: {
    q1: "Sí", // Es ecuatoriano y pleno goce de derechos (Debe ser Sí)
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
  consentimiento: {
    aceptacionTratamientoDatos: true,
    declaracionVeracidad: true,
    aceptacionNormasConcurso: true,
    autorizacionPublicacionNombre: true,
    constanciaEntregaFisica: true,
    firmaManuscritaDataUrl:
      'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" width="300" height="80"><path d="M10 50 Q 50 10, 90 45 T 160 40 T 230 45 T 280 30" stroke="%230A3871" stroke-width="2.5" fill="none"/></svg>',
    fechaFirma: "2026-10-30T16:50:00Z",
  },
}
