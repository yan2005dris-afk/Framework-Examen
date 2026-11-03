package identity

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"
)

/**
 * OTPPurpose identifica la finalidad de un desafío OTP. La finalidad separa
 * el código de registro del de recuperación: un código de una finalidad nunca
 * habilita la otra (EF-11 / ADR-008).
 */
type OTPPurpose string

const (
	// PurposeRegistration es la finalidad del código que habilita el registro.
	PurposeRegistration OTPPurpose = "registro"
	// PurposeRecovery es la finalidad del código que habilita el restablecimiento.
	PurposeRecovery OTPPurpose = "recuperacion"
)

/**
 * OTPTTL es la vigencia de un desafío OTP (EF-11 / RN-03). El valor lo decide
 * el dominio y lo recibe la infraestructura como parámetro: el almacén no
 * define la política, la aplica.
 */
const OTPTTL = 10 * time.Minute

/**
 * OTPChallenge identifica un desafío OTP emitido. No contiene el código ni
 * datos personales del postulante: la clave es interna y jamás se expone.
 */
type OTPChallenge struct {
	ID        string
	Key       string
	ExpiresAt time.Time
}

/**
 * GenerateOTP genera un código de seis dígitos con fuente criptográfica,
 * nunca con el reloj ni con math/rand.
 *
 * @returns {string} Código de seis dígitos.
 * @throws {error} Si falla el generador de entropía.
 */
func GenerateOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return "", errors.New("no se pudo generar el código")
	}
	return fmt.Sprintf("%06d", n.Int64()+100000), nil
}
