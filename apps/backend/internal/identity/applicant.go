// Package identity es el contexto de identidad: postulantes, registro,
// autenticación y sesiones. Por ahora solo define el modelo de postulante;
// los casos de uso de registro y acceso llegan en la siguiente iteración.
package identity

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

/**
 * Errores del contexto de identidad.
 */
var (
	ErrInvalidIDCardFormat = errors.New("la cédula debe tener exactamente 10 dígitos numéricos")
	ErrInvalidIDCard       = errors.New("la cédula no cumple el algoritmo del Registro Civil")
	ErrInvalidEmail        = errors.New("el correo electrónico no tiene un formato válido")
	ErrInvalidApplicant    = errors.New("los datos del postulante están incompletos")
)

/**
 * IDCard es el número de identidad de una persona natural ecuatoriana. Solo
 * puede existir en un estado válido: el algoritmo módulo 10 del Registro
 * Civil se aplica en el constructor.
 */
type IDCard string

/**
 * NewIDCard normaliza y valida una cédula ecuatoriana.
 *
 * @param {string} value - Cédula tal como la ingresa el usuario.
 * @returns {IDCard} Cédula validada y normalizada.
 * @throws {Error} ErrInvalidIDCardFormat o ErrInvalidIDCard si la validación falla.
 */
func NewIDCard(value string) (IDCard, error) {
	cleaned := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, strings.TrimSpace(value))

	if len(cleaned) != 10 {
		return "", fmt.Errorf("%w: se esperaban 10 dígitos", ErrInvalidIDCardFormat)
	}

	province, err := strconv.Atoi(cleaned[:2])
	if err != nil {
		return "", fmt.Errorf("%w: el código de provincia no es válido", ErrInvalidIDCard)
	}
	if (province < 1 || province > 24) && province != 30 {
		return "", fmt.Errorf("%w: el código de provincia no es válido", ErrInvalidIDCard)
	}

	thirdDigit := int(cleaned[2] - '0')
	if thirdDigit >= 6 {
		return "", fmt.Errorf("%w: el tercer dígito debe ser menor a 6", ErrInvalidIDCard)
	}

	coefficients := []int{2, 1, 2, 1, 2, 1, 2, 1, 2}
	sum := 0
	for i, coefficient := range coefficients {
		digit := int(cleaned[i]-'0') * coefficient
		if digit >= 10 {
			digit -= 9
		}
		sum += digit
	}

	checkDigit := 0
	if sum%10 != 0 {
		checkDigit = 10 - sum%10
	}

	if checkDigit != int(cleaned[9]-'0') {
		return "", fmt.Errorf("%w: el dígito verificador no coincide", ErrInvalidIDCard)
	}

	return IDCard(cleaned), nil
}

/**
 * String devuelve la cédula sin formato.
 *
 * @returns {string} Cédula como texto de 10 dígitos.
 */
func (c IDCard) String() string {
	return string(c)
}

/**
 * Applicant es la persona natural que se registra y postula al concurso.
 */
type Applicant struct {
	ID               uuid.UUID
	IDCard           IDCard
	FirstName        string
	LastName         string
	Email            string
	ValidationMethod string
	RegisteredAt     time.Time
}

/**
 * NewApplicant crea un postulante validando los datos obligatorios.
 *
 * @param {uuid.UUID} id - Identificador único del postulante.
 * @param {IDCard} idCard - Cédula ya validada.
 * @param {string} firstName - Nombres del postulante.
 * @param {string} lastName - Apellidos del postulante.
 * @param {string} email - Correo ya validado.
 * @param {string} validationMethod - Método de validación de identidad.
 * @param {time.Time} registeredAt - Fecha de registro.
 * @returns {Applicant} Postulante listo para persistir.
 * @throws {Error} ErrInvalidApplicant si faltan nombres o apellidos.
 */
func NewApplicant(
	id uuid.UUID,
	idCard IDCard,
	firstName string,
	lastName string,
	email string,
	validationMethod string,
	registeredAt time.Time,
) (*Applicant, error) {
	firstName = strings.TrimSpace(firstName)
	lastName = strings.TrimSpace(lastName)

	if firstName == "" || lastName == "" {
		return nil, fmt.Errorf("%w: nombres y apellidos son obligatorios", ErrInvalidApplicant)
	}

	return &Applicant{
		ID:               id,
		IDCard:           idCard,
		FirstName:        firstName,
		LastName:         lastName,
		Email:            strings.ToLower(strings.TrimSpace(email)),
		ValidationMethod: validationMethod,
		RegisteredAt:     registeredAt,
	}, nil
}

/**
 * FullName devuelve el nombre completo del postulante.
 *
 * @returns {string} Nombres y apellidos concatenados.
 */
func (a *Applicant) FullName() string {
	return a.FirstName + " " + a.LastName
}
