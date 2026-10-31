package cache

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"
)

// OTPTTL es el tiempo de expiración estándar para códigos OTP (10 minutos según EF-11).
const OTPTTL = 10 * time.Minute

/**
 * OTPStore gestiona la generación, guardado y verificación de códigos OTP de 6 dígitos.
 * Los códigos se almacenan en caché con expiración automática (TTL).
 */
type OTPStore struct {
	cache Cache
}

/**
 * NewOTPStore crea una instancia de OTPStore asociada al adaptador de caché.
 *
 * @param {Cache} c - Adaptador de caché (Redis u otro).
 * @returns {*OTPStore} Store listo para usar.
 */
func NewOTPStore(c Cache) *OTPStore {
	return &OTPStore{cache: c}
}

/**
 * GenerateOTP genera un código numérico aleatorio de 6 dígitos (100000 - 999999)
 * usando un generador criptográficamente seguro.
 *
 * @returns {string} Código OTP de 6 dígitos.
 * @throws {error} Si falla crypto/rand.
 */
func GenerateOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return "", fmt.Errorf("otp: error al generar código aleatorio: %w", err)
	}
	return fmt.Sprintf("%06d", n.Int64()+100000), nil
}

/**
 * SaveOTP guarda un código OTP asociado a una clave (cédula o correo) con su TTL.
 *
 * @param {Context} ctx - Contexto de la petición.
 * @param {string} identifier - Cédula o correo del usuario.
 * @param {string} code - Código OTP a guardar.
 * @returns {error} Error de la caché si ocurre.
 */
func (s *OTPStore) SaveOTP(ctx context.Context, identifier string, code string) error {
	key := otpKey(identifier)
	if err := s.cache.Set(ctx, key, code, OTPTTL); err != nil {
		return fmt.Errorf("otp: SaveOTP(%q): %w", identifier, err)
	}
	return nil
}

/**
 * VerifyOTP comprueba si el código provisto coincide con el almacenado.
 * Si coincide, elimina el código de la caché para evitar que sea reutilizado.
 *
 * @param {Context} ctx - Contexto de la petición.
 * @param {string} identifier - Cédula o correo del usuario.
 * @param {string} code - Código OTP ingresado por el usuario.
 * @returns {bool} true si el código es válido y coincide, false si no coincide o expiró.
 * @throws {error} Error si la caché falla.
 */
func (s *OTPStore) VerifyOTP(ctx context.Context, identifier string, code string) (bool, error) {
	key := otpKey(identifier)
	stored, err := s.cache.Get(ctx, key)
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("otp: VerifyOTP(%q): %w", identifier, err)
	}

	if stored != code {
		return false, nil
	}

	// Al ser verificado exitosamente, se elimina el OTP (uso único)
	_ = s.cache.Delete(ctx, key)
	return true, nil
}

// otpKey genera la clave de almacenamiento para un identificador.
func otpKey(identifier string) string {
	return fmt.Sprintf("identity:otp:%s", identifier)
}
