package identity

import "time"

/**
 * MaxLoginAttempts es el número de intentos fallidos que dispara el bloqueo
 * de la cuenta (EF-11 / RN-03). El dominio es el dueño de la regla: el
 * contador de la infraestructura recibe este valor por parámetro para que
 * ambas capas no puedan discrepar.
 */
const MaxLoginAttempts = 3

/**
 * LockoutDuration es la duración del bloqueo de la cuenta (EF-11 / RN-03).
 * El mensaje de ErrAccountLocked y la cabecera Retry-After deben reflejar
 * este mismo valor.
 */
const LockoutDuration = 10 * time.Minute
