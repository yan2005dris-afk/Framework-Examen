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
	CompetitionID *uuid.UUID      `json:"convocatoriaId,omitempty"`
	ApplicantID   string          `json:"applicantId"`
	Code          string          `json:"code"`
	Content       json.RawMessage `json:"content"`
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
	Notification  *DeliveryStatus `json:"notificacion,omitempty"`
	CompetitionID *uuid.UUID      `json:"convocatoriaId,omitempty"`
	ID            uuid.UUID       `json:"id"`
	ApplicantID   uuid.UUID       `json:"applicantId"`
	Code          string          `json:"code"`
	Status        Status          `json:"status"`
	Content       json.RawMessage `json:"content"`
	Hash          string          `json:"hash,omitempty"`
	SubmittedAt   *time.Time      `json:"submittedAt,omitempty"`
	CreatedAt     time.Time       `json:"createdAt"`
	UpdatedAt     time.Time       `json:"updatedAt"`
	Progress      Progress        `json:"progress"`
}

/**
 * NewResponse construye la representación pública de una postulación.
 *
 * @param {Submission} item - Postulación del dominio.
 * @returns {Response} DTO listo para serializar.
 */
func NewResponse(item *Submission) Response {
	return Response{
		Notification:  item.Notification,
		CompetitionID: item.CompetitionID,
		ID:            item.ID,
		ApplicantID:   item.ApplicantID,
		Code:          item.Code,
		Status:        item.Status,
		Content:       item.Content,
		Hash:          item.Hash,
		SubmittedAt:   item.SubmittedAt,
		CreatedAt:     item.CreatedAt,
		UpdatedAt:     item.UpdatedAt,
		Progress:      FormProgress(item.Content, time.Now()),
	}
}

/** Progress identifica las secciones completas; el consentimiento no es una décima sección. */
type Progress struct {
	Total      int               `json:"total"`
	Completed  int               `json:"completed"`
	Percentage int               `json:"percentage"`
	Sections   []SectionProgress `json:"sections"`
}

/** SectionProgress indica si una sección satisface sus campos y reglas. */
type SectionProgress struct {
	Number   int  `json:"number"`
	Complete bool `json:"complete"`
}

/**
 * FormProgress calcula el avance desde datos validados, sin banderas del cliente.
 * @param {json.RawMessage} content - Formulario almacenado.
 * @param {time.Time} now - Momento de referencia.
 * @returns {Progress} Avance sobre las nueve secciones.
 */
func FormProgress(content json.RawMessage, now time.Time) Progress {
	p := Progress{Total: 9, Sections: make([]SectionProgress, 0, 9)}
	form, err := DecodeForm(content)
	for n := 1; n <= 9; n++ {
		complete := err == nil && validateSection(form, n, true, now) == nil
		p.Sections = append(p.Sections, SectionProgress{Number: n, Complete: complete})
		if complete {
			p.Completed++
		}
	}
	p.Percentage = p.Completed * 100 / p.Total
	return p
}

/** SubmitRequest permite persistir las cinco aceptaciones en la misma operación de envío. */
type SubmitRequest struct {
	Consent *Consent `json:"consentimiento,omitempty"`
}
