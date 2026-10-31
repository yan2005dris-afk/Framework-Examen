import { apiGet, apiPost } from "../api"
import type { RespuestaMensaje } from "@/types/api"

/**
 * Rol de la cuenta dentro del portal.
 * - `postulante`: postulante al concurso EF-12.
 * - `analista`: revisor de postulaciones (Talento Humano).
 * - `administrador`: Comisión de Selección (gestión de convocatorias, fases y banco EF-14).
 */
export type RolPostulante = "postulante" | "analista" | "administrador"

/**
 * Datos del postulante devueltos por la sesión o el registro.
 */
export interface PostulanteDto {
  id: string
  cedula: string
  nombres: string
  apellidos: string
  nombre_completo: string
  correo: string
  metodo_validacion: string
  fecha_registro: string
  /**
   * Rol de la cuenta en el portal, según lo asigna el servidor.
   *
   * La API lo envía siempre en `/auth/session`, en el login y en el registro:
   * el registro público crea `"postulante"` y `"analista"` se asigna del lado
   * del servidor. El tipo lo deja opcional a propósito: si una respuesta vieja
   * o un proxy recorta el campo, la cuenta se trata como `"postulante"` y nunca
   * hereda permisos de analista.
   */
  rol?: RolPostulante
}

/**
 * Respuesta que contiene la entidad postulante autenticada.
 */
export interface RespuestaPostulante {
  postulante: PostulanteDto
}

/**
 * Desafío OTP emitido por el servidor.
 * El `desafioId` debe enviarse en el paso de verificación.
 * El cliente debe mostrar `expiraEn` tal como viene del servidor, sin recalcular.
 */
export interface DesafioOtp {
  /** Identificador opaco del desafío. Enviarlo en la verificación. */
  desafioId: string
  /** Instante UTC en que expira el código OTP. */
  expiraEn: Date
  /** Mensaje informativo del servidor. */
  mensaje: string
}

/**
 * Petición para solicitar código de registro (Paso 1).
 */
export interface SolicitudRegistroDto {
  cedula: string
  correo: string
  metodo_validacion?: "fecha_expedicion" | "codigo_dactilar" | "correo"
}

/**
 * Petición para verificar el código OTP y finalizar el registro (Paso 2/3).
 */
export interface VerificacionRegistroDto {
  cedula: string
  correo: string
  codigo_otp: string
  nombres: string
  apellidos: string
  password: string
}

/**
 * Credenciales para iniciar sesión.
 */
export interface LoginDto {
  cedula: string
  password: string
}

/**
 * Solicitud de código de recuperación de contraseña.
 */
export interface SolicitudRecuperacionDto {
  cedula: string
}

/**
 * Confirmación de código y nueva contraseña.
 */
export interface RestablecerPasswordDto {
  cedula: string
  codigo_otp: string
  new_password: string
}

/**
 * Solicita el envío de un código OTP al correo electrónico para iniciar el registro.
 * Devuelve el desafío OTP emitido por el servidor, incluyendo el `desafioId` que
 * debe adjuntarse en el paso de verificación y la fecha de expiración del código.
 */
export async function solicitarRegistro(
  dto: SolicitudRegistroDto,
  signal?: AbortSignal,
): Promise<DesafioOtp> {
  const respuesta = await apiPost<{
    mensaje: string
    desafio_id: string
    expira_en: string
  }>("/auth/register/request", dto, { signal })

  return {
    desafioId: respuesta.desafio_id,
    expiraEn: new Date(respuesta.expira_en),
    mensaje: respuesta.mensaje,
  }
}

/**
 * Verifica el código OTP y crea la cuenta del postulante, iniciando su sesión.
 */
export async function verificarYRegistrar(
  dto: VerificacionRegistroDto,
  signal?: AbortSignal,
): Promise<RespuestaPostulante> {
  return apiPost<RespuestaPostulante>("/auth/register/verify", dto, { signal })
}

/**
 * Inicia sesión con cédula y contraseña. Establece la cookie HttpOnly de sesión.
 */
export async function iniciarSesion(
  dto: LoginDto,
  signal?: AbortSignal,
): Promise<RespuestaPostulante> {
  return apiPost<RespuestaPostulante>("/auth/login", dto, { signal })
}

/**
 * Cierra la sesión activa revocando la cookie de sesión en el servidor.
 */
export async function cerrarSesion(signal?: AbortSignal): Promise<void> {
  return apiPost<void>("/auth/logout", undefined, { signal })
}

/**
 * Obtiene el perfil del postulante autenticado en la sesión actual.
 */
export async function obtenerSesion(
  signal?: AbortSignal,
): Promise<RespuestaPostulante> {
  return apiGet<RespuestaPostulante>("/auth/session", { signal })
}

/**
 * Inicia el proceso de recuperación de contraseña enviando un código OTP al correo registrado.
 * Devuelve el desafío OTP, pero el destino del correo es el registrado en el servidor —
 * el cliente no elige el correo de destino. Por seguridad, el servidor responde igual
 * si la cédula no existe (protección anti-oráculo).
 */
export async function solicitarRecuperacionPassword(
  dto: SolicitudRecuperacionDto,
  signal?: AbortSignal,
): Promise<DesafioOtp> {
  const respuesta = await apiPost<{
    mensaje: string
    desafio_id: string
    expira_en: string
  }>("/auth/password/forgot", dto, { signal })

  return {
    desafioId: respuesta.desafio_id,
    expiraEn: new Date(respuesta.expira_en),
    mensaje: respuesta.mensaje,
  }
}

/**
 * Restablece la contraseña utilizando el código OTP recibido.
 */
export async function restablecerPassword(
  dto: RestablecerPasswordDto,
  signal?: AbortSignal,
): Promise<RespuestaMensaje> {
  return apiPost<RespuestaMensaje>("/auth/password/reset", dto, { signal })
}
