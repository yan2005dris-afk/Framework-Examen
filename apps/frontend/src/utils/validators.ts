/**
 * Validador oficial del número de cédula de identidad del Ecuador (Algoritmo Módulo 10)
 */
export function validarCedulaEcuatoriana(cedula: string): {
  esValida: boolean
  error?: string
} {
  const clean = cedula.trim().replace(/\D/g, "")

  if (clean.length !== 10) {
    return {
      esValida: false,
      error: "La cédula debe contener exactamente 10 dígitos numéricos.",
    }
  }

  const provincia = parseInt(clean.substring(0, 2), 10)
  if ((provincia < 1 || provincia > 24) && provincia !== 30) {
    return {
      esValida: false,
      error: "El código de provincia (dos primeros dígitos) no es válido.",
    }
  }

  const tercerDigito = parseInt(clean.charAt(2), 10)
  if (tercerDigito >= 6) {
    return {
      esValida: false,
      error: "El tercer dígito para personas naturales debe ser menor a 6.",
    }
  }

  const coeficientes = [2, 1, 2, 1, 2, 1, 2, 1, 2]
  let suma = 0

  for (let i = 0; i < coeficientes.length; i++) {
    let valor = parseInt(clean.charAt(i), 10) * coeficientes[i]
    if (valor >= 10) {
      valor -= 9
    }
    suma += valor
  }

  const digitoVerificador = suma % 10 === 0 ? 0 : 10 - (suma % 10)
  const ultimoDigito = parseInt(clean.charAt(9), 10)

  if (digitoVerificador !== ultimoDigito) {
    return {
      esValida: false,
      error:
        "El dígito verificador no coincide con el algoritmo del Registro Civil.",
    }
  }

  return { esValida: true }
}

/**
 * Convierte una fecha "YYYY-MM-DD" en una fecha local. `new Date("2020-01-01")`
 * se interpreta como UTC y, en Ecuador (UTC-5), al leer las partes locales la
 * fecha se corre un día. El cálculo de experiencia debe trabajar siempre con
 * fechas locales.
 */
function parsearFechaLocal(valor: string): Date | null {
  const soloFecha = /^(\d{4})-(\d{2})-(\d{2})$/.exec(valor.trim())
  if (soloFecha) {
    const fecha = new Date(
      Number(soloFecha[1]),
      Number(soloFecha[2]) - 1,
      Number(soloFecha[3]),
    )
    return fecha.getFullYear() === Number(soloFecha[1]) &&
      fecha.getMonth() === Number(soloFecha[2]) - 1 &&
      fecha.getDate() === Number(soloFecha[3])
      ? fecha
      : null
  }

  const fecha = new Date(valor)
  return isNaN(fecha.getTime()) ? null : fecha
}

/**
 * Cálculo de duración exacta entre dos fechas en Años, Meses y Días
 */
export function calcularDuracionExperiencia(
  fechaInicioStr: string,
  fechaFinStr?: string,
  actualmenteLaborando?: boolean,
) {
  if (!fechaInicioStr) {
    return { anios: 0, meses: 0, dias: 0, texto: "0 días" }
  }

  const inicio = parsearFechaLocal(fechaInicioStr)
  const fin =
    actualmenteLaborando || !fechaFinStr
      ? new Date()
      : parsearFechaLocal(fechaFinStr)

  if (!inicio || !fin || fin < inicio) {
    return { anios: 0, meses: 0, dias: 0, texto: "Fecha no válida" }
  }

  let anios = fin.getFullYear() - inicio.getFullYear()
  let meses = fin.getMonth() - inicio.getMonth()
  let dias = fin.getDate() - inicio.getDate()

  for (let prestados = 0; dias < 0; prestados++) {
    meses -= 1
    // Días del mes anterior
    const prevMonth = new Date(fin.getFullYear(), fin.getMonth() - prestados, 0)
    dias += prevMonth.getDate()
  }

  if (meses < 0) {
    anios -= 1
    meses += 12
  }

  const partes = []
  if (anios > 0) partes.push(`${anios} ${anios === 1 ? "año" : "años"}`)
  if (meses > 0) partes.push(`${meses} ${meses === 1 ? "mes" : "meses"}`)
  if (dias > 0 || partes.length === 0)
    partes.push(`${dias} ${dias === 1 ? "día" : "días"}`)

  return {
    anios,
    meses,
    dias,
    texto: partes.join(", "),
  }
}

/**
 * Evalúa la fortaleza de una contraseña
 */
export function evaluarFortalezaPassword(pwd: string): {
  nivel: "muy_debil" | "debil" | "media" | "fuerte"
  etiqueta: string
  porcentaje: number
  color: string
  cumpleLongitud: boolean
  cumpleMayuscula: boolean
  cumpleMinuscula: boolean
  cumpleNumero: boolean
  cumpleEspecial: boolean
} {
  const cumpleLongitud = pwd.length >= 8
  const cumpleMayuscula = /[A-Z]/.test(pwd)
  const cumpleMinuscula = /[a-z]/.test(pwd)
  const cumpleNumero = /[0-9]/.test(pwd)
  const cumpleEspecial = /[^A-Za-z0-9]/.test(pwd)

  const score = [
    cumpleLongitud,
    cumpleMayuscula,
    cumpleMinuscula,
    cumpleNumero,
    cumpleEspecial,
  ].filter(Boolean).length

  if (score <= 1) {
    return {
      nivel: "muy_debil",
      etiqueta: "Muy débil",
      porcentaje: 20,
      color: "bg-red-500",
      cumpleLongitud,
      cumpleMayuscula,
      cumpleMinuscula,
      cumpleNumero,
      cumpleEspecial,
    }
  } else if (score === 2 || score === 3) {
    return {
      nivel: "debil",
      etiqueta: "Débil",
      porcentaje: 45,
      color: "bg-amber-500",
      cumpleLongitud,
      cumpleMayuscula,
      cumpleMinuscula,
      cumpleNumero,
      cumpleEspecial,
    }
  } else if (score === 4) {
    return {
      nivel: "media",
      etiqueta: "Aceptable",
      porcentaje: 75,
      color: "bg-blue-600",
      cumpleLongitud,
      cumpleMayuscula,
      cumpleMinuscula,
      cumpleNumero,
      cumpleEspecial,
    }
  } else {
    return {
      nivel: "fuerte",
      etiqueta: "Óptima y segura",
      porcentaje: 100,
      color: "bg-emerald-600",
      cumpleLongitud,
      cumpleMayuscula,
      cumpleMinuscula,
      cumpleNumero,
      cumpleEspecial,
    }
  }
}

/**
 * Validador de formato de correo
 */
export function validarEmail(email: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)
}
