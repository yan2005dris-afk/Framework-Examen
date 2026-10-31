package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/crypto/argon2"
)

/**
 * Parámetros recomendados por OWASP para Argon2id (seguridad del EF-11):
 * - Memoria: 64 MiB (65536 KiB)
 * - Iteraciones (tiempo): 3
 * - Paralelismo: 2 hilos
 * - Longitud de la sal: 16 bytes
 * - Longitud de la clave: 32 bytes
 */
const (
	argonMemory      = 64 * 1024
	argonIterations  = 3
	argonParallelism = 2
	argonSaltLength  = 16
	argonKeyLength   = 32
)

var (
	ErrInvalidHash         = errors.New("el hash de contraseña no tiene el formato esperado")
	ErrIncompatibleVersion = errors.New("versión incompatible de Argon2")
	ErrWeakPassword        = errors.New("la contraseña debe tener al menos 8 caracteres, una mayúscula, una minúscula, un número y un símbolo")
)

/**
 * ValidateStrength verifica que la contraseña cumpla con las reglas del negocio (RN-02):
 * - Al menos 8 caracteres
 * - Al menos una letra mayúscula
 * - Al menos una letra minúscula
 * - Al menos un número
 * - Al menos un símbolo o carácter especial
 *
 * @param {string} password - Contraseña a evaluar.
 * @returns {error} ErrWeakPassword si no cumple los requisitos, nil si es válida.
 */
func ValidateStrength(password string) error {
	if len(password) < 8 {
		return ErrWeakPassword
	}

	var hasUpper, hasLower, hasDigit, hasSymbol bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSymbol = true
		}
	}

	if !hasUpper || !hasLower || !hasDigit || !hasSymbol {
		return ErrWeakPassword
	}

	return nil
}

/**
 * Hash genera un hash Argon2id seguro con una sal aleatoria criptográfica de 16 bytes.
 * El formato resultante sigue el estándar PHC:
 * $argon2id$v=19$m=65536,t=3,p=2$<sal_b64>$<hash_b64>
 *
 * @param {string} password - Contraseña en texto plano.
 * @returns {string} Hash formateado para almacenamiento.
 * @throws {error} Si falla la generación de bytes aleatorios.
 */
func Hash(password string) (string, error) {
	salt := make([]byte, argonSaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("password: error al generar sal aleatoria: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, argonIterations, argonMemory, argonParallelism, argonKeyLength)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonIterations, argonParallelism, b64Salt, b64Hash)

	return encoded, nil
}

/**
 * Verify compara una contraseña en texto plano contra un hash Argon2id en tiempo constante.
 *
 * @param {string} password - Contraseña ingresada.
 * @param {string} encodedHash - Hash almacenado en la base de datos.
 * @returns {bool} true si la contraseña coincide con el hash.
 * @throws {error} Si el formato del hash es inválido o incompatible.
 */
func Verify(password, encodedHash string) (bool, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false, ErrInvalidHash
	}

	if parts[1] != "argon2id" {
		return false, fmt.Errorf("%w: variante no soportada (%s)", ErrInvalidHash, parts[1])
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, ErrInvalidHash
	}
	if version != argon2.Version {
		return false, ErrIncompatibleVersion
	}

	var memory uint32
	var iterations uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return false, ErrInvalidHash
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, ErrInvalidHash
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, ErrInvalidHash
	}

	comparisonHash := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(expectedHash)))

	if subtle.ConstantTimeCompare(expectedHash, comparisonHash) == 1 {
		return true, nil
	}

	return false, nil
}
