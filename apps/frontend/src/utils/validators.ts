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
