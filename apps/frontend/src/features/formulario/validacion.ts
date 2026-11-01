import definicion from "../../../../backend/internal/submission/definition.json"
import { PROVINCIAS_ECUADOR } from "@/data/ecuadorGeo"
import { validarCedulaEcuatoriana, validarEmail } from "@/utils/validators"

type Registro = Record<string, unknown>
interface TipoCampo {
  tipo: string
  estructura?: string
  anulable?: boolean
}

export interface ErrorFormulario {
  campo: string
  mensaje: string
}

export interface ResultadoFormulario {
  esValido: boolean
  errores: ErrorFormulario[]
}

const tipos: Record<string, Record<string, TipoCampo>> = definicion.tipos
const opciones: Record<string, string[]> = definicion.opciones
const texto = (registro: Registro, campo: string): string =>
  typeof registro[campo] === "string" ? registro[campo] : ""
const objeto = (valor: unknown): valor is Registro =>
  valor !== null && typeof valor === "object" && !Array.isArray(valor)

/** Replica los tipos JSON del contrato Go antes de aplicar reglas del EF-12. */
function normalizarRegistro(valor: unknown, estructura: string): Registro {
  if (!objeto(valor)) throw new Error("estructura")
  const campos = tipos[estructura]
  if (!campos) throw new Error("estructura")
  const salida: Registro = {}
  for (const [nombre, entrada] of Object.entries(valor)) {
    const campo = campos[nombre]
    if (!campo) throw new Error("campo desconocido")
    if (entrada === null) {
      salida[nombre] =
        campo.anulable || campo.tipo === "lista" || campo.tipo === "objeto"
          ? null
          : campo.tipo === "texto"
            ? ""
            : campo.tipo === "booleano"
              ? false
              : 0
    } else if (campo.tipo === "texto" && typeof entrada === "string") {
      salida[nombre] = entrada
    } else if (campo.tipo === "booleano" && typeof entrada === "boolean") {
      salida[nombre] = entrada
    } else if (
      campo.tipo === "numero" &&
      typeof entrada === "number" &&
      Number.isInteger(entrada) &&
      entrada >= -(2 ** 63) &&
      entrada < 2 ** 63
    ) {
      salida[nombre] = entrada
    } else if (campo.tipo === "objeto" && campo.estructura) {
      salida[nombre] = normalizarRegistro(entrada, campo.estructura)
    } else if (campo.tipo === "lista" && Array.isArray(entrada)) {
      salida[nombre] = entrada.map((elemento: unknown) => {
        if (campo.estructura === "string") {
          if (elemento === null) return ""
          if (typeof elemento !== "string") throw new Error("tipo")
          return elemento
        }
        return normalizarRegistro(elemento ?? {}, campo.estructura ?? "")
      })
    } else throw new Error("tipo")
  }
  return salida
}

/** Valida fechas civiles sin normalizar días imposibles ni depender del huso del navegador. */
function fechaValida(valor: string, hoy: string): boolean {
  if (!valor) return true
  if (!/^\d{4}-\d{2}-\d{2}$/.test(valor)) return false
  const fecha = new Date(`${valor}T00:00:00Z`)
  return (
    !Number.isNaN(fecha.getTime()) &&
    fecha.toISOString().slice(0, 10) === valor &&
    valor <= hoy
  )
}

