/**
 * Generación de cédulas válidas para las pruebas.
 *
 * El servidor valida la cédula con módulo 10 y además exige que el código de
 * provincia esté entre 1 y 24 (o 30) y que el tercer dígito sea menor a 6.
 * Una cédula aleatoria sin esas reglas nunca llega al correo: el registro se
 * rechaza antes de emitir el código.
 */

/**
 * Códigos de provincia admitidos por el validador.
 */
const PROVINCIAS = [
  "01",
  "02",
  "03",
  "04",
  "05",
  "06",
  "07",
  "08",
  "09",
  "10",
  "11",
  "12",
  "13",
  "14",
  "15",
  "16",
  "17",
  "18",
  "19",
  "20",
  "21",
  "22",
  "23",
  "24",
]

/**
 * Calcula el dígito verificador de una cédula de nueve dígitos.
 *
 * @param {string} base - Los primeros nueve dígitos.
 * @returns {string} La cédula completa de diez dígitos.
 */
export function cedulaDesde(base: string): string {
  let suma = 0
  for (let i = 0; i < 9; i += 1) {
    let digito = Number(base[i]) * (i % 2 === 0 ? 2 : 1)
    if (digito > 9) digito -= 9
    suma += digito
  }
  return base + String((10 - (suma % 10)) % 10)
}

/**
 * Genera una cédulaECUatoriana válida y distinta cada vez.
 *
 * @returns {string} Cédula de diez dígitos que el servidor acepta.
 */
export function generarCedulaValida(): string {
  const provincia = PROVINCIAS[Math.floor(Math.random() * PROVINCIAS.length)]
  const tercerDigito = String(Math.floor(Math.random() * 6))
  const resto = Array.from({ length: 6 }, () =>
    String(Math.floor(Math.random() * 10)),
  ).join("")
  return cedulaDesde(provincia + tercerDigito + resto)
}

/**
 * Genera un correo único para una prueba.
 *
 * @param {string} prefijo - Prefijo legible del correo.
 * @returns {string} Correo único.
 */
export function generarCorreo(prefijo: string): string {
  return `${prefijo}-${Date.now()}-${Math.floor(Math.random() * 100000)}@biess-test.local`
}
