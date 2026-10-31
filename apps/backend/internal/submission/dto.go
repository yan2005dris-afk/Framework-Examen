package submission

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

/**
 * CreateRequest es el cuerpo de la creación de una postulación.
 */
type CreateRequest struct {
	ApplicantID string          `json:"applicantId"`
	Code        string          `json:"code"`
	Content     json.RawMessage `json:"content"`
}

/**
 * UpdateRequest es el cuerpo de la actualización de una postulación.
 */
type UpdateRequest struct {
	Content json.RawMessage `json:"content"`
}

/**
 * Response es la representación pública de una postulación.
 */
type Response struct {
	ID          uuid.UUID       `json:"id"`
	ApplicantID uuid.UUID       `json:"applicantId"`
	Code        string          `json:"code"`
	Status      Status          `json:"status"`
	Content     json.RawMessage `json:"content"`
	Hash        string          `json:"hash,omitempty"`
	SubmittedAt *time.Time      `json:"submittedAt,omitempty"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
}

/**
 * NewResponse construye la representación pública de una postulación.
 *
 * @param {Submission} item - Postulación del dominio.
 * @returns {Response} DTO listo para serializar.
 */
func NewResponse(item *Submission) Response {
	return Response{
		ID:          item.ID,
		ApplicantID: item.ApplicantID,
		Code:        item.Code,
		Status:      item.Status,
		Content:     item.Content,
		Hash:        item.Hash,
		SubmittedAt: item.SubmittedAt,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}
}