/** Aplica las reglas de una sección; los documentos reales se comprueban en T4. */
function errorSeccion(
  formulario: Registro,
  numero: number,
  completo: boolean,
  hoy: string,
): string | null {
  const entrada = formulario[`seccion${numero}`]
  if (!objeto(entrada)) return completo ? "La sección es obligatoria." : null
  const seccion = entrada
  const meta = definicion.secciones.find((seccion) => seccion.numero === numero)
  if (!meta) return "La sección no existe."
  const requeridos = (registro: Registro, campos: string[]): boolean =>
    campos.every((campo) => {
      const valor = texto(registro, campo)
      return (!completo || valor.trim() !== "") && !valor.includes("\0")
    })
  const opcion = (
    registro: Registro,
    campo: string,
    catalogo: string,
  ): boolean =>
    texto(registro, campo) === "" ||
    opciones[catalogo].includes(texto(registro, campo))
  const fecha = (registro: Registro, campo: string): boolean =>
    fechaValida(texto(registro, campo), hoy)
  const cedula = (valor: string): boolean =>
    !valor ||
    (/^\d{10}$/.test(valor) && validarCedulaEcuatoriana(valor).esValida)
  const rango = (registro: Registro, actual = false): boolean => {
    const inicio = texto(registro, "fechaInicio")
    const fin = texto(registro, "fechaFin")
    return (
      fechaValida(inicio, hoy) &&
      (actual
        ? !fin
        : fechaValida(fin, hoy) && (!inicio || !fin || fin >= inicio))
    )
  }
  if (!requeridos(seccion, meta.obligatorios ?? []))
    return "Complete los campos obligatorios."
  const valorLista = meta.lista ? seccion[meta.lista] : undefined
  const registros = Array.isArray(valorLista) ? valorLista.filter(objeto) : []
  const ids = new Set<string>()
  for (const registro of registros) {
    const id = texto(registro, "id")
    if (
      (!id && completo) ||
      (id &&
        (id.trim() !== id ||
          [...id].length > 255 ||
          /\p{Cc}/u.test(id) ||
          ids.has(id)))
    ) {
      return "Cada registro requiere un identificador válido y único."
    }
    if (id) ids.add(id)
  }
  switch (numero) {
    case 1: {
      const nacionalidad = texto(seccion, "nacionalidad")
      if (nacionalidad && nacionalidad !== "Ecuatoriana")
        return "La nacionalidad debe ser ecuatoriana."
      if (
        !opcion(seccion, "genero", "generos") ||
        !opcion(seccion, "estadoCivil", "estadosCiviles") ||
        !cedula(texto(seccion, "cedula")) ||
        !fecha(seccion, "fechaNacimiento")
      )
        return "Los datos personales no son válidos."
      if (
        ["Casado/a", "Unión de hecho"].includes(
          texto(seccion, "estadoCivil"),
        ) &&
        (!requeridos(seccion, ["nombreConyuge", "cedulaConyuge"]) ||
          !cedula(texto(seccion, "cedulaConyuge")))
      )
        return "Complete los datos válidos de la pareja."
      break
    }
    case 2: {
      const pais = texto(seccion, "pais"),
        provincia = texto(seccion, "provincia"),
        canton = texto(seccion, "canton")
      if (!opcion(seccion, "pais", "paises"))
        return "Seleccione un país del catálogo."
      if (
        [pais, provincia, canton].some(
          (lugar) =>
            [...lugar].length > 120 || !/^[\p{L}\p{N} .,'()-]*$/u.test(lugar),
        )
      )
        return "La ubicación no es válida."
      if ((provincia && !pais) || (canton && !provincia))
        return "Seleccione primero país y provincia."
      if (
        pais === "Ecuador" &&
        ((provincia && !Object.hasOwn(PROVINCIAS_ECUADOR, provincia)) ||
          (canton && !PROVINCIAS_ECUADOR[provincia]?.includes(canton)))
      )
        return "La provincia o el cantón no corresponde."
      if (
        ["correo1", "correo2"].some(
          (campo) =>
            texto(seccion, campo) && !validarEmail(texto(seccion, campo)),
        )
      )
        return "El correo no es válido."
      if (
        ["telefonoCelular", "telefonoDomiciliario"].some(
          (campo) =>
            texto(seccion, campo) &&
            !/^\+?[0-9 ()-]{7,20}$/.test(texto(seccion, campo)),
        )
      )
        return "El teléfono no es válido."
      break
    }
    case 3:
      if (
        !opcion(seccion, "tieneFamiliaresIessBiess", "respuestas") ||
        (texto(seccion, "tieneFamiliaresIessBiess") === "No" &&
          registros.length) ||
        (completo &&
          texto(seccion, "tieneFamiliaresIessBiess") === "Sí" &&
          !registros.length)
      )
        return "Declare correctamente los familiares."
      if (
        registros.some(
          (r) =>
            !requeridos(r, meta.registroObligatorios ?? []) ||
            !opcion(r, "parentesco", "parentescos") ||
            !opcion(r, "institucion", "instituciones"),
        )
      )
        return "Complete los datos válidos de cada familiar."
      break
    case 4:
      if (
        !opcion(seccion, "sector", "sectores") ||
        !opcion(seccion, "cuentaConRespaldoOrganizacion", "respuestas")
      )
        return "Seleccione un sector y respuesta válidos."
      if (
        (seccion.verificacionIessCumplida === false &&
          texto(seccion, "sector")) ||
        (completo && seccion.verificacionIessCumplida !== true)
      )
        return "Debe completar la elegibilidad del sector."
      if (
        texto(seccion, "cuentaConRespaldoOrganizacion") === "Sí" &&
        !requeridos(seccion, ["nombreOrganizacion"])
      )
        return "Indique el nombre de la organización."
      break
    case 5:
      if (completo && !registros.length) return "Agregue un título."
      if (
        registros.some(
          (r) =>
            !requeridos(r, meta.registroObligatorios ?? []) ||
            !opcion(r, "nivel", "niveles") ||
            !opcion(r, "tipo", "tiposInstitucion") ||
            !opcion(r, "area", "areas") ||
            !fecha(r, "fechaRegistroSenescyt"),
        )
      )
        return "La formación académica no es válida."
      break
    case 6:
      if (completo && !registros.length) return "Agregue una experiencia."
      if (
        registros.some(
          (r) =>
            !requeridos(r, meta.registroObligatorios ?? []) ||
            (r.actualmenteLaborando !== true && !requeridos(r, ["fechaFin"])) ||
            !opcion(r, "tipo", "tiposExperiencia") ||
            !opcion(r, "actividades", "areas") ||
            [...texto(r, "descripcionFunciones")].length > 1000 ||
            !rango(r, r.actualmenteLaborando === true),
        )
      )
        return "La experiencia profesional no es válida."
      break
    case 7:
      if (
        !opcion(seccion, "cuentaConPublicaciones", "respuestas") ||
        (texto(seccion, "cuentaConPublicaciones") === "No" &&
          registros.length) ||
        (completo &&
          texto(seccion, "cuentaConPublicaciones") === "Sí" &&
          !registros.length)
      )
        return "Declare correctamente las publicaciones."
      if (
        registros.some(
          (r) =>
            !requeridos(r, meta.registroObligatorios ?? []) ||
            !opcion(r, "tipo", "tiposPublicacion") ||
            !opcion(r, "relacionCampo", "respuestas") ||
            !fecha(r, "fechaPublicacion"),
        )
      )
        return "La publicación no es válida."
      break
    case 8:
      if (completo && !Array.isArray(valorLista))
        return "Declare la lista de capacitaciones, aunque esté vacía."
      if (
        registros.some(
          (r) =>
            !requeridos(r, meta.registroObligatorios ?? []) ||
            !opcion(r, "tipo", "tiposEvento") ||
            (typeof r.horas === "number" && r.horas < 0) ||
            (completo && !(typeof r.horas === "number" && r.horas > 0)) ||
            !rango(r),
        )
      )
        return "La capacitación no es válida."
      break
    case 9: {
      if (
        definicion.preguntasInhabilidades.some(
          ({ id }) => !opcion(seccion, id, "respuestas"),
        )
      )
        return "Las trece respuestas deben ser Sí o No."
      const detalles = seccion.q7Detalles
      if (texto(seccion, "q7") === "Sí") {
        if (!objeto(detalles))
          return completo ? "Complete los detalles de remoción." : null
        if (
          !requeridos(detalles, [
            "institucion",
            "cargo",
            "fechaFin",
            "motivo",
          ]) ||
          !fecha(detalles, "fechaFin")
        )
          return "Los detalles de remoción no son válidos."
      } else if (objeto(detalles))
        return "Los detalles solo corresponden a una respuesta Sí."
      break
    }
  }
  return null
}

/**
 * Valida el contenido EF-12 con catálogos compartidos con Go, sin llamadas de red.
 * Los borradores permiten campos pendientes; completo exige nueve secciones y cinco aceptaciones.
 * Los documentos existentes, la cuenta y la elegibilidad certificada se comprueban en servidor.
 */
export function validarFormularioEF12(
  contenido: unknown,
  completo = false,
  ahora = new Date(),
): ResultadoFormulario {
  try {
    const formulario = normalizarRegistro(contenido, "Form")
    const hoy = new Date(ahora.getTime() - 5 * 60 * 60 * 1000)
      .toISOString()
      .slice(0, 10)
    const errores: ErrorFormulario[] = []
    for (let numero = 1; numero <= 9; numero++) {
      const mensaje = errorSeccion(formulario, numero, completo, hoy)
      if (mensaje) errores.push({ campo: `seccion${numero}`, mensaje })
    }
    const consentimiento = formulario.consentimiento
    const puntos = [
      "aceptacionTratamientoDatos",
      "declaracionVeracidad",
      "aceptacionNormasConcurso",
      "autorizacionPublicacionNombre",
      "constanciaEntregaFisica",
    ]
    if (
      completo &&
      (!objeto(consentimiento) ||
        puntos.some((punto) => consentimiento[punto] !== true))
    )
      errores.push({
        campo: "consentimiento",
        mensaje: "Debe aceptar los cinco puntos.",
      })
    return { esValido: errores.length === 0, errores }
  } catch {
    return {
      esValido: false,
      errores: [
        {
          campo: "contenido",
          mensaje: "El formulario contiene datos con un formato no válido.",
        },
      ],
    }
  }
}

/** Valida una sección aislada; sirve para Guardar progreso sin exigir las demás. */
export function validarSeccionEF12(
  contenido: unknown,
  numero: number,
  completo = false,
  ahora = new Date(),
): ResultadoFormulario {
  if (
    !Number.isInteger(numero) ||
    numero < 1 ||
    numero > 9 ||
    !objeto(contenido)
  )
    return {
      esValido: false,
      errores: [{ campo: "seccion", mensaje: "La sección no es válida." }],
    }
  const resultado = validarFormularioEF12(
    { [`seccion${numero}`]: contenido },
    completo,
    ahora,
  )
  const errores = resultado.errores.filter(
    ({ campo }) => campo === "contenido" || campo === `seccion${numero}`,
  )
  return { esValido: errores.length === 0, errores }
}

/**
 * Valida la cédula de la pareja; devuelve `null` si el campo está vacío.
 *
 * @param cedula - Cédula declarada por el postulante.
 * @returns El resultado del validador oficial, o `null` si no se informó.
 */
export function validarCedulaPareja(
  cedula: string | undefined,
): { esValida: boolean; error?: string } | null {
  return cedula ? validarCedulaEcuatoriana(cedula) : null
}

/**
 * Valida el correo de confirmación; devuelve `null` si el campo está vacío.
 *
 * @param correo - Correo de confirmación declarado por el postulante.
 * @returns El mensaje de error, o `null` si el correo es válido o está vacío.
 */
export function validarCorreoSecundario(
  correo: string | undefined,
): string | null {
  return correo && !validarEmail(correo) ? "Correo inválido" : null
}
