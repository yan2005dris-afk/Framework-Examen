package identity

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

/**
 * Session representa una sesión de usuario activa almacenada en PostgreSQL.
 */
type Session struct {
	ID          uuid.UUID
	ApplicantID uuid.UUID
	TokenHash   string
	ExpiresAt   time.Time
	IP          string
	UserAgent   string
	CreatedAt   time.Time
}

/**
 * IsExpired verifica si la sesión ha superado su fecha de expiración.
 *
 * @returns {bool} true si la sesión ya no es válida.
 */
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

/**
 * GenerateSessionToken genera un token opaco criptográficamente seguro de 32 bytes (256 bits).
 *
 * @returns {string} Token en formato base64 URL-safe.
 * @throws {error} Si falla el generador de entropía.
 */
func GenerateSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("session: error al generar token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

/**
 * HashSessionToken calcula el SHA-256 de un token en formato hexadecimal (64 caracteres).
 * Solo el hash se almacena en la base de datos (ADR-006).
 *
 * @param {string} token - Token en claro recibido por cookie.
 * @returns {string} Hash SHA-256 en minúsculas.
 */
func HashSessionToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
