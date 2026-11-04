package postgres

import "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"

// Traducción pura entre el formulario tipado y las filas de las tablas
// normalizadas (00024). No consulta la base: los identificadores del catálogo
// y de los documentos los resuelve el adaptador.

/** formSection marca una sección guardada; items indica si su lista repetible no es null. */
type formSection struct {
	Number int16
	Items  bool
}

/**
 * formRows es el formulario descompuesto en filas, sin los datos derivables:
 * cédula y correo (cuenta), sector (columna de submissions), duración
 * (se calcula), metadatos del PDF (documents) y literales del consentimiento.
 */
type formRows struct {
	Sections          []formSection
	Personal          *submission.PersonalSection
	Contact           *submission.ContactSection
	Additional        *string
	Relatives         []submission.Relative
	Position          *submission.PositionSection
	Degrees           []submission.Degree
	Experiences       []submission.Experience
	HasPublications   *string
	Publications      []submission.Publication
	Trainings         []submission.Training
	Disqualifications *submission.DisqualificationsSection
	Consent           *submission.Consent
}

/** formIdentity son los datos que el formulario toma de otras fuentes al leerlo. */
type formIdentity struct {
	IDCard string
	Email  string
	Sector string
}

/**
 * formToRows descompone el formulario quitando lo derivable.
 * @param {Form} form - Formulario ya decodificado.
 * @returns {formRows} Filas a escribir por sección.
 */
func formToRows(form *submission.Form) formRows {
	rows := formRows{}
	mark := func(number int16, items bool) {
		rows.Sections = append(rows.Sections, formSection{Number: number, Items: items})
	}
	if s := form.Personal; s != nil {
		mark(1, true)
		personal := *s
		personal.IDCard, personal.CivilRegistryVerified = "", false
		rows.Personal = &personal
	}
	if s := form.Contact; s != nil {
		mark(2, true)
		contact := *s
		contact.Email = ""
		rows.Contact = &contact
	}
	if s := form.Additional; s != nil {
		mark(3, s.Relatives != nil)
		rows.Additional = &s.HasRelatives
		rows.Relatives = s.Relatives
	}
	if s := form.Position; s != nil {
		mark(4, true)
		position := *s
		position.Sector = ""
		rows.Position = &position
	}
	if s := form.Education; s != nil {
		mark(5, s.Degrees != nil)
		rows.Degrees = s.Degrees
	}
	if s := form.Experience; s != nil {
		mark(6, s.Experiences != nil)
		rows.Experiences = make([]submission.Experience, len(s.Experiences))
		for i, e := range s.Experiences {
			e.Duration = nil
			rows.Experiences[i] = e
		}
	}
	if s := form.Publications; s != nil {
		mark(7, s.Publications != nil)
		rows.HasPublications = &s.HasPublications
		rows.Publications = s.Publications
	}
	if s := form.Training; s != nil {
		mark(8, s.Events != nil)
		rows.Trainings = s.Events
	}
	if s := form.Disqualifications; s != nil {
		mark(9, true)
		rows.Disqualifications = s
	}
	if c := form.Consent; c != nil {
		consent := *c
		consent.Version, consent.Statements = "", nil
		rows.Consent = &consent
	}
	return rows
}

/**
 * rowsToForm recompone el formulario con los datos derivables de su fuente.
 * Las duraciones quedan sin calcular: las completa submission.NormalizeDurations.
 * @param {formRows} rows - Filas leídas.
 * @param {formIdentity} identity - Cédula, correo y sector vigentes.
 * @returns {Form} Formulario equivalente al guardado.
 */
func rowsToForm(rows formRows, identity formIdentity) *submission.Form {
	form := &submission.Form{}
	items := map[int16]bool{}
	for _, s := range rows.Sections {
		items[s.Number] = s.Items
	}
	list := func(number int16, length int) bool { return items[number] || length > 0 }
	if _, ok := items[1]; ok && rows.Personal != nil {
		personal := *rows.Personal
		personal.IDCard, personal.CivilRegistryVerified = identity.IDCard, false
		form.Personal = &personal
	}
	if _, ok := items[2]; ok && rows.Contact != nil {
		contact := *rows.Contact
		contact.Email = identity.Email
		form.Contact = &contact
	}
	if _, ok := items[3]; ok && rows.Additional != nil {
		form.Additional = &submission.AdditionalSection{HasRelatives: *rows.Additional}
		if list(3, len(rows.Relatives)) {
			form.Additional.Relatives = append([]submission.Relative{}, rows.Relatives...)
		}
	}
	if _, ok := items[4]; ok && rows.Position != nil {
		position := *rows.Position
		position.Sector = identity.Sector
		form.Position = &position
	}
	if _, ok := items[5]; ok {
		form.Education = &submission.EducationSection{}
		if list(5, len(rows.Degrees)) {
			form.Education.Degrees = append([]submission.Degree{}, rows.Degrees...)
		}
	}
	if _, ok := items[6]; ok {
		form.Experience = &submission.ExperienceSection{}
		if list(6, len(rows.Experiences)) {
			form.Experience.Experiences = append([]submission.Experience{}, rows.Experiences...)
		}
	}
	if _, ok := items[7]; ok && rows.HasPublications != nil {
		form.Publications = &submission.PublicationsSection{HasPublications: *rows.HasPublications}
		if list(7, len(rows.Publications)) {
			form.Publications.Publications = append([]submission.Publication{}, rows.Publications...)
		}
	}
	if _, ok := items[8]; ok {
		form.Training = &submission.TrainingSection{}
		if list(8, len(rows.Trainings)) {
			form.Training.Events = append([]submission.Training{}, rows.Trainings...)
		}
	}
	if _, ok := items[9]; ok && rows.Disqualifications != nil {
		disqualifications := *rows.Disqualifications
		form.Disqualifications = &disqualifications
	}
	if rows.Consent != nil {
		consent := *rows.Consent
		submission.ApplyConsentLiterals(&consent)
		form.Consent = &consent
	}
	return form
}
