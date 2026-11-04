package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
)

// Escritura y lectura del formulario en las tablas normalizadas (00024).
// Fase 3 de la normalización: el adaptador escribe las tablas junto con
// `content`, que sigue siendo la fuente de lectura hasta la fase 4.

const consentKey = "consentimiento"

/** formDocument es la clave de un PDF propio de la postulación. */
type formDocument struct {
	Section int32
	Record  string
}

/**
 * writeForm reescribe sólo las secciones que cambiaron respecto de previous.
 * Borrar la sección elimina sus datos en cascada; después se insertan los nuevos.
 * Debe ejecutarse antes de actualizar la cabecera, mientras la postulación sigue
 * en borrador, porque las tablas hijas rechazan escrituras de expedientes enviados.
 *
 * @param {Submission} item - Postulación con el contenido a persistir.
 * @param {json.RawMessage} previous - Contenido confirmado antes del cambio; nil al crear.
 * @returns {error} ErrInvalidContent si una restricción rechaza los datos.
 */
func (r *Repository) writeForm(ctx context.Context, item *submission.Submission, previous json.RawMessage) error {
	current, err := formFields(item.Content)
	if err != nil {
		return err
	}
	before, err := formFields(previous)
	if err != nil {
		return err
	}
	changed := []int16{}
	for number := int16(1); number <= 9; number++ {
		key := fmt.Sprintf("seccion%d", number)
		if !bytes.Equal(current[key], before[key]) {
			changed = append(changed, number)
		}
	}
	consentChanged := !bytes.Equal(current[consentKey], before[consentKey])
	if len(changed) == 0 && !consentChanged {
		return nil
	}
	form, err := submission.DecodeForm(item.Content)
	if err != nil {
		return err
	}
	rows := formToRows(form)
	documents, err := r.formDocuments(ctx, item.ID, changed)
	if err != nil {
		return err
	}
	saved := pgtype.Timestamptz{Time: item.UpdatedAt, Valid: true}
	for _, number := range changed {
		if err := r.queries.DeleteFormSection(ctx, sqlcgen.DeleteFormSectionParams{SubmissionID: item.ID, Section: number}); err != nil {
			return formWriteError(err)
		}
		for _, section := range rows.Sections {
			if section.Number != number {
				continue
			}
			if err := r.queries.InsertFormSection(ctx, sqlcgen.InsertFormSectionParams{SubmissionID: item.ID, Section: number, ItemsDeclared: section.Items, SavedAt: saved}); err != nil {
				return formWriteError(err)
			}
			if err := r.writeSection(ctx, item.ID, number, rows, documents); err != nil {
				return formWriteError(err)
			}
		}
	}
	if consentChanged {
		if err := r.queries.DeleteFormConsent(ctx, item.ID); err != nil {
			return formWriteError(err)
		}
		if c := rows.Consent; c != nil {
			accepted, err := timestampField(c.AcceptedAt)
			if err != nil {
				return err
			}
			if err := r.queries.InsertFormConsent(ctx, sqlcgen.InsertFormConsentParams{
				SubmissionID: item.ID, PrivacyPolicyID: c.PrivacyPolicyID, PrivacyPolicyHash: c.PrivacyPolicyHash, AcceptedAt: accepted,
				DataProcessing: c.DataProcessing, Truthfulness: c.Truthfulness, CompetitionRules: c.CompetitionRules,
				NamePublication: c.NamePublication, PhysicalDelivery: c.PhysicalDelivery, Signature: c.Signature, SignedAt: c.SignedAt,
			}); err != nil {
				return formWriteError(err)
			}
		}
	}
	return nil
}

