package submission

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
)

/** ApplicantReader consulta la cuenta OTP sin acoplar el formulario a PostgreSQL. */
type ApplicantReader interface {
	GetByID(context.Context, uuid.UUID) (*identity.Applicant, error)
}

/** WithApplicants configura la precarga y el vínculo de cédula/correo con su cuenta.
 * @returns {Service} Servicio configurado antes de atender peticiones.
 */
func (s *Service) WithApplicants(reader ApplicantReader) *Service {
	s.applicants = reader
	return s
}

/** prepareContent precarga datos disponibles y protege los identificadores de la cuenta.
 * Nombres y apellidos siguen siendo declarativos; el OTP no certifica identidad civil.
 * @returns {json.RawMessage, error} Formulario normalizado para un borrador.
 */
func (s *Service) prepareContent(ctx context.Context, id uuid.UUID, raw json.RawMessage, now time.Time) (json.RawMessage, error) {
	form, err := DecodeForm(raw)
	if err != nil {
		return nil, err
	}
	if s.applicants != nil {
		account, err := s.applicants.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		if account == nil || account.ID != id {
			return nil, ErrInvalidApplicant
		}
		if form.Personal == nil {
			form.Personal = &PersonalSection{}
		}
		if form.Contact == nil {
			form.Contact = &ContactSection{}
		}
		p, c := form.Personal, form.Contact
		if p.IDCard != "" && p.IDCard != account.IDCard.String() {
			return nil, ErrForbidden
		}
		if c.Email != "" && !strings.EqualFold(c.Email, account.Email) {
			return nil, rule("seccion2.correo1", "el correo principal debe coincidir con la cuenta registrada")
		}
		p.IDCard = account.IDCard.String()
		if strings.TrimSpace(p.FirstName) == "" {
			p.FirstName = account.FirstName
		}
		if strings.TrimSpace(p.LastName) == "" {
			p.LastName = account.LastName
		}
		c.Email = account.Email
	}
	encoded, err := json.Marshal(form)
	if err != nil {
		return nil, ErrInvalidContent
	}
	return validateContent(encoded, false, now)
}
