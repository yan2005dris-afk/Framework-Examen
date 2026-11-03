/**
 * Cálculo de experiencia profesional del EF-12 (sección 6).
 *
 * Vive en `helpers/` y no en `utils/` porque la duración de la experiencia
 * laboral es una regla de este concurso, no una utilidad genérica: la usan
 * el formulario, la revisión y la impresión del expediente.
 */

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
 * Cálculo de duración exacta entre dos fechas en Años, Meses y Días.
 *
 * @param fechaInicioStr - Fecha de inicio en formato "YYYY-MM-DD".
 * @param fechaFinStr - Fecha de fin; se ignora si `actualmenteLaborando` es `true`.
 * @param actualmenteLaborando - Indica que el período sigue vigente.
 * @returns Años, meses, días y su texto legible; "Fecha no válida" si el rango no se puede calcular.
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