/** writeSection inserta los datos de una sección ya marcada como guardada. */
func (r *Repository) writeSection(ctx context.Context, id uuid.UUID, number int16, rows formRows, documents map[string]formDocument) error {
	q := r.queries
	switch number {
	case 1:
		p := rows.Personal
		birth, err := dateField(p.BirthDate)
		if err != nil {
			return err
		}
		return q.InsertFormPersonal(ctx, sqlcgen.InsertFormPersonalParams{
			SubmissionID: id, FirstName: p.FirstName, LastName: p.LastName, Gender: p.Gender, BirthDate: birth,
			Nationality: p.Nationality, MaritalStatus: p.MaritalStatus, SpouseName: p.SpouseName, SpouseIDCard: p.SpouseIDCard,
		})
	case 2:
		return r.writeContact(ctx, id, rows.Contact)
	case 3:
		if err := q.InsertFormAdditional(ctx, sqlcgen.InsertFormAdditionalParams{SubmissionID: id, HasRelatives: *rows.Additional}); err != nil {
			return err
		}
		for i, v := range rows.Relatives {
			if err := q.InsertFormRelative(ctx, sqlcgen.InsertFormRelativeParams{
				SubmissionID: id, Position: int32(i), RecordKey: v.ID, Name: v.Name, Relationship: v.Relationship,
				Institution: v.Institution, Area: v.Area, JobTitle: v.Position,
			}); err != nil {
				return err
			}
		}
	case 4:
		p := rows.Position
		return q.InsertFormPosition(ctx, sqlcgen.InsertFormPositionParams{
			SubmissionID: id, Eligible: p.Eligible, VerificationDetail: p.VerificationDetail, HasOrganization: p.HasOrganization,
			Organization: p.Organization, DocumentID: documentField(documents, 4, "", p.Document),
		})
	case 5:
		for i, d := range rows.Degrees {
			registered, err := dateField(d.RegistryDate)
			if err != nil {
				return err
			}
			if err := q.InsertFormDegree(ctx, sqlcgen.InsertFormDegreeParams{
				SubmissionID: id, Position: int32(i), RecordKey: d.ID, Level: d.Level, Institution: d.Institution, Type: d.Type,
				Title: d.Title, Area: d.Area, Registry: d.Registry, RegistryDate: registered, DocumentID: documentField(documents, 5, d.ID, d.Document),
			}); err != nil {
				return err
			}
		}
	case 6:
		for i, e := range rows.Experiences {
			start, err := dateField(e.StartDate)
			if err != nil {
				return err
			}
			end, err := dateField(e.EndDate)
			if err != nil {
				return err
			}
			if err := q.InsertFormExperience(ctx, sqlcgen.InsertFormExperienceParams{
				SubmissionID: id, Position: int32(i), RecordKey: e.ID, Type: e.Type, JobTitle: e.Position, Institution: e.Institution,
				StartDate: start, EndDate: end, IsCurrent: e.Current, Activities: e.Activities, Functions: e.Functions,
				DocumentID: documentField(documents, 6, e.ID, e.Document),
			}); err != nil {
				return err
			}
		}
	case 7:
		if err := q.InsertFormPublicationsInfo(ctx, sqlcgen.InsertFormPublicationsInfoParams{SubmissionID: id, HasPublications: *rows.HasPublications}); err != nil {
			return err
		}
		for i, p := range rows.Publications {
			published, err := dateField(p.Date)
			if err != nil {
				return err
			}
			if err := q.InsertFormPublication(ctx, sqlcgen.InsertFormPublicationParams{
				SubmissionID: id, Position: int32(i), RecordKey: p.ID, Title: p.Title, Type: p.Type, Medium: p.Medium,
				PublishedOn: published, Related: p.Related, DocumentID: documentField(documents, 7, p.ID, p.Document),
			}); err != nil {
				return err
			}
		}
	case 8:
		for i, e := range rows.Trainings {
			start, err := dateField(e.StartDate)
			if err != nil {
				return err
			}
			end, err := dateField(e.EndDate)
			if err != nil {
				return err
			}
			if err := q.InsertFormTraining(ctx, sqlcgen.InsertFormTrainingParams{
				SubmissionID: id, Position: int32(i), RecordKey: e.ID, Type: e.Type, Name: e.Name, Institution: e.Institution,
				StartDate: start, EndDate: end, Hours: int64(e.Hours), DocumentID: documentField(documents, 8, e.ID, e.Document),
			}); err != nil {
				return err
			}
		}
	case 9:
		d := rows.Disqualifications
		if err := q.InsertFormDisqualifications(ctx, sqlcgen.InsertFormDisqualificationsParams{
			SubmissionID: id, Q1: d.Q1, Q2: d.Q2, Q3: d.Q3, Q4: d.Q4, Q5: d.Q5, Q6: d.Q6, Q7: d.Q7,
			Q8: d.Q8, Q9: d.Q9, Q10: d.Q10, Q11: d.Q11, Q12: d.Q12, Q13: d.Q13,
		}); err != nil {
			return err
		}
		if removal := d.Q7Details; removal != nil {
			end, err := dateField(removal.EndDate)
			if err != nil {
				return err
			}
			return q.InsertFormRemovalDetails(ctx, sqlcgen.InsertFormRemovalDetailsParams{
				SubmissionID: id, Institution: removal.Institution, JobTitle: removal.Position, EndDate: end, Reason: removal.Reason,
			})
		}
	}
	return nil
}

