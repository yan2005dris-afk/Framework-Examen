// Package submission es el contexto de postulaciones: el formulario EF-12
// que un postulante completa, revisa y envía al concurso.
package submission

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
)

/**
 * Status indica en qué etapa del proceso se encuentra una postulación.
 */
type Status string

const (
	/**
	 * StatusDraft es una postulación en construcción, editable por el postulante.
	 */
	StatusDraft Status = "draft"

	/**
	 * StatusInReview es una postulación enviada a revisión por el postulante.
	 */
	StatusInReview Status = "in_review"

	/**
	 * StatusSubmitted es una postulación enviada formalmente, con hash criptográfico.
	 */
	StatusSubmitted Status = "submitted"
)

/**
 * Submission es el formulario EF-12 de un postulante. El contenido guarda las
 * nueve secciones del formulario tal como las entrega el cliente.
 */
type Submission struct {
	Notification  *DeliveryStatus
	ID            uuid.UUID
	CompetitionID *uuid.UUID
	ApplicantID   uuid.UUID
	Code          string
	Status        Status
	Content       json.RawMessage
	Hash          string
	SubmittedAt   *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

/**
 * NewSubmission crea una postulación en estado borrador.
 *
 * @param {uuid.UUID} id - Identificador único de la postulación.
 * @param {uuid.UUID} applicantID - Postulante dueño de la postulación.
 * @param {string} code - Código visible de la postulación.
 * @param {json.RawMessage} content - Contenido con las secciones del formulario.
 * @param {time.Time} now - Momento de creación.
 * @returns {Submission} Postulación lista para persistir.
 * @throws {Error} ErrInvalidContent o ErrInvalidApplicant si los datos no son válidos.
 */
func NewSubmission(
	id uuid.UUID,
	applicantID uuid.UUID,
	code string,
	content json.RawMessage,
	now time.Time,
) (*Submission, error) {
	if applicantID == uuid.Nil {
		return nil, ErrInvalidApplicant
	}

	code = strings.TrimSpace(code)
	if code == "" {
		return nil, ErrInvalidIdentifier
	}

	canonical, err := CanonicalContent(content)
	if err != nil {
		return nil, err
	}

	return &Submission{
		ID:          id,
		ApplicantID: applicantID,
		Code:        code,
		Status:      StatusDraft,
		Content:     canonical,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

/**
 * UpdateContent reemplaza el contenido del formulario mientras la
 * postulación siga siendo editable.
 *
 * @param {json.RawMessage} content - Nuevo contenido del formulario.
 * @param {time.Time} now - Momento de la modificación.
 * @returns {void}
 * @throws {Error} ErrNotEditable si la postulación ya fue enviada.
 */
func (s *Submission) UpdateContent(content json.RawMessage, now time.Time) error {
	if !s.IsEditable() {
		return ErrNotEditable
	}

	canonical, err := CanonicalContent(content)
	if err != nil {
		return err
	}

	s.Content = canonical
	s.UpdatedAt = now
	return nil
}

/**
 * MarkAsSent sella la postulación: calcula el hash del contenido y bloquea
 * cualquier edición posterior.
 *
 * @param {time.Time} now - Momento del envío.
 * @returns {void}
 * @throws {Error} ErrNotEditable si la postulación ya fue enviada.
 */
func (s *Submission) MarkAsSent(now time.Time) error {
	if !s.IsEditable() {
		return ErrNotEditable
	}

	canonical, err := CanonicalContent(s.Content)
	if err != nil {
		return err
	}

	s.Content = canonical
	s.Status = StatusSubmitted
	s.SubmittedAt = &now
	s.UpdatedAt = now
	s.Hash = HashContent(canonical)
	return nil
}

/**
 * IsEditable indica si la postulación todavía admite cambios.
 *
 * @returns {boolean} True mientras no haya sido enviada formalmente.
 */
func (s *Submission) IsEditable() bool {
	return s.Status != StatusSubmitted
}

/**
 * CanonicalContent normaliza el contenido del formulario a un JSON estable:
 * con las claves ordenadas recursivamente (incluso dentro de listas y objetos anidados),
 * preservando números grandes sin pérdida de precisión.
 *
 * @param {json.RawMessage} content - Contenido crudo del formulario.
 * @returns {json.RawMessage} Contenido canónico.
 * @throws {Error} ErrInvalidContent si el JSON no es un objeto válido.
 */
func CanonicalContent(content json.RawMessage) (json.RawMessage, error) {
	trimmed := bytes.TrimSpace(content)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return nil, fmt.Errorf("%w: el contenido debe ser un objeto JSON", ErrInvalidContent)
	}

	var raw any
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidContent, err.Error())
	}
	if err := decoder.Decode(new(json.RawMessage)); err != io.EOF {
		return nil, ErrInvalidContent
	}

	canonicalVal := canonicalValue(raw)
	canonicalBytes, err := json.Marshal(canonicalVal)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidContent, err.Error())
	}

	return canonicalBytes, nil
}

func canonicalValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		m := make(map[string]any, len(val))
		for k, item := range val {
			m[k] = canonicalValue(item)
		}
		return m
	case []any:
		l := make([]any, len(val))
		for i, item := range val {
			l[i] = canonicalValue(item)
		}
		return l
	default:
		return val
	}
}

/**
 * HashContent calcula el hash SHA-256 del contenido canónico de la
 * postulación.
 *
 * @param {json.RawMessage} content - Contenido canónico del formulario.
 * @returns {string} Hash en hexadecimal de 64 caracteres.
 */
func HashContent(content json.RawMessage) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

/**
 * GenerateCode genera el código visible de una postulación.
 *
 * @param {time.Time} now - Momento de referencia para el año.
 * @returns {string} Código con el formato EF12-AAAA-XXXXXX.
 * @throws {Error} Si no se puede generar el componente aleatorio.
 */
func GenerateCode(now time.Time) (string, error) {
	suffix := make([]byte, 3)
	if _, err := rand.Read(suffix); err != nil {
		return "", fmt.Errorf("no se pudo generar el código de postulación: %w", err)
	}

	return fmt.Sprintf("EF12-%d-%s", now.Year(), strings.ToUpper(hex.EncodeToString(suffix))), nil
}
