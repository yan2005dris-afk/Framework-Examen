// Package document administra los respaldos PDF del EF-12.
package document

import (
	"github.com/google/uuid"
	"time"
)

const (
	MaxFileBytes         int64 = 2 * 1024 * 1024
	MaxOrganizationBytes int64 = 4 * 1024 * 1024
	MaxApplicantBytes    int64 = 50 * 1024 * 1024
	DownloadTTL                = 5 * time.Minute
)

/** Document conserva metadatos comprobados por el servidor, sin exponer la clave del objeto. */
type Document struct {
	ID           uuid.UUID `json:"id"`
	SubmissionID uuid.UUID `json:"submissionId"`
	Section      int       `json:"section"`
	RecordID     string    `json:"recordId"`
	Name         string    `json:"name"`
	MIME         string    `json:"mime"`
	SizeBytes    int64     `json:"sizeBytes"`
	SHA256       string    `json:"sha256"`
	CreatedAt    time.Time `json:"createdAt"`
	StorageKey   string    `json:"-"`
}

/** LimitForSection determina el máximo en bytes del respaldo de cada sección.
 * @param {int} section - Sección del EF-12.
 * @returns {int64, error} Límite o sección no válida.
 */
func LimitForSection(section int) (int64, error) {
	if section == 4 {
		return MaxOrganizationBytes, nil
	}
	if section >= 5 && section <= 8 {
		return MaxFileBytes, nil
	}
	return 0, ErrInvalidSection
}