/**
 * writeContact resuelve la ubicación contra el catálogo: Ecuador usa los ids de
 * provincia y cantón; otro país guarda el texto que escribió el postulante.
 */
func (r *Repository) writeContact(ctx context.Context, id uuid.UUID, c *submission.ContactSection) error {
	params := sqlcgen.InsertFormContactParams{
		SubmissionID: id, Mobile: c.Mobile, SecondaryEmail: c.SecondaryEmail, MainStreet: c.MainStreet, Number: c.Number,
		SecondaryStreet: c.SecondaryStreet, Area: c.Area, Reference: c.Reference, Phone: c.Phone,
	}
	if c.Country != "" {
		country, err := r.queries.FindCountryID(ctx, c.Country)
		if err != nil {
			return catalogError(err, "el país no existe en el catálogo")
		}
		params.CountryID = pgtype.UUID{Bytes: country, Valid: true}
	}
	if c.Country != "Ecuador" {
		params.ProvinceName, params.CityName = c.Province, c.City
		return r.queries.InsertFormContact(ctx, params)
	}
	if c.Province != "" {
		province, err := r.queries.FindChildCatalogID(ctx, sqlcgen.FindChildCatalogIDParams{ParentID: params.CountryID, Value: c.Province})
		if err != nil {
			return catalogError(err, "la provincia no existe en el catálogo de Ecuador")
		}
		params.ProvinceID = pgtype.UUID{Bytes: province, Valid: true}
	}
	if c.City != "" {
		city, err := r.queries.FindChildCatalogID(ctx, sqlcgen.FindChildCatalogIDParams{ParentID: params.ProvinceID, Value: c.City})
		if err != nil {
			return catalogError(err, "el cantón no pertenece a la provincia seleccionada")
		}
		params.CityID = pgtype.UUID{Bytes: city, Valid: true}
	}
	return r.queries.InsertFormContact(ctx, params)
}

/** formDocuments lista los PDF propios sólo si cambió una sección con respaldos. */
func (r *Repository) formDocuments(ctx context.Context, id uuid.UUID, changed []int16) (map[string]formDocument, error) {
	documents := map[string]formDocument{}
	needed := false
	for _, number := range changed {
		needed = needed || number >= 4 && number <= 8
	}
	if !needed {
		return documents, nil
	}
	rows, err := r.queries.ListFormDocuments(ctx, id)
	if err != nil {
		return nil, err
	}
	for _, d := range rows {
		documents[d.ID.String()] = formDocument{Section: d.Section, Record: d.RecordID}
	}
	return documents, nil
}

/**
 * documentField vincula la referencia sólo si apunta a un PDF propio de la misma
 * sección y registro. Una referencia ajena o inexistente se descarta: los
 * metadatos válidos salen siempre de `documents`, nunca del cliente.
 */
func documentField(documents map[string]formDocument, section int32, record string, ref *submission.DocumentReference) pgtype.UUID {
	if ref == nil || ref.ID == "" {
		return pgtype.UUID{}
	}
	d, ok := documents[ref.ID]
	if !ok || d.Section != section || d.Record != record {
		return pgtype.UUID{}
	}
	id, err := uuid.Parse(ref.ID)
	if err != nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: id, Valid: true}
}

/** formFields separa el contenido canónico por clave de sección para compararlo. */
func formFields(content json.RawMessage) (map[string]json.RawMessage, error) {
	fields := map[string]json.RawMessage{}
	if len(bytes.TrimSpace(content)) == 0 {
		return fields, nil
	}
	// PostgreSQL reescribe el jsonb con otro formato: se canoniza antes de comparar.
	canonical, err := submission.CanonicalContent(content)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(canonical, &fields); err != nil {
		return nil, submission.ErrInvalidContent
	}
	return fields, nil
}

/** dateField convierte una fecha AAAA-MM-DD; vacía es NULL. */
func dateField(value string) (pgtype.Date, error) {
	if value == "" {
		return pgtype.Date{}, nil
	}
	date, err := time.Parse(time.DateOnly, value)
	if err != nil {
		return pgtype.Date{}, fmt.Errorf("%w: fecha con formato inválido", submission.ErrInvalidContent)
	}
	return pgtype.Date{Time: date, Valid: true}, nil
}

/** dateText devuelve la fecha AAAA-MM-DD o vacío si es NULL. */
func dateText(value pgtype.Date) string {
	if !value.Valid {
		return ""
	}
	return value.Time.Format(time.DateOnly)
}

/** timestampField convierte la aceptación RFC 3339 del sello; vacía es NULL. */
func timestampField(value string) (pgtype.Timestamptz, error) {
	if value == "" {
		return pgtype.Timestamptz{}, nil
	}
	instant, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return pgtype.Timestamptz{}, fmt.Errorf("%w: fecha de aceptación inválida", submission.ErrInvalidContent)
	}
	return pgtype.Timestamptz{Time: instant, Valid: true}, nil
}

/** catalogError distingue un valor inexistente de un fallo de la base. */
func catalogError(err error, message string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%w: %s", submission.ErrInvalidContent, message)
	}
	return err
}

/**
 * formWriteError conserva los errores de ventana y estado y traduce cualquier
 * otra restricción de las tablas del formulario a contenido inválido.
 */
func formWriteError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	if pgErr.ConstraintName == "submission_closed" || pgErr.ConstraintName == "competition_open" {
		return mapError(err)
	}
	if len(pgErr.Code) == 5 && pgErr.Code[:2] == "23" {
		return fmt.Errorf("%w: restricción %s", submission.ErrInvalidContent, pgErr.ConstraintName)
	}
	return err
}

/**
 * snapshotContent devuelve lo que la cabecera guarda en `content`: el snapshot de
 * un expediente que ya no se edita (enviado o cerrado) o NULL para un borrador.
 */
func snapshotContent(item *submission.Submission) []byte {
	if item.IsEditable() {
		return nil
	}
	return item.Content
}

/** formSector toma el sector de la sección 4; NULL si la sección no se guardó. */
func formSector(content json.RawMessage) *string {
	form, err := submission.DecodeForm(content)
	if err != nil || form.Position == nil {
		return nil
	}
	sector := form.Position.Sector
	return &sector
}

/** formContent serializa el borrador tal como está en las tablas, para compararlo. */
func (r *Repository) formContent(ctx context.Context, id uuid.UUID) (json.RawMessage, error) {
	form, err := r.loadForm(ctx, id, time.Now())
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(form)
	if err != nil {
		return nil, err
	}
	return submission.CanonicalContent(raw)
}

/**
 * Hydrate reemplaza el contenido de un borrador por el formulario armado desde
 * las tablas normalizadas, su fuente desde la fase 4. Las postulaciones enviadas
 * o cerradas conservan `content`, que es su snapshot.
 *
 * @param {Submission} item - Postulación recién leída.
 * @returns {error} Fallo de lectura de las tablas.
 */
func (r *Repository) Hydrate(ctx context.Context, item *submission.Submission) error {
	if item == nil || !item.IsEditable() {
		return nil
	}
	content, err := r.formContent(ctx, item.ID)
	if err != nil {
		return fmt.Errorf("no se pudo leer el formulario: %w", err)
	}
	item.Content = content
	return nil
}

/**
 * loadForm arma el formulario desde las tablas normalizadas, con la cédula y el
 * correo de la cuenta, el sector de la cabecera, los metadatos de `documents`
 * y las duraciones calculadas a la fecha indicada.
 *
 * @param {uuid.UUID} id - Postulación.
 * @param {time.Time} now - Fecha de corte de los empleos vigentes.
 * @returns {Form} Formulario equivalente al guardado.
 */
func (r *Repository) loadForm(ctx context.Context, id uuid.UUID, now time.Time) (*submission.Form, error) {
	q := r.queries
	who, err := q.GetFormIdentity(ctx, id)
	if err != nil {
		return nil, err
	}
	sections, err := q.ListFormSections(ctx, id)
	if err != nil {
		return nil, err
	}
	documentRows, err := q.ListFormDocuments(ctx, id)
	if err != nil {
		return nil, err
	}
	documents := make(map[uuid.UUID]*submission.DocumentReference, len(documentRows))
	for _, d := range documentRows {
		documents[d.ID] = submission.NewDocumentReference(d.ID, d.Name, d.SizeBytes, d.Sha256, d.CreatedAt)
	}
	reference := func(id pgtype.UUID) *submission.DocumentReference {
		if !id.Valid {
			return nil
		}
		if ref, ok := documents[uuid.UUID(id.Bytes)]; ok {
			clone := *ref
			return &clone
		}
		return nil
	}
	rows := formRows{}
	for _, s := range sections {
		rows.Sections = append(rows.Sections, formSection{Number: s.Section, Items: s.ItemsDeclared})
		switch s.Section {
		case 1:
			p, err := q.GetFormPersonal(ctx, id)
			if err != nil {
				return nil, err
			}
			rows.Personal = &submission.PersonalSection{FirstName: p.FirstName, LastName: p.LastName, Gender: p.Gender, BirthDate: dateText(p.BirthDate),
				Nationality: p.Nationality, MaritalStatus: p.MaritalStatus, SpouseName: p.SpouseName, SpouseIDCard: p.SpouseIDCard}
		case 2:
			c, err := q.GetFormContact(ctx, id)
			if err != nil {
				return nil, err
			}
			rows.Contact = &submission.ContactSection{Country: c.Country, Province: c.Province, City: c.City, Mobile: c.Mobile,
				SecondaryEmail: c.SecondaryEmail, MainStreet: c.MainStreet, Number: c.Number, SecondaryStreet: c.SecondaryStreet,
				Area: c.Area, Reference: c.Reference, Phone: c.Phone}
		case 3:
			has, err := q.GetFormAdditional(ctx, id)
			if err != nil {
				return nil, err
			}
			rows.Additional = &has
			list, err := q.ListFormRelatives(ctx, id)
			if err != nil {
				return nil, err
			}
			for _, v := range list {
				rows.Relatives = append(rows.Relatives, submission.Relative{ID: v.RecordKey, Name: v.Name, Relationship: v.Relationship,
					Institution: v.Institution, Area: v.Area, Position: v.JobTitle})
			}
		case 4:
			p, err := q.GetFormPosition(ctx, id)
			if err != nil {
				return nil, err
			}
			rows.Position = &submission.PositionSection{Eligible: p.Eligible, VerificationDetail: p.VerificationDetail,
				HasOrganization: p.HasOrganization, Organization: p.Organization, Document: reference(p.DocumentID)}
		case 5:
			list, err := q.ListFormDegrees(ctx, id)
			if err != nil {
				return nil, err
			}
			for _, d := range list {
				rows.Degrees = append(rows.Degrees, submission.Degree{ID: d.RecordKey, Level: d.Level, Institution: d.Institution, Type: d.Type,
					Title: d.Title, Area: d.Area, Registry: d.Registry, RegistryDate: dateText(d.RegistryDate), Document: reference(d.DocumentID)})
			}
		case 6:
			list, err := q.ListFormExperiences(ctx, id)
			if err != nil {
				return nil, err
			}
			for _, e := range list {
				rows.Experiences = append(rows.Experiences, submission.Experience{ID: e.RecordKey, Type: e.Type, Position: e.JobTitle,
					Institution: e.Institution, StartDate: dateText(e.StartDate), EndDate: dateText(e.EndDate), Current: e.IsCurrent,
					Activities: e.Activities, Functions: e.Functions, Document: reference(e.DocumentID)})
			}
		case 7:
			has, err := q.GetFormPublicationsInfo(ctx, id)
			if err != nil {
				return nil, err
			}
			rows.HasPublications = &has
			list, err := q.ListFormPublications(ctx, id)
			if err != nil {
				return nil, err
			}
			for _, p := range list {
				rows.Publications = append(rows.Publications, submission.Publication{ID: p.RecordKey, Title: p.Title, Type: p.Type,
					Medium: p.Medium, Date: dateText(p.PublishedOn), Related: p.Related, Document: reference(p.DocumentID)})
			}
		case 8:
			list, err := q.ListFormTrainings(ctx, id)
			if err != nil {
				return nil, err
			}
			for _, e := range list {
				rows.Trainings = append(rows.Trainings, submission.Training{ID: e.RecordKey, Type: e.Type, Name: e.Name, Institution: e.Institution,
					StartDate: dateText(e.StartDate), EndDate: dateText(e.EndDate), Hours: int(e.Hours), Document: reference(e.DocumentID)})
			}
		case 9:
			d, err := q.GetFormDisqualifications(ctx, id)
			if err != nil {
				return nil, err
			}
			rows.Disqualifications = &submission.DisqualificationsSection{Q1: d.Q1, Q2: d.Q2, Q3: d.Q3, Q4: d.Q4, Q5: d.Q5, Q6: d.Q6, Q7: d.Q7,
				Q8: d.Q8, Q9: d.Q9, Q10: d.Q10, Q11: d.Q11, Q12: d.Q12, Q13: d.Q13}
			removal, err := q.GetFormRemovalDetails(ctx, id)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return nil, err
			}
			if err == nil {
				rows.Disqualifications.Q7Details = &submission.RemovalDetails{Institution: removal.Institution, Position: removal.JobTitle,
					EndDate: dateText(removal.EndDate), Reason: removal.Reason}
			}
		}
	}
	consent, err := q.GetFormConsent(ctx, id)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if err == nil {
		rows.Consent = &submission.Consent{PrivacyPolicyID: consent.PrivacyPolicyID, PrivacyPolicyHash: consent.PrivacyPolicyHash,
			DataProcessing: consent.DataProcessing, Truthfulness: consent.Truthfulness, CompetitionRules: consent.CompetitionRules,
			NamePublication: consent.NamePublication, PhysicalDelivery: consent.PhysicalDelivery, Signature: consent.Signature, SignedAt: consent.SignedAt}
		if consent.AcceptedAt.Valid {
			rows.Consent.AcceptedAt = consent.AcceptedAt.Time.UTC().Format(time.RFC3339)
		}
	}
	form := rowsToForm(rows, formIdentity{IDCard: who.Cedula, Email: who.Correo, Sector: who.Sector})
	submission.NormalizeDurations(form, now)
	return form, nil
}
